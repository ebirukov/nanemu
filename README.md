# nanemu

**nanemu** — минималистичная обертка для упрощения запуска ELF-бинарников в QEMU, поведение которых зависит от архитектуры и версии ядра Linux.

Проект предназначен для:
- 🔬 Тестирования кода использующего низкоуровневые механизмы ОС, такие как eBPF, системные вызовы и т.п.
- 🧪 Отладки поведения кода в различных архитектурных и версионных условиях
- ⚙️ Эмуляции окружения с минимальной загрузкой и настройкой

⚡ **Субсекундное время запуска** на хост-системах с поддержкой KVM и минималистичным ядром.

---

## Быстрый старт

1. **Установите QEMU**

Для Alpine / Debian / Ubuntu:

```bash
sudo apt install qemu-system qemu-user-static
````

Официальная документация: [QEMU installation guide](https://www.qemu.org/download/)

2. **Скачайте ядро Linux**

Например, для Alpine ARM64:

[https://dl-cdn.alpinelinux.org/alpine/edge/releases/](https://dl-cdn.alpinelinux.org/alpine/edge/releases/)

3. **Установите `nanemu`**

```bash
GOBIN=$GOPATH/bin go install github.com/ebirukov/nanemu/cmd/nanemu@latest
```

---

## Примеры использования

🔧 **Запуск одиночного файла init:**

```bash
QEMU_ARGS="-machine virt -cpu cortex-a53" \
go run cmd/qemu-runner/main.go \
  -kernel kernel/arm64/linux-5.10.0-32-arm64
  -rootfs init
  -arch arm64
```

🖥 **Запуск с кастомными параметрами:**

```bash
QEMU_BIN=/usr/bin/qemu-system-amd64 \
KERNEL_ARGS="initrd=/mybin cma=0 audit=0 nowatchdog nosmp maxcpus=1 ipv6.disable=1 net.ifnames=0 lsm= acpi=off ima_appraise=off" \
go run cmd/qemu-runner/main.go \
  -kernel kernel/amd64/linux-6.1.0-35-amd64 \
  -rootfs build/amd64/initramfs \
  -arch arm64
  -timeout 10s
```

---

## Параметры

| Флаг       | Тип        | Описание                                                     |
| ---------- | ---------- | ------------------------------------------------------------ |
| `-kernel`  | `string`   | Путь к образу ядра Linux (**обязательный**)                  |
| `-rootfs`  | `string`   | Путь к `initramfs`                                           |
| `-arch`    | `string`   | Целевая архитектура (**по умолчанию:** `GOARCH`)             |
| `-timeout` | `duration` | Максимальное время выполнения QEMU (**по умолчанию:** `30s`) |

---

## Переменные окружения

| Переменная    | Описание                                                               |
| ------------- | ---------------------------------------------------------------------- |
| `QEMU_BIN`    | Путь к исполняемому файлу QEMU (**по умолчанию:** `qemu-system-$ARCH`) |
| `QEMU_ARGS`   | Дополнительные аргументы, передаваемые напрямую в QEMU                 |
| `KERNEL_ARGS` | Дополнительные параметры загрузки ядра     |

---

## Особенности

* 🚀 Субсекундный запуск при использовании KVM и минимальных ядрах
* 🏗 Автоматически создаёт временный `initramfs` в формате `cpio`, если не указан
* 🧼 Автоматическая очистка временных файлов после завершения
* 🛠 Поддержка архитектур `amd64` и `arm64`

---

## Пример: запуск собственной программы

1. Возьмем hello linux программу:

```go
package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func main() {
	var uts syscall.Utsname
	syscall.Uname(&uts)
	osVersion := unsafe.String((*byte)(unsafe.Pointer(&uts.Release[0])), len(uts.Release))

	fmt.Printf(
		"Hello from process %d on linux kernel version: %s\n", os.Getpid(), osVersion)

	syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
}
```

2. Скомпилируем под нужную архитектуру:

```bash
GOARCH=arm64 go build -o build/hello-arm64 ./cmd/hello
```

3. Запускаем с помощью `nanemu`:

```bash
KERNEL_ARGS='rdinit=/hello-arm64 console=ttyAMA0 loglevel=6'

nanemu \
  -kernel kernel/vmlinuz-5.4.43-1-arm64 \
  -rootfs build/hello-arm64 \
  -arch arm64
```

4. Пример вывода:

```
2025/08/23 12:49:47 executing: /usr/bin/qemu-system-arm64 -nodefaults -serial mon:stdio -machine virt -cpu cortex-a53 -nographic -no-reboot -append rdinit=/hello-arm64 console=ttyAMA0 loglevel=6 -kernel kernel/vmlinuz-5.4.43-1-arm64 -initrd ./initramfs.cpio718173793
2025/08/23 12:49:47 process /usr/bin/qemu-system-arm64 started with pid: 2177874
[    0.000000] Linux version 5.4.43-1-lts (buildozer@build-edge-aarch64) (gcc version 9.3.0 (Alpine 9.3.0)) #2-Alpine SMP Thu, 28 May 2020 20:13:48 UTC
[    0.000000] Kernel command line: rdinit=/hello-arm64 console=ttyAMA0 loglevel=6
Hello from process 1 on linux kernel version: 5.4.43-1-lts
[    0.558718] reboot: Power down
2025/08/23 12:49:48 process 2177874 complete with code 0
```

---
