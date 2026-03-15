package main

import (
	"math"
	"testing"
)

const tolerance = 1e-6

func almostEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

func TestComputeOrbitalParams(t *testing.T) {
	tests := []struct {
		name     string
		altitude float64 // meters
		wantVMin float64 // minimum expected orbital velocity m/s
		wantVMax float64 // maximum expected orbital velocity m/s
	}{
		{"700km", 700e3, 7400, 7600},
		{"725km", 725e3, 7400, 7600},
		{"750km", 750e3, 7400, 7600},
		{"775km", 775e3, 7350, 7550},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orb := ComputeOrbitalParams(tt.altitude)
			if orb.Velocity < tt.wantVMin || orb.Velocity > tt.wantVMax {
				t.Errorf("Velocity = %f, want in [%f, %f]", orb.Velocity, tt.wantVMin, tt.wantVMax)
			}
			if orb.GroundVelocity >= orb.Velocity {
				t.Errorf("Ground velocity %f should be less than orbital velocity %f", orb.GroundVelocity, orb.Velocity)
			}
			if orb.Radius != EarthRadius+tt.altitude {
				t.Errorf("Radius = %f, want %f", orb.Radius, EarthRadius+tt.altitude)
			}
		})
	}
}

func TestComputeOrbitalParams_700km(t *testing.T) {
	orb := ComputeOrbitalParams(700e3)
	// v = sqrt(GM/r) = sqrt(3.986004418e14 / 7071000) = sqrt(5.6369e7) ≈ 7508 m/s
	expectedV := math.Sqrt(GM / (EarthRadius + 700e3))
	if !almostEqual(orb.Velocity, expectedV, 0.01) {
		t.Errorf("Velocity = %f, want %f", orb.Velocity, expectedV)
	}
	expectedVg := expectedV * EarthRadius / (EarthRadius + 700e3)
	if !almostEqual(orb.GroundVelocity, expectedVg, 0.01) {
		t.Errorf("GroundVelocity = %f, want %f", orb.GroundVelocity, expectedVg)
	}
}

func TestComputeGroundFootprint(t *testing.T) {
	// Section 1.1: f=0.75m, d=5e-6m, array=0.30m, H=700km
	gf := ComputeGroundFootprint(0.75, 5e-6, 0.30, 700e3)

	// IFOV = 5e-6 / 0.75 = 6.6667e-6 rad
	expectedIFOV := 5e-6 / 0.75
	if !almostEqual(gf.IFOV, expectedIFOV, 1e-12) {
		t.Errorf("IFOV = %e, want %e", gf.IFOV, expectedIFOV)
	}

	// GSD = IFOV * 700000 = 6.6667e-6 * 700000 ≈ 4.667 m
	expectedGSD := expectedIFOV * 700e3
	if !almostEqual(gf.GSD, expectedGSD, 1e-6) {
		t.Errorf("GSD = %f, want %f", gf.GSD, expectedGSD)
	}

	// Footprint per side = (0.30/0.75) * 700000 = 0.4 * 700000 = 280000 m = 280 km
	expectedFootprint := (0.30 / 0.75) * 700e3
	if !almostEqual(gf.FootprintSide, expectedFootprint, 1e-3) {
		t.Errorf("FootprintSide = %f, want %f", gf.FootprintSide, expectedFootprint)
	}
}

func TestComputeGroundFootprint_AllAltitudes(t *testing.T) {
	altitudes := []struct {
		name string
		h    float64
	}{
		{"700km", 700e3},
		{"725km", 725e3},
		{"750km", 750e3},
		{"775km", 775e3},
	}
	for _, a := range altitudes {
		t.Run(a.name, func(t *testing.T) {
			gf := ComputeGroundFootprint(0.75, 5e-6, 0.30, a.h)
			if gf.IFOV <= 0 {
				t.Error("IFOV should be positive")
			}
			if gf.GSD <= 0 {
				t.Error("GSD should be positive")
			}
			// Footprint should scale linearly with altitude
			expectedFootprint := (0.30 / 0.75) * a.h
			if !almostEqual(gf.FootprintSide, expectedFootprint, 0.01) {
				t.Errorf("FootprintSide = %f, want %f", gf.FootprintSide, expectedFootprint)
			}
		})
	}
}

