#!/system/bin/sh
MODDIR=${0%/*}

until [ "$(getprop sys.boot_completed)" = "1" ]; do
    sleep 2
done

# Apply optimized tweaks via Go binary
chmod 755 "$MODDIR/webui"
"$MODDIR/webui" --apply-tweaks

# Launch WebUI daemon detached from service.sh session with logging
nohup "$MODDIR/webui" > "$MODDIR/webui.log" 2>&1 &
