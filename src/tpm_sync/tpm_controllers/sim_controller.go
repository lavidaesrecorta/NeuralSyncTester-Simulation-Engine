package tpm_controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beevik/ntp"
	"github.com/sourcegraph/conc/pool"
)

type SimulationController struct {
	SyncController     SyncController
	DatabaseController DatabaseController
	WorkerPool         *pool.Pool
}

func ReadFile(filename string) ([]byte, error) {
	// Read file content into a byte slice
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func UnmarshalSettings(data []byte) (BaseSettings, error) {
	var baseSettings BaseSettings
	err := json.Unmarshal(data, &baseSettings)
	if err != nil {
		return BaseSettings{}, err
	}
	return baseSettings, nil
}

// Function to read and deserialize JSON file
func (s *SimulationController) LoadSimulationSettings(filename string) (interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil, err
	}
	defer file.Close()

	// Read the entire file into a byte slice
	fileInfo, err := os.Stat(filename)
	if err != nil {
		fmt.Println("Error getting file stats:", err)
		return nil, err
	}
	fileData := make([]byte, fileInfo.Size())
	_, err = file.Read(fileData)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return nil, err
	}

	// Decode the file into a map
	var rawConfig map[string]interface{}
	if err := json.Unmarshal(fileData, &rawConfig); err != nil {
		fmt.Println("Error unmarshalling config:", err)
		return nil, err
	}

	return rawConfig, nil
}

