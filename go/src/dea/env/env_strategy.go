package env

type Message interface {
	Name() string
	Index() int
	Version() string
	Uris() []string
	Limits() map[string]uint64
	MemoryLimit() uint64
	Services() []map[string]interface{}
	Environment() map[string]string
}

type EnvStrategy interface {
	ExportedSystemEnvironmentVariables() [][]string
	VcapApplication() map[string]interface{}
	Message() Message
}
