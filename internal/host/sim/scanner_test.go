package sim

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanSourcesSkipsBlankFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"empty.txt":   "",
		"blank.md":    "  \n\t\n",
		"sample.txt":  "Một bài văn mẫu.",
		"HUONG-DAN":   "không phải văn mẫu",
		"ghi-chu.doc": "bỏ qua",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := scanSources(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].RelativePath != "sample.txt" {
		t.Fatalf("scanSources = %+v, want only sample.txt", got)
	}
}
