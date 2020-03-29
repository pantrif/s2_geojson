package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
	"github.com/pantrif/s2-geojson/pkg/geo"
	geojson "github.com/paulmach/go.geojson"
	"github.com/uber/h3-go"
	"log"
	"strconv"
	"strings"
)

// GeometryController struct
type GeometryController struct{}

// Cover uses s2 region coverer to cover geometries of geojson (only points and polygons supported)
func (u GeometryController) Cover(c *gin.Context) {
	gJSON := []byte(c.PostForm("geojson"))
	maxLevel, err := strconv.Atoi(c.PostForm("max_level_geojson"))
	minLevel, err := strconv.Atoi(c.PostForm("min_level_geojson"))

	fs, err := geo.DecodeGeoJSON(gJSON)

	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	var tokens []string
	var s2cells [][][]float64

	for _, f := range fs {

		if f.Geometry.IsPolygon() {
			for _, p := range f.Geometry.Polygon {
				p := geo.PointsToPolygon(p)
				_, t, c := geo.CoverPolygon(p, maxLevel, minLevel)
				s2cells = append(s2cells, c...)
				tokens = append(tokens, t...)
			}
		}
		if f.Geometry.IsPoint() {
			point := geo.Point{Lat: f.Geometry.Point[1], Lng: f.Geometry.Point[0]}
			_, t, c := geo.CoverPoint(point, maxLevel)
			s2cells = append(s2cells, c...)
			tokens = append(tokens, t)
		}
	}

	c.JSON(200, gin.H{
		"max_level_geojson": maxLevel,
		"cell_tokens":       strings.Join(tokens, ","),
		"cells":             s2cells,
	})
}

// CoverH3 returns a set of H3 hexagons that cover the input geometry.
func (u GeometryController) CoverH3(c *gin.Context) {
	gJSON := []byte(c.PostForm("geojson"))
	res, err := strconv.Atoi(c.PostForm("h3_resolution"))

	features, err := geo.DecodeGeoJSON(gJSON)
	for _, f := range features {
		log.Print(f.Geometry.Polygon)
	}
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	geoJsonCollection := geojson.NewFeatureCollection()
	for _, f := range features {
		if !f.Geometry.IsPolygon() {
			// Skip non-polygon geometries.
			continue
		}
		for _, p := range f.Geometry.Polygon {
			var hexagons []h3.H3Index
			var h3Points []h3.GeoCoord

			for _, ll := range p {
				h3Points = append(h3Points, h3.GeoCoord{Latitude:ll[1], Longitude: ll[0]})
			}
			hexagons = h3.Polyfill(h3.GeoPolygon{Geofence: h3Points}, res)
			compacted := h3.Compact(hexagons)

			for _, c := range compacted {
				coords := geo.H3IndexToCoordinates(c)
				// Add hexagon to the feature collection.
				geoJsonCollection.AddFeature(geojson.NewPolygonFeature([][][]float64{coords}))
			}
		}
	}

	c.JSON(200, gin.H{
		"hexagons_geojson":  geoJsonCollection,
	})
}

// CheckIntersection checks intersection of geoJSON geometries with a point and with a circle
func (u GeometryController) CheckIntersection(c *gin.Context) {
	lat, err := strconv.ParseFloat(c.PostForm("lat"), 64)
	lng, err := strconv.ParseFloat(c.PostForm("lng"), 64)
	radius, err := strconv.ParseFloat(c.PostForm("radius"), 64)

	gJSON := []byte(c.PostForm("geojson"))
	maxLevel, err := strconv.Atoi(c.PostForm("max_level_geojson"))
	minLevel, err := strconv.Atoi(c.PostForm("min_level_geojson"))
	maxLevelCircle, err := strconv.Atoi(c.PostForm("max_level_circle"))

	fs, err := geo.DecodeGeoJSON(gJSON)

	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	angle := s1.Angle((radius / 1000) / geo.EarthRadius)
	ca := s2.CapFromCenterAngle(s2.PointFromLatLng(s2.LatLngFromDegrees(lat, lng)), angle)
	circeCov := &s2.RegionCoverer{MaxLevel: maxLevelCircle, MaxCells: 300}
	circleRegion := s2.Region(ca)
	circleCovering := circeCov.Covering(circleRegion)

	var values []string
	var s2cells [][][]float64

	for _, c := range circleCovering {
		c1 := s2.CellFromCellID(s2.CellIDFromToken(c.ToToken()))

		var s2cell [][]float64
		for i := 0; i < 4; i++ {
			latlng := s2.LatLngFromPoint(c1.Vertex(i))
			s2cell = append(s2cell, []float64{latlng.Lat.Degrees(), latlng.Lng.Degrees()})
		}

		s2cells = append(s2cells, s2cell)

		values = append(values, c.ToToken())
	}

	ll := s2.LatLngFromDegrees(lat, lng)
	cell := s2.CellFromLatLng(ll)

	intersectsPoint, intersectsCircle := false, false

	for _, f := range fs {

		if f.Geometry.IsPolygon() {
			for _, p := range f.Geometry.Polygon {
				p := geo.PointsToPolygon(p)
				covering, _, _ := geo.CoverPolygon(p, maxLevel, minLevel)

				if covering.IntersectsCell(cell) {
					intersectsPoint = true
				}
				if covering.Intersects(circleCovering) {
					intersectsCircle = true
				}
			}
		}

		if f.Geometry.IsPoint() {
			point := geo.Point{Lat: f.Geometry.Point[1], Lng: f.Geometry.Point[0]}
			cc, _, _ := geo.CoverPoint(point, maxLevel)

			if cell.IntersectsCell(cc) {
				intersectsPoint = true
			}
			if circleCovering.IntersectsCell(cc) {
				intersectsCircle = true
			}
		}
	}

	c.JSON(200, gin.H{
		"intersects_with_point":  intersectsPoint,
		"intersects_with_circle": intersectsCircle,
		"radius":                 radius,
		"cells":                  s2cells,
	})
}
