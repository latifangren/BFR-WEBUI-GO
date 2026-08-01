# BFR-WEBUI-GO API Reference

This document provides complete documentation for the REST API endpoints and WebSocket endpoints exposed by the **BFR-WEBUI-GO** system control panel.

---

## 🔒 Authentication & Security

All API endpoints (except `/api/auth/login` and `/api/auth/status`) require session authentication.
- **Session Cookie**: Set upon login (`SameSite=Strict`, `HttpOnly`).
- **Security Headers**: All responses include `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, and `Referrer-Policy: no-referrer`.
- **Body Limit**: POST/PUT requests enforce a 1MB payload limit via `http.MaxBytesReader` (except file uploads which permit up to 32MB).

---

## 🔑 Authentication Endpoints

### `POST /api/auth/login`
Authenticates a user session.

- **Request Body**:
  ```json
  {
    "password": "your_password"
  }
  ```
- **Response** (`200 OK`):
  ```json
  {
    "success": true
  }
  ```

### `GET /api/auth/status`
Checks if the current session is authenticated.

- **Response** (`200 OK`):
  ```json
  {
    "authenticated": true
  }
  ```

### `POST /api/auth/logout`
Clears the current authentication cookie.

---

## 📊 System Information Endpoints

### `GET /api/sysinfo` (or `/api/stats`)
Returns full hardware metrics, CPU/RAM usage, temperatures, network details, and active services.

- **Response** (`200 OK`):
  ```json
  {
    "cpu_usage": 12.5,
    "cpu_cores": [10.0, 15.0, 8.0, 14.0],
    "cpu_temp": 38.5,
    "ram_total": 7800000000,
    "ram_used": 3400000000,
    "ram_percent": 43.5,
    "battery_level": 85,
    "battery_temp": 31.0,
    "network_detail": {
      "ip_addresses": ["192.168.100.55"],
      "gateway": "192.168.100.1",
      "dns": ["1.1.1.1 (Active)", "8.8.8.8 (Active)"],
      "dns1": "1.1.1.1 (Active)",
      "dns2": "8.8.8.8 (Active)",
      "wifi_ssid": "TOTOLINK_X6000R_5G",
      "wifi_signal": "-38 dBm / 866Mbps",
      "hotspot_clients": 2
    }
  }
  ```

---

## 📁 File Manager Endpoints

### `GET /api/files/list?path={directory_path}`
Lists directory entries and returns storage metrics.

- **Response** (`200 OK`):
  ```json
  {
    "path": "/sdcard",
    "files": [
      {
        "name": "Download",
        "path": "/sdcard/Download",
        "is_dir": true,
        "size": 4096,
        "mod_time": "2026-08-01T12:00:00Z",
        "permissions": "drwxrwxr-x"
      }
    ]
  }
  ```

### `GET /api/files/read?path={file_path}`
Reads file text contents (max 5MB).

### `POST /api/files/save`
Saves text content to a file.

- **Request Body**:
  ```json
  {
    "path": "/sdcard/config.json",
    "content": "{ \"key\": \"value\" }"
  }
  ```

### `POST /api/files/upload`
Uploads a file using `multipart/form-data`.
- Form Fields: `path` (destination directory), `file` (binary payload).

### `GET /api/files/download?path={file_path}`
Downloads a file as `application/octet-stream`.

### `POST /api/files/copy` & `POST /api/files/move`
Copies or moves a file/folder recursively.

- **Request Body**:
  ```json
  {
    "src": "/sdcard/source.txt",
    "dst": "/sdcard/Download/source.txt"
  }
  ```

### `POST /api/files/batch`
Performs bulk operations on multiple items.

- **Request Body**:
  ```json
  {
    "action": "delete | copy | move",
    "paths": ["/sdcard/a.txt", "/sdcard/b.txt"],
    "dest_dir": "/sdcard/Download"
  }
  ```

### `POST /api/files/permissions`
Modifies file octal permissions (`chmod`) and ownership (`chown`).

- **Request Body**:
  ```json
  {
    "path": "/data/local/tmp/script.sh",
    "mode": "0755",
    "owner": "root:root"
  }
  ```

### `POST /api/files/compress` & `POST /api/files/extract`
Compresses paths into a `.zip` file or extracts an existing ZIP file.

- **Compress Request Body**:
  ```json
  {
    "paths": ["/sdcard/folder1", "/sdcard/file.txt"],
    "dest_zip": "/sdcard/archive.zip"
  }
  ```

### `GET /api/files/search?path={root_path}&query={keyword}`
Searches directory recursively for matching files/directories (capped at 500 items).

### `GET /api/files/storage`
Returns system storage metrics.

- **Response**:
  ```json
  {
    "total": 128000000000,
    "free": 64000000000,
    "used": 64000000000,
    "percent": 50.0,
    "mount": "/sdcard",
    "total_str": "119.2 GB",
    "free_str": "59.6 GB",
    "used_str": "59.6 GB"
  }
  ```

---

## 🌐 Network & Tweaks Endpoints

### `POST /api/network/dns`
Configures DNS servers on interface resolvers and `iptables` port 53.

- **Request Body**:
  ```json
  {
    "primary": "1.1.1.1",
    "secondary": "8.8.8.8"
  }
  ```

### `POST /api/network/tweaks`
Applies sysctl, MTU, or TxQueueLen optimizations.

- **Sysctl Request Body**:
  ```json
  {
    "action": "sysctl",
    "key": "net.ipv4.ip_forward",
    "value": "1"
  }
  ```

### `POST /api/network/ping`
Runs ICMP ping diagnostic to a host.

- **Request Body**:
  ```json
  {
    "host": "1.1.1.1",
    "count": 4
  }
  ```

---

## ⚡ Power Management Endpoints

### `POST /api/power`
Triggers power actions asynchronously.

- **Request Body**:
  ```json
  {
    "action": "reboot | shutdown | soft_reboot | recovery | bootloader"
  }
  ```

---

## 🛡️ Proxy (Clash / Mihomo) Endpoints

### `GET /api/proxy/status`
Returns core binary status and Clash REST API state.

### `POST /api/proxy/control`
Executes proxy actions or updates operating modes.

- **Request Body**:
  ```json
  {
    "action": "start | stop | restart",
    "mode": "rule | global | direct | script"
  }
  ```

---

## 💻 Terminal & Scrcpy WebSockets

- `GET /ws/terminal`: Interactive root PTY terminal session over WebSocket.
- `GET /ws/scrcpy`: Android screen streaming & touch input over WebSocket.
