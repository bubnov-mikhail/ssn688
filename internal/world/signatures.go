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
		ID: "victor_iii", Name: "Victor III SSN", Class: "Pr.671RTM", Kind: KindSubmarine,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 45, LevelDB: 108},
			{LowHz: 45, HighHz: 150, LevelDB: 112},
			{LowHz: 150, HighHz: 450, LevelDB: 105},
			{LowHz: 450, HighHz: 1200, LevelDB: 92},
		},
		// Noisy nuclear hunter: dense turbine / pump cluster, stronger than Kilo.
		Tonals: []TonalLine{
			{FreqHz: 28, RelLevel: 0.75},
			{FreqHz: 56, RelLevel: 0.95},
			{FreqHz: 84, RelLevel: 0.88},
			{FreqHz: 112, RelLevel: 1.00},
			{FreqHz: 168, RelLevel: 0.90},
			{FreqHz: 280, RelLevel: 0.78},
			{FreqHz: 420, RelLevel: 0.65},
			{FreqHz: 560, RelLevel: 0.55},
			{FreqHz: 840, RelLevel: 0.45},
			{FreqHz: 1120, RelLevel: 0.38},
			{FreqHz: 1680, RelLevel: 0.30},
		},
		BladeRateHz: 3.6, CavitationDB: 95,
	},
	{
		ID: "foxtrot", Name: "Foxtrot SS", Class: "Pr.641", Kind: KindSubmarine,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 50, LevelDB: 122},
			{LowHz: 50, HighHz: 140, LevelDB: 124},
			{LowHz: 140, HighHz: 400, LevelDB: 115},
			{LowHz: 400, HighHz: 1000, LevelDB: 98},
		},
		// Older diesel boat: loud snort/diesel lines, sparse high band.
		Tonals: []TonalLine{
			{FreqHz: 26, RelLevel: 0.85},
			{FreqHz: 52, RelLevel: 1.00},
			{FreqHz: 78, RelLevel: 0.92},
			{FreqHz: 104, RelLevel: 0.80},
			{FreqHz: 208, RelLevel: 0.75},
			{FreqHz: 312, RelLevel: 0.60},
			{FreqHz: 520, RelLevel: 0.50},
			{FreqHz: 780, RelLevel: 0.40},
			{FreqHz: 1040, RelLevel: 0.32},
		},
		BladeRateHz: 2.8, CavitationDB: 98,
	},
	{
		ID: "udaloy", Name: "Udaloy DDG", Class: "Pr.1155", Kind: KindSurfaceShip,
		Bands: []NoiseBand{
			{LowHz: 20, HighHz: 85, LevelDB: 132},
			{LowHz: 85, HighHz: 260, LevelDB: 124},
			{LowHz: 260, HighHz: 750, LevelDB: 114},
			{LowHz: 750, HighHz: 2000, LevelDB: 100},
		},
		// Twin-shaft COGAG ASW destroyer: strong GT / reduction-gear cluster.
		Tonals: []TonalLine{
			{FreqHz: 16, RelLevel: 0.65},
			{FreqHz: 32, RelLevel: 0.88},
			{FreqHz: 48, RelLevel: 0.82},
			{FreqHz: 80, RelLevel: 1.00},
			{FreqHz: 160, RelLevel: 0.95},
			{FreqHz: 240, RelLevel: 0.78},
			{FreqHz: 400, RelLevel: 0.70},
			{FreqHz: 640, RelLevel: 0.58},
			{FreqHz: 960, RelLevel: 0.48},
			{FreqHz: 1280, RelLevel: 0.42},
			{FreqHz: 1760, RelLevel: 0.32},
		},
		BladeRateHz: 1.65, CavitationDB: 112,
	},
	{
		ID: "krivak", Name: "Krivak FF", Class: "Pr.1135", Kind: KindSurfaceShip,
		Bands: []NoiseBand{
			{LowHz: 20, HighHz: 75, LevelDB: 122},
			{LowHz: 75, HighHz: 230, LevelDB: 116},
			{LowHz: 230, HighHz: 620, LevelDB: 104},
			{LowHz: 620, HighHz: 1650, LevelDB: 92},
		},
		// Twin-screw Burevestnik FF: leaner mid-band than Udaloy, more shaft lines.
		Tonals: []TonalLine{
			{FreqHz: 22, RelLevel: 0.70},
			{FreqHz: 44, RelLevel: 0.90},
			{FreqHz: 66, RelLevel: 0.75},
			{FreqHz: 88, RelLevel: 1.00},
			{FreqHz: 132, RelLevel: 0.85},
			{FreqHz: 220, RelLevel: 0.88},
			{FreqHz: 440, RelLevel: 0.68},
			{FreqHz: 660, RelLevel: 0.55},
			{FreqHz: 880, RelLevel: 0.48},
			{FreqHz: 1210, RelLevel: 0.38},
			{FreqHz: 1540, RelLevel: 0.32},
		},
		BladeRateHz: 2.15, CavitationDB: 102,
	},
	{
		ID: "kresta2", Name: "Kresta II CG", Class: "Pr.1134A", Kind: KindSurfaceShip,
		Bands: []NoiseBand{
			{LowHz: 20, HighHz: 90, LevelDB: 134},
			{LowHz: 90, HighHz: 280, LevelDB: 126},
			{LowHz: 280, HighHz: 800, LevelDB: 116},
			{LowHz: 800, HighHz: 2000, LevelDB: 102},
		},
		// Large ASW cruiser: heavy steam-plant / gear tonals, denser than Udaloy.
		Tonals: []TonalLine{
			{FreqHz: 14, RelLevel: 0.70},
			{FreqHz: 28, RelLevel: 0.90},
			{FreqHz: 42, RelLevel: 0.85},
			{FreqHz: 70, RelLevel: 1.00},
			{FreqHz: 140, RelLevel: 0.98},
			{FreqHz: 210, RelLevel: 0.82},
			{FreqHz: 350, RelLevel: 0.72},
			{FreqHz: 560, RelLevel: 0.60},
			{FreqHz: 840, RelLevel: 0.50},
			{FreqHz: 1120, RelLevel: 0.42},
			{FreqHz: 1680, RelLevel: 0.34},
		},
		BladeRateHz: 1.45, CavitationDB: 115,
	},
	{
		ID: "grisha", Name: "Grisha Corvette", Class: "Pr.1124", Kind: KindSurfaceShip,
		Bands: []NoiseBand{
			{LowHz: 25, HighHz: 90, LevelDB: 118},
			{LowHz: 90, HighHz: 280, LevelDB: 112},
			{LowHz: 280, HighHz: 700, LevelDB: 100},
			{LowHz: 700, HighHz: 1800, LevelDB: 88},
		},
		// Small CODAG ASW corvette: higher blade rate, lean mid-band.
		Tonals: []TonalLine{
			{FreqHz: 30, RelLevel: 0.70},
			{FreqHz: 60, RelLevel: 0.92},
			{FreqHz: 90, RelLevel: 0.80},
			{FreqHz: 120, RelLevel: 1.00},
			{FreqHz: 180, RelLevel: 0.85},
			{FreqHz: 300, RelLevel: 0.70},
			{FreqHz: 480, RelLevel: 0.55},
			{FreqHz: 720, RelLevel: 0.45},
			{FreqHz: 1080, RelLevel: 0.35},
			{FreqHz: 1440, RelLevel: 0.28},
		},
		BladeRateHz: 2.6, CavitationDB: 105,
	},
	{
		ID: "merchant", Name: "Merchant Freighter", Class: "MV", Kind: KindSurfaceShip,
		Bands: []NoiseBand{
			{LowHz: 15, HighHz: 60, LevelDB: 118},
			{LowHz: 60, HighHz: 200, LevelDB: 112},
			{LowHz: 200, HighHz: 500, LevelDB: 100},
			{LowHz: 500, HighHz: 1400, LevelDB: 88},
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
		BladeRateHz: 1.4, CavitationDB: 92,
	},
	{
		ID: "tanker", Name: "Oil Tanker", Class: "VLCC", Kind: KindSurfaceShip,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 50, LevelDB: 124},
			{LowHz: 50, HighHz: 180, LevelDB: 118},
			{LowHz: 180, HighHz: 450, LevelDB: 105},
			{LowHz: 450, HighHz: 1200, LevelDB: 92},
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
		BladeRateHz: 1.1, CavitationDB: 98,
	},
	{
		ID: "fishing", Name: "Fishing Trawler", Class: "FV", Kind: KindSurfaceShip,
		Bands: []NoiseBand{
			{LowHz: 25, HighHz: 90, LevelDB: 110},
			{LowHz: 90, HighHz: 280, LevelDB: 104},
			{LowHz: 280, HighHz: 700, LevelDB: 92},
			{LowHz: 700, HighHz: 1600, LevelDB: 80},
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
		BladeRateHz: 2.4, CavitationDB: 88,
	},
	{
		ID: "mk48", Name: "Mk48 ADCAP", Class: "Torpedo", Kind: KindTorpedo,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 200, LevelDB: 72},
			{LowHz: 200, HighHz: 600, LevelDB: 88},
			{LowHz: 600, HighHz: 1100, LevelDB: 118},
			{LowHz: 1100, HighHz: 2000, LevelDB: 128},
		},
		Tonals: []TonalLine{
			{FreqHz: 880, RelLevel: 0.70},
			{FreqHz: 1100, RelLevel: 0.95},
			{FreqHz: 1320, RelLevel: 1.00},
			{FreqHz: 1540, RelLevel: 0.85},
			{FreqHz: 1760, RelLevel: 0.70},
			{FreqHz: 1980, RelLevel: 0.55},
		},
		BladeRateHz: 22, CavitationDB: 95,
	},
	{
		ID: "umgt1", Name: "UMGT-1 Orlan", Class: "Torpedo", Kind: KindTorpedo,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 200, LevelDB: 66},
			{LowHz: 200, HighHz: 560, LevelDB: 80},
			{LowHz: 560, HighHz: 1150, LevelDB: 110},
			{LowHz: 1150, HighHz: 2000, LevelDB: 120},
		},
		// Lightweight ASW fish: higher blade rate than heavy 53-series.
		Tonals: []TonalLine{
			{FreqHz: 950, RelLevel: 0.60},
			{FreqHz: 1180, RelLevel: 0.88},
			{FreqHz: 1410, RelLevel: 1.00},
			{FreqHz: 1640, RelLevel: 0.78},
			{FreqHz: 1870, RelLevel: 0.58},
		},
		BladeRateHz: 30, CavitationDB: 86,
	},
	{
		ID: "set40", Name: "SET-40 Torpedo", Class: "Torpedo", Kind: KindTorpedo,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 220, LevelDB: 64},
			{LowHz: 220, HighHz: 580, LevelDB: 78},
			{LowHz: 580, HighHz: 1200, LevelDB: 108},
			{LowHz: 1200, HighHz: 2000, LevelDB: 118},
		},
		// Older lightweight ASW fish (Grisha tubes): slightly lower pitch than UMGT-1.
		Tonals: []TonalLine{
			{FreqHz: 880, RelLevel: 0.55},
			{FreqHz: 1100, RelLevel: 0.85},
			{FreqHz: 1320, RelLevel: 1.00},
			{FreqHz: 1540, RelLevel: 0.75},
			{FreqHz: 1760, RelLevel: 0.55},
		},
		BladeRateHz: 26, CavitationDB: 84,
	},
	{
		ID: "type53", Name: "53-65 Torpedo", Class: "Torpedo", Kind: KindTorpedo,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 200, LevelDB: 78},
			{LowHz: 200, HighHz: 550, LevelDB: 92},
			{LowHz: 550, HighHz: 1000, LevelDB: 120},
			{LowHz: 1000, HighHz: 2000, LevelDB: 130},
		},
		Tonals: []TonalLine{
			{FreqHz: 760, RelLevel: 0.65},
			{FreqHz: 980, RelLevel: 0.90},
			{FreqHz: 1220, RelLevel: 1.00},
			{FreqHz: 1460, RelLevel: 0.80},
			{FreqHz: 1700, RelLevel: 0.65},
			{FreqHz: 1940, RelLevel: 0.50},
		},
		BladeRateHz: 18, CavitationDB: 100,
	},
	{
		ID: "adc", Name: "Acoustic Decoy", Class: "CM", Kind: KindCountermeasure,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 80, LevelDB: 125},
			{LowHz: 80, HighHz: 300, LevelDB: 132},
			{LowHz: 300, HighHz: 900, LevelDB: 128},
			{LowHz: 900, HighHz: 2000, LevelDB: 118},
		},
		Tonals: []TonalLine{
			{FreqHz: 55, RelLevel: 0.70},
			{FreqHz: 110, RelLevel: 0.95},
			{FreqHz: 220, RelLevel: 1.00},
			{FreqHz: 440, RelLevel: 0.85},
			{FreqHz: 880, RelLevel: 0.60},
		},
		BladeRateHz: 0, CavitationDB: 70,
	},
	{
		ID: "nixie", Name: "Towed Torpedo Decoy", Class: "CM", Kind: KindCountermeasure,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 100, LevelDB: 128},
			{LowHz: 100, HighHz: 350, LevelDB: 136},
			{LowHz: 350, HighHz: 1000, LevelDB: 130},
			{LowHz: 1000, HighHz: 2000, LevelDB: 122},
		},
		Tonals: []TonalLine{
			{FreqHz: 40, RelLevel: 0.80},
			{FreqHz: 80, RelLevel: 1.00},
			{FreqHz: 160, RelLevel: 0.95},
			{FreqHz: 320, RelLevel: 0.75},
			{FreqHz: 640, RelLevel: 0.55},
		},
		BladeRateHz: 1.5, CavitationDB: 90,
	},
	{
		ID: "jitter", Name: "Acoustic Jammer", Class: "CM", Kind: KindCountermeasure,
		Bands: []NoiseBand{
			{LowHz: 10, HighHz: 80, LevelDB: 130},
			{LowHz: 80, HighHz: 300, LevelDB: 138},
			{LowHz: 300, HighHz: 900, LevelDB: 136},
			{LowHz: 900, HighHz: 2000, LevelDB: 128},
		},
		// Dense overlapping lines — broadband confusion rather than a clean hull fingerprint.
		Tonals: []TonalLine{
			{FreqHz: 70, RelLevel: 0.90},
			{FreqHz: 95, RelLevel: 0.85},
			{FreqHz: 140, RelLevel: 1.00},
			{FreqHz: 210, RelLevel: 0.95},
			{FreqHz: 280, RelLevel: 0.90},
			{FreqHz: 420, RelLevel: 0.80},
			{FreqHz: 560, RelLevel: 0.75},
			{FreqHz: 840, RelLevel: 0.65},
			{FreqHz: 1120, RelLevel: 0.55},
			{FreqHz: 1680, RelLevel: 0.45},
		},
		BladeRateHz: 0, CavitationDB: 60,
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
