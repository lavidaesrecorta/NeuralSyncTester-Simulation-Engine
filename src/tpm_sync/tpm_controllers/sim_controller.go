package tpm_controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/beevik/ntp"
	"github.com/sourcegraph/conc/pool"
)

type SimulationController struct {
	SyncController     SyncController
	DatabaseController DatabaseController
}

// Function to read and deserialize JSON file
func (s *SimulationController) LoadSimulationSettings(filename string) (*SimulationSettings, error) {
	// Read the JSON file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Create an instance of SimulationSettings
	var settings SimulationSettings

	// Unmarshal JSON data into the struct
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &settings, nil
}

func (s *SimulationController) SimulateOnStart(sessionMap *SessionMap) {

	simSettings, err := s.LoadSimulationSettings("simulation_settings.json")
	if err != nil {
		log.Fatalf("Error loading settings: %v", err)
	}

	fmt.Println("Settings loaded:")
	fmt.Println(simSettings)

	workerPool := pool.New().WithMaxGoroutines(simSettings.MaxWorkerCount)
	for _, tpm_type := range simSettings.TpmTypes {
		for _, rule := range simSettings.LearnRules {
			for _, k := range simSettings.KConfigs {
				for _, l := range simSettings.LConfigs {
					for _, n_0 := range simSettings.NConfigs {
						for _, m := range simSettings.MConfigs {
							tpmSettings, err := s.SyncController.SettingsFactory(k, n_0, l, m, tpm_type, rule)
							if err != nil {
								continue
							}

							workerPool.Go(func() {
								startTime, ntpErr := s.getCurrentTimeFromNTP()
								if ntpErr != nil {
									startTime = time.Now()
								}
								token := s.generateToken(startTime, tpmSettings)
								sessionBufferSize := 10
								sessionChannel := make(chan SessionStateMessage, sessionBufferSize)
								simulationData := OpenSession{
									Uid:                 token,
									Config:              tpmSettings,
									StartTime:           startTime,
									MaxSessionCount:     simSettings.MaxSessionCount,
									CurrentSessionCount: 0,
									CurrentStateChannel: sessionChannel,
								}
								sessionMap.Mutex.Lock()
								sessionMap.Sessions[token] = &simulationData
								sessionMap.Mutex.Unlock()

								for i := 0; i < simSettings.MaxSessionCount; i++ {
									startTime, ntpErr = s.getCurrentTimeFromNTP()
									if ntpErr != nil {
										startTime = time.Now()
									}
									seed := time.Now().UnixNano()
									localRand := rand.New(rand.NewSource(seed))
									sendIterThreshold := 10
									sendIterStep := 100
									session := s.SyncController.StartSyncSession(tpmSettings, sessionChannel, simSettings.MaxIterations, sendIterThreshold, sendIterStep, seed, localRand)

									endTime, ntpErr := s.getCurrentTimeFromNTP()
									if ntpErr != nil {
										endTime = time.Now()
									}
									s.DatabaseController.insertIntoDB(tpmSettings, session, startTime, endTime)
									sessionMap.Mutex.Lock()
									sessionMap.Sessions[token].CurrentSessionCount += 1
									sessionMap.Mutex.Unlock()
								}

								sessionMap.Mutex.Lock()
								delete(sessionMap.Sessions, token)
								sessionMap.Mutex.Unlock()
								close(sessionChannel)
							})
						}
					}
				}
			}
		}
	}
	workerPool.Wait()
	fmt.Println("-- All automatic configs finished --")
}

func (s *SimulationController) getCurrentTimeFromNTP() (time.Time, error) {
	// You can use a specific NTP server or use a pool
	ntpServer := "ntp.shoa.cl"
	// Retrieve the time from the NTP server
	time, err := ntp.Time(ntpServer)
	if err != nil {
		return time, fmt.Errorf("failed to get time from NTP server: %v", err)
	}
	return time, nil
}

func (s *SimulationController) generateToken(startTime time.Time, config TPMmSettings) string {
	idStamp := fmt.Sprintf("%v%d%d%d%s", config.K, config.N[0], config.L, config.M, startTime)
	h := sha256.New()
	h.Write([]byte(idStamp))
	token := hex.EncodeToString(h.Sum(nil))
	return token
}

func NewSessionMap() *SessionMap {
	return &SessionMap{
		Sessions: make(map[string]*OpenSession),
	}
}
