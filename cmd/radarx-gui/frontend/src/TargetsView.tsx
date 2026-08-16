import {useEffect, useState} from 'react';
import {AddTarget, AuthorizeTarget, ListTargets} from '../wailsjs/go/main/App';
import {model} from '../wailsjs/go/models';

function formatDate(v: any): string {
    if (!v) return 'never';
    const d = new Date(v);
    if (isNaN(d.getTime()) || d.getFullYear() <= 1) return 'never';
    return d.toLocaleString();
}

function TargetsView() {
    const [targets, setTargets] = useState<model.Target[]>([]);
    const [loading, setLoading] = useState(true);
    const [loadErr, setLoadErr] = useState('');

    const [root, setRoot] = useState('');
    const [label, setLabel] = useState('');
    const [interval, setInterval_] = useState(60);
    const [formErr, setFormErr] = useState('');
    const [submitting, setSubmitting] = useState(false);

    // Shown when AddTarget refuses an out-of-scope domain — the operator
    // must explicitly authorize it before it can be added.
    const [pendingAuth, setPendingAuth] = useState<string | null>(null);
    const [authorizing, setAuthorizing] = useState(false);

    async function refresh() {
        setLoading(true);
        setLoadErr('');
        try {
            const res = await ListTargets();
            setTargets(res || []);
        } catch (err) {
            setLoadErr(String(err));
        } finally {
            setLoading(false);
        }
    }

    useEffect(() => {
        refresh();
    }, []);

    async function handleAdd() {
        const r = root.trim().toLowerCase();
        if (!r) return;
        setFormErr('');
        setSubmitting(true);
        try {
            await AddTarget(r, label.trim(), interval);
            setRoot('');
            setLabel('');
            setInterval_(60);
            setPendingAuth(null);
            await refresh();
        } catch (err) {
            const msg = String(err);
            if (msg.includes('not authorized')) {
                setPendingAuth(r);
            } else {
                setFormErr(msg);
            }
        } finally {
            setSubmitting(false);
        }
    }

    async function handleAuthorizeAndAdd() {
        if (!pendingAuth) return;
        setAuthorizing(true);
        setFormErr('');
        try {
            await AuthorizeTarget(pendingAuth);
            await AddTarget(pendingAuth, label.trim(), interval);
            setRoot('');
            setLabel('');
            setInterval_(60);
            setPendingAuth(null);
            await refresh();
        } catch (err) {
            setFormErr(String(err));
        } finally {
            setAuthorizing(false);
        }
    }

    return (
        <div>
            <h1 className="text-2xl font-bold tracking-tight text-emerald-400">Targets</h1>
            <p className="mt-1 text-sm text-slate-400">
                Domains RadarX is authorized to monitor.
            </p>

            <div className="mt-6 rounded-md border border-slate-800 bg-slate-900/50 p-4">
                <h2 className="text-sm font-semibold text-slate-300">+ Add Target</h2>
                <div className="mt-3 flex flex-wrap gap-3">
                    <input
                        className="flex-1 min-w-[160px] rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm
                                   placeholder:text-slate-500 focus:border-emerald-500 focus:outline-none"
                        type="text"
                        placeholder="root domain, e.g. example.com"
                        value={root}
                        onChange={(e) => setRoot(e.target.value)}
                    />
                    <input
                        className="flex-1 min-w-[160px] rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm
                                   placeholder:text-slate-500 focus:border-emerald-500 focus:outline-none"
                        type="text"
                        placeholder="label (optional)"
                        value={label}
                        onChange={(e) => setLabel(e.target.value)}
                    />
                    <input
                        className="w-32 rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm
                                   placeholder:text-slate-500 focus:border-emerald-500 focus:outline-none"
                        type="number"
                        min={1}
                        placeholder="interval (min)"
                        value={interval}
                        onChange={(e) => setInterval_(Number(e.target.value) || 60)}
                    />
                    <button
                        className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white
                                   hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-40"
                        onClick={handleAdd}
                        disabled={submitting || !root.trim()}
                    >
                        Add Target
                    </button>
                </div>

                {formErr && (
                    <div className="mt-3 rounded-md border border-red-800 bg-red-950/50 px-3 py-2 text-sm text-red-300">
                        {formErr}
                    </div>
                )}

                {pendingAuth && (
                    <div className="mt-3 rounded-md border border-amber-800 bg-amber-950/50 px-3 py-3 text-sm text-amber-200">
                        <p>
                            Bu domen hali sizning ruxsat etilgan scope'ingizda yo'q: <span className="font-mono">{pendingAuth}</span>.
                            Uni scan qilishga ruxsatingiz borligini tasdiqlaysizmi (o'zingizning infratuzilmangiz yoki
                            ruxsat berilgan bug-bounty scope)?
                        </p>
                        <div className="mt-3 flex gap-3">
                            <button
                                className="rounded-md bg-amber-600 px-3 py-1.5 text-xs font-medium text-white
                                           hover:bg-amber-500 disabled:cursor-not-allowed disabled:opacity-40"
                                onClick={handleAuthorizeAndAdd}
                                disabled={authorizing}
                            >
                                Ha, ruxsat beraman
                            </button>
                            <button
                                className="rounded-md bg-slate-700 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-600"
                                onClick={() => setPendingAuth(null)}
                            >
                                Bekor qilish
                            </button>
                        </div>
                    </div>
                )}
            </div>

            {loadErr && (
                <div className="mt-4 rounded-md border border-red-800 bg-red-950/50 px-3 py-2 text-sm text-red-300">
                    {loadErr}
                </div>
            )}

            <div className="mt-6 overflow-hidden rounded-md border border-slate-800">
                <table className="w-full text-left text-sm">
                    <thead className="bg-slate-900 text-slate-400">
                        <tr>
                            <th className="px-3 py-2 font-medium">Root</th>
                            <th className="px-3 py-2 font-medium">Label</th>
                            <th className="px-3 py-2 font-medium">Interval</th>
                            <th className="px-3 py-2 font-medium">Last Scan</th>
                        </tr>
                    </thead>
                    <tbody>
                        {!loading && targets.length === 0 && (
                            <tr>
                                <td colSpan={4} className="px-3 py-6 text-center text-slate-500">
                                    No targets yet. Add one above.
                                </td>
                            </tr>
                        )}
                        {loading && (
                            <tr>
                                <td colSpan={4} className="px-3 py-6 text-center text-slate-500">
                                    Loading...
                                </td>
                            </tr>
                        )}
                        {targets.map((t) => (
                            <tr key={t.id} className="border-t border-slate-800">
                                <td className="px-3 py-2 font-mono text-emerald-400">{t.root}</td>
                                <td className="px-3 py-2 text-slate-300">{t.label || '—'}</td>
                                <td className="px-3 py-2 text-slate-400">{t.interval_m}m</td>
                                <td className="px-3 py-2 text-slate-400">{formatDate(t.last_scan_at)}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}

export default TargetsView;
