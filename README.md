# partdec
[![Go Report Card](https://goreportcard.com/badge/github.com/cjijcb/partdec)](https://goreportcard.com/report/github.com/cjijcb/partdec)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/af4b1b130f194d6caa0edeb4cce4d342)](https://app.codacy.com/gh/cjijcb/partdec/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)

**partdec** is a command-line utility for multipart downloading and file splitting. It can download a file in parts simultaneously from a remote or local source and distribute parts of the file to multiple destination paths.

partdec allows a separate connection per file part and the ability to resume interrupted file transfers.
It supports HTTP and HTTPS protocols.


## Demo
![0-demo](https://github.com/cjijcb/partdec/blob/master/assets/0-demo.gif) 

<details>
<summary><strong>See more ...</strong></summary>
<img src="https://github.com/cjijcb/partdec/blob/master/assets/1-demo.gif">
<img src="https://github.com/cjijcb/partdec/blob/master/assets/2-demo.gif"> 
</details>


## Installation

> [!NOTE]
> For installation, `go` version 1.22 or later is required.

Install it with `go`:
```bash
go install github.com/cjijcb/partdec/cmd/partdec@latest
```

Or, to build the binary, download the latest source code archive from the [releases](https://github.com/cjijcb/partdec/releases) page,
extract it, then run:
```bash
cd partdec/cmd/partdec/
go build -o <BINPATH> partdec.go  # Replace <BINPATH> with the path where the binary file should go
```



## Combining Split Files

`cat` is a command-line utility in Unix-based systems that can be used to combine files. The
following guide also works in the PowerShell terminal on Windows systems. If you'd like to
use other software, refer to the manual of the software of your choice.

### Files in the same or multiple directories.

Sypnosis:
```
cat [PATH]<FILENAME>_* ... > [PATH]<NEWFILENAME>
```
Examples:
```bash
cat archive.zip_* > my_archive.zip
```
```bash
cat /tmp/archive.zip_* > ~/Downloads/my_archive.zip
```
```bash
cat /tmp/archive.zip_* /var/archive.zip_* > ~/Downloads/my_archive.zip
```
> [!NOTE]
> `<FILENAME>` is the filename without `_N` suffix. 

> [!IMPORTANT] 
>The paths must be in ascending order based on the numeric suffix of the files,
>from left (lowest numeric suffix) to (right highest numeric suffix).

## Command-Line Usage

Basic Options:
```
  -p <N>     Split the file into N parts.
  -s <SIZE>  Split the file into parts of SIZE.
  -b <PATH>  Set the base path for output files and also set their filename.
  -d <PATH>  Set the destination directory for output files. PATH can be multiple comma-separated directories.
```

<details>
<summary><strong>See full details ...</strong></summary>

<pre>
Usage: partdec [OPTIONS]... &lt;URL|LOCAL PATH&gt;

Options:
  -p, --part &lt;N&gt;
            Split the file into N parts. If N is zero or less, it defaults to
            1. If -s/--size is used, this option is ignored.

  -s, --size &lt;SIZE&gt;
            Split the file into parts of SIZE. SIZE is in byte size and can
            include the following binary prefixes:
            SI: KB, MB, GB, TB (case-insensitive)
            IEC: KiB, MiB, GiB, TiB, or K, M, G, T (case-insensitive)

  -b, --base &lt;PATH&gt;
            Set the base path for output files and also set their filename.
            For multiple output files, an _N suffix is added, where N is an
            incrementing number starting from 1.

  -d, --dir &lt;PATH&gt;
            Set the destination directory for output files. PATH can be
            multiple comma-separated directories. This option can also be used
            multiple times to specify multiple directories. Each specified
            directory is combined with the base path (dir + base path).

  -t, --timeout &lt;TIME&gt;
            Set the HTTP request timeout. TIME is a number followed by a
            suffix: ms, s, m, or h to represent milliseconds, seconds, minutes,
            or hours, respectively (e.g., -t 1h2m3s). The default is 0, meaning
            no timeout.
  
  -x, --no-connection-reuse
            Disable the HTTP Keep-Alive or connection reuse. This ensures a
            separate connection per file part in multipart HTTP(S) downloads.

  -H, --header &lt;HEADER_NAME:VALUE&gt;
            Set or add an HTTP header. This option can be used multiple times
            to specify multiple headers. The Range header is ignored in
            multipart HTTP(S) downloads. HEADER_NAME is case-insensitive.

  -f, --force
            Override the soft limit (128) on the total number of output files.
            This option also enables quiet mode.

  -q, --quiet
            Enable quiet mode.

  -z, --reset
            Reset files with an initial state of [completed], [resume], or
            [broken] to [new]. Same as -CBR.

  -C, --reset-completed
            Reset files with an initial [completed] state to [new].

  -B, --reset-broken
            Reset files with an initial [broken] state to [new].

  -R, --reset-resume
            Reset files with an initial [resume] state to [new].

  -V, --version
            Display version information.

Output File States:
    File states are based on the initial size of files and may change during
    or after the download. States can also be affected by I/O operation errors
    and the file scope, which determines the maximum size a file can reach.

    [new]       File with zero initial size.
    [resume]    File with non-zero initial size and within scope.
    [completed] File that has reached its maximum size.
    [broken]    File exceeding maximum size or with I/O errors.
    [unknown]   File with undetermined scope.

    A file with the [unknown] state is always truncated to zero size on every
    run with the same arguments. This state occurs when an HTTP(S) server does
    not support multipart downloading. </pre>
</details>

## License
Copyright (C) 2024 Carlo Jay I. Jacaba

partdec is licensed under the Apache License, Version 2.0.
