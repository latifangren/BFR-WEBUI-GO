import type { TranslationFunctions } from '../i18n/i18n-types'

// Resolves a navigation label from the active locale. `LL` is passed in as
// the component's `$LL` store value so the template re-renders when the
// locale changes. Falls back to the English label stored on the item when a
// key is missing, so a gap degrades to English instead of a raw key or crash.
type NavSection = 'tabs' | 'categories' | 'categoryTitles'

export function tNav(
	LL: TranslationFunctions,
	section: NavSection,
	id: string,
	fallback: string,
): string {
	const dict = LL.nav[section] as unknown as Record<string, () => string>
	return dict[id]?.() ?? fallback
}
