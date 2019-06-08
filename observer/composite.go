package observer

type CompositeObserver struct {
	observers []Observer
}

func (o CompositeObserver) OnStep(step string) {
	for _, observer := range o.observers {
		observer.OnStep(step)
	}
}

func (o CompositeObserver) OnError(step string, err error) {
	for _, observer := range o.observers {
		observer.OnError(step, err)
	}
}

func (o CompositeObserver) OnRead(size int) {
	for _, observer := range o.observers {
		observer.OnRead(size)
	}
}

func (o CompositeObserver) OnEndSamples() {
	for _, observer := range o.observers {
		observer.OnEndSamples()
	}
}

func (o CompositeObserver) OnEndAlarms() {
	for _, observer := range o.observers {
		observer.OnEndAlarms()
	}
}
