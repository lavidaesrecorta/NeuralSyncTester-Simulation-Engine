package tpm_handlers

type TPMStimulationHandlers interface {
	CreateStimulationStructure(k []int, n_0 int) []int
	CreateStimulusFromLayerOutput(outputs []int, k_h int, n_h int) [][]int
}
