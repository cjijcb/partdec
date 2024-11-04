/*
Copyright 2024 Carlo Jay I. Jacaba

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package partdec

import (
	_ "embed"
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
)

//go:embed doc/version_page.txt
var VersionPage string

//go:embed doc/help_page.txt
var HelpPage string

var (
	PartFlag          int
	BaseFlag          string
	SizeFlag          ByteSize
	TimeoutFlag       time.Duration
	HeaderFlag        Header
	DirFlag           Paths
	ZeroResumeFlag    bool
	ZeroCompletedFlag bool
	ZeroBrokenFlag    bool
	ZeroAllFlag       bool
	ForcePartFlag     bool
	QuietFlag         bool
	VersionFlag       bool

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

func NewDLOptions() (*DLOptions, error) {

	InitArgs(flag.CommandLine)
	uri, err := ParseArgs(flag.CommandLine)
	err = HandleArgsErr(err)

	if err != nil {
		return nil, err
	}

	var zmap map[FileState]bool
	switch {
	case ZeroAllFlag:
		zmap = map[FileState]bool{Completed: true, Resume: true, Broken: true}
	default:
		zmap = map[FileState]bool{
			Completed: ZeroCompletedFlag,
			Resume:    ZeroResumeFlag,
			Broken:    ZeroBrokenFlag,
		}
	}

	var ui func(*Download)
	switch {
	case QuietFlag || ForcePartFlag:
		ui = nil
	default:
		ui = ShowProgress
	}

	return &DLOptions{
		URI:       uri,
		BasePath:  BaseFlag,
		DstDirs:   DirFlag,
		PartCount: PartFlag,
		PartSize:  int64(SizeFlag),
		ReDL:      zmap,
		UI:        ui,
		Force:     ForcePartFlag,
		IOMode: &IOMode{
			Timeout:    TimeoutFlag,
			UserHeader: HeaderFlag.Header,
		},
	}, nil

}

func InitArgs(fs *flag.FlagSet) {

	fs.Init(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	fs.Var(&DirFlag, "dir", "")
	fs.Var(&DirFlag, "d", "")

	SizeFlag = -1
	fs.Var(&SizeFlag, "size", "")
	fs.Var(&SizeFlag, "s", "")

	HeaderFlag = Header{make(http.Header)}
	fs.Var(&HeaderFlag, "header", "")
	fs.Var(&HeaderFlag, "H", "")

	fs.IntVar(&PartFlag, "part", 1, "")
	fs.IntVar(&PartFlag, "p", 1, "")

	fs.StringVar(&BaseFlag, "base", "", "")
	fs.StringVar(&BaseFlag, "b", "", "")

	fs.DurationVar(&TimeoutFlag, "timeout", 0, "")
	fs.DurationVar(&TimeoutFlag, "t", 0, "")

	fs.BoolVar(&ZeroResumeFlag, "zr", false, "")
	fs.BoolVar(&ZeroCompletedFlag, "zc", false, "")
	fs.BoolVar(&ZeroBrokenFlag, "zb", false, "")
	fs.BoolVar(&ZeroAllFlag, "za", false, "")

	fs.BoolVar(&ForcePartFlag, "fp", false, "")
	fs.BoolVar(&QuietFlag, "q", false, "")

	fs.BoolVar(&VersionFlag, "version", false, "")

}

func ParseArgs(fs *flag.FlagSet) (string, error) {

	err := fs.Parse(os.Args[1:])

	if err != nil {
		return "", err
	}

	if VersionFlag {
		return "", ErrVer
	}

	var uri string

	args := fs.Args()

	for len(args) > 0 {

		switch {
		case uri == "":
			uri = args[0]
		default:
			return "", NewErr("%s: %s", ErrArgs, uri)
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

func HandleArgsErr(err error) error {

	if err != nil {

		if strings.Contains(err.Error(), "flag provided but not defined:") {
			err = NewErr(
				"%s:%s\n",
				ErrArgs,
				strings.SplitAfterN(err.Error(), ":", 2)[1],
			)
		}

		switch {
		case IsErr(err, ErrVer):
			fmt.Fprintf(os.Stderr, "%s", VersionPage)
		case IsErr(err, flag.ErrHelp):
			fmt.Fprintf(os.Stderr, "%s", HelpPage)
		default:
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}

		return err

	}

	return nil

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

	if hKeyVal := strings.SplitN(value, ":", 2); len(hKeyVal) > 1 {

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
