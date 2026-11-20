package vehicle

// Wheel represents one suspension and tire unit attached to the vehicle chassis.
type Wheel struct {
	// Local hardpoint on chassis
	HardpointX, HardpointY, HardpointZ float64

	// Suspension parameters
	SuspensionRestLength     float64 // Uncompressed spring length
	SuspensionMaxTravel      float64 // Maximum spring compression/droop
	SpringStiffness          float64 // Hooke's spring constant k (N/m)
	SpringDampingCompression float64 // Damper coefficient in compression (N*s/m)
	SpringDampingRebound     float64 // Damper coefficient in rebound (N*s/m)
	WheelRadius              float64 // Outer tire radius
	WheelWidth               float64 // Tire tread width

	// Tire friction & cornering properties
	FrictionSlip       float64 // Coulomb friction coefficient (mu)
	CorneringStiffness float64 // Cornering grip curve slope

	// Mechanical roles
	IsFront     bool // Located on front axle
	IsSteerable bool // Steers with wheel input
	IsDrive     bool // Powered by engine

	// Dynamic inputs
	SteeringAngle float64 // Current steer angle in radians
	DriveTorque   float64 // Drive torque in N*m
	BrakeTorque   float64 // Braking torque in N*m
	Handbrake     bool    // Handbrake lock

	// Instantaneous state computed during simulation
	InContact        bool       // Is tire contacting ground?
	HitPoint         [3]float64 // World contact point
	HitNormal        [3]float64 // Ground normal at contact point
	SuspensionLength float64    // Current spring length
	SuspensionForce  float64    // Upward spring force (N)
	RotationAngle    float64    // Cumulative rolling angle for visual rendering (rad)
	AngularVelocity  float64    // Angular wheel spin speed (rad/s)
	SkidFactor       float64    // Lateral skid intensity [0..1] for tire smoke / skidmarks

	// Internal state tracking
	prevSuspensionLength float64
}

func NewWheel(hx, hy, hz float64, radius, restLength, maxTravel float64, stiffness, dampingComp, dampingReb float64, frictionSlip, corneringStiffness float64, isFront, isSteerable, isDrive bool) *Wheel {
	return &Wheel{
		HardpointX:               hx,
		HardpointY:               hy,
		HardpointZ:               hz,
		WheelRadius:              radius,
		WheelWidth:               radius * 0.5,
		SuspensionRestLength:     restLength,
		SuspensionMaxTravel:      maxTravel,
		SpringStiffness:          stiffness,
		SpringDampingCompression: dampingComp,
		SpringDampingRebound:     dampingReb,
		FrictionSlip:             frictionSlip,
		CorneringStiffness:       corneringStiffness,
		IsFront:                  isFront,
		IsSteerable:              isSteerable,
		IsDrive:                  isDrive,
		SuspensionLength:         restLength,
		prevSuspensionLength:     restLength,
	}
}
