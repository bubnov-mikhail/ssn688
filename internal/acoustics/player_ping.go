package acoustics

import (
	"math"

	"github.com/ssn688/sim/internal/world"
)

const (
	// PlayerPingAlertWindowSec is how long after transmit enemies may react.
	PlayerPingAlertWindowSec = 45.0
	// PlayerPingRevealDurationSec is how long passive detectability stays elevated.
	PlayerPingRevealDurationSec = 35.0
	// PlayerPingHearThresholdDB is minimum received level to classify as heard.
	PlayerPingHearThresholdDB = 92.0
)

// PlayerPingHeardAge returns seconds since the ping arrived at listener, or -1.
func PlayerPingHeardAge(listener, player *world.Entity, gameTime float64) float64 {
	if listener == nil || player == nil || player.LastPingTime <= 0 {
		return -1
	}
	pingAge := gameTime - player.LastPingTime
	if pingAge < 0 || pingAge > PlayerPingAlertWindowSec {
		return -1
	}
	rangeYd := listener.RangeYardsTo(player)
	travel := rangeYd / SoundSpeedYdPerSec
	if pingAge < travel {
		return -1
	}
	heardAge := pingAge - travel
	if heardAge > PlayerPingRevealDurationSec {
		return -1
	}
	power := player.LastPingPower
	if power <= 0 {
		power = 0.7
	}
	if ReceivedPlayerPingLevelDB(rangeYd, power) < PlayerPingHearThresholdDB {
		return -1
	}
	return heardAge
}

// HeardPlayerPing reports whether listener has received the player's latest ping.
func HeardPlayerPing(listener, player *world.Entity, gameTime float64) bool {
	return PlayerPingHeardAge(listener, player, gameTime) >= 0
}

// ReceivedPlayerPingLevelDB estimates one-way received ping level at range (dB re 1 µPa).
func ReceivedPlayerPingLevelDB(rangeYd, pingPower float64) float64 {
	if rangeYd < 100 {
		rangeYd = 100
	}
	rangeKy := rangeYd / 1000
	spread := spreadingLossDB(rangeYd)
	abs := absorptionDBPerKy(3000, rangeKy)
	return PingSourceLevel(pingPower) - spread - abs
}

// PlayerPingPassiveBonusDB is extra passive SNR against the pinging submarine.
func PlayerPingPassiveBonusDB(heardAge, pingPower float64) float64 {
	if heardAge < 0 {
		return 0
	}
	if pingPower <= 0 {
		pingPower = 0.7
	}
	flash := math.Exp(-heardAge * 0.11)
	return (10 + 22*flash) * pingPower
}
