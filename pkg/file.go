package opfs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"syscall/js"

	"github.com/insensatestone/afero-opfs/internal/async"
	"github.com/spf13/afero"
)

type File struct {
	fs           *Fs
	name         string
	parent       js.Value
	file_handler js.Value
	file         js.Value
	chunk        *bytes.Reader
	closed       bool
	flag         int
	p            int64
}

func (f *File) Name() string { return f.name }

func (f *File) Readdir(n int) ([]os.FileInfo, error) {

	// if n <= 0 {
	// 	return f.ReaddirAll()
	// }

	// var fis = make([]os.FileInfo, 0)

	// return fis, nil
	return nil, ErrNotImplemented
}

func (f *File) ReaddirAll() ([]os.FileInfo, error) {
	var fileInfos []os.FileInfo
	for {
		infos, err := f.Readdir(100)
		fileInfos = append(fileInfos, infos...)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			} else {
				return nil, err
			}
		}
	}
	return fileInfos, nil
}

func (f *File) Readdirnames(n int) ([]string, error) {
	fi, err := f.Readdir(n)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(fi))
	for i, f := range fi {
		_, names[i] = path.Split(f.Name())
	}
	return names, nil
}

func (f *File) Stat() (os.FileInfo, error) {
	info, err := f.fs.Stat(f.Name())
	return info, err
}

func (f *File) Sync() error {
	if !f.file.IsNull() && !f.file.IsUndefined() {
		f.file.Call("flush")
	}
	return nil
}

func (f *File) Truncate(len int64) error {
	if !f.file.IsNull() && !f.file.IsUndefined() {
		f.file.Call("truncate", len)
	}
	return ErrNotImplemented
}

func (f *File) WriteString(s string) (int, error) {
	return f.Write([]byte(s))
}

func (f *File) Close() error {
	f.closed = true
	if !f.file.IsNull() && !f.file.IsUndefined() {
		f.file.Call("flush")
		f.file.Call("close")
	}
	return nil
}

func (f *File) Read(p []byte) (int, error) {
	if f.closed {
		return 0, afero.ErrFileClosed
	}
	if os.O_WRONLY&f.flag != 0 {
		return 0, fs.ErrPermission
	}
	if f.file.IsNull() {
		fa, err := async.Await(f.file_handler.Call("createSyncAccessHandle"))
		if err != nil {
			return 0, err
		}
		f.file = fa
	}
	jb := js.Global().Get("Uint8Array").New(len(p))
	nval := f.file.Call("read", jb, map[string]interface{}{"at": f.p})
	if nval.Int() == 0 {
		return 0, io.EOF
	}
	js.CopyBytesToGo(p, jb)
	f.p = f.p + int64(nval.Int())
	return nval.Int(), nil
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	_, err = f.Seek(off, io.SeekStart)
	if err != nil {
		return
	}
	n, err = f.Read(p)
	return
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if f.closed {
		return 0, afero.ErrFileClosed
	}

	switch whence {
	case io.SeekStart:
		f.p = offset
	case io.SeekCurrent:
		f.p = f.p + offset
	case io.SeekEnd:
		return 0, ErrNotImplemented
	}

	return f.p, nil
}

func (f *File) Write(b []byte) (int, error) {
	if f.closed {
		return 0, afero.ErrFileClosed
	}
	if os.O_RDONLY&f.flag != 0 {
		return 0, fs.ErrPermission
	}

	if f.file.IsNull() {
		fa, err := async.Await(f.file_handler.Call("createSyncAccessHandle"))
		if err != nil {
			return 0, err
		}
		f.file = fa
	}
	jb := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(jb, b)
	nval := f.file.Call("write", jb, map[string]interface{}{"at": f.p})
	f.p = f.p + int64(nval.Int())
	return nval.Int(), nil
}

func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	_, err = f.Seek(off, 0)
	if err != nil {
		return 0, err
	}
	return f.Write(p)

}
