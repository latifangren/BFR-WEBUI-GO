import { mount } from 'svelte'
import './styles/app.css'
import App from './App.svelte'
import { loadLocaleAsync } from './i18n/i18n-util.async'
import { setLocale } from './i18n/i18n-svelte'

// 'en' loads first (lazy chunk); a failed chunk still mounts the app in English.
await loadLocaleAsync('en').catch(() => {})
setLocale('en')
const stored = localStorage.getItem('locale')
if (stored === 'id') {
	await loadLocaleAsync('id').catch(() => {})
	setLocale('id')
}

const app = mount(App, {
	target: document.getElementById('app')!,
})

export default app
