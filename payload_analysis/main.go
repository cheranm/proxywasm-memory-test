// Package main implements numerical instrument analysis for an Earth observation
// satellite imaging payload in low Earth orbit (LEO). It computes sensor geometry,
// detector configurations, and scanning architecture parameters based on
// Space Mission Engineering: The New SMAD, Chapter 17.1 and 17.2.
package main

import (
	"fmt"
	"math"
)

// Physical constants
const (
	// EarthRadius is the mean radius of the Earth in meters.
	EarthRadius = 6371000.0 // m
	// GM is the standard gravitational parameter for Earth in m^3/s^2.
	GM = 3.986004418e14 // m^3/s^2
)

// OrbitalParams holds computed orbital parameters for a circular LEO orbit.
type OrbitalParams struct {
	Altitude      float64 // orbit altitude in meters
	Radius        float64 // orbital radius (Re + h) in meters
	Velocity      float64 // orbital velocity in m/s
	GroundVelocity float64 // ground-track velocity in m/s
}

// ComputeOrbitalParams computes orbital parameters for a circular orbit
// at the given altitude (in meters).
//
// Orbital velocity: v = sqrt(GM / r), where r = Re + h  (SMAD Eq. for circular orbit)
// Ground-track velocity: Vg = v * Re / r  (accounts for Earth's curvature)
func ComputeOrbitalParams(altitudeM float64) OrbitalParams {
	r := EarthRadius + altitudeM
	v := math.Sqrt(GM / r)
	vg := v * EarthRadius / r
	return OrbitalParams{
		Altitude:       altitudeM,
		Radius:         r,
		Velocity:       v,
		GroundVelocity: vg,
	}
}

// GroundFootprintResult holds the results from Section 1.1.
type GroundFootprintResult struct {
	IFOV           float64 // instantaneous field of view in radians
	GSD            float64 // ground sample distance (single pixel) in meters
	FootprintSide  float64 // total ground footprint per side in meters
}

// ComputeGroundFootprint computes the ground footprint of a nadir-pointing
// remote sensor (Section 1.1).
//
// Parameters:
//   focalLength     - focal length of the telescope in meters
//   detectorSize    - individual detector element size in meters
//   arraySize       - detector array format size (per side) in meters
//   altitudeM       - orbit altitude in meters
//
// Equations (SMAD Ch. 17):
//   IFOV = detectorSize / focalLength           (instantaneous field of view, radians)
//   GSD  = IFOV * H                             (ground sample distance per pixel)
//   Footprint = (arraySize / focalLength) * H   (total ground footprint per side)
func ComputeGroundFootprint(focalLength, detectorSize, arraySize, altitudeM float64) GroundFootprintResult {
	ifov := detectorSize / focalLength
	gsd := ifov * altitudeM
	footprint := (arraySize / focalLength) * altitudeM
	return GroundFootprintResult{
		IFOV:          ifov,
		GSD:           gsd,
		FootprintSide: footprint,
	}
}

// FocalLengthResult holds the results from Section 1.2.
type FocalLengthResult struct {
	FocalLength float64 // computed focal length in meters
}

// ComputeFocalLength computes the focal length needed to achieve a given ground
// projected IFOV (Section 1.2).
//
// Parameters:
//   detectorSize  - detector element size in meters
//   targetGSD     - desired ground projected IFOV (ground sample distance) in meters
//   altitudeM     - orbit altitude in meters
//
// From SMAD Ch. 17:
//   IFOV = d / f = GSD / H
//   Therefore: f = d * H / GSD
func ComputeFocalLength(detectorSize, targetGSD, altitudeM float64) FocalLengthResult {
	f := detectorSize * altitudeM / targetGSD
	return FocalLengthResult{FocalLength: f}
}

// TelescopeAnalysisResult holds the results from Section 1.3.
type TelescopeAnalysisResult struct {
	// 1.3.1 - FOV and Swath
	FOV        float64 // full field of view in radians
	FOVDeg     float64 // full field of view in degrees
	SwathWidth float64 // swath width on ground in meters

	// 1.3.2 - Push-broom sensor
	DetectorSize       float64 // required detector element size in meters
	NumDetectors       int     // number of detectors in single line
	PushBroomDwellTime float64 // dwell time for smearing < 1/3 pixel, in seconds

	// 1.3.3 - Whiskbroom sensor
	WhiskScanTime        float64 // scan period (one line) in seconds
	WhiskDwellTime       float64 // dwell time per pixel in seconds
	WhiskMirrorScanRate  float64 // mirror scan rate in rad/s

	// 1.3.4 - Multi-element whiskbroom (8 along-track elements)
	MultiElemScanTime       float64 // scan period with 8 elements in seconds
	MultiElemDwellTime      float64 // dwell time per pixel in seconds
	MultiElemMirrorScanRate float64 // mirror scan rate in rad/s
}

