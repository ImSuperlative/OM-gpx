package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// parseSNSFile reads an OI.Share .sns sensor recording. Records are groups of
// $OMTIM/$OMPRE/$OMACC sentences; a $OMTIM opens a new record. Sentences that
// cannot be understood are skipped rather than failing the file, so a single
// corrupt line never costs the whole recording.
func parseSNSFile(path string) (points []sensorPoint, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	var current *sensorPoint
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "@") {
			continue
		}
		fields := nmeaPayloadFields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "OMTIM":
			if current != nil {
				points = append(points, *current)
				current = nil
			}
			if timestamp, ok := parseOMTIM(fields); ok {
				current = &sensorPoint{Time: timestamp}
			}
		case "OMPRE":
			if current == nil {
				continue
			}
			current.PressureHPA = snsField(fields, 1)
			current.BarometricAltitude = snsNonZeroField(fields, 2)
		case "OMACC":
			if current == nil {
				continue
			}
			current.AccelerationXMG = snsField(fields, 1)
			current.AccelerationYMG = snsField(fields, 2)
			current.AccelerationZMG = snsField(fields, 3)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if current != nil {
		points = append(points, *current)
	}
	return points, nil
}

// parseOMTIM reads "$OMTIM,YYYYMMDD,HHMMSS". OI.Share records these in the same
// clock as the .log NMEA sentences, so they are treated as UTC.
func parseOMTIM(fields []string) (time.Time, bool) {
	if len(fields) < 3 {
		return time.Time{}, false
	}
	timestamp, err := time.ParseInLocation("20060102150405", fields[1]+fields[2], time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return timestamp, true
}

func snsField(fields []string, index int) *float64 {
	if index >= len(fields) || fields[index] == "" {
		return nil
	}
	value, err := strconv.ParseFloat(fields[index], 64)
	if err != nil {
		return nil
	}
	return &value
}

// snsNonZeroField is snsField for the barometric altitude, which OI.Share
// writes as 0.0 when the device reports no barometric fix. Zero means absent,
// not sea level.
func snsNonZeroField(fields []string, index int) *float64 {
	value := snsField(fields, index)
	if value == nil || *value == 0 {
		return nil
	}
	return value
}
