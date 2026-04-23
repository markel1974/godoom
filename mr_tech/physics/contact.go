package physics

import "math"

// Contact represents a physics collision contact point between two things.
// A and B are the things involved in the collision.
// Nx, Ny, Nz represent the normal vector of the contact.
// Penetration denotes the depth of the intersection between things.
// AccumulatedImpulse tracks the accumulated impulse applied during resolution.
type Contact struct {
	a, b               *Entity
	nx, ny, nz         float64
	penetration        float64
	accumulatedImpulse float64
}

// Update updates the contact with new things, normal vector components, and penetration depth, resetting impulse to zero.
func (c *Contact) Update(a, b *Entity, nx, ny, nz float64, penetration float64) {
	c.a = a
	c.b = b
	c.nx = nx
	c.ny = ny
	c.nz = nz
	c.penetration = penetration
	c.accumulatedImpulse = 0
}

// Resolve handles the collision response between two things by resolving penetration and applying impulses.
func (c *Contact) Resolve() {
	// 1. Velocità relativa
	rvX := c.a.GetVx() - c.b.GetVx()
	rvY := c.a.GetVy() - c.b.GetVy()
	rvZ := c.a.GetVz() - c.b.GetVz()
	// 2. Velocità lungo la normale
	velAlongNormal := (rvX * c.nx) + (rvY * c.ny) + (rvZ * c.nz)
	// Se si stanno già separando, il vincolo è soddisfatto
	if velAlongNormal > 0 {
		return
	}
	// BAUMGARTE STABILIZATION (Positional Bias)
	const slop = 0.05   // Tolleranza: permette agli oggetti di penetrare leggermente senza vibrare
	const percent = 0.2 // Corregge il 20% dell'errore ad ogni frame
	// Calcoliamo la velocità extra necessaria per spingerli fuori
	bias := math.Max(c.penetration-slop, 0.0) * percent
	// Se la velocità relativa (velAlongNormal) è già sufficiente a separarli, ignoriamo il bias
	// altrimenti lo aggiungiamo al calcolo dell'impulso
	e := math.Min(c.a.GetRestitution(), c.b.GetRestitution())
	invMassA := c.a.GetInvMass()
	invMassB := c.b.GetInvMass()
	totalInvMass := invMassA + invMassB
	// PREVIENI LA DIVISIONE PER ZERO (Entrambi gli oggetti statici)
	if totalInvMass <= 0.00001 {
		return
	}
	// Aggiungiamo il termine "bias" all'equazione di J
	j := (-(1.0 + e) * velAlongNormal) + bias
	j /= invMassA + invMassB
	// PASSAGGIO PROIETTIVO (PGS)
	// Conserviamo l'impulso calcolato e lo proiettiamo per non "tirare" i corpi
	oldImpulse := c.accumulatedImpulse
	c.accumulatedImpulse = math.Max(oldImpulse+j, 0.0)
	// L'impulso effettivo da applicare in questa singola iterazione
	jDelta := c.accumulatedImpulse - oldImpulse
	// 4. Applica il delta di velocità
	impulseX := jDelta * c.nx
	impulseY := jDelta * c.ny
	impulseZ := jDelta * c.nz
	c.a.AddV(impulseX*invMassA, impulseY*invMassA, impulseZ*invMassA)
	c.b.SubV(impulseX*invMassB, impulseY*invMassB, impulseZ*invMassB)
}
