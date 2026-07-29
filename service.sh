#!/system/bin/sh
MODDIR=${0%/*}

until [ "$(getprop sys.boot_completed)" = "1" ]; do
    sleep 2
done

# Network sysctl optimizations
sysctl -w net.core.rmem_max=67108864 2>/dev/null
sysctl -w net.core.wmem_max=67108864 2>/dev/null
sysctl -w net.core.rmem_default=33554432 2>/dev/null
sysctl -w net.core.wmem_default=33554432 2>/dev/null
sysctl -w net.ipv4.tcp_rmem="4096 87380 67108864" 2>/dev/null
sysctl -w net.ipv4.tcp_wmem="4096 65536 67108864" 2>/dev/null
sysctl -w net.ipv4.tcp_fastopen=3 2>/dev/null
sysctl -w net.core.default_qdisc=fq 2>/dev/null
sysctl -w net.ipv4.tcp_congestion_control=bbr2 2>/dev/null || sysctl -w net.ipv4.tcp_congestion_control=bbr 2>/dev/null
sysctl -w net.ipv4.tcp_keepalive_time=300 2>/dev/null
sysctl -w net.ipv4.tcp_keepalive_intvl=15 2>/dev/null
sysctl -w net.ipv4.tcp_keepalive_probes=5 2>/dev/null
sysctl -w net.ipv4.tcp_timestamps=1 2>/dev/null
sysctl -w net.ipv4.tcp_sack=1 2>/dev/null
sysctl -w net.ipv4.tcp_window_scaling=1 2>/dev/null

# Interface MTU & TxQueue Length Tuning
for iface in $(ip link show | awk -F: '/^[0-9]+: (wlan|rmnet|eth)/ {print $2}' | tr -d ' '); do
    ifconfig "$iface" txqueuelen 5000 2>/dev/null || ip link set "$iface" txqueuelen 5000 2>/dev/null
    ifconfig "$iface" mtu 1500 2>/dev/null || ip link set "$iface" mtu 1500 2>/dev/null
done

# TTL Spoofing
iptables -t mangle -C POSTROUTING -j TTL --ttl-set 64 2>/dev/null || iptables -t mangle -A POSTROUTING -j TTL --ttl-set 64 2>/dev/null
ip6tables -t mangle -C POSTROUTING -j HL --hl-set 64 2>/dev/null || ip6tables -t mangle -A POSTROUTING -j HL --hl-set 64 2>/dev/null

# Launch WebUI
chmod 755 "$MODDIR/webui"
cd "$MODDIR" || exit 1
./webui &
