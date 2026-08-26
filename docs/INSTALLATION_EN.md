# BFR-WEBUI-GO Installation Guide (English)

This document provides a step-by-step guide to installing and running **BFR-WEBUI-GO** on root-accessed Android devices (using Magisk, KernelSU, or APatch).

---

## 📋 System Requirements

1. Rooted Android device (Android 8.0+ recommended).
2. One of the following Root Managers:
   - **Magisk** v20.4+
   - **KernelSU** v0.6.0+
   - **APatch** v10.7+
3. At least ~15MB of free storage space.

---

## 🚀 Installation Methods

### Method 1: Installing via Modul Zip (Recommended)

1. Download the latest module zip release package `bfr-webui-go-vX.Y.Z.zip` from the Releases page.
2. Open your Root Manager application (**Magisk App**, **KernelSU App**, or **APatch App**).
3. Go to the **Modules** / **Modul** section.
4. Select **Install from storage** and choose the downloaded zip file.
5. Wait for the flashing process to complete, then **Reboot** your device.

---

### Method 2: Manual Installation via Root Terminal / Shell

If you wish to test or install the binary manually without wrapping it into a zip package:

1. Copy the `bfr_webui_go` binary to the modules directory:
   ```bash
   su
   mkdir -p /data/adb/modules/bfr_webui_go
   cp /sdcard/Download/bfr_webui_go /data/adb/modules/bfr_webui_go/
   chmod 755 /data/adb/modules/bfr_webui_go/bfr_webui_go
   ```

2. Create a basic `module.prop` metadata file:
   ```bash
   cat << 'EOF' > /data/adb/modules/bfr_webui_go/module.prop
   id=bfr_webui_go
   name=BFR WebUI Go
   version=v1.2.0
   versionCode=120
   author=BFR
   description=Android System Control Panel & WebUI
   EOF
   ```

3. Create the boot service script `service.sh`:
   ```bash
   cat << 'EOF' > /data/adb/modules/bfr_webui_go/service.sh
   #!/system/bin/sh
   MODDIR=${0%/*}
   until [ "$(getprop sys.boot_completed)" = "1" ]; do
       sleep 2
   done
   sleep 5
   $MODDIR/bfr_webui_go > /dev/null 2>&1 &
   EOF
   chmod 755 /data/adb/modules/bfr_webui_go/service.sh
   ```

4. Launch the boot script manually or reboot:
   ```bash
   /data/adb/modules/bfr_webui_go/service.sh &
   ```

---

## 🌐 Accessing the WebUI

1. Ensure your device is connected to a local Wi-Fi network or use localhost network.
2. Open any web browser (Chrome, Kiwi, Firefox, etc.) on your Android device or another device on the same local network.
3. Access the URL:
   ```text
   http://localhost:80
   ```
   *Or from another device in the same local network:*
   ```text
   http://<YOUR_ANDROID_IP>:80
   ```
4. Enter the default password on login:
   ```text
   bfr
   ```

---

## ⚙️ Configuration Customization (Optional)

You can override default password, ports, or paths by setting **Environment Variables** before launching the service.

Example adding a custom password inside `/data/adb/modules/bfr_webui_go/service.sh`:

```bash
export PORT=80
export BFR_PASSWORD="YourSuperSecurePassword123"
export BFR_SU_BIN="ksu"  # if using KernelSU root manager
$MODDIR/bfr_webui_go > /dev/null 2>&1 &
```

---

## ❓ Troubleshooting & FAQ

- **WebUI cannot be opened/timed out:**
  Verify if the process is running using a root terminal:
  ```bash
  su -c "pgrep -f bfr_webui_go"
  ```
- **Forgot Password:**
  Modify `BFR_PASSWORD` value in your startup script or target environment configuration, then restart the service.
