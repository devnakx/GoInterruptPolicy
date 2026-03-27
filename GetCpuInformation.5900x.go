//go:build debug && AMD && 5900X

package main

import "fmt"

func GetCpuInformation() []SYSTEM_CPU_SET_INFORMATION {
	fmt.Println("AMD 5900X")
	size := 0x20 * 64
	cpuSet := make([]SYSTEM_CPU_SET_INFORMATION, size)
	cpuSet[0].Size = 32
	var lastCoreIndex byte
	var count = 768
	var index = 0x100
	for i := 0; i < count; i++ {
		cs := cpuSet[i].CpuSet()
		cs.Id = uint32(index + i)
		cs.LogicalProcessorIndex = byte(i)
		if i%2 != 0 {
			cs.CoreIndex = lastCoreIndex
		} else {
			cs.CoreIndex = byte(i)
			lastCoreIndex = byte(i)
		}
		if i > 11 {
			cs.LastLevelCacheIndex = 12
		}
	}
	return cpuSet[:count]
}
