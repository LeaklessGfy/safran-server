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

	influx := &InfluxService{c}

	return influx, influx.Ping()
}

func (i InfluxService) Ping() error {
	_, _, err := i.c.Ping(0)
	if err != nil {
		return err
	}
	return i.Install()
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

// PrepareSamples will create batch points for samples
func (i InfluxService) PrepareSamples(experimentID string, measuresID []string, experimentDate time.Time, samples []*entity.Sample) (client.BatchPoints, error) {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return batchPoints, err
	}
	for _, sample := range samples {
		measureID := measuresID[sample.Measure]
		point, err := buildSamplePoint(experimentID, measureID, experimentDate, sample)
		if err != nil {
			return batchPoints, err
		}
		batchPoints.AddPoint(point)
	}
	return batchPoints, nil
}

// PrepareAlarms will create batch points for alarms
func (i InfluxService) PrepareAlarms(experimentID string, experimentDate time.Time, alarms []*entity.Alarm) (client.BatchPoints, error) {
	batchPoints, err := buildBatchPoints()
	if err != nil {
		return batchPoints, err
	}
	for _, alarm := range alarms {
		point, err := buildAlarmPoint(experimentID, experimentDate, alarm)
		if err != nil {
			return batchPoints, err
		}
		batchPoints.AddPoint(point)
	}
	return batchPoints, nil
}

// RemoveExperiment will remove the specified experiment into influx db
func (i InfluxService) RemoveExperiment(experimentID string) error {
	var queries []client.Query

	query1 := client.NewQuery(fmt.Sprintf(`DELETE FROM experiments WHERE "id"='%s'`, experimentID), "safran_db", "")
	query2 := client.NewQuery(fmt.Sprintf(`DELETE FROM measures WHERE "experimentID"='%s'`, experimentID), "safran_db", "")
	query3 := client.NewQuery(fmt.Sprintf(`DELETE FROM samples WHERE "experimentID"='%s'`, experimentID), "safran_db", "")
	query4 := client.NewQuery(fmt.Sprintf(`DELETE FROM alarms WHERE "experimentID"='%s'`, experimentID), "safran_db", "")

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

func (i InfluxService) InsertBatchPoints(batchPoints client.BatchPoints) error {
	return i.c.Write(batchPoints)
}

func (i InfluxService) Size() (string, error) {
	var queries []client.Query

	queries = append(queries, client.NewQuery("SELECT count(*) FROM experiments", "safran_db", ""))
	queries = append(queries, client.NewQuery("SELECT count(*) FROM measures", "safran_db", ""))
	queries = append(queries, client.NewQuery("SELECT count(*) FROM samples", "safran_db", ""))
	queries = append(queries, client.NewQuery("SELECT count(*) FROM alarms", "safran_db", ""))

	total := ""
	for _, query := range queries {
		response, err := i.c.Query(query)
		if err != nil {
			return "", err
		}
		if response.Error() != nil {
			return "", response.Error()
		}
		for _, result := range response.Results {
			total += "Messages :\n"
			for _, msg := range result.Messages {
				total += msg.Level + " " + msg.Text
			}
			total += "Series : \n"
			for _, row := range result.Series {
				total += fmt.Sprintf("%s", row.Values)
			}
		}
	}

	return total, nil
}

func (i InfluxService) Install() error {
	query := client.NewQuery("CREATE DATABASE safran_db", "", "")
	response, err := i.c.Query(query)
	if err != nil {
		return err
	}
	if response.Error() != nil {
		return response.Error()
	}
	return nil
}

func (i InfluxService) Drop() error {
	query := client.NewQuery(`DROP DATABASE "safran_db"`, "", "")
	response, err := i.c.Query(query)
	if err != nil {
		return err
	}
	if response.Error() != nil {
		return response.Error()
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
		"startTime": experiment.StartTime,
		"endDate":   experiment.EndTime,
	}
	p, err := client.NewPoint("experiments", tags, fiels, experiment.Date)
	return id.String(), p, err
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
		"experimentID": experimentID,
		"measureID":    measureID,
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
