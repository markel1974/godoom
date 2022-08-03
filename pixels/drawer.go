package pixels

type CacheMode int

const (
	CacheModeDisable       CacheMode = 0
	CacheModePicture       CacheMode = 1
	CacheModePictureUpdate CacheMode = 2
	CacheModeUpdate        CacheMode = 3
)

// Drawer glues all the fundamental interfaces (ITarget, ITriangles, IPicture) into a coherent and the
// only intended usage pattern.
//
// Drawer makes it possible to draw any combination of ITriangles and IPicture onto any ITarget
// efficiently.
//
// To create a Drawer, just assign it's ITriangles and IPicture fields:
//
//   d := pixel.Drawer{ITriangles: t, IPicture: p}
//
// If ITriangles is nil, nothing will be drawn. If IPicture is nil, ITriangles will be drawn without a
// IPicture.
//
// Whenever you change the ITriangles, call Dirty to notify Drawer that ITriangles changed. You don't
// need to notify Drawer about a change of the IPicture.
//
// Note, that Drawer caches the results of MakePicture from Targets it's drawn to for each IPicture
// it's set to. What it means is that using a Drawer with an unbounded number of Pictures leads to a
// memory leak, since Drawer caches them and never forgets. In such a situation, create a new Drawer
// for each IPicture.
type Drawer struct {
	Triangles ITriangles
	Picture   IPicture
	Cached    CacheMode

	targets     map[ITarget]*drawerTarget
	allTargets  []*drawerTarget
	initialized bool
}

type drawerTarget struct {
	tris  ITargetTriangles
	pics  map[IPicture]ITargetPicture
	clean bool
	pic   ITargetPicture
}

func (d *Drawer) lazyInit() {
	if !d.initialized {
		d.targets = make(map[ITarget]*drawerTarget)
		d.initialized = true
	}
}

// Dirty marks the ITriangles of this Drawer as changed. If not called, changes will not be visible when drawing.
func (d *Drawer) Dirty() {
	d.lazyInit()
	for _, t := range d.allTargets {
		t.clean = false
	}
}

// Draw efficiently draws ITriangles with IPicture onto the provided ITarget.
//
// If ITriangles is nil, nothing will be drawn. If IPicture is nil, ITriangles will be drawn without a IPicture.
func (d *Drawer) Draw(t ITarget) {
	d.lazyInit()

	if d.Triangles == nil {
		return
	}

	dt := d.targets[t]
	if dt == nil {
		dt = &drawerTarget{pics: make(map[IPicture]ITargetPicture)}
		d.targets[t] = dt
		d.allTargets = append(d.allTargets, dt)
	}

	if dt.tris == nil {
		dt.tris = t.MakeTriangles(d.Triangles)
		dt.clean = true
	}

	if !dt.clean {
		dt.tris.SetLen(d.Triangles.Len())
		dt.tris.Update(d.Triangles)
		dt.clean = true
	}

	if d.Picture == nil {
		dt.tris.Draw()
		return
	}

	var pic ITargetPicture

	switch d.Cached {
	case CacheModePicture:
		if pic = dt.pics[d.Picture]; pic == nil {
			pic = t.MakePicture(d.Picture)
			dt.pics[d.Picture] = pic
		}
	case CacheModePictureUpdate:
		if pic = dt.pics[d.Picture]; pic == nil {
			pic = t.MakePicture(d.Picture)
			dt.pics[d.Picture] = pic
		} else {
			pic.Update(d.Picture)
		}
	case CacheModeUpdate:
		if dt.pic == nil {
			pic = t.MakePicture(d.Picture)
			dt.pic = pic
		} else {
			pic = dt.pic
			pic.Update(d.Picture)
		}
	default:
		pic = t.MakePicture(d.Picture)
	}
	pic.Draw(dt.tris)
}
