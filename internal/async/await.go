package async

import (
	"errors"
	"io/fs"
	"syscall/js"
)

func Await(awaitable js.Value) (js.Value, error) {
	then := make(chan []js.Value)
	defer close(then)
	thenFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		then <- args
		return nil
	})
	defer thenFunc.Release()
	catch := make(chan []js.Value)
	defer close(catch)
	catchFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		catch <- args
		return nil
	})
	defer catchFunc.Release()
	awaitable.Call("then", thenFunc).Call("catch", catchFunc)
	select {
	case result := <-then:
		return result[0], nil
	case err := <-catch:
		if err[0].Get("name").String() == "NotFoundError" {
			return js.ValueOf(nil), fs.ErrNotExist
		} else {
			return js.ValueOf(nil), errors.New(err[0].Get("message").String())
		}
	}
}
