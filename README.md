# opfs Backend for Afero

## About
It provides an [afero filesystem](https://github.com/spf13/afero/) implementation of an [opfs](https://developer.mozilla.org/en-US/docs/Web/API/File_System_API/Origin_private_file_system) backend.

This was created to provide a backend when used in wasm.

I'm very opened to any improvement through issues or pull-request that might lead to a better implementation or even
better testing.

## TO DO


## How to use
```golang

import(
	"github.com/insensatestone/afero-opfs/pkg"
)

func main() {
  opfs,_ := opfs.NewFs()

  // And do your thing
  file, _ := fs.OpenFile("file.txt", os.O_WRONLY, 0777)
  file.WriteString("Hello world !")
  file.Close()
}
```
