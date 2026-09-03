package ui

import (
	"fmt"

	"github.com/bubnov-mikhail/ssn688/internal/i18n"
)

// Lang returns the active UI/scenario language from settings.
func (a *App) Lang() string {
	if a == nil {
		return i18n.CurrentLang()
	}
	return i18n.NormalizeLang(a.Settings.Language)
}

// L resolves a TranslatedText for the active language.
func (a *App) L(t i18n.TranslatedText) string {
	return t.GetText(a.Lang())
}

// Lf formats a bilingual format string with args for the active language.
func (a *App) Lf(t i18n.TranslatedText, args ...any) string {
	return fmt.Sprintf(a.L(t), args...)
}

// Status shows a bilingual status line (resolved immediately).
func (a *App) Status(t i18n.TranslatedText) {
	a.StatusMessage = a.L(t)
}

// Statusf shows a bilingual formatted status line.
func (a *App) Statusf(t i18n.TranslatedText, args ...any) {
	a.StatusMessage = a.Lf(t, args...)
}

// StatusRaw sets a status that may still be English from the sim; localized at draw.
func (a *App) StatusRaw(msg string) {
	a.StatusMessage = msg
}

func (a *App) displayStatus() string {
	return i18n.LocalizeRuntimeMessage(a.StatusMessage, a.Lang())
}

func (a *App) syncLanguage() {
	lang := a.Lang()
	a.Settings.Language = lang
	i18n.SetLang(lang)
}
