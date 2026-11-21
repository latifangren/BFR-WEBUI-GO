/**
 * Feature flags for BFR-WEBUI-GO build targets.
 *
 * In the 'dev' branch (full private build), HAS_VENDOR_MODEM is true.
 * Set VITE_HAS_VENDOR_MODEM=false when building the public no-modem flavor to
 * exclude proprietary baseband tools (Qualcomm QRTR/EFS, Samsung SecRIL, Tensor Shannon).
 */
export const HAS_VENDOR_MODEM = import.meta.env.VITE_HAS_VENDOR_MODEM !== 'false'
