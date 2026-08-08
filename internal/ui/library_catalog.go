package ui

import "github.com/ssn688/sim/internal/world"

// libraryAllegiance groups catalog entries for the LIBRARY object table.
type libraryAllegiance int

const (
	libHostile libraryAllegiance = iota
	libNeutral
	libFriendly
)

type libraryEntry struct {
	ID         string
	Allegiance libraryAllegiance
	Kind       world.EntityKind
	Title      string
	Summary    []string
	Specs      []string
	Offense    []string
	Defense    []string
	Neutralize []string
	Evade      []string
	ImageFile  string
	Credit     string
}

// libraryCatalog is the ordered platform handbook (no torpedoes / CMs).
var libraryCatalog = []libraryEntry{
	{
		ID: "udaloy", Allegiance: libHostile, Kind: world.KindSurfaceShip,
		Title:     "Udaloy DDG — Project 1155 Fregat",
		ImageFile: "udaloy.jpg",
		Credit:    "Photo: U.S. DoD / public domain (Wikimedia)",
		Summary: []string{
			"Large Soviet/Russian ASW destroyer built to hunt nuclear submarines with helicopters, rocket-assisted torpedoes, and a capable undersea sensor suite. Twin-shaft gas-turbine plant gives strong mid-band shaft and gear tonals.",
			"In this scenario an Udaloy is a primary surface hunter: expect Rastrub (Metel) ASW rockets, lightweight fish, and point-defense layers against Harpoon.",
		},
		Specs: []string{
			"Displacement ~7,500 t full load",
			"Length ~163 m  |  Speed ~29–30 kn",
			"Crew ~300  |  ASW helicopters: Ka-27 (2)",
			"Acoustic: loud GT / reduction-gear cluster; blade ~1.65 Hz",
			"Radar: MR-320 Fregat (Top Plate) — S-band air/surface, ~6–12 rpm (5–10 s/scan)",
			"Mast detect (calm): ~14 kyd vs raised ESM/periscope stalk",
		},
		Offense: []string{
			"Rastrub (Metel) ASW rocket — magazine ~8; splash → UMGT-1",
			"Ship tubes — UMGT-1 lightweight ASW torpedoes",
			"Active sonar / helicopter cueing for localization",
		},
		Defense: []string{
			"Kinzhal/Osa-M class SAM — medium AAW vs Harpoon (~2–8 kyd)",
			"AK-630 CIWS bursts — inner layer (~0.2–2 kyd)",
			"Hull strength: survives glancing fish; vulnerable to Mk48 under keel",
			"Search radar paints raised masts; storm seas reduce mast RCS",
		},
		Neutralize: []string{
			"Prefer Mk48 from outside helicopter/Rastrub comfort range.",
			"Harpoon can work if SAM/CIWS are depleted or you fire from surprise/shallow snapshot geometry.",
			"Do not linger on a steady bearing while within ASW rocket range.",
		},
		Evade: []string{
			"Go deep / under layer; slow to cut self-noise if he is searching passively.",
			"If Rastrub/UMGT-1 in water: speed change + depth + ADC/jitter; break Doppler.",
			"Avoid periscope/radar exposure while his Helix is airborne.",
		},
	},
	{
		ID: "krivak", Allegiance: libHostile, Kind: world.KindSurfaceShip,
		Title:     "Krivak FF — Project 1135 Burevestnik",
		ImageFile: "krivak.jpg",
		Credit:    "Photo: U.S. DoD / public domain (Wikimedia)",
		Summary: []string{
			"Escort frigate optimized for ASW patrol and convoy work. Smaller and leaner than Udaloy, but still carries rocket ASW and tubes with a noisy twin-screw signature.",
			"Treat as a persistent surface threat that can prosecute a contact once localized by own sensors or cueing.",
		},
		Specs: []string{
			"Displacement ~3,500 t full load",
			"Length ~123 m  |  Speed ~32 kn",
			"Crew ~180–200",
			"Acoustic: shaft lines denser than Udaloy; blade ~2.15 Hz",
			"Radar: MR-310U Angara-M — S-band air/surface, ~12 rpm (≈5 s/scan)",
			"Mast detect (calm): ~11 kyd vs raised ESM stalk",
		},
		Offense: []string{
			"Rastrub ASW rocket — magazine ~8; splash → UMGT-1",
			"Ship tubes — UMGT-1",
			"Limited organic air compared with Udaloy",
		},
		Defense: []string{
			"SAM layer vs sea-skimmers (magazine ~8)",
			"CIWS bursts (magazine ~10)",
			"Thinner hull than cruisers — Mk48 remains decisive",
		},
		Neutralize: []string{
			"Mk48 preferred; Harpoon viable after baiting or depleting PD.",
			"Exploit his smaller ASW aviation footprint vs Udaloy.",
		},
		Evade: []string{
			"Layer + speed management; he relies more on ship sensors.",
			"Same CM doctrine vs lightweight fish as vs Udaloy.",
		},
	},
	{
		ID: "kresta2", Allegiance: libHostile, Kind: world.KindSurfaceShip,
		Title:     "Kresta II CG — Project 1134A Berkut-A",
		ImageFile: "kresta2.jpg",
		Credit:    "Photo: U.S. DoD / public domain (Wikimedia)",
		Summary: []string{
			"Large ASW cruiser with a heavier steam plant and denser LOFAR fingerprint than Udaloy. Carries more Rastrub rounds and stronger point-defense magazines.",
			"A high-value surface unit: dangerous in a prolonged ASW chase and harder to attrit with a single Harpoon.",
		},
		Specs: []string{
			"Displacement ~7,500–8,000 t full load",
			"Length ~159 m  |  Speed ~32–34 kn",
			"Crew ~340+  |  Helicopter: Ka-25/Ka-27 capable",
			"Acoustic: heavy steam/gear tonals; blade ~1.45 Hz",
			"Radar: MR-320 Fregat class — S-band air/surface, ~6–12 rpm (5–10 s/scan)",
			"Mast detect (calm): ~14 kyd vs raised ESM stalk",
		},
		Offense: []string{
			"Rastrub magazine ~12 — sustained rocket ASW",
			"Ship tubes — UMGT-1 (magazine ~8)",
			"Strong search/prosecution endurance",
		},
		Defense: []string{
			"SAM magazine ~12",
			"CIWS magazine ~14",
			"Large target: easier acoustic/TMA hold once detected",
		},
		Neutralize: []string{
			"Plan multi-weapon: Mk48 primary; Harpoon only with PD saturation or distraction.",
			"Expect him to keep firing Rastrub longer than escorts.",
		},
		Evade: []string{
			"Break contact early; do not slug it out inside rocket envelopes.",
			"Use bathymetry/layer; dump CM only on CPA threat, not every launch.",
		},
	},
	{
		ID: "grisha", Allegiance: libHostile, Kind: world.KindSurfaceShip,
		Title:     "Grisha Corvette — Project 1124 Albatros",
		ImageFile: "grisha.jpg",
		Credit:    "Photo: U.S. DoD / DPLA / public domain (Wikimedia)",
		Summary: []string{
			"Small coastal ASW corvette. No Rastrub — instead RBU rocket depth charges and SET-40 lightweight tubes. Quieter mid-band than destroyers but still a surface broadband source.",
			"Dangerous inshore: RBU can punish a shallow or poorly placed boat at short range.",
		},
		Specs: []string{
			"Displacement ~1,000 t full load",
			"Length ~71 m  |  Speed ~35 kn burst",
			"Crew ~80–90",
			"Acoustic: higher blade rate ~2.6 Hz; lean mid-band",
			"Radar: MR-302 Rubka — X-band surface search, ~15 rpm (≈4 s/scan)",
			"Mast detect (calm): ~9 kyd vs thin periscope/ESM stalk",
		},
		Offense: []string{
			"RBU salvos — short-range ASW; debris/light damage in sim",
			"Ship tubes — SET-40 (magazine ~4)",
			"No Metel/Rastrub rocket ASW",
		},
		Defense: []string{
			"Light SAM (magazine ~4)",
			"Light CIWS (magazine ~6)",
			"Fragile vs Mk48; Harpoon often overkill but effective",
			"Search radar will paint a raised ESM mast inside detect range",
		},
		Neutralize: []string{
			"Mk48 or Harpoon from outside RBU comfort; do not duel shallow.",
			"He is easier to kill than Udaloy/Kresta once weapons are on bearing.",
		},
		Evade: []string{
			"Stay deep/outside RBU range; speed away if he closes hot.",
			"SET-40: treat like other lightweight fish — CM + maneuver on CPA.",
		},
	},
	{
		ID: "kilo", Allegiance: libHostile, Kind: world.KindSubmarine,
		Title:     "Kilo SS — Project 877 Paltus",
		ImageFile: "kilo.jpg",
		Credit:    "Photo: U.S. DoD / DPLA / public domain (Wikimedia)",
		Summary: []string{
			"Diesel-electric attack boat with strong low-frequency shaft/diesel lines when snorkeling or high-rate charging. Quiet on the battery relative to older diesels, but still classifiable on SPECTRUM.",
			"Primary weapon threat is heavy 53-series fish at modest cruise speed.",
		},
		Specs: []string{
			"Displacement ~3,000 t submerged",
			"Length ~73 m  |  Speed ~17 kn submerged / ~10 kn snort",
			"Torpedo tubes: 6 × 533 mm  |  Mag ~12 (sim default)",
			"Acoustic: diesel/shaft cluster; blade ~3.1 Hz",
		},
		Offense: []string{
			"53-65 heavy torpedoes — cruise ~48 kn in sim",
			"Passive/active seeker fish; can force CM expenditure",
		},
		Defense: []string{
			"ADC / decoy / jammer soft-kill vs ownship fish",
			"No surface SAMs — vulnerable if forced to snort near escorts",
		},
		Neutralize: []string{
			"Mk48 with good TMA; prefer quiet approach and shot from his baffles.",
			"Do not fire Harpoon at a submerged Kilo.",
		},
		Evade: []string{
			"If he shoots: evade on CPA, deploy CM, change depth/speed.",
			"Own quieting + layer denial reduces his detection advantage.",
		},
	},
	{
		ID: "foxtrot", Allegiance: libHostile, Kind: world.KindSubmarine,
		Title:     "Foxtrot SS — Project 641",
		ImageFile: "foxtrot.jpg",
		Credit:    "Photo: U.S. DoD / public domain (Wikimedia)",
		Summary: []string{
			"Older diesel boat: louder snort/diesel fingerprint and thinner modern quieting. Easier to classify and track than Kilo or Victor once you have solid tonals.",
			"Still lethal at short range with heavy fish, but slower cruise and smaller magazine.",
		},
		Specs: []string{
			"Displacement ~2,500 t submerged",
			"Length ~91 m  |  Speed ~15–16 kn submerged",
			"Mag ~10 heavy fish  |  Cruise fish ~40 kn in sim",
			"Acoustic: loud diesel lines; blade ~2.8 Hz",
		},
		Offense: []string{
			"53-65 heavy torpedoes (slower cruise than Kilo/Victor)",
			"Limited magazine — he cannot spam forever",
		},
		Defense: []string{
			"Soft-kill CM only",
			"High self-noise when snorkeling — exploitable for TMA",
		},
		Neutralize: []string{
			"Mk48; use his noisy signature to hold bearing/range.",
			"Press when he is forced to snort.",
		},
		Evade: []string{
			"Standard fish evasion; his slower fish gives more reaction time.",
			"Do not mirror his depth if he is clearly above/below the layer.",
		},
	},
	{
		ID: "victor_iii", Allegiance: libHostile, Kind: world.KindSubmarine,
		Title:     "Victor III SSN — Project 671RTM Shchuka",
		ImageFile: "victor_iii.jpg",
		Credit:    "Photo: U.S. Navy / DoD / public domain (Wikimedia)",
		Summary: []string{
			"Nuclear hunter with a dense turbine/pump tonal set — noisier than a 688, but faster and better sustained submerged endurance than diesels. Primary peer ASW threat underwater.",
			"Larger magazine and faster fish cruise make him the most dangerous submarine class in the training set.",
		},
		Specs: []string{
			"Displacement ~6,000 t submerged",
			"Length ~107 m  |  Speed ~30+ kn submerged",
			"Mag ~18  |  Fish cruise ~55 kn in sim",
			"Acoustic: dense nuclear plant lines; blade ~3.6 Hz",
		},
		Offense: []string{
			"53-65 heavy torpedoes — high speed, deep magazine",
			"Aggressive ASW AI when DEFCON rises",
		},
		Defense: []string{
			"Soft-kill CM; nuclear mobility to reopen geometry",
			"No SAMs — still a submerged target for Mk48 only",
		},
		Neutralize: []string{
			"Best TMA you can get before committing Mk48.",
			"Expect counterfire; plan escape geometry before launch.",
		},
		Evade: []string{
			"He can outrun poor geometry — prioritize early CM + radical course/depth.",
			"Do not sprint needlessly; his sensors punish cavitation.",
		},
	},
	{
		ID: "merchant", Allegiance: libNeutral, Kind: world.KindSurfaceShip,
		Title:     "Merchant Freighter (MV)",
		ImageFile: "merchant.jpg",
		Credit:    "Photo: U.S. Navy / public domain (Wikimedia) — cargo ship stand-in",
		Summary: []string{
			"Civilian dry-cargo traffic. Broadband and shaft tonals without combat sensors or weapons. Useful for masking/bearing confusion, disastrous if you attack by mistake.",
			"Rules of engagement: classify before shoot — freighters are mission-failure risks if destroyed.",
		},
		Specs: []string{
			"Large merchant displacement (variable)",
			"Speed typically 12–18 kn",
			"Acoustic: steady shaft lines; blade ~1.4 Hz",
			"Radar: commercial X-band nav — ~24 rpm (≈2.5 s/scan)",
			"Mast detect (calm): ~4.5 kyd — still paints a raised stalk nearby",
		},
		Offense: []string{"None — unarmed civilian"},
		Defense: []string{"Nav radar only — no SAM/CIWS/ASW"},
		Neutralize: []string{
			"Do not engage. Reclassify if SPECTRUM looks combatant-like.",
		},
		Evade: []string{
			"Not a weapons threat. Avoid collision; use as acoustic clutter carefully.",
		},
	},
	{
		ID: "tanker", Allegiance: libNeutral, Kind: world.KindSurfaceShip,
		Title:     "Oil Tanker (VLCC / product)",
		ImageFile: "tanker.jpg",
		Credit:    "Photo: U.S. DoD / public domain (Wikimedia)",
		Summary: []string{
			"Large tanker with deep low-frequency plant noise. Extremely visible on broadband; not a combatant.",
			"Collateral damage and political cost of attack are catastrophic — never a valid weapons target in this mission.",
		},
		Specs: []string{
			"Very large displacement",
			"Speed typically 12–16 kn",
			"Acoustic: strong LF plant; blade ~1.1 Hz",
			"Radar: commercial S-band nav — ~20 rpm (≈3 s/scan)",
			"Mast detect (calm): ~5 kyd",
		},
		Offense: []string{"None — unarmed civilian"},
		Defense: []string{"Nav radar only"},
		Neutralize: []string{"Do not engage."},
		Evade: []string{"Collision avoidance only; may mask bearings if on same line of sound."},
	},
	{
		ID: "fishing", Allegiance: libNeutral, Kind: world.KindSurfaceShip,
		Title:     "Fishing Trawler (FV)",
		ImageFile: "fishing.jpg",
		Credit:    "Photo: Jebulon / CC BY-SA 3.0 (Wikimedia)",
		Summary: []string{
			"Small fishing vessel with higher blade rate and intermittent machinery. Easy to confuse with a patrol craft at long range until tonals clarify.",
			"Unarmed; classify carefully before any weapons solution.",
		},
		Specs: []string{
			"Small displacement",
			"Speed typically 8–12 kn",
			"Acoustic: higher blade ~2.4 Hz; leaner spectrum",
			"Radar: small-craft X-band nav — ~24 rpm (≈2.5 s/scan)",
			"Mast detect (calm): ~2.8 kyd",
		},
		Offense: []string{"None — unarmed civilian"},
		Defense: []string{"Nav radar only"},
		Neutralize: []string{"Do not engage."},
		Evade: []string{"Not a weapons threat; watch close-aboard collision risk in traffic."},
	},
	{
		ID: "los_angeles", Allegiance: libFriendly, Kind: world.KindSubmarine,
		Title:     "Los Angeles SSN — SSN-688(I) (Ownship)",
		ImageFile: "los_angeles.jpg",
		Credit:    "Photo: U.S. Navy / public domain (Wikimedia)",
		Summary: []string{
			"Your boat: Improved Los Angeles–class nuclear attack submarine. Quiet machinery set with sparse LOFAR lines relative to Soviet hunters, armed with Mk48 and Harpoon/ASROC loadouts from tubes.",
			"Protect ownship systems (sonar, propulsion, weapons) — damage cascades end the patrol.",
		},
		Specs: []string{
			"Displacement ~6,900 t submerged",
			"Length ~110 m  |  Speed 30+ kn submerged",
			"Tubes: 4 × 533 mm  |  Mk48 / Harpoon / ASROC (sim)",
			"Acoustic: quiet SSN; blade ~4.2 Hz harmonics",
			"ESM mast: raise only at ≤60 ft and ≤8 kn; shear if limits exceeded while up",
			"COMM mast: same limits; receive scheduled fleet traffic when raised",
			"Periscope: same depth/speed limits; train/zoom on MAST; raised optic is radar-detectable",
		},
		Offense: []string{
			"Mk48 ADCAP — primary ASW/ASuW fish",
			"UGM-84 Harpoon — anti-surface cruise missile",
			"ASROC (sim) — rocket-assisted ASW option where loaded",
			"ESM intercept of surface search/nav radars (MAST screen)",
			"COMM inbox: briefing at start; follow-on orders need antenna up",
		},
		Defense: []string{
			"ADC / jitter / Nixie soft-kill vs inbound fish",
			"Depth, speed, layer, and bathymetry are primary survival tools",
			"No organic SAM — do not expose to air/surface fire needlessly",
			"Raised ESM/COMM/periscope masts are radar-detectable; watch illumination bar",
		},
		Neutralize: []string{
			"N/A — this is ownship. Use loadout per threat class above.",
		},
		Evade: []string{
			"Standard torpedo evasion: CPA-driven CM, depth change, speed change.",
			"Manage self-noise; damaged arrays reduce your ability to fight.",
		},
	},
}

