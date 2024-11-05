package tpm_controllers

import (
	"fmt"
	"math/rand"
	"strings"
	"tpm_sync/tpm_core"
	"tpm_sync/tpm_handlers"
	"tpm_sync/tpm_learnRules"
)

// import "fmt"
type SyncController struct {
}

func (SyncController) SettingsFactory(K []int, n_0 int, l int, m int, tpmType string, learnRule string) (TPMmSettings, error) {

	var stimHandler tpm_handlers.TPMStimulationHandlers
	var ruleHandler tpm_learnRules.TPMLearnRuleHandler

	reverseParameters := false // this is because the no overlap os defined by the stimulus, so K[] is actually N[] and n_0 is actually k_last

	K = copySlice(K)

	switch parsed_tpmType := strings.ToUpper(tpmType); parsed_tpmType {
	case "PARTIALLY_CONNECTED":
		stimHandler = tpm_handlers.PartialConnectionTPM{}
	case "FULLY_CONNECTED":
		stimHandler = tpm_handlers.FullConnectionTPM{}
	case "NO_OVERLAP":
		stimHandler = tpm_handlers.NoOverlapTPM{}
		reverseParameters = true
	}
	if stimHandler == nil {
		return TPMmSettings{}, fmt.Errorf("TPM type is invalid: %s", tpmType)
	}

	switch parsed_learnRule := strings.ToUpper(learnRule); parsed_learnRule {
	case "HEBBIAN":
		ruleHandler = tpm_learnRules.HebbianLearnRule{}
	case "ANTI-HEBBIAN":
		ruleHandler = tpm_learnRules.AntiHebbianLearnRule{}
	case "RANDOM-WALK":
		ruleHandler = tpm_learnRules.RandomWalkLearnRule{}
	}
	if ruleHandler == nil {
		return TPMmSettings{}, fmt.Errorf("TPM rule is invalid: %s", learnRule)
	}

	N := stimHandler.CreateStimulationStructure(K, n_0)
	if reverseParameters {
		aux := N
		N = K
		K = aux
	}

	return TPMmSettings{
		K:                   K,
		N:                   N,
		L:                   l,
		M:                   m,
		H:                   len(K),
		LearnRule:           learnRule,
		LinkType:            tpmType,
		learnRuleHandler:    ruleHandler,
		stimulationHandlers: stimHandler,
	}, nil
}

func (s SyncController) CreateSessionInstance(tpmSettings TPMmSettings, localRand *rand.Rand) TPMmSessionState {
	weights_a := make([][][]int, tpmSettings.H)
	weights_b := make([][][]int, tpmSettings.H)
	for layer := 0; layer < tpmSettings.H; layer++ {
		weights_a[layer] = tpm_core.CreateRandomLayerWeightsArray(tpmSettings.K[layer], tpmSettings.N[layer], tpmSettings.L, localRand)
		weights_b[layer] = tpm_core.CreateRandomLayerWeightsArray(tpmSettings.K[layer], tpmSettings.N[layer], tpmSettings.L, localRand)
	}
	stim := tpm_core.CreateRandomStimulusArray(tpmSettings.K[0], tpmSettings.N[0], tpmSettings.M, localRand)
	layer_stim_a := make([][][]int, tpmSettings.H)
	layer_stim_b := make([][][]int, tpmSettings.H)
	outputs_a := make([][]int, tpmSettings.H)
	outputs_b := make([][]int, tpmSettings.H)

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

func (s SyncController) StartSyncSession(tpmSettings TPMmSettings, maxIterations int, seed int64, localRand *rand.Rand) SessionData {

	//Setup simulation
	sessionState := s.CreateSessionInstance(tpmSettings, localRand)
	initialState := sessionState
	//Start simulation
	total_iterations := 0
	learn_iterations := 0
	for !tpm_core.CompareWeights(tpmSettings.H, tpmSettings.K, tpmSettings.N, sessionState.Weights_A, sessionState.Weights_B) {

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
		for layer := 0; layer < tpmSettings.H-1; layer++ {
			sessionState.Outputs_A[layer] = tpm_core.StimulateLayer(sessionState.layer_stimulus_a[layer], sessionState.Weights_A[layer], tpmSettings.K[layer], tpmSettings.N[layer])
			sessionState.Outputs_B[layer] = tpm_core.StimulateLayer(sessionState.layer_stimulus_b[layer], sessionState.Weights_B[layer], tpmSettings.K[layer], tpmSettings.N[layer])
			sessionState.layer_stimulus_a[layer+1] = tpmSettings.stimulationHandlers.CreateStimulusFromLayerOutput(sessionState.Outputs_A[layer], tpmSettings.K[layer+1], tpmSettings.N[layer+1])
			sessionState.layer_stimulus_b[layer+1] = tpmSettings.stimulationHandlers.CreateStimulusFromLayerOutput(sessionState.Outputs_B[layer], tpmSettings.K[layer+1], tpmSettings.N[layer+1])
		}
		sessionState.Outputs_A[tpmSettings.H-1] = tpm_core.StimulateLayer(sessionState.layer_stimulus_a[tpmSettings.H-1], sessionState.Weights_A[tpmSettings.H-1], tpmSettings.K[tpmSettings.H-1], tpmSettings.N[tpmSettings.H-1])
		sessionState.Outputs_B[tpmSettings.H-1] = tpm_core.StimulateLayer(sessionState.layer_stimulus_b[tpmSettings.H-1], sessionState.Weights_B[tpmSettings.H-1], tpmSettings.K[tpmSettings.H-1], tpmSettings.N[tpmSettings.H-1])
		final_output_a := tpm_core.Thau(sessionState.Outputs_A[tpmSettings.H-1], tpmSettings.K[tpmSettings.H-1])
		final_output_b := tpm_core.Thau(sessionState.Outputs_B[tpmSettings.H-1], tpmSettings.K[tpmSettings.H-1])
		total_iterations += 1

		//Check if we need to learn in this iteration
		if final_output_a == final_output_b {
			for layer := 0; layer < tpmSettings.H; layer++ {
				tpmSettings.learnRuleHandler.TPMLearnLayer(tpmSettings.K[layer], tpmSettings.N[layer], tpmSettings.L, sessionState.Weights_A[layer], sessionState.layer_stimulus_a[layer], sessionState.Outputs_A[layer], final_output_a, final_output_b)
				tpmSettings.learnRuleHandler.TPMLearnLayer(tpmSettings.K[layer], tpmSettings.N[layer], tpmSettings.L, sessionState.Weights_B[layer], sessionState.layer_stimulus_b[layer], sessionState.Outputs_B[layer], final_output_b, final_output_a)
			}
			learn_iterations += 1
		}
		sessionState.Stimulus = tpm_core.CreateRandomStimulusArray(tpmSettings.K[0], tpmSettings.N[0], tpmSettings.M, localRand)
	}
	return SessionData{
		Seed:                seed,
		StimulateIterations: total_iterations,
		LearnIterations:     learn_iterations,
		InitialState:        initialState,
		FinalState:          sessionState,
		Status:              "FINISHED",
	}
}

func (SyncController) GetDataSizeFromConfig(config TPMmSettings) int {
	//Count the amount of weights
	//So, count each stimulus, for every neuron, for every layer
	totalDataSize := 0
	for layer := 0; layer < config.H; layer++ {
		totalDataSize += config.K[layer] * config.N[layer]
	}
	return totalDataSize
}

func copySlice(input []int) []int {
	copied := make([]int, len(input))
	copy(copied, input)
	return copied
}
