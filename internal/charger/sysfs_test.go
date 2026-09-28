package charger

import (
	"path/filepath"
	"testing"
)

func TestSysfsNativeIO(t *testing.T) {
	tempDir := t.TempDir()

	// 1. fileExistsAndWritable & writeSysfs
	testFile := filepath.Join(tempDir, "test_node")
	if fileExistsAndWritable("") {
		t.Errorf("expected empty path to not be writable")
	}
	if fileExistsAndWritable(testFile) {
		t.Errorf("expected non-existent file to not be writable")
	}

	if err := writeSysfs("", "1"); err == nil {
		t.Errorf("expected error writing to empty path")
	}

	if err := writeSysfs(testFile, "42"); err != nil {
		t.Fatalf("failed to write to sysfs node: %v", err)
	}

	if !fileExistsAndWritable(testFile) {
		t.Errorf("expected existing file to be writable")
	}

	// 2. readSysfs
	if _, err := readSysfs(""); err == nil {
		t.Errorf("expected error reading from empty path")
	}
	if _, err := readSysfs(filepath.Join(tempDir, "non_existent")); err == nil {
		t.Errorf("expected error reading from non-existent file")
	}

	text, err := readSysfs(testFile)
	if err != nil {
		t.Fatalf("failed to read sysfs node: %v", err)
	}
	if text != "42" {
		t.Errorf("expected '42', got '%s'", text)
	}

	// 3. readSysfsInt
	intVal, err := readSysfsInt(testFile)
	if err != nil {
		t.Fatalf("failed to parse int: %v", err)
	}
	if intVal != 42 {
		t.Errorf("expected 42, got %d", intVal)
	}

	// 4. readSysfsFloat
	floatFile := filepath.Join(tempDir, "float_node")
	if err := writeSysfs(floatFile, "123.45"); err != nil {
		t.Fatalf("failed to write float node: %v", err)
	}

	floatVal, err := readSysfsFloat(floatFile)
	if err != nil {
		t.Fatalf("failed to parse float: %v", err)
	}
	if floatVal != 123.45 {
		t.Errorf("expected 123.45, got %f", floatVal)
	}

	// 5. Invalid parse errors
	invalidFile := filepath.Join(tempDir, "invalid_node")
	_ = writeSysfs(invalidFile, "not_a_number")

	if _, err := readSysfsInt(invalidFile); err == nil {
		t.Errorf("expected error parsing invalid int")
	}
	if _, err := readSysfsFloat(invalidFile); err == nil {
		t.Errorf("expected error parsing invalid float")
	}
}

func TestInferOnOffValues(t *testing.T) {
	onVal, offVal, isBypass := inferOnOffValues("/sys/class/power_supply/main/force_main_fcc")
	if onVal != "550000" || offVal != "0" || isBypass {
		t.Errorf("expected 550000, 0, false for force_main_fcc, got %s, %s, %v", onVal, offVal, isBypass)
	}

	onVal, offVal, isBypass = inferOnOffValues("/sys/class/power_supply/main/force_main_icl")
	if onVal != "550000" || offVal != "0" || isBypass {
		t.Errorf("expected 550000, 0, false for force_main_icl, got %s, %s, %v", onVal, offVal, isBypass)
	}
}

