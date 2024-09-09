package tpm_sync

type TPMmSessionState struct {
	stimulus         [][]int
	layer_stimulus_a [][][]int
	layer_stimulus_b [][][]int
	Weights_A        [][][]int
	Weights_B        [][][]int
	Outputs_A        [][]int
	Outputs_B        [][]int
}

type TPMmSettings struct {
	K                   []int
	N                   []int
	L                   int
	M                   int
	stimulationHandlers TPMStimulationHandlers
	learnRuleHandler    TPMLearnRuleHandler
}

type TPMStimulationHandlers interface {
	CreateStimulationStructure(k []int, n_0 int) []int
	CreateStimulusFromLayerOutput(outputs []int, k_h int, n_h int) [][]int
}

type TPMLearnRuleHandler interface {
	TPMLearnLayer(k int, n int, l int, weights [][]int, stimulus [][]int, outputs []int, output_a int, output_b int)
}
