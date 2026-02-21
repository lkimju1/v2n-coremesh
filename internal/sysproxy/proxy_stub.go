//go:build !windows

package sysproxy

func ConfigureForRun(_ string) (func() error, bool, error) {
	return func() error { return nil }, false, nil
}
