package i18n

import (
	"fmt"
	"regexp"
	"strings"
)

// Common status / notification strings shown in the bottom status line.

var (
	StatusBTCastComplete = T(
		"BT cast complete — thermocline boundaries plotted.",
		"БТ завершён — границы термоклина нанесены.",
	)
	StatusBTCastInProgress = T(
		"BT cast in progress — %.0fs remaining.",
		"БТ в процессе — осталось %.0f с.",
	)
	StatusBTCastUnable = T("Unable to start BT cast.", "Не удалось начать БТ.")

	StatusTorpedoInWaterContact = T(
		"Torpedo in the water — contact %s.",
		"Торпеда в воде — контакт %s.",
	)
	StatusIncomingTorpedoContact = T(
		"Incoming torpedo! Contact %s crossing ownship track.",
		"Торпеда на нас! Контакт %s пересекает наш курс.",
	)
	StatusIncomingTorpedoOwnFish = T(
		"Incoming torpedo! Own fish %s crossing ownship track.",
		"Торпеда на нас! Своя торпеда %s пересекает наш курс.",
	)

	StatusTransmitNeedCOMM = T(
		"Unable to transmit — raise COMM mast first.",
		"Передача невозможна — сначала поднимите мачту COMM.",
	)
	StatusTransmitNeedPD = T(
		"Unable to transmit — come to periscope depth (≤60 ft).",
		"Передача невозможна — выйдите на перископную глубину (≤60 фт).",
	)
	StatusTransmitNeedObjectives = T(
		"Unable to transmit — primary objectives not complete.",
		"Передача невозможна — основные задачи не выполнены.",
	)
	StatusTransmitDone = T(
		"Mission status transmitted on COMM — END MISSION enabled.",
		"Статус миссии передан по COMM — доступно ЗАВЕРШИТЬ МИССИЮ.",
	)

	StatusPeriZoom  = T("Periscope zoom %s", "Зум перископа %s")
	StatusPeriTrain = T(
		"Periscope train %+.0f° rel / %03.0f°T",
		"Поворот перископа %+.0f° отн. / %03.0f° ист.",
	)

	StatusNoWireFish = T(
		"No wire-guided fish on selected tube.",
		"На выбранной трубе нет торпеды с проводом.",
	)
	StatusTorpedoSelfDestruct = T(
		"Torpedo self-destructed (safe abort).",
		"Торпеда самоуничтожена (безопасный сброс).",
	)
	StatusWireCutNoDestruct = T(
		"Wire cut — cannot self-destruct.",
		"Провод обрезан — самоуничтожение невозможно.",
	)
	StatusCannotChangeOrdnance = T(
		"Cannot change ordnance — tube door open or weapon away.",
		"Нельзя сменить боеприпас — крышка открыта или оружие ушло.",
	)
	StatusTubeDamagedNoFire = T(
		"Tube %d damaged — cannot fire.",
		"Труба %d повреждена — стрельба невозможна.",
	)
	StatusOpenDoorFirst = T(
		"Cannot fire — open outer door first.",
		"Стрельба невозможна — сначала откройте наружную крышку.",
	)
	StatusTubeReloading = T(
		"Tube %d reloading %s.",
		"Труба %d: зарядка %s.",
	)

	StatusTowedDamaged = T(
		"Towed array damaged — cannot deploy.",
		"Буксируемая антенна повреждена — выпуск невозможен.",
	)
	StatusDecoyEmpty  = T("DECOY magazine empty.", "Магазин DECOY пуст.")
	StatusJitterEmpty = T("JITTER magazine empty.", "Магазин JITTER пуст.")

	StatusRepairing = T("Repairing %s…", "Ремонт: %s…")

	StatusNoSaves          = T("No save files found.", "Сохранения не найдены.")
	StatusSelectSave       = T("Select a save file first.", "Сначала выберите файл сохранения.")
	StatusLoadFailed       = T("Load failed: %s", "Ошибка загрузки: %s")
	StatusSaveFailed       = T("Save failed: %s", "Ошибка сохранения: %s")
	StatusGameSaved        = T("Game saved: %s", "Игра сохранена: %s")
	StatusImportFailed     = T("Import failed: %s", "Ошибка импорта: %s")
	StatusImported         = T("Imported %s v%s", "Импортирован %s v%s")
	StatusScenarioDeleted  = T("Deleted scenario %s.", "Сценарий %s удалён.")
	StatusDeleteFailed     = T("Delete failed: %s", "Ошибка удаления: %s")
	StatusScenarioIncompat = T(
		"Scenario is incompatible with this game version.",
		"Сценарий несовместим с этой версией игры.",
	)
	StatusNoSaveForScenario = T("No save found for this scenario.", "Нет сохранения для этого сценария.")
	StatusProgressCleared   = T("Scenario progress cleared.", "Прогресс сценария сброшен.")
	StatusScenarioComplete  = T("Scenario already complete.", "Сценарий уже пройден.")
	StatusBuildMissionFail  = T("Failed to build mission.", "Не удалось собрать миссию.")

	StatusMarkerPlaced    = T("Marker %s placed", "Маркер %s поставлен")
	StatusMarkerDeleted   = T("Marker %s deleted", "Маркер %s удалён")
	StatusMarkerSelected  = T("Selected marker %s", "Выбран маркер %s")
	StatusContactSelected = T("Selected %s", "Выбран %s")

	StatusVoiceRunDepth     = T("Torpedo run depth %d feet.", "Глубина хода торпеды %d футов.")
	StatusVoiceSpeedHigh    = T("Torpedo speed HIGH.", "Скорость торпеды ВЫСОКАЯ.")
	StatusVoiceSpeedLow     = T("Torpedo speed LOW.", "Скорость торпеды НИЗКАЯ.")
	StatusVoiceMakeDepth    = T("Make depth %d feet.", "Занять глубину %d футов.")
	StatusVoiceComeLeft     = T("Come left to %d.", "Лево руля на %d.")
	StatusVoiceComeRight    = T("Come right to %d.", "Право руля на %d.")
	StatusVoiceHoldDepth    = T("Hold depth %d feet.", "Держать глубину %d футов.")
	StatusVoiceSurface      = T("Surface the ship.", "Всплытие.")
	StatusVoiceTowedHeld    = T("Towed array held at %d percent.", "Буксируемая антенна удержана на %d%%.")
	StatusVoiceClassified   = T("Contact %s classified as %s.", "Контакт %s классифицирован как %s.")
	StatusVoiceTorpedoAway  = T("Torpedo away, tube %d.", "Торпеда ушла, труба %d.")
	StatusVoiceHarpoonAway  = T("Harpoon away, tube %d.", "Harpoon ушёл, труба %d.")
	StatusVoiceOuterOpen    = T("Outer door open, tube %d.", "Наружная крышка открыта, труба %d.")
	StatusVoiceSelfDestruct = T("Torpedo self-destruct.", "Самоуничтожение торпеды.")

	StatusAutoRetractBoth = T(
		"Masts and towed array auto-stowed to prevent damage.",
		"Мачты и буксируемая антенна убраны автоматически во избежание повреждений.",
	)
	StatusAutoRetractTowed = T(
		"Towed array auto-retract to prevent cable damage.",
		"Буксируемая антенна убрана автоматически во избежание повреждения кабеля.",
	)
	StatusAutoLowerMasts = T(
		"Masts auto-lowering to prevent damage.",
		"Мачты автоматически опускаются во избежание повреждений.",
	)
	StatusRBUBarrage = T("RBU barrage inbound.", "Залп РБУ на подходе.")
)

