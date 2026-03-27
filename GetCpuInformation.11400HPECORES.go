//go:build debug && Intel && 11400HPECORES

package main

import "fmt"

func GetCpuInformation() []SYSTEM_CPU_SET_INFORMATION {
	fmt.Println("Intel i5 11400H With P-/E-Cores") // issue #14
	size := 0x20 * 64
	cpuSet := make([]SYSTEM_CPU_SET_INFORMATION, size)
	cpuSet[0].Size = 32
	var lastCoreIndex byte
	var count = 384
	var index = 0x100
	for i := 0; i < count; i++ {
		cs := cpuSet[i].CpuSet()
		cs.Id = uint32(index + i)
		cs.LogicalProcessorIndex = byte(i)
		if i%2 != 0 {
			cs.CoreIndex = lastCoreIndex
			cs.EfficiencyClass = 1
		} else {
			cs.CoreIndex = byte(i)
			lastCoreIndex = byte(i)
			cs.EfficiencyClass = 0
		}
	}
	return cpuSet[:count]
}
