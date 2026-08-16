import {useEffect, useState} from 'react';
import {ExportReport, GetLatestSnapshot, ListTargets} from '../wailsjs/go/main/App';
import {model} from '../wailsjs/go/models';
import {assetDetail} from './ScanView';
import {pluralKey, useI18n} from './i18n/LanguageContext';

function ResultsView() {
    const {t} = useI18n();
    const [targets, setTargets] = useState<model.Target[]>([]);
    const [selected, setSelected] = useState('');
    const [snapshot, setSnapshot] = useState<model.Snapshot | null>(null);
    const [snapErr, setSnapErr] = useState('');
    const [loading, setLoading] = useState(false);
    const [filter, setFilter] = useState('');

    const [exporting, setExporting] = useState(false);
    const [exportMsg, setExportMsg] = useState('');
    const [exportErr, setExportErr] = useState('');

    useEffect(() => {
        ListTargets().then((res) => {
            const list = res || [];
            setTargets(list);
            if (list.length > 0) setSelected(list[0].id);
        });
    }, []);

    useEffect(() => {
        if (!selected) {
            setSnapshot(null);
            return;
        }
        setLoading(true);
        setSnapErr('');
        setSnapshot(null);
        setExportMsg('');
        setExportErr('');
        GetLatestSnapshot(selected)
            .then((snap) => {
                // Go's json.Marshal renders a nil slice as `null`; a
                // zero-asset snapshot would otherwise crash the
                // .length/.map calls below.
                snap.assets = snap.assets || [];
                setSnapshot(snap);
            })
            .catch((err) => setSnapErr(String(err)))
            .finally(() => setLoading(false));
    }, [selected]);

    async function handleExport() {
        if (!selected) return;
        setExporting(true);
        setExportMsg('');
        setExportErr('');
        try {
            const path = await ExportReport(selected);
            setExportMsg(`${t('results.savedTo')} ${path}`);
        } catch (err) {
            setExportErr(String(err));
        } finally {
            setExporting(false);
        }
    }

    const assets = (snapshot?.assets || []).filter((a) => {
        if (!filter.trim()) return true;
        const q = filter.trim().toLowerCase();
        return (
            a.key.toLowerCase().includes(q) ||
            (a.host || '').toLowerCase().includes(q) ||
            (a.title || '').toLowerCase().includes(q)
        );
    });

    return (
        <div>
            <h1 className="text-2xl font-bold tracking-tight text-blue-400">{t('results.title')}</h1>
            <p className="mt-1 text-sm text-slate-400">
                {t('results.subtitle')}
            </p>

            <div className="mt-6 flex flex-wrap gap-3">
                <select
                    className="rounded-md border border-blue-900/40 bg-slate-900/40 px-3 py-2 text-sm backdrop-blur-md focus:border-blue-500 focus:outline-none"
                    value={selected}
                    onChange={(e) => setSelected(e.target.value)}
                >
                    {targets.length === 0 && <option value="">{t('results.noTargets')}</option>}
                    {targets.map((tgt) => (
                        <option key={tgt.id} value={tgt.id}>
                            {tgt.root}
                        </option>
                    ))}
                </select>
                <input
                    className="flex-1 min-w-[200px] rounded-md border border-blue-900/40 bg-slate-900/40 px-3 py-2 text-sm backdrop-blur-md
                               placeholder:text-slate-500 focus:border-blue-500 focus:outline-none"
                    type="text"
                    placeholder={t('results.filterPlaceholder')}
                    value={filter}
                    onChange={(e) => setFilter(e.target.value)}
                />
                <button
                    className="rounded-md border border-blue-900/40 bg-slate-900/40 px-4 py-2 text-sm font-medium text-blue-400 backdrop-blur-md
                               hover:border-blue-600 disabled:cursor-not-allowed disabled:opacity-40"
                    onClick={handleExport}
                    disabled={!selected || !snapshot || exporting}
                >
                    {exporting ? t('results.exporting') : t('results.exportButton')}
                </button>
            </div>

            {exportMsg && (
                <div className="mt-4 rounded-md border border-blue-800 bg-blue-950/50 px-3 py-2 text-sm text-blue-300">
                    {exportMsg}
                </div>
            )}
            {exportErr && (
                <div className="mt-4 rounded-md border border-red-800 bg-red-950/50 px-3 py-2 text-sm text-red-300">
                    {exportErr}
                </div>
            )}

            {loading && <p className="mt-4 text-sm text-slate-500">{t('results.loading')}</p>}

            {!loading && snapErr && (
                <div className="mt-4 rounded-md border border-slate-700 bg-slate-900/50 px-3 py-2 text-sm text-slate-400">
                    {snapErr}
                </div>
            )}

            {!loading && snapshot && (
                <>
                    <div className="mt-4 text-sm text-slate-500">
                        {t('results.taken')} {new Date(snapshot.taken_at).toLocaleString()} · {assets.length}{' '}
                        {t(pluralKey('results.assetsShown', snapshot.assets.length), {total: snapshot.assets.length})}
                    </div>

                    <div className="mt-3 overflow-hidden rounded-md border border-blue-900/40 bg-slate-900/20 backdrop-blur-md">
                        <table className="w-full text-left text-sm">
                            <thead className="bg-slate-900/60 text-slate-400">
                                <tr>
                                    <th className="px-3 py-2 font-medium">{t('results.colKind')}</th>
                                    <th className="px-3 py-2 font-medium">{t('results.colKey')}</th>
                                    <th className="px-3 py-2 font-medium">{t('results.colDetail')}</th>
                                </tr>
                            </thead>
                            <tbody>
                                {assets.length === 0 && (
                                    <tr>
                                        <td colSpan={3} className="px-3 py-6 text-center text-slate-500">
                                            {t('results.noMatching')}
                                        </td>
                                    </tr>
                                )}
                                {assets.map((a, i) => (
                                    <tr key={`${a.kind}-${a.key}-${i}`} className="border-t border-blue-900/30">
                                        <td className="px-3 py-2 text-blue-400">{a.kind}</td>
                                        <td className="px-3 py-2 font-mono">{a.key}</td>
                                        <td className="px-3 py-2 text-slate-400">{assetDetail(a)}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </>
            )}
        </div>
    );
}

export default ResultsView;
