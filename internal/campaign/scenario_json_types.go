package campaign

// JSON document types for portable scenario files (format major 2).

type scenarioFile struct {
	FormatVersion  string `json:"format_version"`
	Version        string `json:"version"`
	MinGameVersion string `json:"min_game_version"`
	ID             string `json:"id"`
	Title          string `json:"title"`
	Backstory      string `json:"backstory"`
	Cover          *assetBlobJSON `json:"cover,omitempty"`
	PostscriptSuccess string `json:"postscript_success,omitempty"`
	PostscriptFailure string `json:"postscript_failure,omitempty"`
	Theaters       []theaterJSON  `json:"theaters"`
	Missions       []missionJSON  `json:"missions"`
	Events         []EventDef     `json:"events,omitempty"`
}

type assetBlobJSON struct {
	Mime      string `json:"mime,omitempty"`
	DataB64    string `json:"data_b64,omitempty"`
	AssetRef   string `json:"asset_ref,omitempty"` // rejected; use data_b64
	UseDefault bool   `json:"use_game_default,omitempty"`
}

type theaterJSON struct {
	ID    string         `json:"id"`
	Bathy *assetBlobJSON `json:"bathy,omitempty"`
}

type missionJSON struct {
	ID           string              `json:"id"`
	Title        string              `json:"title"`
	Description  string              `json:"description"`
	Cover        *assetBlobJSON      `json:"cover,omitempty"`
	TheaterID    string              `json:"theater_id"`
	Routes       []routeJSON         `json:"routes"`
	Player       unitJSON            `json:"player"`
	Units        []unitJSON          `json:"units"`
	CommBriefing string              `json:"comm_briefing,omitempty"`
	CommSchedule []commMsgJSON       `json:"comm_schedule,omitempty"`
	StartTime    string              `json:"start_time,omitempty"` // HH:MM 24h wall clock
	Objectives   []objectiveJSON     `json:"objectives"`
	Outputs      []outputJSON        `json:"outputs,omitempty"`
	DebriefLead  string              `json:"debrief_lead,omitempty"`
	DebriefLines []debriefLineJSON   `json:"debrief_lines,omitempty"`
	Events       []EventDef          `json:"events,omitempty"`
}

type routeJSON struct {
	ID              string         `json:"id"`
	Mode            string         `json:"mode"`
	Waypoints       []waypointJSON `json:"waypoints"`
	PlayerClearance bool           `json:"player_clearance,omitempty"`
}

type waypointJSON struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type unitJSON struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	Side           string  `json:"side"`
	SignatureID    string  `json:"signature_id"`
	LengthFt       float64 `json:"length_ft,omitempty"`
	SpeedKts       float64 `json:"speed_kts,omitempty"`
	DepthFt        float64 `json:"depth_ft,omitempty"`
	DepthJitter    float64 `json:"depth_jitter,omitempty"`
	HeadingDeg     float64 `json:"heading_deg,omitempty"`
	AIState        string  `json:"ai_state,omitempty"`
	Defcon         int     `json:"defcon,omitempty"`
	CrewSkill      float64 `json:"crew_skill,omitempty"`
	CrewJitter     float64 `json:"crew_jitter,omitempty"`
	Combatant      bool    `json:"combatant,omitempty"`
	Spawn          string  `json:"spawn"`
	Corner         string  `json:"corner,omitempty"`
	MinRouteYd     float64 `json:"min_route_yd,omitempty"`
	MaxRouteYd     float64 `json:"max_route_yd,omitempty"`
	RouteID        string  `json:"route_id,omitempty"`
	RouteFrac      float64 `json:"route_frac,omitempty"`
	FallbackCorner string  `json:"fallback_corner,omitempty"`
	FallbackMinYd  float64 `json:"fallback_min_yd,omitempty"`
	FallbackMaxYd  float64 `json:"fallback_max_yd,omitempty"`
	RequireVar     string  `json:"require_var,omitempty"`
	UnlessVar      string  `json:"unless_var,omitempty"`
	AllyIgnore     bool    `json:"ally_ignore,omitempty"`
	Payload        *unitPayloadJSON `json:"payload,omitempty"`
}

type unitPayloadJSON struct {
	Torpedoes  *int `json:"torpedoes,omitempty"`
	ASWRockets *int `json:"asw_rockets,omitempty"`
	ShipTubes  *int `json:"ship_tubes,omitempty"`
	RBU        *int `json:"rbu,omitempty"`
	SAM        *int `json:"sam,omitempty"`
	CIWS       *int `json:"ciws,omitempty"`
}

type objectiveJSON struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	TargetID     string `json:"target_id"`
	Primary      bool   `json:"primary,omitempty"`
	NeedIdentify bool   `json:"need_identify,omitempty"`
	NeedDestroy  bool   `json:"need_destroy,omitempty"`
	Hidden       bool   `json:"hidden,omitempty"`
	RequireVar   string `json:"require_var,omitempty"`
	UnlessVar    string `json:"unless_var,omitempty"`
}

type outputJSON struct {
	Key                 string `json:"key"`
	Value               string `json:"value"`
	WhenPrimaryComplete bool   `json:"when_primary_complete,omitempty"`
	WhenObjectiveID     string `json:"when_objective_id,omitempty"`
}

type debriefLineJSON struct {
	ObjectiveID string `json:"objective_id"`
	OnSuccess   string `json:"on_success"`
	OnFail      string `json:"on_fail"`
}

type commMsgJSON struct {
	ID    string  `json:"id"`
	AtSec float64 `json:"at_sec"`
	Text  string  `json:"text"`
}
