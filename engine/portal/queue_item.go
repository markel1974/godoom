package portal

import "github.com/markel1974/godoom/engine/model"

// QueueItem represents an item in a rendering or processing queue tied to a specific sector and coordinates.
type QueueItem struct {
	sector *model.Sector
	x1     float64
	x2     float64
	y1t    float64
	y2t    float64
	y1b    float64
	y2b    float64
}

// Hash64 generates a unique 64-bit hash for the QueueItem based on its Sector and positional values.
func (qi *QueueItem) Hash64() uint64 {
	return (uint64(qi.sector.ModelId) << 48) |
		((uint64(int64(qi.x1)) & 0xFFF) << 36) |
		((uint64(int64(qi.x2)) & 0xFFF) << 24) |
		((uint64(int64(qi.y1t)) & 0xFFF) << 12) |
		(uint64(int64(qi.y2t)) & 0xFFF)
}

// Update sets the queue item's Sector and reassigns its position and boundary values provided as parameters.
func (qi *QueueItem) Update(sector *model.Sector, x1 float64, x2 float64, y1t float64, y2t float64, y1b float64, y2b float64) {
	qi.sector = sector
	qi.x1 = x1
	qi.x2 = x2
	qi.y1t = y1t
	qi.y2t = y2t
	qi.y1b = y1b
	qi.y2b = y2b
}
