package gincache

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/Pantani/errors"
	"github.com/Pantani/logger"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

var (
	memoryCache *memCache
)

func init() {
	memoryCache = &memCache{cache: cache.New(5*time.Minute, 5*time.Minute)}
}

type memCache struct {
	sync.RWMutex
	cache *cache.Cache
}

type cacheResponse struct {
	Status int
	Header http.Header
	Data   []byte
}

type cachedWriter struct {
	gin.ResponseWriter
	status  int
	written bool
	expire  time.Duration
	key     string
}

var _ gin.ResponseWriter = &cachedWriter{}

// newCachedWriter create a new cache writer.
func newCachedWriter(expire time.Duration, writer gin.ResponseWriter, key string) *cachedWriter {
	return &cachedWriter{writer, 0, false, expire, key}
}

// WriteHeader satisfy the built-in interface for writers.
func (w *cachedWriter) WriteHeader(code int) {
	w.status = code
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}

// Status satisfy the built-in interface for writers.
func (w *cachedWriter) Status() int {
	return w.ResponseWriter.Status()
}

// Written satisfy the built-in interface for writers.
func (w *cachedWriter) Written() bool {
	return w.ResponseWriter.Written()
}

// Write satisfy the built-in interface for writers.
func (w *cachedWriter) Write(data []byte) (int, error) {
	ret, err := w.ResponseWriter.Write(data)
	if err != nil {
		return 0, errors.E(err, "fail to cache write string", errors.Params{"data": data})
	}
	if w.Status() != 200 {
		return 0, errors.E("Write: invalid cache status", errors.Params{"data": data})
	}
	val := cacheResponse{
		w.Status(),
		w.Header(),
		data,
	}
	b, err := json.Marshal(val)
	if err != nil {
		return 0, errors.E("validator cache: failed to marshal cache object")
	}
	memoryCache.cache.Set(w.key, b, w.expire)
	return ret, nil
}

// WriteString satisfy the built-in interface for writers.
func (w *cachedWriter) WriteString(data string) (n int, err error) {
	ret, err := w.ResponseWriter.WriteString(data)
	if err != nil {
		return 0, errors.E(err, "fail to cache write string", errors.Params{"data": data})
	}
	if w.Status() != 200 {
		return 0, errors.E("WriteString: invalid cache status", errors.Params{"data": data})
	}
	val := cacheResponse{
		w.Status(),
		w.Header(),
		[]byte(data),
	}
	b, err := json.Marshal(val)
	if err != nil {
		return 0, errors.E("validator cache: failed to marshal cache object")
	}
	memoryCache.setCache(w.key, b, w.expire)
	return ret, err
}

// deleteCache remove cache from memory
func (mc *memCache) deleteCache(key string) {
	mc.RLock()
	defer mc.RUnlock()
	memoryCache.cache.Delete(key)
}

// setCache save cache inside memory with duration.
func (mc *memCache) setCache(key string, data interface{}, d time.Duration) {
	b, err := json.Marshal(data)
	if err != nil {
		logger.Error(errors.E(err, "client cache cannot marshal cache object"))
		return
	}
	mc.RLock()
	defer mc.RUnlock()
	memoryCache.cache.Set(key, b, d)
}

// getCache restore cache from memory.
// It returns the cache and an error if occurs.
func (mc *memCache) getCache(key string) (cacheResponse, error) {
	var result cacheResponse
	c, ok := mc.cache.Get(key)
	if !ok {
		return result, fmt.Errorf("gin-cache: invalid cache key %s", key)
	}
	r, ok := c.([]byte)
	if !ok {
		return result, errors.E("validator cache: failed to cast cache to bytes")
	}
	err := json.Unmarshal(r, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// generateKey generate a key to storage cache.
// It returns the key.
func generateKey(c *gin.Context) string {
	url := c.Request.URL.String()
	var b []byte
	if c.Request.Body != nil {
		b, _ = ioutil.ReadAll(c.Request.Body)
		// Restore the io.ReadCloser to its original state
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	}
	hash := sha1.Sum(append([]byte(url), b...))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// CacheMiddleware encapsulates a gin handler function and caches the response with an expiration time.
// It returns the gin handler function.
func CacheMiddleware(expiration time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer c.Next()
		key := generateKey(c)
		mc, err := memoryCache.getCache(key)
		if err != nil || mc.Data == nil {
			writer := newCachedWriter(expiration, c.Writer, key)
			c.Writer = writer
			handle(c)

			if c.IsAborted() {
				memoryCache.deleteCache(key)
			}
			return
		}

		c.Writer.WriteHeader(mc.Status)
		for k, vals := range mc.Header {
			for _, v := range vals {
				c.Writer.Header().Set(k, v)
			}
		}
		_, err = c.Writer.Write(mc.Data)
		if err != nil {
			memoryCache.deleteCache(key)
			logger.Error(err, "cannot write data", mc)
		}
	}
}
