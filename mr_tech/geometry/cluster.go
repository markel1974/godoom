package geometry

import (
	"math"
	"sort"
)

// Cluster represents a 3D cluster with associated point indices and dimensions.
type Cluster struct {
	Indices []int
	d       XYZ
}

// NewCluster creates a new Cluster with the given indices and bounding box dimensions (dx, dy, dz).
func NewCluster(indices []int, dx, dy, dz float64) *Cluster {
	return &Cluster{Indices: indices, d: XYZ{X: dx, Y: dy, Z: dz}}
}

// Clusters is a type designed for handling clustering operations, particularly on 3D geometric data structures.
type Clusters struct {
}

// NewClusters initializes and returns a new instance of the Clusters type.
func NewClusters() *Clusters {
	return &Clusters{}
}

// Create partitions a set of 3D points into clusters based on bounding box dimensions and a target number of volumes.
func (cs *Clusters) Create(centroids []XYZ, targetVolumes int, totalTriangles int) []*Cluster {
	allIndices := make([]int, totalTriangles)
	for i := 0; i < totalTriangles; i++ {
		allIndices[i] = i
	}
	dx, dy, dz := cs.CalcBBox(centroids, allIndices)
	clusters := []*Cluster{NewCluster(allIndices, dx, dy, dz)}

	for len(clusters) < targetVolumes {
		bestIdx := 0
		bestCount := len(clusters[0].Indices)
		for i := 1; i < len(clusters); i++ {
			if len(clusters[i].Indices) > bestCount {
				bestCount = len(clusters[i].Indices)
				bestIdx = i
			}
		}

		c := clusters[bestIdx]
		if len(c.Indices) <= 1 {
			break
		}

		if c.d.X >= c.d.Y && c.d.X >= c.d.Z {
			sort.Slice(c.Indices, func(i, j int) bool {
				return centroids[c.Indices[i]].X < centroids[c.Indices[j]].X
			})
		} else if c.d.Y >= c.d.Z {
			sort.Slice(c.Indices, func(i, j int) bool {
				return centroids[c.Indices[i]].Y < centroids[c.Indices[j]].Y
			})
		} else {
			sort.Slice(c.Indices, func(i, j int) bool {
				return centroids[c.Indices[i]].Z < centroids[c.Indices[j]].Z
			})
		}

		mid := len(c.Indices) / 2
		leftIndices := c.Indices[:mid]
		rightIndices := c.Indices[mid:]

		ldx, ldy, ldz := cs.CalcBBox(centroids, leftIndices)
		rdx, rdy, rdz := cs.CalcBBox(centroids, rightIndices)

		clusters[bestIdx] = NewCluster(leftIndices, ldx, ldy, ldz)
		clusters = append(clusters, NewCluster(rightIndices, rdx, rdy, rdz))
	}

	return clusters
}

// CalcBBox calculates the bounding box dimensions (dx, dy, dz) for a set of 3D points specified by the given indices.
func (cs *Clusters) CalcBBox(centroids []XYZ, indices []int) (dx, dy, dz float64) {
	if len(indices) == 0 {
		return 0, 0, 0
	}
	minX, maxX := math.MaxFloat64, -math.MaxFloat64
	minY, maxY := math.MaxFloat64, -math.MaxFloat64
	minZ, maxZ := math.MaxFloat64, -math.MaxFloat64
	for _, idx := range indices {
		pt := centroids[idx]
		if pt.X < minX {
			minX = pt.X
		}
		if pt.X > maxX {
			maxX = pt.X
		}
		if pt.Y < minY {
			minY = pt.Y
		}
		if pt.Y > maxY {
			maxY = pt.Y
		}
		if pt.Z < minZ {
			minZ = pt.Z
		}
		if pt.Z > maxZ {
			maxZ = pt.Z
		}
	}
	return maxX - minX, maxY - minY, maxZ - minZ
}
