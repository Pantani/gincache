# Simple cache for [gin](github.com/gin-gonic/gin)

Simple memory cache for [gin](github.com/gin-gonic/gin) API. 

E.g.:

- Create an API cache adding the middleware to your route:
```
router.POST("/cache/list", gincache.CacheMiddleware(time.Hour*24, func(c *gin.Context) {
    // handler implementation		
}))
```
