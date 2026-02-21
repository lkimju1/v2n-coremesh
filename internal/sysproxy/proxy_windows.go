//go:build windows

package sysproxy

import (
	"fmt"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const regPath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

const (
	internetOptionSettingsChanged = 39
	internetOptionRefresh         = 37
)

var (
	wininet                = syscall.NewLazyDLL("wininet.dll")
	procInternetSetOptionW = wininet.NewProc("InternetSetOptionW")
)

type proxySnapshot struct {
	ProxyEnable   uint32
	ProxyServer   string
	ProxyOverride string
	AutoConfigURL string
}

func ConfigureForRun(xrayConfigPath string) (func() error, bool, error) {
	snapshot, err := readProxySnapshot()
	if err != nil {
		return nil, false, err
	}

	if snapshot.ProxyEnable != 0 || strings.TrimSpace(snapshot.AutoConfigURL) != "" {
		return func() error { return nil }, false, nil
	}

	endpoint, err := DetectProxyEndpoint(xrayConfigPath)
	if err != nil {
		return nil, false, err
	}

	proxyServer := fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
	bypass := mergeBypass(snapshot.ProxyOverride, RequiredBypassList)
	if err := applyProxySettings(1, proxyServer, bypass, ""); err != nil {
		return nil, false, err
	}

	restore := func() error {
		return applyProxySettings(snapshot.ProxyEnable, snapshot.ProxyServer, snapshot.ProxyOverride, snapshot.AutoConfigURL)
	}
	return restore, true, nil
}

func readProxySnapshot() (*proxySnapshot, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.QUERY_VALUE)
	if err != nil {
		return nil, fmt.Errorf("open proxy registry: %w", err)
	}
	defer key.Close()

	proxyEnable, err := readUint32Value(key, "ProxyEnable")
	if err != nil {
		return nil, err
	}
	proxyServer, err := readStringValue(key, "ProxyServer")
	if err != nil {
		return nil, err
	}
	proxyOverride, err := readStringValue(key, "ProxyOverride")
	if err != nil {
		return nil, err
	}
	autoConfigURL, err := readStringValue(key, "AutoConfigURL")
	if err != nil {
		return nil, err
	}
	return &proxySnapshot{
		ProxyEnable:   proxyEnable,
		ProxyServer:   proxyServer,
		ProxyOverride: proxyOverride,
		AutoConfigURL: autoConfigURL,
	}, nil
}

func applyProxySettings(proxyEnable uint32, proxyServer, proxyOverride, autoConfigURL string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open proxy registry for write: %w", err)
	}
	defer key.Close()

	if err := key.SetDWordValue("ProxyEnable", proxyEnable); err != nil {
		return fmt.Errorf("set ProxyEnable: %w", err)
	}
	if err := writeStringValue(key, "ProxyServer", proxyServer); err != nil {
		return fmt.Errorf("set ProxyServer: %w", err)
	}
	if err := writeStringValue(key, "ProxyOverride", proxyOverride); err != nil {
		return fmt.Errorf("set ProxyOverride: %w", err)
	}
	if err := writeStringValue(key, "AutoConfigURL", autoConfigURL); err != nil {
		return fmt.Errorf("set AutoConfigURL: %w", err)
	}

	if err := refreshSystemProxySettings(); err != nil {
		return err
	}
	return nil
}

func readUint32Value(key registry.Key, name string) (uint32, error) {
	v, _, err := key.GetIntegerValue(name)
	if err != nil {
		if err == registry.ErrNotExist {
			return 0, nil
		}
		return 0, fmt.Errorf("read %s: %w", name, err)
	}
	return uint32(v), nil
}

func readStringValue(key registry.Key, name string) (string, error) {
	v, _, err := key.GetStringValue(name)
	if err != nil {
		if err == registry.ErrNotExist {
			return "", nil
		}
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	return strings.TrimSpace(v), nil
}

func writeStringValue(key registry.Key, name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		if err := key.DeleteValue(name); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}
	return key.SetStringValue(name, value)
}

func refreshSystemProxySettings() error {
	if r, _, callErr := procInternetSetOptionW.Call(0, uintptr(internetOptionSettingsChanged), 0, 0); r == 0 {
		return fmt.Errorf("refresh proxy settings changed: %w", callErr)
	}
	if r, _, callErr := procInternetSetOptionW.Call(0, uintptr(internetOptionRefresh), 0, 0); r == 0 {
		return fmt.Errorf("refresh proxy settings: %w", callErr)
	}
	return nil
}
