import type { Component } from 'svelte'

// No vendor modem tabs in release/no-modem build
export const VENDOR_TABS: Record<string, Component> = {}
