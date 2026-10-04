//go:build nomodem

package handlers

// registerVendorModemRoutes is a no-op in public/no-modem builds.
// Vendor modem endpoints are completely omitted from the binary.
func registerVendorModemRoutes(rc *routeContext) {
}
