package main

// +build amd64,darwin

import "syscall"

var (
	infoSig = syscall.SIGUSR1
)
