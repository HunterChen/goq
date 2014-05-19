package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

// test-creation utils

/* how to setup a test:

// *** universal test cfg setup
skipbye := false
cfg := NewTestConfig()
defer cfg.ByeTestConfig(&skipbye)
// *** end universal test setup

remote := true // or false
defer CleanupServer(cfg, jobservPid, jobserv, remote, &skipbye)
defer CleanupOutdir(cfg)

// during tests, if you want to preserve output directories that would
// normally be mopped up by the deferred functions, set skipbye = true

*/

// make a new fake-home-temp-directory for testing
// and cd into it. Save GOQ_HOME for later restoration.
func NewTestConfig() *Config {
	cfg := NewConfig()

	cfg.origdir, cfg.tempdir = MakeAndMoveToTempDir() // cd to tempdir

	// link back to bin
	err := os.Symlink(cfg.origdir+"/bin", cfg.tempdir+"/bin")
	if err != nil {
		panic(err)
	}

	cfg.orighome = os.Getenv("GOQ_HOME")
	os.Setenv("GOQ_HOME", cfg.tempdir)

	cfg.Home = cfg.tempdir
	cfg.JservPort = 1776
	cfg.JservIP = GetExternalIP()
	cfg.DebugMode = true
	cfg.JservAddr = fmt.Sprintf("tcp://%s:%d", cfg.JservIP, cfg.JservPort)
	cfg.Odir = "o"

	GenNewCreds(cfg)

	WaitUntilAddrAvailable(cfg.JservAddr)

	// not needed. GOQ_HOME should suffice. InjectConfigIntoEnv(cfg)
	return cfg
}

// restore GOQ_HOME and previous working directory
// allow to skip if test goes awry, even if it was deferred.
func (cfg *Config) ByeTestConfig(skip *bool) {
	if skip != nil && !(*skip) {
		TempDirCleanup(cfg.origdir, cfg.tempdir)
		os.Setenv("GOQ_HOME", cfg.orighome)
	}
	VPrintf("\n ByeTestConfig done.\n")
}

func CleanupOutdir(cfg *Config) {
	if DirExists(cfg.Odir) {
		c := exec.Command("/bin/rm", "-rf", cfg.Odir)
		c.CombinedOutput()
	}
	VPrintf("\n CleanupOutdir '%s' done.\n", cfg.Odir)
}

// *important* cleanup, and wait for cleanup to finish, so the next test can run.
// skip lets us say we've already done this
func CleanupServer(cfg *Config, jobservPid int, jobserv *JobServ, remote bool, skip *bool) {

	if skip == nil || !*skip {
		if remote {
			SendShutdown(cfg)
			WaitForShutdownWithTimeout(jobservPid)

		} else {
			// this wait is really important!!! even locally! Otherwise the next test gets hosed
			// because the clients will connect to the old server which then dies.
			jobserv.Ctrl <- die
			<-jobserv.Done

		}

	}
	VPrintf("\n CleanupServer done.\n")
}

func MakeAndMoveToTempDir() (origdir string, tmpdir string) {

	// make new temp dir that will have no ".goqclusterid files in it
	var err error
	origdir, err = os.Getwd()
	if err != nil {
		panic(err)
	}
	tmpdir, err = ioutil.TempDir(origdir, "tempgoqtestdir")
	if err != nil {
		panic(err)
	}
	os.Chdir(tmpdir)

	return origdir, tmpdir
}

func TempDirCleanup(origdir string, tmpdir string) {
	// cleanup
	os.Chdir(origdir)
	err := os.RemoveAll(tmpdir)
	if err != nil {
		panic(err)
	}
	VPrintf("\n TempDirCleanup of '%s' done.\n", tmpdir)
}
