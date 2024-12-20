package opfs

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"os"
	"sync"
	"syscall/js"
	"time"

	"github.com/insensatestone/afero-opfs/internal/async"
	"github.com/spf13/afero"
)

type File struct {
	fs           *Fs
	name         string
	parent       js.Value
	file_handler js.Value
	file         js.Value
	closed       bool
	flag         int
	p            int64
	once         sync.Once
	is_dir       bool
}

func (f *File) getFileHandle() error {
	f.once.Do(func() {
		var fa js.Value
		var err error
		if os.O_RDONLY&f.flag != 0 {
			fa, err = async.Await(f.file_handler.Call("createSyncAccessHandle", map[string]interface{}{"mode": "read-only"}))
		} else {
			fa, err = async.Await(f.file_handler.Call("createSyncAccessHandle"))
		}
		if err != nil {
			slog.Error("init file sync accesss handle failed", "err", err.Error())
		}
		f.file = fa
	})
	if f.file.IsNull() {
		return errors.New("init file sync accesss handle failed")
	}
	return nil
}
func (f *File) Name() string { return f.name }

func (f *File) Readdir(n int) ([]os.FileInfo, error) {
	if !f.is_dir {
		return nil, fs.ErrInvalid
	}
	if n <= 0 {
		n = math.MaxInt32
	}
	entries := f.file_handler.Call("values")

	var fis = make([]os.FileInfo, 0)

	for i := 0; i < n; i++ {
		next, err := async.Await(entries.Call("next"))
		if err != nil {
			return nil, err
		}
		if !next.Get("done").Bool() {
			handle := next.Get("value")
			if handle.Get("kind").String() == "file" {
				file, err := async.Await(handle.Call("getFile"))
				last_modified := 0
				size := 0
				if err != nil {
					last_modified = 0
					size = 0
				} else {
					last_modified = file.Get("lastModified").Int()
					size = file.Get("size").Int()
				}

				fis = append(fis, NewFileInfo(handle.Get("name").String(), false, int64(size), time.UnixMilli(int64(last_modified))))
			} else {
				fis = append(fis, NewFileInfo(handle.Get("name").String(), true, 0, time.Unix(0, 0)))
			}
		} else {
			break
		}
	}

	return fis, nil
}

// func (f *File) ReaddirAll() ([]os.FileInfo, error) {
// 	var fileInfos []os.FileInfo
// 	for {
// 		infos, err := f.Readdir(100)
// 		fileInfos = append(fileInfos, infos...)
// 		if err != nil {
// 			if errors.Is(err, io.EOF) {
// 				break
// 			} else {
// 				return nil, err
// 			}
// 		}
// 	}
// 	return fileInfos, nil
// }

func (f *File) Readdirnames(n int) ([]string, error) {
	if !f.is_dir {
		return nil, fs.ErrInvalid
	}
	fi, err := f.Readdir(n)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(fi))
	for i, f := range fi {
		names[i] = f.Name()
	}
	return names, nil
}

func (f *File) Stat() (os.FileInfo, error) {
	info, err := f.fs.Stat(f.Name())
	return info, err
}

func (f *File) Sync() error {
	if f.is_dir {
		return fs.ErrInvalid
	}
	if !f.file.IsNull() && !f.file.IsUndefined() {
		f.file.Call("flush")
	}
	return nil
}

func (f *File) Truncate(len int64) error {
	if f.is_dir {
		return fs.ErrInvalid
	}
	if !f.closed {
		err := f.getFileHandle()
		if err != nil {
			return err
		}
		f.file.Call("truncate", len)
	} else {
		return fs.ErrClosed
	}
	return nil
}

func (f *File) WriteString(s string) (int, error) {
	if f.is_dir {
		return 0, fs.ErrInvalid
	}
	return f.Write([]byte(s))
}

func (f *File) Close() error {
	if !f.closed {
		f.closed = true
		if !f.file.IsNull() && !f.file.IsUndefined() {
			f.file.Call("flush")
			f.file.Call("close")
		}
	}
	return nil
}

func (f *File) Read(p []byte) (int, error) {
	if f.is_dir {
		return 0, fs.ErrInvalid
	}
	if f.closed {
		return 0, fs.ErrClosed
	}
	if os.O_WRONLY&f.flag != 0 {
		return 0, fs.ErrPermission
	}
	err := f.getFileHandle()
	if err != nil {
		return 0, err
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
	if f.is_dir {
		return 0, fs.ErrInvalid
	}
	_, err = f.Seek(off, io.SeekStart)
	if err != nil {
		return
	}
	n, err = f.Read(p)
	return
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if f.is_dir {
		return 0, fs.ErrInvalid
	}
	if f.closed {
		return 0, afero.ErrFileClosed
	}

	switch whence {
	case io.SeekStart:
		f.p = offset
	case io.SeekCurrent:
		f.p = f.p + offset
	case io.SeekEnd:
		err := f.getFileHandle()
		if err != nil {
			return 0, err
		}
		js_size := f.file.Call("getSize")
		f.p = int64(js_size.Int()) + offset
	}
	if f.p < 0 {
		f.p = 0
	}
	return f.p, nil
}

func (f *File) Write(b []byte) (int, error) {
	if f.is_dir {
		return 0, fs.ErrInvalid
	}
	if f.closed {
		return 0, fs.ErrClosed
	}
	if os.O_RDONLY&f.flag != 0 {
		return 0, fs.ErrPermission
	}

	err := f.getFileHandle()
	if err != nil {
		return 0, err
	}
	jb := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(jb, b)
	nval := f.file.Call("write", jb, map[string]interface{}{"at": f.p})
	f.p = f.p + int64(nval.Int())
	return nval.Int(), nil
}

func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	if f.is_dir {
		return 0, fs.ErrInvalid
	}
	_, err = f.Seek(off, 0)
	if err != nil {
		return 0, err
	}
	return f.Write(p)

}
