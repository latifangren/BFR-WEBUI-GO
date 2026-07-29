#!/system/bin/sh
SKIPUNZIP=0

ui_print "*************************************"
ui_print "*        BFR WEBUI GO               *"
ui_print "*   High-performance Control Panel   *"
ui_print "*************************************"

# Force stop running versions to release file lock on updating
killall -9 webui 2>/dev/null
pkill -9 -f "webui" 2>/dev/null

ui_print "- Setting up module permissions..."
set_perm "$MODPATH/service.sh" 0 0 0755
set_perm "$MODPATH/webui" 0 0 0755 0755

ui_print "- Activating WebUI service..."
# Start the service immediately without requiring reboot
/data/adb/modules/bfr_webui_go/webui &
