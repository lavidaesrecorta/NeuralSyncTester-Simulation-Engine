package tpm_stimHandlers

type NoOverlapTPM struct{}

func (tpm NoOverlapTPM) CreateStimulationStructure(n []int, k_last int) []int {
	h := len(n)
	k := make([]int, h)
	k[h-1] = k_last
	for i := 1; i < h; i++ {
		k[h-1-i] = n[h-i] * k[h-i]
	}
	return k
}

func (tpm NoOverlapTPM) CreateStimulusFromLayerOutput(outputs []int, k_h int, n_h int) [][]int {
	new_stimulus := make([][]int, k_h)
	for i := 0; i < k_h; i++ {
		new_stimulus[i] = make([]int, n_h)
		for j := 0; j < n_h; j++ {
			new_stimulus[i][j] = outputs[n_h*i+j]
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
