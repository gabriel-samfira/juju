// build +windows,!linux

package winreg_test

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"syscall"
	"testing"

	gc "gopkg.in/check.v1"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"

	"github.com/juju/juju/utils/winreg"
)

func TestPackage(t *testing.T) {
	gc.TestingT(t)
}

type registryTestSuite struct {
	testKey          string
	testKeyName      string
	testKeyValueName string
	testKeyValue     string
	fullName         string
}

var _ = gc.Suite(&registryTestSuite{})

func (s *registryTestSuite) checkRegKeyExists(path string) (bool, error) {
	subtree, key, err := winreg.SplitRegPath(path)
	var handle syscall.Handle

	err = syscall.RegOpenKeyEx(subtree, syscall.StringToUTF16Ptr(key), 0, syscall.KEY_READ, &handle)
	if err == nil {
		syscall.RegCloseKey(handle)
		return true, nil
	}

	if err == syscall.ERROR_FILE_NOT_FOUND {
		return false, nil
	}
	return false, err
}

func (s *registryTestSuite) getTestRegKey(c *gc.C) (string, error) {
	for i := 0; i != 20; i++ {
		tmp := fmt.Sprintf("JujudTest-%d", rand.Int())
		joined := filepath.Join(s.testKey, tmp)
		keyExists, err := s.checkRegKeyExists(joined)
		c.Assert(err, gc.IsNil)
		if keyExists == false {
			return tmp, nil
		}
	}
	return "", errors.Errorf("failed to get test reg key")
}

func (s *registryTestSuite) SetUpTest(c *gc.C) {
	s.testKey = "HKLM:\\Software\\Wow6432Node\\Jujud"

	key, err := s.getTestRegKey(c)
	c.Assert(err, jc.ErrorIsNil)

	s.testKeyName = key
	s.fullName = filepath.Join(s.testKey, s.testKeyName)
	s.testKeyValueName = "jujud"
	s.testKeyValue = "meh"
}

func (s *registryTestSuite) TearDownTest(c *gc.C) {
	winreg.DeleteRegistryKey(s.testKey, s.testKeyName)
}

func (s *registryTestSuite) TestWriteReadRegistryString(c *gc.C) {
	err := winreg.WriteRegistryString(s.fullName, s.testKeyValueName, s.testKeyValue)
	c.Assert(err, jc.ErrorIsNil)
	val, err := winreg.ReadRegistryString(s.fullName, s.testKeyValueName)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(val, gc.Equals, s.testKeyValue)
}

func (s *registryTestSuite) TestCreateDeleteKey(c *gc.C) {
	err := winreg.WriteRegistryString(s.fullName, s.testKeyValueName, s.testKeyValue)
	c.Assert(err, jc.ErrorIsNil)
	exists, err := s.checkRegKeyExists(s.fullName)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(exists, jc.IsTrue)

	err = winreg.DeleteRegistryKey(s.testKey, s.testKeyName)
	c.Assert(err, jc.ErrorIsNil)

	exists, err = s.checkRegKeyExists(s.fullName)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(exists, jc.IsFalse)
}

func (s *registryTestSuite) TestWriteReadDeleteRegistryString(c *gc.C) {
	err := winreg.WriteRegistryString(s.fullName, s.testKeyValueName, s.testKeyValue)
	c.Assert(err, jc.ErrorIsNil)

	val, err := winreg.ReadRegistryString(s.fullName, s.testKeyValueName)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(val, gc.Equals, s.testKeyValue)

	err = winreg.DeleteRegistryKeyValue(s.fullName, s.testKeyValueName)
	c.Assert(err, jc.ErrorIsNil)

	err = winreg.DeleteRegistryKeyValue(s.fullName, s.testKeyValueName)
	c.Assert(err, gc.Equals, syscall.ERROR_FILE_NOT_FOUND)
}
