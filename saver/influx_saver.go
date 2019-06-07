package saver

import (
	"fmt"
	"strconv"
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
	date         time.Time
}

func (s *InfluxSaver) SaveExperiment(experiment *entity.Experiment) error {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return err
	}
	id, point, err := buildExperimentPoint(experiment)
	if err != nil {
		return err
	}
	batchPoints.AddPoint(point)
	err = s.c.Write(batchPoints)
	if err != nil {
		return err
	}
	s.experimentID = id
	s.date = experiment.StartDate
	return nil
}

func (s InfluxSaver) SaveMeasures(measures []*entity.Measure) error {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return err
	}
	for _, measure := range measures {
		point, err := buildMeasurePoint(s.experimentID, measure)
		if err != nil {
			return err
		}
		batchPoints.AddPoint(point)
	}
	return s.c.Write(batchPoints)
}

func (s InfluxSaver) SaveSamples(samples []*entity.Sample) error {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return err
	}
	for _, sample := range samples {
		point, err := buildSamplePoint(s.experimentID, s.date, sample)
		if err != nil {
			return err
		}
		batchPoints.AddPoint(point)
	}
	return s.c.Write(batchPoints)
}

func (s InfluxSaver) SaveAlarms(alarms []*entity.Alarm) error {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return err
	}
	for _, alarm := range alarms {
		point, err := buildAlarmPoint(s.experimentID, s.date, alarm)
		if err != nil {
			return err
		}
		batchPoints.AddPoint(point)
	}
	return s.c.Write(batchPoints)
}

func (s InfluxSaver) Cancel() error {
	var queries []client.Query

	query1 := client.NewQuery(fmt.Sprintf(`DELETE FROM experiments WHERE "id"='%s'`, s.experimentID), DB, PRECISION)
	query2 := client.NewQuery(fmt.Sprintf(`DELETE FROM measures WHERE "experimentID"='%s'`, s.experimentID), DB, PRECISION)
	query3 := client.NewQuery(fmt.Sprintf(`DELETE FROM samples WHERE "experimentID"='%s'`, s.experimentID), DB, PRECISION)
	query4 := client.NewQuery(fmt.Sprintf(`DELETE FROM alarms WHERE "experimentID"='%s'`, s.experimentID), DB, PRECISION)

	queries = append(queries, query1)
	queries = append(queries, query2)
	queries = append(queries, query3)
	queries = append(queries, query4)

	for _, query := range queries {
		response, err := s.c.Query(query)
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

func buildMeasurePoint(experimentID string, measure *entity.Measure) (*client.Point, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
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
	return client.NewPoint("measures", tags, fields, time.Now())
}

func buildSamplePoint(experimentID string, experimentDate time.Time, sample *entity.Sample) (*client.Point, error) {
	tags := map[string]string{
		"experimentID": experimentID,
		"inc":          strconv.Itoa(sample.Inc),
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
