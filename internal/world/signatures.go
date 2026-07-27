package world

// SignatureLibrary holds known acoustic profiles for classification.
// Tonals are discrete LOFAR/DEMON peaks (machinery + blade-rate harmonics),
// styled after Cold Waters signature fingerprints and typical DEMON/LOFAR
// analysis (shaft/blade rate lines + distinctive machinery tonals).
var SignatureLibrary = []SignatureProfile{
	{
		ID: "los_angeles", Name: "Los Angeles SSN", Class: "SSN-688", Kind: KindSubmarine,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 50, LevelDB: 88},
			{LowHz: 50, HighHz: 200, LevelDB: 98},
			{LowHz: 200, HighHz: 600, LevelDB: 92},
			{LowHz: 600, HighHz: 1500, LevelDB: 78},
		},
		// Quiet SSN: sparse machinery + 7-blade-ish rate ~4.2 Hz → visible harmonics.
		Tonals: []TonalLine{
			{FreqHz: 42, RelLevel: 0.55},
			{FreqHz: 84, RelLevel: 0.70},
			{FreqHz: 126, RelLevel: 0.85},
			{FreqHz: 210, RelLevel: 0.95},
			{FreqHz: 420, RelLevel: 0.75},
			{FreqHz: 630, RelLevel: 0.55},
			{FreqHz: 1050, RelLevel: 0.45},
			{FreqHz: 1470, RelLevel: 0.35},
		},
		BladeRateHz: 4.2, CavitationDB: 75,
	},
	{
		ID: "kilo", Name: "Kilo SS", Class: "SSK", Kind: KindSubmarine,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 40, LevelDB: 115},
			{LowHz: 40, HighHz: 120, LevelDB: 118},
			{LowHz: 120, HighHz: 350, LevelDB: 108},
			{LowHz: 350, HighHz: 900, LevelDB: 88},
		},
		// Diesel-electric: stronger low-frequency shaft/blade cluster + diesel lines.
		Tonals: []TonalLine{
			{FreqHz: 31, RelLevel: 0.80},
			{FreqHz: 62, RelLevel: 1.00},
			{FreqHz: 93, RelLevel: 0.90},
			{FreqHz: 155, RelLevel: 0.85},
			{FreqHz: 310, RelLevel: 0.70},
			{FreqHz: 465, RelLevel: 0.55},
			{FreqHz: 775, RelLevel: 0.50},
			{FreqHz: 1240, RelLevel: 0.40},
			{FreqHz: 1860, RelLevel: 0.30},
		},
		BladeRateHz: 3.1, CavitationDB: 90,
	},
	{
		ID: "spruance", Name: "Spruance DD", Class: "DD-963", Kind: KindSurfaceShip,
		Bands: []NoiseBand{
			{LowHz: 20, HighHz: 80, LevelDB: 130},
			{LowHz: 80, HighHz: 250, LevelDB: 122},
			{LowHz: 250, HighHz: 700, LevelDB: 112},
			{LowHz: 700, HighHz: 2000, LevelDB: 98},
		},
		// Twin-screw surface combatant: dense blade harmonics + gear/turbine lines.
		Tonals: []TonalLine{
			{FreqHz: 18, RelLevel: 0.70},
			{FreqHz: 36, RelLevel: 0.90},
			{FreqHz: 54, RelLevel: 0.85},
			{FreqHz: 90, RelLevel: 1.00},
			{FreqHz: 180, RelLevel: 0.95},
			{FreqHz: 270, RelLevel: 0.80},
			{FreqHz: 450, RelLevel: 0.65},
			{FreqHz: 720, RelLevel: 0.55},
			{FreqHz: 1080, RelLevel: 0.45},
			{FreqHz: 1440, RelLevel: 0.40},
			{FreqHz: 1800, RelLevel: 0.30},
		},
		BladeRateHz: 1.8, CavitationDB: 110,
	},
	{
		ID: "perry", Name: "Perry FF", Class: "FFG-7", Kind: KindSurfaceShip,
		Bands: []NoiseBand{
			{LowHz: 20, HighHz: 70, LevelDB: 120},
			{LowHz: 70, HighHz: 220, LevelDB: 114},
			{LowHz: 220, HighHz: 600, LevelDB: 102},
			{LowHz: 600, HighHz: 1600, LevelDB: 90},
		},
		// Single-screw FFG: mid-density fingerprint, distinct from Spruance.
		Tonals: []TonalLine{
			{FreqHz: 24, RelLevel: 0.65},
			{FreqHz: 48, RelLevel: 0.85},
			{FreqHz: 96, RelLevel: 1.00},
			{FreqHz: 144, RelLevel: 0.80},
			{FreqHz: 240, RelLevel: 0.90},
			{FreqHz: 480, RelLevel: 0.70},
			{FreqHz: 720, RelLevel: 0.55},
			{FreqHz: 960, RelLevel: 0.45},
			{FreqHz: 1320, RelLevel: 0.35},
			{FreqHz: 1680, RelLevel: 0.30},
		},
		BladeRateHz: 2.0, CavitationDB: 100,
	},
	{
		ID: "merchant", Name: "Merchant Freighter", Class: "MV", Kind: KindSurfaceShip,
		Bands: []NoiseBand{
			{LowHz: 15, HighHz: 60, LevelDB: 128},
			{LowHz: 60, HighHz: 200, LevelDB: 122},
			{LowHz: 200, HighHz: 500, LevelDB: 110},
			{LowHz: 500, HighHz: 1400, LevelDB: 95},
		},
		Tonals: []TonalLine{
			{FreqHz: 22, RelLevel: 0.75},
			{FreqHz: 44, RelLevel: 0.95},
			{FreqHz: 88, RelLevel: 1.00},
			{FreqHz: 132, RelLevel: 0.70},
			{FreqHz: 220, RelLevel: 0.80},
			{FreqHz: 440, RelLevel: 0.55},
			{FreqHz: 880, RelLevel: 0.40},
			{FreqHz: 1320, RelLevel: 0.30},
		},
		BladeRateHz: 1.4, CavitationDB: 105,
	},
	{
		ID: "tanker", Name: "Oil Tanker", Class: "VLCC", Kind: KindSurfaceShip,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 50, LevelDB: 135},
			{LowHz: 50, HighHz: 180, LevelDB: 128},
			{LowHz: 180, HighHz: 450, LevelDB: 115},
			{LowHz: 450, HighHz: 1200, LevelDB: 100},
		},
		Tonals: []TonalLine{
			{FreqHz: 16, RelLevel: 0.85},
			{FreqHz: 32, RelLevel: 1.00},
			{FreqHz: 64, RelLevel: 0.95},
			{FreqHz: 96, RelLevel: 0.75},
			{FreqHz: 160, RelLevel: 0.85},
			{FreqHz: 320, RelLevel: 0.60},
			{FreqHz: 640, RelLevel: 0.45},
			{FreqHz: 960, RelLevel: 0.35},
			{FreqHz: 1600, RelLevel: 0.25},
		},
		BladeRateHz: 1.1, CavitationDB: 112,
	},
	{
		ID: "fishing", Name: "Fishing Trawler", Class: "FV", Kind: KindSurfaceShip,
		Bands: []NoiseBand{
			{LowHz: 25, HighHz: 90, LevelDB: 118},
			{LowHz: 90, HighHz: 280, LevelDB: 112},
			{LowHz: 280, HighHz: 700, LevelDB: 100},
			{LowHz: 700, HighHz: 1600, LevelDB: 88},
		},
		Tonals: []TonalLine{
			{FreqHz: 28, RelLevel: 0.70},
			{FreqHz: 56, RelLevel: 0.90},
			{FreqHz: 112, RelLevel: 1.00},
			{FreqHz: 168, RelLevel: 0.65},
			{FreqHz: 280, RelLevel: 0.75},
			{FreqHz: 560, RelLevel: 0.50},
			{FreqHz: 840, RelLevel: 0.40},
			{FreqHz: 1400, RelLevel: 0.30},
		},
		BladeRateHz: 2.4, CavitationDB: 98,
	},
}

func ProfileByID(id string) (SignatureProfile, bool) {
	for _, p := range SignatureLibrary {
		if p.ID == id {
			return p, true
		}
	}
	return SignatureProfile{}, false
}
