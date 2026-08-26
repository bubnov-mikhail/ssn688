package i18n

import (
	"strings"
	"testing"
)

func TestGetTextFallback(t *testing.T) {
	tt := T("Hello", "Привет")
	if got := tt.GetText(LangEN); got != "Hello" {
		t.Fatalf("en: %q", got)
	}
	if got := tt.GetText(LangRU); got != "Привет" {
		t.Fatalf("ru: %q", got)
	}
	if got := tt.GetText("de"); got != "Hello" {
		t.Fatalf("unknown lang fallback: %q", got)
	}
	onlyEN := TranslatedText{LangEN: "Only"}
	if got := onlyEN.GetText(LangRU); got != "Only" {
		t.Fatalf("missing ru → en: %q", got)
	}
}

func TestGetWavFallback(t *testing.T) {
	a := A("capt/comm_message")
	if got := a.GetWav(LangRU); got != "ru/capt/comm_message" {
		t.Fatalf("ru wav path: %q", got)
	}
	if got := a.GetWav(LangEN); got != "capt/comm_message" {
		t.Fatalf("en wav: %q", got)
	}
	enOnly := TranslatedAudio{LangEN: "sonar/enemy_ping"}
	if got := enOnly.GetWav(LangRU); got != "sonar/enemy_ping" {
		t.Fatalf("missing ru path → en: %q", got)
	}
}

func TestLocalizeRuntimeMessageRU(t *testing.T) {
	got := LocalizeRuntimeMessage("Raising ESM mast.", LangRU)
	if got != "Поднимаю мачту ESM." {
		t.Fatalf("got %q", got)
	}
	got = LocalizeRuntimeMessage("Too deep — ESM mast requires ≤60 ft (periscope depth).", LangRU)
	if !strings.Contains(got, "Слишком глубоко") || !strings.Contains(got, "60") {
		t.Fatalf("got %q", got)
	}
	got = LocalizeRuntimeMessage("Hull Array", LangRU)
	if got != "Hull Array" {
		t.Fatalf("unknown should pass through: %q", got)
	}
	if LocalizeSystemName("Towed Array", LangRU) != "Буксируемая антенна" {
		t.Fatal(LocalizeSystemName("Towed Array", LangRU))
	}
}

func TestSetLang(t *testing.T) {
	SetLang(LangRU)
	if CurrentLang() != LangRU {
		t.Fatal(CurrentLang())
	}
	SetLang("nope")
	if CurrentLang() != LangEN {
		t.Fatal(CurrentLang())
	}
}
