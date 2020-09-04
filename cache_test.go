package cache

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }
type CacheSuite struct {}
var _ = Suite(&CacheSuite{})

func (a *CacheSuite) TestX(c *C) {
	c.Check(true, Equals, true)
}
