# revio-copy

A CLI tool for processing PacBio Revio sequencing data.

[![Build and Release](https://github.com/schnurbe/revio-copy/actions/workflows/build.yml/badge.svg)](https://github.com/schnurbe/revio-copy/actions/workflows/build.yml)

## Installation

### Download Pre-built Binaries

You can download the latest pre-built binaries for your platform from the [Releases](https://github.com/schnurbe/revio-copy/releases) page.

### Prerequisites

- rclone - Required for file copying operations with checksum verification
  - [Installation instructions](https://rclone.org/install/)

## Building from source

### Prerequisites

- Go 1.21 or later
- rclone (for running the application)

### Steps

```bash
git clone https://github.com/schnurbe/revio-copy.git
cd revio-copy
go build -o revio-copy
```

## Usage

```bash
# Process a specific run with file identification
./revio-copy process /path/to/runs --output /path/to/output --run "Run_Name"

# Process with interactive run selection
./revio-copy process /path/to/runs --output /path/to/output

# Show help
./revio-copy --help
```

## License

[MIT License](LICENSE)