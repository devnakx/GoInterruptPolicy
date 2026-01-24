package main

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var _ unsafe.Pointer

// Do the interface allocations only once for common
// Errno values.
const (
	errnoERROR_IO_PENDING = 997
)

var (
	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	// TODO: add more here, after collecting data on the common
	// error values see on Windows. (perhaps when running
	// all.bat?)
	return e
}

var (
	modsetupapi = windows.NewLazySystemDLL("setupapi.dll")

	procSetupDiEnumDeviceInfo             = modsetupapi.NewProc("SetupDiEnumDeviceInfo")
	procSetupDiDestroyDeviceInfoList      = modsetupapi.NewProc("SetupDiDestroyDeviceInfoList")
	procSetupDiCallClassInstaller         = modsetupapi.NewProc("SetupDiCallClassInstaller")
	procSetupDiOpenDevRegKey              = modsetupapi.NewProc("SetupDiOpenDevRegKey")
	procSetupDiGetDeviceRegistryPropertyW = modsetupapi.NewProc("SetupDiGetDeviceRegistryPropertyW")
	procSetupDiGetDeviceInstallParamsW    = modsetupapi.NewProc("SetupDiGetDeviceInstallParamsW")
	procSetupDiSetClassInstallParamsW     = modsetupapi.NewProc("SetupDiSetClassInstallParamsW")
	procSetupDiGetDevicePropertyW         = modsetupapi.NewProc("SetupDiGetDevicePropertyW")
	procSetupDiGetDeviceInstanceIdW       = modsetupapi.NewProc("SetupDiGetDeviceInstanceIdW")
)

