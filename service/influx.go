package service

import (
	"fmt"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/leaklessgfy/safran-server/entity"
	"github.com/leaklessgfy/safran-server/utils"
	uuid "github.com/satori/go.uuid"
)

// InfluxService is an higher abstraction layer between safran entities and influx db
type InfluxService struct {
	c client.Client
}

// NewInfluxService create a new InfluxService
func NewInfluxService() (*InfluxService, error) {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: "http://localhost:8086",
	})
	if err != nil {
		return nil, err
	}
	defer c.Close()

	_, _, err = c.Ping(0)
	if err != nil {
		return nil, err
	}

	query := client.NewQuery("CREATE DATABASE safran_db", "", "")
	response, err := c.Query(query)
	if err != nil {
		return nil, err
	}
	if response.Error() != nil {
		return nil, response.Error()
	}

	return &InfluxService{c}, nil
}

// InsertExperiment will insert an experiment into influx db
func (i InfluxService) InsertExperiment(experiment entity.Experiment) (string, error) {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return "", err
	}
	id, point, err := buildExperimentPoint(experiment)
	if err != nil {
		return "", err
	}
	batchPoints.AddPoint(point)

	err = i.c.Write(batchPoints)
	return id, err
}

// InsertMeasures will insert a bunch of measures into influx db
func (i InfluxService) InsertMeasures(experimentID string, measures []*entity.Measure) ([]string, error) {
	var measuresID []string

	batchPoints, err := buildBatchPoints()
	if err != nil {
		return nil, err
	}
	for _, measure := range measures {
		id, point, err := buildMeasurePoint(experimentID, measure)
		if err != nil {
			return nil, err
		}
		batchPoints.AddPoint(point)
		measuresID = append(measuresID, id)
	}

	err = i.c.Write(batchPoints)
	return measuresID, err
}

// InsertSamples will insert a bunch of samples into influx db
func (i InfluxService) InsertSamples(experimentID string, measuresID []string, experimentDate time.Time, samples []*entity.Sample) error {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return err
	}
	for _, sample := range samples {
		measureID := measuresID[sample.Measure]
		point, err := buildSamplePoint(experimentID, measureID, experimentDate, sample)
		if err != nil {
			return err
		}
		batchPoints.AddPoint(point)
	}
	return i.c.Write(batchPoints)
}

// InsertAlarms will insert a bunch of alarms into influx db
func (i InfluxService) InsertAlarms(experimentID string, experimentDate time.Time, alarms []*entity.Alarm) error {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return err
	}
	for _, alarm := range alarms {
		point, err := buildAlarmPoint(experimentID, experimentDate, alarm)
		if err != nil {
			return err
		}
		batchPoints.AddPoint(point)
	}
	return i.c.Write(batchPoints)
}

// RemoveExperiment will remove the specified experiment into influx db
func (i InfluxService) RemoveExperiment(experimentID string) error {
	var queries []client.Query

	query1 := client.NewQuery(fmt.Sprintf(`DELETE FROM experiments WHERE "id"='%s'`, experimentID), "safran_db", "")
	query2 := client.NewQuery(fmt.Sprintf(`DELETE FROM measures WHERE "experimentId"='%s'`, experimentID), "safran_db", "")
	query3 := client.NewQuery(fmt.Sprintf(`DELETE FROM samples WHERE "experimentId"='%s'`, experimentID), "safran_db", "")
	query4 := client.NewQuery(fmt.Sprintf(`DELETE FROM alarms WHERE "experimentId"='%s'`, experimentID), "safran_db", "")

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
		Database:  "safran_db",
		Precision: "ms",
	})
}

func buildExperimentPoint(experiment entity.Experiment) (string, *client.Point, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", nil, err
	}
	tags := map[string]string{
		"id": id.String(),
	}
	fiels := map[string]interface{}{
		"reference": experiment.Reference,
		"name":      experiment.Name,
		"bench":     experiment.Bench,
		"campaign":  experiment.Campaign,
		"isLocal":   false,
		"startDate": experiment.StartDate.UnixNano() / 1000000,
		"endDate":   experiment.EndDate.UnixNano() / 1000000,
	}
	p, err := client.NewPoint("experiments", tags, fiels, time.Now())
	return id.String(), p, err
}

func buildMeasurePoint(experimentID string, measure *entity.Measure) (string, *client.Point, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", nil, err
	}
	tags := map[string]string{
		"id":           id.String(),
		"experimentId": experimentID,
	}
	fiels := map[string]interface{}{
		"name": measure.Name,
		"type": measure.Typex,
		"unit": measure.Unitx,
	}
	p, err := client.NewPoint("measures", tags, fiels, time.Now())
	return id.String(), p, err
}

func buildSamplePoint(experimentID, measureID string, experimentDate time.Time, sample *entity.Sample) (*client.Point, error) {
	tags := map[string]string{
		"experimentId": experimentID,
		"measureId":    measureID,
	}
	fiels := map[string]interface{}{
		"value": sample.Value,
	}
	date, err := utils.ParseTime(sample.Time, experimentDate)
	if err != nil {
		return nil, err
	}
	return client.NewPoint("samples", tags, fiels, date)
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
