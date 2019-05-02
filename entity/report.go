package entity

const ReportStatusFailure = "failure"
const ReportStatusSuccess = "success"
const ReportStatusPending = "pending"

const (
	ReportStepInit = "1_INIT"

	ReportStepExtractExperiment = "2_EXTRACT_EXPERIMENT"
	ReportStepExtractSamples    = "3_EXTRACT_SAMPLES"
	ReportStepExtractAlarms     = "4_EXTRACT_ALARMS"

	ReportStepInitImport = "5_INIT_IMPORT"

	ReportStepParseHeader      = "6_PARSE_HEADER"
	ReportStepParseDate        = "7_PARSE_DATE"
	ReportStepInsertExperiment = "8_INSERT_EXPERIMENT"

	ReportStepParseMeasures  = "9.1.1_PARSE_MEASURES"
	ReportStepInsertMeasures = "9.1.2_INSERT_MEASURES"

	ReportStepParseSamples  = "9.1.3_PARSE_SAMPLES"
	ReportStepInsertSamples = "9.1.4_INSERT_SAMPLES"

	ReportStepParseAlarms  = "9.2.1_PARSE_ALARMS"
	ReportStepInsertAlarms = "9.2.2_INSERT_ALARMS"

	ReportStepRemoveExperiment = "X_REMOVE_EXPERIMENT"
)

type Report struct {
	ID           int               `json:"id"`
	Channel      string            `json:"channel"`
	Title        string            `json:"title"`
	Status       string            `json:"status"`
	ExperimentID string            `json:"experimentID"`
	HasAlarms    bool              `json:"hasAlarms"`
	Progress     int               `json:"progress"`
	SamplesSize  int64             `json:"samplesSize"`
	AlarmsSize   int64             `json:"alarmsSize"`
	SamplesRead  int64             `json:"samplesRead"`
	AlarmsRead   int64             `json:"alarmsRead"`
	Errors       map[string]string `json:"errors"`
	Steps        map[string]bool   `json:"steps"`
	Current      string            `json:"currentStep"`
}

func NewReport(channel, title string) *Report {
	steps := make(map[string]bool)
	steps[ReportStepInit] = true
	errors := make(map[string]string)
	return &Report{
		ID:           0,
		Channel:      channel,
		Title:        title,
		Status:       ReportStatusPending,
		ExperimentID: "",
		HasAlarms:    false,
		Progress:     0,
		Errors:       errors,
		Steps:        steps,
		Current:      ReportStepInit,
	}
}

func (r *Report) AddSuccess(step string) *Report {
	r.ID++
	r.Steps[step] = true
	r.Current = step
	return r
}

func (r *Report) AddError(step string, err error) *Report {
	r.ID++
	r.Status = ReportStatusFailure
	r.Steps[step] = false
	r.Errors[step] = err.Error()
	r.Current = step
	return r
}

func (r *Report) Step() {
	r.ID++
}

func (r *Report) ReadSamples(size int) {
	r.SamplesRead += int64(size)
	r.Progress = int((r.SamplesRead * 100) / r.SamplesSize)
}

func (r *Report) ReadAlarms(size int) {
	r.AlarmsRead += int64(size)
	r.Progress = int((r.AlarmsRead * 100) / r.AlarmsSize)
}

func (r Report) HasError() bool {
	return len(r.Errors) > 0
}

func (r Report) HasComplete() bool {
	return r.Status == ReportStatusSuccess || r.Status == ReportStatusSuccess
}
