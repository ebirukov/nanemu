#!/usr/bin/env sh

go install github.com/ebirukov/nanemu/cmd/nanemu@latest

nanemu -kernel https://dl-cdn.alpinelinux.org/alpine/edge/releases/x86_64/netboot/vmlinuz-virt -arch amd64 build/hello-amd64

nanemu -kernel https://dl-cdn.alpinelinux.org/alpine/edge/releases/aarch64/netboot/vmlinuz-virt -arch arm64 build/hello-arm64