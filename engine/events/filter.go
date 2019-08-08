package events

import (
	"github.com/globalsign/mgo/bson"
	"github.com/khyurri/speedlog/engine"
)

func GetEvents(req *Event, eng *engine.Engine) (events []Event, err error) {
	dbEngine := eng.DBEngine
	events = make([]Event, 0)
	// TODO: check req can be sent as request
	eng.Logger.Println(req.MetricTimeFrom)
	eng.Logger.Println(req.MetricTimeTo)
	err = dbEngine.Collection(collection).
		Find(bson.M{
			"project_id": req.ProjectId,
			"metric_time": bson.M{
				"$gte": req.MetricTimeFrom,
				"$lt":  req.MetricTimeTo,
			},
			"metric_name": req.MetricName,
		}).All(&events)
	return
}
