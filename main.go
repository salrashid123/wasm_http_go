package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"syscall/js"
)

/*
from:

* `Omri Cohen's` [Run Go In The Browser Using WebAssembly](https://dev.bitolog.com/go-in-the-browser-using-webassembly/)
* `Alessandro Segala` [Go, WebAssembly, HTTP requests and Promises](https://withblue.ink/2020/10/03/go-webassembly-http-requests-and-promises.html)
*/

func main() {
	fmt.Println("============================================")
	fmt.Println("init wasm")
	fmt.Println("============================================")

	js.Global().Set("base64", encodeWrapper())
	js.Global().Set("MyGoFunc", MyGoFunc())
	js.Global().Set("MyGoFuncStream", MyGoFuncStream())
	<-make(chan bool)
}

func encodeWrapper() js.Func {

	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return wrap("", "Not enough arguments")
		}
		input := args[0].String()
		return wrap(base64.StdEncoding.EncodeToString([]byte(input)), "")
	})
}

func wrap(encoded string, err string) map[string]interface{} {
	return map[string]interface{}{
		"error":   err,
		"encoded": encoded,
	}
}

func MyGoFunc() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		requestUrl := args[0].String()
		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			reject := args[1]
			go func() {

				res, err := http.DefaultClient.Get(requestUrl)
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}
				defer res.Body.Close()

				data, err := ioutil.ReadAll(res.Body)
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}

				arrayConstructor := js.Global().Get("Uint8Array")
				dataJS := arrayConstructor.New(len(data))
				js.CopyBytesToJS(dataJS, data)

				responseConstructor := js.Global().Get("Response")
				response := responseConstructor.New(dataJS)

				resolve.Invoke(response)
			}()
			return nil
		})
		promiseConstructor := js.Global().Get("Promise")
		return promiseConstructor.New(handler)
	})
}

func MyGoFuncStream() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		requestUrl := args[0].String()

		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			reject := args[1]
			go func() {
				res, err := http.DefaultClient.Get(requestUrl)
				if err != nil {
					// Handle errors: reject the Promise if we have an error
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
					return
				}
				underlyingSource := map[string]interface{}{
					"start": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
						controller := args[0]

						go func() {
							defer res.Body.Close()
							for {
								// Read up to 16KB at a time
								buf := make([]byte, 16384)
								n, err := res.Body.Read(buf)
								if err != nil && err != io.EOF {
									errorConstructor := js.Global().Get("Error")
									errorObject := errorConstructor.New(err.Error())
									controller.Call("error", errorObject)
									return
								}
								if n > 0 {

									arrayConstructor := js.Global().Get("Uint8Array")
									dataJS := arrayConstructor.New(n)
									js.CopyBytesToJS(dataJS, buf[0:n])
									controller.Call("enqueue", dataJS)
								}
								if err == io.EOF {
									controller.Call("close")
									return
								}
							}
						}()

						return nil
					}),

					"cancel": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
						res.Body.Close()
						return nil
					}),
				}

				readableStreamConstructor := js.Global().Get("ReadableStream")
				readableStream := readableStreamConstructor.New(underlyingSource)
				responseInitObj := map[string]interface{}{
					"status":     http.StatusOK,
					"statusText": http.StatusText(http.StatusOK),
				}
				responseConstructor := js.Global().Get("Response")
				response := responseConstructor.New(readableStream, responseInitObj)
				resolve.Invoke(response)
			}()
			return nil
		})

		promiseConstructor := js.Global().Get("Promise")
		return promiseConstructor.New(handler)
	})
}
