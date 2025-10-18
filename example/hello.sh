#!/usr/bin/env sh

go install github.com/ebirukov/nanemu/cmd/nanemu@latest

nanemu -kernel kernel/vmlinuz-5.4.43-1-arm64 -rootfs build/hello-arm64 -arch arm64 -loglevel=4