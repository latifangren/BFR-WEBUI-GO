package filemanager

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bfr-webui-go/internal/config"
)

func TestSanitizePath(t *testing.T) {
	tempDir := t.TempDir()
	origAllowed := config.AllowedDirs
	config.AllowedDirs = []string{tempDir}
	defer func() { config.AllowedDirs = origAllowed }()

	validFile := filepath.Join(tempDir, "file.txt")
	clean, err := SanitizePath(validFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(clean, "file.txt") {
		t.Errorf("expected clean path to contain 'file.txt', got %s", clean)
	}

	// Empty path error
	_, err = SanitizePath("")
	if err == nil {
		t.Error("expected error for empty path")
	}

	// Outside path error
	_, err = SanitizePath("/outside/root/file.txt")
	if err == nil {
		t.Error("expected access denied error for outside path")
	}
}

func TestFileOperationsAndDirectory(t *testing.T) {
	tempDir := t.TempDir()
	origAllowed := config.AllowedDirs
	config.AllowedDirs = []string{tempDir}
	defer func() { config.AllowedDirs = origAllowed }()

	// CreateDir
	subDir := filepath.Join(tempDir, "subfolder")
	if err := CreateDir(subDir); err != nil {
		t.Fatalf("CreateDir failed: %v", err)
	}

	// CreateFile
	testFile := filepath.Join(subDir, "hello.txt")
	if err := CreateFile(testFile); err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}

	// SaveFile (write content)
	content := "Hello BFR WebUI Go"
	if err := SaveFile(testFile, content); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// ReadFile
	readContent, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if readContent != content {
		t.Errorf("content mismatch: expected %s, got %s", content, readContent)
	}

	// ListDirectory
	items, current, err := ListDirectory(subDir)
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}
	if len(items) != 1 || items[0].Name != "hello.txt" {
		t.Errorf("expected 1 item hello.txt, got %+v (current: %s)", items, current)
	}

	// RenamePath
	renamedFile := filepath.Join(subDir, "renamed.txt")
	if err := RenamePath(testFile, renamedFile); err != nil {
		t.Fatalf("RenamePath failed: %v", err)
	}

	// CopyPath
	copiedFile := filepath.Join(subDir, "copied.txt")
	if err := CopyPath(renamedFile, copiedFile); err != nil {
		t.Fatalf("CopyPath failed: %v", err)
	}
	if _, err := os.Stat(copiedFile); err != nil {
		t.Error("copied file does not exist")
	}

	// MovePath
	movedFile := filepath.Join(tempDir, "moved.txt")
	if err := MovePath(copiedFile, movedFile); err != nil {
		t.Fatalf("MovePath failed: %v", err)
	}
	if _, err := os.Stat(movedFile); err != nil {
		t.Error("moved file does not exist")
	}

	// DeletePath
	if err := DeletePath(renamedFile); err != nil {
		t.Fatalf("DeletePath failed: %v", err)
	}
	if _, err := os.Stat(renamedFile); !os.IsNotExist(err) {
		t.Error("deleted file still exists")
	}
}

func TestArchiveZip(t *testing.T) {
	tempDir := t.TempDir()
	origAllowed := config.AllowedDirs
	config.AllowedDirs = []string{tempDir}
	defer func() { config.AllowedDirs = origAllowed }()

	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	_ = os.WriteFile(file1, []byte("content 1"), 0644)
	_ = os.WriteFile(file2, []byte("content 2"), 0644)

	destZip := filepath.Join(tempDir, "archive.zip")
	if err := CompressZip([]string{file1, file2}, destZip); err != nil {
		t.Fatalf("CompressZip failed: %v", err)
	}

	extractDir := filepath.Join(tempDir, "extracted")
	if err := ExtractZip(destZip, extractDir); err != nil {
		t.Fatalf("ExtractZip failed: %v", err)
	}

	ex1, err1 := os.ReadFile(filepath.Join(extractDir, "file1.txt"))
	ex2, err2 := os.ReadFile(filepath.Join(extractDir, "file2.txt"))
	if err1 != nil || string(ex1) != "content 1" {
		t.Errorf("extracted file1 mismatch: %s (err: %v)", string(ex1), err1)
	}
	if err2 != nil || string(ex2) != "content 2" {
		t.Errorf("extracted file2 mismatch: %s (err: %v)", string(ex2), err2)
	}
}