// ComputeTelescopeAnalysis performs the full multi-parameter analysis (Section 1.3).
//
// Parameters:
//   lensDiameter      - lens (aperture) diameter in meters
//   focalLength       - focal length in meters
//   focalPlaneDiam    - focal plane diameter in meters
//   pixelSizeGround   - desired ground pixel size in meters (for push-broom/whiskbroom)
//   numAlongTrack     - number of along-track elements for multi-element whiskbroom
//   orb               - orbital parameters
//
// Equations (SMAD Ch. 17, Table 17.9):
//
// 1.3.1 FOV and Swath Width:
//   FOV = focalPlaneDiam / focalLength                (angular field of view, radians)
//   SwathWidth = 2 * H * tan(FOV/2)                  (ground swath width)
//
// 1.3.2 Push-broom sensor:
//   detectorSize = focalLength * pixelSizeGround / H  (required detector element size)
//   numDetectors = focalPlaneDiam / detectorSize       (number of detectors across focal plane)
//   dwellTime = pixelSizeGround / (3 * Vg)            (for smearing < 1/3 pixel)
//
//   The dwell time (integration time) must satisfy:
//     Vg * t_dwell < (1/3) * pixelSize
//     => t_dwell < pixelSize / (3 * Vg)
//
// 1.3.3 Whiskbroom sensor:
//   scanTime = pixelSizeGround / Vg                   (time to advance one pixel along-track)
//   numCrossTrack = swathWidth / pixelSizeGround       (pixels per scan line)
//   dwellTime = scanTime / numCrossTrack               (time per pixel)
//   mirrorScanRate = FOV / scanTime                    (angular scan rate, rad/s)
//
// 1.3.4 Multi-element whiskbroom (N along-track elements):
//   scanTime = N * pixelSizeGround / Vg               (N lines captured per scan)
//   dwellTime = scanTime / numCrossTrack               (increased dwell per pixel)
//   mirrorScanRate = FOV / scanTime                    (reduced mirror rate)
func ComputeTelescopeAnalysis(lensDiameter, focalLength, focalPlaneDiam, pixelSizeGround float64, numAlongTrack int, orb OrbitalParams) TelescopeAnalysisResult {
	H := orb.Altitude
	Vg := orb.GroundVelocity

	// 1.3.1 FOV and Swath Width
	fov := focalPlaneDiam / focalLength
	fovDeg := fov * 180.0 / math.Pi
	swath := 2.0 * H * math.Tan(fov/2.0)

	// 1.3.2 Push-broom sensor
	detSize := focalLength * pixelSizeGround / H
	nDet := int(focalPlaneDiam / detSize)
	// Dwell time for smearing < 1/3 pixel:
	// smearing = Vg * t_dwell; require smearing < (1/3) * pixelSize
	// t_dwell = pixelSize / (3 * Vg)
	pbDwell := pixelSizeGround / (3.0 * Vg)

	// 1.3.3 Whiskbroom sensor
	scanTime := pixelSizeGround / Vg
	nCross := math.Round(swath / pixelSizeGround)
	wbDwell := scanTime / nCross
	wbMirrorRate := fov / scanTime

	// 1.3.4 Multi-element whiskbroom
	meScanTime := float64(numAlongTrack) * pixelSizeGround / Vg
	meDwell := meScanTime / nCross
	meMirrorRate := fov / meScanTime

	return TelescopeAnalysisResult{
		FOV:        fov,
		FOVDeg:     fovDeg,
		SwathWidth: swath,

		DetectorSize:       detSize,
		NumDetectors:       nDet,
		PushBroomDwellTime: pbDwell,

		WhiskScanTime:       scanTime,
		WhiskDwellTime:      wbDwell,
		WhiskMirrorScanRate: wbMirrorRate,

		MultiElemScanTime:       meScanTime,
		MultiElemDwellTime:      meDwell,
		MultiElemMirrorScanRate: meMirrorRate,
	}
}

