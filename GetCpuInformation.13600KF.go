//go:build debug && Intel && 13600KF

package main

import "fmt"

func GetCpuInformation() []SYSTEM_CPU_SET_INFORMATION {
	fmt.Println("Intel i5 13600KF")
	size := 0x20 * 64
	cpuSet := make([]SYSTEM_CPU_SET_INFORMATION, size)
	cpuSet[0].Size = 32
	var lastCoreIndex byte
	var count = 640
	var index = 0x100
	for i := 0; i < count; i++ {
		cs := cpuSet[i].CpuSet()
		cs.Id = uint32(index + i)
		cs.LogicalProcessorIndex = byte(i)
		if i < 12 && i%2 != 0 {
			cs.CoreIndex = lastCoreIndex
		} else {
			cs.CoreIndex = byte(i)
			lastCoreIndex = byte(i)
		}
		if i < 12 {
			cs.EfficiencyClass = 1
		}
	}
	return cpuSet[:count]
}
