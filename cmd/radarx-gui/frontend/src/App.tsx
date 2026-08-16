import {useState} from 'react';
import './style.css';
import ScanView from './ScanView';
import TargetsView from './TargetsView';
import ResultsView from './ResultsView';
import DiffView from './DiffView';
import SettingsView from './SettingsView';
import Onboarding from './Onboarding';
import RadarIcon from './components/RadarIcon';
import {LanguageProvider, useI18n} from './i18n/LanguageContext';
import {loadOnboarded} from './i18n/storage';

type Tab = 'targets' | 'scan' | 'results' | 'diff' | 'settings';

function AppShell() {
    const [tab, setTab] = useState<Tab>('scan');
    const {t} = useI18n();

    const TABS: {id: Tab; label: string}[] = [
        {id: 'targets', label: t('nav.targets')},
        {id: 'scan', label: t('nav.scan')},
        {id: 'results', label: t('nav.results')},
        {id: 'diff', label: t('nav.diff')},
        {id: 'settings', label: t('nav.settings')},
    ];

    return (
        <div className="flex min-h-screen bg-gradient-to-b from-slate-950 to-blue-950/30 text-slate-100">
            <aside className="w-48 shrink-0 border-r border-blue-900/40 bg-slate-900/40 px-3 py-6 backdrop-blur-md">
                <div className="flex items-center gap-2 px-2">
                    <RadarIcon size={22} />
                    <span className="font-mono text-lg font-bold tracking-tight text-blue-400">
                        {t('app.name')}
                    </span>
                </div>
                <nav className="mt-6 flex flex-col gap-1">
                    {TABS.map((tb) => (
                        <button
                            key={tb.id}
                            onClick={() => setTab(tb.id)}
                            className={`rounded-md px-3 py-2 text-left text-sm font-medium transition-colors ${
                                tab === tb.id
                                    ? 'bg-blue-900/50 text-blue-300'
                                    : 'text-slate-400 hover:bg-slate-900/60 hover:text-slate-200'
                            }`}
                        >
                            {tb.label}
                        </button>
                    ))}
                </nav>
            </aside>

            <main className="flex-1 px-6 py-8">
                <div className="mx-auto max-w-4xl">
                    {tab === 'targets' && <TargetsView />}
                    {tab === 'scan' && <ScanView />}
                    {tab === 'results' && <ResultsView />}
                    {tab === 'diff' && <DiffView />}
                    {tab === 'settings' && <SettingsView />}
                </div>
            </main>
        </div>
    );
}

function App() {
    const [onboarded, setOnboarded] = useState(() => loadOnboarded());

    return (
        <LanguageProvider>
            {onboarded ? <AppShell /> : <Onboarding onDone={() => setOnboarded(true)} />}
        </LanguageProvider>
    );
}

export default App;
