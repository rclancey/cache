package fscache

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/rclancey/cache"
	"github.com/rclancey/fsutil"
)

type FSCacheFile struct {
	f *fsutil.LockedFile
	expired bool
	reset bool
}

func (cf *FSCacheFile) Read(p []byte) (int, error) {
	if cf.expired {
		return -1, cache.CacheFileExpired
	}
	return cf.f.Read(p)
}

func (cf *FSCacheFile) Write(data []byte) (int, error) {
	if !cf.reset {
		_, err := cf.f.Seek(0, os.SEEK_SET)
		if err != nil {
			return -1, errors.Wrap(err, "error resetting cache file")
		}
		err = cf.f.Truncate(0)
		if err != nil {
			return -1, errors.Wrap(err, "error resetting cache file")
		}
		cf.reset = true
	}
	return cf.f.Write(data)
}

func (cf *FSCacheFile) Close() error {
	return cf.f.Close()
}

type FSCacheStore struct {
	root string
}

func NewFSCacheStore(root string) *FSCacheStore {
	return &FSCacheStore{root: root}
}

func (cs *FSCacheStore) Open(cacheFile string, cacheTime time.Duration) (cache.CacheFile, error) {
	fn := filepath.Join(cs.root, filepath.FromSlash(cacheFile))
	f, err := fsutil.OpenLocked(fn, os.O_RDWR | os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	cf := &FSCacheFile{f: f, expired: true}
	if cacheTime == 0 {
		return cf, nil
	}
	if cacheTime < 0 {
		cf.expired = false
		return cf, nil
	}
	st, err := f.Stat()
	if err != nil {
		if os.IsNotExist(err) {
			return cf, nil
		}
		f.Close()
		return nil, err
	}
	if st.Size() > 0 {
		if st.ModTime().After(time.Now().Add(-1 * cacheTime)) {
			cf.expired = false
		}
	}
	return cf, nil
}

func (cs *FSCacheStore) Delete(cacheFile string) error {
	fn := filepath.Join(cs.root, filepath.FromSlash(cacheFile))
	return os.Remove(fn)
}
