import {useEffect, useState} from 'react';
import {AddTarget, AuthorizeTarget, ListTargets} from '../wailsjs/go/main/App';
import {model} from '../wailsjs/go/models';
import {useI18n} from './i18n/LanguageContext';

function TargetsView() {
    const {t} = useI18n();

    function formatDate(v: any): string {
        if (!v) return t('targets.never');
        const d = new Date(v);
        if (isNaN(d.getTime()) || d.getFullYear() <= 1) return t('targets.never');
        return d.toLocaleString();
    }

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
            <h1 className="text-2xl font-bold tracking-tight text-blue-400">{t('targets.title')}</h1>
            <p className="mt-1 text-sm text-slate-400">
                {t('targets.subtitle')}
            </p>

            <div className="mt-6 rounded-md border border-blue-900/40 bg-slate-900/40 p-4 backdrop-blur-md">
                <h2 className="text-sm font-semibold text-slate-300">{t('targets.addTarget')}</h2>
                <div className="mt-3 flex flex-wrap gap-3">
                    <input
                        className="flex-1 min-w-[160px] rounded-md border border-blue-900/40 bg-slate-900/40 px-3 py-2 text-sm backdrop-blur-md
                                   placeholder:text-slate-500 focus:border-blue-500 focus:outline-none"
                        type="text"
                        placeholder={t('targets.rootPlaceholder')}
                        value={root}
                        onChange={(e) => setRoot(e.target.value)}
                    />
                    <input
                        className="flex-1 min-w-[160px] rounded-md border border-blue-900/40 bg-slate-900/40 px-3 py-2 text-sm backdrop-blur-md
                                   placeholder:text-slate-500 focus:border-blue-500 focus:outline-none"
                        type="text"
                        placeholder={t('targets.labelPlaceholder')}
                        value={label}
                        onChange={(e) => setLabel(e.target.value)}
                    />
                    <input
                        className="w-32 rounded-md border border-blue-900/40 bg-slate-900/40 px-3 py-2 text-sm backdrop-blur-md
                                   placeholder:text-slate-500 focus:border-blue-500 focus:outline-none"
                        type="number"
                        min={1}
                        placeholder={t('targets.intervalPlaceholder')}
                        value={interval}
                        onChange={(e) => setInterval_(Number(e.target.value) || 60)}
                    />
                    <button
                        className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white
                                   hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-40"
                        onClick={handleAdd}
                        disabled={submitting || !root.trim()}
                    >
                        {t('targets.addButton')}
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
                            {t('targets.pendingAuth')} <span className="font-mono">{pendingAuth}</span>.{' '}
                            {t('targets.pendingAuthQuestion')}
                        </p>
                        <div className="mt-3 flex gap-3">
                            <button
                                className="rounded-md bg-amber-600 px-3 py-1.5 text-xs font-medium text-white
                                           hover:bg-amber-500 disabled:cursor-not-allowed disabled:opacity-40"
                                onClick={handleAuthorizeAndAdd}
                                disabled={authorizing}
                            >
                                {t('targets.authorizeYes')}
                            </button>
                            <button
                                className="rounded-md bg-slate-700 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-600"
                                onClick={() => setPendingAuth(null)}
                            >
                                {t('targets.authorizeCancel')}
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

            <div className="mt-6 overflow-hidden rounded-md border border-blue-900/40 bg-slate-900/20 backdrop-blur-md">
                <table className="w-full text-left text-sm">
                    <thead className="bg-slate-900/60 text-slate-400">
                        <tr>
                            <th className="px-3 py-2 font-medium">{t('targets.colRoot')}</th>
                            <th className="px-3 py-2 font-medium">{t('targets.colLabel')}</th>
                            <th className="px-3 py-2 font-medium">{t('targets.colInterval')}</th>
                            <th className="px-3 py-2 font-medium">{t('targets.colLastScan')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        {!loading && targets.length === 0 && (
                            <tr>
                                <td colSpan={4} className="px-3 py-6 text-center text-slate-500">
                                    {t('targets.noTargets')}
                                </td>
                            </tr>
                        )}
                        {loading && (
                            <tr>
                                <td colSpan={4} className="px-3 py-6 text-center text-slate-500">
                                    {t('targets.loading')}
                                </td>
                            </tr>
                        )}
                        {targets.map((tgt) => (
                            <tr key={tgt.id} className="border-t border-blue-900/30">
                                <td className="px-3 py-2 font-mono text-blue-400">{tgt.root}</td>
                                <td className="px-3 py-2 text-slate-300">{tgt.label || '—'}</td>
                                <td className="px-3 py-2 text-slate-400">{tgt.interval_m}m</td>
                                <td className="px-3 py-2 text-slate-400">{formatDate(tgt.last_scan_at)}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
}

export default TargetsView;
