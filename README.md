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
  -d <PATH>  Set the destination directory for output files. PATH can be comma-separated directories.
```
See [Full Details](https://github.com/cjijcb/partdec/wiki/Command%E2%80%90Line-Usage).

## License
Copyright (C) 2024 Carlo Jay I. Jacaba

partdec is licensed under the Apache License, Version 2.0.
