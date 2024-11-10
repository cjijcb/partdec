# TODO



# Command-Line Usage
```
Usage: partdec [OPTIONS]... <URL|LOCAL PATH>
Seamlessly split files from the web or local storage. 

Options:
  -p, -part <N>
    Split the file into N parts.
    If N is zero or less, it defaults to 1.
    If -s/-size is used, this option is ignored.

  -s, -size <SIZE>
    Split the file into parts based on SIZE.
    SIZE is an integer (representing byte size) and can be followed by one of the
    following suffixes:
    SI: KB, MB, GB, TB (e.g., 1KB = 1 * 1000 bytes)
    IEC: KiB, MiB, GiB, TiB, or K, M, G, T (e.g., 1K or 1KiB = 1 * 1024 bytes)
    Multipliers follow SI and IEC unit standards.
    Suffixes are case-insensitive.

  -b, -base <PATH>
    Set the base path for output files and also rename them.
    For multiple output files, an _N suffix is added, where N is an incrementing
    number starting from 1.
    
  -d, -dir <PATH>
    Set the destination directory for output files.
    Can be used multiple times to set multiple directories.
    The base path is combined with each specified directory (dir + base path).
     
  -t, timeout <TIME>
    Set the HTTP connection timeout. TIME is an integer (representing seconds) and can
    be followed by the suffix s, m, or h for seconds, minutes, and hours, respectively
    (e.g., -t 1h2m3s). The default is 0, meaning no timeout.

  -H, -header <HEADER_NAME:VALUE>
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
    downloading. 
```

# Combinig Spit Files