func (s *SimulationController) SimulateInstance(sessionMap *SessionMap, tpmSettings TPMmSettings, simSettings BaseSettings) string {

	startTime, ntpErr := s.getCurrentTimeFromNTP()
	if ntpErr != nil {
		startTime = time.Now()
	}
	token := s.generateToken(startTime, tpmSettings)
	// sessionBufferSize := 10
	enableTrackingChannel := make(chan bool)
	sessionChannel := make(chan SessionStateMessage)
	simulationData := OpenSession{
		Uid:                 token,
		Config:              tpmSettings,
		StartTime:           startTime,
		MaxSessionCount:     simSettings.MaxSessionCount,
		CurrentSessionCount: 0,
		Tracking:            false,
		CurrentStateChannel: sessionChannel,
		EnableStateChannel:  enableTrackingChannel,
	}
	sessionMap.Mutex.Lock()
	sessionMap.Sessions[token] = &simulationData
	sessionMap.Mutex.Unlock()

	s.WorkerPool.Go(func() {

		for i := 0; i < simSettings.MaxSessionCount; i++ {
			startTime, ntpErr = s.getCurrentTimeFromNTP()
			if ntpErr != nil {
				startTime = time.Now()
			}
			seed := time.Now().UnixNano()
			localRand := rand.New(rand.NewSource(seed))
			sendIterThreshold := 10
			sendIterStep := 100
			sessionMap.Mutex.RLock()
			tracking := sessionMap.Sessions[token].Tracking
			sessionMap.Mutex.RUnlock()
			session := s.SyncController.StartSyncSession(tpmSettings, tracking, sessionChannel, enableTrackingChannel, simSettings.MaxIterations, sendIterThreshold, sendIterStep, seed, localRand)

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
	return token
}

func (s *SimulationController) SimulateOnStart(sessionMap *SessionMap) {

	rawConfig, err := s.LoadSimulationSettings("simulation_settings.json")
	if err != nil {
		return
	}
	// Initialize a variable for the base settings
	var baseSettings BaseSettings
	baseSettingsData, _ := json.Marshal(rawConfig)

	if err := json.Unmarshal(baseSettingsData, &baseSettings); err != nil {
		fmt.Println("Error unmarshalling base settings:", err)
		return
	}

	fmt.Println("Settings loaded:")
	fmt.Println(baseSettings)

	for _, rule := range baseSettings.LearnRules {
		for _, m := range baseSettings.MConfigs {
			for _, l := range baseSettings.LConfigs {
				switch strings.ToUpper(baseSettings.TpmType) {
				case "NO_OVERLAP":
					var noOverlapSettings NonOverlappedSettings

					if err := json.Unmarshal(baseSettingsData, &noOverlapSettings); err != nil {
						fmt.Println("Error unmarshalling noOverlap settings:", err)
					}

					for _, n := range noOverlapSettings.NConfigs {
						for _, k_last := range noOverlapSettings.KlastConfigs {
							tpmInstanceSettings, err := s.SyncController.SettingsFactory(n, k_last, l, m, noOverlapSettings.TpmType, rule)
							if err != nil {
								fmt.Println("Error while creating settings for an instance: ", err)
								return
							}
							s.SimulateInstance(sessionMap, tpmInstanceSettings, baseSettings)
						}
					}
				default:
					var overlapSettings OverlappedSettings

					if err := json.Unmarshal(baseSettingsData, &overlapSettings); err != nil {
						fmt.Println("Error unmarshalling overlapped settings:", err)
					}

					for _, k := range overlapSettings.KConfigs {
						for _, n_0 := range overlapSettings.N0Configs {
							tpmInstanceSettings, err := s.SyncController.SettingsFactory(k, n_0, l, m, overlapSettings.TpmType, rule)
							if err != nil {
								fmt.Println("Error while creating settings for an instance: ", err)
								return
							}
							s.SimulateInstance(sessionMap, tpmInstanceSettings, baseSettings)

						}
					}
				}
			}
		}
	}
	s.WorkerPool.Wait()
	fmt.Println("-- All automatic configs finished --")
}

func (s *SimulationController) SimulateMultipleFiles(sessionMap *SessionMap, configFileDirectory string) {

	files, err := os.ReadDir(configFileDirectory)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, file := range files {
		fmt.Printf("Reading config file: %s\n", file.Name())

		rawConfig, err := s.LoadSimulationSettings(filepath.Join(configFileDirectory, file.Name()))
		if err != nil {
			fmt.Printf("Error loading base settings for file %s: %s", file.Name(), err)
			continue
		}
		// Initialize a variable for the base settings
		var baseSettings BaseSettings
		baseSettingsData, _ := json.Marshal(rawConfig)

		if err := json.Unmarshal(baseSettingsData, &baseSettings); err != nil {
			fmt.Printf("Error unmarshalling base settings for file %s: %s", file.Name(), err)
			continue
		}

		fmt.Printf("%s Settings loaded: \n", file.Name())
		fmt.Println(baseSettings)

		for _, rule := range baseSettings.LearnRules {
			for _, m := range baseSettings.MConfigs {
				for _, l := range baseSettings.LConfigs {
					switch strings.ToUpper(baseSettings.TpmType) {
					case "NO_OVERLAP":
						var noOverlapSettings NonOverlappedSettings

						if err := json.Unmarshal(baseSettingsData, &noOverlapSettings); err != nil {
							fmt.Printf("Error unmarshalling noOverlap settings for file %s: %s\n", file.Name(), err)
							continue
						}

						for _, n := range noOverlapSettings.NConfigs {
							for _, k_last := range noOverlapSettings.KlastConfigs {
								tpmInstanceSettings, err := s.SyncController.SettingsFactory(n, k_last, l, m, noOverlapSettings.TpmType, rule)
								if err != nil {
									fmt.Printf("Error while creating settings for an instance for file %s: %s \n", file.Name(), err)
									continue
								}
								s.SimulateInstance(sessionMap, tpmInstanceSettings, baseSettings)
							}
						}
					default:
						var overlapSettings OverlappedSettings

						if err := json.Unmarshal(baseSettingsData, &overlapSettings); err != nil {
							fmt.Printf("Error unmarshalling overlapped settings for file %s: %s\n", file.Name(), err)
						}

						for _, k := range overlapSettings.KConfigs {
							for _, n_0 := range overlapSettings.N0Configs {
								tpmInstanceSettings, err := s.SyncController.SettingsFactory(k, n_0, l, m, overlapSettings.TpmType, rule)
								if err != nil {
									fmt.Printf("Error while creating settings for an instance for file %s: %s \n", file.Name(), err)
									continue
								}
								s.SimulateInstance(sessionMap, tpmInstanceSettings, baseSettings)

							}
						}
					}
				}
			}
		}
		// fmt.Printf("-- All automatic configs finished for file %s --\n", file.Name())
	}
	s.WorkerPool.Wait()
	fmt.Printf("-- All automatic configs finished for all files --\n")

}

func (s *SimulationController) SimulateOnDemand(sessionMap *SessionMap, tpmInstanceSettings TPMmSettings, baseSettings BaseSettings) string {
	return s.SimulateInstance(sessionMap, tpmInstanceSettings, baseSettings)
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
