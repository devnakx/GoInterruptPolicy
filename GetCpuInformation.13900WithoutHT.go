//go:build debug && Intel && 13900WithoutHT

package main

import "log"

func GetCpuInformation() []SYSTEM_CPU_SET_INFORMATION {
	log.Println("Intel i9 13900 Without HT")
	size := 0x20 * 64
	cpuSet := make([]SYSTEM_CPU_SET_INFORMATION, size)
	cpuSet[0].Size = 32
	var count = 768
	var index = 0x100
	for i := 0; i < count; i++ {
		cs := cpuSet[i].CpuSet()
		cs.Id = uint32(index + i)
		cs.LogicalProcessorIndex = byte(i)
		cs.CoreIndex = byte(i)
		if i < 8 {
			cs.EfficiencyClass = 1
		}
	}
	return cpuSet[:count]
}
