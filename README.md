# om-gpx

Convert OM System / Olympus OI.Share GPS logs to GPX for photo geotagging tools such as Lightroom, and to GeoJSON for archival.

The tool reads OI.Share `.log` files and converts the NMEA GPS fixes to GPX 1.1 track points. Each `.log` produces its own GPX and GeoJSON, so a folder of recordings converts in one pass.

If a `.sns` sensor file was recorded alongside a `.log`, it is paired automatically and its readings — atmospheric pressure, barometric altitude and acceleration — are added to the **GeoJSON only**. The GPX is unaffected, keeping it clean and predictable for Lightroom.

A `.log` converts perfectly well on its own; a missing `.sns` is a warning, not an error.

## Usage

```bash
om-gpx [options] input [input...]
```

Inputs may be individual `.log` files or directories containing `.log` files.
Options may be placed before or after inputs.

Options:

```text
-o, --output string
    single input only: write outputs using this path as the stem
-d, --out-dir string
    write all outputs into this directory
    --gpx
    write only GPX
    --geojson
    write only GeoJSON
-n, --dry-run
    print planned pairings and outputs without writing
-v, -version, --version
    print version and exit
```

Passing neither `--gpx` nor `--geojson` writes both.

## Output names

Outputs are named after the log, with spaces replaced by underscores:

```text
20260619.log     + 20260619.sns    ->  20260619.gpx     20260619.geojson
20260619 2.log   + 20260619.sns 2  ->  20260619_2.gpx   20260619_2.geojson
20260619 3.log   + 20260619.sns 3  ->  20260619_3.gpx   20260619_3.geojson
```

If two inputs would produce the same output name, `om-gpx` stops before writing anything and names both.

## Examples

Convert a folder of OI.Share files in place:

```bash
om-gpx /path/to/OI.Share
```

Convert a folder into a separate output directory:

```bash
om-gpx --out-dir ~/tracks /path/to/OI.Share
```

Check which `.sns` will be paired with which `.log` before converting:

```bash
om-gpx --dry-run /path/to/OI.Share
```

Convert one log and choose the output path:

```bash
om-gpx sample/20260620.log -o 20260620.gpx
```

Produce only GPX for Lightroom:

```bash
om-gpx --gpx /path/to/OI.Share
```

## SNS pairing

OI.Share names repeated recordings inconsistently — the `.log` keeps its extension last (`20260619 2.log`) while the `.sns` does not (`20260619.sns 2`). Both spellings are checked, in that order.

Sensor records carry whole-second timestamps while GPS fixes carry fractional seconds, so readings are matched per whole second. Where several fixes share a second, the *n*th fix takes the *n*th sensor record of that second, which keeps distinct readings distinct.

A `$OMPRE` barometric altitude of `0.0` means the device reported no barometric fix; it is treated as absent rather than as sea level. Pressure from the same record is still used.

## GeoJSON

One `Point` feature per GPS fix. GPS altitude is the third coordinate; sensor readings are properties with explicit units:

```json
{
  "type": "Feature",
  "geometry": {
    "type": "Point",
    "coordinates": [11.80516856666667, 57.627842433333335, 2.262905]
  },
  "properties": {
    "time": "2026-06-19T10:44:40Z",
    "trackName": "20260619",
    "speedMs": 0.38611079976,
    "courseDeg": 188.4375,
    "pressureHpa": 1016,
    "accelerationXMg": 84.2,
    "accelerationYMg": 858.1,
    "accelerationZMg": 70.8
  }
}
```

A `barometricAltitudeM` property joins these when the recording contains one.

Readings that were not recorded are omitted rather than written as `null`, so a missing key means "no data" and never "zero". The collection also carries `creator` and a `source` naming the `.log` and `.sns` it was built from.

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
