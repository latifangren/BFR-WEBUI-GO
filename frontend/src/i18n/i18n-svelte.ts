// Hand-written (NOT generated): wires the generated locale dictionaries
// into the typesafe-i18n Svelte store. The generator never creates this file.
import { initI18nSvelte } from 'typesafe-i18n/svelte'
import { loadedFormatters, loadedLocales } from './i18n-util'

export const { locale, LL, setLocale } = initI18nSvelte(loadedLocales, loadedFormatters)
