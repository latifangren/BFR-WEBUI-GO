#!/system/bin/sh
SKIPUNZIP=0

ui_print "*************************************"
ui_print "*        BFR WEBUI GO               *"
ui_print "*   High-performance Control Panel   *"
ui_print "*************************************"

ui_print "- Setting up module permissions..."
set_perm "$MODPATH/service.sh" 0 0 0755
set_perm "$MODPATH/webui" 0 0 0755 0755
