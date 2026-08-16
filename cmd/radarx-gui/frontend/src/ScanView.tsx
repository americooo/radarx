import {useEffect, useRef, useState} from 'react';
import {StartScan, StopScan} from '../wailsjs/go/main/App';
import {EventsOff, EventsOn} from '../wailsjs/runtime/runtime';
import {pluralKey, useI18n} from './i18n/LanguageContext';

// Mirrors internal/model.Asset (JSON tags) — only the fields the UI shows.
export interface Asset {
    kind: string;
    key: string;
    host?: string;
    status_code?: number;
    title?: string;
    server?: string;
    cert_cn?: string;
    port?: number;
}

interface ScanDoneEvent {
    err?: string;
    cancelled: boolean;
}

type ScanStatus = 'idle' | 'running' | 'done' | 'stopped' | 'error';

export function assetDetail(a: Asset): string {
    const parts: string[] = [];
    if (a.status_code) parts.push(`HTTP ${a.status_code}`);
    if (a.title) parts.push(a.title);
    if (a.server) parts.push(a.server);
    if (a.cert_cn) parts.push(`CN=${a.cert_cn}`);
    if (a.port) parts.push(`port ${a.port}`);
    return parts.join(' · ');
}

function StatusBadge({status}: {status: ScanStatus}) {
    const {t} = useI18n();
    const styles: Record<ScanStatus, string> = {
        idle: 'bg-slate-800 text-slate-300',
        running: 'bg-blue-900 text-blue-300 animate-pulse',
        done: 'bg-blue-900 text-blue-300',
        stopped: 'bg-amber-900 text-amber-300',
        error: 'bg-red-900 text-red-300',
    };
    return (
        <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${styles[status]}`}>
            {t(`scan.status.${status}` as const)}
        </span>
    );
}

function ScanView() {
    const {t} = useI18n();
    const [target, setTarget] = useState('');
    const [assets, setAssets] = useState<Asset[]>([]);
    const [status, setStatus] = useState<ScanStatus>('idle');
    const [errorMsg, setErrorMsg] = useState('');
    const statusRef = useRef(status);
    statusRef.current = status;

    useEffect(() => {
        EventsOn('scan:asset', (asset: Asset) => {
            setAssets((prev) => [...prev, asset]);
        });
        EventsOn('scan:done', (ev: ScanDoneEvent) => {
            if (ev.cancelled) {
                setStatus('stopped');
            } else if (ev.err) {
                setErrorMsg(ev.err);
                setStatus('error');
            } else {
                setStatus('done');
            }
        });

        return () => {
            EventsOff('scan:asset');
            EventsOff('scan:done');
        };
    }, []);

    async function handleScan() {
        if (!target.trim() || status === 'running') return;
        setAssets([]);
        setErrorMsg('');
        setStatus('running');
        try {
            await StartScan(target.trim());
        } catch (err) {
            setErrorMsg(String(err));
            setStatus('error');
        }
    }

    async function handleStop() {
        try {
            await StopScan();
        } catch (err) {
            setErrorMsg(String(err));
        }
    }

    const running = status === 'running';

    return (
        <div>
            <h1 className="text-2xl font-bold tracking-tight text-blue-400">
                {t('scan.title')}
            </h1>
            <p className="mt-1 text-sm text-slate-400">
                {t('scan.subtitle')}
            </p>

            <div className="mt-6 flex gap-3">
                <input
                    className="flex-1 rounded-md border border-blue-900/40 bg-slate-900/40 px-3 py-2 text-sm backdrop-blur-md
                               placeholder:text-slate-500 focus:border-blue-500 focus:outline-none"
                    type="text"
                    placeholder={t('scan.placeholder')}
                    value={target}
                    disabled={running}
                    onChange={(e) => setTarget(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleScan()}
                />
                <button
                    className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white
                               hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-40"
                    onClick={handleScan}
                    disabled={running || !target.trim()}
                >
                    {t('scan.scanButton')}
                </button>
                <button
                    className="rounded-md bg-red-700 px-4 py-2 text-sm font-medium text-white
                               hover:bg-red-600 disabled:cursor-not-allowed disabled:opacity-40"
                    onClick={handleStop}
                    disabled={!running}
                >
                    {t('scan.stopButton')}
                </button>
            </div>

            <div className="mt-4 flex items-center gap-2 text-sm">
                <span className="text-slate-400">{t('scan.status')}</span>
                <StatusBadge status={status} />
                <span className="text-slate-500">
                    {assets.length} {t(pluralKey('scan.assetsFound', assets.length))}
                </span>
            </div>

            {errorMsg && (
                <div className="mt-3 rounded-md border border-red-800 bg-red-950/50 px-3 py-2 text-sm text-red-300">
                    {errorMsg}
                </div>
            )}

            <div className="mt-6 overflow-hidden rounded-md border border-blue-900/40 bg-slate-900/20 backdrop-blur-md">
                <table className="w-full text-left text-sm">
                    <thead className="bg-slate-900/60 text-slate-400">
                        <tr>
                            <th className="px-3 py-2 font-medium">{t('scan.colKind')}</th>
                            <th className="px-3 py-2 font-medium">{t('scan.colKey')}</th>
                            <th className="px-3 py-2 font-medium">{t('scan.colDetail')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        {assets.length === 0 && (
                            <tr>
                                <td colSpan={3} className="px-3 py-6 text-center text-slate-500">
                                    {t('scan.noResults')}
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
        </div>
    );
}

export default ScanView;
