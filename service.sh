#!/system/bin/sh
MODDIR=${0%/*}

until [ "$(getprop sys.boot_completed)" = "1" ]; do
    sleep 2
done

# Apply optimized tweaks via Go binary
chmod 755 "$MODDIR/webui"
"$MODDIR/webui" --apply-tweaks

# Launch WebUI in background
cd "$MODDIR" || exit 1
./webui &
