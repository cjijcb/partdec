# partdec
**partdec** is a command-line utility for multipart downloading and file splitting. It can
seamlessly split a file from the web or local storage and distribute parts of the file to
multiple destination paths.

In web downloading, partdec allows a dedicated connection per output file and handles interruptions safely,
allowing for resumable downloads. It supports HTTP and HTTPS.


## Demo
![0-demo](https://github.com/cjijcb/partdec/blob/master/doc/gif/0-demo.gif) 

<details>
<summary><strong>See more ...</strong></summary>
<img src="https://github.com/cjijcb/partdec/blob/master/doc/gif/1-demo.gif">
<img src="https://github.com/cjijcb/partdec/blob/master/doc/gif/2-demo.gif"> 
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
cd <PATH>  # Replace <PATH> with the path where the extracted folder is located.
go build -o <BINPATH> cmd/partdec/partdec.go  # Replace <BINPATH> with the path where the binary file should go
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
  -s <SIZE>  Split the file into parts based on SIZE.
  -b <PATH>  Set the base path for output files and also set their filename.
  -d <PATH>  Set the destination directory for output files. Can be used multiple times.
```

<details>
<summary><strong>See full details ...</strong></summary>

<pre>
Usage: partdec [OPTIONS]... &lt;URL|LOCAL PATH&gt;
Seamlessly split files from the web or local storage. 

Options:
  -p, -part &lt;N&gt;
    Split the file into N parts.
    If N is zero or less, it defaults to 1.
    If -s/-size is used, this option is ignored.

  -s, -size &lt;SIZE&gt;
    Split the file into parts based on SIZE.
    SIZE is an integer (representing byte size) and can be followed by one of the
    following suffixes:
    SI: KB, MB, GB, TB (e.g., 1KB = 1 * 1000 bytes)
    IEC: KiB, MiB, GiB, TiB, or K, M, G, T (e.g., 1K or 1KiB = 1 * 1024 bytes)
    Multipliers follow SI and IEC unit standards.
    Suffixes are case-insensitive.

  -b, -base &lt;PATH&gt;
    Set the base path for output files and also set their filename.
    For multiple output files, an _N suffix is added, where N is an incrementing
    number starting from 1.
    
  -d, -dir &lt;PATH&gt;
    Set the destination directory for output files.
    Can be used multiple times to set multiple directories.
    The base path is combined with each specified directory (dir + base path).
     
  -t, timeout &lt;TIME&gt;
    Set the HTTP connection timeout. TIME is an integer (representing seconds) and can
    be followed by the suffix s, m, or h for seconds, minutes, and hours, respectively
    (e.g., -t 1h2m3s). The default is 0, meaning no timeout.

  -H, -header &lt;HEADER_NAME:VALUE&gt;
    Set or add an HTTP header.
    Can be used multiple times to set or add multiple headers. The Range header is
    ignored in multipart web downloads.
    HEADER_NAME is case-insensitive.
  
  -V, -version
    Display version information.
     
  -h, -help
    Display this help message.
  
  -f
    Override the soft limit (128) on the total number of output files.
    Also enable quiet mode.

  -q
    Enable quiet mode.

  -x 
    Disable HTTP Keep-Alive or connection reuse. This ensures a separate connection
    per output file in multipart web downloads.
    
  -z
    Reset files with an initial state of [completed], [resume], or [broken]
    to [new].
    
  -C
    Reset files with an initial [completed] state to [new].
    
  -B
    Reset files with an initial [broken] state to [new].
    
  -R
    Reset files with an initial [resume] state to [new].

Output File States:
    File states are based on the initial size of files and may change during or after
    the download. States can also be affected by I/O operation errors and the file scope,
    which determines the maximum size a file can reach.

    [new]           File with initial 0 size.
    [resume]        File with initial size greater than 0 and is within file scope.
    [completed]     File that has reached its maximum size.
    [broken]        File that exceeds the maximum size or has an I/O operation error.
    [unknown]       File with a scope that cannot be determined.

    A file with the [unknown] state is always truncated to 0 size on every run with the
    same arguments. This state occurs when a web server does not support multipart
    downloading. </pre>
</details>

## Contributing
Contributions are welcome from anyone interested in helping improve partdec. If you’d like to contribute:

* Create a [GitHub pull request](https://github.com/cjijcb/partdec/pulls) if you find any bugs, and provide clear descriptions.
* Create a [GitHub issue](https://github.com/cjijcb/partdec/issues) to discuss improvement suggestions.

## License
Copyright (C) 2024 Carlo Jay I. Jacaba

partdec is licensed under the Apache License, Version 2.0. See the [LICENSE](https://github.com/cjijcb/partdec/blob/master/LICENSE) file for details.
