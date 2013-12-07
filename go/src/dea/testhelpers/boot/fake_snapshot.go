package boot

type FakeSnapshot struct {
	LoadInvoked bool
	SaveInvoked bool
}

func (fs *FakeSnapshot) Path() string {
	return "fakepath"
}
func (fs *FakeSnapshot) Load() error {
	fs.LoadInvoked = true
	return nil
}

func (fs *FakeSnapshot) Save() {
	fs.SaveInvoked = true
}
