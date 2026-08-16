import {useEffect, useState} from 'react';
import {
    AuthorizeTarget,
    DetectTelegramChatID,
    GetScopeRoots,
    GetTelegramStatus,
    IsMonitoring,
    ListModules,
    SaveTelegramChatID,
    SaveTelegramToken,
    SendTelegramTest,
    SetModuleEnabled,
    StartMonitoring,
    StopMonitoring,
} from '../wailsjs/go/main/App';
import {main} from '../wailsjs/go/models';
import {useI18n} from './i18n/LanguageContext';
import {Language} from './i18n/storage';

function SettingsView() {
    const {language, setLanguage, t} = useI18n();
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
    const [tgManualChatID, setTgManualChatID] = useState('');
    const [tgSavingChatID, setTgSavingChatID] = useState(false);
    const [tgTesting, setTgTesting] = useState(false);
    const [tgMsg, setTgMsg] = useState('');
    const [tgErr, setTgErr] = useState('');

    const [monitoring, setMonitoring] = useState(false);
    const [monLoading, setMonLoading] = useState(false);
    const [monErr, setMonErr] = useState('');

    const [modules, setModules] = useState<main.ModuleInfo[]>([]);
    const [modulesLoading, setModulesLoading] = useState(true);
    const [modulesErr, setModulesErr] = useState('');
    const [togglingModule, setTogglingModule] = useState('');

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

    async function refreshModules() {
        setModulesLoading(true);
        setModulesErr('');
        try {
            const res = await ListModules();
            setModules(res || []);
        } catch (err) {
            setModulesErr(String(err));
        } finally {
            setModulesLoading(false);
        }
    }

    useEffect(() => {
        refresh();
        refreshMonitoring();
        refreshTelegramStatus();
        refreshModules();
    }, []);

    async function handleToggleModule(m: main.ModuleInfo) {
        setTogglingModule(m.name);
        setModulesErr('');
        try {
            await SetModuleEnabled(m.name, !m.enabled);
            await refreshModules();
        } catch (err) {
            setModulesErr(String(err));
        } finally {
            setTogglingModule('');
        }
    }

    async function handleSaveToken() {
        const tok = tgToken.trim();
        if (!tok) return;
        setTgSaving(true);
        setTgMsg('');
        setTgErr('');
        try {
            await SaveTelegramToken(tok);
            setTgToken('');
            setTgMsg(t('settings.telegram.tokenSaved'));
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
            setTgMsg(t('settings.telegram.chatIdDetected', {chatID}));
            refreshTelegramStatus();
        } catch (err) {
            setTgErr(String(err));
        } finally {
            setTgDetecting(false);
        }
    }

    async function handleSaveChatIDManually() {
        const id = tgManualChatID.trim();
        if (!id) return;
        setTgSavingChatID(true);
        setTgMsg('');
        setTgErr('');
        try {
            await SaveTelegramChatID(id);
            setTgManualChatID('');
            setTgMsg(t('settings.telegram.chatIdSaved', {id}));
            refreshTelegramStatus();
        } catch (err) {
            setTgErr(String(err));
        } finally {
            setTgSavingChatID(false);
        }
    }

    async function handleSendTest() {
        setTgTesting(true);
        setTgMsg('');
        setTgErr('');
        try {
            await SendTelegramTest();
            setTgMsg(t('settings.telegram.testSent'));
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

    function handleSelectLanguage(lang: Language) {
        setLanguage(lang);
    }

    return (
        <div>
            <h1 className="text-2xl font-bold tracking-tight text-blue-400">{t('settings.title')}</h1>
            <p className="mt-1 text-sm text-slate-400">{t('settings.subtitle')}</p>

            <div className="mt-6 rounded-md border border-blue-900/40 bg-slate-900/40 p-4 backdrop-blur-md">
                <h2 className="text-sm font-semibold text-slate-300">{t('settings.language.title')}</h2>
                <p className="mt-1 text-xs text-slate-500">{t('settings.language.description')}</p>

                <div className="mt-3 flex gap-3">
                    <button
                        className={`rounded-md border px-4 py-2 text-sm font-medium transition-colors ${
                            language === 'uz'
                                ? 'border-blue-500 bg-blue-900/50 text-blue-300'
                                : 'border-blue-900/40 bg-slate-900/40 text-slate-300 hover:border-blue-700/60'
                        }`}
                        onClick={() => handleSelectLanguage('uz')}
                    >
                        {t('onboarding.lang.uz')}
                    </button>
                    <button
                        className={`rounded-md border px-4 py-2 text-sm font-medium transition-colors ${
                            language === 'en'
                                ? 'border-blue-500 bg-blue-900/50 text-blue-300'
                                : 'border-blue-900/40 bg-slate-900/40 text-slate-300 hover:border-blue-700/60'
                        }`}
                        onClick={() => handleSelectLanguage('en')}
                    >
                        {t('onboarding.lang.en')}
                    </button>
                </div>
            </div>

            <div className="mt-6 rounded-md border border-blue-900/40 bg-slate-900/40 p-4 backdrop-blur-md">
                <h2 className="text-sm font-semibold text-slate-300">{t('settings.scope.title')}</h2>
                <p className="mt-1 text-xs text-slate-500">
                    {t('settings.scope.description')}
                </p>

                <div className="mt-3 flex gap-3">
                    <input
                        className="flex-1 rounded-md border border-blue-900/40 bg-slate-900/40 px-3 py-2 text-sm backdrop-blur-md
                                   placeholder:text-slate-500 focus:border-blue-500 focus:outline-none"
                        type="text"
                        placeholder={t('settings.scope.placeholder')}
                        value={newDomain}
                        onChange={(e) => setNewDomain(e.target.value)}
                        onKeyDown={(e) => e.key === 'Enter' && handleAddDomain()}
                    />
                    <button
                        className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white
                                   hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-40"
                        onClick={handleAddDomain}
                        disabled={adding || !newDomain.trim()}
                    >
                        {t('settings.scope.addButton')}
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
                    {loading && <p className="text-sm text-slate-500">{t('settings.scope.loading')}</p>}
                    {!loading && roots.length === 0 && (
                        <p className="text-sm text-slate-500">
                            {t('settings.scope.empty')}
                        </p>
                    )}
                    {!loading && roots.length > 0 && (
                        <ul className="space-y-1">
                            {roots.map((r) => (
                                <li
                                    key={r}
                                    className="rounded-md border border-blue-900/40 bg-slate-900/40 px-3 py-1.5 font-mono text-sm text-blue-400 backdrop-blur-md"
                                >
                                    {r}
                                </li>
                            ))}
                        </ul>
                    )}
                </div>
            </div>

            <div className="mt-6 rounded-md border border-blue-900/40 bg-slate-900/40 p-4 backdrop-blur-md">
                <h2 className="text-sm font-semibold text-slate-300">{t('settings.monitoring.title')}</h2>
                <p className="mt-1 text-xs text-slate-500">
                    {t('settings.monitoring.description', {
                        tgSuffix: tgEnabled ? t('settings.monitoring.tgSuffix') : '',
                    })}
                </p>

                <div className="mt-3 flex items-center gap-3">
                    <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                            monitoring ? 'bg-blue-900 text-blue-300' : 'bg-slate-800 text-slate-400'
                        }`}
                    >
                        {monitoring ? t('settings.monitoring.running') : t('settings.monitoring.stopped')}
                    </span>
                    <button
                        className={`rounded-md px-4 py-2 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40 ${
                            monitoring ? 'bg-red-700 hover:bg-red-600' : 'bg-blue-600 hover:bg-blue-500'
                        }`}
                        onClick={toggleMonitoring}
                        disabled={monLoading}
                    >
                        {monitoring ? t('settings.monitoring.stopButton') : t('settings.monitoring.startButton')}
                    </button>
                </div>

                {monErr && (
                    <div className="mt-3 rounded-md border border-red-800 bg-red-950/50 px-3 py-2 text-sm text-red-300">
                        {monErr}
                    </div>
                )}
            </div>

            <div className="mt-6 rounded-md border border-blue-900/40 bg-slate-900/40 p-4 backdrop-blur-md">
                <h2 className="text-sm font-semibold text-slate-300">{t('settings.telegram.title')}</h2>
                <div className="mt-3 flex items-center gap-2 text-sm">
                    <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                            tgEnabled ? 'bg-blue-900 text-blue-300' : 'bg-slate-800 text-slate-400'
                        }`}
                    >
                        {tgEnabled ? t('settings.telegram.enabled') : t('settings.telegram.disabled')}
                    </span>
                </div>

                <ol className="mt-3 space-y-3 text-xs text-slate-500">
                    <li>
                        {t('settings.telegram.step1')}
                        <div className="mt-2 flex gap-3">
                            <input
                                className="flex-1 min-w-[200px] rounded-md border border-blue-900/40 bg-slate-900/40 px-3 py-2 text-sm backdrop-blur-md
                                           placeholder:text-slate-500 focus:border-blue-500 focus:outline-none"
                                type="password"
                                placeholder={t('settings.telegram.tokenPlaceholder')}
                                value={tgToken}
                                onChange={(e) => setTgToken(e.target.value)}
                            />
                            <button
                                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white
                                           hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-40"
                                onClick={handleSaveToken}
                                disabled={tgSaving || !tgToken.trim()}
                            >
                                {tgSaving ? t('settings.telegram.saving') : t('settings.telegram.saveToken')}
                            </button>
                        </div>
                    </li>
                    <li>
                        {t('settings.telegram.step2')}
                        <div className="mt-2">
                            <button
                                className="rounded-md border border-blue-900/40 bg-slate-900/40 px-4 py-2 text-sm font-medium
                                           text-blue-400 backdrop-blur-md hover:border-blue-600 disabled:cursor-not-allowed disabled:opacity-40"
                                onClick={handleDetectChatID}
                                disabled={tgDetecting}
                            >
                                {tgDetecting ? t('settings.telegram.detecting') : t('settings.telegram.detectButton')}
                            </button>
                        </div>
                        <div className="mt-3">
                            {t('settings.telegram.manualHelp')}
                            <div className="mt-2 flex gap-3">
                                <input
                                    className="flex-1 min-w-[160px] rounded-md border border-blue-900/40 bg-slate-900/40 px-3 py-2 text-sm backdrop-blur-md
                                               placeholder:text-slate-500 focus:border-blue-500 focus:outline-none"
                                    type="text"
                                    placeholder={t('settings.telegram.chatIdPlaceholder')}
                                    value={tgManualChatID}
                                    onChange={(e) => setTgManualChatID(e.target.value)}
                                />
                                <button
                                    className="rounded-md border border-blue-900/40 bg-slate-900/40 px-4 py-2 text-sm font-medium
                                               text-blue-400 backdrop-blur-md hover:border-blue-600 disabled:cursor-not-allowed disabled:opacity-40"
                                    onClick={handleSaveChatIDManually}
                                    disabled={tgSavingChatID || !tgManualChatID.trim()}
                                >
                                    {tgSavingChatID ? t('settings.telegram.saving') : t('settings.telegram.saveChatId')}
                                </button>
                            </div>
                        </div>
                    </li>
                    <li>
                        {t('settings.telegram.step3')}
                        <div className="mt-2">
                            <button
                                className="rounded-md border border-blue-900/40 bg-slate-900/40 px-4 py-2 text-sm font-medium
                                           text-blue-400 backdrop-blur-md hover:border-blue-600 disabled:cursor-not-allowed disabled:opacity-40"
                                onClick={handleSendTest}
                                disabled={tgTesting}
                            >
                                {tgTesting ? t('settings.telegram.testing') : t('settings.telegram.testButton')}
                            </button>
                        </div>
                    </li>
                </ol>

                {tgMsg && (
                    <div className="mt-3 rounded-md border border-blue-800 bg-blue-950/50 px-3 py-2 text-sm text-blue-300">
                        {tgMsg}
                    </div>
                )}
                {tgErr && (
                    <div className="mt-3 rounded-md border border-red-800 bg-red-950/50 px-3 py-2 text-sm text-red-300">
                        {tgErr}
                    </div>
                )}

                <p className="mt-3 text-xs text-slate-500">
                    {t('settings.telegram.envHint')}
                </p>
            </div>

            <div className="mt-6 rounded-md border border-blue-900/40 bg-slate-900/40 p-4 backdrop-blur-md">
                <h2 className="text-sm font-semibold text-slate-300">{t('settings.modules.title')}</h2>
                <p className="mt-1 text-xs text-slate-500">
                    {t('settings.modules.description')}
                </p>

                {modulesErr && (
                    <div className="mt-3 rounded-md border border-red-800 bg-red-950/50 px-3 py-2 text-sm text-red-300">
                        {modulesErr}
                    </div>
                )}

                <div className="mt-4">
                    {modulesLoading && <p className="text-sm text-slate-500">{t('settings.modules.loading')}</p>}
                    {!modulesLoading && modules.length === 0 && (
                        <p className="text-sm text-slate-500">{t('settings.modules.empty')}</p>
                    )}
                    {!modulesLoading && modules.length > 0 && (
                        <div className="overflow-hidden rounded-md border border-blue-900/40">
                            <table className="w-full text-left text-sm">
                                <thead className="bg-slate-900/60 text-slate-400">
                                    <tr>
                                        <th className="px-3 py-2 font-medium">{t('settings.modules.colName')}</th>
                                        <th className="px-3 py-2 font-medium">{t('settings.modules.colCategory')}</th>
                                        <th className="px-3 py-2 font-medium">{t('settings.modules.colTrigger')}</th>
                                        <th className="px-3 py-2 font-medium">{t('settings.modules.colEnabled')}</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {modules.map((m) => (
                                        <tr key={m.name} className="border-t border-blue-900/30">
                                            <td className="px-3 py-2 font-mono text-blue-400">{m.name}</td>
                                            <td className="px-3 py-2 text-slate-400">{m.category}</td>
                                            <td className="px-3 py-2 text-slate-400">{m.trigger}</td>
                                            <td className="px-3 py-2">
                                                <button
                                                    className={`rounded-full px-3 py-1 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${
                                                        m.enabled
                                                            ? 'bg-blue-900 text-blue-300 hover:bg-blue-800'
                                                            : 'bg-slate-800 text-slate-400 hover:bg-slate-700'
                                                    }`}
                                                    onClick={() => handleToggleModule(m)}
                                                    disabled={togglingModule === m.name}
                                                >
                                                    {m.enabled ? t('settings.telegram.enabled') : t('settings.telegram.disabled')}
                                                </button>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}

export default SettingsView;
