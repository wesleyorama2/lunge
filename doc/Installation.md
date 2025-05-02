# Installing Lunge

This guide covers the different ways to install Lunge on your system.

## Prerequisites

- Go 1.18 or higher (for building from source)
- Git (for cloning the repository)

## Installation Methods

### Using Go Install

If you have Go installed, you can install Lunge directly using:

```bash
go install github.com/yourusername/lunge/cmd/lunge@latest
```

This will download, compile, and install the latest version of Lunge to your `$GOPATH/bin` directory.

### Building from Source

1. Clone the repository:

```bash
git clone https://github.com/yourusername/lunge.git
cd lunge
```

2. Build the binary:

```bash
go build -o lunge ./cmd/lunge
```

3. Move the binary to a location in your PATH (optional):

```bash
# Linux/macOS
sudo mv lunge /usr/local/bin/

# Or add to your user bin directory
mv lunge ~/bin/
```

### Using Pre-built Binaries

1. Download the appropriate binary for your platform from the [releases page](https://github.com/yourusername/lunge/releases).

2. Extract the archive:

```bash
# Linux/macOS
tar -xzf lunge_<version>_<platform>.tar.gz

# Windows
unzip lunge_<version>_windows_amd64.zip
```

3. Move the binary to a location in your PATH (optional):

```bash
# Linux/macOS
sudo mv lunge /usr/local/bin/

# Windows
# Move to a directory in your PATH
```

## Verifying Installation

To verify that Lunge is installed correctly, run:

```bash
lunge --version
```

This should display the version of Lunge that you have installed.

## Next Steps

Once you have Lunge installed, check out the [Getting Started](./Getting-Started.md) guide to learn how to use it.