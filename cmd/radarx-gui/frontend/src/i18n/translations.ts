import type {Language} from './storage';

// Flat key -> string dictionaries. Keys are grouped by view via naming
// convention (e.g. `scan.*`, `targets.*`) but stored flat for simplicity.
export type TranslationKey = keyof typeof en;

const en = {
    // App shell / sidebar
    'app.name': 'RadarX',
    'app.tagline': 'Attack Surface Monitor',
    'nav.targets': 'Targets',
    'nav.scan': 'Scan',
    'nav.results': 'Results',
    'nav.diff': 'Diff',
    'nav.settings': 'Settings',

    // Onboarding
    'onboarding.tagline': 'Attack Surface Monitor',
    'onboarding.continue': 'Continue / Davom etish',
    'onboarding.chooseLanguage': 'Choose your language',
    'onboarding.chooseLanguageSubtitle': 'You can change this later in Settings.',
    'onboarding.lang.uz': "O'zbek",
    'onboarding.lang.en': 'English',
    'onboarding.continueButton': 'Continue',

    // ScanView
    'scan.title': 'RadarX',
    'scan.subtitle': 'Attack surface scanner — enter a target domain and scan.',
    'scan.placeholder': 'example.com',
    'scan.scanButton': 'Scan',
    'scan.stopButton': 'Stop',
    'scan.status': 'Status:',
    'scan.assetsFound_one': 'asset found',
    'scan.assetsFound_other': 'assets found',
    'scan.noResults': 'No results yet.',
    'scan.colKind': 'Kind',
    'scan.colKey': 'Key',
    'scan.colDetail': 'Detail',
    'scan.status.idle': 'idle',
    'scan.status.running': 'running',
    'scan.status.done': 'done',
    'scan.status.stopped': 'stopped',
    'scan.status.error': 'error',

    // TargetsView
    'targets.title': 'Targets',
    'targets.subtitle': 'Domains RadarX is authorized to monitor.',
    'targets.addTarget': '+ Add Target',
    'targets.rootPlaceholder': 'root domain, e.g. example.com',
    'targets.labelPlaceholder': 'label (optional)',
    'targets.intervalPlaceholder': 'interval (min)',
    'targets.addButton': 'Add Target',
    'targets.pendingAuth': "Bu domen hali sizning ruxsat etilgan scope'ingizda yo'q:",
    'targets.pendingAuthQuestion':
        "Uni scan qilishga ruxsatingiz borligini tasdiqlaysizmi (o'zingizning infratuzilmangiz yoki ruxsat berilgan bug-bounty scope)?",
    'targets.authorizeYes': 'Ha, ruxsat beraman',
    'targets.authorizeCancel': 'Bekor qilish',
    'targets.colRoot': 'Root',
    'targets.colLabel': 'Label',
    'targets.colInterval': 'Interval',
    'targets.colLastScan': 'Last Scan',
    'targets.noTargets': 'No targets yet. Add one above.',
    'targets.loading': 'Loading...',
    'targets.never': 'never',

    // ResultsView
    'results.title': 'Results',
    'results.subtitle': 'Latest snapshot per target: subdomains, ports, TLS info.',
    'results.noTargets': 'No targets',
    'results.filterPlaceholder': 'filter / search...',
    'results.exportButton': 'Export report',
    'results.exporting': 'Exporting...',
    'results.savedTo': 'Saved to',
    'results.loading': 'Loading...',
    'results.taken': 'Taken:',
    'results.assetsShown_one': 'of {total} asset shown',
    'results.assetsShown_other': 'of {total} assets shown',
    'results.colKind': 'Kind',
    'results.colKey': 'Key',
    'results.colDetail': 'Detail',
    'results.noMatching': 'No matching assets.',

    // DiffView
    'diff.title': 'Diff / Changes',
    'diff.subtitle': 'What changed between the two most recent scans of a target.',
    'diff.noTargets': 'No targets',
    'diff.loading': 'Loading...',
    'diff.totalChanges_one': 'total change',
    'diff.totalChanges_other': 'total changes',
    'diff.colChange': 'Change',
    'diff.colKind': 'Kind',
    'diff.colKey': 'Key',
    'diff.colFields': 'Fields',
    'diff.noChanges': 'No changes between the last two scans.',
    'diff.new': '+ NEW',
    'diff.removed': '- REMOVED',
    'diff.modified': '~ CHANGED',

    // SettingsView
    'settings.title': 'Settings',
    'settings.subtitle': 'Scope and notification configuration.',
    'settings.scope.title': 'Authorized Scope',
    'settings.scope.description':
        'Root domains RadarX is allowed to scan. A scan is refused before any network request is made unless its target is covered here.',
    'settings.scope.placeholder': 'add domain to scope, e.g. example.com',
    'settings.scope.addButton': 'Add to scope',
    'settings.scope.loading': 'Loading...',
    'settings.scope.empty': "Hali hech narsa yo'q — scope bo'sh. Yuqoridagi input orqali domen qo'shing.",

    'settings.monitoring.title': 'Continuous Monitoring',
    'settings.monitoring.description':
        'Periodically rescans every enabled target on its interval, diffs against the last snapshot, and notifies you (desktop{tgSuffix}) when something changes. Closing the window while this is on keeps it running in the background instead of quitting.',
    'settings.monitoring.tgSuffix': ' + Telegram',
    'settings.monitoring.running': 'running',
    'settings.monitoring.stopped': 'stopped',
    'settings.monitoring.stopButton': 'Stop monitoring',
    'settings.monitoring.startButton': 'Start monitoring',

    'settings.telegram.title': 'Telegram Notifications',
    'settings.telegram.enabled': 'enabled',
    'settings.telegram.disabled': 'disabled',
    'settings.telegram.step1':
        "1. @BotFather orqali bot yarating, tokenni shu yerga kiriting va saqlang (token faqat sizning kompyuteringizda, ~/.radarx/config.json da saqlanadi — hech qayerga yuborilmaydi).",
    'settings.telegram.tokenPlaceholder': 'bot token',
    'settings.telegram.saveToken': 'Save token',
    'settings.telegram.saving': 'Saving...',
    'settings.telegram.tokenSaved':
        'Token saqlandi. Endi botga Telegramda bitta xabar yozing (masalan /start), keyin "Chat ID aniqlash" tugmasini bosing.',
    'settings.telegram.step2':
        "2. Telegramda o'z botingizga bitta xabar yozing (masalan /start), keyin bosing — faqat shu botga yozgan odam xabarnoma oladi:",
    'settings.telegram.detectButton': 'Chat ID aniqlash',
    'settings.telegram.detecting': 'Detecting...',
    'settings.telegram.chatIdDetected': 'Chat ID aniqlandi va saqlandi: {chatID}. Endi "Test xabar yuborish" bilan tekshiring.',
    'settings.telegram.manualHelp':
        "Ishlamasa (masalan Telegram eski xabarni topolmasa) — chat ID'ni qo'lda kiriting. Uni bilish uchun Telegram'da @userinfobot ga yozing, u darhol raqamli ID'ingizni qaytaradi:",
    'settings.telegram.chatIdPlaceholder': 'chat id, masalan 123456789',
    'settings.telegram.saveChatId': 'Chat ID saqlash',
    'settings.telegram.chatIdSaved': 'Chat ID saqlandi: {id}. Endi "Test xabar yuborish" bilan tekshiring.',
    'settings.telegram.step3': '3. Tekshiring:',
    'settings.telegram.testButton': 'Test xabar yuborish',
    'settings.telegram.testing': 'Sending...',
    'settings.telegram.testSent': 'Test xabar yuborildi — Telegramni tekshiring.',
    'settings.telegram.envHint':
        "Muqobil: RADARX_TG_TOKEN / RADARX_TG_CHAT_ID environment variable'lar orqali ham sozlash mumkin (ular saqlangan konfiguratsiyadan ustuvor).",

    'settings.language.title': 'Language / Til',
    'settings.language.description': 'Interface language for RadarX.',
} as const;

