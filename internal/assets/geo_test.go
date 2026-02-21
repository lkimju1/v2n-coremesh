package assets

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNeedsRefresh(t *testing.T) {
	now := time.Now()
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "missing.dat")

	need, err := needsRefresh(missing, now, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("missing needsRefresh error: %v", err)
	}
	if !need {
		t.Fatal("missing file should require refresh")
	}

	fresh := filepath.Join(tmp, "fresh.dat")
	if err := os.WriteFile(fresh, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(fresh, now, now.Add(-10*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	need, err = needsRefresh(fresh, now, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("fresh needsRefresh error: %v", err)
	}
	if need {
		t.Fatal("fresh file should not require refresh")
	}

	stale := filepath.Join(tmp, "stale.dat")
	if err := os.WriteFile(stale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(stale, now, now.Add(-40*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	need, err = needsRefresh(stale, now, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("stale needsRefresh error: %v", err)
	}
	if !need {
		t.Fatal("stale file should require refresh")
	}
}

func TestEnsureGeoFilesTriggersDownload(t *testing.T) {
	now := time.Now()
	tmp := t.TempDir()

	var downloaded []string
	oldDownloader := downloadGeoFile
	downloadGeoFile = func(url, target string) error {
		downloaded = append(downloaded, filepath.Base(target))
		return os.WriteFile(target, []byte(url), 0o644)
	}
	defer func() { downloadGeoFile = oldDownloader }()

	if err := EnsureGeoFiles(tmp, now); err != nil {
		t.Fatalf("ensure missing files: %v", err)
	}
	if len(downloaded) != 2 {
		t.Fatalf("unexpected downloaded files: %#v", downloaded)
	}

	downloaded = downloaded[:0]
	if err := EnsureGeoFiles(tmp, now.Add(5*24*time.Hour)); err != nil {
		t.Fatalf("ensure fresh files: %v", err)
	}
	if len(downloaded) != 0 {
		t.Fatalf("fresh files should not be downloaded: %#v", downloaded)
	}
}