// Exact English → bilingual map for messages emitted by sim/acoustics/world.
var statusExact = map[string]TranslatedText{
	"Raising ESM mast.":                    T("Raising ESM mast.", "Поднимаю мачту ESM."),
	"Lowering ESM mast.":                   T("Lowering ESM mast.", "Опускаю мачту ESM."),
	"Raising COMM mast.":                   T("Raising COMM mast.", "Поднимаю мачту COMM."),
	"Lowering COMM mast.":                  T("Lowering COMM mast.", "Опускаю мачту COMM."),
	"Raising periscope.":                   T("Raising periscope.", "Поднимаю перископ."),
	"Lowering periscope.":                  T("Lowering periscope.", "Опускаю перископ."),
	"ESM mast destroyed — beyond repair.":  T("ESM mast destroyed — beyond repair.", "Мачта ESM уничтожена — восстановлению не подлежит."),
	"COMM mast destroyed — beyond repair.": T("COMM mast destroyed — beyond repair.", "Мачта COMM уничтожена — восстановлению не подлежит."),
	"Periscope destroyed — beyond repair.": T("Periscope destroyed — beyond repair.", "Перископ уничтожен — восстановлению не подлежит."),
	"No ownship.":                          T("No ownship.", "Нет своего корабля."),
	"Invalid system.":                      T("Invalid system.", "Неверная система."),
	"System already at full efficiency.":   T("System already at full efficiency.", "Система уже на полной эффективности."),
	"Torpedo launch detected (hostile)":    T("Torpedo launch detected (hostile)", "Обнаружен пуск торпеды (враг)"),
	"Deploy towed array.":                  T("Deploy towed array.", "Выпускаю буксируемую антенну."),
	"Retract towed array.":                 T("Retract towed array.", "Убираю буксируемую антенну."),
	"Active sonar online.":                 VoiceActiveOnline,
	"Active sonar standby.":                VoiceActiveStandby,
	"Layer survey complete.":               VoiceLayerSurveyComplete,
	"Launching bathythermograph.":          VoiceBTLaunch,
	"Torpedo in the water.":                VoiceTorpedoInWater,
	"Incomming torpedo!":                   VoiceIncomingTorpedo,
	"Incoming torpedo!":                    VoiceIncomingTorpedo,
	"Torpedo impact.":                      T("Torpedo impact.", "Удар торпеды."),
}

