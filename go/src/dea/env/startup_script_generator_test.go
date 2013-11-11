package env_test

import (
	. "dea/env"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StartupScriptGenerator", func() {
	var start_command string
	var user_envs string
	var system_envs string
	var script string

	BeforeEach(func() {
		user_envs = "export usr1=\"usrval1\";\nexport usr2=\"usrval2\";\nunset unset_var;\n"
		system_envs = "export usr1=\"sys_user_val1\";\nexport sys1=\"sysval1\";\n"
		start_command = "go_nuts"
	})

	JustBeforeEach(func() {
		generator := NewStartupScriptGenerator(start_command, user_envs, system_envs)
		script = generator.Generate()
	})

	Describe("generate", func() {

		Describe("umask", func() {
			It("sets the umask to 077", func() {
				Expect(script).To(ContainSubstring("umask 077"))
			})
		})

		Describe("environment variables", func() {
			It("exports the user env variables", func() {
				Expect(script).To(ContainSubstring(user_envs))
			})
			It("exports the system env variables", func() {
				Expect(script).To(ContainSubstring(system_envs))
			})

			It("sources the buildpack env variables", func() {
				Expect(script).To(ContainSubstring("in app/.profile.d/*.sh"))
				Expect(script).To(ContainSubstring(". $i"))
			})

			It("exports user variables after system variables", func() {
				Expect(script).To(MatchRegexp("(?s)usr1=\"sys_user_val1\".*usr1=\"usrval1"))
			})

			It("exports build pack variables after system variables", func() {
				Expect(script).To(MatchRegexp("(?s)\"sysval1\".*\\.profile\\.d"))
			})

			It("sets user variables after buildpack variables", func() {
				Expect(script).To(MatchRegexp("(?s)\\.profile\\.d.*usrval1"))
			})

			It("print env to a log file after user envs", func() {
				Expect(script).To(ContainSubstring("env > logs/env.log"))
				Expect(script).To(MatchRegexp("(?s)usrval1.*env\\.log"))
			})

		})

	})

	Describe("starting app", func() {
		It("includes the start command in the starting script", func() {
			Expect(script).To(ContainSubstring(fmt.Sprintf(Start_SCRIPT, start_command)))
		})
	})

})
