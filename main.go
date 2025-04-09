package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"syscall/js"
)

type ResponseData struct {
	Data    interface{}            `json:"data"`
	Status  int                    `json:"status"`
	Headers map[string]interface{} `json:"headers"`
	Error   string                 `json:"error,omitempty"`
}

func processJSON(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return js.ValueOf(map[string]interface{}{
			"error": "Missing JSON input",
		})
	}

	jsonStr := args[0].String()

	var data interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": "Failed parse JSON: " + err.Error(),
		})
	}

	processedJSON, err := json.Marshal(data)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": "Failed stringify JSON: " + err.Error(),
		})
	}

	return js.ValueOf(string(processedJSON))
}

func extractFields(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return js.ValueOf(map[string]interface{}{
			"error": "Missing args",
		})
	}

	jsonStr := args[0].String()
	fieldsArr := args[1]

	var data map[string]interface{}

	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": "Failed to parse JSON",
		})
	}

	result := make(map[string]interface{})
	for i := 0; i < fieldsArr.Length(); i++ {
		field := fieldsArr.Index(i).String()
		if value, ok := data[field]; ok {
			result[field] = value
		}
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": "Failed to stringify JSON",
		})
	}

	return js.ValueOf(string(resultJSON))
}

func makeRequest(this js.Value, args []js.Value) interface{} {
	done := make(chan struct{}, 0)

	promise := js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			defer func() {
				done <- struct{}{}
			}()

			if len(args) < 1 {
				errorObj := map[string]interface{}{
					"error": "URL not set",
				}
				reject.Invoke(js.ValueOf(errorObj))
				return
			}

			url := args[0].String()
			method := "GET"
			headers := make(map[string]string)
			var reqBody io.Reader

			if len(args) > 1 && !args[1].IsNull() && !args[1].IsUndefined() {
				config := args[1]

				if methodVal := config.Get("method"); !methodVal.IsUndefined() {
					method = methodVal.String()
				}

				if headersVal := config.Get("headers"); !headersVal.IsUndefined() {
					headerNames := js.Global().Get("Object").Call("keys", headersVal)
					for i := 0; i < headerNames.Length(); i++ {
						name := headerNames.Index(i).String()
						value := headersVal.Get(name).String()
						headers[name] = value
					}
				}

				if bodyVal := config.Get("body"); !bodyVal.IsUndefined() && !bodyVal.IsNull() {
					var body []byte
					contentType := ""

					for k, v := range headers {
						if strings.ToLower(k) == "content-type" {
							contentType = v
							break
						}
					}

					if bodyVal.Type() == js.TypeString {
						body = []byte(bodyVal.String())
						if contentType == "" {
							contentType = "application/json"
						}
					} else if bodyVal.Type() == js.TypeObject {
						jsonStr := js.Global().Get("JSON").Call("stringify", bodyVal).String()
						body = []byte(jsonStr)
						if contentType == "" {
							contentType = "application/json"
						}
					}

					if len(body) > 0 {
						reqBody = bytes.NewBuffer(body)

						if contentType != "" && headers["Content-Type"] == "" {
							headers["Content-Type"] = contentType
						}
					}
				}
			}

			req, err := http.NewRequest(method, url, reqBody)
			if err != nil {
				errorObj := map[string]interface{}{
					"error": "Failed to create request: " + err.Error(),
				}
				reject.Invoke(js.ValueOf(errorObj))
				return
			}

			for name, value := range headers {
				req.Header.Add(name, value)
			}

			client := &http.Client{}
			res, err := client.Do(req)
			if err != nil {
				errorObj := map[string]interface{}{
					"error": "Failed to execute request: " + err.Error(),
				}
				reject.Invoke(js.ValueOf(errorObj))
				return
			}

			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				errorObj := map[string]interface{}{
					"error": "Failed to read response body: " + err.Error(),
				}
				reject.Invoke(js.ValueOf(errorObj))
				return
			}

			respHeaders := make(map[string]interface{})
			for name, values := range res.Header {
				if len(values) == 1 {
					respHeaders[name] = values[0]
				} else {
					respHeaders[name] = values
				}
			}

			responseData := ResponseData{
				Status:  res.StatusCode,
				Headers: respHeaders,
			}

			var jsonData interface{}
			if err := json.Unmarshal(body, &jsonData); err == nil {
				responseData.Data = jsonData
			} else {
				responseData.Data = string(body)
			}

			responseJSON, err := json.Marshal(responseData)
			if err != nil {
				errorObj := map[string]interface{}{
					"error": "Failed to marshal response: " + err.Error(),
				}
				reject.Invoke(js.ValueOf(errorObj))
				return
			}

			responseObj := js.Global().Get("JSON").Call("parse", string(responseJSON))
			resolve.Invoke(responseObj)
		}()
		return nil
	}))

	go func() {
		<-done
	}()
	return promise
}

func main() {
	js.Global().Set("goProcessJSON", js.FuncOf(processJSON))
	js.Global().Set("goExtractFields", js.FuncOf(extractFields))
	js.Global().Set("goMakeRequest", js.FuncOf(makeRequest))

	<-make(chan bool)
}
