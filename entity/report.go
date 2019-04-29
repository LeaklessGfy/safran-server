package entity

const StatusFail = "failure"
const StatusSuccess = "success"
const StatusPending = "pending"

type Report struct {
	Status       string   `json:"status"`
	Progress     int      `json:"progress"`
	Errors       []string `json:"errors"`
	ExperimentID string   `json:"experimentID"`
	HasAlarms    bool     `json:"hasAlarms"`
}

func NewReport(err ...string) Report {
	return Report{
		Status:       StatusPending,
		Progress:     0,
		Errors:       err,
		ExperimentID: "",
		HasAlarms:    false,
	}
}

func CopyReport(report Report) Report {
	return Report{
		Status:       report.Status,
		Progress:     report.Progress,
		Errors:       report.Errors,
		ExperimentID: report.ExperimentID,
		HasAlarms:    report.HasAlarms,
	}
}

func (r *Report) AddError(err error) *Report {
	r.Status = StatusFail
	r.Errors = append(r.Errors, err.Error())
	return r
}
