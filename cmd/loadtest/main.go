package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func main() {
	client := &http.Client{}
	for i := 1; i <= 4; i++ {
		reqBody := bytes.NewBufferString(`{"password":"newpass"}`)
		req, err := http.NewRequest(http.MethodPatch, "http://localhost:8080/v1/users/1/change-password", reqBody)
		if err != nil {
			panic(err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("attempt %d: status=%d body=%s\n", i, resp.StatusCode, string(body))
	}
}
