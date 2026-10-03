package handlers

func registerKernelRoutes(rc *routeContext) {
	// Kernel Tuning & SoC Management URLs
	rc.mux.HandleFunc("/api/kernel/status", rc.wrap(HandleKernelStatus, true))
	rc.mux.HandleFunc("/api/kernel/profile", rc.wrap(HandleKernelProfile, true))
	rc.mux.HandleFunc("/api/kernel/cluster", rc.wrap(HandleKernelCluster, true))
	rc.mux.HandleFunc("/api/kernel/vm", rc.wrap(HandleKernelVM, true))
}
