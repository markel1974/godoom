package config

import "github.com/markel1974/godoom/mr_tech/model/geometry"

// MD2Vertex represents a 3D vertex with position coordinates and UV texture mapping coordinates.
type MD2Vertex struct {
	Pos geometry.XYZ
	U   float32
	V   float32
}

// MD2Frame represents a 3D frame composed of a collection of triangles, each defined by three MD2Vertex points.
type MD2Frame struct {
	Triangles [][3]MD2Vertex
}

// MD2 represents a 3D model containing multiple frames, where each frame is a collection of triangular geometric data.
type MD2 struct {
	Frames []MD2Frame
}
