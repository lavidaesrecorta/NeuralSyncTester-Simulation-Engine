package tpm_controllers

import (
	"sync"
	"time"
)

type OpenSession struct {
	Uid                 string
	Config              TPMmSettings
	StartTime           time.Time
	MaxSessionCount     int
	CurrentSessionCount int
}

type SessionMap struct {
	Sessions map[string]*OpenSession
	Mutex    sync.RWMutex
}

type SimulationSettings struct {
	MaxSessionCount int      `json:"max_session_count"`
	MaxIterations   int      `json:"max_iterations"`
	MaxWorkerCount  int      `json:"max_worker_count"`
	KConfigs        [][]int  `json:"k_configs"`
	NConfigs        []int    `json:"n_configs"`
	MConfigs        []int    `json:"m_configs"`
	LConfigs        []int    `json:"l_configs"`
	TpmTypes        []string `json:"tpm_types"`
	LearnRules      []string `json:"learn_rules"`
}
