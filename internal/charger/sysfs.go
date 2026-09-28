package charger

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// readSysfs reads the full text of a sysfs/proc node using pure native Go os.ReadFile.
func readSysfs(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty sysfs path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// readSysfsInt parses an integer from a sysfs path.
func readSysfsInt(path string) (int, error) {
	text, err := readSysfs(path)
	if err != nil {
		return 0, err
	}
	val, err := strconv.Atoi(text)
	if err != nil {
		return 0, err
	}
	return val, nil
}

// readSysfsFloat parses a float64 from a sysfs path.
func readSysfsFloat(path string) (float64, error) {
	text, err := readSysfs(path)
	if err != nil {
		return 0, err
	}
	val, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

// writeSysfs writes a control value to a sysfs or proc node using pure native Go os.WriteFile.
func writeSysfs(path string, val string) error {
	if path == "" {
		return fmt.Errorf("empty sysfs path")
	}
	return os.WriteFile(path, []byte(val+"\n"), 0644)
}

// fileExistsAndWritable checks whether the path exists and can be written to using pure native Go os.OpenFile.
func fileExistsAndWritable(path string) bool {
	if path == "" {
		return false
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
