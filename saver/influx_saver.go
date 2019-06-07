package saver

import (
	"fmt"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/leaklessgfy/safran-server/entity"
	"github.com/leaklessgfy/safran-server/utils"
	uuid "github.com/satori/go.uuid"
)

const (
	DB        = "safran_db"
	PRECISION = "ms"
)

type InfluxSaver struct {
	c            client.Client
	experimentID string
	measuresID   []string
	date         time.Time
}

func (i *InfluxSaver) SaveExperiment(experiment *entity.Experiment) error {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return err
	}
	id, point, err := buildExperimentPoint(experiment)
	if err != nil {
		return err
	}
	batchPoints.AddPoint(point)
	err = i.c.Write(batchPoints)
	if err != nil {
		return err
	}
	i.experimentID = id
	i.date = experiment.StartDate
	return nil
}

func (i *InfluxSaver) SaveMeasures(measures []*entity.Measure) error {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return err
	}
	var measuresID []string
	for _, measure := range measures {
		id, point, err := buildMeasurePoint(i.experimentID, measure)
		if err != nil {
			return err
		}
		batchPoints.AddPoint(point)
		measuresID = append(measuresID, id)
	}
	err = i.c.Write(batchPoints)
	if err != nil {
		return err
	}
	i.measuresID = measuresID
	return nil
}

func (i InfluxSaver) SaveSamples(samples []*entity.Sample) error {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return err
	}
	for _, sample := range samples {
		measureID := i.measuresID[sample.Measure]
		point, err := buildSamplePoint(i.experimentID, measureID, i.date, sample)
		if err != nil {
			return err
		}
		batchPoints.AddPoint(point)
	}
	// send those points to thread
	return nil
}

func (i InfluxSaver) SaveAlarms(alarms []*entity.Alarm) error {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return err
	}
	for _, alarm := range alarms {
		point, err := buildAlarmPoint(i.experimentID, i.date, alarm)
		if err != nil {
			return err
		}
		batchPoints.AddPoint(point)
	}
	// send those points to thread
	return nil
}

func (i InfluxSaver) Cancel() error {
	var queries []client.Query

	query1 := client.NewQuery(fmt.Sprintf(`DELETE FROM experiments WHERE "id"='%s'`, i.experimentID), DB, PRECISION)
	query2 := client.NewQuery(fmt.Sprintf(`DELETE FROM measures WHERE "experimentID"='%s'`, i.experimentID), DB, PRECISION)
	query3 := client.NewQuery(fmt.Sprintf(`DELETE FROM samples WHERE "experimentID"='%s'`, i.experimentID), DB, PRECISION)
	query4 := client.NewQuery(fmt.Sprintf(`DELETE FROM alarms WHERE "experimentID"='%s'`, i.experimentID), DB, PRECISION)

	queries = append(queries, query1)
	queries = append(queries, query2)
	queries = append(queries, query3)
	queries = append(queries, query4)

	for _, query := range queries {
		response, err := i.c.Query(query)
		if err != nil {
			return err
		}
		if response.Error() != nil {
			return response.Error()
		}
	}

	return nil
}

func buildBatchPoints() (client.BatchPoints, error) {
	return client.NewBatchPoints(client.BatchPointsConfig{
		Database:  DB,
		Precision: PRECISION,
	})
}

func buildExperimentPoint(experiment *entity.Experiment) (string, *client.Point, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", nil, err
	}
	tags := map[string]string{
		"id": id.String(),
	}
	fields := map[string]interface{}{
		"reference": experiment.Reference,
		"name":      experiment.Name,
		"bench":     experiment.Bench,
		"campaign":  experiment.Campaign,
		"startDate": experiment.StartDate,
		"endDate":   experiment.EndDate,
	}
	point, err := client.NewPoint("experiments", tags, fields, experiment.StartDate)
	return id.String(), point, err
}

func buildMeasurePoint(experimentID string, measure *entity.Measure) (string, *client.Point, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", nil, err
	}
	tags := map[string]string{
		"id":           id.String(),
		"experimentID": experimentID,
	}
	fields := map[string]interface{}{
		"name": measure.Name,
		"type": measure.Typex,
		"unit": measure.Unitx,
	}
	point, err := client.NewPoint("measures", tags, fields, time.Now())
	return id.String(), point, err
}

func buildSamplePoint(experimentID, measureID string, experimentDate time.Time, sample *entity.Sample) (*client.Point, error) {
	tags := map[string]string{
		"experimentID": experimentID,
		"measureID":    measureID,
	}
	fields := map[string]interface{}{
		"value": sample.Value,
	}
	date, err := utils.ParseTime(sample.Time, experimentDate)
	if err != nil {
		return nil, err
	}
	return client.NewPoint("samples", tags, fields, date)
}

func buildAlarmPoint(experimentID string, experimentDate time.Time, alarm *entity.Alarm) (*client.Point, error) {
	tags := map[string]string{
		"experimentID": experimentID,
	}
	fields := map[string]interface{}{
		"level":   alarm.Level,
		"message": alarm.Message,
	}
	date, err := utils.ParseTime(alarm.Time, experimentDate)
	if err != nil {
		return nil, err
	}
	return client.NewPoint("alarms", tags, fields, date)
}
