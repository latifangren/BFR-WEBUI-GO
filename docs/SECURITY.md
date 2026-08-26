# BFR-WEBUI-GO Security Policy

This document outlines the security architecture, security boundaries, hardening practices, and vulnerability disclosure process for **BFR-WEBUI-GO**.

---

## 🛡️ Threat Model & Security Posture

Because **BFR-WEBUI-GO** runs as a root process on Android devices via Magisk / KernelSU / APatch, securing HTTP endpoints and system interaction is paramount. A security flaw could allow remote unauthorized execution or data corruption with root privileges.

---

## 🔒 Implemented Security Controls

### 1. Input Sanitization & Command Injection Defense
- **Strict Parameter Validation**: Input values for sysctl keys, DNS IP addresses, network interface names, MTU ranges (68–9000), TxQueueLen (0–100000), RPS bitmasks, and power actions are validated against strict whitelist regular expressions and `net.ParseIP` checks before shell execution.
- **Root Shell Command Isolation**: Parameters passed to `config.SUBin` (`su`, `ksu`, `apatch`) are checked against strict character sets.

### 2. Path Traversal & File Manager Sandboxing
- **Base Directory Restriction (`SanitizePath`)**: All file operations (`List`, `Read`, `Save`, `Upload`, `Delete`, `Copy`, `Move`, `Permissions`, `Search`) sanitize user paths with `filepath.Clean`, resolve symlink targets (`filepath.EvalSymlinks`), and verify that the target remains strictly within allowed base directories (`config.AllowedDirs`, defaulting to `/sdcard`, `/storage`, `/data/adb`, `/data/local/tmp`, `/data/system`).
- **Zip Slip Vulnerability Protection**: Archive extraction (`ExtractZip`) evaluates clean destination paths against target entries before extraction to block malicious relative path traversal within `.zip` archives.

### 3. Session & Authentication Hardening
- **Secure Password Validation**: Unauthenticated requests are rejected with `HTTP 401 Unauthorized`.
- **CSPRNG Session Tokens**: Session tokens are generated using `crypto/rand` (CSPRNG). If secure random bytes cannot be read, authentication requests return an explicit error.
- **Cookie Security**: Authentication cookies use `SameSite=Strict` and `HttpOnly` attributes.

### 4. HTTP Transport & Payload Protection
- **Request Body Limits**: All POST/PUT/PATCH requests enforce a 1MB payload ceiling via `http.MaxBytesReader` (except file uploads which permit up to 32MB).
- **Security Headers**: Every HTTP response sets `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, and `Referrer-Policy: no-referrer`.
- **Graceful Shutdown**: The HTTP server handles `SIGINT` and `SIGTERM` OS signals to cleanly terminate active requests.

---

## ⚠️ Security Best Practices for Users

1. **Change Default Password**:
   Always set a custom password via the `BFR_PASSWORD` environment variable in your module startup script:
   ```bash
   export BFR_PASSWORD="YourStrongCustomPasswordHere"
   ```
2. **Bind to Private Networks**:
   Avoid exposing port `80` to public untrusted Wi-Fi networks unless protected by a secure local firewall or VPN.

---

## 🐞 Vulnerability Disclosure

If you discover a security vulnerability or security flaw in BFR-WEBUI-GO:
- **Do NOT create a public issue**.
- Please report security vulnerabilities responsibly via private disclosure or contact the maintainer directly on GitHub.
