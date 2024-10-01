package tpm_sync

type HebbianLearnRule struct{}
type AntiHebbianLearnRule struct{}
type RandomWalkLearnRule struct{}

func (learnRule HebbianLearnRule) TPMLearnLayer(k int, n int, l int, weights [][]int, stimulus [][]int, outputs []int, output_a int, output_b int) {
	for i := 0; i < k; i++ {
		for j := 0; j < n; j++ {
			newWeight := weights[i][j] + stimulus[i][j]*output_a*heavisideStep(outputs[i]*output_a)*heavisideStep(output_a*output_b)
			weights[i][j] = gFunction(newWeight, l)
		}
	}
}

func (learnRule AntiHebbianLearnRule) TPMLearnLayer(k int, n int, l int, weights [][]int, stimulus [][]int, outputs []int, output_a int, output_b int) {
	for i := 0; i < k; i++ {
		for j := 0; j < n; j++ {
			newWeight := weights[i][j] - stimulus[i][j]*output_a*heavisideStep(outputs[i]*output_a)*heavisideStep(output_a*output_b)
			weights[i][j] = gFunction(newWeight, l)
		}
	}
}

func (learnRule RandomWalkLearnRule) TPMLearnLayer(k int, n int, l int, weights [][]int, stimulus [][]int, outputs []int, output_a int, output_b int) {
	for i := 0; i < k; i++ {
		for j := 0; j < n; j++ {
			newWeight := weights[i][j] + stimulus[i][j]*heavisideStep(outputs[i]*output_a)*heavisideStep(output_a*output_b)
			weights[i][j] = gFunction(newWeight, l)
		}
	}
}
