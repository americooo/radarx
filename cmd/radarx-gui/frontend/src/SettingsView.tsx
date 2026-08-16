import {useEffect, useState} from 'react';
import {AuthorizeTarget, GetScopeRoots, GetTelegramStatus} from '../wailsjs/go/main/App';

function SettingsView() {
    const [roots, setRoots] = useState<string[]>([]);
    const [loading, setLoading] = useState(true);
    const [loadErr, setLoadErr] = useState('');

    const [newDomain, setNewDomain] = useState('');
    const [addErr, setAddErr] = useState('');
    const [adding, setAdding] = useState(false);

    const [tgEnabled, setTgEnabled] = useState(false);

    async function refresh() {
        setLoading(true);
        setLoadErr('');
        try {
            const res = await GetScopeRoots();
            setRoots(res || []);
        } catch (err) {
            setLoadErr(String(err));
        } finally {
            setLoading(false);
        }
    }

    useEffect(() => {
        refresh();
        GetTelegramStatus().then(setTgEnabled).catch(() => setTgEnabled(false));
    }, []);

    async function handleAddDomain() {
        const d = newDomain.trim().toLowerCase();
        if (!d) return;
        setAdding(true);
        setAddErr('');
        try {
            await AuthorizeTarget(d);
            setNewDomain('');
            await refresh();
        } catch (err) {
            setAddErr(String(err));
        } finally {
            setAdding(false);
        }
    }

    return (
        <div>
            <h1 className="text-2xl font-bold tracking-tight text-emerald-400">Settings</h1>
            <p className="mt-1 text-sm text-slate-400">Scope and notification configuration.</p>

            <div className="mt-6 rounded-md border border-slate-800 bg-slate-900/50 p-4">
                <h2 className="text-sm font-semibold text-slate-300">Authorized Scope</h2>
                <p className="mt-1 text-xs text-slate-500">
                    Root domains RadarX is allowed to scan. A scan is refused before any network
                    request is made unless its target is covered here.
                </p>

                <div className="mt-3 flex gap-3">
                    <input
                        className="flex-1 rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm
                                   placeholder:text-slate-500 focus:border-emerald-500 focus:outline-none"
                        type="text"
                        placeholder="add domain to scope, e.g. example.com"
                        value={newDomain}
                        onChange={(e) => setNewDomain(e.target.value)}
                        onKeyDown={(e) => e.key === 'Enter' && handleAddDomain()}
                    />
                    <button
                        className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white
                                   hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-40"
                        onClick={handleAddDomain}
                        disabled={adding || !newDomain.trim()}
                    >
                        Add to scope
                    </button>
                </div>

                {addErr && (
                    <div className="mt-3 rounded-md border border-red-800 bg-red-950/50 px-3 py-2 text-sm text-red-300">
                        {addErr}
                    </div>
                )}
                {loadErr && (
                    <div className="mt-3 rounded-md border border-red-800 bg-red-950/50 px-3 py-2 text-sm text-red-300">
                        {loadErr}
                    </div>
                )}

                <div className="mt-4">
                    {loading && <p className="text-sm text-slate-500">Loading...</p>}
                    {!loading && roots.length === 0 && (
                        <p className="text-sm text-slate-500">
                            Hali hech narsa yo'q — scope bo'sh. Yuqoridagi input orqali domen qo'shing.
                        </p>
                    )}
                    {!loading && roots.length > 0 && (
                        <ul className="space-y-1">
                            {roots.map((r) => (
                                <li
                                    key={r}
                                    className="rounded-md border border-slate-800 bg-slate-900 px-3 py-1.5 font-mono text-sm text-emerald-400"
                                >
                                    {r}
                                </li>
                            ))}
                        </ul>
                    )}
                </div>
            </div>

            <div className="mt-6 rounded-md border border-slate-800 bg-slate-900/50 p-4">
                <h2 className="text-sm font-semibold text-slate-300">Telegram Notifications</h2>
                <div className="mt-3 flex items-center gap-2 text-sm">
                    <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                            tgEnabled ? 'bg-emerald-900 text-emerald-300' : 'bg-slate-800 text-slate-400'
                        }`}
                    >
                        {tgEnabled ? 'enabled' : 'disabled'}
                    </span>
                </div>
                <p className="mt-2 text-xs text-slate-500">
                    RADARX_TG_TOKEN / RADARX_TG_CHAT_ID environment variable'lar orqali sozlanadi.
                </p>
            </div>
        </div>
    );
}

export default SettingsView;
