package directory_server

type FakeDirectoryServerV2 struct {
	Stopped bool
}

func (fds *FakeDirectoryServerV2) Stop() error {
	fds.Stopped = true
	return nil
}
