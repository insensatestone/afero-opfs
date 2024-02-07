package opfs

import (
	"os"
	"time"
)

type FileInfo struct {
	modTime     time.Time
	name        string
	directory   bool
	sizeInBytes int64
}

func NewFileInfo(name string, directory bool, sizeInBytes int64, modTime time.Time) FileInfo {
	return FileInfo{
		name:        name,
		directory:   directory,
		sizeInBytes: sizeInBytes,
		modTime:     modTime,
	}
}

func (fi FileInfo) Name() string {
	return fi.name
}

func (fi FileInfo) Size() int64 {
	return fi.sizeInBytes
}

func (fi FileInfo) Mode() os.FileMode {
	if fi.directory {
		return 0755
	}
	return 0664
}

func (fi FileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi FileInfo) IsDir() bool {
	return fi.directory
}

func (fi FileInfo) Sys() interface{} {
	return nil
}