var (
	reTooDeepESM  = regexp.MustCompile(`^Too deep — ESM mast requires ≤([\d.]+) ft \(periscope depth\)\.$`)
	reTooFastESM  = regexp.MustCompile(`^Too fast — ESM mast requires ≤([\d.]+) kn\.$`)
	reTooDeepCOMM = regexp.MustCompile(`^Too deep — COMM mast requires ≤([\d.]+) ft \(periscope depth\)\.$`)
	reTooFastCOMM = regexp.MustCompile(`^Too fast — COMM mast requires ≤([\d.]+) kn\.$`)
	reTooDeepPeri = regexp.MustCompile(`^Too deep — periscope requires ≤([\d.]+) ft\.$`)
	reTooFastPeri = regexp.MustCompile(`^Too fast — periscope requires ≤([\d.]+) kn\.$`)
	reAlreadyRep  = regexp.MustCompile(`^Already repairing (.+)\.$`)
	reBeyondRep   = regexp.MustCompile(`^(.+) destroyed beyond repair\.$`)
	reOwnShipHit  = regexp.MustCompile(`^OWN SHIP HIT`)
	reOwnShipCrit = regexp.MustCompile(`^OWN SHIP CRITICAL`)
	rePlayerLost  = regexp.MustCompile(`^PLAYER SUBMARINE (FATAL DAMAGE|LOST)`)
	reAutoRetract = regexp.MustCompile(`^AUTO-RETRACT`)
	reRBU         = regexp.MustCompile(`RBU barrage`)
	reContactID   = regexp.MustCompile(`^Contact (.+) identified: (.+)$`)
	reDamagePct   = regexp.MustCompile(`^(.+): (.+) ([\d.]+)% → ([\d.]+)%$`)
	reTubeStatus  = regexp.MustCompile(`^Tube (\d+) (.+)$`)
)

