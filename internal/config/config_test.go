package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetDataRootOverridesPaths(t *testing.T) {
	t.Cleanup(func() { SetDataRoot("") })

	root := t.TempDir()
	SetDataRoot(root)

	wantBase := filepath.Join(root, "ssn688")
	got, err := appDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != wantBase {
		t.Fatalf("appDataDir=%q want %q", got, wantBase)
	}

	saves, err := SavesDir()
	if err != nil {
		t.Fatal(err)
	}
	if saves != filepath.Join(wantBase, "saves") {
		t.Fatalf("SavesDir=%q", saves)
	}

	scen, err := ScenariosDir()
	if err != nil {
		t.Fatal(err)
	}
	if scen != filepath.Join(wantBase, "scenarios") {
		t.Fatalf("ScenariosDir=%q", scen)
	}

	imp, err := ScenarioImportDir()
	if err != nil {
		t.Fatal(err)
	}
	if imp != filepath.Join(wantBase, "import") {
		t.Fatalf("ScenarioImportDir=%q", imp)
	}

	settings, err := SettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if settings != filepath.Join(wantBase, "settings.json") {
		t.Fatalf("SettingsPath=%q", settings)
	}

	if DataRoot() != filepath.Clean(root) {
		t.Fatalf("DataRoot=%q", DataRoot())
	}
}

func TestSaveCreatesUnderDataRoot(t *testing.T) {
	t.Cleanup(func() { SetDataRoot("") })
	root := t.TempDir()
	SetDataRoot(root)

	s := DefaultSettings()
	s.Debug = false
	if err := Save(s); err != nil {
		t.Fatal(err)
	}
	path, err := SettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Debug {
		t.Fatal("expected Debug=false after round-trip")
	}
}
