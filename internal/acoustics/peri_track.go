package acoustics

import (
	"math"
	"strings"

	"github.com/bubnov-mikhail/ssn688/internal/world"
)

// UpdateContactsFromPeriscope refines (or creates) surface tracks from what the
// IR optic actually shows: true LOS bearing and stadimeter-equivalent range.
// Silhouette size/aspect on the optic always come from entity truth; this feeds
// those observations back into EstimatedRange / TMA course-speed.
func UpdateContactsFromPeriscope(
	sonar *SonarState,
	peri *PeriscopeState,
	player *world.Entity,
	entities []*world.Entity,
	weather world.Weather,
	gameTime float64,
) {
	if sonar == nil || peri == nil || player == nil || !peri.MastUp() || peri.Sheared {
		return
	}
	eye := EyeAboveWaterFt(player.DepthFt, peri.Extension)
	if eye < 0.5 {
		return
	}
	maxR := OpticalMaxRangeYd(eye, weather)
	look := peri.TrueBearingDeg(player.HeadingDeg)
	halfFOV := peri.FOVDeg() * 0.5

	existing := make(map[string]int, len(sonar.Contacts))
	for i := range sonar.Contacts {
		id := sonar.Contacts[i].SourceEntityID
		if id != "" {
			existing[id] = i
		}
	}

	for _, ship := range entities {
		if ship == nil || ship.Kind != world.KindSurfaceShip {
			continue
		}
		if !ship.Alive() && ship.Status != world.StatusSinking {
			continue
		}
		rangeYd := player.RangeYardsTo(ship)
		if rangeYd < 30 || rangeYd > maxR {
			continue
		}
		brg := player.BearingDegTo(ship)
		if math.Abs(AngleDiffSigned(brg, look)) > halfFOV {
			continue
		}

		// Stadimeter-class optical range: what the silhouette implies (±~2%).
		measRange := rangeYd * (1 + clampNorm(randNorm()*0.02, -0.05, 0.05))
		if measRange < 30 {
			measRange = 30
		}

		if idx, ok := existing[ship.ID]; ok {
			c := &sonar.Contacts[idx]
			updateContactFromVisual(c, brg, measRange, gameTime, ship)
			updateContactTMA(c, sampleTMAPosition(player, c.BearingDeg, c.EstimatedRangeYd, gameTime, 0.97))
			tryVisualIdentify(c, ship, rangeYd, gameTime)
			continue
		}

		sonar.contactSeq++
		c := Contact{
			ID:               formatContactID(sonar.contactSeq),
			BearingDeg:       brg,
			EstimatedRangeYd: measRange,
			SNR:              18,
			BandsAbove:       4,
			BestMatchID:      ship.SignatureID,
			BestMatchName:    ship.Name,
			Confidence:       0.55,
			SourceEntityID:   ship.ID,
			Kind:             world.KindSurfaceShip,
			DetectedBy:       "visual",
			LastUpdate:       gameTime,
			FirstSeen:        gameTime,
		}
		initContactUncertainty(&c)
		// Optical fix collapses the blob quickly.
		c.UncBearingDeg = 3
		c.UncRangeYd = math.Max(100, measRange*0.06)
		updateContactTMA(&c, sampleTMAPosition(player, brg, measRange, gameTime, 0.97))
		tryVisualIdentify(&c, ship, rangeYd, gameTime)
		sonar.Contacts = append(sonar.Contacts, c)
		existing[ship.ID] = len(sonar.Contacts) - 1
	}
}

func updateContactFromVisual(c *Contact, measBearing, measRange, gameTime float64, ship *world.Entity) {
	if c == nil {
		return
	}
	c.ListenTime = gameTime - c.FirstSeen
	c.LastUpdate = gameTime
	if c.SNR < 14 {
		c.SNR = 14
	}
	if ship != nil {
		if c.BestMatchID == "" {
			c.BestMatchID = ship.SignatureID
			c.BestMatchName = ship.Name
		}
		c.Kind = world.KindSurfaceShip
	}
	if c.DetectedBy == "" {
		c.DetectedBy = "visual"
	} else if !strings.Contains(c.DetectedBy, "visual") {
		c.DetectedBy += "/visual"
	}
	// Optical fixes pull the track hard toward what the optic shows.
	const alpha = 0.42
	c.BearingDeg = lerpAngleDeg(c.BearingDeg, measBearing, alpha)
	if c.EstimatedRangeYd <= 0 {
		c.EstimatedRangeYd = measRange
	} else {
		c.EstimatedRangeYd = c.EstimatedRangeYd*(1-alpha) + measRange*alpha
	}
	c.UncBearingDeg = shrinkUncertainty(c.UncBearingDeg, 1.8)
	c.UncRangeYd = shrinkUncertainty(c.UncRangeYd, math.Max(80, measRange*0.045))
	if c.Confidence < 0.75 {
		c.Confidence = math.Min(0.92, c.Confidence+0.04)
	}
}

func clampNorm(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// randNorm is isolated so tests can stay deterministic when needed.
var randNorm = func() float64 {
	return math.Sin(float64(len("peri")) ) // replaced below
}
