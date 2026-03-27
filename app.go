//go:generate goversioninfo -64

package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"unsafe"

	"github.com/tailscale/walk"
	"github.com/tailscale/win"

	//lint:ignore ST1001 standard behavior tailscale/walk
	. "github.com/tailscale/walk/declarative"
)

var cs CpuSets

const (
	tableColumnContentPadding = 16
	tableColumnHeaderPadding  = 28
	tableAutoWidthRowLimit    = 1200
)

func main() {
	cs.Init()

	var devices []Device
	devices, handle = FindAllDevices()

	if CLIMode {
		var newItem *Device
		for i := 0; i < len(devices); i++ {
			if devices[i].DevObjName == flagDevObjName {
				newItem = &devices[i]
				break
			}
		}
		if newItem == nil {
			SetupDiDestroyDeviceInfoList(handle)
			os.Exit(1)
		}
		orgItem := *newItem

		var assignmentSetOverride Bits
		if flagCPU != "" {
			for val := range strings.SplitSeq(flagCPU, ",") {
				i, err := strconv.Atoi(val)
				if err != nil {
					log.Println(err)
					continue
				}
				assignmentSetOverride = Set(assignmentSetOverride, CPUBits[i])
			}
		}

		if flagMsiSupported != -1 || flagMessageNumberLimit != -1 {
			if flagMsiSupported != -1 {
				newItem.MsiSupported = uint32(flagMsiSupported)
			}
			if flagMessageNumberLimit != -1 {
				newItem.MessageNumberLimit = uint32(flagMessageNumberLimit)
			}
			orgItem.setMSIMode(newItem)
		}

		if flagDevicePolicy != -1 || flagDevicePriority != -1 || assignmentSetOverride != ZeroBit {
			if flagDevicePolicy != -1 {
				newItem.DevicePolicy = uint32(flagDevicePolicy)
			}
			if flagDevicePriority != -1 {
				newItem.DevicePriority = uint32(flagDevicePriority)
			}
			if assignmentSetOverride != ZeroBit {
				newItem.AssignmentSetOverride = assignmentSetOverride
			}
			orgItem.setAffinityPolicy(newItem)
		}

		changed := orgItem.MsiSupported != newItem.MsiSupported || orgItem.MessageNumberLimit != newItem.MessageNumberLimit || orgItem.DevicePolicy != newItem.DevicePolicy || orgItem.DevicePriority != newItem.DevicePriority || orgItem.AssignmentSetOverride != newItem.AssignmentSetOverride
		if flagRestart || (flagRestartOnChange && changed) {
			propChangeParams := PropChangeParams{
				ClassInstallHeader: *MakeClassInstallHeader(DIF_PROPERTYCHANGE),
				StateChange:        DICS_PROPCHANGE,
				Scope:              DICS_FLAG_GLOBAL,
			}

			if err := SetupDiSetClassInstallParams(handle, &newItem.Idata, &propChangeParams.ClassInstallHeader, uint32(unsafe.Sizeof(propChangeParams))); err != nil {
				log.Println(err)
				return
			}

			if err := SetupDiCallClassInstaller(DIF_PROPERTYCHANGE, handle, &newItem.Idata); err != nil {
				log.Println(err)
				return
			}

			if err := SetupDiSetClassInstallParams(handle, &newItem.Idata, &propChangeParams.ClassInstallHeader, uint32(unsafe.Sizeof(propChangeParams))); err != nil {
				log.Println(err)
				return
			}

			if err := SetupDiCallClassInstaller(DIF_PROPERTYCHANGE, handle, &newItem.Idata); err != nil {
				log.Println(err)
				return
			}

			DeviceInstallParams, err := SetupDiGetDeviceInstallParams(handle, &newItem.Idata)
			if err != nil {
				log.Println(err)
				return
			}

			if DeviceInstallParams.Flags&DI_NEEDREBOOT != 0 {
				fmt.Println("Device could not be restarted. Changes will take effect the next time you reboot.")
			} else {
				fmt.Println("Device successfully restarted.")
			}
		}
		SetupDiDestroyDeviceInfoList(handle)
		os.Exit(0)
	}
	defer SetupDiDestroyDeviceInfoList(handle)

	sort.Slice(devices, func(i, j int) bool {
		if devices[i].LocationInformation == devices[j].LocationInformation {
			return devices[i].DeviceDesc < devices[j].DeviceDesc
		}
		return devices[i].LocationInformation > devices[j].LocationInformation
	})

	AllDevices := devices

	var LineEditSearch *walk.LineEdit
	var devicePolicyAction *walk.Action
	var hardwarePropertiesAction *walk.Action
	var registryParametersAction *walk.Action
	mw := &MyMainWindow{
		model: &Model{items: devices},
		tv:    &walk.TableView{},
	}
	applyDefaultSort := func() {
		sorter, ok := mw.tv.TableModel().(walk.Sorter)
		if !ok {
			return
		}
		if err := sorter.Sort(3, walk.SortDescending); err != nil {
			log.Println(err)
		}
	}
	if err := (MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "GoInterruptPolicy - " + Version,
		Icon:     2,
		MinSize: Size{
			Width:  240,
			Height: 320,
		},
		Size: Size{
			Width:  750,
			Height: 600,
		},
		Layout: VBox{
			MarginsZero: true,
			SpacingZero: true,
		},
		Children: []Widget{
			Composite{
				Layout: VBox{},
				Children: []Widget{
					LineEdit{
						AssignTo:  &LineEditSearch,
						CueBanner: "Search",
						OnTextChanged: func() {
							text := strings.ToLower(LineEditSearch.Text())
							if text == "" {
								mw.tv.SetModel(&Model{items: AllDevices})
								applyDefaultSort()
								mw.autoAdjustColumns(false)
								mw.sbi.SetText(fmt.Sprintf("%d Devices Found", len(devices)))
							} else {
								newDevices := []Device{}
								for i := range AllDevices {
									if strings.Contains(strings.ToLower(AllDevices[i].DeviceDesc), text) ||
										strings.Contains(strings.ToLower(AllDevices[i].DevObjName), text) ||
										strings.Contains(strings.ToLower(AllDevices[i].LocationInformation), text) ||
										strings.Contains(strings.ToLower(AllDevices[i].FriendlyName), text) {
										newDevices = append(newDevices, AllDevices[i])
									}
								}
								mw.tv.SetModel(&Model{items: newDevices})
								applyDefaultSort()
								mw.autoAdjustColumns(false)
								mw.sbi.SetText(fmt.Sprintf("%d Devices Found", len(newDevices)))
							}
						},
					},
				},
			},
			TableView{
				OnItemActivated:     mw.lb_ItemActivated,
				AlternatingRowBG:    true,
				ColumnsOrderable:    true,
				ColumnsSizable:      true,
				LastColumnStretched: true,
				MultiSelection:      true,
				ContextMenuItems: []MenuItem{
					Action{
						AssignTo: &devicePolicyAction,
						Text:     "&Device policy",
						OnTriggered: func() {
							SelectedIndexes := mw.tv.SelectedIndexes()
							if len(SelectedIndexes) != 0 {
								deviceList := make([]Device, len(SelectedIndexes))
								for arrayIndex, selectionIndex := range SelectedIndexes {
									deviceList[arrayIndex] = mw.tv.Model().(*Model).items[selectionIndex]
								}

								result, d, err := RunDialog(mw, deviceList)
								if err != nil {
									log.Print(err)
									return
								}

								if result == 0 || result == win.IDCANCEL {
									return
								}

								// mw.restartPopup = true
								for _, selectionIndex := range SelectedIndexes {
									mw.tv.Model().(*Model).items[selectionIndex] = mw.SetNewDevice(&mw.tv.Model().(*Model).items[selectionIndex], &d)
								}

								for _, selectionIndex := range SelectedIndexes {
									mw.tv.UpdateItem(selectionIndex)
								}
							}
						},
					},

					Action{
						AssignTo: &hardwarePropertiesAction,
						Text:     "&Hardware properties",
						OnTriggered: func() {
							SelectedIndexes := mw.tv.SelectedIndexes()
							if len(SelectedIndexes) != 0 {
								for _, index := range SelectedIndexes {
									d := mw.tv.Model().(*Model).items[index]
									if id, err := d.getInstanceID(); err == nil {
										go showDeviceProperties(win.HWND(0), id)
									}
								}
							}
						},
					},

					Action{
						AssignTo: &registryParametersAction,
						Text:     "&Registry parameters",
						OnTriggered: func() {
							index := mw.tv.CurrentIndex()
							d := mw.tv.Model().(*Model).items[index]
							OpenRegistry(mw.Form(), d.reg)
						},
					},
				},
				Model:    mw.model,
				AssignTo: &mw.tv,
				OnKeyUp: func(key walk.Key) {
					model, ok := mw.tv.Model().(*Model)
					if !ok {
						return
					}

					i := mw.tv.CurrentIndex()
					if i == -1 {
						i = 0
					}
					for ; i < len(model.items); i++ {
						item := &model.items[i]
						if item.DeviceDesc != "" && key.String() == item.DeviceDesc[0:1] {
							if err := mw.tv.SetCurrentIndex(i); err != nil {
								log.Println(err)
							}
							return
						}
					}
				},
				Columns: []TableViewColumn{
					{
						Name:  "DeviceDesc",
						Title: "Name",
						Width: 150,
					},
					{
						Name:  "FriendlyName",
						Title: "Friendly Name",
					},

					{
						Name: "Class",
					},
					{
						Name:  "LocationInformation",
						Title: "Location Info",
						Width: 150,
					},
					{
						Name:      "MsiSupported",
						Title:     "MSI Mode",
						Width:     60,
						Alignment: AlignCenter,
						FormatFunc: func(value any) string {
							switch value.(uint32) {
							case 0:
								return "✖"
							case 1:
								return "✔"
							}
							return ""
						},
					},
					{
						Name:  "DevicePolicy",
						Title: "Device Policy",
						FormatFunc: func(value any) string {
							// https://docs.microsoft.com/en-us/windows-hardware/drivers/kernel/interrupt-affinity-and-priority
							switch value.(uint32) {
							case IrqPolicyMachineDefault: // 0x00
								return "Default"
							case IrqPolicyAllCloseProcessors: // 0x01
								return "All Close Proc"
							case IrqPolicyOneCloseProcessor: // 0x02
								return "One Close Proc"
							case IrqPolicyAllProcessorsInMachine: // 0x03
								return "All Proc in Machine"
							case IrqPolicySpecifiedProcessors: // 0x04
								return "Specified Proc"
							case IrqPolicySpreadMessagesAcrossAllProcessors: // 0x05
								return "Spread Messages Across All Proc"
							default:
								return fmt.Sprintf("%d", value.(uint32))
							}
						},
					},
					{
						Name:  "AssignmentSetOverride",
						Title: "Specified Processor",
						FormatFunc: func(value any) string {
							if value == ZeroBit {
								return ""
							}
							bits := value.(Bits)
							var result []string
							for bit, cpu := range CPUMap {
								if Has(bit, bits) {
									result = append(result, cpu)
								}
							}

							result, err := sortNumbers(result)
							if err != nil {
								log.Println(err)
							}
							return strings.Join(result, ",")
						},
						LessFunc: func(i, j int) bool {
							model, ok := mw.tv.Model().(*Model)
							if !ok || model == nil {
								return false
							}
							return model.items[i].AssignmentSetOverride < model.items[j].AssignmentSetOverride
						},
					},
					{
						Name:  "DevicePriority",
						Title: "Device Priority",
						FormatFunc: func(value any) string {
							switch value.(uint32) {
							case 0:
								return "Undefined"
							case 1:
								return "Low"
							case 2:
								return "Normal"
							case 3:
								return "High"
							default:
								return fmt.Sprintf("%d", value.(uint32))
							}
						},
					},
					{
						Name:  "InterruptTypeMap",
						Title: "Interrupt Type",
						Width: 120,
						FormatFunc: func(value any) string {
							return interruptType(value.(Bits))
						},
						LessFunc: func(i, j int) bool {
							model, ok := mw.tv.Model().(*Model)
							if !ok || model == nil {
								return false
							}
							return model.items[i].InterruptTypeMap < model.items[j].InterruptTypeMap
						},
					},
					{
						Name:  "MessageNumberLimit",
						Title: "MSI Limit",
						FormatFunc: func(value any) string {
							switch value.(uint32) {
							case 0:
								return ""
							default:
								return fmt.Sprintf("%d", value.(uint32))
							}
						},
					},
					{
						Name:  "MaxMSILimit",
						Title: "Max MSI Limit",
						FormatFunc: func(value any) string {
							switch value.(uint32) {
							case 0:
								return ""
							default:
								return fmt.Sprintf("%d", value.(uint32))
							}
						},
					},

					{
						Name:  "IRQLanes",
						Title: "IRQ Lanes",
						FormatFunc: func(value any) string {
							return strings.Join(value.([]string), ", ")
						},
					},

					{
						Name:  "DevObjName",
						Title: "DevObj Name",
					},
				},
			},
		},
		StatusBarItems: []StatusBarItem{
			{
				AssignTo: &mw.sbi,
				Text:     fmt.Sprintf("%d Devices Found", len(devices)),
			},
		},
	}).Create(); err != nil {
		log.Println(err)
		return
	}
	mw.autoAdjustColumns(true)
	applyDefaultSort()

	win.ShowWindow(mw.Handle(), win.SW_SHOWMAXIMIZED)
	mw.tv.SetFocus()
	mw.Run()
}

