package pmtiles

import (
	"github.com/RoaringBitmap/roaring/roaring64"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/maptile"
	"github.com/paulmach/orb/maptile/tilecover"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/project"
)

func bitmapMultiPolygon(zoom uint8, multipolygon orb.MultiPolygon) (*roaring64.Bitmap, *roaring64.Bitmap) {
	boundarySet := roaring64.New()

	for _, polygon := range multipolygon {
		for _, ring := range polygon {
			boundaryTiles, _ := tilecover.Geometry(orb.LineString(ring), maptile.Zoom(zoom)) // TODO is this buffer-aware?
			for tile := range boundaryTiles {
				boundarySet.Add(ZxyToID(uint8(tile.Z), tile.X, tile.Y))
			}
		}
	}

	multipolygonProjected := project.MultiPolygon(multipolygon.Clone(), project.WGS84.ToMercator)

	interiorSet := roaring64.New()
	i := boundarySet.Iterator()
	for i.HasNext() {
		id := i.Next()
		if !boundarySet.Contains(id+1) && i.HasNext() {
			z, x, y := IDToZxy(id + 1)
			tile := maptile.New(x, y, maptile.Zoom(z))
			if planar.MultiPolygonContains(multipolygonProjected, project.Point(tile.Center(), project.WGS84.ToMercator)) {
				interiorSet.AddRange(id+1, i.PeekNext())
			}
		}
	}

	return boundarySet, interiorSet
}

func generalizeOr(r *roaring64.Bitmap, minzoom uint8) {
	if r.GetCardinality() == 0 {
		return
	}
	maxZ, _, _ := IDToZxy(r.ReverseIterator().Next())

	var temp *roaring64.Bitmap
	var toIterate *roaring64.Bitmap

	temp = roaring64.New()
	toIterate = r

	for currentZ := int(maxZ); currentZ > int(minzoom); currentZ-- {
		iter := toIterate.Iterator()
		for iter.HasNext() {
			parentID := ParentID(iter.Next())
			temp.Add(parentID)
		}
		toIterate = temp
		r.Or(temp)
		temp = roaring64.New()
	}
}

func generalizeAnd(r *roaring64.Bitmap) {
	if r.GetCardinality() == 0 {
		return
	}
	maxZ, _, _ := IDToZxy(r.ReverseIterator().Next())

	var temp *roaring64.Bitmap
	var toIterate *roaring64.Bitmap

	temp = roaring64.New()
	toIterate = r

	for currentZ := int(maxZ); currentZ > 0; currentZ-- {
		iter := toIterate.Iterator()
		filled := 0
		current := uint64(0) // check me...
		for iter.HasNext() {
			id := iter.Next()
			parentID := ParentID(id)
			if parentID == current {
				filled++
				if filled == 4 {
					temp.Add(parentID)
				}
			} else {
				current = parentID
				filled = 1
			}
		}
		toIterate = temp
		r.Or(temp)
		temp = roaring64.New()
	}
}
