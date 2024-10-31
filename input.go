package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

type (
	ErrBuilder struct {
		*strings.Builder
	}
	Paths    []string
	ByteSize int64
)

const (
	Kibi = 1024
	Mebi = 1024 * 1024
	Gibi = 1024 * 1024 * 1024
	Tebi = 1024 * 1024 * 1024 * 1024

	Kilo = 1000
	Mega = 1000 * 1000
	Giga = 1000 * 1000 * 1000
	Tera = 1000 * 1000 * 1000 * 1000
)

var (
	PartFlag   int      = 1
	SizeFlag   ByteSize = -1
	DirFlag    Paths
	ErrOnFlags = &ErrBuilder{new(strings.Builder)}

	ByteUnit = map[string]int64{
		"":  1,
		"B": 1,

		"KIB": Kibi,
		"MIB": Mebi,
		"GIB": Gibi,
		"TIB": Tebi,

		"K": Kibi,
		"M": Mebi,
		"G": Gibi,
		"T": Tebi,

		"KB": Kilo,
		"MB": Mega,
		"GB": Giga,
		"TB": Tera,
	}
)

func main() {

	flag.CommandLine.SetOutput(ErrOnFlags)

	flag.Var(&DirFlag, "dir", "Specify directories (can be used times)")

	flag.Var(&SizeFlag, "size", "Specify directories (can be used times)")

	flag.IntVar(&PartFlag, "part", 1, "help message")

	flag.Usage = func() {
		fmt.Fprint(ErrOnFlags, "This is a custom help message for our application.\n\n")
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, ErrOnFlags.String())
	}

	flag.Parse()

	fmt.Println("part value is: ", PartFlag)
	fmt.Println("Directories:", DirFlag)
	fmt.Println("Size:", SizeFlag)

}

func (e *ErrBuilder) Error() string {
	return e.String()
}

func (ps *Paths) String() string {
	return strings.Join(*ps, ",")
}

func (ps *Paths) Set(value string) error {
	*ps = append(*ps, value)
	return nil
}

func (bs *ByteSize) String() string {
	return fmt.Sprintf("%d", bs)
}

func (bs *ByteSize) Set(value string) error {

	unitStr := strings.TrimLeftFunc(
		value,
		func(r rune) bool { return unicode.IsNumber(r) },
	)

	byteStr := strings.TrimRightFunc(
		value,
		func(r rune) bool { return !unicode.IsNumber(r) },
	)

	multiplier, found := ByteUnit[strings.ToUpper(unitStr)]

	if !found {
		return fmt.Errorf("parse error")
	}

	if byteCount, err := strconv.ParseInt(byteStr, 10, 64); err == nil {
		*bs = ByteSize(byteCount * multiplier)
	} else {
		return err
	}

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
