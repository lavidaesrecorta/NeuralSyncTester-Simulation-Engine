package tpm_sync

type NoOverlapTPM struct{}

const branches_per_node = 2 //each neuron has two branches

func (tpm NoOverlapTPM) CreateStimulationStructure(k []int, n_0 int) []int {
	h := len(k)
	n := make([]int, h)

	n[0] = n_0
	if k[h-1] < 2 {
		return nil
	}
	for layer := 1; layer < h; layer++ {
		if k[h-1-layer] != k[h-1]*IntPow(branches_per_node, layer) {
			return nil
		}
		n[layer] = branches_per_node
	}
	return n
}

func (tpm NoOverlapTPM) CreateStimulusFromLayerOutput(outputs []int, k_h int, n_h int) [][]int {
	new_stimulus := make([][]int, k_h)
	for i := 0; i < k_h; i++ {
		new_stimulus[i] = make([]int, n_h)
		for j := 0; j < n_h; j++ {
			new_stimulus[i][j] = outputs[branches_per_node*i+j]
		}
	}

	return new_stimulus
}

func IntPow(base, exp int) int {
	result := 1
	for {
		if exp&1 == 1 {
			result *= base
		}
		exp >>= 1
		if exp == 0 {
			break
		}
		base *= base
	}

	return result
}
