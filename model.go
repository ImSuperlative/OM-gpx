package main

import (
	"time"

	nmea "github.com/adrianmo/go-nmea"
)

const (
	version        = "0.1.0"
	knotsToMeters  = 0.514444
	gpxCreator     = "OM SYSTEM GPX https://github.com/ImSuperlative/OM-gpx"
	gpxNamespace   = "http://www.topografix.com/GPX/1/1"
	gpxTPXNS       = "http://www.garmin.com/xmlschemas/TrackPointExtension/v2"
	gpxGarminNS    = "http://www.garmin.com/xmlschemas/GpxExtensions/v3"
	gpxSchemaValue = "http://www.topografix.com/GPX/1/1 http://www.topografix.com/GPX/1/1/gpx.xsd http://www.garmin.com/xmlschemas/TrackPointExtension/v2 http://www.garmin.com/xmlschemas/TrackPointExtensionv2.xsd http://www.garmin.com/xmlschemas/GpxExtensions/v3 http://www.garmin.com/xmlschemas/GpxExtensionsv3.xsd"
)

type trackPoint struct {
	TrackName string
	Time      time.Time
	Lat       float64
	Lon       float64
	Ele       *float64
	Speed     *float64
	Course    *float64
}

type ggaFix struct {
	Time nmea.Time
	Lat  float64
	Lon  float64
	Ele  *float64
}
