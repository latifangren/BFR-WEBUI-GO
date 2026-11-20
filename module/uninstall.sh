#!/system/bin/sh
# BFR WEBUI GO - cleanup executed when the module is uninstalled.
# Stops the daemon; user configuration (*.json) is intentionally left untouched.

MODID="bfr_webui_go"

# Stop the daemon in both the active and the staged module directories
for dir in "/data/adb/modules/$MODID" "/data/adb/modules_update/$MODID"; do
    if [ -x "$dir/webui" ]; then
        pkill -9 -f "$dir/webui" 2>/dev/null
    fi
done
# Catch-all in case the module dir was renamed
pkill -9 -f "bfr_webui_go/webui" 2>/dev/null

# NOTE: system tweaks applied via 'webui --apply-tweaks' (sysctl and
# similar) are NOT auto-reverted on uninstall. Reboot to restore
# stock behavior.

exit 0
