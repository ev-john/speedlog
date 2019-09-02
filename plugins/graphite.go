package plugins

import (
	"fmt"
	"github.com/khyurri/speedlog/engine/mongo"
	"net"
	"time"
)

type graphite struct {
	host     string
	ticker   *time.Ticker
	location *time.Location
}

func NewGraphite(host string, location *time.Location) *graphite {
	return &graphite{host: host, location: location}
}

func gPath(project, event, metric string) string {
	return fmt.Sprintf("speedlog.tests.%s.%s.%s", project, event, metric)
}

func (gr *graphite) Load(dbEngine mongo.DataStore) {
	interval := 1 * time.Minute
	rng := 1 * time.Minute
	fmt.Println("[debug] graphite module loaded")
	gr.ticker = time.NewTicker(interval)
	go func() {
		fmt.Println("[debug] graphite module started")
		var dateFrom, dateTo time.Time
		now := time.Now()
		// Each circle dateFrom takes the value of dateTo, and dateTo increases by rng
		for _ = range gr.ticker.C {
			if dateTo.IsZero() {
				dateTo = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, gr.location)
				dateFrom = dateTo
			}
			// increase by rng
			dateTo = dateFrom.Add(rng)
			events, err := dbEngine.AllEvents(dateFrom, dateTo)
			if err != nil {
				fmt.Printf("[error] fetching events %v", err)
			}

			fmt.Printf("[debug] Fetching from: %s, to: %s\n", dateFrom, dateTo)
			fmt.Println(len(events))

			for _, group := range events {
				aggregatedEvents, _ := mongo.GroupBy("minutes", group.Events)

				for _, event := range aggregatedEvents {
					name := event.MetricName
					// TODO: fix it
					fmt.Println(group.Meta.ProjectId)
					proj := dbEngine.GetProjectById(group.Meta.ProjectId).Title
					sendMap := map[string]interface{}{
						gPath(proj, name, "median"): event.MedianDurationMs,
						gPath(proj, name, "max"):    event.MaxDurationMs,
						gPath(proj, name, "min"):    event.MinDurationMs,
					}
					sendDataToGraphite(gr.host, sendMap)
					fmt.Printf("[debug] sended")
				}
			}
			dateFrom = dateTo
		}
	}()
}

func (gr *graphite) Unload() {
	gr.ticker.Stop()
}

func sendDataToGraphite(host string, data map[string]interface{}) {
	conn, err := net.Dial("tcp", host)
	now := time.Now().Unix()
	if err != nil {
		fmt.Printf("[error] connect to graphite: %v", err)
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			fmt.Printf("[error] close graphite connection: %v", err)
		}
	}()
	for k, v := range data {
		switch tp := v.(type) {
		case int:
			n, err := conn.Write([]byte(fmt.Sprintf("%s %d %d\r\n\r\n", k, tp, now)))
			if err != nil {
				fmt.Printf("[error] error sending data %v\n", err)
			}
			fmt.Printf("[debug] wrote %d bytes\n", n)
		case float64:
			n, err := conn.Write([]byte(fmt.Sprintf("%s %f %d\r\n\r\n", k, tp, now)))
			if err != nil {
				fmt.Printf("[error] error sending data %v\n", err)
			}
			fmt.Printf("[debug] wrote %d bytes\n", n)
		default:
			fmt.Printf("[error] unsopported type %s\n", tp)
		}
	}
}
