package i18n

// Screen chrome: menu windows + in-game titles/buttons/labels.
// Prefer short RU so button widths stay close to EN (abbrev with period when needed).

var (
	// --- Common ---
	UIBack   = T("BACK", "НАЗАД")
	UIYes    = T("YES", "ДА")
	UINo     = T("NO", "НЕТ")
	UICancel = T("CANCEL", "ОТМЕНА")

	// --- Scenario list / brief ---
	UISelectScenario       = T("SELECT SCENARIO", "ВЫБОР СЦЕНАРИЯ")
	UINoArt                = T("NO ART", "НЕТ ОБЛ.")
	UIContinueScenario     = T("CONTINUE SCENARIO", "ПРОДОЛЖИТЬ")
	UIRestartScenario      = T("RESTART SCENARIO", "СБРОС")
	UIOpenScenario         = T("OPEN SCENARIO", "ОТКРЫТЬ")
	UIDeleteScenario       = T("DELETE", "УДАЛИТЬ")
	UINextMission          = T("NEXT MISSION", "СЛЕД. МИССИЯ")
	UIStartMission         = T("START MISSION", "СТАРТ МИССИИ")
	UIScenarioComplete     = T("SCENARIO COMPLETE", "СЦЕНАРИЙ ПРОЙДЕН")
	UIIncompatibleBadge    = T(" — INCOMPATIBLE", " — НЕСОВМЕСТ.")
	UIDoneBadge            = T(" — DONE", " — ГОТОВО")
	UIScenarioIncompatBody = T(
		"This scenario is incompatible with this game version.",
		"Сценарий несовместим с этой версией игры.",
	)
	UIConfirmRestartTitle = T("RESTART SCENARIO", "СБРОС СЦЕНАРИЯ")
	UIConfirmRestartBody  = T(
		"Delete all saves for this scenario and start the campaign from the first mission?",
		"Удалить все сейвы сценария и начать кампанию с первой миссии?",
	)
	UIConfirmDeleteTitle = T("DELETE SCENARIO", "УДАЛИТЬ СЦЕНАРИЙ")
	UIConfirmDeleteBody  = T(
		"Remove this imported scenario and all of its saves? Bundled scenarios cannot be deleted.",
		"Удалить импортированный сценарий и все его сохранения? Встроенные сценарии удалить нельзя.",
	)

	// --- Load game ---
	UILoadGameTitle = T("LOAD GAME", "ЗАГРУЗИТЬ ИГРУ")
	UILoad          = T("LOAD", "ЗАГР.")
	UINoSavesFound  = T("No save files found.", "Сохранений нет.")

	// --- In-game header ---
	UISave            = T("SAVE", "СОХР.")
	UIEndMission      = T("END MISSION", "ЗАВЕРШ.")
	UIExit            = T("EXIT", "ВЫХОД")
	UIConfirmEndTitle = T("END MISSION", "ЗАВЕРШИТЬ МИССИЮ")
	UIConfirmEndBody  = T(
		"End the current mission? Progress will be saved automatically and you will return to the scenario briefing.",
		"Завершить миссию? Прогресс сохранится, возврат к брифингу сценария.",
	)
	UIConfirmExitTitle = T("EXIT TO MENU", "ВЫХОД В МЕНЮ")
	UIConfirmExitBody  = T(
		"Leave the current mission and return to the main menu? Unsaved progress since the last save will be lost.",
		"Выйти в меню? Несохранённый прогресс с последнего сейва будет потерян.",
	)

	// --- Overlays / header strip ---
	UIPaused         = T("PAUSED", "ПАУЗА")
	UIOwnshipSinking = T("OWN SHIP HIT - SINKING", "ПОПАДАНИЕ — ТОНЕМ")
	UIOwnshipLost    = T("OWN SHIP LOST", "СВОЯ ПЛ ПОТЕРЯНА")
	UIHeaderTime     = T("TIME", "ВРЕМЯ")
	UIHeaderSpeed    = T("SPEED", "СКОР.")
	UIHeaderScreen   = T("SCREEN", "ЭКРАН")

	// --- Screen titles ---
	UITitleWaterfall = T("BEARING WATERFALL", "ВОДОПАД ПЕЛЕНГОВ")
	UITitleActive    = T("ACTIVE SONAR", "АКТИВНЫЙ СОНАР")
	UITitleSpectrum  = T("SPECTRUM ANALYZER", "СПЕКТР-АНАЛИЗАТОР")
	UITitleWeps      = T("FIRE CONTROL — WEAPONS", "УПРАВЛЕНИЕ ОГНЁМ")
	UITitleHelm      = T("MANEUVERING ROOM — HELM", "РУБКА — РУЛЬ")
	UITitleMast      = T("MAST — ESM / COMM / SCOPE", "МАЧТЫ — ESM / COMM / ПЕРИСКОП")
	UITitleDamage    = T("DAMAGE CONTROL", "ПОВРЕЖДЕНИЯ")
	UITitlePlot      = T("TACTICAL PLOT", "ТАКТ. ПЛАНШЕТ")
	UILand           = T("LAND", "СУША")

	// --- Shared table / status bits ---
	UIColID         = T("ID", "ID")
	UIColBRG        = T("BRG", "ПЕЛ")
	UIColBRGDeg     = T("BRG°", "ПЕЛ°")
	UIColRNG        = T("RNG kyd", "ДЛН kyd")
	UIColClass      = T("CLASS", "КЛАСС")
	UIColSource     = T("SOURCE", "ИСТОЧН.")
	UISelected      = T("SELECTED", "ВЫБРАН")
	UIArray         = T("ARRAY", "АНТЕННА")
	UILayer         = T("LAYER", "СЛОЙ")
	UIDamagedPct    = T("DAMAGED (%.0f%%)", "ПОВРЕЖД. (%.0f%%)")
	UINoData        = T("NO DATA", "НЕТ ДАННЫХ")
	UIDamagedNoData = T("DAMAGED — NO DATA", "ПОВРЕЖД. — НЕТ ДАННЫХ")

	// --- PASSIVE ---
	UIHull          = T("HULL", "КОРПУС")
	UITowed         = T("TOWED", "БУКС.")
	UIDeploy        = T("DEPLOY", "ВЫПУСК")
	UIRetract       = T("RETRACT", "УБОРКА")
	UIShipBand      = T("SHIP", "КОРАБ.")
	UITorpBand      = T("TORP", "ТОРП.")
	UIPassiveArray  = T("PASSIVE ARRAY", "ПАССИВ. АНТЕННА")
	UIListenBand    = T("LISTEN BAND", "ПОЛОСА")
	UITowedArray    = T("TOWED ARRAY", "БУКС. АНТЕННА")
	UISelfNoise     = T("SELF-NOISE", "СОБСТВ. ШУМ")
	UIQuiet         = T("QUIET", "ТИХО")
	UIDeafening     = T("DEAFENING", "ОГЛУШ.")
	UIFlowNoise     = T("FLOW NOISE", "ШУМ ПОТОКА")
	UIRising        = T("RISING", "РАСТЁТ")
	UISonarBlind    = T("SONAR BLIND", "СОНАР ГЛУХ")
	UIDeploying     = T("DEPLOYING", "ВЫПУСК")
	UIRetracting    = T("RETRACTING", "УБОРКА")
	UIHeld          = T("HELD", "УДЕРЖ.")
	UIDeployed      = T("DEPLOYED", "ВЫПУЩ.")
	UIStowed        = T("STOWED", "УБРАНА")
	UICableStress   = T("CABLE STRESS", "НАГР. КАБЕЛЯ")
	UITipHullArray  = T("Hull passive array", "Корпусной пассивный массив")
	UITipTowedArray = T("Towed array", "Буксируемая антенна")
	UITipDeploy     = T("Deploy towed array", "Выпустить буксируемую антенну")
	UITipRetract    = T("Retract towed array", "Убрать буксируемую антенну")
	UITipShipBand   = T("Listen for ships / submarines", "Полоса: корабли / ПЛ")
	UITipTorpBand   = T("Listen for torpedoes", "Полоса: торпеды")

	// --- ACTIVE ---
	UIStandby       = T("STANDBY", "ДЕЖУР.")
	UIActiveOn      = T("ACTIVE ON", "АКТИВ ВКЛ")
	UIPingNow       = T("PING NOW", "ПИНГ")
	UITransmitReady = T("TRANSMIT READY", "ГОТОВ К ИЗЛ.")
	UIActiveTX      = T("ACTIVE TX", "АКТИВ TX")
	UIAutoPing      = T("AUTO PING", "АВТОПИНГ")
	UIPower         = T("POWER", "МОЩН.")
	UIRange         = T("RANGE", "ДАЛЬН.")
	UIOwn           = T("OWN", "СВОЙ")
	UIActiveDamaged = T("ACTIVE SONAR DAMAGED — NO TRANSMIT", "АКТИВ ПОВРЕЖД. — НЕТ ИЗЛ.")
	UIContactCol    = T("CONTACT", "КОНТ.")

	// --- SPECTRUM ---
	UIClassify          = T("CLASSIFY", "КЛАССИФ.")
	UINoTonalLock       = T("NO TONAL LOCK", "НЕТ ТОНАЛ. ЗАХВАТА")
	UIRefUnavailable    = T("REFERENCE UNAVAILABLE — insufficient harmonics", "ЭТАЛОН НЕДОСТУПЕН — мало гармоник")
	UIProfile           = T("PROFILE:", "ПРОФИЛЬ:")
	UIInsufficientTonal = T("INSUFFICIENT TONAL DATA", "НЕДОСТАТОЧНО ТОНАЛОВ")
	UIHFTorpedoSet      = T("HF — TORPEDO CLASS SET", "ВЧ — НАБОР ТОРПЕД")
	UILFPlatformSet     = T("LF/MF — PLATFORM CLASS SET", "НЧ/СЧ — НАБОР ПЛАТФОРМ")
	UIFullLibrary       = T("FULL LIBRARY", "ПОЛНАЯ БИБЛИОТЕКА")
	UIBearingMixFull    = T("BEARING MIX — FULL LIBRARY", "СМЕШЕНИЕ ПЕЛЕНГА — ПОЛНАЯ БИБЛ.")
	UIContactSignalAt   = T("CONTACT SIGNAL @", "СИГНАЛ КОНТАКТА @")
	UIManualBearing     = T("MANUAL BEARING", "РУЧН. ПЕЛЕНГ")
	UITowedDamagedData  = T("TOWED ARRAY DAMAGED — NO DATA", "БУКС. АНТЕННА ПОВРЕЖД. — НЕТ ДАННЫХ")
	UIHullDamagedData   = T("HULL ARRAY DAMAGED — NO DATA", "КОРП. АНТЕННА ПОВРЕЖД. — НЕТ ДАННЫХ")
	UITipPrevLib        = T("Previous library target", "Пред. эталон")
	UITipNextLib        = T("Next library target", "След. эталон")

	// --- LIBRARY leftovers ---
	UISelectPlatform = T("Select a platform type", "Выберите тип платформы")
	UINoImage        = T("NO IMAGE", "НЕТ ФОТО")

	// --- WEPS ---
	UIOpen         = T("OPEN", "ОТКР.")
	UIClose        = T("CLOSE", "ЗАКР.")
	UIFire         = T("FIRE", "ОГОНЬ")
	UISeekOff      = T("SEEK OFF", "ГСН ВЫКЛ")
	UISeekOn       = T("SEEK ON", "ГСН ВКЛ")
	UISpeedLowBtn  = T("LOW", "НИЗК.")
	UISpeedHighBtn = T("HIGH", "ВЫСОК.")
	UICutWire      = T("CUT", "ОБРЕЗ")
	UISD           = T("S/D", "С/У")
	UIDecoy        = T("DECOY", "DECOY")
	UIJitter       = T("JITTER", "JITTER")
	UICountermeas  = T("COUNTERMEASURES", "ПОМЕХИ")
	UINextShotPrep = T("NEXT SHOT PREP", "ПОДГОТ. ВЫСТРЕЛА")
	UIWireGuide    = T("WIRE GUIDE", "ПРОВОД")
	UIHarpoonPrep  = T("HARPOON PREP", "HARPOON")
	UIGyro         = T("GYRO", "КУРС")
	UIDep          = T("DEP", "ГЛУБ.")
	UIRecognized   = T("RECOGNIZED TARGETS", "ОПОЗНАННЫЕ ЦЕЛИ")
	UITacticalMap  = T("TACTICAL MAP", "ТАКТ. КАРТА")
	UIEmpty        = T("EMPTY", "ПУСТО")
	UIReloading    = T("RELOADING", "ЗАРЯДКА")
	UIReloadSecs   = T("RELOAD %ds", "ЗАРЯД. %dс")
	UIBeamWide     = T("BEAM WIDE", "ЛУЧ ШИР.")
	UIBeamNarrow   = T("BEAM NARROW", "ЛУЧ УЗК.")
	UITipOpenDoor  = T("Open outer door", "Открыть наружную крышку")
	UITipCloseDoor = T("Close outer door", "Закрыть наружную крышку")
	UITipFire      = T("Fire selected tube", "Выстрел из выбранной трубы")
	UITipCutWire   = T("Cut wire", "Обрезать провод")
	UITipSelfDestr = T("Self-destruct torpedo", "Самоуничтожение торпеды")
	UITipOrdnance  = T("Select ordnance", "Выбор боеприпаса")

	// --- CM panel ---
	UIADCLeft       = T("ADC LEFT", "ADC ОСТ.")
	UIJitterLeft    = T("JITTER LEFT", "JITTER ОСТ.")
	UIMagazineEmpty = T("MAGAZINE EMPTY", "МАГАЗИН ПУСТ")
	UICMSubtitle    = T("Expendable soft-kill — deploy toward the threat", "Расходуемые помехи — к угрозе")

	// --- HELM ---
	UIFlank        = T("FLANK", "ПОЛН.")
	UIFull         = T("FULL", "ПОЛН.")
	UITwoThirds    = T("2/3", "2/3")
	UIOneThird     = T("1/3", "1/3")
	UIOneThirdAst  = T("1/3 AST", "1/3 ЗХ")
	UITwoThirdsAst = T("2/3 AST", "2/3 ЗХ")
	UIFullAst      = T("FULL AST", "ПОЛН. ЗХ")
	UIPort         = T("◄ PORT", "◄ ЛЕВ")
	UIStbd         = T("STBD ►", "ПРВ ►")
	UISurface      = T("SURFACE", "ВСПЛЫТИЕ")
	UIPeriscope    = T("PERISCOPE", "ПГ")
	UIBTCast       = T("BT CAST", "БТ")
	UIBottomFt     = T("BOTTOM %.0f FT", "ДНО %.0f ФТ")
	UIEngineOrder  = T("ENGINE ORDER", "ТЕЛЕГРАФ")
	UIClickCompass = T("CLICK COMPASS TO ORDER COURSE", "КЛИК ПО КОМПАСУ — КУРС")
	UIClickScale   = T("CLICK SCALE TO ORDER DEPTH", "КЛИК ПО ШКАЛЕ — ГЛУБИНА")
	UIBTLayers     = T("BT / LAYERS", "БТ / СЛОИ")
	UIKeel         = T("KEEL", "КИЛЬ")
	UIFt           = T("FT", "ФТ")
	UITipFlank     = T("Flank ahead — maximum speed", "Полный вперёд — макс. ход")
	UITipFull      = T("Full ahead", "Полный вперёд")
	UITip23        = T("Two-thirds ahead", "Два третьих вперёд")
	UITip13        = T("One-third ahead", "Одна треть вперёд")
	UITip13Ast     = T("One-third astern", "Одна треть назад")
	UITip23Ast     = T("Two-thirds astern", "Два третьих назад")
	UITipFullAst   = T("Full astern", "Полный назад")
	UITipSurface   = T("Surface — order zero depth", "Всплытие — глубина 0")
	UITipPeriscope = T("Periscope depth — order 60 feet", "Перископная — 60 футов")
	UITipBTCast    = T("Launch SSXBT — survey thermocline", "Запуск БТ — съёмка термоклина")

	// --- MAST ---
	UIRaiseESM     = T("RAISE ESM", "ПОДН. ESM")
	UILowerESM     = T("LOWER ESM", "ОПУСК. ESM")
	UIRaise        = T("RAISE", "ПОДН.")
	UILower        = T("LOWER", "ОПУСК.")
	UIReport       = T("REPORT", "ДОКЛАД")
	UIZoomIn       = T("ZOOM +", "ЗУМ +")
	UIZoomOut      = T("ZOOM −", "ЗУМ −")
	UISeaState     = T("SEA STATE:", "МОРЕ:")
	UIESMSuite     = T("ESM SUITE", "ESM")
	UIDestroyed    = T("DESTROYED", "УНИЧТ.")
	UIRaisedRecv   = T("RAISED — RECEIVING", "ПОДНЯТА — ПРИЁМ")
	UIRaising      = T("RAISING", "ПОДЪЁМ")
	UILowering     = T("LOWERING", "ОПУСК")
	UIRaisedIR     = T("RAISED — IR SENSOR", "ПОДНЯТ — ИК")
	UIZoom         = T("ZOOM", "ЗУМ")
	UITrain        = T("TRAIN", "ПОВОР.")
	UIBearing      = T("BEARING", "ПЕЛЕНГ")
	UIOwnMastIllum = T("OWN MAST ILLUMINATION (hostile/neutral search radars)", "ОБЛУЧЕНИЕ СВОЕЙ МАЧТЫ (РЛС поиска)")
	UISafe         = T("SAFE", "БЕЗОП.")
	UIDetectable   = T("DETECTABLE", "ОБНАР.")
	UIPainted      = T("PAINTED", "ПОДСВЕТ")
	UIMastDown     = T("MAST DOWN", "МАЧТА УБРАНА")
	UICOMContacts  = T("COMM / CONTACTS", "COMM / КОНТАКТЫ")
	UICOMMast      = T("COMM MAST:", "МАЧТА COMM:")
	UINoTraffic    = T("No traffic", "Нет сообщений")
	UIRFLog        = T("RF INTERCEPT LOG", "ЖУРНАЛ RF")
	UIEQIPHint     = T("EQIP = radar set, not hull ID", "EQIP = РЛС, не корпус")
	UIColSRC       = T("SRC", "ИСТ")
	UIColEQIP      = T("EQIP", "EQIP")
	UIColRF        = T("RF%", "RF%")
	UIColLast      = T("LAST", "ПОСЛ.")
	UIColSgnl      = T("SGNL", "СИГН.")
	UIIRSensor     = T("IR SENSOR", "ИК ДАТЧИК")
	UIScopeStowed  = T("SCOPE STOWED", "ПЕРИСКОП УБРАН")
	UINoOptic      = T("NO OPTIC", "НЕТ ОПТИКИ")
	UIOpticMotion  = T("OPTIC MOTION", "ОПТИКА В ДВИЖ.")

	// --- DC ---
	UIRepair          = T("REPAIR", "РЕМОНТ")
	UIRepairing       = T("REPAIRING", "РЕМОНТ")
	UIOK              = T("OK", "НОРМА")
	UINA              = T("N/A", "Н/Д")
	UISystem          = T("SYSTEM", "СИСТЕМА")
	UIStatusCol       = T("STATUS", "СТАТУС")
	UIEfficiency      = T("EFFICIENCY", "ЭФФ.")
	UIDamageHint      = T("Repair one system at a time. Critical systems first.", "Ремонт по одной системе. Сначала критические.")
	UIDepthLost       = T("DEPTH CONTROL LOST — ordered depth ignored", "НЕТ УПР. ГЛУБИНОЙ — заказ игнорируется")
	UIRudderJam       = T("RUDDER JAMMED — ordered course ignored", "РУЛЬ ЗАКЛИНЕН — заказ игнорируется")
	UIStatusNominal   = T("NOMINAL", "НОРМА")
	UIStatusDegraded  = T("DEGRADED", "СНИЖ.")
	UIStatusCritical  = T("CRITICAL", "КРИТ.")
	UIStatusDestroyed = T("DESTROYED", "УНИЧТ.")
	UITipRepair       = T("Begin repair", "Начать ремонт")
	UITipRepairing    = T("Repair in progress", "Идёт ремонт")

	// --- PLOT ---
	UIFit       = T("FIT", "ВМЕСТ.")
	UISplash    = T("SPLASH", "ВСПЛЕСК")
	UITipFitMap = T("Fit map to contacts", "Вписать карту по контактам")

	UIContactLog   = T("CONTACT LOG", "ЖУРНАЛ КОНТАКТОВ")
	UINow          = T("NOW", "СЕЙЧ.")
	UIMastLabel    = T("MAST:", "МАЧТА:")
	UIScopeLabel   = T("SCOPE:", "ПЕРИСКОП:")
	UILock         = T("LOCK", "ЗАХВ.")
	UIPeriLeft     = T("< LEFT", "< ЛЕВ")
	UIPeriRight    = T("RIGHT >", "ПРВ >")
	UITipPeriLeft  = T("Train periscope left %.0f°", "Поворот перископа влево %.0f°")
	UITipPeriRight = T("Train periscope right %.0f°", "Поворот перископа вправо %.0f°")

	UIWaterLayer     = T("WATER LAYER:", "СЛОЙ ВОДЫ:")
	UILayerOnFile    = T("LAYER PROFILE: ON FILE", "ПРОФИЛЬ СЛОЯ: ЕСТЬ")
	UIBTCastRemain   = T("BT CAST: %.0fs remaining", "БТ: осталось %.0f с")
	UILayerUnknown   = T("LAYER PROFILE: UNKNOWN — launch BT CAST", "ПРОФИЛЬ СЛОЯ: НЕТ — запустите БТ")
	UICavitationRisk = T("CAVITATION RISK %.0f%%", "РИСК КАВИТАЦИИ %.0f%%")
	UIPropDestroyed  = T("PROPULSION DESTROYED — NO THRUST", "ГЭУ УНИЧТОЖЕНА — НЕТ ХОДА")
	UIPropDegraded   = T("PROPULSION DEGRADED — MAX %.0f KTS", "ГЭУ СНИЖЕНА — МАКС %.0f УЗ")
	UILibFooter      = T(
		"select platform or classified contact  |  scroll tables / detail",
		"выберите платформу или контакт  |  прокрутка таблиц / деталей",
	)

	// --- Weather (SeaState display) ---
	UIWeatherCalm  = T("Calm", "Штиль")
	UIWeatherLight = T("Light", "Слабое")
	UIWeatherMod   = T("Moderate", "Умеренное")
	UIWeatherRough = T("Rough", "Сильное")
	UIWeatherHigh  = T("High", "Жёсткое")
)

// LocalizeSystemStatus maps world.SystemStatusLabel English tokens.
func LocalizeSystemStatus(en, lang string) string {
	lang = NormalizeLang(lang)
	switch en {
	case "NOMINAL":
		return UIStatusNominal.GetText(lang)
	case "DEGRADED":
		return UIStatusDegraded.GetText(lang)
	case "CRITICAL":
		return UIStatusCritical.GetText(lang)
	case "DESTROYED":
		return UIStatusDestroyed.GetText(lang)
	default:
		return en
	}
}

// LocalizeLayerName maps Environment layer ids (mixed/thermocline/deep/unknown).
func LocalizeLayerName(name, lang string) string {
	lang = NormalizeLang(lang)
	if lang == LangEN {
		return name
	}
	switch name {
	case "mixed":
		return "смешанный"
	case "thermocline":
		return "термоклин"
	case "deep":
		return "глубинный"
	case "unknown":
		return "неизвестно"
	default:
		return name
	}
}
