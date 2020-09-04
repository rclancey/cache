package s3cache

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }
type S3CacheSuite struct {}
var _ = Suite(&S3CacheSuite{})

func (a *S3CacheSuite) TestX(c *C) {
	c.Check(true, Equals, true)
}
