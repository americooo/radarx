import {createContext, ReactNode, useContext, useMemo, useState} from 'react';
import {dictionaries, TranslationKey} from './translations';
import {Language, loadLanguagePref, saveLanguagePref} from './storage';

interface LanguageContextValue {
    language: Language;
    setLanguage: (lang: Language) => void;
    t: (key: TranslationKey, vars?: Record<string, string | number>) => string;
}

const LanguageContext = createContext<LanguageContextValue | null>(null);

function interpolate(template: string, vars?: Record<string, string | number>): string {
    if (!vars) return template;
    return Object.entries(vars).reduce(
        (acc, [k, v]) => acc.replace(new RegExp(`\\{${k}\\}`, 'g'), String(v)),
        template
    );
}

export function LanguageProvider({children}: {children: ReactNode}) {
    const [language, setLanguageState] = useState<Language>(() => loadLanguagePref() || 'uz');

    function setLanguage(lang: Language) {
        setLanguageState(lang);
        saveLanguagePref(lang);
    }

    const t = useMemo(() => {
        return (key: TranslationKey, vars?: Record<string, string | number>) => {
            const dict = dictionaries[language];
            const template = dict[key] ?? dictionaries.en[key] ?? key;
            return interpolate(template, vars);
        };
    }, [language]);

    const value = useMemo(() => ({language, setLanguage, t}), [language, t]);

    return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>;
}

export function useI18n(): LanguageContextValue {
    const ctx = useContext(LanguageContext);
    if (!ctx) {
        throw new Error('useI18n must be used within a LanguageProvider');
    }
    return ctx;
}

// Picks the singular/plural translation key based on count (English-style:
// 1 => singular, everything else => plural). Uzbek doesn't inflect nouns for
// plural, so both dictionary entries can hold the same text there.
export function pluralKey<K extends string>(base: K, count: number): `${K}_one` | `${K}_other` {
    return (count === 1 ? `${base}_one` : `${base}_other`) as `${K}_one` | `${K}_other`;
}
