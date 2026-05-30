package shell

import "apigo.cc/go/jsmod"

func init() {
	jsmod.Register("shell", map[string]any{
		"run":        Run,
		"runCommand": RunCommand,
	}, "run", "runCommand")
}
