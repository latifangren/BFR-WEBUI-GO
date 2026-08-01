package filemanager

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"bfr-webui-go/internal/config"
)

func RenamePath(oldPath string, newPath string) error {
	cleanOld, err := SanitizePath(oldPath)
	if err != nil {
		return err
	}
	cleanNew, err := SanitizePath(newPath)
	if err != nil {
		return err
	}
	return os.Rename(cleanOld, cleanNew)
}

func DeletePath(targetPath string) error {
	cleanPath, err := SanitizePath(targetPath)
	if err != nil {
		return err
	}
	for _, base := range config.AllowedDirs {
		if cleanPath == filepath.Clean(base) {
			return fmt.Errorf("cannot delete root allowed directory: %s", cleanPath)
		}
	}
	return os.RemoveAll(cleanPath)
}

func CreateDir(dirPath string) error {
	cleanPath, err := SanitizePath(dirPath)
	if err != nil {
		return err
	}
	return os.MkdirAll(cleanPath, 0755)
}

func CreateFile(filePath string) error {
	cleanPath, err := SanitizePath(filePath)
	if err != nil {
		return err
	}
	file, err := os.Create(cleanPath)
	if err != nil {
		return err
	}
	return file.Close()
}

func UploadFile(dirPath string, fileName string, reader io.Reader) error {
	cleanDir, err := SanitizePath(dirPath)
	if err != nil {
		return err
	}
	safeFileName := filepath.Base(fileName)
	targetPath, err := SanitizePath(filepath.Join(cleanDir, safeFileName))
	if err != nil {
		return err
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, reader)
	return err
}

func CopyPath(src, dst string) error {
	cleanSrc, err := SanitizePath(src)
	if err != nil {
		return err
	}
	cleanDst, err := SanitizePath(dst)
	if err != nil {
		return err
	}

	info, err := os.Stat(cleanSrc)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDirRecursive(cleanSrc, cleanDst)
	}
	return copyFileSingle(cleanSrc, cleanDst)
}

func copyFileSingle(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func copyDirRecursive(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirRecursive(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFileSingle(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func MovePath(src, dst string) error {
	cleanSrc, err := SanitizePath(src)
	if err != nil {
		return err
	}
	cleanDst, err := SanitizePath(dst)
	if err != nil {
		return err
	}

	err = os.Rename(cleanSrc, cleanDst)
	if err == nil {
		return nil
	}

	// Fallback to copy + delete for cross-device moves
	if err := CopyPath(cleanSrc, cleanDst); err != nil {
		return fmt.Errorf("move failed (copy fallback error): %w", err)
	}
	return DeletePath(cleanSrc)
}

func BatchDelete(paths []string) error {
	var errs []string
	for _, p := range paths {
		if err := DeletePath(p); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", p, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("batch delete errors: %s", fmt.Sprintf("%v", errs))
	}
	return nil
}

func BatchCopy(srcs []string, destDir string) error {
	cleanDest, err := SanitizePath(destDir)
	if err != nil {
		return err
	}

	var errs []string
	for _, src := range srcs {
		target := filepath.Join(cleanDest, filepath.Base(src))
		if err := CopyPath(src, target); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", src, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("batch copy errors: %s", fmt.Sprintf("%v", errs))
	}
	return nil
}

func BatchMove(srcs []string, destDir string) error {
	cleanDest, err := SanitizePath(destDir)
	if err != nil {
		return err
	}

	var errs []string
	for _, src := range srcs {
		target := filepath.Join(cleanDest, filepath.Base(src))
		if err := MovePath(src, target); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", src, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("batch move errors: %s", fmt.Sprintf("%v", errs))
	}
	return nil
}
