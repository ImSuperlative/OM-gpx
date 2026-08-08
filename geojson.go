package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

// trackSource records which files a track was built from, so an archived
// GeoJSON says where its data came from.
type trackSource struct {
	Log string
	SNS string
}

func writeGeoJSON(path string, points []trackPoint, source trackSource) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	return writeGeoJSONTo(file, points, source)
}

func writeGeoJSONTo(writer io.Writer, points []trackPoint, source trackSource) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(toGeoJSON(points, source))
}

func toGeoJSON(points []trackPoint, source trackSource) geoJSONFeatureCollection {
	collection := geoJSONFeatureCollection{
		Type:     "FeatureCollection",
		Creator:  gpxCreator,
		Source:   toGeoJSONSource(source),
		Features: make([]geoJSONFeature, 0, len(points)),
	}
	for _, point := range points {
		collection.Features = append(collection.Features, toGeoJSONFeature(point))
	}
	return collection
}

func toGeoJSONSource(source trackSource) *geoJSONSource {
	if source.Log == "" && source.SNS == "" {
		return nil
	}
	result := &geoJSONSource{Log: filepath.Base(source.Log)}
	if source.SNS != "" {
		result.SNS = filepath.Base(source.SNS)
	}
	return result
}

func toGeoJSONFeature(point trackPoint) geoJSONFeature {
	coordinates := []float64{point.Lon, point.Lat}
	if point.GPSAltitude != nil {
		coordinates = append(coordinates, *point.GPSAltitude)
	}

	return geoJSONFeature{
		Type: "Feature",
		Geometry: geoJSONGeometry{
			Type:        "Point",
			Coordinates: coordinates,
		},
		Properties: geoJSONProperties{
			Time:                formatTime(point.Time),
			TrackName:           point.TrackName,
			SpeedMs:             point.Speed,
			CourseDeg:           point.Course,
			PressureHpa:         point.PressureHPA,
			BarometricAltitudeM: point.BarometricAltitude,
			AccelerationXMg:     point.AccelerationXMG,
			AccelerationYMg:     point.AccelerationYMG,
			AccelerationZMg:     point.AccelerationZMG,
		},
	}
}

// creator and source are GeoJSON foreign members, which RFC 7946 permits on a
// FeatureCollection.
type geoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Creator  string           `json:"creator"`
	Source   *geoJSONSource   `json:"source,omitempty"`
	Features []geoJSONFeature `json:"features"`
}

type geoJSONSource struct {
	Log string `json:"log"`
	SNS string `json:"sns,omitempty"`
}

type geoJSONFeature struct {
	Type       string            `json:"type"`
	Geometry   geoJSONGeometry   `json:"geometry"`
	Properties geoJSONProperties `json:"properties"`
}

type geoJSONGeometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

// Every optional reading is a pointer so that omitempty drops only genuinely
// absent values. A zero course means due north and must still be written.
type geoJSONProperties struct {
	Time                string   `json:"time"`
	TrackName           string   `json:"trackName"`
	SpeedMs             *float64 `json:"speedMs,omitempty"`
	CourseDeg           *float64 `json:"courseDeg,omitempty"`
	PressureHpa         *float64 `json:"pressureHpa,omitempty"`
	BarometricAltitudeM *float64 `json:"barometricAltitudeM,omitempty"`
	AccelerationXMg     *float64 `json:"accelerationXMg,omitempty"`
	AccelerationYMg     *float64 `json:"accelerationYMg,omitempty"`
	AccelerationZMg     *float64 `json:"accelerationZMg,omitempty"`
}
