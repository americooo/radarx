import {useEffect, useRef, useState} from 'react';
import {StartScan, StopScan} from '../wailsjs/go/main/App';
import {EventsOff, EventsOn} from '../wailsjs/runtime/runtime';

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
    const styles: Record<ScanStatus, string> = {
        idle: 'bg-slate-800 text-slate-300',
        running: 'bg-emerald-900 text-emerald-300 animate-pulse',
        done: 'bg-emerald-900 text-emerald-300',
        stopped: 'bg-amber-900 text-amber-300',
        error: 'bg-red-900 text-red-300',
    };
    return (
        <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${styles[status]}`}>
            {status}
        </span>
    );
}

function ScanView() {
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
            <h1 className="text-2xl font-bold tracking-tight text-emerald-400">
                RadarX
            </h1>
            <p className="mt-1 text-sm text-slate-400">
                Attack surface scanner — enter a target domain and scan.
            </p>

            <div className="mt-6 flex gap-3">
                <input
                    className="flex-1 rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm
                               placeholder:text-slate-500 focus:border-emerald-500 focus:outline-none"
                    type="text"
                    placeholder="example.com"
                    value={target}
                    disabled={running}
                    onChange={(e) => setTarget(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleScan()}
                />
                <button
                    className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white
                               hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-40"
                    onClick={handleScan}
                    disabled={running || !target.trim()}
                >
                    Scan
                </button>
                <button
                    className="rounded-md bg-red-700 px-4 py-2 text-sm font-medium text-white
                               hover:bg-red-600 disabled:cursor-not-allowed disabled:opacity-40"
                    onClick={handleStop}
                    disabled={!running}
                >
                    Stop
                </button>
            </div>

            <div className="mt-4 flex items-center gap-2 text-sm">
                <span className="text-slate-400">Status:</span>
                <StatusBadge status={status} />
                <span className="text-slate-500">
                    {assets.length} asset{assets.length === 1 ? '' : 's'} found
                </span>
            </div>

            {errorMsg && (
                <div className="mt-3 rounded-md border border-red-800 bg-red-950/50 px-3 py-2 text-sm text-red-300">
                    {errorMsg}
                </div>
            )}

            <div className="mt-6 overflow-hidden rounded-md border border-slate-800">
                <table className="w-full text-left text-sm">
                    <thead className="bg-slate-900 text-slate-400">
                        <tr>
                            <th className="px-3 py-2 font-medium">Kind</th>
                            <th className="px-3 py-2 font-medium">Key</th>
                            <th className="px-3 py-2 font-medium">Detail</th>
                        </tr>
                    </thead>
                    <tbody>
                        {assets.length === 0 && (
                            <tr>
                                <td colSpan={3} className="px-3 py-6 text-center text-slate-500">
                                    No results yet.
                                </td>
                            </tr>
                        )}
                        {assets.map((a, i) => (
                            <tr key={`${a.kind}-${a.key}-${i}`} className="border-t border-slate-800">
                                <td className="px-3 py-2 text-emerald-400">{a.kind}</td>
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
