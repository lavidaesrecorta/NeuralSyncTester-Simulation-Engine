package tpm_core

import (
	"math"
	"math/rand"
)

func StimulateLayer(stimu [][]int, weights [][]int, k int, n int) []int {

	layerOutputs := make([]int, k)
	for i := 0; i < k; i++ {
		localField := NeuronLocalField(n, weights[i], stimu[i])
		localOutput := OutputSigma(localField)
		layerOutputs[i] = localOutput
	}

	return layerOutputs
}

func NeuronLocalField(n int, w_k []int, stim_k []int) float64 {
	dot_prod := 0
	for i := 0; i < n; i++ {
		dot_prod += w_k[i] * stim_k[i]
	}
	return float64(dot_prod) * (float64(n))
}

func OutputSigma(x float64) int {
	if x > 0 {
		return 1
	}
	return -1
}

func Thau(outputs []int, k int) int {
	mul := 1
	for i := 0; i < k; i++ {
		mul *= outputs[i]
	}

	return mul
}

func HeavisideStep(x int) int {
	if x > 0 {
		return 1
	}
	return 0
}

func GFunction(w int, l int) int {
	sign := 1
	if w < 0 {
		sign = -1
	}
	if w*sign > l {
		return l * sign
	}
	return w
}

func FastInverseSqrt(x float64) float64 {
	i := math.Float64bits(x)
	i = 0x5fe6eb50c7b537a9 - (i >> 1)
	y := math.Float64frombits(i)

	y = y * (1.5 - 0.5*x*y*y)
	return y
}

func CompareWeights(h int, k []int, n []int, weights_a [][][]int, weights_b [][][]int) bool {
	for layer := 0; layer < h; layer++ {
		for i := 0; i < k[layer]; i++ {
			for j := 0; j < n[layer]; j++ {
				if weights_a[layer][i][j] != weights_b[layer][i][j] {
					return false
				}
			}
		}
	}

	return true
}

func CreateRandomStimulusArray(k int, n int, m int, localRand *rand.Rand) [][]int {
	stim := make([][]int, k)
	for i := 0; i < k; i++ {
		stim[i] = make([]int, n)
		for j := 0; j < n; j++ {
			stim[i][j] = (localRand.Intn(2)*2 - 1) * (localRand.Intn(m) + 1)
		}
	}
	return stim
}

func CreateRandomLayerWeightsArray(k int, n int, l int, localRand *rand.Rand) [][]int {
	w := make([][]int, k)
	for i := 0; i < k; i++ {
		w[i] = make([]int, n)
		for j := 0; j < n; j++ {
			w[i][j] = (localRand.Intn(2)*2 - 1) * (localRand.Intn(l + 1)) // l + 1 because the function goes from [0,l[
		}
	}
	return w
}

func GetNetworkDataSize(H int, K []int, N []int) int {
	totalDataSize := 0
	for layer := 0; layer < H; layer++ {
		totalDataSize += K[layer] * N[layer]
	}
	return totalDataSize
}
