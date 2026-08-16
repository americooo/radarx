import {useEffect, useState} from 'react';
import {GetDiff, ListTargets} from '../wailsjs/go/main/App';
import {model} from '../wailsjs/go/models';
import {pluralKey, useI18n} from './i18n/LanguageContext';
import type {TranslationKey} from './i18n/translations';

function changeRowClass(type: string): string {
    switch (type) {
        case 'new':
            return 'border-t border-blue-900/30 bg-blue-950/40';
        case 'removed':
            return 'border-t border-blue-900/30 bg-red-950/40';
        case 'modified':
            return 'border-t border-blue-900/30 bg-amber-950/40';
        default:
            return 'border-t border-blue-900/30';
    }
}

function changeLabel(type: string, t: (key: TranslationKey) => string): {text: string; className: string} {
    switch (type) {
        case 'new':
            return {text: t('diff.new'), className: 'text-blue-400'};
        case 'removed':
            return {text: t('diff.removed'), className: 'text-red-400'};
        case 'modified':
            return {text: t('diff.modified'), className: 'text-amber-400'};
        default:
            return {text: type, className: 'text-slate-400'};
    }
}

function DiffView() {
    const {t} = useI18n();
    const [targets, setTargets] = useState<model.Target[]>([]);
    const [selected, setSelected] = useState('');
    const [result, setResult] = useState<model.DiffResult | null>(null);
    const [diffErr, setDiffErr] = useState('');
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        ListTargets().then((res) => {
            const list = res || [];
            setTargets(list);
            if (list.length > 0) setSelected(list[0].id);
        });
    }, []);

    useEffect(() => {
        if (!selected) {
            setResult(null);
            return;
        }
        setLoading(true);
        setDiffErr('');
        setResult(null);
        GetDiff(selected)
            .then((d) => {
                // Go's json.Marshal renders a nil slice as `null`; a diff
                // with no changes (the common case) would otherwise crash
                // the length/map calls below.
                d.changes = d.changes || [];
                setResult(d);
            })
            .catch((err) => setDiffErr(String(err)))
            .finally(() => setLoading(false));
    }, [selected]);

    return (
        <div>
            <h1 className="text-2xl font-bold tracking-tight text-blue-400">{t('diff.title')}</h1>
            <p className="mt-1 text-sm text-slate-400">
                {t('diff.subtitle')}
            </p>

            <div className="mt-6">
                <select
                    className="rounded-md border border-blue-900/40 bg-slate-900/40 px-3 py-2 text-sm backdrop-blur-md focus:border-blue-500 focus:outline-none"
                    value={selected}
                    onChange={(e) => setSelected(e.target.value)}
                >
                    {targets.length === 0 && <option value="">{t('diff.noTargets')}</option>}
                    {targets.map((tgt) => (
                        <option key={tgt.id} value={tgt.id}>
                            {tgt.root}
                        </option>
                    ))}
                </select>
            </div>

            {loading && <p className="mt-4 text-sm text-slate-500">{t('diff.loading')}</p>}

            {!loading && diffErr && (
                <div className="mt-4 rounded-md border border-slate-700 bg-slate-900/50 px-3 py-2 text-sm text-slate-400">
                    {diffErr}
                </div>
            )}

            {!loading && result && (
                <>
                    <div className="mt-4 flex gap-4 text-sm text-slate-500">
                        <span>
                            {result.changes.length} {t(pluralKey('diff.totalChanges', result.changes.length))}
                        </span>
                    </div>

                    <div className="mt-3 overflow-hidden rounded-md border border-blue-900/40 bg-slate-900/20 backdrop-blur-md">
                        <table className="w-full text-left text-sm">
                            <thead className="bg-slate-900/60 text-slate-400">
                                <tr>
                                    <th className="px-3 py-2 font-medium">{t('diff.colChange')}</th>
                                    <th className="px-3 py-2 font-medium">{t('diff.colKind')}</th>
                                    <th className="px-3 py-2 font-medium">{t('diff.colKey')}</th>
                                    <th className="px-3 py-2 font-medium">{t('diff.colFields')}</th>
                                </tr>
                            </thead>
                            <tbody>
                                {result.changes.length === 0 && (
                                    <tr>
                                        <td colSpan={4} className="px-3 py-6 text-center text-slate-500">
                                            {t('diff.noChanges')}
                                        </td>
                                    </tr>
                                )}
                                {result.changes.map((c, i) => {
                                    const label = changeLabel(c.type, t);
                                    return (
                                        <tr key={`${c.type}-${c.key}-${i}`} className={changeRowClass(c.type)}>
                                            <td className={`px-3 py-2 font-medium ${label.className}`}>{label.text}</td>
                                            <td className="px-3 py-2 text-slate-300">{c.kind}</td>
                                            <td className="px-3 py-2 font-mono">{c.key}</td>
                                            <td className="px-3 py-2 text-slate-400">
                                                {(c.fields || []).join(', ')}
                                            </td>
                                        </tr>
                                    );
                                })}
                            </tbody>
                        </table>
                    </div>
                </>
            )}
        </div>
    );
}

export default DiffView;
