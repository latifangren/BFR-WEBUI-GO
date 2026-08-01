package filemanager

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"

	"bfr-webui-go/internal/config"
)

var (
	reMode  = regexp.MustCompile(`^[0-7]{3,4}$`)
	reOwner = regexp.MustCompile(`^[a-zA-Z0-9_-]+(:[a-zA-Z0-9_-]+)?$`)
)

func ChangePermissions(targetPath string, modeStr string, ownerStr string) error {
	cleanPath, err := SanitizePath(targetPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return err
	}
	isDir := info.IsDir()

	if modeStr != "" {
		if !reMode.MatchString(modeStr) {
			return fmt.Errorf("invalid mode format: %s (expected octal string like 755 or 644)", modeStr)
		}

		if !isDir {
			parsedMode, err := strconv.ParseUint(modeStr, 8, 32)
			if err == nil {
				_ = os.Chmod(cleanPath, os.FileMode(parsedMode))
			}
		}

		chmodFlags := ""
		if isDir {
			chmodFlags = "-R "
		}
		cmdStr := fmt.Sprintf("chmod %s%s %s", chmodFlags, modeStr, cleanPath)
		if out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput(); err != nil {
			return fmt.Errorf("chmod error: %v, out: %s", err, string(out))
		}
	}

	if ownerStr != "" {
		if !reOwner.MatchString(ownerStr) {
			return fmt.Errorf("invalid owner format: %s (expected user or user:group)", ownerStr)
		}

		chownFlags := ""
		if isDir {
			chownFlags = "-R "
		}
		cmdStr := fmt.Sprintf("chown %s%s %s", chownFlags, ownerStr, cleanPath)
		if out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput(); err != nil {
			return fmt.Errorf("chown error: %v, out: %s", err, string(out))
		}
	}

	return nil
}
