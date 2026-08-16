import {useEffect, useState} from 'react';
import {
    AuthorizeTarget,
    DetectTelegramChatID,
    GetScopeRoots,
    GetTelegramStatus,
    IsMonitoring,
    SaveTelegramToken,
    SendTelegramTest,
    StartMonitoring,
    StopMonitoring,
} from '../wailsjs/go/main/App';

function SettingsView() {
    const [roots, setRoots] = useState<string[]>([]);
    const [loading, setLoading] = useState(true);
    const [loadErr, setLoadErr] = useState('');

    const [newDomain, setNewDomain] = useState('');
    const [addErr, setAddErr] = useState('');
    const [adding, setAdding] = useState(false);

    const [tgEnabled, setTgEnabled] = useState(false);
    const [tgToken, setTgToken] = useState('');
    const [tgSaving, setTgSaving] = useState(false);
    const [tgDetecting, setTgDetecting] = useState(false);
    const [tgTesting, setTgTesting] = useState(false);
    const [tgMsg, setTgMsg] = useState('');
    const [tgErr, setTgErr] = useState('');

    const [monitoring, setMonitoring] = useState(false);
    const [monLoading, setMonLoading] = useState(false);
    const [monErr, setMonErr] = useState('');

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

    async function refreshMonitoring() {
        try {
            setMonitoring(await IsMonitoring());
        } catch {
            setMonitoring(false);
        }
    }

    function refreshTelegramStatus() {
        GetTelegramStatus().then(setTgEnabled).catch(() => setTgEnabled(false));
    }

    useEffect(() => {
        refresh();
        refreshMonitoring();
        refreshTelegramStatus();
    }, []);

    async function handleSaveToken() {
        const tok = tgToken.trim();
        if (!tok) return;
        setTgSaving(true);
        setTgMsg('');
        setTgErr('');
        try {
            await SaveTelegramToken(tok);
            setTgToken('');
            setTgMsg('Token saqlandi. Endi botga Telegramda bitta xabar yozing (masalan /start), keyin "Chat ID aniqlash" tugmasini bosing.');
            refreshTelegramStatus();
        } catch (err) {
            setTgErr(String(err));
        } finally {
            setTgSaving(false);
        }
    }

    async function handleDetectChatID() {
        setTgDetecting(true);
        setTgMsg('');
        setTgErr('');
        try {
            const chatID = await DetectTelegramChatID();
            setTgMsg(`Chat ID aniqlandi va saqlandi: ${chatID}. Endi "Test xabar yuborish" bilan tekshiring.`);
            refreshTelegramStatus();
        } catch (err) {
            setTgErr(String(err));
        } finally {
            setTgDetecting(false);
        }
    }

    async function handleSendTest() {
        setTgTesting(true);
        setTgMsg('');
        setTgErr('');
        try {
            await SendTelegramTest();
            setTgMsg('Test xabar yuborildi — Telegramni tekshiring.');
        } catch (err) {
            setTgErr(String(err));
        } finally {
            setTgTesting(false);
        }
    }

    async function toggleMonitoring() {
        setMonLoading(true);
        setMonErr('');
        try {
            if (monitoring) {
                await StopMonitoring();
            } else {
                await StartMonitoring();
            }
            await refreshMonitoring();
        } catch (err) {
            setMonErr(String(err));
        } finally {
            setMonLoading(false);
        }
    }

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
                <h2 className="text-sm font-semibold text-slate-300">Continuous Monitoring</h2>
                <p className="mt-1 text-xs text-slate-500">
                    Periodically rescans every enabled target on its interval, diffs against the
                    last snapshot, and notifies you (desktop{tgEnabled ? ' + Telegram' : ''}) when
                    something changes. Closing the window while this is on keeps it running in the
                    background instead of quitting.
                </p>

                <div className="mt-3 flex items-center gap-3">
                    <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                            monitoring ? 'bg-emerald-900 text-emerald-300' : 'bg-slate-800 text-slate-400'
                        }`}
                    >
                        {monitoring ? 'running' : 'stopped'}
                    </span>
                    <button
                        className={`rounded-md px-4 py-2 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40 ${
                            monitoring ? 'bg-red-700 hover:bg-red-600' : 'bg-emerald-600 hover:bg-emerald-500'
                        }`}
                        onClick={toggleMonitoring}
                        disabled={monLoading}
                    >
                        {monitoring ? 'Stop monitoring' : 'Start monitoring'}
                    </button>
                </div>

                {monErr && (
                    <div className="mt-3 rounded-md border border-red-800 bg-red-950/50 px-3 py-2 text-sm text-red-300">
                        {monErr}
                    </div>
                )}
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

                <ol className="mt-3 space-y-3 text-xs text-slate-500">
                    <li>
                        1. @BotFather orqali bot yarating, tokenni shu yerga kiriting va saqlang (token
                        faqat sizning kompyuteringizda, <span className="font-mono">~/.radarx/config.json</span> da
                        saqlanadi — hech qayerga yuborilmaydi).
                        <div className="mt-2 flex gap-3">
                            <input
                                className="flex-1 min-w-[200px] rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm
                                           placeholder:text-slate-500 focus:border-emerald-500 focus:outline-none"
                                type="password"
                                placeholder="bot token"
                                value={tgToken}
                                onChange={(e) => setTgToken(e.target.value)}
                            />
                            <button
                                className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white
                                           hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-40"
                                onClick={handleSaveToken}
                                disabled={tgSaving || !tgToken.trim()}
                            >
                                {tgSaving ? 'Saving...' : 'Save token'}
                            </button>
                        </div>
                    </li>
                    <li>
                        2. Telegramda o'z botingizga bitta xabar yozing (masalan{' '}
                        <span className="font-mono">/start</span>), keyin bosing — faqat shu botga yozgan odam
                        xabarnoma oladi:
                        <div className="mt-2">
                            <button
                                className="rounded-md border border-slate-700 bg-slate-900 px-4 py-2 text-sm font-medium
                                           text-emerald-400 hover:border-emerald-600 disabled:cursor-not-allowed disabled:opacity-40"
                                onClick={handleDetectChatID}
                                disabled={tgDetecting}
                            >
                                {tgDetecting ? 'Detecting...' : 'Chat ID aniqlash'}
                            </button>
                        </div>
                    </li>
                    <li>
                        3. Tekshiring:
                        <div className="mt-2">
                            <button
                                className="rounded-md border border-slate-700 bg-slate-900 px-4 py-2 text-sm font-medium
                                           text-emerald-400 hover:border-emerald-600 disabled:cursor-not-allowed disabled:opacity-40"
                                onClick={handleSendTest}
                                disabled={tgTesting}
                            >
                                {tgTesting ? 'Sending...' : 'Test xabar yuborish'}
                            </button>
                        </div>
                    </li>
                </ol>

                {tgMsg && (
                    <div className="mt-3 rounded-md border border-emerald-800 bg-emerald-950/50 px-3 py-2 text-sm text-emerald-300">
                        {tgMsg}
                    </div>
                )}
                {tgErr && (
                    <div className="mt-3 rounded-md border border-red-800 bg-red-950/50 px-3 py-2 text-sm text-red-300">
                        {tgErr}
                    </div>
                )}

                <p className="mt-3 text-xs text-slate-500">
                    Muqobil: RADARX_TG_TOKEN / RADARX_TG_CHAT_ID environment variable'lar orqali ham sozlash mumkin
                    (ular saqlangan konfiguratsiyadan ustuvor).
                </p>
            </div>
        </div>
    );
}

export default SettingsView;
