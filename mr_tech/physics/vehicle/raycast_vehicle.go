package vehicle

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// RaycastFn defines the ray-testing signature against the physical world geometry.
// fromX, fromY, fromZ: ray origin.
// dirX, dirY, dirZ: normalized ray direction.
// maxDist: maximum raycast penetration distance.
// Returns: hit distance, surface normal (nx, ny, nz), and whether a hit occurred.
type RaycastFn func(fromX, fromY, fromZ, dirX, dirY, dirZ, maxDist float64) (dist float64, nx, ny, nz float64, hit bool)

// DriveType specifies which axles receive engine drive torque.
type DriveType int

const (
	DriveRearWheel  DriveType = iota // RWD: classic 1930s-1950s Mafia cars
	DriveFrontWheel                  // FWD: modern compact cars
	DriveAllWheel                    // AWD / 4WD: trucks and military jeeps
)

// RaycastVehicle implements a 4-wheel (or N-wheel) raycast car dynamics simulator.
// It applies suspension spring-damper forces, longitudinal traction/braking, lateral cornering
// and anti-roll bar distribution directly to an underlying 6-DoF rigid-body Entity.
type RaycastVehicle struct {
	entity    *rigid_body.Entity
	wheels    []*Wheel
	wheelsLen int
	raycastFn RaycastFn

	// Drivetrain configuration
	driveType            DriveType
	maxEngineTorque      float64 // Peak engine torque in N*m
	maxBrakeTorque       float64 // Peak service brake torque in N*m
	maxSteerAngle        float64 // Maximum steering lock (radians)
	antiRollBarStiffness float64 // Anti-roll bar coupling stiffness (N/m)
	downforceCoeff       float64 // Aerodynamic downforce coefficient

	// Driver controls
	throttleInput  float64 // [0..1]
	brakeInput     float64 // [0..1]
	steerInput     float64 // [-1..+1]
	handbrakeInput bool

	// Scratch structures to guarantee zero heap allocations in Update
	qChassisScratch *aabb.Quaternion
}

// NewRaycastVehicle creates a vehicle bound to the dynamic chassis entity with raycast terrain queries.
func NewRaycastVehicle(entity *rigid_body.Entity, raycastFn RaycastFn) *RaycastVehicle {
	return &RaycastVehicle{
		entity:               entity,
		wheels:               make([]*Wheel, 0, 8),
		raycastFn:            raycastFn,
		driveType:            DriveRearWheel,
		maxEngineTorque:      2500.0,
		maxBrakeTorque:       4000.0,
		maxSteerAngle:        math.Pi * 0.18, // ~32.4 degrees lock
		antiRollBarStiffness: 8000.0,
		downforceCoeff:       0.25,
		qChassisScratch:      aabb.NewQuaternionIdentity(),
	}
}

// AddWheel attaches a new wheel to the vehicle chassis at local position (hx, hy, hz).
func (v *RaycastVehicle) AddWheel(hx, hy, hz float64, radius, restLength, maxTravel float64, stiffness, dampingComp, dampingReb float64, frictionSlip, corneringStiffness float64, isFront, isSteerable, isDrive bool) int {
	w := NewWheel(hx, hy, hz, radius, restLength, maxTravel, stiffness, dampingComp, dampingReb, frictionSlip, corneringStiffness, isFront, isSteerable, isDrive)
	v.wheels = append(v.wheels, w)
	v.wheelsLen = len(v.wheels)
	return v.wheelsLen - 1
}

