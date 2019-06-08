package observer

import (
	"log"
)

type LoggerObserver struct{}

func (o LoggerObserver) OnStep(step string) {
	log.Println("[STEP]", step)
}

func (o LoggerObserver) OnError(step string, err error) {
	log.Println("[ERROR]", step, err)
}

func (o LoggerObserver) OnRead(size int) {
	log.Println("[READ]", size)
}

func (o LoggerObserver) OnEndSamples() {
	log.Println("[END] Samples")
}

func (o LoggerObserver) OnEndAlarms() {
	log.Println("[END] Alarms")
}
