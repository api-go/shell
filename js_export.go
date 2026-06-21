package shell

import "apigo.cc/go/jsmod"

func init() {
	jsmod.Register("shell", map[string]any{
		"Run":        jsRun,
		"RunCommand": jsRunCommand,
	}, "Run", "RunCommand")
}

func jsRun(name string, args []string, opts *Options) (*CommandResult, error) {
	res, err := Run(name, args, opts)
	if err != nil {
		return nil, jsmod.MakeError(err)
	}
	return res, nil
}

func jsRunCommand(command string, opts *Options) (*CommandResult, error) {
	res, err := RunCommand(command, opts)
	if err != nil {
		return nil, jsmod.MakeError(err)
	}
	return res, nil
}
