package observer

type ReportObserver struct {
}

func (o ReportObserver) OnStep(step string) {

}

func (o ReportObserver) OnError(step string, err error) {

}

func (o ReportObserver) OnRead(size int) {

}

func (o ReportObserver) OnEndSamples() {

}

func (o ReportObserver) OnEndAlarms() {

}
