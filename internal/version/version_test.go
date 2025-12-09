package version

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func capture(f func()) string {
	originalStdout := os.Stdout

	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = originalStdout

	var buf = make([]byte, 5096)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func Test_ReleaseReturnsVersion(t *testing.T) {
	expected := "1.0.0"
	Version = expected

	assert.Equal(t, expected, Release(), "version should match expected")
}

func Test_ReleaseReturnsDev_EmptyVersion(t *testing.T) {
	Version = ""

	assert.Equal(t, "dev", Release(), "version should be 'dev' when empty")
}

func Test_CommitReturnsCommitHash(t *testing.T) {
	expected := "abc123def"
	GitCommit = expected

	assert.Equal(t, expected, Commit(), "commit hash should match expected")
}

func Test_BannerReturnsString(t *testing.T) {
	assert.IsType(t, "", Banner(), "banner should be a string")
	assert.Len(t, Banner(), 118, "banner should not be empty")
}

func Test_PrintOutputsBannerAndInfo(t *testing.T) {
	Version = "1.0.0"
	GitCommit = "abc123def"

	output := capture(Print)

	assert.Contains(t, output, Banner(), "output should contain banner")
	assert.Contains(t, output, "Release: 1.0.0", "output should contain release info")
	assert.Contains(t, output, "Commit:  abc123def", "output should contain commit info")
}