type MyMainWindow struct {
	*walk.MainWindow
	tv           *walk.TableView
	model        *Model
	sbi          *walk.StatusBarItem
	restartPopup bool
	autoWidths   []int
}

func (mw *MyMainWindow) lb_ItemActivated() {
	currentIndex := mw.tv.CurrentIndex()
	newItem := mw.tv.Model().(*Model).items[currentIndex]

	result, d, err := RunDialog(mw, []Device{newItem})
	if err != nil {
		log.Print(err)
		return
	}
	if result == 0 || result == win.IDCANCEL {
		return
	}

	// mw.restartPopup = true
	mw.tv.Model().(*Model).items[currentIndex] = mw.SetNewDevice(&mw.tv.Model().(*Model).items[currentIndex], &d)

	mw.tv.UpdateItem(currentIndex)
}

func (mw *MyMainWindow) SetNewDevice(orgItem *Device, newItem *Device) Device {
	var changed bool

	if orgItem.MsiSupported != newItem.MsiSupported || orgItem.MessageNumberLimit != newItem.MessageNumberLimit {
		changed = orgItem.setMSIMode(newItem)
	}

	if orgItem.DevicePolicy != newItem.DevicePolicy || orgItem.DevicePriority != newItem.DevicePriority || orgItem.AssignmentSetOverride != newItem.AssignmentSetOverride {
		changed = orgItem.setAffinityPolicy(newItem) || changed
	}

	if mw.restartPopup && changed {
		switch walk.MsgBox(mw.WindowBase.Form(), "Restart Device?", fmt.Sprintf("Your changes will not take effect until the device is restarted.\n\nWould you like to attempt to restart the device now?\n\n%s\n%s", orgItem.DeviceDesc, orgItem.DevObjName), walk.MsgBoxYesNoCancel|walk.MsgBoxIconQuestion) {
		case win.IDCANCEL:
			mw.restartPopup = false
			fallthrough
		case win.IDNO:
			mw.sbi.SetText("Restart required")
		case win.IDYES:
			propChangeParams := PropChangeParams{
				ClassInstallHeader: *MakeClassInstallHeader(DIF_PROPERTYCHANGE),
				StateChange:        DICS_PROPCHANGE,
				Scope:              DICS_FLAG_GLOBAL,
			}

			if err := SetupDiSetClassInstallParams(handle, &orgItem.Idata, &propChangeParams.ClassInstallHeader, uint32(unsafe.Sizeof(propChangeParams))); err != nil {
				log.Println(err)
				return *orgItem
			}

			if err := SetupDiCallClassInstaller(DIF_PROPERTYCHANGE, handle, &orgItem.Idata); err != nil {
				log.Println(err)
				return *orgItem
			}

			if err := SetupDiSetClassInstallParams(handle, &orgItem.Idata, &propChangeParams.ClassInstallHeader, uint32(unsafe.Sizeof(propChangeParams))); err != nil {
				log.Println(err)
				return *orgItem
			}

			if err := SetupDiCallClassInstaller(DIF_PROPERTYCHANGE, handle, &orgItem.Idata); err != nil {
				log.Println(err)
				return *orgItem
			}

			DeviceInstallParams, err := SetupDiGetDeviceInstallParams(handle, &orgItem.Idata)
			if err != nil {
				log.Println(err)
				return *orgItem
			}

			if DeviceInstallParams.Flags&DI_NEEDREBOOT != 0 {
				walk.MsgBox(mw.WindowBase.Form(), "Notice", "Device could not be restarted. Changes will take effect the next time you reboot.", walk.MsgBoxIconWarning)
			} else {
				walk.MsgBox(mw.WindowBase.Form(), "Notice", "Device successfully restarted.", walk.MsgBoxIconInformation)
			}

		}
	}
	return *orgItem
}

