package entity

import "encoding/json"

const (
	TypeExperiment = "Experiment"
	TypeSamples    = "Samples"
	TypeAlarms     = "Alarms"
	TypeClient     = "Client"
)

const (
	StatusFailure  = "failure"
	StatusSuccess  = "success"
	StatusProgress = "progress"
)

const (
	StepInit = "1_INIT"

	StepExtractExperiment = "2_EXTRACT_EXPERIMENT"
	StepExtractSamples    = "3.1_EXTRACT_SAMPLES"
	StepExtractAlarms     = "3.2_EXTRACT_ALARMS"

	StepInitImport = "4_INIT_IMPORT"

	StepParseHeader    = "5_PARSE_HEADER"
	StepParseStartDate = "6.1_PARSE_START_DATE"
	StepParseEndDate   = "6.2_PARSE_END_DATE"
	StepSaveExperiment = "7_SAVE_EXPERIMENT"

	StepParseMeasures = "8.1.1_PARSE_MEASURES"
	StepSaveMeasures  = "8.1.2_SAVE_MEASURES"
	StepParseSamples  = "8.1.3_PARSE_SAMPLES_"
	StepSaveSamples   = "8.1.4_SAVE_SAMPLES_"

	StepParseAlarms = "8.2.1_PARSE_ALARMS_"
	StepSaveAlarms  = "8.2.2_SAVE_ALARMS_"

	StepFullEnd = "9_END"

	StepInsertPoints = "Y_INSERT_POINTS"
	StepCancel       = "X_CANCEL"
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
	steps[StepInit] = true

	return &Report{
		ID:           1,
		Channel:      channel,
		Type:         TypeExperiment,
		Status:       StatusProgress,
		ExperimentID: "",
		HasAlarms:    false,
		Progress:     0,
		SamplesSize:  0,
		AlarmsSize:   0,
		Errors:       errors,
		Steps:        steps,
		Current:      StepInit,
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
	r.Status = StatusFailure
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
	r.Status = StatusSuccess
	r.Progress = 100
}

func (r Report) HasError() bool {
	return len(r.Errors) > 0
}

func (r Report) HasComplete() bool {
	return r.Status != StatusProgress
}

func (r Report) ToJSON() []byte {
	b, err := json.Marshal(r)
	if err != nil {
		return []byte("{}")
	}
	return b
}
