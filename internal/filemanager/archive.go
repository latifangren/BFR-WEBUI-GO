package filemanager

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"bfr-webui-go/internal/bufferpool"
)

func CompressZip(srcPaths []string, destZipPath string) error {
	cleanZip, err := SanitizePath(destZipPath)
	if err != nil {
		return err
	}

	zipFile, err := os.Create(cleanZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	for _, src := range srcPaths {
		cleanSrc, err := SanitizePath(src)
		if err != nil {
			return err
		}

		if _, err := os.Stat(cleanSrc); err != nil {
			return err
		}

		baseDir := filepath.Dir(cleanSrc)

		err = filepath.Walk(cleanSrc, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}
			relPath = filepath.ToSlash(relPath)

			if path == cleanZip {
				return nil
			}

			header, err := zip.FileInfoHeader(fi)
			if err != nil {
				return err
			}
			header.Name = relPath

			if fi.IsDir() {
				header.Name += "/"
			} else {
				header.Method = zip.Deflate
			}

			writer, err := archive.CreateHeader(header)
			if err != nil {
				return err
			}

			if fi.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			buf := bufferpool.GetBytes(32768)
			defer bufferpool.PutBytes(buf)
			_, err = io.CopyBuffer(writer, file, buf)
			return err
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func ensureExtractPath(destDir, targetPath string, isDir bool) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	relPath, err := filepath.Rel(destDir, targetPath)
	if err != nil || relPath == ".." || filepath.IsAbs(relPath) || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes extraction destination")
	}

	parts := strings.Split(relPath, string(filepath.Separator))
	limit := len(parts)
	if !isDir {
		limit--
	}

	current := destDir
	for i := 0; i < limit; i++ {
		current = filepath.Join(current, parts[i])
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if err := os.Mkdir(current, 0755); err != nil && !os.IsExist(err) {
				return err
			}
			info, err = os.Lstat(current)
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("unsafe extraction path component: %s", current)
		}
	}

	if !isDir {
		if info, err := os.Lstat(targetPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to overwrite symlink: %s", targetPath)
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func ExtractZip(zipPath, destDir string) error {
	cleanZip, err := SanitizePath(zipPath)
	if err != nil {
		return err
	}
	cleanDest, err := SanitizePath(destDir)
	if err != nil {
		return err
	}

	r, err := zip.OpenReader(cleanZip)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		targetPath := filepath.Join(cleanDest, f.Name)

		// Ensure the cleaned entry remains within the selected destination.
		targetPath = filepath.Clean(targetPath)
		if err := ensureExtractPath(cleanDest, targetPath, f.FileInfo().IsDir()); err != nil {
			return fmt.Errorf("illegal file path in zip %q: %w", f.Name, err)
		}

		sanitizedPath, err := SanitizePath(targetPath)
		if err != nil {
			return fmt.Errorf("zip entry path forbidden: %w", err)
		}
		if sanitizedPath != targetPath {
			return fmt.Errorf("zip entry resolves outside destination: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			continue
		}

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		buf := bufferpool.GetBytes(32768)
		_, copyErr := io.CopyBuffer(outFile, rc, buf)
		bufferpool.PutBytes(buf)
		outFile.Close()
		rc.Close()

		if copyErr != nil {
			return copyErr
		}
	}

	return nil
}