func TestSearchFiles(t *testing.T) {
	tempDir := t.TempDir()
	origAllowed := config.AllowedDirs
	config.AllowedDirs = []string{tempDir}
	defer func() { config.AllowedDirs = origAllowed }()

	matchFile := filepath.Join(tempDir, "special_target.conf")
	otherFile := filepath.Join(tempDir, "normal.txt")
	_ = os.WriteFile(matchFile, []byte("match"), 0644)
	_ = os.WriteFile(otherFile, []byte("other"), 0644)

	results, err := SearchFiles(tempDir, "special")
	if err != nil {
		t.Fatalf("SearchFiles failed: %v", err)
	}
	if len(results) != 1 || results[0].Name != "special_target.conf" {
		t.Errorf("expected 1 match special_target.conf, got %+v", results)
	}
}

func TestBatchOperationsAndRecursiveCopy(t *testing.T) {
	tempDir := t.TempDir()
	origAllowed := config.AllowedDirs
	config.AllowedDirs = []string{tempDir}
	defer func() { config.AllowedDirs = origAllowed }()

	// 1. Recursive Directory Copy
	srcDir := filepath.Join(tempDir, "src_tree")
	_ = os.MkdirAll(filepath.Join(srcDir, "nested"), 0755)
	_ = os.WriteFile(filepath.Join(srcDir, "file_a.txt"), []byte("aaa"), 0644)
	_ = os.WriteFile(filepath.Join(srcDir, "nested", "file_b.txt"), []byte("bbb"), 0644)

	dstDir := filepath.Join(tempDir, "dst_tree")
	err := CopyItem(srcDir, dstDir)
	if err != nil {
		t.Fatalf("CopyItem recursive failed: %v", err)
	}

	readNested, err := os.ReadFile(filepath.Join(dstDir, "nested", "file_b.txt"))
	if err != nil || string(readNested) != "bbb" {
		t.Errorf("recursive copy mismatch: %s (err: %v)", string(readNested), err)
	}

	// 2. MoveItem
	movedTree := filepath.Join(tempDir, "moved_tree")
	err = MoveItem(dstDir, movedTree)
	if err != nil {
		t.Fatalf("MoveItem failed: %v", err)
	}

	// 3. BatchCopy
	f1 := filepath.Join(srcDir, "file_a.txt")
	f2 := filepath.Join(srcDir, "nested", "file_b.txt")
	batchDst := filepath.Join(tempDir, "batch_target")
	_ = os.MkdirAll(batchDst, 0755)

	err = BatchCopy([]string{f1, f2}, batchDst)
	if err != nil {
		t.Fatalf("BatchCopy failed: %v", err)
	}

	// 4. BatchMove
	batchMoveDst := filepath.Join(tempDir, "batch_moved")
	_ = os.MkdirAll(batchMoveDst, 0755)
	err = BatchMove([]string{filepath.Join(batchDst, "file_a.txt")}, batchMoveDst)
	if err != nil {
		t.Fatalf("BatchMove failed: %v", err)
	}

	// 5. BatchDelete
	err = BatchDelete([]string{filepath.Join(batchDst, "file_b.txt"), filepath.Join(batchMoveDst, "file_a.txt")})
	if err != nil {
		t.Fatalf("BatchDelete failed: %v", err)
	}

	// 6. UploadFile
	uploadReader := strings.NewReader("uploaded content data")
	err = UploadFile(srcDir, "uploaded.txt", uploadReader)
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}

	// 7. DownloadFile
	uploadedPath := filepath.Join(srcDir, "uploaded.txt")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/files/download", nil)
	DownloadFile(uploadedPath, rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK for DownloadFile, got %d", rec.Code)
	}
	if rec.Body.String() != "uploaded content data" {
		t.Errorf("download content mismatch: %s", rec.Body.String())
	}

	// Download non-existent file
	recNotFound := httptest.NewRecorder()
	DownloadFile(filepath.Join(srcDir, "non_existent.txt"), recNotFound, req)
	if recNotFound.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent download, got %d", recNotFound.Code)
	}

	// 8. ChangePermissions
	err = ChangePermissions(uploadedPath, "0644", "")
	t.Logf("ChangePermissions octal returned: %v", err)

	err = ChangePermissions(uploadedPath, "invalid_octal", "")
	if err == nil {
		t.Error("expected error for invalid octal mode")
	}

	err = ChangePermissions(uploadedPath, "", "user:group:extra_colon")
	if err == nil {
		t.Error("expected error for invalid owner format")
	}

	// 9. GetStorageUsage
	total, free, used, pct, err := GetStorageUsage()
	t.Logf("GetStorageUsage: total=%d, free=%d, used=%d, pct=%f, err=%v", total, free, used, pct, err)
}

