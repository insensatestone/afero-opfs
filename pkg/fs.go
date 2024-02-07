package opfs

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall/js"
	"time"

	"github.com/insensatestone/afero-opfs/internal/async"
	"github.com/spf13/afero"
)

var (
	ErrInvalidPath    = errors.New("invalid path")
	ErrNotImplemented = errors.New("not implemented")
	ErrNotSupported   = errors.New("opfs doesn't support this operation")
	ErrInvalidSeek    = errors.New("invalid seek offset")
)

type Fs struct {
	root js.Value
}

func NewFs() (*Fs, error) {
	fs := js.Global().Get("navigator").Get("storage")

	opfs, err := async.Await(fs.Call("getDirectory"))
	if err != nil {
		return nil, err
	}

	return &Fs{
		root: opfs,
	}, nil
}

func (*Fs) Name() string { return "opfs" }

func (fs *Fs) Create(name string) (afero.File, error) {
	return fs.OpenFile(name, os.O_RDWR|os.O_CREATE, 0777)
}

func (fs *Fs) Mkdir(name string, perm os.FileMode) error {
	dirs, filename := filepath.Split(name)
	if dirs == "" || filename != "" {
		return ErrInvalidPath
	}

	dira := strings.Split(strings.Trim(dirs, string(filepath.Separator)), string(filepath.Separator))
	var dir js.Value
	var err error
	for _, dirname := range dira {
		dir, err = async.Await(dir.Call("getDirectoryHandle", dirname, map[string]interface{}{"create": true}))
		if err != nil {
			return err
		}
	}
	return nil
}

func (fs *Fs) MkdirAll(path string, perm os.FileMode) error {
	return fs.Mkdir(path, perm)
}

func (fs *Fs) Open(name string) (afero.File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0777)
}

func (fs *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	dirs, filename := filepath.Split(name)
	if filename == "" {
		return nil, ErrInvalidPath
	}

	create := (flag&os.O_CREATE != 0)

	dir := fs.root
	if dirs != "" {
		dira := strings.Split(strings.Trim(dirs, string(filepath.Separator)), string(filepath.Separator))
		var err error
		for _, dirname := range dira {
			dir, err = async.Await(dir.Call("getDirectoryHandle", dirname, map[string]interface{}{"create": create}))
			if err != nil {
				return nil, err
			}
		}
	}

	file, err := async.Await(dir.Call("getFileHandle", filename, map[string]interface{}{"create": create}))
	if err != nil {
		return nil, err
	}
	return &File{
		fs:           fs,
		name:         name,
		parent:       dir,
		file_handler: file,
		file:         js.ValueOf(nil),
		chunk:        nil,
		closed:       false,
		flag:         flag,
	}, nil
}

func (fs *Fs) Remove(name string) error {
	dirs, filename := filepath.Split(name)
	if filename == "" {
		return ErrInvalidPath
	}
	var err error
	dir := fs.root
	if dirs != "" {
		dira := strings.Split(strings.Trim(dirs, string(filepath.Separator)), string(filepath.Separator))
		for _, dirname := range dira {
			dir, err = async.Await(dir.Call("getDirectoryHandle", dirname, map[string]interface{}{"create": false}))
			if err != nil {
				return err
			}
		}
	}
	_, err = async.Await(dir.Call("removeEntry", filename))
	if err != nil {
		return err
	}
	return nil
}

func (fs *Fs) RemoveAll(name string) error {
	dirs, filename := filepath.Split(name)
	if dirs == "" || filename != "" {
		return ErrInvalidPath
	}
	var err error
	dir := fs.root
	dira := strings.Split(strings.Trim(dirs, string(filepath.Separator)), string(filepath.Separator))
	for idx, dirname := range dira {
		if idx < len(dira)-1 {
			dir, err = async.Await(dir.Call("getDirectoryHandle", dirname, map[string]interface{}{"create": false}))
			if err != nil {
				return err
			}
		}
	}
	_, err = async.Await(dir.Call("removeEntry", dira[len(dira)-1]))
	if err != nil {
		return err
	}
	return nil
}

func (fs *Fs) Rename(oldname, newname string) error {
	return ErrNotImplemented
}

func (fs *Fs) Stat(name string) (os.FileInfo, error) {
	dirs, filename := filepath.Split(name)
	if filename == "" {
		return fs.statDirectory(name)
	}

	dir := fs.root
	if dirs != "" {
		dira := strings.Split(strings.Trim(dirs, string(filepath.Separator)), string(filepath.Separator))
		var err error
		for _, dirname := range dira {
			dir, err = async.Await(dir.Call("getDirectoryHandle", dirname, map[string]interface{}{"create": false}))
			if err != nil {
				return nil, err
			}
		}
	}

	filehandle, err := async.Await(dir.Call("getFileHandle", filename, map[string]interface{}{"create": false}))
	if err != nil {
		return nil, err
	}

	fa, err := async.Await(filehandle.Call("createSyncAccessHandle"))
	if err != nil {
		return nil, err
	}

	file_size := fa.Call("getSize").Int()
	fa.Call("close")
	last_modified := 0
	file, err := async.Await(filehandle.Call("getFile"))
	if err != nil {
		last_modified = 0
	}
	last_modified = file.Get("lastModified").Int()
	return NewFileInfo(path.Base(name), false, int64(file_size), time.UnixMilli(int64(last_modified))), nil
}

func (fs *Fs) statDirectory(name string) (os.FileInfo, error) {

	return NewFileInfo(path.Base(name), true, 0, time.Unix(0, 0)), nil
}

func (fs *Fs) Chmod(name string, mode os.FileMode) error {
	return ErrNotSupported
}

func (*Fs) Chown(string, int, int) error {
	return ErrNotSupported
}

func (*Fs) Chtimes(string, time.Time, time.Time) error {
	return ErrNotSupported
}