// SetupDefault configures a default four-wheel rear-drive sedan layout using the given wheel placement offsets.
func (v *RaycastVehicle) SetupDefault(halfWidth, frontDist, rearDist, downDist float64) {
	//configures a classic 1930s heavy 4-door sedan chassis.
	//width ~1.8m, length ~4.2m, height ~1.6m, mass ~1500 kg.
	const (
		radius      = 0.38   // 38 cm wheels
		restLen     = 0.35   // 35 cm suspension travel
		maxTravel   = 0.22   // 22 cm bump travel
		stiffness   = 32000. // 32 kN/m
		dampComp    = 2800.  // Bump damping
		dampReb     = 4200.  // Rebound damping
		grip        = 1.15   // Dry asphalt tire friction
		cornerStiff = 16.0   // Cornering stiffness
	)
	// Front-Left (0), Front-Right (1)
	v.AddWheel(halfWidth, frontDist, downDist, radius, restLen, maxTravel, stiffness, dampComp, dampReb, grip, cornerStiff, true, true, false)
	v.AddWheel(-halfWidth, frontDist, downDist, radius, restLen, maxTravel, stiffness, dampComp, dampReb, grip, cornerStiff, true, true, false)
	// Rear-Left (2), Rear-Right (3) - RWD
	v.AddWheel(halfWidth, -rearDist, downDist, radius, restLen, maxTravel, stiffness, dampComp, dampReb, grip, cornerStiff, false, false, true)
	v.AddWheel(-halfWidth, -rearDist, downDist, radius, restLen, maxTravel, stiffness, dampComp, dampReb, grip, cornerStiff, false, false, true)
	v.driveType = DriveRearWheel
	v.maxEngineTorque = 3000.0
	v.maxBrakeTorque = 4500.0
	v.antiRollBarStiffness = 9500.0
}

// GetWheel returns the wheel at index i.
func (v *RaycastVehicle) GetWheel(i int) *Wheel {
	if i < 0 || i >= v.wheelsLen {
		return nil
	}
	return v.wheels[i]
}

// WheelsLen returns the total number of wheels.
func (v *RaycastVehicle) WheelsLen() int {
	return v.wheelsLen
}

// GetEntity returns the chassis rigid-body entity.
func (v *RaycastVehicle) GetEntity() *rigid_body.Entity {
	return v.entity
}

// SetThrottle sets the accelerator pedal input in range [0..1].
func (v *RaycastVehicle) SetThrottle(throttle float64) {
	v.throttleInput = math.Max(0.0, math.Min(1.0, throttle))
}

// SetBrake sets the service brake pedal input in range [0..1].
func (v *RaycastVehicle) SetBrake(brake float64) {
	v.brakeInput = math.Max(0.0, math.Min(1.0, brake))
}

// SetSteer sets the steering wheel input in range [-1..+1] (negative = right, positive = left).
func (v *RaycastVehicle) SetSteer(steer float64) {
	v.steerInput = math.Max(-1.0, math.Min(1.0, steer))
}

// SetHandbrake engages or releases the emergency handbrake on rear wheels.
func (v *RaycastVehicle) SetHandbrake(engaged bool) {
	v.handbrakeInput = engaged
}

// SetDriveType sets the vehicle drivetrain layout (RWD, FWD, AWD).
func (v *RaycastVehicle) SetDriveType(dt DriveType) {
	v.driveType = dt
}

// SetMaxEngineTorque configures the peak engine output torque.
func (v *RaycastVehicle) SetMaxEngineTorque(torque float64) {
	v.maxEngineTorque = torque
}

// SetMaxBrakeTorque configures the peak service brake torque.
func (v *RaycastVehicle) SetMaxBrakeTorque(torque float64) {
	v.maxBrakeTorque = torque
}

// SetMaxSteerAngle configures the maximum wheel steering lock in radians.
func (v *RaycastVehicle) SetMaxSteerAngle(rad float64) {
	v.maxSteerAngle = rad
}

// SetAntiRollBarStiffness sets the anti-roll torsion bar coupling force.
func (v *RaycastVehicle) SetAntiRollBarStiffness(stiffness float64) {
	v.antiRollBarStiffness = stiffness
}