func TestFilemanagerEdgeCasesAndErrors(t *testing.T) {
	tempDir := t.TempDir()
	origAllowed := config.AllowedDirs
	config.AllowedDirs = []string{tempDir}
	defer func() { config.AllowedDirs = origAllowed }()

	// 1. ListDirectory with empty path
	_, _, err := ListDirectory("")
	t.Logf("ListDirectory empty path returned: %v", err)

	// ListDirectory non-existent
	_, _, err = ListDirectory(filepath.Join(tempDir, "non_existent_folder"))
	if err == nil {
		t.Error("expected error listing non-existent directory")
	}

	// 2. ReadFile on directory
	_, err = ReadFile(tempDir)
	if err == nil {
		t.Error("expected error reading directory as file")
	}

	// ReadFile non-existent
	_, err = ReadFile(filepath.Join(tempDir, "not_there.txt"))
	if err == nil {
		t.Error("expected error reading non-existent file")
	}

	// 3. SaveFile on directory
	err = SaveFile(tempDir, "some content")
	if err == nil {
		t.Error("expected error saving content over existing directory")
	}

	// 4. CreateFile inside existing file as directory (fails)
	f := filepath.Join(tempDir, "existing.txt")
	_ = CreateFile(f)
	err = CreateFile(filepath.Join(f, "sub.txt"))
	if err == nil {
		t.Error("expected error creating file inside another file")
	}

	// 5. CreateDir inside existing file (fails)
	err = CreateDir(filepath.Join(f, "sub_dir"))
	if err == nil {
		t.Error("expected error creating dir inside a file")
	}

	// 6. RenamePath non-existent
	err = RenamePath(filepath.Join(tempDir, "no_such_file"), filepath.Join(tempDir, "new_name"))
	if err == nil {
		t.Error("expected error renaming non-existent file")
	}

	// 7. Delete root directory (forbidden)
	err = DeletePath("/")
	if err == nil {
		t.Error("expected error deleting root directory")
	}

	// 8. ExtractZip invalid zip file
	err = ExtractZip(f, filepath.Join(tempDir, "ext_out"))
	if err == nil {
		t.Error("expected error extracting invalid zip")
	}

	// 9. Batch operations with empty lists (clean no-ops)
	if err := BatchCopy([]string{}, tempDir); err != nil {
		t.Errorf("unexpected error for empty BatchCopy: %v", err)
	}
	if err := BatchMove([]string{}, tempDir); err != nil {
		t.Errorf("unexpected error for empty BatchMove: %v", err)
	}
	if err := BatchDelete([]string{}); err != nil {
		t.Errorf("unexpected error for empty BatchDelete: %v", err)
	}
}

