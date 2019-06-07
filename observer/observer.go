package observer

type Observer interface {
	OnStep(string)
	OnError(string, error)
	OnRead(int)
	OnEndSamples()
	OnEndAlarms()
}
