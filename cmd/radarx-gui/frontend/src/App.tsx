import {useState} from 'react';
import './style.css';
import ScanView from './ScanView';
import TargetsView from './TargetsView';
import ResultsView from './ResultsView';
import DiffView from './DiffView';
import SettingsView from './SettingsView';

type Tab = 'targets' | 'scan' | 'results' | 'diff' | 'settings';

const TABS: {id: Tab; label: string}[] = [
    {id: 'targets', label: 'Targets'},
    {id: 'scan', label: 'Scan'},
    {id: 'results', label: 'Results'},
    {id: 'diff', label: 'Diff'},
    {id: 'settings', label: 'Settings'},
];

function App() {
    const [tab, setTab] = useState<Tab>('scan');

    return (
        <div className="flex min-h-screen bg-slate-950 text-slate-100">
            <aside className="w-48 shrink-0 border-r border-slate-800 bg-slate-950/80 px-3 py-6">
                <div className="px-2 font-mono text-lg font-bold tracking-tight text-emerald-400">
                    RadarX
                </div>
                <nav className="mt-6 flex flex-col gap-1">
                    {TABS.map((t) => (
                        <button
                            key={t.id}
                            onClick={() => setTab(t.id)}
                            className={`rounded-md px-3 py-2 text-left text-sm font-medium transition-colors ${
                                tab === t.id
                                    ? 'bg-emerald-900/60 text-emerald-300'
                                    : 'text-slate-400 hover:bg-slate-900 hover:text-slate-200'
                            }`}
                        >
                            {t.label}
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

export default App;
