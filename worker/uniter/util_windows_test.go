// Copyright 2012-2014 Canonical Ltd.
// Copyright 2014 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

// +build windows

package uniter_test

import (
	"fmt"
	"path/filepath"

	gc "gopkg.in/check.v1"
)

var cmdSuffix = ".cmd"

var goodHook = `
juju-log.exe %%JUJU_ENV_UUID%% %s %%JUJU_REMOTE_UNIT%%
`[1:]

var badHook = `
#!/bin/bash --norc
juju-log.exe %%JUJU_ENV_UUID%% fail-%s %%JUJU_REMOTE_UNIT%%
exit 1
`[1:]

var rebootHook = `
juju-reboot.exe
`[1:]

var badRebootHook = `
juju-reboot.exe
exit 1
`[1:]

var rebootNowHook = `
if EXIST i_have_risen (
	exit 0
) else (
	echo a > i_have_risen && juju-reboot.exe --now
)
`[1:]

var actions = map[string]string{
	"action-log": `
juju-log.exe %%JUJU_ENV_UUID%% action-log
`[1:],
	"snapshot": `
action-set.exe outfile.name="snapshot-01.tar" outfile.size="10.3GB"
action-set.exe outfile.size.magnitude="10.3" outfile.size.units="GB"
action-set.exe completion.status="yes" completion.time="5m"
action-set.exe completion="yes"
`[1:],
	"action-log-fail": `
action-fail.exe "I'm afraid I can't let you do that, Dave."
action-set.exe foo="still works"
`[1:],
	"action-log-fail-error": `
action-fail.exe too many arguments
action-set.exe foo="still works"
action-fail.exe "A real message"
`[1:],
	"action-reboot": `
juju-reboot.exe || action-set.exe reboot-delayed="good"
juju-reboot.exe --now || action-set.exe reboot-now="good"
`[1:],
}

var appendConfigChanged = "config-get.exe --format yaml --output config.out"

func echoUnitNameToFileHelper(testDir, name string) string {
	path := filepath.Join(testDir, name)
	template := `Set-Content %s "juju run $env:JUJU_UNIT_NAME"`
	return fmt.Sprintf(template, path)
}

func (s *UniterSuite) TestRunCommand(c *gc.C) {
	testDir := c.MkDir()
	testFile := func(name string) string {
		return filepath.Join(testDir, name)
	}
	adminTag := s.AdminUserTag(c)
	echoUnitNameToFile := func(name string) string {
		return echoUnitNameToFileHelper(testDir, name)
	}

	s.runUniterTests(c, []uniterTest{
		ut(
			"run commands: environment",
			quickStart{},
			runCommands{echoUnitNameToFile("run.output")},
			verifyFile{filepath.Join(testDir, "run.output"), "juju run u/0\n"},
		), ut(
			"run commands: jujuc commands",
			quickStartRelation{},
			runCommands{
				fmt.Sprintf("Set-Content %s $(owner-get tag)", testFile("jujuc.output")),
				fmt.Sprintf("Add-Content %s $(unit-get private-address)", testFile("jujuc.output")),
				fmt.Sprintf("Add-Content %s $(unit-get public-address)", testFile("jujuc.output")),
			},
			verifyFile{
				testFile("jujuc.output"),
				adminTag.String() + "\nprivate.address.example.com\npublic.address.example.com\n",
			},
		), ut(
			"run commands: jujuc environment",
			quickStartRelation{},
			relationRunCommands{
				fmt.Sprintf("Set-Content %s $env:JUJU_RELATION_ID", testFile("jujuc-env.output")),
				fmt.Sprintf("Add-Content %s $env:JUJU_REMOTE_UNIT", testFile("jujuc-env.output")),
			},
			verifyFile{
				testFile("jujuc-env.output"),
				"db:0\nmysql/0\n",
			},
		), ut(
			"run commands: proxy settings set",
			quickStartRelation{},
			setProxySettings{Http: "http", Https: "https", Ftp: "ftp", NoProxy: "localhost"},
			runCommands{
				fmt.Sprintf("Set-Content %s $env:http_proxy", testFile("proxy.output")),
				fmt.Sprintf("Add-Content %s $env:HTTP_PROXY", testFile("proxy.output")),
				fmt.Sprintf("Add-Content %s $env:https_proxy", testFile("proxy.output")),
				fmt.Sprintf("Add-Content %s $env:HTTPS_PROXY", testFile("proxy.output")),
				fmt.Sprintf("Add-Content %s $env:ftp_proxy", testFile("proxy.output")),
				fmt.Sprintf("Add-Content %s $env:FTP_PROXY", testFile("proxy.output")),
				fmt.Sprintf("Add-Content %s $env:no_proxy", testFile("proxy.output")),
				fmt.Sprintf("Add-Content %s $env:NO_PROXY", testFile("proxy.output")),
			},
			verifyFile{
				testFile("proxy.output"),
				"http\nhttp\nhttps\nhttps\nftp\nftp\nlocalhost\nlocalhost\n",
			}), ut(
			"run commands: async using rpc client",
			quickStart{},
			asyncRunCommands{echoUnitNameToFile("run.output")},
			verifyFile{testFile("run.output"), "juju run u/0\n"},
		), ut(
			"run commands: waits for lock",
			quickStart{},
			acquireHookSyncLock{},
			asyncRunCommands{echoUnitNameToFile("wait.output")},
			verifyNoFile{testFile("wait.output")},
			releaseHookSyncLock,
			verifyFile{testFile("wait.output"), "juju run u/0\n"},
		),
	})
}