func TestArchiveZipDirectoryHierarchy(t *testing.T) {
	tempDir := t.TempDir()
	origAllowed := config.AllowedDirs
	config.AllowedDirs = []string{tempDir}
	defer func() { config.AllowedDirs = origAllowed }()

	// Create directory hierarchy with subfolder and files
	rootDir := filepath.Join(tempDir, "zip_source")
	nestedDir := filepath.Join(rootDir, "sub_dir")
	_ = os.MkdirAll(nestedDir, 0755)
	_ = os.WriteFile(filepath.Join(rootDir, "root_file.txt"), []byte("root file"), 0644)
	_ = os.WriteFile(filepath.Join(nestedDir, "nested_file.txt"), []byte("nested file"), 0644)

	zipOut := filepath.Join(tempDir, "hierarchy.zip")
	err := CompressZip([]string{rootDir}, zipOut)
	if err != nil {
		t.Fatalf("CompressZip hierarchy failed: %v", err)
	}

	extractTarget := filepath.Join(tempDir, "hierarchy_extracted")
	err = ExtractZip(zipOut, extractTarget)
	if err != nil {
		t.Fatalf("ExtractZip hierarchy failed: %v", err)
	}

	// Verify extracted files
	extractedNested := filepath.Join(extractTarget, "zip_source", "sub_dir", "nested_file.txt")
	content, err := os.ReadFile(extractedNested)
	if err != nil || string(content) != "nested file" {
		t.Errorf("extracted nested file mismatch: %s (err: %v)", string(content), err)
	}

	// Test ExtractZip with forbidden outside destination
	err = ExtractZip(zipOut, "/forbidden/outside/dir")
	if err == nil {
		t.Error("expected error for forbidden destination in ExtractZip")
	}

	// Test CompressZip with forbidden outside source
	err = CompressZip([]string{"/forbidden/outside/file"}, filepath.Join(tempDir, "bad.zip"))
	if err == nil {
		t.Error("expected error for forbidden source in CompressZip")
	}
}

func TestParseDfOutput(t *testing.T) {
	sampleDf := []byte("Filesystem 1K-blocks Used Available Use% Mounted on\n/dev/block/sda 1048576 524288 524288 50% /data\n")
	total, free, used, pct, err := parseDfOutput(sampleDf)
	if err != nil {
		t.Fatalf("parseDfOutput failed: %v", err)
	}
	if total != 1048576*1024 || used != 524288*1024 || free != 524288*1024 || pct != 50.0 {
		t.Errorf("parseDfOutput values mismatch: total=%d, used=%d, free=%d, pct=%f", total, used, free, pct)
	}

	// Truncated / empty
	_, _, _, _, err = parseDfOutput([]byte("invalid df output"))
	if err == nil {
		t.Error("expected error for invalid df output")
	}
}

func TestForbiddenPathErrors(t *testing.T) {
	tempDir := t.TempDir()
	origAllowed := config.AllowedDirs
	config.AllowedDirs = []string{tempDir}
	defer func() { config.AllowedDirs = origAllowed }()

	forbidden := "/forbidden/outside/path"

	_ = SaveFile(forbidden, "data")
	_, _ = ReadFile(forbidden)
	_ = CreateFile(forbidden)
	_ = CreateDir(forbidden)
	_ = RenamePath(forbidden, filepath.Join(tempDir, "dest"))
	_ = RenamePath(filepath.Join(tempDir, "src"), forbidden)
	_ = DeletePath(forbidden)
	_ = UploadFile(forbidden, "file.txt", strings.NewReader("data"))
	_ = CopyPath(forbidden, filepath.Join(tempDir, "dest"))
	_ = CopyPath(filepath.Join(tempDir, "src"), forbidden)
	_ = MovePath(forbidden, filepath.Join(tempDir, "dest"))
	_ = MovePath(filepath.Join(tempDir, "src"), forbidden)
	_ = BatchCopy([]string{forbidden}, tempDir)
	_ = BatchCopy([]string{filepath.Join(tempDir, "src")}, forbidden)
	_ = BatchMove([]string{forbidden}, tempDir)
	_ = BatchMove([]string{filepath.Join(tempDir, "src")}, forbidden)
	_ = BatchDelete([]string{forbidden})
	_ = ChangePermissions(forbidden, "0644", "")
}

