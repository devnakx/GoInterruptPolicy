package main

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// sys	setupDiEnumDeviceInfo(deviceInfoSet DevInfo, memberIndex uint32, deviceInfoData *DevInfoData) (err error) = setupapi.SetupDiEnumDeviceInfo

// SetupDiEnumDeviceInfo function returns a DevInfoData structure that specifies a device information element in a device information set.
func SetupDiEnumDeviceInfo(deviceInfoSet DevInfo, memberIndex int) (*DevInfoData, error) {
	data := &DevInfoData{}
	data.size = uint32(unsafe.Sizeof(*data))

	return data, setupDiEnumDeviceInfo(deviceInfoSet, uint32(memberIndex), data)
}

// sys	setupDiOpenDevRegKey(deviceInfoSet DevInfo, deviceInfoData *DevInfoData, Scope DICS_FLAG, HwProfile uint32, KeyType DIREG, samDesired uint32) (key windows.Handle, err error) [failretval==windows.InvalidHandle] = setupapi.SetupDiOpenDevRegKey

// SetupDiOpenDevRegKey function opens a registry key for device-specific configuration information.
func SetupDiOpenDevRegKey(deviceInfoSet DevInfo, deviceInfoData *DevInfoData, scope DICS_FLAG, hwProfile uint32, keyType DIREG, samDesired uint32) (registry.Key, error) {
	handle, err := setupDiOpenDevRegKey(deviceInfoSet, deviceInfoData, scope, hwProfile, keyType, samDesired)
	return registry.Key(handle), err
}

// sys	setupDiGetDeviceRegistryProperty(deviceInfoSet DevInfo, deviceInfoData *DevInfoData, property SPDRP, propertyRegDataType *uint32, propertyBuffer *byte, propertyBufferSize uint32, requiredSize *uint32) (err error) = setupapi.SetupDiGetDeviceRegistryPropertyW

// SetupDiGetDeviceRegistryProperty function retrieves a specified Plug and Play device property.
func SetupDiGetDeviceRegistryProperty(deviceInfoSet DevInfo, deviceInfoData *DevInfoData, property SPDRP) (value any, err error) {
	buf := make([]byte, 0x100)
	var dataType, bufLen uint32
	err = setupDiGetDeviceRegistryProperty(deviceInfoSet, deviceInfoData, property, &dataType, &buf[0], uint32(cap(buf)), &bufLen)
	if err == nil {
		// The buffer was sufficiently big.
		return getRegistryValue(buf[:bufLen], dataType)
	}

	if errWin, ok := err.(syscall.Errno); ok && errWin == windows.ERROR_INSUFFICIENT_BUFFER {
		// The buffer was too small. Now that we got the required size, create another one big enough and retry.
		buf = make([]byte, bufLen)
		err = setupDiGetDeviceRegistryProperty(deviceInfoSet, deviceInfoData, property, &dataType, &buf[0], uint32(cap(buf)), &bufLen)
		if err == nil {
			return getRegistryValue(buf[:bufLen], dataType)
		}
	}

	return
}

func getRegistryValue(buf []byte, dataType uint32) (any, error) {
	switch dataType {
	case windows.REG_SZ:
		return windows.UTF16ToString(BufToUTF16(buf)), nil
	case windows.REG_EXPAND_SZ:
		return registry.ExpandString(windows.UTF16ToString(BufToUTF16(buf)))
	case windows.REG_BINARY:
		return buf, nil
	case windows.REG_DWORD_LITTLE_ENDIAN:
		return binary.LittleEndian.Uint32(buf), nil
	case windows.REG_DWORD_BIG_ENDIAN:
		return binary.BigEndian.Uint32(buf), nil
	case windows.REG_MULTI_SZ:
		bufW := BufToUTF16(buf)
		a := []string{}
		for i := 0; i < len(bufW); {
			j := i + wcslen(bufW[i:])
			if i < j {
				a = append(a, windows.UTF16ToString(bufW[i:j]))
			}
			i = j + 1
		}
		return a, nil
	case windows.REG_QWORD_LITTLE_ENDIAN:
		return binary.LittleEndian.Uint64(buf), nil
	default:
		return nil, fmt.Errorf("unsupported registry value type: %v", dataType)
	}
}

func BufToUTF16(buf []byte) []uint16 {
	if len(buf) == 0 {
		return nil
	}
	return unsafe.Slice((*uint16)(unsafe.Pointer(&buf[0])), len(buf)/2)
}

func wcslen(str []uint16) int {
	for i := range str {
		if str[i] == 0 {
			return i
		}
	}
	return len(str)
}

// sys	setupDiGetDeviceInstallParams(deviceInfoSet DevInfo, deviceInfoData *DevInfoData, deviceInstallParams *DevInstallParams) (err error) = setupapi.SetupDiGetDeviceInstallParamsW

// SetupDiGetDeviceInstallParams function retrieves device installation parameters for a device information set or a particular device information element.
func SetupDiGetDeviceInstallParams(deviceInfoSet DevInfo, deviceInfoData *DevInfoData) (*DevInstallParams, error) {
	params := &DevInstallParams{}
	params.size = uint32(unsafe.Sizeof(*params))

	return params, setupDiGetDeviceInstallParams(deviceInfoSet, deviceInfoData, params)
}
