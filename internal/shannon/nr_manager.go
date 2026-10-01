package shannon

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"bfr-webui-go/internal/logger"
)

var (
	nrEditorMu        sync.RWMutex
	currentNRDocument *NRComboDocument
	currentNRFilePath string
	currentNRBytes    []byte
)

// Standard search directories for Shannon 5G NR & EN-DC capabilities on Google Pixel devices
var knownNRPaths = []string{
	"/vendor/firmware/stable_modem/uecap",
	"/vendor/firmware/uecap",
	"/vendor/etc/carrier_config",
	"/data/adb/modules/bfr_webui_go/ca_combos",
	"/sdcard/Download",
}

//go:embed fixtures/sub6.binarypb
var demoSub6Bytes []byte

// LoadDemoNRDocument loads the built-in Pixel Sub-6 / EN-DC demo dataset.
func LoadDemoNRDocument() (*NRComboDocument, error) {
	return LoadNRCARawBytes("demo_pixel_sub6_endc.binarypb", demoSub6Bytes)
}


// NREditorStatus represents device compatibility and status for 5G NR / EN-DC capability.
type NREditorStatus struct {
	SupportedDevice bool     `json:"supported_device"`
	DeviceModel     string   `json:"device_model"`
	TensorGen       string   `json:"tensor_gen"`
	DetectedPaths   []string `json:"detected_paths"`
	ActiveFilePath  string   `json:"active_file_path,omitempty"`
	TotalCombos     int      `json:"total_combos"`
	EndcCount       int      `json:"endc_count"`
	NrcaCount       int      `json:"nrca_count"`
	NrdcCount       int      `json:"nrdc_count"`
	Version         uint32   `json:"version"`
	ID              int32    `json:"id"`
	WarningMessage  string   `json:"warning_message,omitempty"`
}

// GetNREditorStatus returns device information and active 5G NR / EN-DC document status.
func GetNREditorStatus() NREditorStatus {
	nrEditorMu.RLock()
	defer nrEditorMu.RUnlock()

	st := GetStatus()
	model := strings.TrimSpace(st.DeviceModel)
	soc := strings.ToUpper(strings.TrimSpace(st.SoC))

	// Tensor classification
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

	detected := discoverNRPaths()

	var total, endc, nrca, nrdc int
	var ver uint32
	var id int32

	if currentNRDocument != nil {
		total = currentNRDocument.TotalCombos
		endc = currentNRDocument.EndcCount
		nrca = currentNRDocument.NrcaCount
		nrdc = currentNRDocument.NrdcCount
		ver = currentNRDocument.Version
		id = currentNRDocument.ID
	}

	return NREditorStatus{
		SupportedDevice: supported,
		DeviceModel:     model,
		TensorGen:       tensorGen,
		DetectedPaths:   detected,
		ActiveFilePath:  currentNRFilePath,
		TotalCombos:     total,
		EndcCount:       endc,
		NrcaCount:       nrca,
		NrdcCount:       nrdc,
		Version:         ver,
		ID:              id,
		WarningMessage:  warnMsg,
	}
}

func discoverNRPaths() []string {
	found := make([]string, 0, 8)
	for _, dir := range knownNRPaths {
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

// LoadNRCAFile loads a 5G NR / EN-DC .binarypb file from the filesystem.
func LoadNRCAFile(path string) (*NRComboDocument, error) {
	nrEditorMu.Lock()
	defer nrEditorMu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	doc, err := ParseShannonNrUECap(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ShannonNrUECap protobuf: %w", err)
	}

	doc.SourceFile = path
	currentNRDocument = doc
	currentNRFilePath = path
	currentNRBytes = data
	logger.Get().Infof("ShannonNR", "Loaded %d combos (%d EN-DC, %d NR-CA, %d NR-DC) from %s",
		doc.TotalCombos, doc.EndcCount, doc.NrcaCount, doc.NrdcCount, path)
	return doc, nil
}

// LoadNRCARawBytes parses raw .binarypb bytes uploaded by the user.
func LoadNRCARawBytes(name string, data []byte) (*NRComboDocument, error) {
	nrEditorMu.Lock()
	defer nrEditorMu.Unlock()

	doc, err := ParseShannonNrUECap(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ShannonNrUECap protobuf: %w", err)
	}

	doc.SourceFile = name
	currentNRDocument = doc
	currentNRFilePath = name
	currentNRBytes = data
	logger.Get().Infof("ShannonNR", "Imported %d combos (%d EN-DC, %d NR-CA, %d NR-DC) from %s",
		doc.TotalCombos, doc.EndcCount, doc.NrcaCount, doc.NrdcCount, name)
	return doc, nil
}

// GetCurrentNRDocument returns the active in-memory 5G NR document.
func GetCurrentNRDocument() *NRComboDocument {
	nrEditorMu.RLock()
	defer nrEditorMu.RUnlock()
	return currentNRDocument
}

// GetCurrentNRBytes returns the raw bytes of the active document if available.
func GetCurrentNRBytes() ([]byte, string) {
	nrEditorMu.RLock()
	defer nrEditorMu.RUnlock()
	return currentNRBytes, currentNRFilePath
}

// ClearNRDocument unloads and clears the active in-memory 5G NR document.
func ClearNRDocument() {
	nrEditorMu.Lock()
	defer nrEditorMu.Unlock()
	currentNRDocument = nil
	currentNRFilePath = ""
	currentNRBytes = nil
	logger.Get().Infof("ShannonNR", "Unloaded active 5G NR / EN-DC document")
}

