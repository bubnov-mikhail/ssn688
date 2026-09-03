package ui

import (
	"github.com/bubnov-mikhail/ssn688/internal/i18n"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// libraryWeaponCatalog — in-game ordnance handbook (torpedoes, missiles, ASW rockets, CM).
var libraryWeaponCatalog = []libraryEntry{
	{
		ID: "wpn_mk48", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("Mk 48 ADCAP — heavy torpedo", "Mk 48 ADCAP — тяжёлая торпеда"),
		ImageFile: "wpn_mk48.jpg",
		Credit:    i18n.T("Photo: U.S. Navy / public domain (Wikimedia Commons)", "Фото: U.S. Navy / общественное достояние (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Primary U.S. submarine heavyweight: wire-guided to enable, then active/passive homing. Player tubes 1–2 default; magazine ~26 rounds in sim.", "Основная американская тяжёлая торпеда: проводное наведение до включения ГСН, затем активный/пассивный самонаводящийся режим. ТА 1–2 по умолчанию; БК ~26 в симе."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Diameter 21 in (533 mm)  |  Length ~5.8 m", "Калибр 533 мм  |  Длина ~5,8 м"),
			i18n.T("Speed presets LOW/HIGH — up to ~55 kn cruise in sim", "Скорости LOW/HIGH — до ~55 уз крейсера в симе"),
			i18n.T("Seeker acquisition ~1,600 yd published; wire cut → search", "Захват ГСН ~1 600 ярд (публикации); обрыв провода → поиск"),
			i18n.T("Terminal modes: circle, snake, straight", "Конечные режимы: круг, змейка, прямой"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Wire guide from WEPS; enable seeker inside acquisition envelope", "Проводное наведение с WEPS; включение ГСН в зоне захвата"),
			i18n.T("Exercise round available — no warhead detonation", "Учебная версия — без боевого подрыва"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Enemy soft-kill (ADC/jitter) can seduce seeker if fired early", "Вражеский soft-kill (ADC/jitter) может увести ГСН при раннем пуске"),
			i18n.T("Hostile heavy fish are slower but numerous — prioritize TMA", "Вражеские тяжёлые торпеды медленнее, но многочисленны — приоритет TMA"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Classify before shoot; Mk48 is magazine-limited — make first shot count.", "Классифицируйте до выстрела; Mk48 ограничен БК — первый выстрел решает."),
			i18n.T("Enable seeker late for sharp geometry; wire until inside envelope.", "Включайте ГСН поздно для острой геометрии; провод до зоны захвата."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Incoming Mk48-class: CM on CPA, depth change, speed burst, knuckle.", "Входящая Mk48-класса: ПМ по CPA, смена глубины, рывок хода, «колено»."),
		},
	},
	{
		ID: "wpn_harpoon", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("UGM-84 Sub-Harpoon — ASCM", "UGM-84 Sub-Harpoon — ПКР"),
		ImageFile: "wpn_harpoon.jpg",
		Credit:    i18n.T("Photo: U.S. Navy / public domain (Wikimedia Commons)", "Фото: U.S. Navy / общественное достояние (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Submarine-launched anti-ship cruise missile: capsule egress, then sea-skimming radar homing. Player tubes 3–4; allied 688 carries 8 rounds.", "Подводная противокорабельная ракета: выход капсулы, затем полёт на малой высоте с РЛ ГСН. ТА 3–4 игрока; союзный 688 — 8 ракет."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Max range ~75 nm (sim)  |  Cruise ~Mach 0.85", "Дальность до ~75 м.миль (сим)  |  Крейсер ~M 0,85"),
			i18n.T("SRCH presets 1–8 nm radar enable; DSTR self-destruct ranges", "SRCH: включение РЛ 1–8 м.миль; DSTR — дистанции самоликвидации"),
			i18n.T("WIDE/NARROW acquisition beam on WEPS", "Широкий/узкий сектор захвата на WEPS"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Surface targets only; periscope-depth launch", "Только надводные цели; пуск с глубины перископа"),
			i18n.T("Allied AI fires at classified surface contacts in DEFCON prosecution", "Союзный ИИ стреляет по классифицированным надводным целям при DEFCON-преследовании"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Hostile SAM/CIWS can kill in cruise phase (point defense in sim)", "Вражеские ЗРК/ЗАК могут сбить на крейсерском участке (ПВО в симе)"),
			i18n.T("Underwater egress is acoustically loud — raises enemy DEFCON", "Подводный выход громкий акустически — повышает DEFCON врага"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Deplete SAM/CIWS first or fire from unexpected bearing; snap shots at periscope depth beat long wire-guide setups.", "Истощите ЗРК/ЗАК или стреляйте с неожиданного пеленга; быстрый пуск с перископа лучше долгой подготовки."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Turn into seeker cone, deploy Nixie if available, hard maneuvers at terminal range.", "Развернитесь в сектор ГСН, выбросьте Nixie при наличии, резкие манёвры на конечном участке."),
		},
	},
	{
		ID: "wpn_klub", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("3M-54 Klub-S — ASCM (Kilo)", "3М-54 «Клуб-С» — ПКР (Кило)"),
		ImageFile: "wpn_klub.jpg",
		Credit:    i18n.T("Photo: Vitaly V. Kuzmin / CC BY-SA 4.0 (Wikimedia Commons)", "Фото: Vitaly V. Kuzmin / CC BY-SA 4.0 (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Export-family Klub-S fired from 533 mm torpedo tubes; subsonic sea-skimmer with terminal active radar. Kilo-class default mag ~4 in sim.", "Экспортный «Клуб-С» из 533-мм ТА; дозвуковой полёт на малой высоте с активной РЛ на конце. БК «Кило» ~4 в симе."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Launch depth ≤~40 m (~130 ft) per open sources", "Глубина пуска ≤~40 м (~130 фут) по открытым данным"),
			i18n.T("Range up to ~120 nm class; sim cap ~65 nm engagement band", "Дальность класса до ~120 м.миль; в симе полоса ~65 м.миль"),
			i18n.T("Cruise ~520 kn in sim (subsonic)", "Крейсер ~520 уз в симе (дозвук)"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("AI fires at classified allied/neutral surface combatants beyond torpedo range", "ИИ стреляет по классифицированным надводным союзникам/нейтралам вне дальности торпед"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Similar defeat mechanisms to Harpoon — SAM/CIWS, maneuver", "Схожие способы поражения с «Гарпуном» — ЗРК/ЗАК, манёвр"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Sink Kilo before he reaches periscope snapshot range; force snorkeling if diesel.", "Уничтожьте «Кило» до дистанции перископного пуска; вынуждайте РДП у дизеля."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Treat like inbound Harpoon: PD layers, chaff/Nixie doctrine, beam maneuvers.", "Как входящий «Гарпун»: эшелон ПВО, Nixie, манёвры поперёк пеленга."),
		},
	},
	{
		ID: "wpn_oniks", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("3M-55 Oniks — supersonic ASCM", "3М-55 «Оникс» — сверхзвуковая ПКР"),
		ImageFile: "wpn_oniks.jpg",
		Credit:    i18n.T("Photo: Allocer / CC BY 3.0 (Wikimedia Commons)", "Фото: Allocer / CC BY 3.0 (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Supersonic anti-ship missile from Yasen UKSK; shorter range but very fast terminal phase in sim.", "Сверхзвуковая ПКР с УКСК «Ясеня»; меньшая дальность, но очень быстрый конечный участок в симе."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Speed ~Mach 2 class (~1,200 kn in sim cruise)", "Скорость класса ~M 2 (~1 200 уз крейсер в симе)"),
			i18n.T("Range ~40 nm sim cap", "Дальность ~40 м.миль в симе"),
			i18n.T("VLS / torpedo-tube compatible on Pr.885", "Совместим с УВП / ТА на пр.885"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Yasen AI alternates Oniks and Kalibr salvos at surface targets", "ИИ «Ясеня» чередует «Оникс» и «Калибр» по надводным целям"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Harder for CIWS timing — prioritize SAM layer", "Сложнее для ЗАК по времени — приоритет ЗРК"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Engage Yasen early; do not rely on last-second CIWS alone.", "Атакуйте «Ясень» рано; не полагайтесь только на ЗАК в последнюю секунду."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Pre-turn before seeker lock; maximize PD magazine before entering UKSK range.", "Развернитесь до захвата ГСН; накопите БК ПВО до входа в зону УКСК."),
		},
	},
	{
		ID: "wpn_kalibr", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("3M-54 Kalibr-PL — ASCM", "3М-54 «Калибр-ПЛ» — ПКР"),
		ImageFile: "wpn_kalibr.jpg",
		Credit:    i18n.T("Photo: Vitaly V. Kuzmin / CC BY-SA 4.0 (Wikimedia Commons)", "Фото: Vitaly V. Kuzmin / CC BY-SA 4.0 (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Subsonic anti-ship variant of Kalibr family from UKSK; longer reach than Oniks, Harpoon-like profile in sim.", "Дозвуковой противокорабельный вариант «Калибра» с УКСК; дальше «Оникса», профиль как у «Гарпуна» в симе."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Range ~70 nm sim cap", "Дальность ~70 м.миль в симе"),
			i18n.T("Cruise ~500 kn subsonic", "Крейсер ~500 уз дозвук"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Yasen magazine split ~8 Oniks / ~8 Kalibr gameplay default", "БК «Ясеня» по умолчанию ~8 «Оникс» / ~8 «Калибр»"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Defeat like Harpoon/Klub — radar homing cruise phase", "Поражение как «Гарпун»/«Клуб» — крейсер с РЛ ГСН"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Suppress Yasen before VLS salvo; allied Harpoon at similar ranges if permitted.", "Подавите «Ясень» до залпа УВП; союзный «Гарпун» на схожих дистанциях при разрешении."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("SAM engagement window longer than Oniks — use layered PD.", "Окно ЗРК длиннее, чем у «Оникса» — эшелонируйте ПВО."),
		},
	},
	{
		ID: "wpn_53_65", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("53-65 / SET-65 — heavy Soviet fish", "53-65 / СЭТ-65 — тяжёлая советская торпеда"),
		ImageFile: "wpn_53_65.jpg",
		Credit:    i18n.T("Photo: U.S. Navy / public domain (Wikimedia Commons)", "Фото: U.S. Navy / общественное достояние (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Primary heavy torpedo on Kilo, Foxtrot, Victor III: wire then active/passive homing; slower cruise than Mk48 on some classes.", "Основная тяжёлая торпеда «Кило», «Фокстрота», «Виктор-III»: провод, затем актив/пассив; на части классов медленнее Mk48."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("533 mm  |  Cruise ~40–55 kn by launcher class in sim", "533 мм  |  Крейсер ~40–55 уз по классу пуска в симе"),
			i18n.T("Magazines 10–24 depending on boat", "БК 10–24 в зависимости от лодки"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Enemy sub AI standoff band ~1.2–4 kyd", "Дистанция пуска вражеской ПЛ ~1,2–4 кярд"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Player ADC/jitter effective vs gullible hostile seeker tuning", "ADC/jitter игрока эффективны против «доверчивой» настройки вражеской ГСН"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Kill shooter before tube open; crush Foxtrot/Kilo before close range.", "Уничтожьте стрелка до открытия ТА; «Фокстрот»/«Кило» на близкой дистанции."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("CPA evade + CM; knuckle after wire cut.", "Уклонение по CPA + ПМ; «колено» после обрыва провода."),
		},
	},
	{
		ID: "wpn_umgt1", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("UMGT-1 / SET-40 — lightweight ASW torpedo", "УМГТ-1 / СЭТ-40 — лёгкая ПЛ торпеда"),
		ImageFile: "wpn_umgt1.jpg",
		Credit:    i18n.T("Photo: U.S. DoD / public domain (Wikimedia Commons)", "Фото: U.S. DoD / общественное достояние (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Ship-tube and Rastrub-splash lightweight fish; fast search after brief runout.", "Лёгкая торпеда из корабельных ТА и после всплеска «Раструба»; быстрый поиск после короткого разбега."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("SET-40 on Grisha  |  Mk 46 stand-in on Spruance in sim", "СЭТ-40 на «Грише»  |  Mk 46 на «Спруэнсе» в симе"),
			i18n.T("Shallow-depth threat — periscope to ~400 ft envelope", "Угроза на малых глубинах — перископ до ~400 фут"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Splash from Rastrub/ASROC or direct ship tubes", "Всплеск от «Раструба»/ASROC или прямой пуск с корабля"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Less lethal than Mk48 but forces depth/speed changes", "Менее смертельна, чем Mk48, но вынуждает менять глубину/ход"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Stay below layer; avoid hovering at periscope depth in ASW rocket range.", "Держитесь под слоем; не зависайте на перископе в зоне ПЛУР."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Go deep past Rastrub bracket; CM on CPA.", "Уходите глубже зоны «Раструба»; ПМ по CPA."),
		},
	},
	{
		ID: "wpn_rastrub", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("URPK-5 Rastrub / Metel — ASW rocket", "УРПК-5 «Раструб» / «Метель» — ПЛ ракета"),
		ImageFile: "wpn_rastrub.jpg",
		Credit:    i18n.T("Photo: U.S. DoD / public domain (Wikimedia Commons)", "Фото: U.S. DoD / общественное достояние (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Rocket-assisted torpedo (ASROC-class): flies to datum, splashes lightweight fish. Otvet/ASROC use same mechanics on Gorshkov/Spruance.", "Ракето-торпеда (класс ASROC): полёт к датуму, всплеск лёгкой торпеды. «Ответ»/ASROC — те же механики на «Горшкове»/«Спруэнсе»."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Range ~1–8 kyd bracket in sim", "Дальность ~1–8 кярд в симе"),
			i18n.T("Magazine ~8–12 cells by class", "БК ~8–12 ячеек по классу"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Fired when enemy has classified shallow submarine contact", "Пуск при классифицированном мелком контакте ПЛ"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Splash point visible on DEBUG; fish follows", "Точка всплеска видна в DEBUG; далее торпеда"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Open range from escorts; do not present steady shallow track.", "Открывайте дистанцию от эскорта; не давайте устойчивый мелкий след."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Depth below 400 ft before splash lands; knuckle after splash.", "Глубже 400 фут до всплеска; «колено» после всплеска."),
		},
	},
	{
		ID: "wpn_rbu", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("RBU-6000 — ASW rocket barrage", "РБУ-6000 — ПЛ реактивная бомбомёт"),
		ImageFile: "wpn_rbu.jpg",
		Credit:    i18n.T("Photo: U.S. DoD / public domain (Wikimedia Commons)", "Фото: U.S. DoD / общественное достояние (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Grisha-class short-range ASW rockets: bracket shallow submarine with underwater blasts.", "Корвет «Гриша»: ближняя ПЛ реактивная бомбомётная завеса по мелкой ПЛ."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Range ~400–2,200 yd  |  Max target depth ~120 ft", "Дальность ~400–2 200 ярд  |  Макс. глубина цели ~120 фут"),
			i18n.T("Magazine ~10 salvos", "БК ~10 залпов"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Preferred over tubes when target at periscope depth", "Предпочтительнее ТА при цели на глубине перископа"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Area effect — no homing, but fast salvo", "Площадное поражение — без наведения, но быстрый залп"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Sink Grisha outside RBU envelope before closing.", "Уничтожьте «Гришу» вне зоны РБУ до сближения."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Go deep immediately; RBU cannot reach far below periscope depth.", "Срочно на глубину; РБУ не достаёт далеко ниже перископа."),
		},
	},
	{
		ID: "wpn_sam", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("Point-defense SAM — sea skimmer kill", "ЗРК ближней ПВО — поражение низколетящих"),
		ImageFile: "wpn_sam.jpg",
		Credit:    i18n.T("Photo: U.S. Navy / public domain (Wikimedia Commons)", "Фото: U.S. Navy / общественное достояние (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Simulated Kinzhal/Osa/Sea Sparrow/Redut layer vs inbound Harpoon/Klub/Kalibr in cruise phase.", "Моделируемый эшелон «Кинжал»/«Оса»/Sea Sparrow/«Редут» против входящих «Гарпун»/«Клуб»/«Калибр» на крейсере."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Engagement ~2–8 kyd vs cruise missiles in sim", "Поражение ~2–8 кярд против крылатых в симе"),
			i18n.T("Magazine 4–32 by ship class", "БК 4–32 по классу корабля"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Automatic when hostile Harpoon/ASCM in radar search", "Автоматически при вражеском «Гарпуне»/ПКР в РЛ поиске"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Depletes before CIWS inner layer", "Истощается до ближнего рубежа ЗАК"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Saturation: multiple missiles or split bearings exhaust SAM.", "Насыщение: несколько ракет или разнесённые пеленги истощают ЗРК."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Not player-evadable — plan magazine drain before shooting Harpoon.", "Игрок не уклоняется — планируйте расход БК перед пуском «Гарпуна»."),
		},
	},
	{
		ID: "wpn_ciws", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("CIWS — last-ditch missile kill", "ЗАК — последний рубеж по ракетам"),
		ImageFile: "wpn_ciws.jpg",
		Credit:    i18n.T("Photo: U.S. Navy / public domain (Wikimedia Commons)", "Фото: U.S. Navy / общественное достояние (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("AK-630 / Phalanx burst model vs terminal cruise missiles after SAM layer.", "Модель очередей АК-630 / Phalanx против крылатых на конечном участке после ЗРК."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Inner zone ~200–2,000 yd in sim", "Ближняя зона ~200–2 000 ярд в симе"),
			i18n.T("Burst magazine 6–16 by class", "БК очередей 6–16 по классу"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Triggers if SAM miss or leak-through", "Срабатывает при промахе ЗРК или прорыве"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Less effective vs Oniks speed — prioritize SAM", "Менее эффективен против скорости «Оникса» — приоритет ЗРК"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("First Harpoon may burn CIWS; follow-up shots more lethal.", "Первый «Гарпун» может сжечь ЗАК; последующие пуски опаснее."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("N/A for player submarine; surface allies rely on PD.", "Н/П для ПЛ игрока; надводные союзники полагаются на ПВО."),
		},
	},
	{
		ID: "wpn_adc", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("ADC — expendable acoustic decoy", "ADC — одноразовая акустическая приманка"),
		ImageFile: "wpn_adc.jpg",
		Credit:    i18n.T("Illustration: U.S. Navy concept / public domain (Wikimedia Commons)", "Иллюстрация: концепт U.S. Navy / общественное достояние (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Mobile noise source deployed against homing torpedoes; magazine ~6 default.", "Подвижный шумовой источник против самонаводящихся торпед; БК ~6 по умолчанию."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Life ~90 s  |  Deploy cooldown ~8 s", "Время жизни ~90 с  |  Кулдаун ~8 с"),
			i18n.T("Seduction bonus vs seekers; verify/reject timers differ player vs enemy fish", "Бонус к уводу ГСН; таймеры verify/reject различаются для своих/вражеских торпед"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Defensive only — no offensive use", "Только оборона"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Best on CPA with speed/depth change", "Лучше на CPA со сменой хода/глубины"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Force enemy to waste ADC before final torpedo shot.", "Вынудите врага потратить ADC до финального пуска."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Deploy early on inbound fish; combine with knuckle.", "Выбрасывайте рано на входящую; сочетайте с «коленом»."),
		},
	},
	{
		ID: "wpn_jitter", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("Jitter — broadband acoustic jammer", "Jitter — широкополосная акустическая помеха"),
		ImageFile: "wpn_jitter.jpg",
		Credit:    i18n.T("Diagram: U.S. Navy AN/SLQ-25 concept / public domain", "Схема: концепт AN/SLQ-25 U.S. Navy / общественное достояние"),
		Summary: []i18n.TranslatedText{
			i18n.T("Confusion jammer vs torpedo seekers; separate magazine from ADC.", "Постановщик помех против ГСН торпед; отдельный БК от ADC."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Life ~75 s  |  Magazine ~6", "Время жизни ~75 с  |  БК ~6"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Defensive only", "Только оборона"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Weaker seduction than ADC but broadens confusion window", "Слабее увод, чем ADC, но шире окно замешательства"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Enemy jitter similarly degrades your fish — expect longer verify times.", "Вражеский jitter так же ухудшает ваши торпеды — ожидайте длинные verify."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Pair with ADC on multi-fish salvo; stagger deploys.", "Сочетайте с ADC при залпе; разнесите выбросы."),
		},
	},
	{
		ID: "wpn_nixie", Allegiance: libWeapons, Kind: world.KindTorpedo,
		Title:     i18n.T("Nixie — towed acoustic decoy", "Nixie — буксируемая акустическая приманка"),
		ImageFile: "wpn_nixie.jpg",
		Credit:    i18n.T("Photo: U.S. Navy / public domain (Wikimedia Commons)", "Фото: U.S. Navy / общественное достояние (Wikimedia Commons)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Surface-ship towed decoy trail (~220 yd); strong seduction vs torpedo seekers when enabled.", "Буксируемая приманка надводного корабля (~220 ярд); сильный увод ГСН торпед при включении."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Toggle on/off; no round magazine — availability per platform", "Вкл/выкл; без расходуемого БК — по платформе"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Surface ASW ships only in sim", "Только надводные ПЛО-корабли в симе"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("High attract multiplier vs torpedoes", "Высокий множитель привлечения для торпед"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Shoot fish from inside turn to keep seeker on hull not trail.", "Пускайте торпеду из поворота, чтобы ГСН держала корпус, а не хвост."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Enable before fish enable; cut tow by maneuver if available.", "Включите до включения ГСН торпеды; срежьте буксир манёвром при возможности."),
		},
	},
}
