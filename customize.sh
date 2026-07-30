#!/system/bin/sh
SKIPUNZIP=0

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
set_perm "$MODPATH/webui" 0 0 0755

ui_print "- Launching WebUI dynamically..."
# Start the newly installed binary from active MODPATH (which is modules_update during flash)
chmod 755 "$MODPATH/webui"
"$MODPATH"/webui &

ui_print "- Server started in background."
ui_print "- Web Panel URL: http://127.0.0.1:8080 or http://[your-ip-address]:8080"
