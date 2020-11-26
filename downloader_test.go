package downloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {

	tests := []struct {
		name      string
		url       string
		expectErr bool
	}{
		{name: "Success", url: "http://flat-icon-design.com/f/f_object_174/s512_f_object_174_0bg.png", expectErr: false},
		{name: "Fail: Invalid URL", url: "https://flat-icon-design.com/f/f_object_174/s512_f_object_174_0bg.png", expectErr: true},
		{name: "Fail: Invalid Accept-Range", url: "https://www.youtube.com/watch?v=RsEnknXJgPM", expectErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// set flag
			os.Args[1] = tt.url

			if err := Run(); tt.expectErr != true && err != nil {
				t.Error(err)
			}

			if tt.expectErr == true {
				return
			}
			if err := deleteFile(t, tt.url); err != nil {
				t.Error(err)
			}
		})
	}
}

func deleteFile(t *testing.T, url string) error {
	t.Helper()

	name := filepath.Base(url)

	if err := os.RemoveAll(name); err != nil {
		return err
	}

	return nil
}
