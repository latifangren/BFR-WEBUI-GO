//go:build !nomodem

package handlers

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"bfr-webui-go/internal/efs"
	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/modem"
)

// HandleEfsStatus returns status metrics and current connection info for Qualcomm EFS2.
func HandleEfsStatus(w http.ResponseWriter, r *http.Request) {
	mgr := efs.GetManager()
	st := mgr.GetStatus()
	st.IsQualcomm = modem.IsQRTRSupported()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

// HandleEfsConnect establishes or tests connection to Qualcomm DIAG service on QRTR.
func HandleEfsConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mgr := efs.GetManager()
	sess, err := mgr.GetSession()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	st := mgr.GetStatus()
	st.IsQualcomm = modem.IsQRTRSupported()
	_ = sess

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"status": st,
	})
}

// HandleEfsDisconnect closes the active DIAG connection.
func HandleEfsDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	efs.GetManager().Close()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// HandleEfsReadOnly toggles write protection for the EFS2 filesystem.
func HandleEfsReadOnly(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ReadOnly bool `json:"readonly"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	efs.GetManager().SetReadOnly(req.ReadOnly)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"readonly": req.ReadOnly,
	})
}

// HandleEfsList returns items inside an EFS2 directory path.
func HandleEfsList(w http.ResponseWriter, r *http.Request) {
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = "/"
	}

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	entries, err := sess.ListDir(dirPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
			"path":  dirPath,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"path":    dirPath,
		"entries": entries,
	})
}

// HandleEfsRead retrieves file contents from EFS2 as text, hex, or file download stream.
func HandleEfsRead(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "Missing 'path' query parameter", http.StatusBadRequest)
		return
	}

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	data, err := sess.ReadFile(filePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	format := r.URL.Query().Get("format")
	if format == "download" {
		fileName := filepath.Base(filePath)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		_, _ = w.Write(data)
		return
	}

	isBinary := !utf8.Valid(data)
	var textContent string
	if !isBinary {
		textContent = string(data)
	}

	resp := map[string]any{
		"ok":           true,
		"path":         filePath,
		"size":         len(data),
		"is_binary":    isBinary,
		"text_content": textContent,
		"data_hex":     hex.EncodeToString(data),
		"data_base64":  base64.StdEncoding.EncodeToString(data),
	}

	if !isBinary && (strings.HasSuffix(filePath, ".xml") || strings.Contains(textContent, "ca_band_combos")) {
		parsed := efs.ParseCarrierPolicyXML(textContent)
		if parsed != nil && parsed.TotalCombos > 0 {
			resp["carrier_policy"] = parsed
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleEfsParseCarrierPolicy accepts XML or raw combo string in request body and returns parsed CA combos.
func HandleEfsParseCarrierPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	parsed := efs.ParseCarrierPolicyXML(req.Content)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"result": parsed,
	})
}


// HandleEfsWrite saves file contents or item file to EFS2.
func HandleEfsWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path      string `json:"path"`
		Data      string `json:"data"`
		Format    string `json:"format"` // "text", "hex", "base64"
		Mode      int16  `json:"mode"`
		ForceItem *int   `json:"force_item,omitempty"` // nil or -1: auto, 0: regular file, 1: item file
		Backup    *bool  `json:"backup,omitempty"`     // default true
		Verify    *bool  `json:"verify,omitempty"`     // default true
		SSR       bool   `json:"ssr"`                  // restart modem if write succeeds
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "Missing 'path' field", http.StatusBadRequest)
		return
	}

	var rawData []byte
	switch req.Format {
	case "hex":
		decoded, err := hex.DecodeString(strings.TrimSpace(req.Data))
		if err != nil {
			http.Error(w, "Invalid hex data string", http.StatusBadRequest)
			return
		}
		rawData = decoded
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.Data))
		if err != nil {
			http.Error(w, "Invalid base64 data string", http.StatusBadRequest)
			return
		}
		rawData = decoded
	default:
		rawData = []byte(req.Data)
	}

	mode := req.Mode
	if mode == 0 {
		mode = 0644
	}

	forceItem := -1
	if req.ForceItem != nil {
		forceItem = *req.ForceItem
	}
	backup := true
	if req.Backup != nil {
		backup = *req.Backup
	}
	verify := true
	if req.Verify != nil {
		verify = *req.Verify
	}

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	opts := efs.WriteOptions{
		ForceItem: forceItem,
		Backup:    backup,
		Verify:    verify,
	}

	writeRes, err := sess.WriteFileWithOptions(req.Path, rawData, mode, opts)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	ssrTriggered := false
	if req.SSR {
		if err := efs.GetManager().RestartModem(); err != nil {
			logger.Get().Warnf("EFS", "Write succeeded on %s but SSR restart failed: %v", req.Path, err)
		} else {
			ssrTriggered = true
		}
	}
	writeRes.SSRTriggered = ssrTriggered

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":            true,
		"path":          writeRes.Path,
		"size":          writeRes.Size,
		"is_item":       writeRes.IsItem,
		"backed_up":     writeRes.BackedUp,
		"backup_id":     writeRes.BackupID,
		"verified":      writeRes.Verified,
		"ssr_triggered": writeRes.SSRTriggered,
	})
}

// HandleEfsMkdir creates a directory in EFS2.
func HandleEfsMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
		Mode int16  `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	mode := req.Mode
	if mode == 0 {
		mode = 0755
	}

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	if err := sess.Mkdir(req.Path, mode); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "path": req.Path})
}