func setupDiEnumDeviceInfo(deviceInfoSet DevInfo, memberIndex uint32, deviceInfoData *DevInfoData) (err error) {
	r1, _, e1 := syscall.SyscallN(procSetupDiEnumDeviceInfo.Addr(), uintptr(deviceInfoSet), uintptr(memberIndex), uintptr(unsafe.Pointer(deviceInfoData)))
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func SetupDiDestroyDeviceInfoList(deviceInfoSet DevInfo) (err error) {
	r1, _, e1 := syscall.SyscallN(procSetupDiDestroyDeviceInfoList.Addr(), uintptr(deviceInfoSet))
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

// func SetupDiBuildDriverInfoList(deviceInfoSet DevInfo, deviceInfoData *DevInfoData, driverType SPDIT) (err error) {
// 	r1, _, e1 := syscall.SyscallN(procSetupDiBuildDriverInfoList.Addr(), uintptr(deviceInfoSet), uintptr(unsafe.Pointer(deviceInfoData)), uintptr(driverType))
// 	if r1 == 0 {
// 		if e1 != 0 {
// 			err = errnoErr(e1)
// 		} else {
// 			err = syscall.EINVAL
// 		}
// 	}
// 	return
// }

// func SetupDiCancelDriverInfoSearch(deviceInfoSet DevInfo) (err error) {
// 	r1, _, e1 := syscall.SyscallN(procSetupDiCancelDriverInfoSearch.Addr(), uintptr(deviceInfoSet))
// 	if r1 == 0 {
// 		if e1 != 0 {
// 			err = errnoErr(e1)
// 		} else {
// 			err = syscall.EINVAL
// 		}
// 	}
// 	return
// }

// func SetupDiSetSelectedDriver(deviceInfoSet DevInfo, deviceInfoData *DevInfoData, driverInfoData *DrvInfoData) (err error) {
// 	r1, _, e1 := syscall.SyscallN(procSetupDiSetSelectedDriverW.Addr(), uintptr(deviceInfoSet), uintptr(unsafe.Pointer(deviceInfoData)), uintptr(unsafe.Pointer(driverInfoData)))
// 	if r1 == 0 {
// 		if e1 != 0 {
// 			err = errnoErr(e1)
// 		} else {
// 			err = syscall.EINVAL
// 		}
// 	}
// 	return
// }

// func SetupDiDestroyDriverInfoList(deviceInfoSet DevInfo, deviceInfoData *DevInfoData, driverType SPDIT) (err error) {
// 	r1, _, e1 := syscall.SyscallN(procSetupDiDestroyDriverInfoList.Addr(), uintptr(deviceInfoSet), uintptr(unsafe.Pointer(deviceInfoData)), uintptr(driverType))
// 	if r1 == 0 {
// 		if e1 != 0 {
// 			err = errnoErr(e1)
// 		} else {
// 			err = syscall.EINVAL
// 		}
// 	}
// 	return
// }

func SetupDiCallClassInstaller(installFunction DI_FUNCTION, deviceInfoSet DevInfo, deviceInfoData *DevInfoData) (err error) {
	r1, _, e1 := syscall.SyscallN(
		procSetupDiCallClassInstaller.Addr(),
		uintptr(installFunction),
		uintptr(deviceInfoSet),
		uintptr(unsafe.Pointer(deviceInfoData)),
	)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func setupDiOpenDevRegKey(deviceInfoSet DevInfo, deviceInfoData *DevInfoData, scope DICS_FLAG, hwProfile uint32, keyType DIREG, samDesired uint32) (key windows.Handle, err error) {
	r0, _, e1 := syscall.SyscallN(procSetupDiOpenDevRegKey.Addr(), uintptr(deviceInfoSet), uintptr(unsafe.Pointer(deviceInfoData)), uintptr(scope), uintptr(hwProfile), uintptr(keyType), uintptr(samDesired))
	key = windows.Handle(r0)
	if key == windows.InvalidHandle {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func setupDiGetDeviceRegistryProperty(deviceInfoSet DevInfo, deviceInfoData *DevInfoData, property SPDRP, propertyRegDataType *uint32, propertyBuffer *byte, propertyBufferSize uint32, requiredSize *uint32) (err error) {
	r1, _, e1 := syscall.SyscallN(procSetupDiGetDeviceRegistryPropertyW.Addr(), uintptr(deviceInfoSet), uintptr(unsafe.Pointer(deviceInfoData)), uintptr(property), uintptr(unsafe.Pointer(propertyRegDataType)), uintptr(unsafe.Pointer(propertyBuffer)), uintptr(propertyBufferSize), uintptr(unsafe.Pointer(requiredSize)))
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func setupDiGetDeviceInstallParams(deviceInfoSet DevInfo, deviceInfoData *DevInfoData, deviceInstallParams *DevInstallParams) (err error) {
	r1, _, e1 := syscall.SyscallN(procSetupDiGetDeviceInstallParamsW.Addr(), uintptr(deviceInfoSet), uintptr(unsafe.Pointer(deviceInfoData)), uintptr(unsafe.Pointer(deviceInstallParams)))
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func SetupDiSetClassInstallParams(deviceInfoSet DevInfo, deviceInfoData *DevInfoData, classInstallParams *ClassInstallHeader, classInstallParamsSize uint32) (err error) {
	r1, _, e1 := syscall.SyscallN(procSetupDiSetClassInstallParamsW.Addr(),
		uintptr(deviceInfoSet),
		uintptr(unsafe.Pointer(deviceInfoData)),
		uintptr(unsafe.Pointer(classInstallParams)),
		uintptr(classInstallParamsSize),
	)
	if r1 == 0 {
		if e1 != 0 {
			err = errnoErr(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

// https://github.com/tajtiattila/hid/blob/2bd63ffd4c8c0b81e5999c7b183cc4325f773527/platform/zsys_windows.go
func SetupDiGetDeviceProperty(devInfoSet DevInfo, devInfoData *DevInfoData, propKey *DEVPROPKEY, propType *uint32, propBuf *byte, propBufSize uint32, reqsize *uint32, flags uint32) (err error) {
	r1, _, e1 := syscall.SyscallN(procSetupDiGetDevicePropertyW.Addr(), uintptr(devInfoSet), uintptr(unsafe.Pointer(devInfoData)), uintptr(unsafe.Pointer(propKey)), uintptr(unsafe.Pointer(propType)), uintptr(unsafe.Pointer(propBuf)), uintptr(propBufSize), uintptr(unsafe.Pointer(reqsize)), uintptr(flags))
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func setupDiGetDeviceInstanceId(set DevInfo, devInfo *DevInfoData, devInstanceId unsafe.Pointer, devInstanceIdSize uint32, requiredSize *uint32) (err error) {
	r1, _, e1 := syscall.SyscallN(procSetupDiGetDeviceInstanceIdW.Addr(), uintptr(set), uintptr(unsafe.Pointer(devInfo)), uintptr(devInstanceId), uintptr(devInstanceIdSize), uintptr(unsafe.Pointer(requiredSize)), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
