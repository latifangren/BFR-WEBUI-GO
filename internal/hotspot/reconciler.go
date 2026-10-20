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