// Update advances the vehicle simulation by dt seconds.
// It executes raycasts, computes suspension reaction, tire friction (traction and cornering),
// anti-roll bar forces, and integrates wheel visuals without heap allocations.
func (v *RaycastVehicle) Update(dt float64) {
	if dt <= 0.0 || v.entity == nil || v.wheelsLen == 0 {
		return
	}

	qChassis := v.entity.GetQRotation()
	v.qChassisScratch.CopyFrom(qChassis)

	// Chassis local axes transformed to world space:
	// a0: lateral right (X)
	// a1: forward longitudinal (Y)
	// a2: vertical upward (Z)
	a0x, a0y, a0z, a1x, a1y, a1z, a2x, a2y, a2z := qChassis.ToAxes()
	cx, cy, cz := v.entity.GetCenter()
	vx, vy, vz := v.entity.GetVelocity()
	wx, wy, wz := v.entity.GetAngularVelocity()

	// 1. Distribute driver inputs across wheels
	driveWheelCount := 0
	for i := 0; i < v.wheelsLen; i++ {
		w := v.wheels[i]
		isDrive := false
		switch v.driveType {
		case DriveRearWheel:
			isDrive = !w.IsFront
		case DriveFrontWheel:
			isDrive = w.IsFront
		case DriveAllWheel:
			isDrive = true
		}
		w.IsDrive = isDrive
		if isDrive {
			driveWheelCount++
		}

		if w.IsSteerable {
			w.SteeringAngle = v.steerInput * v.maxSteerAngle
		} else {
			w.SteeringAngle = 0.0
		}

		// Brake distribution: 60% front, 40% rear for natural pitch balance
		brakeBias := 0.60
		if !w.IsFront {
			brakeBias = 0.40
		}
		w.BrakeTorque = v.brakeInput * v.maxBrakeTorque * brakeBias
		w.Handbrake = v.handbrakeInput && !w.IsFront
	}

	torquePerWheel := 0.0
	if driveWheelCount > 0 {
		torquePerWheel = (v.throttleInput * v.maxEngineTorque) / float64(driveWheelCount)
	}
	for i := 0; i < v.wheelsLen; i++ {
		w := v.wheels[i]
		if w.IsDrive {
			w.DriveTorque = torquePerWheel
		} else {
			w.DriveTorque = 0.0
		}
	}

	// Perform suspension raycasts and compute spring compression
	for i := 0; i < v.wheelsLen; i++ {
		w := v.wheels[i]

		// Mount point in world space
		mx := cx + w.HardpointX*a0x + w.HardpointY*a1x + w.HardpointZ*a2x
		my := cy + w.HardpointX*a0y + w.HardpointY*a1y + w.HardpointZ*a2y
		mz := cz + w.HardpointX*a0z + w.HardpointY*a1z + w.HardpointZ*a2z

		// Raycast down along chassis negative Z axis
		rayDirX := -a2x
		rayDirY := -a2y
		rayDirZ := -a2z
		maxRayDist := w.SuspensionRestLength + w.WheelRadius + w.SuspensionMaxTravel

		hitDist, nx, ny, nz, hit := v.raycastFn(mx, my, mz, rayDirX, rayDirY, rayDirZ, maxRayDist)
		if hit && hitDist > 1e-4 {
			w.InContact = true
			w.HitNormal[0] = nx
			w.HitNormal[1] = ny
			w.HitNormal[2] = nz
			w.HitPoint[0] = mx + rayDirX*hitDist
			w.HitPoint[1] = my + rayDirY*hitDist
			w.HitPoint[2] = mz + rayDirZ*hitDist

			// Compressed suspension length
			suspLen := hitDist - w.WheelRadius
			if suspLen < w.SuspensionRestLength-w.SuspensionMaxTravel {
				suspLen = w.SuspensionRestLength - w.SuspensionMaxTravel
			}
			if suspLen > w.SuspensionRestLength+w.SuspensionMaxTravel {
				suspLen = w.SuspensionRestLength + w.SuspensionMaxTravel
			}
			w.SuspensionLength = suspLen

			// Spring compression distance x > 0 when compressed
			compression := w.SuspensionRestLength - w.SuspensionLength

			// Spring velocity (positive = compressing, negative = extending)
			suspVel := (w.prevSuspensionLength - w.SuspensionLength) / dt
			w.prevSuspensionLength = w.SuspensionLength

			damping := w.SpringDampingCompression
			if suspVel < 0.0 {
				damping = w.SpringDampingRebound
			}

			// Hooke's spring force with viscous damping: F = k*x + c*v
			force := w.SpringStiffness*compression + damping*suspVel
			if force < 0.0 {
				force = 0.0
			}
			w.SuspensionForce = force
		} else {
			w.InContact = false
			w.SuspensionLength = w.SuspensionRestLength
			w.prevSuspensionLength = w.SuspensionRestLength
			w.SuspensionForce = 0.0
			w.SkidFactor = 0.0
		}
	}

	// Apply Anti-Roll Bar coupling (left vs right per axle)
	if v.antiRollBarStiffness > 0.0 && v.wheelsLen >= 4 {
		// Front axle: wheels 0 & 1
		v.applyAntiRollPair(v.wheels[0], v.wheels[1], a2x, a2y, a2z)
		// Rear axle: wheels 2 & 3
		v.applyAntiRollPair(v.wheels[2], v.wheels[3], a2x, a2y, a2z)
	}

	// Apply suspension normal forces to chassis
	for i := 0; i < v.wheelsLen; i++ {
		w := v.wheels[i]
		if !w.InContact || w.SuspensionForce <= 0.0 {
			continue
		}
		// Suspension pushes chassis upward along chassis vertical axis
		fx := a2x * w.SuspensionForce
		fy := a2y * w.SuspensionForce
		fz := a2z * w.SuspensionForce

		mx := cx + w.HardpointX*a0x + w.HardpointY*a1x + w.HardpointZ*a2x
		my := cy + w.HardpointX*a0y + w.HardpointY*a1y + w.HardpointZ*a2y
		mz := cz + w.HardpointX*a0z + w.HardpointY*a1z + w.HardpointZ*a2z

		v.applyWorldForceAtPoint(fx*dt, fy*dt, fz*dt, mx, my, mz, cx, cy, cz)
	}

	// Compute Tire Friction (Traction, Braking, Cornering Lateral Grip)
	for i := 0; i < v.wheelsLen; i++ {
		w := v.wheels[i]
		if !w.InContact || w.SuspensionForce <= 0.0 {
			continue
		}

		// Wheel orientation considering steering angle around chassis Z
		cosS := math.Cos(w.SteeringAngle)
		sinS := math.Sin(w.SteeringAngle)
		// Forward wheel direction on horizontal plane
		wfX := a1x*cosS + a0x*sinS
		wfY := a1y*cosS + a0y*sinS
		wfZ := a1z*cosS + a0z*sinS
		// Lateral right wheel direction
		wlX := a0x*cosS - a1x*sinS
		wlY := a0y*cosS - a1y*sinS
		wlZ := a0z*cosS - a1z*sinS

		// Velocity of tire contact point: V_contact = V_chassis + w x r
		rx := w.HitPoint[0] - cx
		ry := w.HitPoint[1] - cy
		rz := w.HitPoint[2] - cz
		vcx := vx + (wy*rz - wz*ry)
		vcy := vy + (wz*rx - wx*rz)
		vcz := vz + (wx*ry - wy*rx)

		// Decompose contact velocity into longitudinal and lateral speeds
		vLong := vcx*wfX + vcy*wfY + vcz*wfZ
		vLat := vcx*wlX + vcy*wlY + vcz*wlZ

		// Normal load on tire
		fn := w.SuspensionForce
		maxFrictionForce := fn * w.FrictionSlip

		// A. Lateral force (Cornering stiffness + Pacejka-like grip saturation)
		latForce := -vLat * w.CorneringStiffness * 800.0
		if latForce > maxFrictionForce {
			latForce = maxFrictionForce
		} else if latForce < -maxFrictionForce {
			latForce = -maxFrictionForce
		}

		// Skid factor proportional to relative side-slip
		w.SkidFactor = math.Min(1.0, math.Abs(vLat)/(math.Abs(vLong)+2.0))

		// B. Longitudinal force (Engine traction vs Braking)
		longForce := 0.0
		if w.Handbrake {
			// Rear locked: maximum opposing friction up to sliding limit
			longForce = -math.Copysign(math.Min(maxFrictionForce, math.Abs(vLong)*1200.0), vLong)
			w.SkidFactor = 1.0
		} else {
			// Drive traction
			if w.DriveTorque != 0.0 && w.WheelRadius > 0.0 {
				longForce += w.DriveTorque / w.WheelRadius
			}
			// Brake friction
			if w.BrakeTorque > 0.0 && w.WheelRadius > 0.0 {
				maxBrakeF := w.BrakeTorque / w.WheelRadius
				if math.Abs(vLong) > 0.05 {
					longForce -= math.Copysign(math.Min(maxBrakeF, math.Abs(vLong)*1500.0), vLong)
				}
			}
		}

		// Friction circle clamping: F_long^2 + F_lat^2 <= (mu * Fn)^2
		totalGripSq := longForce*longForce + latForce*latForce
		maxGripSq := maxFrictionForce * maxFrictionForce
		if totalGripSq > maxGripSq && totalGripSq > 1e-6 {
			scale := math.Sqrt(maxGripSq / totalGripSq)
			longForce *= scale
			latForce *= scale
			w.SkidFactor = math.Max(w.SkidFactor, 0.75)
		}

		// Combined force in world space
		totalFx := (wfX*longForce + wlX*latForce) * dt
		totalFy := (wfY*longForce + wlY*latForce) * dt
		totalFz := (wfZ*longForce + wlZ*latForce) * dt

		v.applyWorldForceAtPoint(totalFx, totalFy, totalFz, w.HitPoint[0], w.HitPoint[1], w.HitPoint[2], cx, cy, cz)

		// Update wheel rolling visual rotation angle
		wheelSpeed := vLong / w.WheelRadius
		if w.Handbrake {
			wheelSpeed = 0.0
		}
		w.AngularVelocity = wheelSpeed
		w.RotationAngle += wheelSpeed * dt
	}

	// Aerodynamic Downforce (prevents high-speed takeoff over crests)
	speedSq := vx*vx + vy*vy + vz*vz
	if speedSq > 25.0 && v.downforceCoeff > 0.0 {
		downforceImp := v.downforceCoeff * speedSq * dt
		v.entity.SetV(vx, vy, vz-downforceImp*v.entity.GetMass()*0.00005)
	}
}

