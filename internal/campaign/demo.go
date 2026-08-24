package campaign

import (
	"github.com/ssn688/sim/internal/world"
)

const DemoMissionTraining MissionID = "catalina_training"

// DemoScenario is the introductory campaign with the existing training mission.
func DemoScenario() ScenarioDef {
	return ScenarioDef{
		ID:    DemoScenarioID,
		Title: "Shadows off Catalina",
		Backstory: "Pacific Fleet operations order 44-7 places USS Los Angeles (SSN-688) on a covert " +
			"barrier patrol south of Santa Catalina Island. COMSUBPAC warns of a Foxtrot-class diesel " +
			"prowling merchant lanes and a Grisha corvette running inshore ASW sweeps. Allied Spruance " +
			"and a sister 688 boat patrol the eastern edge — identify before you shoot.\n\n" +
			"Maintain acoustic discretion until tasked. Neutral merchant traffic is dense; ROE requires " +
			"positive identification of all contacts before weapons release.",
		CoverFile: "scenarios/demo_cover.jpg",
		PostscriptSuccess: "COMSUBPAC acknowledges your report. Hostile units neutralized without " +
			"civilian losses. Los Angeles is released to continue the Catalina barrier while follow-on " +
			"forces sanitize the area.",
		PostscriptFailure: "COMSUBPAC notes mission failure. Whether through loss of the boat or breach " +
			"of orders, the Catalina barrier is compromised. A relief SSN is being routed to your datum.",
		Missions: []MissionDef{
			{
				ID:    DemoMissionTraining,
				Title: "Santa Catalina Approaches",
				Description: "Establish covert patrol in assigned OP AREA south of Santa Catalina Island. " +
					"Intelligence indicates diesel submarine activity and inshore ASW surface units along " +
					"merchant transit lanes. Expect mixed neutral shipping — merchants, tankers, and " +
					"trawlers — with possible hostile contacts among routine traffic.\n\n" +
					"Allied forces hold the eastern patrol belt; maintain identification discipline. " +
					"Remain undetected until follow-on tasking via COMM. When directed, execute assigned " +
					"objectives and transmit a periscope-depth REPORT before ending the mission.",
				Build: buildDemoTrainingMission,
				Objectives: []ObjectiveTemplate{
					{ID: "obj_foxtrot", Description: "Locate and sink hostile diesel submarine", TargetID: "enemy_foxtrot", Primary: true, NeedDestroy: true},
					{ID: "obj_grisha", Description: "Identify and sink hostile surface combatant", TargetID: "enemy_grisha", Primary: true, NeedIdentify: true, NeedDestroy: true},
					{ID: "obj_tanker", Description: "Locate and identify tanker (do not engage)", TargetID: "civ_tanker", Primary: false, NeedIdentify: true},
				},
				Outputs: []OutputRule{
					{Key: "foxrot_neutralized", Value: "true", WhenPrimaryComplete: true},
					{Key: "grisha_neutralized", Value: "true", WhenPrimaryComplete: true},
				},
			},
		},
	}
}

func buildDemoTrainingMission(_ BuildContext) *world.Scenario {
	return world.NewTrainingScenario()
}
