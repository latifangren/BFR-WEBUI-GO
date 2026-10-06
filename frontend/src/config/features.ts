/**
 * Feature flags for BFR-WEBUI-GO build targets.
 *
 * In the 'dev' branch (full private build), HAS_VENDOR_MODEM is true.
 * In the 'release/no-modem' branch, set HAS_VENDOR_MODEM to false to exclude
 * proprietary baseband tools (Qualcomm QRTR/EFS, Samsung SecRIL, Tensor Shannon).
 */
export const HAS_VENDOR_MODEM = false