func TestComputeFocalLength(t *testing.T) {
	// Section 1.2: d=20e-6m, GSD=50m, H=700km
	// f = d * H / GSD = 20e-6 * 700000 / 50 = 0.28 m
	fl := ComputeFocalLength(20e-6, 50.0, 700e3)
	expectedF := 20e-6 * 700e3 / 50.0
	if !almostEqual(fl.FocalLength, expectedF, 1e-9) {
		t.Errorf("FocalLength = %f, want %f", fl.FocalLength, expectedF)
	}
}

func TestComputeFocalLength_AllAltitudes(t *testing.T) {
	altitudes := []float64{700e3, 725e3, 750e3, 775e3}
	for _, h := range altitudes {
		fl := ComputeFocalLength(20e-6, 50.0, h)
		expected := 20e-6 * h / 50.0
		if !almostEqual(fl.FocalLength, expected, 1e-9) {
			t.Errorf("H=%.0f: FocalLength = %f, want %f", h, fl.FocalLength, expected)
		}
		// Focal length should increase with altitude (need longer focal length for same GSD at higher altitude)
		if h > 700e3 {
			fl700 := ComputeFocalLength(20e-6, 50.0, 700e3)
			if fl.FocalLength <= fl700.FocalLength {
				t.Errorf("H=%.0f: focal length should be > focal length at 700km", h)
			}
		}
	}
}

func TestComputeTelescopeAnalysis_FOVAndSwath(t *testing.T) {
	// Section 1.3.1: D_lens=0.30m, f=1.0m, D_fp=0.25m
	orb := ComputeOrbitalParams(700e3)
	ta := ComputeTelescopeAnalysis(0.30, 1.0, 0.25, 30.0, 8, orb)

	// FOV = 0.25 / 1.0 = 0.25 rad
	if !almostEqual(ta.FOV, 0.25, 1e-10) {
		t.Errorf("FOV = %f, want 0.25", ta.FOV)
	}

	// FOV in degrees = 0.25 * 180/pi ≈ 14.3239 deg
	expectedFOVDeg := 0.25 * 180.0 / math.Pi
	if !almostEqual(ta.FOVDeg, expectedFOVDeg, 1e-4) {
		t.Errorf("FOVDeg = %f, want %f", ta.FOVDeg, expectedFOVDeg)
	}

	// Swath = 2 * H * tan(FOV/2) = 2 * 700000 * tan(0.125) ≈ 175,913 m
	expectedSwath := 2.0 * 700e3 * math.Tan(0.125)
	if !almostEqual(ta.SwathWidth, expectedSwath, 1.0) {
		t.Errorf("SwathWidth = %f, want %f", ta.SwathWidth, expectedSwath)
	}
}

func TestComputeTelescopeAnalysis_PushBroom(t *testing.T) {
	orb := ComputeOrbitalParams(700e3)
	ta := ComputeTelescopeAnalysis(0.30, 1.0, 0.25, 30.0, 8, orb)

	// Detector size = f * pixel / H = 1.0 * 30 / 700000 ≈ 4.286e-5 m
	expectedDetSize := 1.0 * 30.0 / 700e3
	if !almostEqual(ta.DetectorSize, expectedDetSize, 1e-10) {
		t.Errorf("DetectorSize = %e, want %e", ta.DetectorSize, expectedDetSize)
	}

	// Number of detectors = D_fp / detSize = 0.25 / 4.286e-5 ≈ 5833
	expectedN := int(0.25 / expectedDetSize)
	if ta.NumDetectors != expectedN {
		t.Errorf("NumDetectors = %d, want %d", ta.NumDetectors, expectedN)
	}

	// Dwell time = pixel / (3 * Vg)
	expectedDwell := 30.0 / (3.0 * orb.GroundVelocity)
	if !almostEqual(ta.PushBroomDwellTime, expectedDwell, 1e-10) {
		t.Errorf("PushBroomDwellTime = %e, want %e", ta.PushBroomDwellTime, expectedDwell)
	}

	// Dwell time should be positive and reasonable (on the order of milliseconds)
	if ta.PushBroomDwellTime <= 0 || ta.PushBroomDwellTime > 1.0 {
		t.Errorf("PushBroomDwellTime = %e, should be between 0 and 1 second", ta.PushBroomDwellTime)
	}
}

