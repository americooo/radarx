// Persistence for onboarding/language preferences.
//
// Deliberately isolated in this file: today it only touches `localStorage`,
// but once the backend gains a settings store (SQLite via a Go binding),
// only the bodies of these functions need to change — every caller
// (LanguageContext, Onboarding) keeps working unmodified.

export type Language = 'uz' | 'en';

const LANGUAGE_KEY = 'radarx.language';
const ONBOARDED_KEY = 'radarx.onboarded';

export function loadLanguagePref(): Language | null {
    const v = localStorage.getItem(LANGUAGE_KEY);
    return v === 'uz' || v === 'en' ? v : null;
}

export function saveLanguagePref(lang: Language): void {
    localStorage.setItem(LANGUAGE_KEY, lang);
}

export function loadOnboarded(): boolean {
    return localStorage.getItem(ONBOARDED_KEY) === '1';
}

export function saveOnboarded(): void {
    localStorage.setItem(ONBOARDED_KEY, '1');
}
