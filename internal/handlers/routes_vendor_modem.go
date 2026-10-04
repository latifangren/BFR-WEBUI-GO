package handlers

import (
	"encoding/json"
	"net/http"
)

// registerVendorModemRoutes stubs out proprietary vendor modem controls in public/no-modem builds.
func registerVendorModemRoutes(rc *routeContext) {
	privateHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      false,
			"error":   "Vendor modem controls are private features not included in this build",
			"private": true,
		})
	}

	routes := []string{
		"/api/modem/bands",
		"/api/modem/at",
		"/api/modem/reset",
		"/api/modem/bands/reset",
		"/api/modem/cell_lock",
		"/api/modem/cell_lock/clear",
		"/api/modem/plmn_lock",
		"/api/modem/hardware_bands",
		"/api/samsung/status",
		"/api/samsung/bands",
		"/api/samsung/lock",
		"/api/samsung/auto",
		"/api/samsung/sim/switch",
		"/api/samsung/thermal",
		"/api/shannon/status",
		"/api/shannon/lock",
		"/api/shannon/reset",
		"/api/shannon/ca/status",
		"/api/shannon/ca/document",
		"/api/shannon/ca/load",
		"/api/shannon/ca/upload",
		"/api/shannon/ca/save",
		"/api/shannon/ca/download",
		"/api/shannon/ca/validate",
		"/api/shannon/ca/autogen",
		"/api/shannon/ca/prune",
		"/api/shannon/ca/clear",
		"/api/shannon/nr/status",
		"/api/shannon/nr/document",
		"/api/shannon/nr/load",
		"/api/shannon/nr/upload",
		"/api/shannon/nr/demo",
		"/api/shannon/nr/download",
		"/api/shannon/nr/clear",
		"/api/qualcomm/bands",
		"/api/qualcomm/bands/reset",
		"/api/qualcomm/reset",
		"/api/qualcomm/cell_lock",
		"/api/qualcomm/cell_lock/clear",
		"/api/qualcomm/cell_scan",
		"/api/qualcomm/plmn_lock",
		"/api/qualcomm/hardware_bands",
		"/api/efs/status",
		"/api/efs/connect",
		"/api/efs/disconnect",
		"/api/efs/readonly",
		"/api/efs/list",
		"/api/efs/read",
		"/api/efs/write",
		"/api/efs/mkdir",
		"/api/efs/unlink",
		"/api/efs/rename",
		"/api/efs/chmod",
		"/api/efs/nv/items",
		"/api/efs/nv/read",
		"/api/efs/nv/write",
		"/api/efs/nv/backup",
		"/api/efs/nv/restore",
		"/api/efs/nv/format",
		"/api/efs/spc",
		"/api/efs/ssr",
		"/api/efs/backup",
		"/api/efs/carrier-policy/parse",
		"/api/efs/imei",
		"/api/efs/imei/write",
	}

	for _, pattern := range routes {
		rc.mux.HandleFunc(pattern, rc.wrap(privateHandler, true))
	}
}
