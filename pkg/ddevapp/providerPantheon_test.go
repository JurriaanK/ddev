package ddevapp_test

import (
	"fmt"
	. "github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/testcommon"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

/**
 * These tests rely on an external test account. To run them, you'll
 * need to set an environment variable called "DDEV_PANTHEON_API_TOKEN" with credentials for
 * this account. If no such environment variable is present, these tests will be skipped.
 *
 */

const pantheonPullTestSite = "ddev-test-site-do-not-delete.dev"
const pantheonPushTestSite = "ddev-pantheon-push.dev"
const pantheonSiteURL = "https://dev-ddev-test-site-do-not-delete.pantheonsite.io/"
const pantheonSiteExpectation = "DDEV DRUPAL8 TEST SITE"
const pantheonPullGitURL = "ssh://codeserver.dev.009a2cda-2c22-4eee-8f9d-96f017321627@codeserver.dev.009a2cda-2c22-4eee-8f9d-96f017321627.drush.in:2222/~/repository.git"
const pantheonPushGitURL = "ssh://codeserver.dev.d32c631e-c998-480f-93bc-7c36e6ae4142@codeserver.dev.d32c631e-c998-480f-93bc-7c36e6ae4142.drush.in:2222/~/repository.git"

// Note that these tests won't run with GitHub actions on a forked PR.
// Thie is a security feature, but means that PRs intended to test this
// must be done in the ddev repo.

// TestPantheonPull ensures we can pull from pantheon.
func TestPantheonPull(t *testing.T) {
	token := ""
	sshkey := ""
	if token = os.Getenv("DDEV_PANTHEON_API_TOKEN"); token == "" {
		t.Skipf("No DDEV_PANTHEON_API_TOKEN env var has been set. Skipping %v", t.Name())
	}
	if sshkey = os.Getenv("DDEV_PANTHEON_SSH_KEY"); sshkey == "" {
		t.Skipf("No DDEV_PANTHEON_SSH_KEY env var has been set. Skipping %v", t.Name())
	}
	sshkey = strings.Replace(sshkey, "<SPLIT>", "\n", -1)

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	require.True(t, isPullSiteValid(pantheonSiteURL, pantheonSiteExpectation), "pantheonSiteURL %s isn't working right", pantheonSiteURL)

	webEnvSave := globalconfig.DdevGlobalConfig.WebEnvironment
	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"TERMINUS_MACHINE_TOKEN=" + token}
	err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	assert.NoError(err)

	siteDir := testcommon.CreateTmpDir(t.Name())
	err = os.MkdirAll(filepath.Join(siteDir, "sites/default"), 0777)
	require.NoError(t, err)
	err = os.Chdir(siteDir)
	require.NoError(t, err)

	err = setupSSHKey(t, sshkey, filepath.Join(origDir, "testdata", t.Name()))
	require.NoError(t, err)

	app, err := NewApp(siteDir, true)
	assert.NoError(err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		globalconfig.DdevGlobalConfig.WebEnvironment = webEnvSave
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)

		_ = os.Chdir(origDir)
		err = os.RemoveAll(siteDir)
		assert.NoError(err)
	})

	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal8
	app.Hooks = map[string][]YAMLTask{"post-pull": {{"exec-host": "touch hello-post-pull-" + app.Name}}, "pre-pull": {{"exec-host": "touch hello-pre-pull-" + app.Name}}}

	_ = app.Stop(true, false)

	err = app.WriteConfig()
	assert.NoError(err)

	testcommon.ClearDockerEnv()

	err = PopulateExamplesCommandsHomeadditions(app.Name)
	require.NoError(t, err)

	// Build our pantheon.yaml from the example file
	s, err := os.ReadFile(app.GetConfigPath("providers/pantheon.yaml.example"))
	require.NoError(t, err)
	x := strings.Replace(string(s), "project:", fmt.Sprintf("project: %s\n#project:", pantheonPullTestSite), 1)
	err = os.WriteFile(app.GetConfigPath("providers/pantheon.yaml"), []byte(x), 0666)
	assert.NoError(err)
	err = app.WriteConfig()
	require.NoError(t, err)

	provider, err := app.GetProvider("pantheon")
	require.NoError(t, err)
	err = app.Start()
	require.NoError(t, err)

	// Make sure we have drush
	_, _, err = app.Exec(&ExecOpts{
		Cmd: "composer require --no-interaction drush/drush:* >/dev/null 2>/dev/null",
	})
	require.NoError(t, err)

	err = app.Pull(provider, false, false, false)
	require.NoError(t, err)

	assert.FileExists(filepath.Join(app.GetHostUploadDirFullPath(), "2017-07/22-24_tn.jpg"))
	out, err := exec.RunHostCommand("bash", "-c", fmt.Sprintf(`echo 'select COUNT(*) from users_field_data where mail="admin@example.com";' | %s mysql -N`, DdevBin))
	assert.NoError(err)
	assert.True(strings.HasPrefix(out, "1\n"))

	err = app.MutagenSyncFlush()
	assert.NoError(err)
	assert.FileExists("hello-pre-pull-" + app.Name)
	assert.FileExists("hello-post-pull-" + app.Name)
	err = os.Remove("hello-pre-pull-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-pull-" + app.Name)
	assert.NoError(err)
}