func TestComputeTelescopeAnalysis_Whiskbroom(t *testing.T) {
	orb := ComputeOrbitalParams(700e3)
	ta := ComputeTelescopeAnalysis(0.30, 1.0, 0.25, 30.0, 8, orb)

	// Scan time = pixel / Vg
	expectedScanTime := 30.0 / orb.GroundVelocity
	if !almostEqual(ta.WhiskScanTime, expectedScanTime, 1e-10) {
		t.Errorf("WhiskScanTime = %e, want %e", ta.WhiskScanTime, expectedScanTime)
	}

	// Mirror scan rate = FOV / scanTime
	expectedRate := 0.25 / expectedScanTime
	if !almostEqual(ta.WhiskMirrorScanRate, expectedRate, 1e-6) {
		t.Errorf("WhiskMirrorScanRate = %f, want %f", ta.WhiskMirrorScanRate, expectedRate)
	}

	// Dwell time should be much smaller than scan time
	if ta.WhiskDwellTime >= ta.WhiskScanTime {
		t.Errorf("Whiskbroom dwell time (%e) should be less than scan time (%e)",
			ta.WhiskDwellTime, ta.WhiskScanTime)
	}
}

func TestComputeTelescopeAnalysis_MultiElementWhiskbroom(t *testing.T) {
	orb := ComputeOrbitalParams(700e3)
	ta := ComputeTelescopeAnalysis(0.30, 1.0, 0.25, 30.0, 8, orb)

	// Multi-element scan time should be 8x the single whiskbroom scan time
	if !almostEqual(ta.MultiElemScanTime, 8.0*ta.WhiskScanTime, 1e-10) {
		t.Errorf("MultiElemScanTime = %e, want %e (8x whisk scan time)",
			ta.MultiElemScanTime, 8.0*ta.WhiskScanTime)
	}

	// Multi-element dwell time should be 8x the single whiskbroom dwell time
	if !almostEqual(ta.MultiElemDwellTime, 8.0*ta.WhiskDwellTime, 1e-10) {
		t.Errorf("MultiElemDwellTime = %e, want %e (8x whisk dwell time)",
			ta.MultiElemDwellTime, 8.0*ta.WhiskDwellTime)
	}

	// Multi-element mirror scan rate should be 1/8 the single whiskbroom rate
	if !almostEqual(ta.MultiElemMirrorScanRate, ta.WhiskMirrorScanRate/8.0, 1e-6) {
		t.Errorf("MultiElemMirrorScanRate = %f, want %f (1/8 whisk rate)",
			ta.MultiElemMirrorScanRate, ta.WhiskMirrorScanRate/8.0)
	}
}

func TestComputeTelescopeAnalysis_AllAltitudes(t *testing.T) {
	altitudes := []float64{700e3, 725e3, 750e3, 775e3}

	for _, h := range altitudes {
		orb := ComputeOrbitalParams(h)
		ta := ComputeTelescopeAnalysis(0.30, 1.0, 0.25, 30.0, 8, orb)

		// FOV should be independent of altitude (it's a sensor property)
		if !almostEqual(ta.FOV, 0.25, 1e-10) {
			t.Errorf("H=%.0f: FOV = %f, want 0.25", h, ta.FOV)
		}

		// Swath width should increase with altitude
		if h > 700e3 {
			orb700 := ComputeOrbitalParams(700e3)
			ta700 := ComputeTelescopeAnalysis(0.30, 1.0, 0.25, 30.0, 8, orb700)
			if ta.SwathWidth <= ta700.SwathWidth {
				t.Errorf("H=%.0f: swath width should increase with altitude", h)
			}
		}

		// Number of detectors should increase with altitude (larger swath, same pixel size)
		if h > 700e3 {
			orb700 := ComputeOrbitalParams(700e3)
			ta700 := ComputeTelescopeAnalysis(0.30, 1.0, 0.25, 30.0, 8, orb700)
			if ta.NumDetectors <= ta700.NumDetectors {
				t.Errorf("H=%.0f: number of detectors should increase with altitude", h)
			}
		}
	}
}
