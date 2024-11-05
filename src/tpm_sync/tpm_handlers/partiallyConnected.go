package tpm_handlers

type PartialConnectionTPM struct{}

func (tpm PartialConnectionTPM) CreateStimulationStructure(k []int, n_0 int) []int {
	prev := -1
	h := len(k)
	n := make([]int, h)

	n[0] = n_0
	prev = k[0]

	for layer := 1; layer < h; layer++ {

		if k[layer] >= prev {
			return nil
		}
		n[layer] = k[layer-1] - k[layer] + 1
		prev = k[layer]
	}
	return n
}

func (tpm PartialConnectionTPM) CreateStimulusFromLayerOutput(outputs []int, k_h int, n_h int) [][]int {
	new_stimulus := make([][]int, k_h)
	for i := 0; i < k_h; i++ {
		new_stimulus[i] = make([]int, n_h)
		for j := 0; j < n_h; j++ {
			new_stimulus[i][j] = outputs[j+i]
		}
	}

	return new_stimulus
}
