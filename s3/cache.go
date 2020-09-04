package s3cache

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	//"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	//"github.com/pkg/errors"
	"github.com/rclancey/cache"
	//"github.com/rclancey/fsutil"
)

type S3CacheFile struct {
	svc *s3.S3
	bucket string
	key string
	obj *s3.GetObjectOutput
	out *bytes.Buffer
	r io.Reader
	w io.Writer
}

func (cf *S3CacheFile) Read(p []byte) (int, error) {
	if cf.obj == nil {
		return -1, cache.CacheFileExpired
	}
	if cf.r == nil {
		cf.r = cf.obj.Body
		if cf.obj.ContentEncoding != nil {
			if *cf.obj.ContentEncoding == "gzip" {
				gzr, err := gzip.NewReader(cf.obj.Body)
				if err == nil {
					cf.r = gzr
				}
			}
		}
	}
	return cf.r.Read(p)
}

func (cf *S3CacheFile) Write(data []byte) (int, error) {
	if cf.w == nil {
		cf.out = bytes.NewBuffer([]byte{})
		cf.w = gzip.NewWriter(cf.out)
	}
	return cf.w.Write(data)
}

func (cf *S3CacheFile) Close() error {
	if cf.obj != nil {
		if cf.r != nil && cf.obj.Body != cf.r {
			rc, isa := cf.r.(io.Closer)
			if isa {
				rc.Close()
			}
		}
		cf.obj.Body.Close()
	}
	if cf.w != nil {
		wc, isa := cf.w.(io.Closer)
		if isa {
			wc.Close()
		}
		req := &s3.PutObjectInput{}
		req.SetBucket(cf.bucket)
		req.SetKey(cf.key)
		req.SetBody(bytes.NewReader(cf.out.Bytes()))
		req.SetContentEncoding("gzip")
		_, err := cf.svc.PutObject(req)
		return err
	}
	return nil
}

type S3CacheStore struct {
	svc *s3.S3
	bucket string
}

func NewS3CacheStore(bucket string) (*S3CacheStore, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	cfg := aws.NewConfig().WithRegion(region)
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, err
	}
	svc := s3.New(sess)
	return &S3CacheStore{svc: svc, bucket: bucket}, nil
}

func (cs *S3CacheStore) Open(cacheFile string, cacheTime time.Duration) (cache.CacheFile, error) {
	cf := &S3CacheFile{svc: cs.svc, bucket: cs.bucket, key: cacheFile}
	req := &s3.GetObjectInput{}
	req.SetBucket(cs.bucket)
	req.SetKey(cacheFile)
	if cacheTime >= 0 {
		req.SetIfModifiedSince(time.Now().Add(-1 * cacheTime))
	}
	res, err := cs.svc.GetObject(req)
	if err == nil {
		cf.obj = res
		return cf, nil
	}
	return cf, nil
}

func (cs *S3CacheStore) Delete(cacheFile string) error {
	req := &s3.DeleteObjectInput{}
	req.SetBucket(cs.bucket)
	req.SetKey(cacheFile)
	_, err := cs.svc.DeleteObject(req)
	return err
}
