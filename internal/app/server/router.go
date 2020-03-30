package server

import (
	"github.com/gin-gonic/gin"
	"github.com/pantrif/s2-geojson/internal/app/controllers"
	"net/http"
)

// NewRouter setups all gin routes, templates & static files
func NewRouter(root string) *gin.Engine {
	health := new(controllers.HealthController)
	p := new(controllers.GeometryController)

	r := gin.Default()
	r.GET("/health", health.Status)
	r.LoadHTMLGlob(root + "/*.html")
	r.Static("/js", root+"/js")
	r.Static("/css", root+"/css")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.POST("/cover", p.Cover)
	r.POST("/check_intersection", p.CheckIntersection)

	r.POST("/cover-h3", p.CoverH3)

	return r
}