const uz: Record<TranslationKey, string> = {
    // App shell / sidebar
    'app.name': 'RadarX',
    'app.tagline': 'Hujum yuzasi monitoringi',
    'nav.targets': 'Nishonlar',
    'nav.scan': 'Skanerlash',
    'nav.results': 'Natijalar',
    'nav.diff': 'Farqlar',
    'nav.settings': 'Sozlamalar',

    // Onboarding
    'onboarding.tagline': 'Hujum yuzasi monitoringi',
    'onboarding.continue': 'Continue / Davom etish',
    'onboarding.chooseLanguage': 'Tilni tanlang',
    'onboarding.chooseLanguageSubtitle': "Buni keyinroq Sozlamalar bo'limida o'zgartirishingiz mumkin.",
    'onboarding.lang.uz': "O'zbek",
    'onboarding.lang.en': 'English',
    'onboarding.continueButton': 'Davom etish',

    // ScanView
    'scan.title': 'RadarX',
    'scan.subtitle': "Hujum yuzasi skaneri — nishon domenni kiriting va skanerlashni boshlang.",
    'scan.placeholder': 'example.com',
    'scan.scanButton': 'Skanerlash',
    'scan.stopButton': "To'xtatish",
    'scan.status': 'Holat:',
    'scan.assetsFound_one': 'asset topildi',
    'scan.assetsFound_other': 'asset topildi',
    'scan.noResults': "Hozircha natija yo'q.",
    'scan.colKind': 'Turi',
    'scan.colKey': 'Kalit',
    'scan.colDetail': "Tafsilot",
    'scan.status.idle': 'kutilmoqda',
    'scan.status.running': 'ishlamoqda',
    'scan.status.done': 'tugadi',
    'scan.status.stopped': "to'xtatildi",
    'scan.status.error': 'xato',

    // TargetsView
    'targets.title': 'Nishonlar',
    'targets.subtitle': 'RadarX monitoring qilishga ruxsat berilgan domenlar.',
    'targets.addTarget': '+ Nishon qo\'shish',
    'targets.rootPlaceholder': 'root domen, masalan example.com',
    'targets.labelPlaceholder': 'yorliq (ixtiyoriy)',
    'targets.intervalPlaceholder': 'interval (daqiqa)',
    'targets.addButton': "Nishon qo'shish",
    'targets.pendingAuth': "Bu domen hali sizning ruxsat etilgan scope'ingizda yo'q:",
    'targets.pendingAuthQuestion':
        "Uni scan qilishga ruxsatingiz borligini tasdiqlaysizmi (o'zingizning infratuzilmangiz yoki ruxsat berilgan bug-bounty scope)?",
    'targets.authorizeYes': 'Ha, ruxsat beraman',
    'targets.authorizeCancel': 'Bekor qilish',
    'targets.colRoot': 'Root',
    'targets.colLabel': 'Yorliq',
    'targets.colInterval': 'Interval',
    'targets.colLastScan': 'Oxirgi skan',
    'targets.noTargets': "Hali nishon yo'q. Yuqorida biror narsa qo'shing.",
    'targets.loading': 'Yuklanmoqda...',
    'targets.never': 'hech qachon',

    // ResultsView
    'results.title': 'Natijalar',
    'results.subtitle': "Har bir nishon uchun so'nggi snapshot: subdomenlar, portlar, TLS ma'lumoti.",
    'results.noTargets': "Nishonlar yo'q",
    'results.filterPlaceholder': 'filtr / qidiruv...',
    'results.exportButton': 'Hisobotni eksport qilish',
    'results.exporting': 'Eksport qilinmoqda...',
    'results.savedTo': 'Saqlandi:',
    'results.loading': 'Yuklanmoqda...',
    'results.taken': 'Olingan vaqt:',
    'results.assetsShown_one': "{total} tadan asset ko'rsatildi",
    'results.assetsShown_other': "{total} tadan asset ko'rsatildi",
    'results.colKind': 'Turi',
    'results.colKey': 'Kalit',
    'results.colDetail': 'Tafsilot',
    'results.noMatching': "Mos asset topilmadi.",

    // DiffView
    'diff.title': "Farqlar / O'zgarishlar",
    'diff.subtitle': "Nishonning so'nggi ikki skani orasida nima o'zgargani.",
    'diff.noTargets': "Nishonlar yo'q",
    'diff.loading': 'Yuklanmoqda...',
    'diff.totalChanges_one': "jami o'zgarish",
    'diff.totalChanges_other': "jami o'zgarish",
    'diff.colChange': "O'zgarish",
    'diff.colKind': 'Turi',
    'diff.colKey': 'Kalit',
    'diff.colFields': 'Maydonlar',
    'diff.noChanges': "So'nggi ikki skan orasida o'zgarish yo'q.",
    'diff.new': '+ YANGI',
    'diff.removed': "- OLIB TASHLANDI",
    'diff.modified': "~ O'ZGARDI",

    // SettingsView
    'settings.title': 'Sozlamalar',
    'settings.subtitle': 'Scope va bildirishnoma sozlamalari.',
    'settings.scope.title': 'Ruxsat etilgan Scope',
    'settings.scope.description':
        "RadarX skanerlashga ruxsat berilgan root domenlar. Agar nishon shu yerda bo'lmasa, hech qanday tarmoq so'rovi yuborilmasdan turib skan rad etiladi.",
    'settings.scope.placeholder': "scope'ga domen qo'shish, masalan example.com",
    'settings.scope.addButton': "Scope'ga qo'shish",
    'settings.scope.loading': 'Yuklanmoqda...',
    'settings.scope.empty': "Hali hech narsa yo'q — scope bo'sh. Yuqoridagi input orqali domen qo'shing.",

    'settings.monitoring.title': 'Uzluksiz monitoring',
    'settings.monitoring.description':
        "Har bir yoqilgan nishonni o'z intervalida qayta skanerlaydi, so'nggi snapshot bilan solishtiradi va o'zgarish bo'lsa sizga xabar beradi (desktop{tgSuffix}). Bu yoqilgan holatda oynani yopish ilovani to'xtatmaydi, u fonda ishlashda davom etadi.",
    'settings.monitoring.tgSuffix': ' + Telegram',
    'settings.monitoring.running': 'ishlamoqda',
    'settings.monitoring.stopped': "to'xtatilgan",
    'settings.monitoring.stopButton': "Monitoringni to'xtatish",
    'settings.monitoring.startButton': 'Monitoringni boshlash',

    'settings.telegram.title': 'Telegram bildirishnomalari',
    'settings.telegram.enabled': 'yoqilgan',
    'settings.telegram.disabled': "o'chirilgan",
    'settings.telegram.step1':
        "1. @BotFather orqali bot yarating, tokenni shu yerga kiriting va saqlang (token faqat sizning kompyuteringizda, ~/.radarx/config.json da saqlanadi — hech qayerga yuborilmaydi).",
    'settings.telegram.tokenPlaceholder': 'bot token',
    'settings.telegram.saveToken': 'Tokenni saqlash',
    'settings.telegram.saving': 'Saqlanmoqda...',
    'settings.telegram.tokenSaved':
        'Token saqlandi. Endi botga Telegramda bitta xabar yozing (masalan /start), keyin "Chat ID aniqlash" tugmasini bosing.',
    'settings.telegram.step2':
        "2. Telegramda o'z botingizga bitta xabar yozing (masalan /start), keyin bosing — faqat shu botga yozgan odam xabarnoma oladi:",
    'settings.telegram.detectButton': 'Chat ID aniqlash',
    'settings.telegram.detecting': 'Aniqlanmoqda...',
    'settings.telegram.chatIdDetected': 'Chat ID aniqlandi va saqlandi: {chatID}. Endi "Test xabar yuborish" bilan tekshiring.',
    'settings.telegram.manualHelp':
        "Ishlamasa (masalan Telegram eski xabarni topolmasa) — chat ID'ni qo'lda kiriting. Uni bilish uchun Telegram'da @userinfobot ga yozing, u darhol raqamli ID'ingizni qaytaradi:",
    'settings.telegram.chatIdPlaceholder': 'chat id, masalan 123456789',
    'settings.telegram.saveChatId': 'Chat ID saqlash',
    'settings.telegram.chatIdSaved': 'Chat ID saqlandi: {id}. Endi "Test xabar yuborish" bilan tekshiring.',
    'settings.telegram.step3': '3. Tekshiring:',
    'settings.telegram.testButton': 'Test xabar yuborish',
    'settings.telegram.testing': 'Yuborilmoqda...',
    'settings.telegram.testSent': 'Test xabar yuborildi — Telegramni tekshiring.',
    'settings.telegram.envHint':
        "Muqobil: RADARX_TG_TOKEN / RADARX_TG_CHAT_ID environment variable'lar orqali ham sozlash mumkin (ular saqlangan konfiguratsiyadan ustuvor).",

    'settings.language.title': 'Til / Language',
    'settings.language.description': 'RadarX interfeys tili.',
};

export const dictionaries: Record<Language, Record<TranslationKey, string>> = {
    en,
    uz,
};
