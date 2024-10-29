package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type (
	ErrBuilder struct {
		*strings.Builder
	}
	FlagMultiStr []string
)

var (
	PartFlag   int
	DirFlag    FlagMultiStr
	ErrOnFlags = &ErrBuilder{new(strings.Builder)}
)

func maintest() {

	flag.CommandLine.SetOutput(ErrOnFlags)

	flag.Var(&DirFlag, "dir", "Specify directories (can be used times)")

	flag.IntVar(&PartFlag, "part", 1, "help message")

	flag.Usage = func() {
		fmt.Fprint(ErrOnFlags, "This is a custom help message for our application.\n\n")
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, ErrOnFlags.String())
	}

	flag.Parse()

	fmt.Println("part value is: ", PartFlag)
	fmt.Println("Directories:", DirFlag)

}

func (e *ErrBuilder) Error() string {
	return e.String()
}

func (fms *FlagMultiStr) String() string {
	return strings.Join(*fms, ",")
}

func (fms *FlagMultiStr) Set(value string) error {
	*fms = append(*fms, value)
	return nil
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
