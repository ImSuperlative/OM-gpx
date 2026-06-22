package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func writeGPX(path string, points []trackPoint) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	return writeGPXTo(file, points)
}

func writeGPXTo(writer io.Writer, points []trackPoint) error {
	_, err := io.WriteString(writer, "<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n")
	if err != nil {
		return err
	}

	encodedGPX, err := encodeIndentedGPX(points)
	if err != nil {
		return err
	}
	output := trimRootIndent(encodedGPX)
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	_, err = io.WriteString(writer, output)
	return err
}

func encodeIndentedGPX(points []trackPoint) (encoded string, err error) {
	var buffer strings.Builder
	encoder := xml.NewEncoder(&buffer)
	defer func() {
		if flushErr := encoder.Flush(); err == nil && flushErr != nil {
			err = flushErr
		}
	}()

	encoder.Indent("", "  ")
	if err := encoder.Encode(toGPX(points)); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

type trackGroup struct {
	Name   string
	Points []trackPoint
}

func groupByTrack(points []trackPoint) []trackGroup {
	var groups []trackGroup
	for _, point := range points {
		if len(groups) == 0 || groups[len(groups)-1].Name != point.TrackName {
			groups = append(groups, trackGroup{Name: point.TrackName})
		}
		groups[len(groups)-1].Points = append(groups[len(groups)-1].Points, point)
	}
	return groups
}

func toGPX(points []trackPoint) gpxDocument {
	doc := gpxDocument{
		Version:         "1.1",
		Creator:         gpxCreator,
		Xmlns:           gpxNamespace,
		XmlnsXSI:        "http://www.w3.org/2001/XMLSchema-instance",
		SchemaLocation:  gpxSchemaValue,
		XmlnsTrackPoint: gpxTPXNS,
		XmlnsGarmin:     gpxGarminNS,
	}
	for _, group := range groupByTrack(points) {
		track := gpxTrack{Name: group.Name}
		for _, point := range group.Points {
			track.Segment.Points = append(track.Segment.Points, toGPXPoint(point))
		}
		doc.Tracks = append(doc.Tracks, track)
	}
	return doc
}

func toGPXPoint(point trackPoint) gpxTrackPoint {
	gpxPoint := gpxTrackPoint{
		Lat:  formatFloat(point.Lat),
		Lon:  formatFloat(point.Lon),
		Time: formatTime(point.Time),
	}
	if point.Ele != nil {
		ele := fmt.Sprintf("%.1f", *point.Ele)
		gpxPoint.Ele = &ele
	}
	if point.Speed != nil || point.Course != nil {
		ext := gpxExtensions{}
		if point.Speed != nil {
			speed := fmt.Sprintf("%.2f", *point.Speed)
			ext.TrackPoint.Speed = &speed
		}
		if point.Course != nil {
			course := fmt.Sprintf("%.6f", *point.Course)
			ext.TrackPoint.Course = &course
		}
		gpxPoint.Extensions = &ext
	}
	return gpxPoint
}

func formatFloat(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.9f", value), "0"), ".")
}

func formatTime(t time.Time) string {
	t = t.UTC()
	if t.Nanosecond() == 0 {
		return t.Format("2006-01-02T15:04:05Z")
	}
	return strings.TrimRight(t.Format("2006-01-02T15:04:05.999999999"), "0") + "Z"
}

func trimRootIndent(s string) string {
	lines := strings.SplitAfter(s, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.TrimPrefix(lines[i], "  ")
	}
	return strings.Join(lines, "")
}

type gpxDocument struct {
	XMLName         xml.Name   `xml:"gpx"`
	Version         string     `xml:"version,attr"`
	Creator         string     `xml:"creator,attr"`
	Xmlns           string     `xml:"xmlns,attr"`
	XmlnsXSI        string     `xml:"xmlns:xsi,attr"`
	SchemaLocation  string     `xml:"xsi:schemaLocation,attr"`
	XmlnsTrackPoint string     `xml:"xmlns:gpxtpx,attr"`
	XmlnsGarmin     string     `xml:"xmlns:gpxx,attr"`
	Tracks          []gpxTrack `xml:"trk"`
}

type gpxTrack struct {
	Name    string          `xml:"name"`
	Segment gpxTrackSegment `xml:"trkseg"`
}

type gpxTrackSegment struct {
	Points []gpxTrackPoint `xml:"trkpt"`
}

type gpxTrackPoint struct {
	Lat        string         `xml:"lat,attr"`
	Lon        string         `xml:"lon,attr"`
	Ele        *string        `xml:"ele,omitempty"`
	Time       string         `xml:"time"`
	Extensions *gpxExtensions `xml:"extensions,omitempty"`
}

type gpxExtensions struct {
	TrackPoint gpxTrackPointExtension `xml:"gpxtpx:TrackPointExtension"`
}

type gpxTrackPointExtension struct {
	Speed  *string `xml:"gpxtpx:speed,omitempty"`
	Course *string `xml:"gpxtpx:course,omitempty"`
}
