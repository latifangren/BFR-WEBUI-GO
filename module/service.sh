#!/system/bin/sh
MODDIR=${0%/*}

# Wait for boot to complete, but don't hang forever if the property
# never flips: give up after ~5 minutes (150 x 2s) and continue anyway.
BOOT_WAIT=0
until [ "$(getprop sys.boot_completed)" = "1" ]; do
    BOOT_WAIT=$((BOOT_WAIT + 1))
    if [ "$BOOT_WAIT" -ge 150 ]; then
        echo "- [BFR] sys.boot_completed never became 1 after 5 min, continuing anyway" >> "$MODDIR/webui.log" 2>/dev/null
        break
    fi
    sleep 2
done

# Apply optimized tweaks via Go binary
chmod 755 "$MODDIR/webui"
"$MODDIR/webui" --apply-tweaks

# Launch WebUI daemon detached from service.sh session with logging if not already running
if ! pgrep -f "$MODDIR/webui" >/dev/null 2>&1; then
    nohup "$MODDIR/webui" > "$MODDIR/webui.log" 2>&1 &
fi
