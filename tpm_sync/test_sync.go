package tpm_sync

import "fmt"

func SimulateOnStart() {

	neuronConfigs := [][]int{
		{3},
		{5, 3},
		{7, 5, 3},
		{13, 11, 9, 7, 5, 3},
		{15, 13, 11, 9, 7, 5, 3},
	}
	nConfigs := []int{2, 3, 4, 5, 6, 7, 8, 9}
	mConfigs := []int{1}

	lConfigs := []int{
		3,
		4,
		5,
		6,
		7,
		8,
		9,
		// 10,
		// 11,
		// 12,
		// 13,
	}

	tpmTypes := []string{
		"fully_connected",
		"partially_connected",
		"no_overlap",
	}

	learnRules := []string{
		"Hebbian",
		"Anti-Hebbian",
		"Random-Walk",
	}

	for _, tpm_type := range tpmTypes {
		for _, rule := range learnRules {
			for _, k := range neuronConfigs {
				for _, l := range lConfigs {
					for _, n_0 := range nConfigs {
						for _, m := range mConfigs {
							tpmSettings, err := SettingsFactory(k, n_0, l, m, tpm_type, rule)
							if err != nil {
								continue
							}
							session := SyncSession(tpmSettings, 100_000)
							fmt.Println(session.Status)
							// fmt.Println(session.FinalWeights)
						}
					}
				}
			}
		}
	}
	fmt.Println("-- All automatic configs finished --")
}
