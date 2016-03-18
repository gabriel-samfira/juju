// Copyright 2012-2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package runner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/loggo"
	utilexec "github.com/juju/utils/exec"

	"github.com/juju/juju/version"
	"github.com/juju/juju/worker/uniter/runner/debug"
	"github.com/juju/juju/worker/uniter/runner/jujuc"
)

// Runner is responsible for invoking commands in a context.
type Runner interface {

	// Context returns the context against which the runner executes.
	Context() Context

	// RunHook executes the hook with the supplied name.
	RunHook(name string) error

	// RunAction executes the action with the supplied name.
	RunAction(name string) error

	// RunCommands executes the supplied script.
	RunCommands(commands string) (*utilexec.ExecResponse, error)
}

// Context exposes jujuc.Context, and additional methods needed by Runner.
type Context interface {
	jujuc.Context
	Id() string
	HookVars(paths Paths) []string
	ActionData() (*ActionData, error)
	SetProcess(process *os.Process)
	HasExecutionSetUnitStatus() bool
	ResetExecutionSetUnitStatus()

	Prepare() error
	Flush(badge string, failure error) error
}

// Paths exposes the paths needed by Runner.
type Paths interface {

	// GetToolsDir returns the filesystem path to the dirctory containing
	// the hook tool symlinks.
	GetToolsDir() string

	// GetCharmDir returns the filesystem path to the directory in which
	// the charm is installed.
	GetCharmDir() string

	// GetJujucSocket returns the path to the socket used by the hook tools
	// to communicate back to the executing uniter process. It might be a
	// filesystem path, or it might be abstract.
	GetJujucSocket() string

	// GetMetricsSpoolDir returns the path to a metrics spool dir, used
	// to store metrics recorded during a single hook run.
	GetMetricsSpoolDir() string
}

// NewRunner returns a Runner backed by the supplied context and paths.
func NewRunner(context Context, paths Paths) Runner {
	return &runner{context, paths}
}

// runner implements Runner.
type runner struct {
	context Context
	paths   Paths
}

func (runner *runner) Context() Context {
	return runner.context
}

// RunCommands exists to satisfy the Runner interface.
func (runner *runner) RunCommands(commands string) (*utilexec.ExecResponse, error) {
	srv, err := runner.startJujucServer()
	if err != nil {
		return nil, err
	}
	defer srv.Close()

	env := runner.context.HookVars(runner.paths)
	command := utilexec.RunParams{
		Commands:    commands,
		WorkingDir:  runner.paths.GetCharmDir(),
		Environment: env,
	}

	err = command.Run()
	if err != nil {
		return nil, err
	}
	runner.context.SetProcess(command.Process())

	// Block and wait for process to finish
	result, err := command.Wait()
	return result, runner.context.Flush("run commands", err)
}

// RunAction exists to satisfy the Runner interface.
func (runner *runner) RunAction(actionName string) error {
	if _, err := runner.context.ActionData(); err != nil {
		return errors.Trace(err)
	}
	return runner.runCharmHookWithLocation(actionName, "actions")
}

// RunHook exists to satisfy the Runner interface.
func (runner *runner) RunHook(hookName string) error {
	return runner.runCharmHookWithLocation(hookName, "hooks")
}

func (runner *runner) runCharmHookWithLocation(hookName, charmLocation string) error {
	srv, err := runner.startJujucServer()
	if err != nil {
		return err
	}
	defer srv.Close()

	env := runner.context.HookVars(runner.paths)
	if version.Current.OS == version.Windows {
		// TODO(fwereade): somehow consolidate with utils/exec?
		// We don't do this on the other code path, which uses exec.RunCommands,
		// because that already has handling for windows environment requirements.
		env = mergeWindowsEnvironment(env, os.Environ())
	}

	debugctx := debug.NewHooksContext(runner.context.UnitName())
	if session, _ := debugctx.FindSession(); session != nil && session.MatchHook(hookName) {
		logger.Infof("executing %s via debug-hooks", hookName)
		err = session.RunHook(hookName, runner.paths.GetCharmDir(), env)
	} else {
		err = runner.runCharmHook(hookName, env, charmLocation)
	}
	return runner.context.Flush(hookName, err)
}

func (runner *runner) runCharmHook(hookName string, env []string, charmLocation string) (err error) {
	charmDir := runner.paths.GetCharmDir()
	var hook string
	hook, err = searchHook(charmDir, filepath.Join(charmLocation, hookName))
	if err != nil {
		if IsMissingHookError(err) {
			// Missing hook is perfectly valid, but worth mentioning.
			logger.Infof("skipped %q hook (not implemented)", hookName)
		}
		return
	}
	hookCmd := hookCommand(hook)
	ps := exec.Command(hookCmd[0], hookCmd[1:]...)
	ps.Env = env
	ps.Dir = charmDir
	outReader, outWriter, errW := os.Pipe()
	if errW != nil {
		return errors.Errorf("cannot make logging pipe: %v", err)
	}

	hackReader, hackWriter, errW := os.Pipe()
	if errW != nil {
		return errors.Errorf("cannot make logging pipe: %v", err)
	}
	hasError := false
	if version.IsWindowsNano() && strings.HasSuffix(hook, "ps1") {
		wrt := io.MultiWriter(outWriter, hackWriter)
		ps.Stdout = wrt
		ps.Stderr = wrt
		go func(hasErr *bool) {
			br := bufio.NewReaderSize(hackReader, 4096)
            *hasErr = true
			for {
				line, _, errRead := br.ReadLine()
				if errRead != nil {
					break
				}
				if strings.Contains(string(line), "Errors Count: 0") {
					*hasErr = false
					return
				}
			}
		}(&hasError)
	} else {
		ps.Stdout = outWriter
		ps.Stderr = outWriter
	}

	defer func() {
		if hasError {
			err = errors.Trace(fmt.Errorf("Hook Returned error"))
		}
	}()

	hookLogger := &hookLogger{
		r:      outReader,
		done:   make(chan struct{}),
		logger: runner.getLogger(hookName),
	}
	go hookLogger.run()

	err = ps.Start()
	if err == nil {
		// Record the *os.Process of the hook
		runner.context.SetProcess(ps.Process)
		// Block until execution finishes
		err = ps.Wait()
	}
	outWriter.Close()
	hackWriter.Close()
	hookLogger.stop()
	err = errors.Trace(err)
	return
}

func (runner *runner) startJujucServer() (*jujuc.Server, error) {
	// Prepare server.
	getCmd := func(ctxId, cmdName string) (cmd.Command, error) {
		if ctxId != runner.context.Id() {
			return nil, errors.Errorf("expected context id %q, got %q", runner.context.Id(), ctxId)
		}
		return jujuc.NewCommand(runner.context, cmdName)
	}
	srv, err := jujuc.NewServer(getCmd, runner.paths.GetJujucSocket())
	if err != nil {
		return nil, err
	}
	go srv.Run()
	return srv, nil
}

func (runner *runner) getLogger(hookName string) loggo.Logger {
	return loggo.GetLogger(fmt.Sprintf("unit.%s.%s", runner.context.UnitName(), hookName))
}
