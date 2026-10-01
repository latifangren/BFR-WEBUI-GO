package shannon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"bfr-webui-go/internal/logger"
)

var (
	caEditorMu        sync.RWMutex
	currentDocument   *ComboDocument
	currentFilePath   string
)

// Standard search directories for Shannon LTE CA combos on Google Pixel
var knownCAPaths = []string{
	"/vendor/firmware/stable_modem/uecap",
	"/vendor/firmware/uecap",
	"/vendor/etc/carrier_config",
	"/data/adb/modules/bfr_webui_go/ca_combos",
	"/sdcard/Download",
}

// GetCAEditorStatus returns capability, detected files, and current loaded state.
func GetCAEditorStatus() CAEditorStatus {
	caEditorMu.RLock()
	defer caEditorMu.RUnlock()

	st := GetStatus()
	model := strings.TrimSpace(st.DeviceModel)
	soc := strings.ToUpper(strings.TrimSpace(st.SoC))

	// Evaluate compatibility across Google Tensor devices (Pixel 6 through Pixel 10)
	isPixel6 := strings.Contains(strings.ToLower(model), "pixel 6") || strings.Contains(soc, "GS101") || strings.Contains(soc, "WHITECHAPEL")
	isPixel7 := strings.Contains(strings.ToLower(model), "pixel 7") || strings.Contains(soc, "GS201") || strings.Contains(soc, "CLOUDRIPPER")
	isPixel8 := strings.Contains(strings.ToLower(model), "pixel 8") || strings.Contains(soc, "ZUMA") && !strings.Contains(soc, "PRO")
	isPixel9 := strings.Contains(strings.ToLower(model), "pixel 9") || strings.Contains(soc, "ZUMAPRO") || strings.Contains(soc, "G4")
	isPixel10 := strings.Contains(strings.ToLower(model), "pixel 10") || strings.Contains(soc, "LAGUNA") || strings.Contains(soc, "G5")

	tensorGen := "Google Tensor Shannon Modem"
	if isPixel10 {
		tensorGen = "Tensor G5 (Pixel 10)"
	} else if isPixel9 {
		tensorGen = "Tensor G4 (Pixel 9)"
	} else if isPixel8 {
		tensorGen = "Tensor G3 (Pixel 8)"
	} else if isPixel7 {
		tensorGen = "Tensor G2 (Pixel 7)"
	} else if isPixel6 {
		tensorGen = "Tensor G1 (Pixel 6)"
	}

	supported := st.IsShannon || isPixel6 || isPixel7 || isPixel8 || isPixel9 || isPixel10

	var warnMsg string
	if !supported {
		warnMsg = fmt.Sprintf("Notice: Device '%s' (%s) does not match a Google Tensor device (Pixel 6–10). You can still analyze and visualize uploaded .binarypb capability files.", model, tensorGen)
	}


	detected := discoverCAPaths()

	comboCount := 0
	var ver int32
	var mask uint32
	if currentDocument != nil {
		comboCount = len(currentDocument.Combos)
		ver = currentDocument.Version
		mask = currentDocument.Bitmask
	}

	return CAEditorStatus{
		SupportedDevice: supported,
		DeviceModel:     model,
		TensorGen:       tensorGen,
		DetectedPaths:   detected,
		ActiveFilePath:  currentFilePath,
		ComboCount:      comboCount,
		Version:         ver,
		Bitmask:         mask,
		WarningMessage:  warnMsg,
	}
}

func discoverCAPaths() []string {
	found := make([]string, 0, 8)
	for _, dir := range knownCAPaths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".binarypb") {
				found = append(found, filepath.Join(dir, e.Name()))
			}
		}
	}
	return found
}

// LoadCAFile loads a .binarypb file from the filesystem.
func LoadCAFile(path string) (*ComboDocument, error) {
	caEditorMu.Lock()
	defer caEditorMu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	doc, err := ParseShannonLteUECap(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ShannonLteUECap protobuf: %w", err)
	}

	currentDocument = doc
	currentFilePath = path
	logger.Get().Infof("ShannonCA", "Loaded %d combos from %s", len(doc.Combos), path)
	return doc, nil
}

// LoadCARawBytes parses a raw .binarypb uploaded by the user.
func LoadCARawBytes(name string, data []byte) (*ComboDocument, error) {
	caEditorMu.Lock()
	defer caEditorMu.Unlock()

	doc, err := ParseShannonLteUECap(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ShannonLteUECap protobuf: %w", err)
	}

	currentDocument = doc
	currentFilePath = name
	return doc, nil
}

// SaveCAFile saves the current document to disk.
func SaveCAFile(path string) error {
	caEditorMu.RLock()
	doc := currentDocument
	caEditorMu.RUnlock()

	if doc == nil {
		return fmt.Errorf("no active CA combo document loaded")
	}

	bytes, err := EncodeShannonLteUECap(doc)
	if err != nil {
		return fmt.Errorf("failed to encode protobuf: %w", err)
	}

	// Create backup first if file already exists
	if _, err := os.Stat(path); err == nil {
		bakPath := path + ".bak"
		_ = os.Rename(path, bakPath)
	}

	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write binarypb: %w", err)
	}

	caEditorMu.Lock()
	currentFilePath = path
	caEditorMu.Unlock()

	logger.Get().Infof("ShannonCA", "Saved %d combos to %s", len(doc.Combos), path)
	return nil
}

// GetCurrentDocument returns the active in-memory document.
func GetCurrentDocument() *ComboDocument {
	caEditorMu.RLock()
	defer caEditorMu.RUnlock()
	return currentDocument
}

// UpdateCurrentDocument updates the in-memory combos.
func UpdateCurrentDocument(combos []Combo) *ComboDocument {
	caEditorMu.Lock()
	defer caEditorMu.Unlock()

	if currentDocument == nil {
		currentDocument = &ComboDocument{
			Version: 1,
			Bitmask: 0,
		}
	}

	// Recompute visual fields
	for i := range combos {
		combos[i].ConfIDs = MasksToConfIDs(combos[i].ConfigMaskLow, combos[i].ConfigMaskHigh)
		combos[i].ComboString = FormatComboString(combos[i].Components)
		combos[i].TotalStreams = CalculateTotalStreams(combos[i].Components)
	}

	currentDocument.Combos = combos
	return currentDocument
}

// ClearCADocument unloads and clears the active in-memory LTE CA document.
func ClearCADocument() {
	caEditorMu.Lock()
	defer caEditorMu.Unlock()
	currentDocument = nil
	currentFilePath = ""
	logger.Get().Infof("ShannonCA", "Unloaded active LTE CA document")
}

