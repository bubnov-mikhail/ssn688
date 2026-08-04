package world

import (
	"math"
	"strings"
	"testing"
)

func TestWorldToLatLonOrigin(t *testing.T) {
	lat, lon := WorldToLatLon(0, 0)
	if math.Abs(lat-GeoOriginLatDeg) > 1e-9 || math.Abs(lon-GeoOriginLonDeg) > 1e-9 {
		t.Fatalf("origin: got %.6f, %.6f", lat, lon)
	}
}

func TestWorldToLatLonNorthEast(t *testing.T) {
	// ~1 nm north ≈ +2025 yd → ~1/60 deg latitude.
	lat, lon := WorldToLatLon(0, YardsPerNM)
	if math.Abs(lat-(GeoOriginLatDeg+1.0/60.0)) > 0.002 {
		t.Fatalf("1 nm north lat=%.6f want ~%.6f", lat, GeoOriginLatDeg+1.0/60.0)
	}
	if math.Abs(lon-GeoOriginLonDeg) > 1e-6 {
		t.Fatalf("lon should be unchanged, got %.6f", lon)
	}
	_, lonE := WorldToLatLon(YardsPerNM, 0)
	if lonE <= GeoOriginLonDeg {
		t.Fatalf("1 nm east should increase lon (less west), got %.6f", lonE)
	}
}

func TestFormatNavLatLon(t *testing.T) {
	got := FormatNavLatLon(33.30, -118.45)
	if !strings.Contains(got, "N") || !strings.Contains(got, "W") {
		t.Fatalf("expected N/W hemispheres: %q", got)
	}
	// 0.30° = 18.00'
	if !strings.HasPrefix(got, "33°18.00'N") {
		t.Fatalf("lat format: %q", got)
	}
	if !strings.Contains(got, "118°27.00'W") {
		t.Fatalf("lon format: %q", got)
	}
}
