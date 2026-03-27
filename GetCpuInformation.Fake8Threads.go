//go:build debug && Intel && Fake8Threads

package main

import "fmt"

func GetCpuInformation() []SYSTEM_CPU_SET_INFORMATION {
	fmt.Println("Fake8Threads")
	size := 0x20 * 64
	cpuSet := make([]SYSTEM_CPU_SET_INFORMATION, size)
	cpuSet[0].Size = 32
	var count = 256
	var index = 0x100
	for i := 0; i < count; i++ {
		cs := cpuSet[i].CpuSet()
		cs.Id = uint32(index + i)
		cs.LogicalProcessorIndex = byte(i)
		cs.CoreIndex = byte(i)
	}
	return cpuSet[:count]
}