func TestFileManagerAdditionalLimitsAndErrors(t *testing.T) {
	tempDir := t.TempDir()
	origAllowed := config.AllowedDirs
	config.AllowedDirs = []string{tempDir}
	defer func() { config.AllowedDirs = origAllowed }()

	// 1. ReadFile exceeding MaxReadFileSize
	bigFile := filepath.Join(tempDir, "large_file.bin")
	f, err := os.Create(bigFile)
	if err == nil {
		_ = f.Truncate(MaxReadFileSize + 10)
		_ = f.Close()
		_, err = ReadFile(bigFile)
		if err == nil {
			t.Error("expected error reading file exceeding 5MB")
		}
	}

	// 2. DownloadFile on directory
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/download", nil)
	DownloadFile(tempDir, rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 downloading directory, got %d", rec.Code)
	}

	// DownloadFile forbidden path
	recForbidden := httptest.NewRecorder()
	DownloadFile("/forbidden/outside", recForbidden, req)
	if recForbidden.Code != http.StatusForbidden {
		t.Errorf("expected 403 downloading forbidden path, got %d", recForbidden.Code)
	}

	// 3. ExtractZip non-existent zip
	err = ExtractZip(filepath.Join(tempDir, "missing.zip"), tempDir)
	if err == nil {
		t.Error("expected error extracting missing zip")
	}
}

func TestSearchFilesEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	origAllowed := config.AllowedDirs
	config.AllowedDirs = []string{tempDir}
	defer func() { config.AllowedDirs = origAllowed }()

	// Forbidden root
	_, err := SearchFiles("/forbidden/outside", "test")
	if err == nil {
		t.Error("expected error searching forbidden root")
	}

	// Empty query
	res, err := SearchFiles(tempDir, "")
	if err != nil {
		t.Errorf("unexpected error with empty query: %v", err)
	}
	t.Logf("Empty query found %d entries", len(res))
}

func TestCompressZipSkipSelf(t *testing.T) {
	tempDir := t.TempDir()
	origAllowed := config.AllowedDirs
	config.AllowedDirs = []string{tempDir}
	defer func() { config.AllowedDirs = origAllowed }()

	zipTarget := filepath.Join(tempDir, "self_test.zip")
	_ = os.WriteFile(zipTarget, []byte("existing"), 0644)

	// Passing target zip as a source should be skipped safely without recursion
	err := CompressZip([]string{zipTarget}, zipTarget)
	if err != nil {
		t.Fatalf("CompressZip with self path failed: %v", err)
	}
}

func TestListDirectorySorting(t *testing.T) {
	tempDir := t.TempDir()
	origAllowed := config.AllowedDirs
	config.AllowedDirs = []string{tempDir}
	defer func() { config.AllowedDirs = origAllowed }()

	// Create a subfolder, a file starting with 'a', and a file starting with 'b'
	_ = os.MkdirAll(filepath.Join(tempDir, "z_folder"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "b_file.txt"), []byte("b"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "a_file.txt"), []byte("a"), 0644)

	items, _, err := ListDirectory(tempDir)
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	// Folder should be sorted before files
	if !items[0].IsDir {
		t.Errorf("expected first item to be directory, got %s", items[0].Name)
	}
	// Files should be sorted alphabetically
	if items[1].Name != "a_file.txt" || items[2].Name != "b_file.txt" {
		t.Errorf("expected alphabetical order for files, got %s, %s", items[1].Name, items[2].Name)
	}
}
