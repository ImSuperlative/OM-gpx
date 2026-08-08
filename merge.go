package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// duplicateSuffix matches the " 2" that gets appended to a repeated recording's
// name, e.g. the "20260619 2" stem of "20260619 2.log".
var duplicateSuffix = regexp.MustCompile(`^(.*) (\d+)$`)

// companionSNSPath locates the .sns recorded alongside a .log. OI.Share names
// repeated recordings inconsistently: the .log keeps its extension last
// ("20260619 2.log") while the .sns does not ("20260619.sns 2"), so both
// spellings are tried.
func companionSNSPath(logPath string) (string, bool) {
	for _, candidate := range companionSNSCandidates(logPath) {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

func companionSNSCandidates(logPath string) []string {
	stem := strings.TrimSuffix(logPath, filepath.Ext(logPath))
	candidates := []string{stem + ".sns"}
	if match := duplicateSuffix.FindStringSubmatch(stem); match != nil {
		candidates = append(candidates, match[1]+".sns "+match[2])
	}
	return candidates
}

// mergeSensorData copies .sns readings onto the GPS fixes they belong to.
//
// Fixes carry fractional seconds while $OMTIM has whole-second resolution, so
// matching happens per whole second. A single second can hold several fixes and
// several sensor records; the Nth fix in a second takes the Nth record in that
// second, which keeps distinct readings distinct instead of handing every fix
// the same nearest record. A second with more fixes than records reuses the
// last record, and a fix with no record for its second keeps nil sensor fields.
func mergeSensorData(points []trackPoint, sensors []sensorPoint) []trackPoint {
	if len(points) == 0 || len(sensors) == 0 {
		return points
	}

	bySecond := make(map[int64][]sensorPoint, len(sensors))
	for _, sensor := range sensors {
		second := sensor.Time.Unix()
		bySecond[second] = append(bySecond[second], sensor)
	}

	consumed := make(map[int64]int, len(bySecond))
	merged := make([]trackPoint, len(points))
	for i, point := range points {
		merged[i] = point

		second := point.Time.Unix()
		records := bySecond[second]
		if len(records) == 0 {
			continue
		}
		index := consumed[second]
		if index >= len(records) {
			index = len(records) - 1
		}
		consumed[second] = index + 1

		sensor := records[index]
		merged[i].PressureHPA = sensor.PressureHPA
		merged[i].BarometricAltitude = sensor.BarometricAltitude
		merged[i].AccelerationXMG = sensor.AccelerationXMG
		merged[i].AccelerationYMG = sensor.AccelerationYMG
		merged[i].AccelerationZMG = sensor.AccelerationZMG
	}
	return merged
}