func main() {
	altitudes := []float64{700e3, 725e3, 750e3, 775e3}
	labels := []string{
		"700 km (Last A-L, First A-L)",
		"725 km (Last M-Z, First A-L)",
		"750 km (Last A-L, First M-Z)",
		"775 km (Last M-Z, First M-Z)",
	}

	for i, alt := range altitudes {
		orb := ComputeOrbitalParams(alt)

		fmt.Println("=================================================================")
		fmt.Printf("  Orbit: %s\n", labels[i])
		fmt.Println("=================================================================")
		fmt.Printf("  Altitude:         %.0f km\n", alt/1e3)
		fmt.Printf("  Orbital Radius:   %.3f km\n", orb.Radius/1e3)
		fmt.Printf("  Orbital Velocity: %.2f m/s\n", orb.Velocity)
		fmt.Printf("  Ground Velocity:  %.2f m/s\n", orb.GroundVelocity)
		fmt.Println()

		// ---------------------------------------------------------------
		// 1.1 Ground Footprint Analysis
		// ---------------------------------------------------------------
		fmt.Println("----- 1.1 Ground Footprint Analysis -----")
		fmt.Println("  Given: f=75cm, D_aperture=15cm, d=5μm, array=30cm per side")
		gf := ComputeGroundFootprint(0.75, 5e-6, 0.30, alt)
		fmt.Printf("  IFOV             = %.4e rad (= d/f = 5e-6/0.75)\n", gf.IFOV)
		fmt.Printf("  GSD (per pixel)  = %.3f m   (= IFOV * H)\n", gf.GSD)
		fmt.Printf("  Ground Footprint = %.2f km x %.2f km  (= (array/f) * H per side)\n",
			gf.FootprintSide/1e3, gf.FootprintSide/1e3)
		fmt.Println()

		// ---------------------------------------------------------------
		// 1.2 Focal Length Analysis
		// ---------------------------------------------------------------
		fmt.Println("----- 1.2 Focal Length Analysis -----")
		fmt.Println("  Given: d=20μm, target GSD=50m")
		fl := ComputeFocalLength(20e-6, 50.0, alt)
		fmt.Printf("  Focal Length = %.4f m = %.2f cm  (= d*H/GSD)\n", fl.FocalLength, fl.FocalLength*100)
		fmt.Println()

		// ---------------------------------------------------------------
		// 1.3 Multi-Parameter Analysis
		// ---------------------------------------------------------------
		fmt.Println("----- 1.3 Multi-Parameter Analysis -----")
		fmt.Println("  Given: D_lens=30cm, f=1m, D_fp=25cm, pixel=30m x 30m")
		ta := ComputeTelescopeAnalysis(0.30, 1.0, 0.25, 30.0, 8, orb)

		fmt.Println()
		fmt.Println("  1.3.1 FOV and Swath Width:")
		fmt.Printf("    FOV        = %.4f rad = %.4f deg  (= D_fp/f)\n", ta.FOV, ta.FOVDeg)
		fmt.Printf("    Swath Width = %.2f km  (= 2*H*tan(FOV/2))\n", ta.SwathWidth/1e3)

		fmt.Println()
		fmt.Println("  1.3.2 Push-Broom Sensor (single line of detectors):")
		fmt.Printf("    Detector Size     = %.2f μm  (= f*pixel/H)\n", ta.DetectorSize*1e6)
		fmt.Printf("    Number of Detectors = %d  (= D_fp / detector_size)\n", ta.NumDetectors)
		fmt.Printf("    Dwell Time (smear<1/3 pix) = %.4e s = %.4f ms  (= pixel/(3*Vg))\n",
			ta.PushBroomDwellTime, ta.PushBroomDwellTime*1e3)

		fmt.Println()
		fmt.Println("  1.3.3 Whiskbroom Sensor:")
		fmt.Printf("    Scan Time (one line)  = %.4e s = %.4f ms  (= pixel/Vg)\n",
			ta.WhiskScanTime, ta.WhiskScanTime*1e3)
		fmt.Printf("    Dwell Time per pixel  = %.4e s = %.6f ms  (= scanTime/N_cross)\n",
			ta.WhiskDwellTime, ta.WhiskDwellTime*1e3)
		fmt.Printf("    Mirror Scan Rate      = %.2f rad/s = %.2f deg/s  (= FOV/scanTime)\n",
			ta.WhiskMirrorScanRate, ta.WhiskMirrorScanRate*180.0/math.Pi)

		fmt.Println()
		fmt.Println("  1.3.4 Multi-Element Whiskbroom (8 along-track elements):")
		fmt.Printf("    Scan Time (8 lines)   = %.4e s = %.4f ms  (= 8*pixel/Vg)\n",
			ta.MultiElemScanTime, ta.MultiElemScanTime*1e3)
		fmt.Printf("    Dwell Time per pixel  = %.4e s = %.6f ms  (= scanTime/N_cross)\n",
			ta.MultiElemDwellTime, ta.MultiElemDwellTime*1e3)
		fmt.Printf("    Mirror Scan Rate      = %.2f rad/s = %.2f deg/s  (= FOV/scanTime)\n",
			ta.MultiElemMirrorScanRate, ta.MultiElemMirrorScanRate*180.0/math.Pi)

		fmt.Println()
		fmt.Println()
	}
}
