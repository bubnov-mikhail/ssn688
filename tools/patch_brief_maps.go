// Command patch_brief_maps inlines mission brief_map PNGs from theater_previews
// into a scenario JSON document.
//
//	go run ./tools/patch_brief_maps.go -scenario scenarios_generated/taiwan_formosa_watch.json
package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	scenarioPath := flag.String("scenario", "", "path to scenario JSON")
	previewDir := flag.String("previews", "scenarios_generated/theater_previews", "brief_map PNG directory")
	bumpPatch := flag.Bool("bump", true, "increment scenario version patch")
	flag.Parse()
	if *scenarioPath == "" {
		fmt.Fprintln(os.Stderr, "usage: patch_brief_maps -scenario <path>")
		os.Exit(2)
	}

	data, err := os.ReadFile(*scenarioPath)
	if err != nil {
		panic(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		panic(err)
	}
	missions, ok := doc["missions"].([]any)
	if !ok || len(missions) == 0 {
		panic("scenario has no missions")
	}

	patched := 0
	for i, raw := range missions {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		if id == "" {
			continue
		}
		num := i + 1
		pngPath := filepath.Join(*previewDir, fmt.Sprintf("%02d_%s__brief_map.png", num, id))
		png, err := os.ReadFile(pngPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", id, err)
			continue
		}
		m["brief_map"] = map[string]any{
			"mime":     "image/png",
			"data_b64": base64.StdEncoding.EncodeToString(png),
		}
		patched++
		fmt.Printf("inlined brief_map for %s (%d bytes)\n", id, len(png))
	}
	if patched == 0 {
		panic("no brief_map PNGs patched")
	}

	if *bumpPatch {
		ver, _ := doc["version"].(string)
		parts := strings.Split(ver, ".")
		if len(parts) == 3 {
			var patch int
			fmt.Sscanf(parts[2], "%d", &patch)
			doc["version"] = fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch+1)
			fmt.Printf("version %s -> %s\n", ver, doc["version"])
		}
	}

	out, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*scenarioPath, out, 0644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s (%d missions patched)\n", *scenarioPath, patched)
}
