// Package fingerprint implements the deterministic, infrastructure-free
// landmark-v1 acoustic fingerprint extractor.
package fingerprint

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

const (
	AlgorithmLandmarkV1 = int16(1)
	SampleRate          = 11025
	FFTSize             = 512
	HopSize             = 256
	MinDTFrames         = 2
	MaxDTFrames         = 63
	MaxAbsDFBins        = 31
	Fanout              = 3
	maxPeaksPerFrame    = 5
)

var ErrInsufficientSignal = errors.New("insufficient audio signal")

// Fingerprint is the compact inverted-index payload for a landmark pair.
type Fingerprint struct {
	Hash        uint32
	AnchorFrame int
}

type peak struct {
	frame, bin int
	magnitude  float64
}

// Extract produces a stable landmark-v1 output from canonical mono PCM.
func Extract(samples []int16) ([]Fingerprint, error) {
	if len(samples) < FFTSize {
		return nil, ErrInsufficientSignal
	}
	spectrogram := stft(samples)
	peaks := detectPeaks(spectrogram)
	if len(peaks) < 2 {
		return nil, ErrInsufficientSignal
	}
	fps := make([]Fingerprint, 0, len(peaks)*2)
	for i, anchor := range peaks {
		paired := 0
		for j := i + 1; j < len(peaks) && paired < Fanout; j++ {
			target := peaks[j]
			dt := target.frame - anchor.frame
			if dt < MinDTFrames {
				continue
			}
			if dt > MaxDTFrames {
				break
			}
			df := target.bin - anchor.bin
			if abs(df) > MaxAbsDFBins {
				continue
			}
			h, err := EncodeHash(anchor.bin, df, dt)
			if err != nil {
				return nil, fmt.Errorf("encode landmark: %w", err)
			}
			fps = append(fps, Fingerprint{Hash: h, AnchorFrame: anchor.frame})
			paired++
		}
	}
	if len(fps) == 0 {
		return nil, ErrInsufficientSignal
	}
	return fps, nil
}

func stft(samples []int16) [][]float64 {
	frames := 1 + (len(samples)-FFTSize)/HopSize
	result := make([][]float64, frames)
	input := make([]float64, FFTSize)
	for frame := 0; frame < frames; frame++ {
		base := frame * HopSize
		for i := range input {
			// Hann is frozen as a part of landmark-v1.
			window := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(FFTSize-1)))
			input[i] = float64(samples[base+i]) / 32768 * window
		}
		coeffs := fft(input)
		bins := make([]float64, FFTSize/2)
		for bin := range bins {
			bins[bin] = math.Log1p(math.Hypot(real(coeffs[bin]), imag(coeffs[bin])))
		}
		result[frame] = bins
	}
	return result
}

// fft is a radix-2 Cooley-Tukey FFT. Keeping it local makes the phase-0
// spike runnable with the repository's installed Go toolchain; phase 3 will
// replace this tested implementation with Gonum Fourier once dependencies are
// resolved in CI/deployment.
func fft(input []float64) []complex128 {
	n := len(input)
	out := make([]complex128, n)
	for i, v := range input {
		out[i] = complex(v, 0)
	}
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			out[i], out[j] = out[j], out[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		angle := -2 * math.Pi / float64(size)
		wlen := complex(math.Cos(angle), math.Sin(angle))
		for start := 0; start < n; start += size {
			w := complex(1, 0)
			for j := 0; j < size/2; j++ {
				u, v := out[start+j], out[start+j+size/2]*w
				out[start+j], out[start+j+size/2] = u+v, u-v
				w *= wlen
			}
		}
	}
	return out
}

func detectPeaks(spec [][]float64) []peak {
	all := make([]peak, 0)
	for t, bins := range spec {
		candidates := make([]peak, 0)
		mean := 0.0
		for _, v := range bins {
			mean += v
		}
		mean /= float64(len(bins))
		for f := 2; f < len(bins)-2; f++ {
			v := bins[f]
			if v < mean+0.12 || v < bins[f-1] || v < bins[f+1] || v < bins[f-2] || v < bins[f+2] {
				continue
			}
			localMax := true
			for tt := max(0, t-2); tt <= min(len(spec)-1, t+2); tt++ {
				if tt != t && spec[tt][f] > v {
					localMax = false
					break
				}
			}
			if localMax {
				candidates = append(candidates, peak{t, f, v})
			}
		}
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].magnitude == candidates[j].magnitude {
				return candidates[i].bin < candidates[j].bin
			}
			return candidates[i].magnitude > candidates[j].magnitude
		})
		if len(candidates) > maxPeaksPerFrame {
			candidates = candidates[:maxPeaksPerFrame]
		}
		all = append(all, candidates...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].frame == all[j].frame {
			return all[i].bin < all[j].bin
		}
		return all[i].frame < all[j].frame
	})
	return all
}

// EncodeHash packs f1, signed df, and dt into the frozen 20-bit v1 layout.
func EncodeHash(f1, df, dt int) (uint32, error) {
	if f1 < 0 || f1 > 255 || df < -31 || df > 31 || dt < MinDTFrames || dt > MaxDTFrames {
		return 0, fmt.Errorf("landmark out of range f1=%d df=%d dt=%d", f1, df, dt)
	}
	return uint32(f1)<<12 | uint32(df&0x3f)<<6 | uint32(dt), nil
}

// DecodeHash exists to make the wire/storage layout exhaustively testable.
func DecodeHash(hash uint32) (f1, df, dt int) {
	f1, df, dt = int((hash>>12)&0xff), int((hash>>6)&0x3f), int(hash&0x3f)
	if df&0x20 != 0 {
		df -= 0x40
	}
	return
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
