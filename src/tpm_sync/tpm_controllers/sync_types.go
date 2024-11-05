package tpm_controllers

import (
	"tpm_sync/tpm_handlers"
	"tpm_sync/tpm_learnRules"
)

type TPMmSessionState struct {
	Stimulus         [][]int
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
	H                   int
	LearnRule           string
	LinkType            string
	stimulationHandlers tpm_handlers.TPMStimulationHandlers
	learnRuleHandler    tpm_learnRules.TPMLearnRuleHandler
}

type SessionData struct {
	Seed                int64
	StimulateIterations int
	LearnIterations     int
	InitialState        TPMmSessionState
	FinalState          TPMmSessionState
	Status              string
}
