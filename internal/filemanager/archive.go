package filemanager

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

			_, err = io.Copy(writer, file)
			return err
		})

		if err != nil {
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

		// Zip Slip Vulnerability check
		if !strings.HasPrefix(filepath.Clean(targetPath), cleanDest) {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if _, err := SanitizePath(targetPath); err != nil {
			return fmt.Errorf("zip entry path forbidden: %w", err)
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(targetPath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			return err
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

		_, copyErr := io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if copyErr != nil {
			return copyErr
		}
	}

	return nil
}
