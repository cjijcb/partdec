# partdec
[![Go Report Card](https://goreportcard.com/badge/github.com/cjijcb/partdec)](https://goreportcard.com/report/github.com/cjijcb/partdec)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/af4b1b130f194d6caa0edeb4cce4d342)](https://app.codacy.com/gh/cjijcb/partdec/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)

**partdec** is a command-line utility for multipart downloading and file splitting.
It can download a file in parts simultaneously from a remote or local source and distribute parts of the file to multiple destination paths.

partdec allows a separate connection per file part and the ability to resume interrupted file transfers.
It supports HTTP and HTTPS protocols.


## Demo
![0-demo](https://github.com/cjijcb/partdec/blob/master/assets/0-demo.gif) 

<details>
<summary><strong>See more ...</strong></summary>
<img src="https://github.com/cjijcb/partdec/blob/master/assets/1-demo.gif">
<img src="https://github.com/cjijcb/partdec/blob/master/assets/2-demo.gif"> 
</details>

## Use Cases

- **Download Acceleration** <br>
  Multipart or segmented download is a commonly used technique to accelerate download
  speeds by retrieving multiple parts of a file simultaneously.

- **Data Partitioning and Distribution** <br>
  In cases when data has to come from one endpoint and then be split into parts and
  distributed across multiple nodes or servers.

- **Large File Management** <br>
  Transferring a file in parts solves the problem of file size restrictions imposed by
  file systems. For instance, transferring a file larger than 4GB to a FAT32 system.

## Installation
> [!NOTE]
> For installation, `go` version 1.22.8 or later is required.

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
## Usage Information

Basic Options:
```
  -p <N>     Split the file into N parts.
  -s <SIZE>  Split the file into parts of SIZE.
  -b <PATH>  Set the base path for output files and also set their filename.
  -d <PATH>  Set the destination directory for output files. PATH can be comma-separated directories.
```
See [Full Usage Information](https://github.com/cjijcb/partdec/wiki/Command%E2%80%90Line-Usage-Information).

## Merging Files
You can use `cat`, a standard Unix utility, to merge files. Other similar applications can work as well.
To merge files using `cat`, the paths must be passed in ascending order based on the numeric suffix of the files.
You can also use a wildcard (`*`) to represent these numeric suffixes. As follows:

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

## Development
It is still in early development, and a lot can still change. Any contributions are welcome.
There are also plans to support other protocols such as FTP(S) and SFTP.

## License
Copyright (C) 2024 Carlo Jay I. Jacaba

partdec is licensed under the Apache License, Version 2.0.
