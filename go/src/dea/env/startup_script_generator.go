package env

import (
	"fmt"
	"strings"
)

const export_BUILDPACK_ENV_VARIABLES_SCRIPT = `
unset GEM_PATH
if [ -d app/.profile.d ]; then
	for i in app/.profile.d/*.sh; do
		if [ -r $i ]; then
			. $i
		fi
	done
	unset i
fi
`

const start_SCRIPT = `
DROPLET_BASE_DIR=$PWD
cd app
(%s) > >(tee $DROPLET_BASE_DIR/logs/stdout.log) 2> >(tee $DROPLET_BASE_DIR/logs/stderr.log >&2) &
STARTED=$!
echo "$STARTED" >> $DROPLET_BASE_DIR/run.pid

wait $STARTED
`

type StartupScriptGenerator struct {
	startCommand string
	userEnvs     string
	systemEnvs   string
}

func NewStartupScriptGenerator(start_command string, user_envs, system_envs string) StartupScriptGenerator {
	return StartupScriptGenerator{
		startCommand: start_command,
		userEnvs:     user_envs,
		systemEnvs:   system_envs,
	}
}

func (ssg StartupScriptGenerator) Generate() string {
	script := make([]string, 0, 7)
	script = append(script, "umask 077")
	script = append(script, ssg.systemEnvs)
	script = append(script, export_BUILDPACK_ENV_VARIABLES_SCRIPT)
	script = append(script, ssg.userEnvs)
	script = append(script, "env > logs/env.log")
	script = append(script, fmt.Sprintf(start_SCRIPT, ssg.startCommand))
	return strings.Join(script, "\n")
}
