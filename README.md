[![codecov](https://codecov.io/gh/Pantani/gincache/branch/master/graph/badge.svg?token=SOD3RT9AOW)](https://codecov.io/gh/Pantani/gincache)
[![Go Reference](https://pkg.go.dev/badge/github.com/Pantani/gincache.svg)](https://pkg.go.dev/github.com/Pantani/gincache)

# Simple cache for [gin](https://github.com/gin-gonic/gin)

Simple memory cache for [gin](https://github.com/gin-gonic/gin) API. 

E.g.:

- Create an API cache adding the middleware to your route:
```go
router.POST("/cache/list", gincache.CacheMiddleware(time.Hour*24, func(c *gin.Context) {
    // handler implementation		
}))
```