func (mw *MyMainWindow) TextWidthSize(text string) int {
	canvas, err := (*mw.tv).CreateCanvas()
	if err != nil {
		return 0
	}
	defer canvas.Dispose()

	bounds, _, err := canvas.MeasureTextPixels(text, (*mw.tv).Font(), walk.Rectangle{Width: 9999999}, walk.TextCalcRect)
	if err != nil {
		return 0
	}

	return bounds.Size().Width
}

type tableAutoColumn struct {
	Title    string
	MinWidth int
	MaxWidth int
	Value    func(Device) string
}

type tableAutoWidthColumnInput struct {
	Title    string
	Values   []string
	MinWidth int
	MaxWidth int
}

func splitCellLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	lines := strings.Split(value, "\n")
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func calculateAutoColumnWidths(columns []tableAutoWidthColumnInput, measure func(string) int, cellPadding int, headerPadding int) []int {
	widths := make([]int, len(columns))
	for i, column := range columns {
		maxWidth := measure(column.Title) + headerPadding
		for _, value := range column.Values {
			for _, line := range splitCellLines(value) {
				candidate := measure(line) + cellPadding
				if candidate > maxWidth {
					maxWidth = candidate
				}
				if column.MaxWidth > 0 && maxWidth >= column.MaxWidth {
					maxWidth = column.MaxWidth
					break
				}
			}
			if column.MaxWidth > 0 && maxWidth >= column.MaxWidth {
				break
			}
		}
		if maxWidth < column.MinWidth {
			maxWidth = column.MinWidth
		}
		if column.MaxWidth > 0 && maxWidth > column.MaxWidth {
			maxWidth = column.MaxWidth
		}
		widths[i] = maxWidth
	}
	return widths
}

