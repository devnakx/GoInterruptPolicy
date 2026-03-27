//go:build debug && AMD

package main

func isAMD() bool {
	return true
}

func isIntel() bool {
	return false
}
