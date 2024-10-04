package main

import (
	"net/url"
	"os"
	"path/filepath"
	//"errors"
)

func getRawURL(a []string) string {
	return a[len(a)-1]
}

func isFile(path string) bool {
	if info, err := os.Stat(filepath.Clean(path)); err == nil {
		return info.Mode().IsRegular()
	} else {
		return false
	}
}

func isDir(path string) bool {
	if info, err := os.Stat(filepath.Clean(path)); err == nil {
		return info.Mode().IsDir()
	} else {
		return false
	}
}

func isURL(rawURL string) bool {
	if u, err := url.Parse(rawURL); err == nil {
		return (u.Scheme == "http" || u.Scheme == "https")
	} else {
		return false
	}
}
