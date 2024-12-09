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
	"fmt"
	flag "github.com/spf13/pflag"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type (
	header struct {
		h http.Header
	}

	byteSize int64

	options struct {
		fs          *flag.FlagSet
		part        int
		base        string
		size        byteSize
		timeout     time.Duration
		header      header
		dir         []string
		reset       FileResets
		force       bool
		quiet       bool
		version     bool
		noConnReuse bool
	}
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

//go:embed docs/version_page
var VersionPage string

//go:embed docs/help_page
var HelpPage string

var (
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

	opt := &options{fs: flag.CommandLine}

	opt.init()
	uri, err := opt.parse()

	err = reqErrInfo(err)

	if err != nil {
		return nil, err
	}

	var ui func(*Download)

	if opt.quiet || opt.force {
		ui = nil
	} else {
		ui = ShowProgress
	}

	return &DLOptions{
		URI:       uri,
		BasePath:  opt.base,
		DstDirs:   opt.dir,
		PartCount: opt.part,
		PartSize:  int64(opt.size),
		ReDL:      opt.reset,
		UI:        ui,
		Force:     opt.force,
		IOMode: &IOMode{
			Timeout:     opt.timeout,
			UserHeader:  opt.header.h,
			NoConnReuse: opt.noConnReuse,
		},
	}, nil

}

func (opt *options) init() {

	opt.reset = map[FileState]bool{
		Broken: false, Completed: false, Resume: false,
	}
	opt.header.h = make(http.Header)

	fs := opt.fs
	fs.Init(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	fs.IntVarP(&opt.part, "part", "p", 1, "")

	fs.VarP(&opt.size, "size", "s", "")

	fs.StringVarP(&opt.base, "base", "b", "", "")

	fs.StringSliceVarP(&opt.dir, "dir", "d", []string{""}, "")

	fs.DurationVarP(&opt.timeout, "timeout", "t", 0, "")

	fs.VarP(&opt.header, "header", "H", "")

	fs.BoolVarP(&opt.force, "force", "f", false, "")

	fs.BoolVarP(&opt.quiet, "quiet", "q", false, "")

	fs.BoolVarP(&opt.noConnReuse, "no-connection-reuse", "x", false, "")

	fs.VarP(&opt.reset, "reset", "z", "")
	flag.Lookup("reset").NoOptDefVal = "0,1,2"

	fs.BoolVarP(&opt.version, "version", "V", false, "")

}

func (opt *options) parse() (uri string, err error) {

	fs := opt.fs

	if err = fs.Parse(os.Args[1:]); err != nil {
		return "", err
	}

	if opt.version {
		return "", ErrVer
	}

	args := fs.Args()

	if len(args) > 0 {
		uri = args[0]
	}

	if uri == "" {
		return "", flag.ErrHelp
	}

	return uri, nil

}

func reqErrInfo(err error) error {

	if err != nil {
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

func (fr *FileResets) String() string {
	return fmt.Sprintf("%v", (*fr))
}

func (fr *FileResets) Type() string {
	return "FileResets"
}

func (fr *FileResets) Set(value string) error {

	(*fr) = map[FileState]bool{
		Broken: false, Completed: false, Resume: false,
	}

	opt := strings.SplitN(value, ",", 3)

	for _, o := range opt {

		switch strings.TrimSpace(o) {
		case "0":
			(*fr)[Broken] = true
		case "1":
			(*fr)[Completed] = true
		case "2":
			(*fr)[Resume] = true
		default:
			return NewErr("parse error")
		}

	}

	return nil

}

func (h *header) String() string {
	return fmt.Sprintf("%+v", h.h)
}

func (h *header) Type() string {
	return "Header"
}

func (h *header) Set(value string) error {

	if kv := strings.SplitN(value, ":", 2); len(kv) > 1 {

		h.h.Add(kv[0], strings.Trim(kv[1], " "))
	}

	return nil

}

func (bs *byteSize) String() string {
	return fmt.Sprintf("%d", *bs)
}

func (bs *byteSize) Type() string {
	return "ByteSize"
}

func (bs *byteSize) Set(value string) error {

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
		return NewErr("parse error")
	}

	if byteCount, err := strconv.ParseInt(byteStr, 10, 64); err == nil {
		*bs = byteSize(byteCount * multiplier)
	} else {
		return err
	}

	return nil

}

func IsFile(path string) bool {

	if info, err := os.Stat(path); err == nil {
		return info.Mode().IsRegular()
	} else {
		return false
	}

}

func IsEndSeparator(path string) bool {

	path = strings.TrimSpace(path)
	return path[len(path)-1:] == PathSeparator

}

func IsURL(rawURL string) bool {

	if u, err := url.Parse(rawURL); err == nil {
		return (u.Scheme == "http" || u.Scheme == "https")
	} else {
		return false
	}

}
