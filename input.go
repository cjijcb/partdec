package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type (
	Header struct {
		http.Header
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

	ManPage = "TODO manpage"
)

var (
	PartFlag    int           = 1
	SizeFlag    ByteSize      = -1
	TimeoutFlag time.Duration = 0
	DirFlag     Paths
	HeaderFlag  Header = Header{make(http.Header)}

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

func init() {

	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	flag.CommandLine.Usage = func() {}

	flag.Var(&DirFlag, "dir", "")
	flag.Var(&SizeFlag, "size", "")
	flag.Var(&HeaderFlag, "header", "")
	flag.IntVar(&PartFlag, "part", 1, "")
	flag.DurationVar(&TimeoutFlag, "timeout", 0, "")

}

func main() {

	uri, err := ParseArgs(flag.CommandLine)

	if err != nil {

		switch {
		case errors.Is(err, flag.ErrHelp):
			fmt.Fprintf(os.Stderr, "%s\n", ManPage)
		case strings.Contains(err.Error(), "flag provided but not defined:"):
			fmt.Fprintf(
				os.Stderr,
				"invalid argument:%s\n",
				strings.SplitAfterN(err.Error(), ":", 2)[1],
			)
		default:
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}

		os.Exit(2)

	}

	fmt.Println("uri: ", uri)
	fmt.Printf("header: %+v\n", HeaderFlag)
	fmt.Println("timeout: ", TimeoutFlag)
	fmt.Println("part value is: ", PartFlag)
	fmt.Println("Directories:", DirFlag)
	fmt.Println("Size:", SizeFlag)

}

func ParseArgs(fs *flag.FlagSet) (string, error) {

	err := fs.Parse(os.Args[1:])

	if err != nil {
		return "", err
	}

	var uri string

	args := fs.Args()

	for len(args) > 0 {

		switch {
		case uri == "":
			uri = args[0]
		default:
			return "", fmt.Errorf("invalid argument: %s", uri)
		}

		err := fs.Parse(args[1:])

		if err != nil {
			return "", err
		}

		args = fs.Args()
	}

	if uri == "" {
		return "", flag.ErrHelp
	}

	return uri, nil

}

func (ps *Paths) String() string {
	return strings.Join(*ps, ",")
}

func (ps *Paths) Set(value string) error {
	*ps = append(*ps, value)
	return nil
}

func (h *Header) String() string {
	return fmt.Sprintf("%+v", h.Header)
}

func (h *Header) Set(value string) error {

	if hKeyVal := strings.SplitAfterN(value, ":", 2); len(hKeyVal) > 1 {

		h.Add(hKeyVal[0], strings.Trim(hKeyVal[1], " "))
	}

	return nil
}

func (bs *ByteSize) String() string {
	return fmt.Sprintf("%d", *bs)
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
