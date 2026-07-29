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
	flashSec := 8.0
	if hit != nil && hit.Kind == world.KindSurfaceShip {
		// Under-keel / shallow warhead: loud broadband for many kiloyards.
		washRange = weapons.BlastDeafRadiusYd * 4.0 // ~10 kyd visual flash
		deafRange = weapons.BlastDeafRadiusYd * 2.2
		flashSec = 12.0
	}
	sonar.LastBlastAt = gameTime
	sonar.LastBlastX = x
	sonar.LastBlastY = y
	sonar.LastBlastRangeYd = washRange
	sonar.LastBlastFlashSec = flashSec
	dist := math.Hypot(listener.X-x, listener.Y-y)
	if dist <= deafRange {
		until := gameTime + weapons.BlastDeafDurationSec*(1-dist/deafRange*0.4)
		if until > sonar.SonarDeafUntil {
			sonar.SonarDeafUntil = until
		}
	}
}
