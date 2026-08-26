package ui

import (
	"github.com/ssn688/sim/internal/i18n"
	"github.com/ssn688/sim/internal/world"
)

// libraryAllegiance groups catalog entries for the LIBRARY object table.
type libraryAllegiance int

const (
	libHostile libraryAllegiance = iota
	libNeutral
	libFriendly
)

type libraryEntry struct {
	ID         string
	Allegiance libraryAllegiance
	Kind       world.EntityKind
	Title      i18n.TranslatedText
	Summary    []i18n.TranslatedText
	Specs      []i18n.TranslatedText
	Offense    []i18n.TranslatedText
	Defense    []i18n.TranslatedText
	Neutralize []i18n.TranslatedText
	Evade      []i18n.TranslatedText
	ImageFile  string
	Credit     i18n.TranslatedText
}

// libraryCatalog is the ordered platform handbook (no torpedoes / CMs).
var libraryCatalog = []libraryEntry{
	{
		ID: "udaloy", Allegiance: libHostile, Kind: world.KindSurfaceShip,
		Title:     i18n.T("Udaloy DDG — Project 1155 Fregat", "ЭМ «Удалой» — проект 1155 «Фрегат»"),
		ImageFile: "udaloy.jpg",
		Credit:    i18n.T("Photo: U.S. DoD / public domain (Wikimedia)", "Фото: U.S. DoD / общественное достояние (Wikimedia)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Large Soviet/Russian ASW destroyer built to hunt nuclear submarines with helicopters, rocket-assisted torpedoes, and a capable undersea sensor suite. Twin-shaft gas-turbine plant gives strong mid-band shaft and gear tonals.", "Крупный советский/российский противолодочный эсминец: вертолёты, ракето-торпеды и сильный подводный сенсорный комплекс. Двухвальная ГТУ даёт яркие среднечастотные тоналы вала и редуктора."),
			i18n.T("In this scenario an Udaloy is a primary surface hunter: expect Rastrub (Metel) ASW rockets, lightweight fish, and point-defense layers against Harpoon.", "В сценарии «Удалой» — главный надводный охотник: ракеты «Раструб» («Метель»), лёгкие торпеды и эшелон ПВО против «Гарпуна»."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Displacement ~7,500 t full load", "Водоизмещение ~7 500 т полное"),
			i18n.T("Length ~163 m  |  Speed ~29–30 kn", "Длина ~163 м  |  Скорость ~29–30 уз"),
			i18n.T("Crew ~300  |  ASW helicopters: Ka-27 (2)", "Экипаж ~300  |  ПЛК вертолёты: Ка-27 (2)"),
			i18n.T("Acoustic: loud GT / reduction-gear cluster; blade ~1.65 Hz", "Акустика: громкий кластер ГТ/редуктора; лопастной ~1,65 Гц"),
			i18n.T("Radar: MR-320 Fregat (Top Plate) — S-band air/surface, ~6–12 rpm (5–10 s/scan)", "РЛС: МР-320 «Фрегат» (Top Plate) — S-диапазон воздух/поверхность, ~6–12 об/мин (5–10 с/обзор)"),
			i18n.T("Mast detect (calm): ~14 kyd vs raised ESM/periscope stalk", "Обнаружение мачты (штиль): ~14 кярд против поднятой ESM/перископа"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Rastrub (Metel) ASW rocket — magazine ~8; splash → UMGT-1", "ПЛУР «Раструб» («Метель») — БК ~8; всплеск → УМГТ-1"),
			i18n.T("Ship tubes — UMGT-1 lightweight ASW torpedoes", "Корабельные ТА — лёгкие ПЛ торпеды УМГТ-1"),
			i18n.T("Active sonar / helicopter cueing for localization", "Активный ГАС / наведение вертолёта для локализации"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Kinzhal/Osa-M class SAM — medium AAW vs Harpoon (~2–8 kyd)", "ЗРК класса «Кинжал»/«Оса-М» — средняя ПВО против «Гарпуна» (~2–8 кярд)"),
			i18n.T("AK-630 CIWS bursts — inner layer (~0.2–2 kyd)", "Очереди АК-630 ЗАК — ближний рубеж (~0,2–2 кярд)"),
			i18n.T("Hull strength: survives glancing fish; vulnerable to Mk48 under keel", "Прочность корпуса: выдерживает касательные попадания; уязвим к Mk48 под килем"),
			i18n.T("Search radar paints raised masts; storm seas reduce mast RCS", "Обзорная РЛС засвечивает поднятые мачты; шторм снижает ЭПР мачты"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Prefer Mk48 from outside helicopter/Rastrub comfort range.", "Предпочтителен Mk48 извне комфортной зоны вертолёта/«Раструба»."),
			i18n.T("Harpoon can work if SAM/CIWS are depleted or you fire from surprise/shallow snapshot geometry.", "«Гарпун» возможен при истощении ЗРК/ЗАК или внезапном/мелководном залпе."),
			i18n.T("Do not linger on a steady bearing while within ASW rocket range.", "Не держитесь на устойчивом пеленге в зоне ПЛУР."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Go deep / under layer; slow to cut self-noise if he is searching passively.", "Уйдите глубже / под слой; снизьте ход, если он ищет пассивно."),
			i18n.T("If Rastrub/UMGT-1 in water: speed change + depth + ADC/jitter; break Doppler.", "При «Раструб»/УМГТ-1 в воде: смена хода + глубина + ADC/jitter; сорвите доплер."),
			i18n.T("Avoid periscope/radar exposure while his Helix is airborne.", "Не светите перископ/РЛС, пока его Helix в воздухе."),
		},
	},
	{
		ID: "krivak", Allegiance: libHostile, Kind: world.KindSurfaceShip,
		Title:     i18n.T("Krivak FF — Project 1135 Burevestnik", "СКР «Кривак» — проект 1135 «Буревестник»"),
		ImageFile: "krivak.jpg",
		Credit:    i18n.T("Photo: U.S. DoD / public domain (Wikimedia)", "Фото: U.S. DoD / общественное достояние (Wikimedia)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Escort frigate optimized for ASW patrol and convoy work. Smaller and leaner than Udaloy, but still carries rocket ASW and tubes with a noisy twin-screw signature.", "Эскортный фрегат для ПЛО-патруля и конвоев. Меньше и легче «Удалого», но с ПЛУР и ТА; шумная двухвинтовая сигнатура."),
			i18n.T("Treat as a persistent surface threat that can prosecute a contact once localized by own sensors or cueing.", "Считайте устойчивой надводной угрозой: может преследовать контакт после локализации своими средствами или по наведению."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Displacement ~3,500 t full load", "Водоизмещение ~3 500 т полное"),
			i18n.T("Length ~123 m  |  Speed ~32 kn", "Длина ~123 м  |  Скорость ~32 уз"),
			i18n.T("Crew ~180–200", "Экипаж ~180–200"),
			i18n.T("Acoustic: shaft lines denser than Udaloy; blade ~2.15 Hz", "Акустика: валовые линии плотнее, чем у «Удалого»; лопастной ~2,15 Гц"),
			i18n.T("Radar: MR-310U Angara-M — S-band air/surface, ~12 rpm (≈5 s/scan)", "РЛС: МР-310У «Ангара-М» — S воздух/поверхность, ~12 об/мин (≈5 с/обзор)"),
			i18n.T("Mast detect (calm): ~11 kyd vs raised ESM stalk", "Обнаружение мачты (штиль): ~11 кярд против поднятой ESM"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Rastrub ASW rocket — magazine ~8; splash → UMGT-1", "ПЛУР «Раструб» — БК ~8; всплеск → УМГТ-1"),
			i18n.T("Ship tubes — UMGT-1", "Корабельные ТА — УМГТ-1"),
			i18n.T("Limited organic air compared with Udaloy", "Ограниченная авиация по сравнению с «Удалым»"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("SAM layer vs sea-skimmers (magazine ~8)", "ЗРК против низколетящих (БК ~8)"),
			i18n.T("CIWS bursts (magazine ~10)", "Очереди ЗАК (БК ~10)"),
			i18n.T("Thinner hull than cruisers — Mk48 remains decisive", "Корпус тоньше крейсеров — Mk48 по-прежнему решает"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Mk48 preferred; Harpoon viable after baiting or depleting PD.", "Предпочтителен Mk48; «Гарпун» возможен после изматывания/истощения ПВО."),
			i18n.T("Exploit his smaller ASW aviation footprint vs Udaloy.", "Используйте меньший авиационный след ПЛО по сравнению с «Удалым»."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Layer + speed management; he relies more on ship sensors.", "Слой + управление ходом; он больше опирается на корабельные сенсоры."),
			i18n.T("Same CM doctrine vs lightweight fish as vs Udaloy.", "Та же доктрина ПМ против лёгких торпед, что и против «Удалого»."),
		},
	},
	{
		ID: "kresta2", Allegiance: libHostile, Kind: world.KindSurfaceShip,
		Title:     i18n.T("Kresta II CG — Project 1134A Berkut-A", "КР «Креста-II» — проект 1134А «Беркут-А»"),
		ImageFile: "kresta2.jpg",
		Credit:    i18n.T("Photo: U.S. DoD / public domain (Wikimedia)", "Фото: U.S. DoD / общественное достояние (Wikimedia)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Large ASW cruiser with a heavier steam plant and denser LOFAR fingerprint than Udaloy. Carries more Rastrub rounds and stronger point-defense magazines.", "Крупный ПЛО-крейсер с более тяжёлой паросиловой установкой и плотным LOFAR-отпечатком. Больше «Раструбов» и сильнее БК точечной ПВО."),
			i18n.T("A high-value surface unit: dangerous in a prolonged ASW chase and harder to attrit with a single Harpoon.", "Ценная надводная цель: опасен в затяжной ПЛО-охоте и труднее «выгрызть» одним «Гарпуном»."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Displacement ~7,500–8,000 t full load", "Водоизмещение ~7 500–8 000 т полное"),
			i18n.T("Length ~159 m  |  Speed ~32–34 kn", "Длина ~159 м  |  Скорость ~32–34 уз"),
			i18n.T("Crew ~340+  |  Helicopter: Ka-25/Ka-27 capable", "Экипаж ~340+  |  Вертолёт: Ка-25/Ка-27"),
			i18n.T("Acoustic: heavy steam/gear tonals; blade ~1.45 Hz", "Акустика: тяжёлые паровые/редукторные тоналы; лопастной ~1,45 Гц"),
			i18n.T("Radar: MR-320 Fregat class — S-band air/surface, ~6–12 rpm (5–10 s/scan)", "РЛС: класса МР-320 «Фрегат» — S воздух/поверхность, ~6–12 об/мин (5–10 с/обзор)"),
			i18n.T("Mast detect (calm): ~14 kyd vs raised ESM stalk", "Обнаружение мачты (штиль): ~14 кярд против поднятой ESM"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Rastrub magazine ~12 — sustained rocket ASW", "БК «Раструб» ~12 — длительный ракетный ПЛО"),
			i18n.T("Ship tubes — UMGT-1 (magazine ~8)", "Корабельные ТА — УМГТ-1 (БК ~8)"),
			i18n.T("Strong search/prosecution endurance", "Высокая автономность поиска/преследования"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("SAM magazine ~12", "БК ЗРК ~12"),
			i18n.T("CIWS magazine ~14", "БК ЗАК ~14"),
			i18n.T("Large target: easier acoustic/TMA hold once detected", "Крупная цель: проще удерживать акустикой/TMA после обнаружения"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Plan multi-weapon: Mk48 primary; Harpoon only with PD saturation or distraction.", "Планируйте многооружие: Mk48 основной; «Гарпун» — при насыщении/отвлечении ПВО."),
			i18n.T("Expect him to keep firing Rastrub longer than escorts.", "Ожидайте более длительных залпов «Раструба», чем у эскорта."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Break contact early; do not slug it out inside rocket envelopes.", "Рвите контакт рано; не тяните бой внутри ракетных зон."),
			i18n.T("Use bathymetry/layer; dump CM only on CPA threat, not every launch.", "Используйте рельеф/слой; ПМ — только при угрозе CPA, не на каждый пуск."),
		},
	},
	{
		ID: "grisha", Allegiance: libHostile, Kind: world.KindSurfaceShip,
		Title:     i18n.T("Grisha Corvette — Project 1124 Albatros", "Корвет «Гриша» — проект 1124 «Альбатрос»"),
		ImageFile: "grisha.jpg",
		Credit:    i18n.T("Photo: U.S. DoD / DPLA / public domain (Wikimedia)", "Фото: U.S. DoD / DPLA / общественное достояние (Wikimedia)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Small coastal ASW corvette. No Rastrub — instead RBU rocket depth charges and SET-40 lightweight tubes. Quieter mid-band than destroyers but still a surface broadband source.", "Малый прибрежный ПЛО-корвет. Без «Раструба» — РБУ и лёгкие СЭТ-40. Тише эсминцев в середине полосы, но всё ещё широкополосный надводный источник."),
			i18n.T("Dangerous inshore: RBU can punish a shallow or poorly placed boat at short range.", "Опасен у берега: РБУ бьёт по мелкой или плохо стоящей лодке на короткой дистанции."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Displacement ~1,000 t full load", "Водоизмещение ~1 000 т полное"),
			i18n.T("Length ~71 m  |  Speed ~35 kn burst", "Длина ~71 м  |  Скорость ~35 уз рывком"),
			i18n.T("Crew ~80–90", "Экипаж ~80–90"),
			i18n.T("Acoustic: higher blade rate ~2.6 Hz; lean mid-band", "Акустика: выше лопастной ~2,6 Гц; бедная середина полосы"),
			i18n.T("Radar: MR-302 Rubka — X-band surface search, ~15 rpm (≈4 s/scan)", "РЛС: МР-302 «Рубка» — X обзор поверхности, ~15 об/мин (≈4 с/обзор)"),
			i18n.T("Mast detect (calm): ~9 kyd vs thin periscope/ESM stalk", "Обнаружение мачты (штиль): ~9 кярд против тонкого перископа/ESM"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("RBU salvos — short-range ASW; debris/light damage in sim", "Залпы РБУ — ближний ПЛО; обломки/лёгкие повреждения в симе"),
			i18n.T("Ship tubes — SET-40 (magazine ~4)", "Корабельные ТА — СЭТ-40 (БК ~4)"),
			i18n.T("No Metel/Rastrub rocket ASW", "Нет ПЛУР «Метель»/«Раструб»"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Light SAM (magazine ~4)", "Лёгкий ЗРК (БК ~4)"),
			i18n.T("Light CIWS (magazine ~6)", "Лёгкий ЗАК (БК ~6)"),
			i18n.T("Fragile vs Mk48; Harpoon often overkill but effective", "Хрупок против Mk48; «Гарпун» часто избыточен, но эффективен"),
			i18n.T("Search radar will paint a raised ESM mast inside detect range", "Обзорная РЛС засветит поднятую ESM в зоне обнаружения"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Mk48 or Harpoon from outside RBU comfort; do not duel shallow.", "Mk48 или «Гарпун» извне зоны РБУ; не дуэлируйте на мели."),
			i18n.T("He is easier to kill than Udaloy/Kresta once weapons are on bearing.", "Его легче поразить, чем «Удалой»/«Кресту», когда оружие на пеленге."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Stay deep/outside RBU range; speed away if he closes hot.", "Держитесь глубже/вне зоны РБУ; уходите ходом при горячем сближении."),
			i18n.T("SET-40: treat like other lightweight fish — CM + maneuver on CPA.", "СЭТ-40: как другие лёгкие торпеды — ПМ + манёвр по CPA."),
		},
	},
	{
		ID: "gorshkov", Allegiance: libHostile, Kind: world.KindSurfaceShip,
		Title:     i18n.T("Gorshkov FFG — Project 22350 Admiral Gorshkov", "Фрегат «Горшков» — проект 22350 «Адмирал Горшков»"),
		ImageFile: "gorshkov.jpg",
		Credit:    i18n.T("Photo: Russian MoD / CC BY 4.0 (Wikimedia)", "Фото: МО РФ / CC BY 4.0 (Wikimedia)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Modern multipurpose frigate: Poliment-Redut air defense, UKSK strike/ASW cells, and Paket-NK lightweight ASW. Quieter CODAG plant than Cold War destroyers, but still a strong mid-band surface source.", "Современный многоцелевой фрегат: «Полимент-Редут», УКСК удар/ПЛО и «Пакет-НК». Тише ГТУ холодной войны, но всё ещё сильный среднечастотный надводный источник."),
			i18n.T("Not in the demo mission yet. ASW rocket employment is Otvet (UKSK) using the same splash→lightweight fish model as Metel.", "Пока нет в демо-миссии. ПЛУР — «Ответ» (УКСК) по той же модели всплеск→лёгкая торпеда, что и «Метель»."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Displacement ~5,400 t full load", "Водоизмещение ~5 400 т полное"),
			i18n.T("Length ~135 m  |  Speed ~29–30 kn", "Длина ~135 м  |  Скорость ~29–30 уз"),
			i18n.T("Crew ~210  |  Helicopter: Ka-27 capable", "Экипаж ~210  |  Вертолёт: Ка-27"),
			i18n.T("Acoustic: CODAG shaft/gear cluster; blade ~1.9 Hz", "Акустика: CODAG вал/редуктор; лопастной ~1,9 Гц"),
			i18n.T("Radar: Poliment / Furke-4 — S/X air+surface, ~4.5 s/scan", "РЛС: «Полимент» / «Фуркэ-4» — S/X воздух+поверхность, ~4,5 с/обзор"),
			i18n.T("Mast detect (calm): ~13 kyd vs raised ESM/periscope stalk", "Обнаружение мачты (штиль): ~13 кярд против поднятой ESM/перископа"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Otvet ASW missile (UKSK) — magazine ~8; splash → UMGT-1", "ПЛУР «Ответ» (УКСК) — БК ~8; всплеск → УМГТ-1"),
			i18n.T("Paket-NK ship tubes — UMGT-1 stand-in (magazine ~8)", "ТА «Пакет-НК» — замена УМГТ-1 (БК ~8)"),
			i18n.T("No RBU; UKSK AShM (Kalibr/Oniks/Zircon) not modeled vs player", "Нет РБУ; ПКР УКСК (Калибр/Оникс/Циркон) против игрока не моделируются"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Poliment-Redut SAM — deep magazine (~32)", "ЗРК «Полимент-Редут» — глубокий БК (~32)"),
			i18n.T("Palma / Pantsir-M class CIWS (~16 bursts)", "ЗАК класса «Пальма»/«Панцирь-М» (~16 очередей)"),
			i18n.T("Frigate hull: typically 1–2 Mk48 hits; Harpoon contested by PD", "Корпус фрегата: обычно 1–2 попадания Mk48; «Гарпун» оспаривается ПВО"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Mk48 from outside Otvet comfort; respect Redut vs Harpoon.", "Mk48 извне зоны «Ответа»; уважайте «Редут» против «Гарпуна»."),
			i18n.T("Quiet approach — his search radar will paint a raised mast.", "Тихий подход — его обзорная РЛС засветит поднятую мачту."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Deep / under layer vs Otvet splash fish; CM on CPA.", "Глубоко / под слой против торпед после «Ответа»; ПМ по CPA."),
			i18n.T("Avoid mast exposure while he has a solid track.", "Не светите мачтой, пока у него устойчивый трек."),
		},
	},
	{
		ID: "kilo", Allegiance: libHostile, Kind: world.KindSubmarine,
		Title:     i18n.T("Kilo SS — Project 877 Paltus", "ДЭПЛ «Кило» — проект 877 «Палтус»"),
		ImageFile: "kilo.jpg",
		Credit:    i18n.T("Photo: U.S. DoD / DPLA / public domain (Wikimedia)", "Фото: U.S. DoD / DPLA / общественное достояние (Wikimedia)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Diesel-electric attack boat with strong low-frequency shaft/diesel lines when snorkeling or high-rate charging. Quiet on the battery relative to older diesels, but still classifiable on SPECTRUM.", "Дизель-электрическая атакующая лодка с яркими НЧ валовыми/дизельными линиями на РДП или зарядке. На батареях тише старых дизелей, но всё ещё классифицируема на SPECTRUM."),
			i18n.T("Primary weapon threat is heavy 53-series fish at modest cruise speed.", "Главная угроза — тяжёлые торпеды серии 53 на умеренной крейсерской скорости."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Displacement ~3,000 t submerged", "Водоизмещение ~3 000 т подводное"),
			i18n.T("Length ~73 m  |  Speed ~17 kn submerged / ~10 kn snort", "Длина ~73 м  |  Скорость ~17 уз подводная / ~10 уз на РДП"),
			i18n.T("Torpedo tubes: 6 × 533 mm  |  Mag ~12 (sim default)", "ТА: 6 × 533 мм  |  БК ~12 (по умолчанию в симе)"),
			i18n.T("Acoustic: diesel/shaft cluster; blade ~3.1 Hz", "Акустика: дизель/вал; лопастной ~3,1 Гц"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("53-65 heavy torpedoes — cruise ~48 kn in sim", "Тяжёлые 53-65 — крейсер ~48 уз в симе"),
			i18n.T("Passive/active seeker fish; can force CM expenditure", "Пассивный/активный самонаводящийся аппарат; может вынудить расход ПМ"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("ADC / decoy / jammer soft-kill vs ownship fish", "ADC / ложная цель / помехи soft-kill против ваших торпед"),
			i18n.T("No surface SAMs — vulnerable if forced to snort near escorts", "Нет надводного ЗРК — уязвим при вынужденном РДП у эскорта"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Mk48 with good TMA; prefer quiet approach and shot from his baffles.", "Mk48 при хорошем TMA; тихий подход и выстрел из кормовых секторов."),
			i18n.T("Do not fire Harpoon at a submerged Kilo.", "Не стреляйте «Гарпуном» по погружённому «Кило»."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("If he shoots: evade on CPA, deploy CM, change depth/speed.", "Если он стреляет: уклонение по CPA, ПМ, смена глубины/хода."),
			i18n.T("Own quieting + layer denial reduces his detection advantage.", "Собственная скрытность + слой снижают его преимущество в обнаружении."),
		},
	},
	{
		ID: "foxtrot", Allegiance: libHostile, Kind: world.KindSubmarine,
		Title:     i18n.T("Foxtrot SS — Project 641", "ДЭПЛ «Фокстрот» — проект 641"),
		ImageFile: "foxtrot.jpg",
		Credit:    i18n.T("Photo: U.S. DoD / public domain (Wikimedia)", "Фото: U.S. DoD / общественное достояние (Wikimedia)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Older diesel boat: louder snort/diesel fingerprint and thinner modern quieting. Easier to classify and track than Kilo or Victor once you have solid tonals.", "Старая дизельная лодка: громче РДП/дизель и слабее современное заглушение. Легче классифицировать и вести, чем «Кило» или «Виктор», при устойчивых тоналах."),
			i18n.T("Still lethal at short range with heavy fish, but slower cruise and smaller magazine.", "По-прежнему смертелен на короткой дистанции тяжёлыми торпедами, но медленнее и с меньшим БК."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Displacement ~2,500 t submerged", "Водоизмещение ~2 500 т подводное"),
			i18n.T("Length ~91 m  |  Speed ~15–16 kn submerged", "Длина ~91 м  |  Скорость ~15–16 уз подводная"),
			i18n.T("Mag ~10 heavy fish  |  Cruise fish ~40 kn in sim", "БК ~10 тяжёлых  |  Крейсер торпеды ~40 уз в симе"),
			i18n.T("Acoustic: loud diesel lines; blade ~2.8 Hz", "Акустика: громкие дизельные линии; лопастной ~2,8 Гц"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("53-65 heavy torpedoes (slower cruise than Kilo/Victor)", "Тяжёлые 53-65 (медленнее крейсер, чем у Кило/Виктор)"),
			i18n.T("Limited magazine — he cannot spam forever", "Ограниченный БК — он не может стрелять бесконечно"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Soft-kill CM only", "Только soft-kill ПМ"),
			i18n.T("High self-noise when snorkeling — exploitable for TMA", "Высокий собственный шум на РДП — удобно для TMA"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Mk48; use his noisy signature to hold bearing/range.", "Mk48; используйте шумную сигнатуру для удержания пеленга/дистанции."),
			i18n.T("Press when he is forced to snort.", "Давите, когда он вынужден идти на РДП."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Standard fish evasion; his slower fish gives more reaction time.", "Стандартное уклонение; его более медленные торпеды дают больше времени."),
			i18n.T("Do not mirror his depth if he is clearly above/below the layer.", "Не копируйте его глубину, если он явно выше/ниже слоя."),
		},
	},
	{
		ID: "victor_iii", Allegiance: libHostile, Kind: world.KindSubmarine,
		Title:     i18n.T("Victor III SSN — Project 671RTM Shchuka", "ПЛА «Виктор-III» — проект 671РТМ «Щука»"),
		ImageFile: "victor_iii.jpg",
		Credit:    i18n.T("Photo: U.S. Navy / DoD / public domain (Wikimedia)", "Фото: U.S. Navy / DoD / общественное достояние (Wikimedia)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Nuclear hunter with a dense turbine/pump tonal set — noisier than a 688, but faster and better sustained submerged endurance than diesels. Primary peer ASW threat underwater.", "Атомный охотник с плотным турбинно-насосным набором — шумнее 688, но быстрее и выносливее дизелей. Главная равноценная ПЛО-угроза под водой."),
			i18n.T("Larger magazine and faster fish cruise make him the most dangerous submarine class in the training set.", "Больший БК и более быстрый крейсер торпед делают его самым опасным классом ПЛ в учебном наборе."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Displacement ~6,000 t submerged", "Водоизмещение ~6 000 т подводное"),
			i18n.T("Length ~107 m  |  Speed ~30+ kn submerged", "Длина ~107 м  |  Скорость ~30+ уз подводная"),
			i18n.T("Mag ~18  |  Fish cruise ~55 kn in sim", "БК ~18  |  Крейсер торпеды ~55 уз в симе"),
			i18n.T("Acoustic: dense nuclear plant lines; blade ~3.6 Hz", "Акустика: плотные линии АЭУ; лопастной ~3,6 Гц"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("53-65 heavy torpedoes — high speed, deep magazine", "Тяжёлые 53-65 — высокая скорость, глубокий БК"),
			i18n.T("Aggressive ASW AI when DEFCON rises", "Агрессивный ПЛО ИИ при росте DEFCON"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Soft-kill CM; nuclear mobility to reopen geometry", "Soft-kill ПМ; атомная подвижность для перестройки геометрии"),
			i18n.T("No SAMs — still a submerged target for Mk48 only", "Нет ЗРК — по-прежнему подводная цель только для Mk48"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Best TMA you can get before committing Mk48.", "Лучший возможный TMA до выстрела Mk48."),
			i18n.T("Expect counterfire; plan escape geometry before launch.", "Ожидайте ответный огонь; планируйте отход до пуска."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("He can outrun poor geometry — prioritize early CM + radical course/depth.", "Он перегонит плохую геометрию — приоритет: ранние ПМ + резкий курс/глубина."),
			i18n.T("Do not sprint needlessly; his sensors punish cavitation.", "Не спринтуйте без нужды; его сенсоры наказывают кавитацию."),
		},
	},
	{
		ID: "yasen_m", Allegiance: libHostile, Kind: world.KindSubmarine,
		Title:     i18n.T("Yasen-M SSN — Project 885M Yaseny", "ПЛА «Ясень-М» — проект 885М"),
		ImageFile: "yasen_m.jpg",
		Credit:    i18n.T("Photo: kremlin.ru / CC BY 4.0 (Wikimedia) — K-560 Severodvinsk class stand-in", "Фото: kremlin.ru / CC BY 4.0 (Wikimedia) — замена класса К-560 «Северодвинск»"),
		Summary: []i18n.TranslatedText{
			i18n.T("Newest Russian multipurpose nuclear boat: pump-jet quieting, sparse plant tonals, and a deep mixed magazine. Harder to classify than Victor III and closer to a 688 acoustic problem.", "Новейшая российская многоцелевая АПЛ: водомётное заглушение, редкие тоналы установки и глубокий смешанный БК. Труднее классифицировать, чем «Виктор-III», ближе к акустике 688."),
			i18n.T("Not in the demo mission yet — treat as a future peer threat once spawned in custom scenarios.", "Пока нет в демо — считайте будущей равноценной угрозой в пользовательских сценариях."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Displacement ~13,800 t submerged (class figures)", "Водоизмещение ~13 800 т подводное (по классу)"),
			i18n.T("Length ~139 m  |  Speed ~31 kn submerged (sim)", "Длина ~139 м  |  Скорость ~31 уз подводная (сим)"),
			i18n.T("Tubes: 10 × 533 mm  |  Mag ~24 heavy fish  |  Cruise ~55 kn", "ТА: 10 × 533 мм  |  БК ~24 тяжёлых  |  Крейсер ~55 уз"),
			i18n.T("Acoustic: quiet nuclear + pump-jet; blade ~2.4 Hz", "Акустика: тихая АЭУ + водомёт; лопастной ~2,4 Гц"),
			i18n.T("UKSK VLS (Kalibr/Oniks/Zircon) — not modeled as AI weapons in this build", "УКСК УВП (Калибр/Оникс/Циркон) — как оружие ИИ в этой сборке не моделируются"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("UGST / Fizik-class heavy torpedoes — peer speed, large magazine", "Тяжёлые УГСТ / «Физик» — равноценная скорость, большой БК"),
			i18n.T("Sticky prosecute AI once contact is held", "Настойчивое преследование ИИ после удержания контакта"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Soft-kill CM; high submerged mobility", "Soft-kill ПМ; высокая подводная подвижность"),
			i18n.T("No SAMs — Mk48 remains the kill weapon", "Нет ЗРК — Mk48 остаётся оружием поражения"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Invest in TMA; he is quieter — do not shoot on a weak class call.", "Вложитесь в TMA; он тише — не стреляйте по слабому классу."),
			i18n.T("Expect counterfire from a deep magazine if you are noisy.", "Ожидайте ответ из глубокого БК, если вы шумны."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("His fish are fast — early CM + depth/course change on CPA.", "Его торпеды быстры — ранние ПМ + глубина/курс по CPA."),
			i18n.T("Layer denial and own quieting matter more than vs Foxtrot.", "Слой и собственная скрытность важнее, чем против «Фокстрота»."),
		},
	},
	{
		ID: "merchant", Allegiance: libNeutral, Kind: world.KindSurfaceShip,
		Title:     i18n.T("Merchant Freighter (MV)", "Торговый сухогруз (ТР)"),
		ImageFile: "merchant.jpg",
		Credit:    i18n.T("Photo: U.S. Navy / public domain (Wikimedia) — cargo ship stand-in", "Фото: U.S. Navy / общественное достояние (Wikimedia) — замена грузового судна"),
		Summary: []i18n.TranslatedText{
			i18n.T("Civilian dry-cargo traffic. Broadband and shaft tonals without combat sensors or weapons. Useful for masking/bearing confusion, disastrous if you attack by mistake.", "Гражданский сухогруз. Широкополосный шум и валовые тоналы без боевых сенсоров и оружия. Полезен для маскировки/путаницы пеленгов; катастрофа при ошибочной атаке."),
			i18n.T("Rules of engagement: classify before shoot — freighters are mission-failure risks if destroyed.", "Правила применения: классифицируйте до выстрела — уничтожение сухогруза проваливает миссию."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Large merchant displacement (variable)", "Большое торговое водоизмещение (переменное)"),
			i18n.T("Speed typically 12–18 kn", "Скорость обычно 12–18 уз"),
			i18n.T("Acoustic: steady shaft lines; blade ~1.4 Hz", "Акустика: устойчивые валовые линии; лопастной ~1,4 Гц"),
			i18n.T("Radar: commercial X-band nav — ~24 rpm (≈2.5 s/scan)", "РЛС: коммерческая X-навигация — ~24 об/мин (≈2,5 с/обзор)"),
			i18n.T("Mast detect (calm): ~4.5 kyd — still paints a raised stalk nearby", "Обнаружение мачты (штиль): ~4,5 кярд — всё ещё засвечивает поднятый стебель рядом"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("None — unarmed civilian", "Нет — безоружный гражданский"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Nav radar only — no SAM/CIWS/ASW", "Только навигационная РЛС — нет ЗРК/ЗАК/ПЛО"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Do not engage. Reclassify if SPECTRUM looks combatant-like.", "Не атакуйте. Переклассифицируйте, если SPECTRUM похож на боевой."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Not a weapons threat. Avoid collision; use as acoustic clutter carefully.", "Не оружейная угроза. Избегайте столкновения; акустический фон используйте осторожно."),
		},
	},
	{
		ID: "tanker", Allegiance: libNeutral, Kind: world.KindSurfaceShip,
		Title:     i18n.T("Oil Tanker (VLCC / product)", "Нефтетанкер (VLCC / продуктовоз)"),
		ImageFile: "tanker.jpg",
		Credit:    i18n.T("Photo: U.S. DoD / public domain (Wikimedia)", "Фото: U.S. DoD / общественное достояние (Wikimedia)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Large tanker with deep low-frequency plant noise. Extremely visible on broadband; not a combatant.", "Крупный танкер с глубоким НЧ шумом установки. Очень заметен на широкополосном; не боевой."),
			i18n.T("Collateral damage and political cost of attack are catastrophic — never a valid weapons target in this mission.", "Сопутствующий ущерб и политическая цена атаки катастрофичны — никогда не цель для оружия в этой миссии."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Very large displacement", "Очень большое водоизмещение"),
			i18n.T("Speed typically 12–16 kn", "Скорость обычно 12–16 уз"),
			i18n.T("Acoustic: strong LF plant; blade ~1.1 Hz", "Акустика: сильная НЧ установка; лопастной ~1,1 Гц"),
			i18n.T("Radar: commercial S-band nav — ~20 rpm (≈3 s/scan)", "РЛС: коммерческая S-навигация — ~20 об/мин (≈3 с/обзор)"),
			i18n.T("Mast detect (calm): ~5 kyd", "Обнаружение мачты (штиль): ~5 кярд"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("None — unarmed civilian", "Нет — безоружный гражданский"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Nav radar only", "Только навигационная РЛС"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Do not engage.", "Не атакуйте."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Collision avoidance only; may mask bearings if on same line of sound.", "Только избегание столкновения; может маскировать пеленги на одной линии звука."),
		},
	},
	{
		ID: "fishing", Allegiance: libNeutral, Kind: world.KindSurfaceShip,
		Title:     i18n.T("Fishing Trawler (FV)", "Рыболовный траулер (РТ)"),
		ImageFile: "fishing.jpg",
		Credit:    i18n.T("Photo: Jebulon / CC BY-SA 3.0 (Wikimedia)", "Фото: Jebulon / CC BY-SA 3.0 (Wikimedia)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Small fishing vessel with higher blade rate and intermittent machinery. Easy to confuse with a patrol craft at long range until tonals clarify.", "Малое рыболовное судно с более высоким лопастным и прерывистыми механизмами. Легко спутать с патрульным на дальней дистанции, пока тоналы не прояснятся."),
			i18n.T("Unarmed; classify carefully before any weapons solution.", "Безоружен; тщательно классифицируйте до любого оружейного решения."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Small displacement", "Малое водоизмещение"),
			i18n.T("Speed typically 8–12 kn", "Скорость обычно 8–12 уз"),
			i18n.T("Acoustic: higher blade ~2.4 Hz; leaner spectrum", "Акустика: выше лопастной ~2,4 Гц; беднее спектр"),
			i18n.T("Radar: small-craft X-band nav — ~24 rpm (≈2.5 s/scan)", "РЛС: маломерная X-навигация — ~24 об/мин (≈2,5 с/обзор)"),
			i18n.T("Mast detect (calm): ~2.8 kyd", "Обнаружение мачты (штиль): ~2,8 кярд"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("None — unarmed civilian", "Нет — безоружный гражданский"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("Nav radar only", "Только навигационная РЛС"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Do not engage.", "Не атакуйте."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Not a weapons threat; watch close-aboard collision risk in traffic.", "Не оружейная угроза; следите за риском столкновения вблизи в трафике."),
		},
	},
	{
		ID: "los_angeles", Allegiance: libFriendly, Kind: world.KindSubmarine,
		Title:     i18n.T("Los Angeles SSN — SSN-688 (Ownship)", "ПЛА «Лос-Анджелес» — SSN-688 (свой корабль)"),
		ImageFile: "los_angeles.jpg",
		Credit:    i18n.T("Photo: U.S. Navy / public domain (Wikimedia)", "Фото: U.S. Navy / общественное достояние (Wikimedia)"),
		Summary: []i18n.TranslatedText{
			i18n.T("Your boat: Improved Los Angeles–class nuclear attack submarine. Quiet machinery set with sparse LOFAR lines relative to Soviet hunters, armed with Mk48 and Harpoon/ASROC loadouts from tubes.", "Ваша лодка: улучшенная АПЛ класса «Лос-Анджелес». Тихий машинный набор с редкими LOFAR-линиями относительно советских охотников; Mk48 и «Гарпун»/ASROC из ТА."),
			i18n.T("Protect ownship systems (sonar, propulsion, weapons) — damage cascades end the patrol. Ally LA boats use the same signature when spawned as AI friendlies.", "Берегите системы своего корабля (ГАС, ход, оружие) — каскад повреждений срывает патруль. Союзные ЛА используют ту же сигнатуру как ИИ-друзья."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Displacement ~6,900 t submerged", "Водоизмещение ~6 900 т подводное"),
			i18n.T("Length ~110 m  |  Speed 30+ kn submerged", "Длина ~110 м  |  Скорость 30+ уз подводная"),
			i18n.T("Tubes: 4 × 533 mm  |  Mk48 / Harpoon / ASROC (sim)", "ТА: 4 × 533 мм  |  Mk48 / «Гарпун» / ASROC (сим)"),
			i18n.T("Acoustic: quiet SSN; blade ~4.2 Hz harmonics", "Акустика: тихая ПЛА; гармоники лопастного ~4,2 Гц"),
			i18n.T("ESM mast: raise at periscope depth (≤60 ft, ≤8 kn); auto-lowers at ≥65 ft or ≥8.5 kn", "Мачта ESM: подъём на перископной (≤60 фут, ≤8 уз); автоспуск при ≥65 фут или ≥8,5 уз"),
			i18n.T("COMM mast: same limits; receive scheduled fleet traffic when raised", "Мачта COMM: те же пределы; приём расписанного флотского трафика при подъёме"),
			i18n.T("Periscope: same depth/speed limits; train/zoom on MAST; raised optic is radar-detectable", "Перископ: те же пределы глубины/хода; наводка/зум на MAST; поднятая оптика обнаруживается РЛС"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("Mk48 ADCAP — primary ASW/ASuW fish", "Mk48 ADCAP — основная ПЛО/ПКО торпеда"),
			i18n.T("UGM-84 Harpoon — anti-surface cruise missile", "UGM-84 «Гарпун» — противокорабельная крылатая ракета"),
			i18n.T("ASROC (sim) — rocket-assisted ASW option where loaded", "ASROC (сим) — ракетный ПЛО-вариант при наличии в БК"),
			i18n.T("ESM intercept of surface search/nav radars (MAST screen)", "Перехват ESM надводных обзорных/навигационных РЛС (экран MAST)"),
			i18n.T("COMM inbox: briefing at start; follow-on orders need antenna up", "Входящие COMM: брифинг в начале; последующие приказы — при поднятой антенне"),
			i18n.T("Positive ID: periscope visual inside 3000 yd, or 80% harmonic match with library fingerprint held 2 minutes", "Положительная ИД: визуал перископом <3000 ярд или 80% гармоническое совпадение с библиотекой 2 минуты"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("ADC / jitter / Nixie soft-kill vs inbound fish", "ADC / jitter / Nixie soft-kill против входящих торпед"),
			i18n.T("Depth, speed, layer, and bathymetry are primary survival tools", "Глубина, ход, слой и рельеф — главные инструменты выживания"),
			i18n.T("No organic SAM — do not expose to air/surface fire needlessly", "Нет собственного ЗРК — не светитесь под воздух/надводный огонь без нужды"),
			i18n.T("Raised ESM/COMM/periscope masts are radar-detectable; watch illumination bar", "Поднятые мачты ESM/COMM/перископ обнаруживаются РЛС; следите за шкалой облучения"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("N/A — this is ownship. Use loadout per threat class above.", "Н/Д — это свой корабль. Используйте БК по классу угрозы выше."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Standard torpedo evasion: CPA-driven CM, depth change, speed change.", "Стандартное уклонение от торпед: ПМ по CPA, смена глубины и хода."),
			i18n.T("Manage self-noise; damaged arrays reduce your ability to fight.", "Управляйте собственным шумом; повреждённые антенны снижают боеспособность."),
		},
	},
	{
		ID: "spruance", Allegiance: libFriendly, Kind: world.KindSurfaceShip,
		Title:     i18n.T("Spruance DDG — DD-963 class", "ЭМ «Спрюэнс» — класс DD-963"),
		ImageFile: "spruance.jpg",
		Credit:    i18n.T("Photo: U.S. Navy / public domain (Wikimedia)", "Фото: U.S. Navy / общественное достояние (Wikimedia)"),
		Summary: []i18n.TranslatedText{
			i18n.T("US ASW destroyer: twin LM2500 COGAG plant, ASROC rocket ASW, Mk46 lightweight tubes, and a layered Sea Sparrow / Phalanx self-defense suite.", "Американский ПЛО-эсминец: сдвоенная COGAG LM2500, ПЛУР ASROC, лёгкие Mk46 и эшелон Sea Sparrow / Phalanx."),
			i18n.T("Available as an AI ally (SidePlayer). No blue-force tracker — you only see him via honest acoustic / visual / ESM contact. In DEBUG, oracle plot shows true position.", "Доступен как союзный ИИ (SidePlayer). Нет синего трекера — видите его только честным акустическим / визуальным / ESM контактом. В DEBUG оракул показывает истинную позицию."),
		},
		Specs: []i18n.TranslatedText{
			i18n.T("Displacement ~8,000 t full load", "Водоизмещение ~8 000 т полное"),
			i18n.T("Length ~172 m  |  Speed ~32 kn (sim)", "Длина ~172 м  |  Скорость ~32 уз (сим)"),
			i18n.T("Crew ~300+  |  Helicopter: ASW capable", "Экипаж ~300+  |  Вертолёт: ПЛО"),
			i18n.T("Acoustic: GT/gear mid-band; blade ~1.75 Hz", "Акустика: ГТ/редуктор середина полосы; лопастной ~1,75 Гц"),
			i18n.T("Radar: SPS-40 / SPS-55 — S/X air+surface, ~5 s/scan", "РЛС: SPS-40 / SPS-55 — S/X воздух+поверхность, ~5 с/обзор"),
			i18n.T("Mast detect (calm): ~13 kyd vs raised ESM/periscope stalk", "Обнаружение мачты (штиль): ~13 кярд против поднятой ESM/перископа"),
		},
		Offense: []i18n.TranslatedText{
			i18n.T("ASROC — magazine ~8; splash → Mk46", "ASROC — БК ~8; всплеск → Mk46"),
			i18n.T("Ship tubes — Mk46 (magazine ~6)", "Корабельные ТА — Mk46 (БК ~6)"),
			i18n.T("No RBU; Harpoon not modeled as ally AI weapon", "Нет РБУ; «Гарпун» как оружие союзного ИИ не моделируется"),
		},
		Defense: []i18n.TranslatedText{
			i18n.T("NATO Sea Sparrow / VLS-era SAM stand-in (~24)", "Замена ЗРК NATO Sea Sparrow / эпохи УВП (~24)"),
			i18n.T("Phalanx CIWS (~12 bursts)", "ЗАК Phalanx (~12 очередей)"),
			i18n.T("Destroyer hull: typically 2 Mk48 hits", "Корпус эсминца: обычно 2 попадания Mk48"),
		},
		Neutralize: []i18n.TranslatedText{
			i18n.T("Friendly — do not engage. Classify before shoot if the contact looks American.", "Свой — не атакуйте. Классифицируйте до выстрела, если контакт похож на американский."),
		},
		Evade: []i18n.TranslatedText{
			i18n.T("Not a threat to ownship; he prosecutes hostiles on DEFCON.", "Не угроза своему кораблю; он преследует врагов по DEFCON."),
		},
	},
}

func libraryEntryByID(id string) *libraryEntry {
	for i := range libraryCatalog {
		if libraryCatalog[i].ID == id {
			return &libraryCatalog[i]
		}
	}
	return nil
}

func libraryAllegianceLabel(a libraryAllegiance) i18n.TranslatedText {
	switch a {
	case libHostile:
		return i18n.T("HOSTILE", "ВРАЖДЕБНЫЕ")
	case libNeutral:
		return i18n.T("NEUTRAL", "НЕЙТРАЛЬНЫЕ")
	case libFriendly:
		return i18n.T("FRIENDLY", "СВОИ")
	default:
		return i18n.T("UNKNOWN", "НЕИЗВЕСТНО")
	}
}

// libraryTableRows builds non-selectable section headers + selectable platform rows.
// Within each allegiance: surface ships first, then submarines.
func libraryTableRows() []libraryTableRow {
	var out []libraryTableRow
	order := []libraryAllegiance{libHostile, libNeutral, libFriendly}
	for _, all := range order {
		var surfaces, subs []libraryEntry
		for _, e := range libraryCatalog {
			if e.Allegiance != all {
				continue
			}
			if e.Kind == world.KindSurfaceShip {
				surfaces = append(surfaces, e)
			} else if e.Kind == world.KindSubmarine {
				subs = append(subs, e)
			}
		}
		if len(surfaces)+len(subs) == 0 {
			continue
		}
		out = append(out, libraryTableRow{Header: true, Label: libraryAllegianceLabel(all)})
		for _, e := range surfaces {
			out = append(out, libraryTableRow{Label: e.Title, EntryID: e.ID})
		}
		for _, e := range subs {
			out = append(out, libraryTableRow{Label: e.Title, EntryID: e.ID})
		}
	}
	return out
}

type libraryTableRow struct {
	Header  bool
	Label   i18n.TranslatedText
	EntryID string
}
