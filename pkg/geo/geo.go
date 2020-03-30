package geo

import (
	"github.com/golang/geo/s2"
	"github.com/paulmach/go.geojson"
	"github.com/uber/h3-go"
)

const (
	//EarthRadius the radius of earth in kilometers
	EarthRadius = 6371.01
	maxCells    = 100
)

// Point struct contains the lat/lng of a point
type Point struct {
	Lat float64
	Lng float64
}

// DecodeGeoJSON decodes a feature collection
func DecodeGeoJSON(json []byte) ([]*geojson.Feature, error) {
	f, err := geojson.UnmarshalFeatureCollection(json)
	if err != nil {
		return nil, err
	}
	return f.Features, nil
}

// PointsToPolygon converts points to s2 polygon
func PointsToPolygon(points [][]float64) *s2.Polygon {
	var pts []s2.Point
	for _, pt := range points {
		pts = append(pts, s2.PointFromLatLng(s2.LatLngFromDegrees(pt[1], pt[0])))
	}
	loop := s2.LoopFromPoints(pts)

	return s2.PolygonFromLoops([]*s2.Loop{loop})
}

// CoverPolygon converts s2 polygon to cell union and returns the respective cells
func CoverPolygon(p *s2.Polygon, maxLevel, minLevel int) (s2.CellUnion, []string, [][][]float64) {
	var tokens []string
	var s2cells [][][]float64

	rc := &s2.RegionCoverer{MaxLevel: maxLevel, MinLevel: minLevel, MaxCells: maxCells}
	r := s2.Region(p)
	covering := rc.Covering(r)

	for _, c := range covering {
		cell := s2.CellFromCellID(s2.CellIDFromToken(c.ToToken()))

		s2cells = append(s2cells, edgesOfCell(cell))

		tokens = append(tokens, c.ToToken())
	}
	return covering, tokens, s2cells
}

// CoverPoint converts a point to cell based on given level
func CoverPoint(p Point, maxLevel int) (s2.Cell, string, [][][]float64) {
	var s2cells [][][]float64

	cid := s2.CellFromLatLng(s2.LatLngFromDegrees(p.Lat, p.Lng)).ID().Parent(maxLevel)
	cell := s2.CellFromCellID(cid)
	token := cid.ToToken()

	s2cells = append(s2cells, edgesOfCell(cell))

	return cell, token, s2cells
}

func edgesOfCell(c s2.Cell) [][]float64 {
	var edges [][]float64
	for i := 0; i < 4; i++ {
		latLng := s2.LatLngFromPoint(c.Vertex(i))
		edges = append(edges, []float64{latLng.Lat.Degrees(), latLng.Lng.Degrees()})
	}
	return edges
}

func H3IndexToCoordinates(i h3.H3Index) [][]float64 {
	// Turns an H3 hexagon into a list of (lon, lat) pairs.
	var coords [][]float64

	boundary := h3.ToGeoBoundary(i)
	for _, b := range boundary {
		coords = append(coords, []float64{b.Longitude, b.Latitude})
	}

	return coords
}