func libraryEntryByID(id string) *libraryEntry {
	for i := range libraryCatalog {
		if libraryCatalog[i].ID == id {
			return &libraryCatalog[i]
		}
	}
	return nil
}

func libraryAllegianceLabel(a libraryAllegiance) string {
	switch a {
	case libHostile:
		return "HOSTILE"
	case libNeutral:
		return "NEUTRAL"
	case libFriendly:
		return "FRIENDLY"
	default:
		return "UNKNOWN"
	}
}

// libraryTableRows builds non-selectable section headers + selectable platform rows.
// Within each allegiance: surface ships first, then submarines.
func libraryTableRows() []libraryTableRow {
	var out []libraryTableRow
	order := []libraryAllegiance{libHostile, libNeutral, libFriendly}
	for _, all := range order {
		var surfaces, subs []libraryEntry
		for _, e := range libraryCatalog {
			if e.Allegiance != all {
				continue
			}
			if e.Kind == world.KindSurfaceShip {
				surfaces = append(surfaces, e)
			} else if e.Kind == world.KindSubmarine {
				subs = append(subs, e)
			}
		}
		if len(surfaces)+len(subs) == 0 {
			continue
		}
		out = append(out, libraryTableRow{Header: true, Label: libraryAllegianceLabel(all)})
		for _, e := range surfaces {
			out = append(out, libraryTableRow{Label: e.Title, EntryID: e.ID})
		}
		for _, e := range subs {
			out = append(out, libraryTableRow{Label: e.Title, EntryID: e.ID})
		}
	}
	return out
}

type libraryTableRow struct {
	Header  bool
	Label   string
	EntryID string
}
