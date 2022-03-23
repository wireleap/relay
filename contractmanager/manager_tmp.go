// Copyright (c) 2022 Wireleap

package contractmanager

import (
	"encoding/json"
	"fmt"
	"log"
)

func (m *Manager) PrintStatus() {
	ms := m.Status()

	msJSON, err := json.MarshalIndent(ms, "", "  ")
	if err != nil {
		log.Println(err)
	}
	fmt.Println(string(msJSON))
}
