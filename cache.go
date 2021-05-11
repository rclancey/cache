package cache

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	//"net/url"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var CacheFileExpired = errors.New("Cache file expired")
var NotJSONData = errors.New("data is not json formatted")

type CacheFile interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	Valid() bool
}

type CacheStore interface {
	Open(string, time.Duration) (CacheFile, error)
	Delete(string) error
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Cache struct {
	store CacheStore
	client HTTPClient
}

func NewCache(store CacheStore, client HTTPClient) *Cache {
	return &Cache{store: store, client: client}
}

// CacheFunc checks if the file cacheFile exists and is less than cacheTime
// old.  If so, it returns the contents of cacheFile.  Otherwise, it executes
// the function f and, if no error is returned, writes the output to cacheFile
// and returns the result.
func (c *Cache) CacheFunc(f func() ([]byte, error), name string, cacheTime time.Duration) ([]byte, error) {
	cf, err := c.store.Open(name, cacheTime)
	if err != nil {
		return nil, err
	}
	defer cf.Close()
	if cf.Valid() {
		data, err := ioutil.ReadAll(cf)
		if err == nil {
			return data, nil
		}
	}
	data, err := f()
	if err != nil {
		return data, err
	}
	_, err = cf.Write(data)
	if err != nil {
		return data, err
	}
	return data, nil
}

func (c *Cache) CacheRequest(req *http.Request, cacheTime time.Duration) (*http.Response, error) {
	sum := sha1.Sum([]byte(req.Method + " " + req.URL.String()))
	code := hex.EncodeToString(sum[:])
	name := path.Join(code[0:2], code[2:4], code[4:])
	cf, err := c.store.Open(name, cacheTime)
	if err != nil {
		return nil, err
	}
	if cf.Valid() {
		rd := bufio.NewReader(cf)
		res, err := http.ReadResponse(rd, req)
		if err == nil {
			return res, nil
		}
	}
	defer cf.Close()
	res, err := c.client.Do(req)
	if err != nil {
		return res, err
	}
	if req.Method == http.MethodGet && res.StatusCode == http.StatusOK {
		data, err := httputil.DumpResponse(res, true)
		if err != nil {
			return res, err
		}
		_, err = cf.Write(data)
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

func (c *Cache) CacheRequestData(req *http.Request, cacheTime time.Duration) ([]byte, error) {
	res, err := c.CacheRequest(req, cacheTime)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(res.Body)
}

func (c *Cache) CacheURL(u string, cacheTime time.Duration) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return c.CacheRequest(req, cacheTime)
}

func (c *Cache) CacheURLData(u string, cacheTime time.Duration) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return c.CacheRequestData(req, cacheTime)
}

func (c *Cache) CacheFuncJSON(f func(interface{}) error, obj interface{}, name string, cacheTime time.Duration) error {
	cf, err := c.store.Open(name, cacheTime)
	if err != nil {
		return err
	}
	defer cf.Close()
	if cf.Valid() {
		data, err := ioutil.ReadAll(cf)
		if err == nil {
			err = json.Unmarshal(data, obj)
			if err == nil {
				return nil
			}
		}
	}
	err = f(obj)
	if err != nil {
		return err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = cf.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) CacheRequestJSON(req *http.Request, obj interface{}, cacheTime time.Duration) error {
	res, err := c.CacheRequest(req, cacheTime)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return err
	}
	ct := strings.ToLower(strings.Split(res.Header.Get("Content-Type"), ";")[0])
	switch ct {
	case "application/json", "text/json", "application/javascript":
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, obj)
	}
	return NotJSONData
}

func (c *Cache) CacheURLJSON(u string, obj interface{}, cacheTime time.Duration) error {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	return c.CacheRequestJSON(req, obj, cacheTime)
}
