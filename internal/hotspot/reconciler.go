package hotspot

import (
	"sync"
	"time"

	"bfr-webui-go/internal/logger"
)

// Reconciler keeps the kernel in sync with the persisted router settings.
// Android recreates tethering interfaces and their addresses on its own
// schedule, so BFR-owned objects have to be re-applied and re-verified
// periodically rather than only once at startup.
type Reconciler struct {
	mu       sync.Mutex
	running  bool
	stopChan chan struct{}
}

var (
	reconcilerOnce sync.Once
	reconciler     *Reconciler
)

// GetReconciler returns the singleton reconciler.
func GetReconciler() *Reconciler {
	reconcilerOnce.Do(func() {
		reconciler = &Reconciler{}
	})
	return reconciler
}

// Start begins the periodic reconciliation loop.
func (r *Reconciler) Start() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	r.stopChan = make(chan struct{})
	stop := r.stopChan
	r.mu.Unlock()

	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				_ = ReconcileRouterSettings()
			}
		}
	}()
}

// Stop halts the periodic reconciliation loop.
func (r *Reconciler) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running {
		return
	}
	r.running = false
	close(r.stopChan)
}

// AutoStartOnBoot activates tethering channels configured for boot auto-start.
// It is invoked once at startup so subsequent manual stops via WebUI or Android
// system controls are respected without periodic forced re-activation.
func AutoStartOnBoot() {
	cfg, err := LoadRouterConfig()
	if err != nil || cfg == nil {
		return
	}
	if !cfg.AutoStartWifi && !cfg.AutoStartUSB && !cfg.AutoStartEthernet {
		return
	}

	currentStatus := GetHotspotStatus()
	triggeredAny := false

	if cfg.AutoStartWifi && !currentStatus.WifiEnabled {
		logger.Get().Infof("hotspot", "Auto-starting Wi-Fi Hotspot on boot (SSID: %s)", cfg.SSID)
		if err := ToggleHotspot(true, cfg.SSID, cfg.Password, cfg.SecurityType); err != nil {
			logger.Get().Warnf("hotspot", "Auto-start Wi-Fi Hotspot failed: %v", err)
		} else {
			triggeredAny = true
		}
	}

	if cfg.AutoStartUSB && !currentStatus.UsbEnabled {
		logger.Get().Infof("hotspot", "Auto-starting USB Tethering on boot")
		if err := toggleUSBTethering(true); err != nil {
			logger.Get().Warnf("hotspot", "Auto-start USB Tethering failed: %v", err)
		} else {
			triggeredAny = true
		}
	}

	if cfg.AutoStartEthernet && !currentStatus.EthernetEnabled {
		logger.Get().Infof("hotspot", "Auto-starting Ethernet Tethering on boot")
		if err := toggleEthernetTethering(true); err != nil {
			logger.Get().Warnf("hotspot", "Auto-start Ethernet Tethering failed: %v", err)
		} else {
			triggeredAny = true
		}
	}

	if triggeredAny {
		time.Sleep(2 * time.Second)
	}
}

// ReconcileRouterSettings compares persisted intent with the live tethering
// state and either applies the desired state or removes stale BFR-owned state.
//
// The rules are:
//   - Auto mode: no BFR-owned gateway state may exist; remove it if it does.
//   - Static mode: the BFR-owned address and policy rules must be present on
//     the active tether interface; re-apply them if Android recreated the
//     interface and dropped them.
//   - No active tether interface: remove BFR-owned state and stay inactive.
func ReconcileRouterSettings() error {
	cfg, err := LoadRouterConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		return nil
	}

	names := GetActiveInterfaceNames()
	if len(names) == 0 {
		// No tether interface is up: make sure no BFR-owned state lingers.
		RemoveStaticGateway("")
		return nil
	}

	if cfg.GatewayMode != "static" {
		RemoveStaticGateway(names[0])
		return nil
	}

	// Already fully applied on the current interface: nothing to do.
	observed := ObserveStaticGateway()
	if observed.Iface == names[0] && observed.AddressPresent && observed.RuleToSubnet && observed.RuleFromAP {
		return nil
	}

	if err := ApplyStaticGateway(names[0], cfg); err != nil {
		logger.Get().Warnf("hotspot", "Periodic router reconciliation failed: %v", err)
		return err
	}
	return nil
}
