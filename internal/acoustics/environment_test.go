package acoustics

import "testing"

func TestLayerSurveyCompletes(t *testing.T) {
	env := DefaultEnvironment()
	if !env.StartLayerSurvey(10) {
		t.Fatal("expected survey to start")
	}
	if env.LayerSurveyProgress(10) != 0 {
		t.Fatalf("progress at start: %v", env.LayerSurveyProgress(10))
	}
	env.UpdateLayerSurvey(10 + LayerSurveyDurationSec/2)
	if env.LayerSurveyKnown {
		t.Fatal("should not be known mid-cast")
	}
	p := env.LayerSurveyProgress(10 + LayerSurveyDurationSec/2)
	if p < 0.4 || p > 0.6 {
		t.Fatalf("mid progress = %v", p)
	}
	env.UpdateLayerSurvey(10 + LayerSurveyDurationSec)
	if !env.LayerSurveyKnown {
		t.Fatal("expected survey complete")
	}
	if len(env.KnownBoundaryDepthsFt()) == 0 {
		t.Fatal("expected boundary depths after survey")
	}
}

func TestLayerSurveyCanRestart(t *testing.T) {
	env := DefaultEnvironment()
	if !env.StartLayerSurvey(0) {
		t.Fatal("start")
	}
	env.UpdateLayerSurvey(LayerSurveyDurationSec)
	if !env.LayerSurveyKnown {
		t.Fatal("expected known")
	}
	if env.LayerSurveyActive(LayerSurveyDurationSec + 1) {
		t.Fatal("should be idle after complete")
	}
	if !env.StartLayerSurvey(20) {
		t.Fatal("expected re-cast after complete")
	}
	if !env.LayerSurveyActive(20) {
		t.Fatal("expected active re-cast")
	}
}
