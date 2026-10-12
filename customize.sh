#!/system/bin/sh
SKIPUNZIP=0

# Config files preserved across module updates. Add new JSON config
# filenames here so they survive reinstalls/updates.
PRESERVE_CONFIGS="network_config.json cell_lock.json kernel_config.json modem_config.json hotspot_mac_filter.json charger_config.json ssh_config.json telegram_config.json qos.json shortcuts.json nas.json nas_config.json tunnel.json tunnel_config.json auth.json auth_config.json cloud_config.json power_schedule.json"

ui_print "*************************************"
ui_print "*        BFR WEBUI GO               *"
ui_print "*   High-performance Control Panel   *"
ui_print "*************************************"

ui_print "- Checking for active WebUI processes..."
# Force stop running versions to release file lock on updating
killall -9 webui 2>/dev/null
pkill -9 -f "webui" 2>/dev/null
ui_print "- Prior instances terminated."

ui_print "- Installing module to: $MODPATH"
ui_print "- Setting up execution permissions..."
set_perm "$MODPATH/service.sh" 0 0 0755
set_perm "$MODPATH/uninstall.sh" 0 0 0755
set_perm "$MODPATH/webui" 0 0 0755
if [ -d "$MODPATH/bin" ]; then
    set_perm_recursive "$MODPATH/bin" 0 0 0755 0755
    chmod -R 755 "$MODPATH/bin" 2>/dev/null
fi

OLD_DIR="/data/adb/modules/bfr_webui_go"
if [ -d "$OLD_DIR" ]; then
    ui_print "- Preserving existing configuration files..."
    for cfg in $PRESERVE_CONFIGS; do
        if [ -f "$OLD_DIR/$cfg" ]; then
            cp -f "$OLD_DIR/$cfg" "$MODPATH/$cfg"
            ui_print "- Preserved: $cfg"
        fi
    done
fi

ui_print "- Launching WebUI dynamically..."
chmod 755 "$MODPATH/webui"
nohup "$MODPATH/webui" > "$MODPATH/webui.log" 2>&1 &

# Health check: verify background daemon status
sleep 2
if pgrep -f "$MODPATH/webui" >/dev/null 2>&1; then
    ui_print "- Server started in background."
else
    ui_print "- Notice: WebUI daemon will start automatically on boot via service.sh."
fi
ui_print "- Web Panel URL: http://127.0.0.1:80 or http://[your-ip-address]:80"
