package main

import (
	"net/url"
	"os"
	"path/filepath"
)

func getRawURL(a []string) string {
	return a[len(a)-1]
}

func isFile(path string) (bool, error) {
	if info, err := os.Stat(filepath.Clean(path)); err == nil {
		return info.Mode().IsRegular(), nil
	} else {
		return false, err
	}
}

func isDir(path string) (bool, error) {
	if info, err := os.Stat(filepath.Clean(path)); err == nil {
		return info.Mode().IsDir(), nil
	} else {
		return false, err
	}
}

func isURL(rawURL string) (bool, error) {
	if u, err := url.Parse(rawURL); err == nil {
		return (u.Scheme == "http" || u.Scheme == "https"), nil
	} else {
		return false, err
	}
}
