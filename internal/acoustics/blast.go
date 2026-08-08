package acoustics

import (
	"math"

	"github.com/ssn688/sim/internal/weapons"
	"github.com/ssn688/sim/internal/world"
)

// ApplyDetonationDeaf blinds the listener within blast radius and stamps washout.
// Surface-ship (incl. neutral) detonations couple strongly into the water column —
// longer-range washout so distant merchants still light up the passive display.
func ApplyDetonationDeaf(sonar *SonarState, listener *world.Entity, x, y, gameTime float64, hit *world.Entity) {
	if sonar == nil || listener == nil {
		return
	}
	washRange := weapons.BlastDeafRadiusYd * 1.2
	deafRange := weapons.BlastDeafRadiusYd
	flashSec := 24.0
	if hit != nil && hit.Kind == world.KindSurfaceShip {
		// Under-keel / shallow warhead: loud broadband for many kiloyards.
		washRange = weapons.BlastDeafRadiusYd * 4.0 // ~10 kyd visual flash
		deafRange = weapons.BlastDeafRadiusYd * 2.2
		flashSec = 32.0
	}
	stampBlastWashout(sonar, listener, x, y, gameTime, washRange, deafRange, flashSec, hit)
}

// ApplyCookOffDeaf is a lighter secondary magazine/fuel flash on a sinking wreck.
func ApplyCookOffDeaf(sonar *SonarState, listener *world.Entity, x, y, gameTime float64, hit *world.Entity) {
	if sonar == nil || listener == nil {
		return
	}
	washRange := weapons.BlastDeafRadiusYd * 0.85
	deafRange := weapons.BlastDeafRadiusYd * 0.55
	flashSec := 9.0
	if hit != nil && hit.Kind == world.KindSurfaceShip {
		washRange = weapons.BlastDeafRadiusYd * 2.4
		deafRange = weapons.BlastDeafRadiusYd * 1.2
		flashSec = 12.0
	}
	stampBlastWashout(sonar, listener, x, y, gameTime, washRange, deafRange, flashSec, hit)
}

func stampBlastWashout(sonar *SonarState, listener *world.Entity, x, y, gameTime, washRange, deafRange, flashSec float64, hit *world.Entity) {
	sonar.LastBlastAt = gameTime
	sonar.LastBlastX = x
	sonar.LastBlastY = y
	sonar.LastBlastRangeYd = washRange
	sonar.LastBlastFlashSec = flashSec
	sonar.LastBlastEntityID = ""
	if hit != nil {
		sonar.LastBlastEntityID = hit.ID
		// Anchor to the hull so the visual follows a moving/sinking target.
		sonar.LastBlastX = hit.X
		sonar.LastBlastY = hit.Y
	}
	dist := math.Hypot(listener.X-x, listener.Y-y)
	if dist <= deafRange {
		until := gameTime + weapons.BlastDeafDurationSec*(0.35+0.45*(1-dist/deafRange))
		if until > sonar.SonarDeafUntil {
			sonar.SonarDeafUntil = until
		}
	}
}
