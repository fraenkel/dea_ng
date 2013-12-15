package responders

type FakeResponder struct {
	Stopped    bool
	Advertised bool
}

func (fr *FakeResponder) Start() {
}
func (fr *FakeResponder) Stop() {
	fr.Stopped = true
}
func (fr *FakeResponder) Advertise() {
	fr.Advertised = true
}
