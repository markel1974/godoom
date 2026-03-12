package portal

import "github.com/markel1974/godoom/engine/model"

// QueueItem represents an element in a queue, linking a sector with positional and rendering boundaries in 3D space.
type QueueItem struct {
	sector *model.Sector
	x1     float64
	x2     float64
	y1t    float64
	y2t    float64
	y1b    float64
	y2b    float64
}

// NewQueueItem creates and returns a new instance of QueueItem with default values.
func NewQueueItem() *QueueItem {
	return &QueueItem{}
}

// Hash64 generates a unique 64-bit hash combining sector model ID and quantized position values for the QueueItem.
func (qi *QueueItem) Hash64() uint64 {
	return (uint64(qi.sector.ModelId) << 48) |
		((uint64(int64(qi.x1)) & 0xFFF) << 36) |
		((uint64(int64(qi.x2)) & 0xFFF) << 24) |
		((uint64(int64(qi.y1t)) & 0xFFF) << 12) |
		(uint64(int64(qi.y2t)) & 0xFFF)
}

// Update sets the sector and updates the coordinate boundaries for a QueueItem.
func (qi *QueueItem) Update(sector *model.Sector, x1 float64, x2 float64, y1t float64, y2t float64, y1b float64, y2b float64) {
	qi.sector = sector
	qi.x1 = x1
	qi.x2 = x2
	qi.y1t = y1t
	qi.y2t = y2t
	qi.y1b = y1b
	qi.y2b = y2b
}
