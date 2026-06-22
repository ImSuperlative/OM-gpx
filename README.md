# om-gpx

Convert OM System / Olympus OI.Share GPS logs to GPX for photo geotagging tools such as Lightroom.

The tool reads OI.Share `.log` files, converts the NMEA GPS fixes to GPX 1.1 track points, and writes one GPX track per day. If a matching `.sns` file exists next to the `.log`, `om-gpx` detects it automatically. `.sns` files are sensor-only files and are not used for GPX location output.

## Usage

```bash
om-gpx [options] input [input...]
```

Inputs may be individual `.log` files or directories containing `.log` files.
Options may be placed before or after inputs.

Options:

```text
-o, --output string
    write GPX to this file
-v, -version, --version
    print version and exit
```

## Examples

Convert a folder of OI.Share files:

```bash
om-gpx /path/to/OI.Share
```

Convert one log and choose the output path:

```bash
om-gpx sample/20260620.log -o 20260620.gpx
```

## Install Globally

From this repository:

```bash
./deploy.sh
```

By default this builds the binary and installs it as:

```text
/usr/local/bin/om-gpx
```

You can override the install directory:

```bash
INSTALL_DIR="$HOME/bin" ./deploy.sh
```

## Development

Run tests:

```bash
go test ./...
```
