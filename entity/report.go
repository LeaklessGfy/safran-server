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
	ReportStepParseDate        = "7_ARSE_DATE"
	ReportStepInsertExperiment = "8_INSERT_EXPERIMENT"

	ReportStepParseMeasures  = "9.1.1_PARSE_MEASURES"
	ReportStepInsertMeasures = "9.1.2_INSERT_MEASURES"

	ReportStepParseSamples  = "9.1.3_PARSE_SAMPLES"
	ReportStepInsertSamples = "9.1.4_INSERT_SAMPLES"

	ReportStepParseAlarms  = "9.1.1_PARSE_ALARMS"
	ReportStepInsertAlarms = "9.1.2_INSERT_ALARMS"

	ReportStepRemoveExperiment = "X_REMOVE_EXPERIMENT"
)

type Report struct {
	ID           int               `json:"id"`
	Title        string            `json:"title"`
	Status       string            `json:"status"`
	ExperimentID string            `json:"experimentID"`
	HasAlarms    bool              `json:"hasAlarms"`
	Progress     int               `json:"progress"`
	Errors       map[string]string `json:"errors"`
	Steps        map[string]bool   `json:"steps"`
}

func NewReport(title string) *Report {
	steps := make(map[string]bool)
	steps[ReportStepInit] = true
	errors := make(map[string]string)
	return &Report{
		ID:           0,
		Title:        title,
		Status:       ReportStatusPending,
		ExperimentID: "",
		HasAlarms:    false,
		Progress:     0,
		Errors:       errors,
		Steps:        steps,
	}
}

func (r *Report) AddSuccess(step string) *Report {
	r.ID++
	r.Steps[step] = true
	return r
}

func (r *Report) AddError(step string, err error) *Report {
	r.ID++
	r.Status = ReportStatusFailure
	r.Steps[step] = false
	r.Errors[step] = err.Error()
	return r
}

func (r Report) HasError() bool {
	return len(r.Errors) > 0
}

func (r Report) HasComplete() bool {
	return r.Status == ReportStatusSuccess || r.Status == ReportStatusSuccess
}
