package shell

import "apigo.cc/go/jsmod"

func init() {
	jsmod.Register("shell", map[string]any{
		"Run":        Run,
		"RunCommand": RunCommand,
	}, "Run", "RunCommand")
}
