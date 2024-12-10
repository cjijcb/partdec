package partdec

import (
	"fmt"
	"os"
	"testing"
)

var (
	opt = DLOptions{
		URI:       "http://example.com",
		BasePath:  "example.html",
		DstDirs:   []string{"test/dira", "test/dirb"},
		PartCount: 0,
		PartSize:  0,
		ReDL:      map[FileState]bool{Completed: true, Resume: true, Broken: true},
		UI:        nil,
		Force:     false,
		Mod: &IOMod{
			Timeout:     0,
			UserHeader:  nil,
			NoConnReuse: false,
		},
	}
)

func TestNewDownload(t *testing.T) {

	var d *Download
	var err error

	os.MkdirAll("test/dira", 0750)
	os.MkdirAll("test/dirb", 0750)
	defer os.RemoveAll("test/")

	newOpt := opt
	if d, err = NewDownload(&newOpt); err != nil {
		t.Errorf("unexpected error: %s\n", err)
	}
	dataSize := d.DataSize

	newOpt = opt
	newOpt.PartSize = (dataSize / 10)
	if d, err = NewDownload(&newOpt); err != nil {
		t.Errorf("unexpected error: %s\n", err)
	}

	for i := range newOpt.PartCount {
		switch {
		case i <= 5:
			path := "test/dira/" + "example.html_" + fmt.Sprintf("%02d", i+1)
			if !IsFile(path) {
				t.Errorf("file not found: %s\n", path)
			}
		case i > 5 && i < 10:
			path := "test/dirb/" + "example.html_" + fmt.Sprintf("%02d", i+1)
			if !IsFile(path) {
				t.Errorf("file not found: %s\n", path)
			}
		}
	}

	for i := range 6 {
		newOpt := opt

		switch i {
		case 0:
			newOpt.PartCount = -1
			if _, err := NewDownload(&newOpt); err != nil {
				t.Errorf("unexpected error: %s\n", err)
			}
			fallthrough
		case 1:
			newOpt.PartSize = -1
			if _, err := NewDownload(&newOpt); err != nil {
				t.Errorf("unexpected error: %s\n", err)
			}

		case 2:
			newOpt.PartCount = (PartSoftLimit + 1)
			newOpt.Force = true
			if _, err := NewDownload(&newOpt); err != nil {
				t.Errorf("unexpected error: %s\n", err)
			}
		case 3:
			newOpt.PartCount = (PartSoftLimit + 1)
			if _, err := NewDownload(&newOpt); err == nil {
				t.Errorf("error is expected")
			}
		case 4:
			newOpt.PartCount = (int(dataSize) + 1)
			newOpt.Force = true
			if _, err := NewDownload(&newOpt); err == nil {
				t.Errorf("error is expected")
			}
		case 5:
			newOpt.PartSize = (dataSize + 1)
			if _, err := NewDownload(&newOpt); err == nil {
				t.Errorf("error is expected")
			}
		}

	}

}
