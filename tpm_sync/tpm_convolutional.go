package tpm_sync

func CreateConvolutionalStimulusStructure(k []int, n_0 int) []int {
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

func createOverlappedStimulusFromOutput(output []int, k int, n int) [][]int {
	new_stimulus := make([][]int, k)
	for i := 0; i < k; i++ {
		new_stimulus[i] = make([]int, n)
		for j := 0; j < n; j++ {
			new_stimulus[i][j] = output[j+i]
		}
	}

	return new_stimulus
}
