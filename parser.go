package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	nmea "github.com/adrianmo/go-nmea"
)

func parseLogFile(path string) (points []trackPoint, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	var lastGGA *ggaFix
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "@") {
			continue
		}
		if isGGASentence(line) {
			fix, ok := parseGGA(line)
			if ok {
				lastGGA = &fix
			}
			continue
		}
		if isRMCSentence(line) {
			if lastGGA == nil {
				continue
			}
			point, ok, err := parseRMC(line, *lastGGA)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %w", path, lineNumber, err)
			}
			if ok {
				points = append(points, point)
			}
			lastGGA = nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return points, nil
}

func parseGGA(line string) (ggaFix, bool) {
	sentence, err := nmea.Parse(line)
	if err != nil {
		return ggaFix{}, false
	}
	gga, ok := sentence.(nmea.GGA)
	if !ok {
		return ggaFix{}, false
	}
	if gga.FixQuality == "" || gga.FixQuality == "0" {
		return ggaFix{}, false
	}
	return ggaFix{
		Time: gga.Time,
		Lat:  gga.Latitude,
		Lon:  gga.Longitude,
		Ele:  &gga.Altitude,
	}, true
}

func parseRMC(line string, fix ggaFix) (trackPoint, bool, error) {
	rawFields := nmeaPayloadFields(line)
	sentence, err := nmea.Parse(normalizeRMCLine(line))
	if err != nil {
		return trackPoint{}, false, err
	}
	rmc, ok := sentence.(nmea.RMC)
	if !ok {
		return trackPoint{}, false, nil
	}
	if rmc.Validity != "A" {
		return trackPoint{}, false, nil
	}

	timestamp, err := parseNMEADateTime(rmc.Date, bestNMEATime(fix.Time, rmc.Time))
	if err != nil {
		return trackPoint{}, false, err
	}

	return trackPoint{
		TrackName: timestamp.Format("20060102"),
		Time:      timestamp,
		Lat:       fix.Lat,
		Lon:       fix.Lon,
		Ele:       fix.Ele,
		Speed:     parseNMEASpeed(rmc, rawFields),
		Course:    parseNMEACourse(rmc, rawFields),
	}, true, nil
}

func isGGASentence(line string) bool {
	return strings.HasPrefix(line, "$GPGGA,") || strings.HasPrefix(line, "$GNGGA,")
}

func isRMCSentence(line string) bool {
	return strings.HasPrefix(line, "$GPRMC,") || strings.HasPrefix(line, "$GNRMC,")
}

func parseNMEASpeed(rmc nmea.RMC, rawFields []string) *float64 {
	if len(rawFields) <= 7 || rawFields[7] == "" {
		return nil
	}
	metersPerSecond := rmc.Speed * knotsToMeters
	return &metersPerSecond
}

func parseNMEACourse(rmc nmea.RMC, rawFields []string) *float64 {
	if len(rawFields) <= 8 || rawFields[8] == "" {
		return nil
	}
	return &rmc.Course
}

func bestNMEATime(ggaTime, rmcTime nmea.Time) nmea.Time {
	if ggaTime.Valid {
		return ggaTime
	}
	return rmcTime
}

func parseNMEADateTime(date nmea.Date, clock nmea.Time) (time.Time, error) {
	timestamp := nmea.DateTime(2000, date, clock)
	if timestamp.IsZero() {
		return time.Time{}, fmt.Errorf("invalid NMEA date/time")
	}
	return timestamp, nil
}

func normalizeRMCLine(line string) string {
	fields := nmeaPayloadFields(line)
	if len(fields) == 0 {
		return line
	}
	normalized := append([]string(nil), fields...)
	for len(normalized) <= 11 {
		normalized = append(normalized, "")
	}
	if normalized[7] == "" {
		normalized[7] = "0"
	}
	if normalized[8] == "" {
		normalized[8] = "0"
	}
	if normalized[10] == "" {
		normalized[10] = "0"
	}
	if normalized[11] == "" {
		normalized[11] = "E"
	}
	body := strings.Join(normalized, ",")
	return "$" + body + "*" + nmea.Checksum(body)
}

func nmeaPayloadFields(line string) []string {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "$") {
		line = line[1:]
	}
	if i := strings.IndexByte(line, '*'); i >= 0 {
		line = line[:i]
	}
	if line == "" {
		return nil
	}
	return strings.Split(line, ",")
}