// LocalizeRuntimeMessage translates known English sim/UI status lines for lang.
// Unknown messages are returned unchanged (keeps debug / new events readable).
func LocalizeRuntimeMessage(msg, lang string) string {
	if msg == "" {
		return ""
	}
	lang = NormalizeLang(lang)
	if lang == LangEN {
		// Still normalize legacy typo for display consistency.
		if strings.HasPrefix(msg, "Incomming torpedo") {
			return strings.Replace(msg, "Incomming torpedo", "Incoming torpedo", 1)
		}
		return msg
	}
	if tt, ok := statusExact[msg]; ok {
		return tt.GetText(lang)
	}
	if m := reTooDeepESM.FindStringSubmatch(msg); m != nil {
		return fmt.Sprintf("Слишком глубоко — мачта ESM требует ≤%s фт (перископная).", m[1])
	}
	if m := reTooFastESM.FindStringSubmatch(msg); m != nil {
		return fmt.Sprintf("Слишком быстро — мачта ESM требует ≤%s уз.", m[1])
	}
	if m := reTooDeepCOMM.FindStringSubmatch(msg); m != nil {
		return fmt.Sprintf("Слишком глубоко — мачта COMM требует ≤%s фт (перископная).", m[1])
	}
	if m := reTooFastCOMM.FindStringSubmatch(msg); m != nil {
		return fmt.Sprintf("Слишком быстро — мачта COMM требует ≤%s уз.", m[1])
	}
	if m := reTooDeepPeri.FindStringSubmatch(msg); m != nil {
		return fmt.Sprintf("Слишком глубоко — перископ требует ≤%s фт.", m[1])
	}
	if m := reTooFastPeri.FindStringSubmatch(msg); m != nil {
		return fmt.Sprintf("Слишком быстро — перископ требует ≤%s уз.", m[1])
	}
	if m := reAlreadyRep.FindStringSubmatch(msg); m != nil {
		return fmt.Sprintf("Уже идёт ремонт: %s.", LocalizeSystemName(m[1], lang))
	}
	if m := reBeyondRep.FindStringSubmatch(msg); m != nil {
		return fmt.Sprintf("%s уничтожена — восстановлению не подлежит.", LocalizeSystemName(m[1], lang))
	}
	if m := reContactID.FindStringSubmatch(msg); m != nil {
		return fmt.Sprintf("Контакт %s опознан: %s", m[1], m[2])
	}
	if m := reDamagePct.FindStringSubmatch(msg); m != nil {
		return fmt.Sprintf("%s: %s %s%% → %s%%", m[1], LocalizeSystemName(m[2], lang), m[3], m[4])
	}
	if reOwnShipHit.MatchString(msg) {
		return strings.Replace(msg, "OWN SHIP HIT", "ПОПАДАНИЕ ПО СВОЕМУ КОРАБЛЮ", 1)
	}
	if reOwnShipCrit.MatchString(msg) {
		return strings.Replace(msg, "OWN SHIP CRITICAL", "КРИТИЧЕСКОЕ ПОВРЕЖДЕНИЕ", 1)
	}
	if rePlayerLost.MatchString(msg) {
		out := strings.Replace(msg, "PLAYER SUBMARINE FATAL DAMAGE", "СВОЯ ПЛ: ФАТАЛЬНЫЕ ПОВРЕЖДЕНИЯ", 1)
		out = strings.Replace(out, "PLAYER SUBMARINE LOST", "СВОЯ ПЛ ПОТЕРЯНА", 1)
		return out
	}
	if reAutoRetract.MatchString(msg) {
		switch {
		case strings.Contains(msg, "towed array") && strings.Contains(msg, "mast"):
			return StatusAutoRetractBoth.GetText(lang)
		case strings.Contains(msg, "towed array"):
			return StatusAutoRetractTowed.GetText(lang)
		default:
			return StatusAutoLowerMasts.GetText(lang)
		}
	}
	if reRBU.MatchString(msg) {
		return StatusRBUBarrage.GetText(lang)
	}
	if strings.HasPrefix(msg, "Torpedo struck bottom") {
		return "Торпеда ударилась о дно."
	}
	if strings.HasPrefix(msg, "MISSION FAILED") {
		return strings.Replace(msg, "MISSION FAILED", UIStatusMissionFailedBare.GetText(lang), 1)
	}
	if strings.HasPrefix(msg, "MISSION COMPLETE") {
		return UIStatusMissionComplete.GetText(lang)
	}
	return msg
}

// LocalizeSystemName maps English SystemName() labels to the active language.
func LocalizeSystemName(en, lang string) string {
	lang = NormalizeLang(lang)
	if lang == LangEN {
		return en
	}
	switch en {
	case "Hull Array":
		return "Корпусной массив"
	case "Towed Array":
		return "Буксируемая антенна"
	case "Active Sonar":
		return "Активный сонар"
	case "Tube 1":
		return "Труба 1"
	case "Tube 2":
		return "Труба 2"
	case "Tube 3":
		return "Труба 3"
	case "Tube 4":
		return "Труба 4"
	case "Depth Control":
		return "Управление глубиной"
	case "Steering":
		return "Управление курсом"
	case "Propulsion":
		return "Главная установка"
	case "Pressure Hull":
		return "Прочный корпус"
	case "ESM Mast":
		return "Мачта ESM"
	case "COMM Mast":
		return "Мачта COMM"
	case "Periscope":
		return "Перископ"
	case "Unknown":
		return "Неизвестно"
	default:
		return en
	}
}
