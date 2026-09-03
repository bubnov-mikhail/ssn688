//go:build ignore
package main
import (
	"fmt"
	"os"
	"github.com/bubnov-mikhail/ssn688/internal/campaign"
	"github.com/bubnov-mikhail/ssn688/internal/world"
)
func main() {
	data,_:=os.ReadFile("scenarios_generated/taiwan_formosa_watch.json")
	sc,_:=campaign.ParseScenarioJSON(data,"t")
	var m *campaign.MissionDef
	for i:=range sc.Missions{if sc.Missions[i].ID=="tw_twin_exercises"{m=&sc.Missions[i];break}}
	routes,_:=campaign.RuntimeRoutes(m.Routes)
	var r *world.Route
	for _,rt:=range routes{if rt.ID=="route_rf_shadow"{r=rt;break}}
	fmt.Printf("loop=%v pingpong=%v n=%d unique=%d\n",r.Looped,r.PingPong,len(r.Waypoints),r.UniqueCount())
	for i,wp:=range r.Waypoints{fmt.Printf("  wp%d %.0f,%.0f\n",i,wp.X,wp.Y)}
	e:=&world.Entity{Kind:world.KindSubmarine,DepthFt:60}
	bathy:=sc.Theaters[0].Chart
	for _,frac:=range []float64{0,0.12,0.5}{
		ent:=&world.Entity{Kind:world.KindSubmarine,DepthFt:60}
		ok:=world.PlaceOnRouteFraction(ent,r,frac,bathy)
		fmt.Printf("frac %.2f ok=%v pos %.0f,%.0f\n",frac,ok,ent.X,ent.Y)
	}
	_ = e
}
