//go:build debug && Intel

package main

func isAMD() bool {
	return false
}

func isIntel() bool {
	return true
}
