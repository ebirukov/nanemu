package fsutil

// Inode идентифицирует файл на хосте по inode и числу ссылок. Позволяет
// распознавать хардлинки (несколько имён с одинаковым inode). На платформах без
// поддержки inode (например Windows) InodeKeyOf возвращает false.
type Inode struct {
	Ino   int64
	Nlink int
}
