package health_check_test

import (
	. "dea/health_check"
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"time"
)

var _ = Describe("StateFileReady", func() {
	var state_file_path string

	BeforeEach(func() {
		f, _ := ioutil.TempFile("", "state.json")
		f.Close()
		state_file_path = f.Name()
	})

	AfterEach(func() {
		os.RemoveAll(state_file_path)
	})

	It("fails if the file never exists", func() {
		Expect(run_state_file_ready(state_file_path, 25*time.Millisecond)).To(BeFalse())
	})

	It("fails if the file exists but the state is never 'RUNNING'", func() {
		write_state_file(state_file_path, "CRASHED")
		Expect(run_state_file_ready(state_file_path, 25*time.Millisecond)).To(BeFalse())
	})

	It("fails if the state file is corrupted", func() {
		ioutil.WriteFile(state_file_path, []byte("{{{"), 0755)
		Expect(run_state_file_ready(state_file_path, 25*time.Millisecond)).To(BeFalse())
	})

	It("succeeds if the file exists prior to starting the health check", func() {
		write_state_file(state_file_path, "RUNNING")
		Expect(run_state_file_ready(state_file_path, 25*time.Millisecond)).To(BeTrue())
	})

	It("succeeds if the file exists before the timeout", func() {
		time.AfterFunc(25*time.Millisecond, func() {
			write_state_file(state_file_path, "RUNNING")
		})

		Expect(run_state_file_ready(state_file_path, 100*time.Millisecond)).To(BeTrue())
	})

})

func run_state_file_ready(path string, timeout time.Duration) bool {
	hcc := newHCCallback()

	sfr := NewStateFileReady(path, 2*time.Millisecond, hcc, timeout)
	defer sfr.Destroy()
	return <-hcc.result
}

func write_state_file(path string, state string) {
	statemap := map[string]string{"state": state}
	bytes, err := json.Marshal(statemap)
	if err == nil {
		ioutil.WriteFile(path, bytes, 0755)
	}
}
