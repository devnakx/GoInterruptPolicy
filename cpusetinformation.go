package main

import (
	"fmt"
	"sort"
)

var SystemCpuSets = []SYSTEM_CPU_SET_INFORMATION{}

const (
	ToolTipTextNumaNode        = "A group-relative value indicating which NUMA node a CPU Set is on. All CPU Sets in a given group that are on the same NUMA node will have the same value for this field."
	ToolTipTextLastLevelCache  = "A group-relative value indicating which CPU Sets share at least one level of cache with each other. This value is the same for all CPU Sets in a group that are on processors that share cache with each other."
	ToolTipTextEfficiencyClass = "A value indicating the intrinsic energy efficiency of a processor for systems that support heterogeneous processors (such as ARM big.LITTLE systems). CPU Sets with higher numerical values of this field have home processors that are faster but less power-efficient than ones with lower values."
)

type CpuSets struct {
	MaxThreadsPerCore int
	CPU               []CpuSet
	CoreGroups        []CoreGroups
	CoreLayout        *CoreLayout
	HyperThreading    bool
	NumaNode          bool // A group-relative value indicating which NUMA node a CPU Set is on. All CPU Sets in a given group that are on the same NUMA node will have the same value for this field.
	LastLevelCache    bool // A group-relative value indicating which CPU Sets share at least one level of cache with each other. This value is the same for all CPU Sets in a group that are on processors that share cache with each other.
	EfficiencyClass   bool // A value indicating the intrinsic energy efficiency of a processor for systems that support heterogeneous processors (such as ARM big.LITTLE systems). CPU Sets with higher numerical values of this field have home processors that are faster but less power-efficient than ones with lower values.
}

type CpuSet struct {
	Id                    uint32
	CoreIndex             byte
	LogicalProcessorIndex byte
	LastLevelCacheIndex   byte // A group-relative value indicating which CPU Sets share at least one level of cache with each other. This value is the same for all CPU Sets in a group that are on processors that share cache with each other.
	EfficiencyClass       byte // A value indicating the intrinsic energy efficiency of a processor for systems that support heterogeneous processors (such as ARM big.LITTLE systems). CPU Sets with higher numerical values of this field have home processors that are faster but less power-efficient than ones with lower values.
	NumaNodeIndex         byte // A group-relative value indicating which NUMA node a CPU Set is on. All CPU Sets in a given group that are on the same NUMA node will have the same value for this field.
}

type CoreGroups struct {
	Rows int
	Cols int
}

type CoreLayout struct {
	Numa []NumaItem
}

type NumaItem struct {
	Ccd []CcdItem
}

type CcdItem struct {
	Eff EffFields
}

type EffFields struct {
	Nums map[int][][]int
}

func (item *EffFields) isNil() bool {
	if item.Nums == nil {
		return true
	}
	return false
}

func (item *CoreLayout) add(numa, ccd, effClass, core, thread int) {
	if len(item.Numa) <= numa {
		item.Numa = append(item.Numa, make([]NumaItem, numa+1-len(item.Numa))...)
	}

	if len(item.Numa[numa].Ccd) <= ccd {
		item.Numa[numa].Ccd = append(item.Numa[numa].Ccd, make([]CcdItem, ccd+1-len(item.Numa[numa].Ccd))...)
	}

	if item.Numa[numa].Ccd[ccd].Eff.Nums == nil {
		item.Numa[numa].Ccd[ccd].Eff.Nums = make(map[int][][]int)
	}
	rows := item.Numa[numa].Ccd[ccd].Eff.Nums[effClass]

	if len(rows) <= core {
		rows = append(rows, make([][]int, core+1-len(rows))...)
	}

	rows[core] = append(rows[core], thread)

	item.Numa[numa].Ccd[ccd].Eff.Nums[effClass] = rows
}

func (cs *CpuSets) Init() {
	SystemCpuSets = GetCpuInformation()

	cs.CoreLayout = new(CoreLayout)

	var ClassGroup = []int{}
	var lastEfficiencyClass, lastLevelCache, lastNumaNodeIndex byte
	for i := 0; i < int(uint32(len(SystemCpuSets))/SystemCpuSets[0].Size); i++ {
		cpu := SystemCpuSets[i].CpuSet()
		cs.CPU = append(cs.CPU, CpuSet{
			Id:                    cpu.Id,
			CoreIndex:             cpu.CoreIndex,
			LogicalProcessorIndex: cpu.LogicalProcessorIndex,
			EfficiencyClass:       cpu.EfficiencyClass,
			LastLevelCacheIndex:   cpu.LastLevelCacheIndex,
			NumaNodeIndex:         cpu.NumaNodeIndex,
		})

		fmt.Printf("(%02d) [%d/%x] %02d/%02d Eff%d CCD%d NUMA%d\n", i, cpu.Id, cpu.Id, cpu.CoreIndex, cpu.LogicalProcessorIndex, cpu.EfficiencyClass, cpu.LastLevelCacheIndex, cpu.NumaNodeIndex)

		cs.CoreLayout.add(int(cpu.NumaNodeIndex), int(cpu.LastLevelCacheIndex), int(cpu.EfficiencyClass), int(cpu.CoreIndex), int(cpu.LogicalProcessorIndex))

		if cs.MaxThreadsPerCore < int(cpu.LogicalProcessorIndex-cpu.CoreIndex) {
			cs.MaxThreadsPerCore = int(cpu.LogicalProcessorIndex-cpu.CoreIndex) + 1
		}

		for len(ClassGroup) <= int(cpu.EfficiencyClass) {
			ClassGroup = append(ClassGroup, 0)
		}
		ClassGroup[int(cpu.EfficiencyClass)]++

		if !cs.HyperThreading && cpu.CoreIndex != cpu.LogicalProcessorIndex {
			cs.HyperThreading = true
		}

		if !cs.EfficiencyClass && lastEfficiencyClass != cpu.EfficiencyClass {
			cs.EfficiencyClass = true
		}

		if !cs.LastLevelCache && lastLevelCache != cpu.LastLevelCacheIndex {
			cs.LastLevelCache = true
		}

		if !cs.NumaNode && lastNumaNodeIndex != cpu.NumaNodeIndex {
			cs.NumaNode = true
		}
	}

	sort.Slice(cs.CPU, func(i, j int) bool {
		if cs.CPU[i].EfficiencyClass != cs.CPU[j].EfficiencyClass {
			return cs.CPU[i].EfficiencyClass < cs.CPU[j].EfficiencyClass
		}
		if cs.CPU[i].CoreIndex != cs.CPU[j].CoreIndex {
			return cs.CPU[i].CoreIndex < cs.CPU[j].CoreIndex
		}
		return cs.CPU[i].LogicalProcessorIndex < cs.CPU[j].LogicalProcessorIndex
	})

	rows, cols := getLayout(ClassGroup...)
	for _, col := range cols {
		cs.CoreGroups = append(cs.CoreGroups, CoreGroups{Rows: rows, Cols: col})
	}
}
