package entity

import "encoding/json"

const (
	ReportTypeExperiment = "Experiment"
	ReportTypeSamples    = "Samples"
	ReportTypeAlarms     = "Alarms"
	ReportTypeClient     = "Client"
)

const (
	ReportStatusFailure  = "failure"
	ReportStatusSuccess  = "success"
	ReportStatusProgress = "progress"
)

const (
	ReportStepInit = "1_INIT"

	ReportStepExtractExperiment = "2_EXTRACT_EXPERIMENT"
	ReportStepExtractSamples    = "3_EXTRACT_SAMPLES"
	ReportStepExtractAlarms     = "4_EXTRACT_ALARMS"

	ReportStepInitImport = "5_INIT_IMPORT"

	ReportStepParseHeader      = "6_PARSE_HEADER"
	ReportStepParseDate        = "7_PARSE_DATE"
	ReportStepInsertExperiment = "8_INSERT_EXPERIMENT"

	ReportStepParseMeasures  = "1_PARSE_MEASURES"
	ReportStepInsertMeasures = "2_INSERT_MEASURES"
	ReportStepParseSamples   = "3_PARSE_SAMPLES"
	ReportStepPrepareSamples = "4_PREPARE_SAMPLES_"
	ReportStepInsertSamples  = "5_INSERT_SAMPLES_"

	ReportStepParseAlarms   = "1_PARSE_ALARMS"
	ReportStepPrepareAlarms = "2_PREPARE_ALARMS_"
	ReportStepInsertAlarms  = "3_INSERT_ALARMS_"

	ReportStepInsertPoints     = "Y_INSERT_POINTS"
	ReportStepRemoveExperiment = "X_REMOVE_EXPERIMENT"
)

type Report struct {
	ID           int               `json:"id"`
	Channel      string            `json:"channel"`
	Type         string            `json:"type"`
	Status       string            `json:"status"`
	ExperimentID string            `json:"experimentID"`
	HasAlarms    bool              `json:"hasAlarms"`
	Progress     int               `json:"progress"`
	SamplesSize  int64             `json:"samplesSize"`
	AlarmsSize   int64             `json:"alarmsSize"`
	Read         int64             `json:"read"`
	Errors       map[string]string `json:"errors"`
	Steps        map[string]bool   `json:"steps"`
	Current      string            `json:"currentStep"`
}

func NewReport(channel string) *Report {
	errors := make(map[string]string)
	steps := make(map[string]bool)
	steps[ReportStepInit] = true

	return &Report{
		ID:           1,
		Channel:      channel,
		Type:         ReportTypeExperiment,
		Status:       ReportStatusProgress,
		ExperimentID: "",
		HasAlarms:    false,
		Progress:     0,
		SamplesSize:  0,
		AlarmsSize:   0,
		Errors:       errors,
		Steps:        steps,
		Current:      ReportStepInit,
	}
}

func (r Report) Copy(t string) *Report {
	errors := make(map[string]string)
	steps := make(map[string]bool)

	return &Report{
		ID:           1,
		Channel:      r.Channel,
		Type:         t,
		Status:       r.Status,
		ExperimentID: r.ExperimentID,
		HasAlarms:    r.HasAlarms,
		Progress:     r.Progress,
		SamplesSize:  r.SamplesSize,
		AlarmsSize:   r.AlarmsSize,
		Read:         0,
		Errors:       errors,
		Steps:        steps,
		Current:      r.Current,
	}
}

func (r *Report) AddSuccess(step string) *Report {
	r.Current = step
	r.Steps[step] = true
	return r
}

func (r *Report) AddError(step string, err error) *Report {
	r.Current = step
	r.Status = ReportStatusFailure
	r.Steps[step] = false
	r.Errors[step] = err.Error()
	return r
}

func (r *Report) Step() *Report {
	r.ID++
	return r
}

func (r *Report) AddRead(size int) *Report {
	r.Read += int64(size)
	r.Progress = int((r.Read * 100) / r.SamplesSize)
	return r
}

func (r *Report) End() {
	r.Status = ReportStatusSuccess
	r.Progress = 100
}

func (r Report) HasError() bool {
	return len(r.Errors) > 0
}

func (r Report) HasComplete() bool {
	return r.Status != ReportStatusProgress
}

func (r Report) ToJSON() []byte {
	b, err := json.Marshal(r)
	if err != nil {
		return []byte("{}")
	}
	return b
}