// HandleEfsUnlink removes a file or item file from EFS2.
func HandleEfsUnlink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	if err := sess.Unlink(req.Path); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "path": req.Path})
}

// HandleEfsRename renames an EFS2 item.
func HandleEfsRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	if err := sess.Rename(req.From, req.To); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "from": req.From, "to": req.To})
}

// HandleEfsChmod modifies permissions on an EFS2 path.
func HandleEfsChmod(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
		Mode int16  `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	if err := sess.Chmod(req.Path, req.Mode); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "path": req.Path})
}




// HandleEfsSPC sends a 6-digit SPC unlock code to the modem.
func HandleEfsSPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SPC string `json:"spc"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	spc := strings.TrimSpace(req.SPC)
	if spc == "" {
		spc = "000000"
	}

	unlocked, err := efs.GetManager().UnlockSPC(spc)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"unlocked": unlocked,
	})
}

// HandleEfsSSR restarts the Qualcomm modem over DIAG Subsystem Restart.
func HandleEfsSSR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := efs.GetManager().RestartModem(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"message": "Qualcomm modem SSR (Subsystem Restart) issued successfully.",
	})
}

// HandleEfsBackup requests modem-generated TAR archive of EFS and streams it as attachment.
func HandleEfsBackup(w http.ResponseWriter, r *http.Request) {
	efsPath := r.URL.Query().Get("path")
	if efsPath == "" {
		efsPath = "/"
	}

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	tmpFile, err := os.CreateTemp("", "efs_backup_*.tar")
	if err != nil {
		http.Error(w, "Cannot create temporary file", http.StatusInternalServerError)
		return
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	bytesWritten, err := sess.DumpImage(efsPath, tmpPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Backup failed: %v", err), http.StatusInternalServerError)
		return
	}

	f, err := os.Open(tmpPath)
	if err != nil {
		http.Error(w, "Failed to read generated archive", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Disposition", "attachment; filename=\"efs_image.tar\"")
	w.Header().Set("Content-Type", "application/x-tar")
	w.Header().Set("Content-Length", strconv.FormatUint(bytesWritten, 10))
	_, _ = io.Copy(w, f)
}

// HandleEfsBackupList returns the list of auto-created EFS file backups.
func HandleEfsBackupList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filterPath := r.URL.Query().Get("path")
	backups, err := efs.ListEfsBackups(filterPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"backups": backups,
	})
}

// HandleEfsBackupRestore restores a previous backup to EFS.
func HandleEfsBackupRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID  string `json:"id"`
		SSR bool   `json:"ssr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "Missing 'id' field", http.StatusBadRequest)
		return
	}

	entry, data, err := efs.GetEfsBackup(req.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	opts := efs.WriteOptions{
		ForceItem: -1,
		Backup:    true,
		Verify:    true,
	}
	writeRes, err := sess.WriteFileWithOptions(entry.OriginalPath, data, 0644, opts)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	ssrTriggered := false
	if req.SSR {
		if err := efs.GetManager().RestartModem(); err != nil {
			logger.Get().Warnf("EFS", "Restore succeeded on %s but SSR restart failed: %v", entry.OriginalPath, err)
		} else {
			ssrTriggered = true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":            true,
		"restored_path": entry.OriginalPath,
		"size":          len(data),
		"verified":      writeRes.Verified,
		"ssr_triggered": ssrTriggered,
	})
}

// HandleEfsBackupDelete removes a single backup entry and its file.
func HandleEfsBackupDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "Missing 'id' field", http.StatusBadRequest)
		return
	}

	if err := efs.DeleteEfsBackup(req.ID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": req.ID})
}
