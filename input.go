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

func isFile(path string) (bool, error) {
	if info, err := os.Stat(filepath.Clean(path)); info.Mode().IsRegular() {
		return true, err
	} else {
		return false, err
	} 
}

func isDir(path string) (bool, error) {
	if info, err := os.Stat(filepath.Clean(path)); info.Mode().IsDir() {
		return true, err
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
