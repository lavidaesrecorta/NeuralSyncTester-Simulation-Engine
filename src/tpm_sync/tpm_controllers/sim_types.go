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
	Tracking            bool                     `json:"-"`
	CurrentStateChannel chan SessionStateMessage `json:"-"`
	EnableStateChannel  chan bool                `json:"-"`
}

type SessionMap struct {
	Sessions map[string]*OpenSession
	Mutex    sync.RWMutex
}

type SessionStateMessage struct {
	CommandType  string //stimualte or finished
	SessionState interface{}
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

type BaseSettings struct {
	TpmType         string   `json:"tpm_type"`
	MaxSessionCount int      `json:"max_session_count"`
	MaxIterations   int      `json:"max_iterations"`
	MaxWorkerCount  int      `json:"max_worker_count"`
	LearnRules      []string `json:"learn_rules"`
	MConfigs        []int    `json:"m_configs"`
	LConfigs        []int    `json:"l_configs"`
}

type OverlappedSettings struct {
	BaseSettings
	KConfigs  [][]int `json:"k_configs"`
	N0Configs []int   `json:"n0_configs"`
}

type NonOverlappedSettings struct {
	BaseSettings
	KlastConfigs []int   `json:"klast_configs"`
	NConfigs     [][]int `json:"n_configs"`
}
