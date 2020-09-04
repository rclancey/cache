package fscache

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }
type FSCacheSuite struct {}
var _ = Suite(&FSCacheSuite{})

func (a *FSCacheSuite) TestX(c *C) {
	c.Check(true, Equals, true)
}
