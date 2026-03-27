//go:build debug && Intel && FakeNumaCCD12Core

package main

import "fmt"

func GetCpuInformation() []SYSTEM_CPU_SET_INFORMATION {
	fmt.Println("FakeNumaCCD12Core")
	size := 0x20 * 64
	cpuSet := make([]SYSTEM_CPU_SET_INFORMATION, size)
	cpuSet[0].Size = 32
	var count = 384
	var index = 0x100
	for i := 0; i < count; i++ {
		cs := cpuSet[i].CpuSet()
		cs.Id = uint32(index + i)
		cs.LogicalProcessorIndex = byte(i)
		cs.CoreIndex = byte(i)

		if i > 5 {
			cs.LastLevelCacheIndex = 6
			cs.NumaNodeIndex = 6
		}
	}
	return cpuSet[:count]
}
