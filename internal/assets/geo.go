package assets

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	geoMaxAge       = 30 * 24 * time.Hour
	geoURLTemplate  = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/%s.dat"
	downloadTimeout = 90 * time.Second
)

var geoFileNames = []string{"geosite", "geoip"}

var downloadGeoFile = defaultDownloadGeoFile

func EnsureGeoFiles(confDir string, now time.Time) error {
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		return fmt.Errorf("create conf dir: %w", err)
	}

	for _, name := range geoFileNames {
		target := filepath.Join(confDir, name+".dat")
		need, err := needsRefresh(target, now, geoMaxAge)
		if err != nil {
			return fmt.Errorf("check %s: %w", target, err)
		}
		if !need {
			continue
		}
		url := fmt.Sprintf(geoURLTemplate, name)
		if err := downloadGeoFile(url, target); err != nil {
			return fmt.Errorf("download %s: %w", name, err)
		}
	}
	return nil
}

func needsRefresh(path string, now time.Time, maxAge time.Duration) (bool, error) {
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	if st.IsDir() {
		return false, fmt.Errorf("expected file but got directory: %s", path)
	}
	return now.Sub(st.ModTime()) > maxAge, nil
}

func defaultDownloadGeoFile(url, target string) error {
	client := &http.Client{Timeout: downloadTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	tmp := target + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp, target)
}
