package tpm_learnRules

import "tpm_sync/tpm_core"

type TPMLearnRuleHandler interface {
	TPMLearnLayer(k int, n int, l int, weights [][]int, stimulus [][]int, outputs []int, output_a int, output_b int)
}

type HebbianLearnRule struct{}
type AntiHebbianLearnRule struct{}
type RandomWalkLearnRule struct{}

func (learnRule HebbianLearnRule) TPMLearnLayer(k int, n int, l int, weights [][]int, stimulus [][]int, outputs []int, output_a int, output_b int) {
	for i := 0; i < k; i++ {
		for j := 0; j < n; j++ {
			newWeight := weights[i][j] + stimulus[i][j]*output_a*tpm_core.HeavisideStep(outputs[i]*output_a)*tpm_core.HeavisideStep(output_a*output_b)
			weights[i][j] = tpm_core.GFunction(newWeight, l)
		}
	}
}

func (learnRule AntiHebbianLearnRule) TPMLearnLayer(k int, n int, l int, weights [][]int, stimulus [][]int, outputs []int, output_a int, output_b int) {
	for i := 0; i < k; i++ {
		for j := 0; j < n; j++ {
			newWeight := weights[i][j] - stimulus[i][j]*output_a*tpm_core.HeavisideStep(outputs[i]*output_a)*tpm_core.HeavisideStep(output_a*output_b)
			weights[i][j] = tpm_core.GFunction(newWeight, l)
		}
	}
}

func (learnRule RandomWalkLearnRule) TPMLearnLayer(k int, n int, l int, weights [][]int, stimulus [][]int, outputs []int, output_a int, output_b int) {
	for i := 0; i < k; i++ {
		for j := 0; j < n; j++ {
			newWeight := weights[i][j] + stimulus[i][j]*tpm_core.HeavisideStep(outputs[i]*output_a)*tpm_core.HeavisideStep(output_a*output_b)
			weights[i][j] = tpm_core.GFunction(newWeight, l)
		}
	}
}