// TestPantheonPush ensures we can push to pantheon for a configured environment.
func TestPantheonPush(t *testing.T) {
	token := ""
	sshkey := ""
	if token = os.Getenv("DDEV_PANTHEON_API_TOKEN"); token == "" {
		t.Skipf("No DDEV_PANTHEON_API_TOKEN env var has been set. Skipping %v", t.Name())
	}
	if sshkey = os.Getenv("DDEV_PANTHEON_SSH_KEY"); sshkey == "" {
		t.Skipf("No DDEV_PANTHEON_SSH_KEY env var has been set. Skipping %v", t.Name())
	}
	sshkey = strings.Replace(sshkey, "<SPLIT>", "\n", -1)

	// Set up tests and give ourselves a working directory.
	assert := asrt.New(t)
	origDir, _ := os.Getwd()

	webEnvSave := globalconfig.DdevGlobalConfig.WebEnvironment
	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"TERMINUS_MACHINE_TOKEN=" + token}
	err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	assert.NoError(err)

	// Use a D9 codebase for drush to work right
	d9code := FullTestSites[8]
	d9code.Name = t.Name()
	err = globalconfig.RemoveProjectInfo(t.Name())
	require.NoError(t, err)
	err = d9code.Prepare()
	require.NoError(t, err)
	app, err := NewApp(d9code.Dir, false)
	require.NoError(t, err)
	_ = app.Stop(true, false)

	err = os.Chdir(d9code.Dir)
	require.NoError(t, err)

	err = setupSSHKey(t, sshkey, filepath.Join(origDir, "testdata", t.Name()))
	require.NoError(t, err)

	t.Cleanup(func() {
		err = app.Stop(true, false)
		assert.NoError(err)

		globalconfig.DdevGlobalConfig.WebEnvironment = webEnvSave
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)

		_ = os.Chdir(origDir)
	})

	app.Name = t.Name()
	app.Type = nodeps.AppTypeDrupal9
	app.Hooks = map[string][]YAMLTask{"post-push": {{"exec-host": "touch hello-post-push-" + app.Name}}, "pre-push": {{"exec-host": "touch hello-pre-push-" + app.Name}}}
	_ = app.Stop(true, false)

	err = app.WriteConfig()
	require.NoError(t, err)

	testcommon.ClearDockerEnv()

	err = PopulateExamplesCommandsHomeadditions(app.Name)
	require.NoError(t, err)

	tval := nodeps.RandomString(10)
	err = os.MkdirAll(filepath.Join(app.AppRoot, app.Docroot, "sites/default/files"), 0777)
	require.NoError(t, err)
	fName := tval + ".txt"
	fContent := []byte(tval)
	err = os.WriteFile(filepath.Join(app.AppRoot, app.Docroot, "sites/default/files", fName), fContent, 0644)
	require.NoError(t, err)

	// Build our pantheon.yaml from the example file
	s, err := os.ReadFile(app.GetConfigPath("providers/pantheon.yaml.example"))
	require.NoError(t, err)
	x := strings.Replace(string(s), "project:", fmt.Sprintf("project: %s\n#project:", pantheonPushTestSite), 1)
	err = os.WriteFile(app.GetConfigPath("providers/pantheon.yaml"), []byte(x), 0666)
	assert.NoError(err)
	err = app.WriteConfig()
	require.NoError(t, err)

	provider, err := app.GetProvider("pantheon")
	require.NoError(t, err)
	err = app.Start()
	require.NoError(t, err)

	// Since allow-plugins isn't there and you can't even set it with composer...
	_, _, err = app.Exec(&ExecOpts{
		Cmd: `composer config --no-plugins allow-plugins true`,
	})
	require.NoError(t, err)

	// Make sure we have drush
	_, _, err = app.Exec(&ExecOpts{
		Cmd: "composer require --no-interaction drush/drush:* >/dev/null 2>/dev/null",
	})
	require.NoError(t, err)
	err = app.MutagenSyncFlush()
	assert.NoError(err)

	// Do minimal install so it can find %file dir
	_, _, err = app.Exec(&ExecOpts{
		Cmd: "time drush si -y minimal",
	})
	require.NoError(t, err)

	// Create database and files entries that we can verify after push
	_, _, err = app.Exec(&ExecOpts{
		Cmd: fmt.Sprintf(`mysql -e 'CREATE TABLE IF NOT EXISTS %s ( title VARCHAR(255) NOT NULL ); INSERT INTO %s VALUES("%s");'`, t.Name(), t.Name(), tval),
	})
	require.NoError(t, err)

	err = app.Push(provider, false, false)
	require.NoError(t, err)

	// Test that the database row was added
	out, _, err := app.Exec(&ExecOpts{
		Cmd: fmt.Sprintf(`echo 'SELECT title FROM %s WHERE title="%s"' | drush @%s sql-cli --extra=-N`, t.Name(), tval, pantheonPushTestSite),
	})
	require.NoError(t, err)
	assert.Contains(out, tval)

	// Test that the file arrived there (by rsyncing it back)
	out, _, err = app.Exec(&ExecOpts{
		Cmd: fmt.Sprintf("drush rsync -y @%s:%%files/%s /tmp && cat /tmp/%s", pantheonPushTestSite, fName, fName),
	})
	require.NoError(t, err)
	assert.Contains(out, tval)

	err = app.MutagenSyncFlush()
	assert.NoError(err)

	assert.FileExists("hello-pre-push-" + app.Name)
	assert.FileExists("hello-post-push-" + app.Name)
	err = os.Remove("hello-pre-push-" + app.Name)
	assert.NoError(err)
	err = os.Remove("hello-post-push-" + app.Name)
	assert.NoError(err)
}

