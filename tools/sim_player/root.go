package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// workspaceRoot is the repo directory containing scenarios_generated/.
var workspaceRoot string

func initWorkspaceRoot() error {
	if root := findScenariosRootFrom(func() (string, error) { return os.Getwd() }); root != "" {
		workspaceRoot = root
		return chdirRoot(root)
	}
	if root := findScenariosRootFrom(os.Executable); root != "" {
		workspaceRoot = root
		return chdirRoot(root)
	}
	return fmt.Errorf("cannot find %s/ — run from the repo root or keep ssn688-player next to %s/", scenariosDir, scenariosDir)
}

func findScenariosRootFrom(base func() (string, error)) string {
	start, err := base()
	if err != nil || start == "" {
		return ""
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for i := 0; i < 8; i++ {
		if isScenariosDir(filepath.Join(dir, scenariosDir)) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func isScenariosDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func chdirRoot(root string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absWD, err := filepath.Abs(wd)
	if err != nil {
		return err
	}
	if absWD == absRoot {
		return nil
	}
	if err := os.Chdir(absRoot); err != nil {
		return fmt.Errorf("chdir %s: %w", absRoot, err)
	}
	fmt.Printf("workspace: %s\n", absRoot)
	return nil
}

// resolvePath makes flag paths absolute when relative; leaves absolute paths unchanged.
func resolvePath(p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	if workspaceRoot != "" {
		return filepath.Join(workspaceRoot, p)
	}
	return p
}