// applyAntiRollPair applies torsion bar balancing between left and right wheels of one axle.
func (v *RaycastVehicle) applyAntiRollPair(wL, wR *Wheel, upX, upY, upZ float64) {
	if !wL.InContact && !wR.InContact {
		return
	}
	travelL := 1.0
	if wL.InContact {
		travelL = (wL.SuspensionLength - (wL.SuspensionRestLength - wL.SuspensionMaxTravel)) / (2.0 * wL.SuspensionMaxTravel)
	}
	travelR := 1.0
	if wR.InContact {
		travelR = (wR.SuspensionLength - (wR.SuspensionRestLength - wR.SuspensionMaxTravel)) / (2.0 * wR.SuspensionMaxTravel)
	}

	diff := travelL - travelR
	antiRollForce := diff * v.antiRollBarStiffness
	if wL.InContact {
		wL.SuspensionForce -= antiRollForce
		if wL.SuspensionForce < 0.0 {
			wL.SuspensionForce = 0.0
		}
	}
	if wR.InContact {
		wR.SuspensionForce += antiRollForce
		if wR.SuspensionForce < 0.0 {
			wR.SuspensionForce = 0.0
		}
	}
}

// applyWorldForceAtPoint applies a 3D impulse (fx, fy, fz) at world point (px, py, pz) to the vehicle chassis.
func (v *RaycastVehicle) applyWorldForceAtPoint(fx, fy, fz, px, py, pz, cx, cy, cz float64) {
	ent := v.entity
	mass := ent.GetMass()
	if mass <= 0.0 {
		return
	}
	invMass := 1.0 / mass

	// Linear delta velocity
	curVx, curVy, curVz := ent.GetVelocity()
	ent.SetV(curVx+fx*invMass, curVy+fy*invMass, curVz+fz*invMass)

	// Torque lever arm: r = p - COM
	rx := px - cx
	ry := py - cy
	rz := pz - cz

	// Torque = r x F
	tx := ry*fz - rz*fy
	ty := rz*fx - rx*fz
	tz := rx*fy - ry*fx

	// Rotational response using chassis inverse inertia tensor
	dwx, dwy, dwz := ent.ApplyWorldInvInertia(tx, ty, tz)
	ent.ApplyClampedAngularDelta(dwx, dwy, dwz)
}
