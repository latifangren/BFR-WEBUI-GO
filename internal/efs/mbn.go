//go:build android || linux

package efs

import (
	"fmt"
	"path"
	"strings"

	"bfr-webui-go/internal/logger"
)

// ReadMbnAutoselect reads autoselect flags from EFS item files.
func (s *EfsSession) ReadMbnAutoselect() (MbnAutoselectStatus, error) {
	status := MbnAutoselectStatus{}

	if data, err := s.ReadFile(EfsMcfgAutoselectSim1); err == nil && len(data) >= 1 {
		status.Sim1Mode = int(data[0])
		status.Sim1Auto = data[0] > 0
	}
	if data, err := s.ReadFile(EfsMcfgAutoselectSim2); err == nil && len(data) >= 1 {
		status.Sim2Mode = int(data[0])
		status.Sim2Auto = data[0] > 0
	}

	return status, nil
}

// WriteMbnAutoselect writes autoselect flags to EFS item files.
func (s *EfsSession) WriteMbnAutoselect(simSlot int, mode int, opts WriteOptions) error {
	targetPath := EfsMcfgAutoselectSim1
	if simSlot == 1 {
		targetPath = EfsMcfgAutoselectSim2
	}

	valByte := byte(mode)
	_, err := s.WriteFileWithOptions(targetPath, []byte{valByte}, 0644, opts)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", targetPath, err)
	}

	logger.Get().Infof("EFS", "Updated MCFG autoselect for SIM%d to mode %d", simSlot+1, mode)
	return nil
}

// ScanMbnDirectory recursively scans an EFS directory (e.g. /mcfg_sw/generic) for .mbn files.
func (s *EfsSession) ScanMbnDirectory(baseDir string, configType string) ([]MbnConfigItem, error) {
	var results []MbnConfigItem
	s.scanMbnRecursive(baseDir, baseDir, configType, &results, 0)
	return results, nil
}

func (s *EfsSession) scanMbnRecursive(currentDir string, rootDir string, configType string, results *[]MbnConfigItem, depth int) {
	if depth > 5 || len(*results) > 200 {
		return // Guard against deep hierarchies or massive file lists
	}

	entries, err := s.ListDir(currentDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		fullPath := path.Join(currentDir, entry.Name)
		if entry.Type == "dir" {
			s.scanMbnRecursive(fullPath, rootDir, configType, results, depth+1)
		} else if strings.HasSuffix(strings.ToLower(entry.Name), ".mbn") || entry.Type == "file" {
			rel := strings.TrimPrefix(fullPath, rootDir+"/")
			category := ""
			parts := strings.Split(rel, "/")
			if len(parts) > 1 {
				category = parts[0]
			}

			*results = append(*results, MbnConfigItem{
				Name:     entry.Name,
				Path:     fullPath,
				Type:     configType,
				Category: category,
				Size:     entry.Size,
			})
		}
	}
}
