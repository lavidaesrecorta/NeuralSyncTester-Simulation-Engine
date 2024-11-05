package tpm_handlers

type FullConnectionTPM struct{}

func (tpm FullConnectionTPM) CreateStimulationStructure(k []int, n_0 int) []int {
	h := len(k)
	n := make([]int, h)

	n[0] = n_0

	for layer := 1; layer < h; layer++ {
		n[layer] = k[layer-1]
	}
	return n
}

func (tpm FullConnectionTPM) CreateStimulusFromLayerOutput(outputs []int, k_h int, n_h int) [][]int {
	new_stimulus := make([][]int, k_h)
	for i := 0; i < k_h; i++ {
		new_stimulus[i] = make([]int, n_h)
		//When fully connected, the stim count is the same as the neuron count from the prev layer
		for j := 0; j < n_h; j++ {
			new_stimulus[i][j] = outputs[j] //So this maps outputs to inputs, 1 to 1
		}
	}

	return new_stimulus
}
