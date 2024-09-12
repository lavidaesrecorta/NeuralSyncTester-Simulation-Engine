package tpm_sync

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
)

// import "fmt"

type Iteration struct {
	Stimulus      [][]int
	LayersOutput  [][]int
	Weights       [][][]int
	NetworkOutput int
}

type SessionData struct {
	Seed                int64
	StimulateIterations int
	LearnIterations     int
	InitialState        TPMmSessionState
	FinalState          TPMmSessionState
	Status              string
}

func SettingsFactory(K []int, n_0 int, l int, m int, tpmType string, learnRule string) (TPMmSettings, error) {

	var stimHandler TPMStimulationHandlers
	var ruleHandler TPMLearnRuleHandler

	switch parsed_tpmType := strings.ToUpper(tpmType); parsed_tpmType {
	case "PARTIALLY_CONNECTED":
		stimHandler = PartialConnectionTPM{}
	}
	if stimHandler == nil && ruleHandler == nil {
		return TPMmSettings{}, fmt.Errorf("TPM type is invalid: %s", tpmType)
	}

	switch parsed_learnRule := strings.ToUpper(learnRule); parsed_learnRule {
	case "HEBBIAN":
		ruleHandler = HebbianLearnRule{}
	}
	if ruleHandler == nil {
		return TPMmSettings{}, fmt.Errorf("TPM rule is invalid: %s", learnRule)
	}

	N := stimHandler.CreateStimulationStructure(K, n_0)

	return TPMmSettings{
		K:                   K,
		N:                   N,
		L:                   l,
		M:                   m,
		LearnRule:           learnRule,
		LinkType:            tpmType,
		learnRuleHandler:    ruleHandler,
		stimulationHandlers: stimHandler,
	}, nil
}

func InitializeSession(tpmSettings TPMmSettings) TPMmSessionState {
	h := len(tpmSettings.K)
	weights_a := make([][][]int, h)
	weights_b := make([][][]int, h)
	for layer := 0; layer < h; layer++ {
		weights_a[layer] = createRandomLayerWeightsArray(tpmSettings.K[layer], tpmSettings.N[layer], tpmSettings.L)
		weights_b[layer] = createRandomLayerWeightsArray(tpmSettings.K[layer], tpmSettings.N[layer], tpmSettings.L)
	}
	stim := createRandomStimulusArray(tpmSettings.K[0], tpmSettings.N[0], tpmSettings.M)
	layer_stim_a := make([][][]int, h)
	layer_stim_b := make([][][]int, h)
	outputs_a := make([][]int, h)
	outputs_b := make([][]int, h)

	return TPMmSessionState{
		Stimulus:         stim,
		layer_stimulus_a: layer_stim_a,
		layer_stimulus_b: layer_stim_b,
		Weights_A:        weights_a,
		Weights_B:        weights_b,
		Outputs_A:        outputs_a,
		Outputs_B:        outputs_b,
	}
}

func SyncSession(tpmSettings TPMmSettings, maxIterations int, seed int64) SessionData {

	//setup simulation
	h := len(tpmSettings.K)
	sessionState := InitializeSession(tpmSettings)
	initialState := sessionState
	fmt.Println("A_0:", sessionState.Weights_A)
	fmt.Println("B_0:", sessionState.Weights_B)
	fmt.Println("----------------------------")
	//Start simulation
	total_iterations := 0
	learn_iterations := 0
	for !CompareWeights(h, tpmSettings.K, tpmSettings.N, sessionState.Weights_A, sessionState.Weights_B) {

		//Health Check: has the simulation has been running for too long?
		if total_iterations > maxIterations && maxIterations != 0 {
			return SessionData{
				Seed:                seed,
				StimulateIterations: total_iterations,
				LearnIterations:     learn_iterations,
				InitialState:        initialState,
				FinalState:          sessionState,
				Status:              "LIMIT_REACHED",
			}
		}

		//Setup first layer, next layers will be calculated on the stimulation process
		sessionState.layer_stimulus_a[0] = sessionState.Stimulus
		sessionState.layer_stimulus_b[0] = sessionState.Stimulus

		//Stimulate layers, stimulate the last layer separate from the rest to avoid creating unnecesary stimulus arrays
		for layer := 0; layer < h-1; layer++ {
			sessionState.Outputs_A[layer] = stimulateLayer(sessionState.layer_stimulus_a[layer], sessionState.Weights_A[layer], tpmSettings.K[layer], tpmSettings.N[layer])
			sessionState.Outputs_B[layer] = stimulateLayer(sessionState.layer_stimulus_b[layer], sessionState.Weights_B[layer], tpmSettings.K[layer], tpmSettings.N[layer])
			sessionState.layer_stimulus_a[layer+1] = tpmSettings.stimulationHandlers.CreateStimulusFromLayerOutput(sessionState.Outputs_A[layer], tpmSettings.K[layer+1], tpmSettings.N[layer+1])
			sessionState.layer_stimulus_b[layer+1] = tpmSettings.stimulationHandlers.CreateStimulusFromLayerOutput(sessionState.Outputs_B[layer], tpmSettings.K[layer+1], tpmSettings.N[layer+1])
		}
		sessionState.Outputs_A[h-1] = stimulateLayer(sessionState.layer_stimulus_a[h-1], sessionState.Weights_A[h-1], tpmSettings.K[h-1], tpmSettings.N[h-1])
		sessionState.Outputs_B[h-1] = stimulateLayer(sessionState.layer_stimulus_b[h-1], sessionState.Weights_B[h-1], tpmSettings.K[h-1], tpmSettings.N[h-1])
		final_output_a := thau(sessionState.Outputs_A[h-1], tpmSettings.K[h-1])
		final_output_b := thau(sessionState.Outputs_B[h-1], tpmSettings.K[h-1])
		total_iterations += 1

		//Check if we need to learn in this iteration
		if final_output_a == final_output_b {
			for layer := 0; layer < h; layer++ {
				tpmSettings.learnRuleHandler.TPMLearnLayer(tpmSettings.K[layer], tpmSettings.N[layer], tpmSettings.L, sessionState.Weights_A[layer], sessionState.layer_stimulus_a[layer], sessionState.Outputs_A[layer], final_output_a, final_output_b)
				tpmSettings.learnRuleHandler.TPMLearnLayer(tpmSettings.K[layer], tpmSettings.N[layer], tpmSettings.L, sessionState.Weights_B[layer], sessionState.layer_stimulus_b[layer], sessionState.Outputs_B[layer], final_output_b, final_output_a)
			}
			learn_iterations += 1
		}
		sessionState.Stimulus = createRandomStimulusArray(tpmSettings.K[0], tpmSettings.N[0], tpmSettings.M)
	}

	fmt.Println("A:", sessionState.Weights_A)
	fmt.Println("B:", sessionState.Weights_B)
	fmt.Println("++++++++++++++++++++++++++++")
	return SessionData{
		Seed:                seed,
		StimulateIterations: total_iterations,
		InitialState:        initialState,
		FinalState:          sessionState,
		Status:              "FINISHED",
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