func (mw *MyMainWindow) tableAutoColumns() []tableAutoColumn {
	return []tableAutoColumn{
		{
			Title:    "Name",
			MinWidth: 120,
			MaxWidth: 600,
			Value: func(device Device) string {
				return device.DeviceDesc
			},
		},
		{
			Title:    "Friendly Name",
			MinWidth: 120,
			MaxWidth: 600,
			Value: func(device Device) string {
				return device.FriendlyName
			},
		},
		{
			Title:    "Class",
			MinWidth: 90,
			MaxWidth: 220,
			Value: func(device Device) string {
				return device.Class
			},
		},
		{
			Title:    "Location Info",
			MinWidth: 150,
			MaxWidth: 640,
			Value: func(device Device) string {
				return device.LocationInformation
			},
		},
		{
			Title:    "MSI Mode",
			MinWidth: 72,
			MaxWidth: 120,
			Value: func(device Device) string {
				switch device.MsiSupported {
				case 0:
					return "✖"
				case 1:
					return "✔"
				default:
					return ""
				}
			},
		},
		{
			Title:    "Device Policy",
			MinWidth: 140,
			MaxWidth: 360,
			Value: func(device Device) string {
				switch device.DevicePolicy {
				case IrqPolicyMachineDefault:
					return "Default"
				case IrqPolicyAllCloseProcessors:
					return "All Close Proc"
				case IrqPolicyOneCloseProcessor:
					return "One Close Proc"
				case IrqPolicyAllProcessorsInMachine:
					return "All Proc in Machine"
				case IrqPolicySpecifiedProcessors:
					return "Specified Proc"
				case IrqPolicySpreadMessagesAcrossAllProcessors:
					return "Spread Messages Across All Proc"
				default:
					return fmt.Sprintf("%d", device.DevicePolicy)
				}
			},
		},
		{
			Title:    "Specified Processor",
			MinWidth: 150,
			MaxWidth: 620,
			Value: func(device Device) string {
				if device.AssignmentSetOverride == ZeroBit {
					return ""
				}
				bits := device.AssignmentSetOverride
				result := make([]string, 0, len(CPUMap))
				for bit, cpu := range CPUMap {
					if Has(bit, bits) {
						result = append(result, cpu)
					}
				}
				result, err := sortNumbers(result)
				if err != nil {
					log.Println(err)
				}
				return strings.Join(result, ",")
			},
		},
		{
			Title:    "Device Priority",
			MinWidth: 120,
			MaxWidth: 220,
			Value: func(device Device) string {
				switch device.DevicePriority {
				case 0:
					return "Undefined"
				case 1:
					return "Low"
				case 2:
					return "Normal"
				case 3:
					return "High"
				default:
					return fmt.Sprintf("%d", device.DevicePriority)
				}
			},
		},
		{
			Title:    "Interrupt Type",
			MinWidth: 110,
			MaxWidth: 220,
			Value: func(device Device) string {
				return interruptType(device.InterruptTypeMap)
			},
		},
		{
			Title:    "MSI Limit",
			MinWidth: 90,
			MaxWidth: 160,
			Value: func(device Device) string {
				if device.MessageNumberLimit == 0 {
					return ""
				}
				return fmt.Sprintf("%d", device.MessageNumberLimit)
			},
		},
		{
			Title:    "Max MSI Limit",
			MinWidth: 110,
			MaxWidth: 180,
			Value: func(device Device) string {
				if device.MaxMSILimit == 0 {
					return ""
				}
				return fmt.Sprintf("%d", device.MaxMSILimit)
			},
		},
		{
			Title:    "IRQ Lanes",
			MinWidth: 90,
			MaxWidth: 90,
			Value: func(device Device) string {
				return strings.Join(device.IRQLanes, ", ")
			},
		},
		{
			Title:    "DevObj Name",
			MinWidth: 160,
			MaxWidth: 520,
			Value: func(device Device) string {
				return device.DevObjName
			},
		},
	}
}

