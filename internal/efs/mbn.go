//go:build android || linux

package efs

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
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
// Candidate base directories on Android OS filesystem where Qualcomm carrier MBNs live
var OsMcfgCandidatePaths = []string{
	"/data/adb/modules/bfr_webui_go/custom_mbn",
	"/data/adb/bfr_webui_go/data/custom_mbn",
	"/vendor/rfs/msm/mpss/readonly/vendor/mbn",
	"/vendor/mbn",
	"/vendor/firmware_mnt/image/modem_pr/mcfg/configs",
	"/firmware/image/modem_pr/mcfg/configs",
	"/data/vendor/modem_config",
}

// DiscoverMbnConfigs scans the host OS filesystem (RFS/vendor/firmware partitions) first,
// and falls back to modem EFS if no local files are discovered and s != nil.
func DiscoverMbnConfigs(s *EfsSession, configType string) []MbnConfigItem {
	var results []MbnConfigItem
	seen := make(map[string]bool)

	// 1. Scan OS filesystem
	subDirName := "mcfg_" + configType // e.g. "mcfg_sw" or "mcfg_hw"
	txtIndexName := "mbn_" + configType + ".txt"

	for _, basePath := range OsMcfgCandidatePaths {
		targetDir := filepath.Join(basePath, subDirName)
		isDirectCustom := false
		if _, err := os.Stat(targetDir); err != nil {
			if strings.Contains(basePath, "custom_mbn") {
				if _, err := os.Stat(basePath); err == nil {
					targetDir = basePath
					isDirectCustom = true
				} else {
					continue
				}
			} else {
				continue
			}
		}

		if isDirectCustom {
			_ = filepath.WalkDir(targetDir, func(p string, d fs.DirEntry, err error) error {
				if err != nil || d == nil || d.IsDir() {
					return nil
				}
				if strings.HasSuffix(strings.ToLower(d.Name()), ".mbn") {
					rel, _ := filepath.Rel(basePath, p)
					rel = filepath.ToSlash(rel)
					if !seen[p] && !seen[rel] {
						seen[p] = true
						seen[rel] = true
						fi, _ := d.Info()
						var sz int64 = 0
						if fi != nil {
							sz = fi.Size()
						}
						// Use folder name or parent as name
						parent := filepath.Base(filepath.Dir(p))
						name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
						if parent != "." && parent != "custom_mbn" && parent != "" {
							name = fmt.Sprintf("%s (%s)", parent, name)
						}
						results = append(results, MbnConfigItem{
							Name:     name,
							Path:     p,
							Type:     configType,
							Category: "Custom",
							Size:     sz,
						})
					}
				}
				return nil
			})
			continue
		}
		// A. Check index file (e.g. mbn_sw.txt or mbn_sw_merged.txt)
		indexPath := filepath.Join(targetDir, txtIndexName)
		mergedIndexPath := filepath.Join(targetDir, "mbn_"+configType+"_merged.txt")
		if _, err := os.Stat(mergedIndexPath); err == nil {
			indexPath = mergedIndexPath
		}

		if idxData, err := os.ReadFile(indexPath); err == nil && len(idxData) > 0 {
			lines := strings.Split(string(idxData), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				cleanPath := strings.TrimPrefix(line, subDirName+"/")
				absPath := filepath.Join(basePath, line)
				var fileSize int64 = 0
				if fi, err := os.Stat(absPath); err == nil {
					fileSize = fi.Size()
				}

				category, friendlyName := parseMbnHierarchy(cleanPath)
				if !seen[line] {
					seen[line] = true
					results = append(results, MbnConfigItem{
						Name:     friendlyName,
						Path:     line,
						Type:     configType,
						Category: category,
						Size:     fileSize,
					})
				}
			}
		}

		// B. Recursive filesystem scan in targetDir/generic
		genericDir := filepath.Join(targetDir, "generic")
		if _, err := os.Stat(genericDir); err == nil {
			_ = filepath.WalkDir(genericDir, func(p string, d fs.DirEntry, err error) error {
				if err != nil || d == nil || d.IsDir() {
					return nil
				}
				if strings.HasSuffix(strings.ToLower(d.Name()), ".mbn") {
					relToSubDir, err := filepath.Rel(basePath, p)
					if err == nil {
						relToSubDir = filepath.ToSlash(relToSubDir)
						if !seen[relToSubDir] {
							seen[relToSubDir] = true
							cleanPath := strings.TrimPrefix(relToSubDir, subDirName+"/")
							category, friendlyName := parseMbnHierarchy(cleanPath)
							fi, _ := d.Info()
							var sz int64 = 0
							if fi != nil {
								sz = fi.Size()
							}
							results = append(results, MbnConfigItem{
								Name:     friendlyName,
								Path:     relToSubDir,
								Type:     configType,
								Category: category,
								Size:     sz,
							})
						}
					}
				}
				return nil
			})
		}
	}

	// 2. If OS filesystem has results, return them
	if len(results) > 0 {
		logger.Get().Infof("MBN", "Discovered %d %s configurations on OS filesystem", len(results), configType)
		return results
	}

	// 3. Fallback to modem internal EFS via DIAG if available
	if s != nil {
		efsDir := EfsDirMcfgSwGeneric
		if configType == "hw" {
			efsDir = EfsDirMcfgHwGeneric
		}
		if efsList, err := s.ScanMbnDirectory(efsDir, configType); err == nil {
			return efsList
		}
	}

	return results
}

func (s *EfsSession) DiscoverMbnConfigs(configType string) []MbnConfigItem {
	return DiscoverMbnConfigs(s, configType)
}

func parseMbnHierarchy(relPath string) (category string, friendlyName string) {
	relPath = strings.TrimPrefix(relPath, "generic/")
	parts := strings.Split(relPath, "/")

	category = "generic"
	if len(parts) >= 2 {
		for _, part := range parts {
			upper := strings.ToUpper(part)
			if upper == "SEA" || upper == "APAC" || upper == "EU" || upper == "NA" ||
				upper == "AUNZ" || upper == "SA" || upper == "MEA" || upper == "CHINA" || upper == "COMMON" {
				category = upper
				break
			}
		}
	}

	var nameParts []string
	for _, p := range parts {
		lower := strings.ToLower(p)
		if lower == "generic" || lower == "pixel" || strings.HasSuffix(lower, ".mbn") {
			continue
		}
		nameParts = append(nameParts, p)
	}

	if len(nameParts) > 0 {
		friendlyName = strings.Join(nameParts, " ")
	} else {
		friendlyName = filepath.Base(relPath)
	}

	return category, friendlyName
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
