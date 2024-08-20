package tpm_sync

import (
	"fmt"
	"math"
	"math/rand"
)

// import "fmt"

type Iteration struct {
	Stimulus      [][]int
	LayersOutput  [][]int
	Weights       [][][]int
	NetworkOutput int
}

func SyncSession(K []int, N []int, L int, M int, TpmType string, LearnRule string) {
	h := len(K)
	weights_a := make([][][]int, h)
	weights_b := make([][][]int, h)
	for layer := 0; layer < h; layer++ {
		weights_a[layer] = createRandomLayerWeightsArray(K[layer], N[layer], L)
		weights_b[layer] = createRandomLayerWeightsArray(K[layer], N[layer], L)
	}
	stim := createRandomStimulusArray(K[0], N[0], M)

	total_iterations := 0
	learn_iterations := 0

	for !CompareWeights(h, K, N, weights_a, weights_b) {

		//setup input/output
		stim_a := stim
		stim_b := stim
		outputs_a := make([][]int, h)
		outputs_b := make([][]int, h)

		//Stimulate layers
		for layer := 0; layer < h; layer++ {
			outputs_a[layer] = stimulateLayer(stim_a, weights_a[layer], K[layer], N[layer])
			outputs_b[layer] = stimulateLayer(stim_b, weights_b[layer], K[layer], N[layer])
			if layer+1 < h {
				stim_a = createOverlappedStimulusFromOutput(outputs_a[layer], K[layer+1], N[layer+1]) //Stimulus are created for each neuron of the NEXT layer
				stim_b = createOverlappedStimulusFromOutput(outputs_b[layer], K[layer+1], N[layer+1])
			}
		}
		final_output_a := thau(outputs_a[h-1], K[h-1])
		final_output_b := thau(outputs_b[h-1], K[h-1])
		total_iterations += 1

		//Check if we need to learn in this iteration
		if final_output_a == final_output_b {
			stim_a = stim
			stim_b = stim
			for layer := 0; layer < h; layer++ {
				learn_layer(K[layer], N[layer], L, weights_a[layer], stim_a, outputs_a[layer], final_output_a, final_output_b)
				learn_layer(K[layer], N[layer], L, weights_b[layer], stim_b, outputs_b[layer], final_output_b, final_output_a)

				if layer+1 < h {
					stim_a = createOverlappedStimulusFromOutput(outputs_a[layer], K[layer+1], N[layer+1]) //Stimulus are created for each neuron of the NEXT layer
					stim_b = createOverlappedStimulusFromOutput(outputs_b[layer], K[layer+1], N[layer+1])
				}

			}
			learn_iterations += 1
		}
		stim = createRandomStimulusArray(K[0], N[0], M)
	}

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

	fmt.Println("Synchronization finished.")
	return true
}

func FastInverseSqrt(x float64) float64 {
	i := math.Float64bits(x)
	i = 0x5fe6eb50c7b537a9 - (i >> 1)
	y := math.Float64frombits(i)

	// One iteration of Newton's method to improve accuracy
	y = y * (1.5 - 0.5*x*y*y)

	return y
}

func neuron_localField(n int, w_k []int, stim_k []int) float64 {
	dot_prod := 0
	for i := 0; i < n; i++ {
		dot_prod += w_k[i] * stim_k[i]
	}
	return float64(dot_prod) * (float64(n))
}

func outputSigma(x float64) int {
	if x > 0 {
		return 1
	}
	return -1
}

func thau(outputs []int, k int) int {
	mul := 1
	for i := 0; i < k; i++ {
		mul *= outputs[i]
	}

	return mul
}

func heavisideStep(x int) int {
	if x > 0 {
		return 1
	}
	return 0
}
func gFunction(w int, l int) int {
	sign := 1
	if w < 0 {
		sign = -1
	}
	if w*sign > l {
		return l * sign
	}
	return w
}

// stimulateLayer
func stimulateLayer(stimu [][]int, weights [][]int, k int, n int) []int {

	layerOutputs := make([]int, k)
	for i := 0; i < k; i++ {
		localField := neuron_localField(n, weights[i], stimu[i])
		localOutput := outputSigma(localField)
		layerOutputs[i] = localOutput
	}

	return layerOutputs

}

// learn_layer:
func learn_layer(k int, n int, l int, weights [][]int, stimulus [][]int, outputs []int, output_a int, output_b int) {
	for i := 0; i < k; i++ {
		for j := 0; j < n; j++ {
			newWeight := weights[i][j] + stimulus[i][j]*output_a*heavisideStep(outputs[i]*output_a)*heavisideStep(output_a*output_b)
			weights[i][j] = gFunction(newWeight, l)

		}
	}

}

func createRandomStimulusArray(k int, n int, m int) [][]int {
	stim := make([][]int, k)
	for i := 0; i < k; i++ {
		stim[i] = make([]int, n)
		for j := 0; j < n; j++ {
			stim[i][j] = (rand.Intn(2)*2 - 1) * (rand.Intn(m) + 1)
		}
	}
	return stim
}

func createRandomLayerWeightsArray(k int, n int, l int) [][]int {
	w := make([][]int, k)
	for i := 0; i < k; i++ {
		w[i] = make([]int, n)
		for j := 0; j < n; j++ {
			w[i][j] = (rand.Intn(2)*2 - 1) * (rand.Intn(l + 1)) // l + 1 because the function goes from [0,l[
		}
	}
	return w
}
