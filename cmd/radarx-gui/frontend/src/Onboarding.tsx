import {useState} from 'react';
import RadarIcon from './components/RadarIcon';
import {useI18n} from './i18n/LanguageContext';
import {Language, saveOnboarded} from './i18n/storage';

type Screen = 'banner' | 'language';

function Onboarding({onDone}: {onDone: () => void}) {
    const [screen, setScreen] = useState<Screen>('banner');
    const {language, setLanguage, t} = useI18n();

    function finish(lang: Language) {
        setLanguage(lang);
        saveOnboarded();
        onDone();
    }

    return (
        <div className="flex min-h-screen items-center justify-center bg-gradient-to-b from-slate-950 to-blue-950/30 text-slate-100">
            {screen === 'banner' && (
                <div className="flex flex-col items-center gap-6 px-6 text-center">
                    <RadarIcon size={140} animated className="drop-shadow-[0_0_30px_rgba(59,130,246,0.35)]" />
                    <div>
                        <h1 className="text-5xl font-bold tracking-[0.15em] text-blue-300">RADARX</h1>
                        <p className="mt-3 text-sm font-medium uppercase tracking-[0.3em] text-blue-400/70">
                            {t('onboarding.tagline')}
                        </p>
                    </div>
                    <button
                        className="mt-4 rounded-md border border-blue-800/40 bg-blue-600/80 px-8 py-3 text-sm font-semibold text-white
                                   backdrop-blur-md transition-colors hover:bg-blue-500"
                        onClick={() => setScreen('language')}
                    >
                        {t('onboarding.continue')}
                    </button>
                </div>
            )}

            {screen === 'language' && (
                <div className="flex flex-col items-center gap-6 px-6 text-center">
                    <RadarIcon size={64} className="opacity-80" />
                    <div>
                        <h2 className="text-2xl font-bold text-blue-200">{t('onboarding.chooseLanguage')}</h2>
                        <p className="mt-1 text-sm text-slate-400">{t('onboarding.chooseLanguageSubtitle')}</p>
                    </div>

                    <div className="mt-2 flex gap-4">
                        <button
                            className={`w-40 rounded-xl border px-6 py-8 text-lg font-semibold backdrop-blur-md transition-colors ${
                                language === 'uz'
                                    ? 'border-blue-500 bg-blue-900/40 text-blue-200'
                                    : 'border-blue-900/40 bg-slate-900/40 text-slate-300 hover:border-blue-700/60 hover:bg-blue-950/30'
                            }`}
                            onClick={() => finish('uz')}
                        >
                            {t('onboarding.lang.uz')}
                        </button>
                        <button
                            className={`w-40 rounded-xl border px-6 py-8 text-lg font-semibold backdrop-blur-md transition-colors ${
                                language === 'en'
                                    ? 'border-blue-500 bg-blue-900/40 text-blue-200'
                                    : 'border-blue-900/40 bg-slate-900/40 text-slate-300 hover:border-blue-700/60 hover:bg-blue-950/30'
                            }`}
                            onClick={() => finish('en')}
                        >
                            {t('onboarding.lang.en')}
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}

export default Onboarding;