// setupSSHKey takes a privatekey string and turns it into a file and then does `ddev auth ssh`
func setupSSHKey(t *testing.T, privateKey string, expectScriptDir string) error {
	// Provide an ssh key for `ddev auth ssh`
	err := os.Mkdir("sshtest", 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join("sshtest", "id_rsa_test"), []byte(privateKey), 0600)
	require.NoError(t, err)
	out, err := exec.RunHostCommand("expect", filepath.Join(expectScriptDir, "ddevauthssh.expect"), DdevBin, "./sshtest")
	require.NoError(t, err)
	require.Contains(t, string(out), "Identity added:")
	return nil
}

// Monthly do a push to pantheon repos to keep them active
func TestPantheonDoMonthlyPush(t *testing.T) {
	// Pantheon freezes inactive sites, so why not do a commit when we run to prevent that?
	_, _, day := time.Now().Date()
	if day != 10 {
		t.Skipf("It's not the right day to do pantheon code push.")
	}

	assert := asrt.New(t)
	token := ""
	sshkey := ""

	origDir, _ := os.Getwd()
	if token = os.Getenv("DDEV_PANTHEON_API_TOKEN"); token == "" {
		t.Skipf("No DDEV_PANTHEON_API_TOKEN env var has been set. Skipping %v", t.Name())
	}
	if sshkey = os.Getenv("DDEV_PANTHEON_SSH_KEY"); sshkey == "" {
		t.Skipf("No DDEV_PANTHEON_SSH_KEY env var has been set. Skipping %v", t.Name())
	}
	sshkey = strings.Replace(sshkey, "<SPLIT>", "\n", -1)

	webEnvSave := globalconfig.DdevGlobalConfig.WebEnvironment
	globalconfig.DdevGlobalConfig.WebEnvironment = []string{"TERMINUS_MACHINE_TOKEN=" + token}
	err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	assert.NoError(err)

	tmpDir := testcommon.CreateTmpDir(t.Name())
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	_ = os.Mkdir("sshtest", 0755)
	err = os.WriteFile(filepath.Join("sshtest", "id_rsa_test"), []byte(sshkey), 0600)
	require.NoError(t, err)

	// ssh-add the key for later pull/push
	out, err := exec.RunHostCommand("ssh-add", "./sshtest/id_rsa_test")
	if err != nil {
		t.Logf("Failed to ssh add; out=%s, err=%v", out, err)
	}

	t.Cleanup(func() {
		globalconfig.DdevGlobalConfig.WebEnvironment = webEnvSave
		err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
		assert.NoError(err)

		_ = os.Chdir(origDir)
		err = os.RemoveAll(tmpDir)
		assert.NoError(err)
	})

	for _, gitURL := range []string{pantheonPullGitURL, pantheonPushGitURL} {
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		checkoutDir := "checkoutdir"
		_ = os.RemoveAll(checkoutDir)
		_ = os.Setenv("GIT_SSH_COMMAND", "ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no")
		out, err := exec.RunHostCommand("git", "clone", gitURL, checkoutDir)
		assert.NoError(err, "Failed to git clone '%s'; out=%s, err=%v", gitURL, out, err)
		_ = os.Chdir(checkoutDir)

		out, err = exec.RunHostCommand("git", "commit", "--allow-empty", "-m", "Dummy commmit to keep pantheon alive")
		assert.NoError(err, "Failed to make git commit; out=%s, err=%v", out, err)

		out, err = exec.RunHostCommand("git", "push")
		assert.NoError(err, "Failed to make git push; out=%s, err=%v", out, err)
	}
}