func (mw *MyMainWindow) autoAdjustColumns(force bool) {
	items, ok := currentDevicesFromTableModel(mw.tv.Model())
	if !ok && mw.model != nil {
		items = mw.model.items
		ok = true
	}
	if !ok {
		return
	}

	autoColumns := mw.tableAutoColumns()
	columnInputs := make([]tableAutoWidthColumnInput, len(autoColumns))
	valueRowLimit := len(items)
	if valueRowLimit > tableAutoWidthRowLimit {
		valueRowLimit = tableAutoWidthRowLimit
	}
	for i, column := range autoColumns {
		columnInputs[i] = tableAutoWidthColumnInput{
			Title:    column.Title,
			MinWidth: column.MinWidth,
			MaxWidth: column.MaxWidth,
			Values:   make([]string, 0, valueRowLimit),
		}
		for rowIndex := 0; rowIndex < valueRowLimit; rowIndex++ {
			columnInputs[i].Values = append(columnInputs[i].Values, column.Value(items[rowIndex]))
		}
	}

	widthCache := map[string]int{}
	measure := func(text string) int {
		if width, ok := widthCache[text]; ok {
			return width
		}
		width := mw.TextWidthSize(text)
		widthCache[text] = width
		return width
	}

	targetWidths := calculateAutoColumnWidths(columnInputs, measure, tableColumnContentPadding, tableColumnHeaderPadding)

	if !force && len(mw.autoWidths) == len(targetWidths) {
		changed := false
		for i := range targetWidths {
			if mw.autoWidths[i] != targetWidths[i] {
				changed = true
				break
			}
		}
		if !changed {
			return
		}
	}

	columns := mw.tv.Columns()
	for i := 0; i < len(targetWidths) && i < columns.Len(); i++ {
		if columns.At(i).Width() == targetWidths[i] {
			continue
		}
		columns.At(i).SetWidth(targetWidths[i])
	}
	mw.autoWidths = targetWidths
}

func currentDevicesFromTableModel(model any) ([]Device, bool) {
	switch typedModel := model.(type) {
	case *Model:
		return typedModel.items, true
	case interface{ Items() any }:
		items := typedModel.Items()
		if devices, ok := items.([]Device); ok {
			return devices, true
		}
	}
	return nil, false
}

type Model struct {
	// walk.SortedReflectTableModelBase
	items []Device
}

func (m *Model) Items() any {
	return m.items
}

func sortNumbers(data []string) ([]string, error) {
	var lastErr error
	sort.Slice(data, func(i, j int) bool {
		a, err := strconv.ParseInt(data[i], 10, 64)
		if err != nil {
			lastErr = err
			return false
		}
		b, err := strconv.ParseInt(data[j], 10, 64)
		if err != nil {
			lastErr = err
			return false
		}
		return a < b
	})
	return data, lastErr
}
