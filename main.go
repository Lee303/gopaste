package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

var url = "https://p.lee.io"

type apiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type syntax struct {
	Label   string `json:"label"`
	Syntax  string `json:"syntax"`
	Default bool   `json:"default"`
}

type expires struct {
	Label   string `json:"label"`
	Expires string `json:"expires"`
	Default bool   `json:"default"`
}

type paste struct {
	Content string `json:"content"`
	Syntax  string `json:"syntax"`
	Expires string `json:"expires"`
}

func main() {

	var syntaxFlag = flag.String("syntax", "", "(Optional) Syntax to use for paste")
	var expiresFlag = flag.String("expires", "", "(Optional) Expire type to use for paste (Optional)")
	var fileFlag = flag.String("file", "", "(Optional) File to read from. Stdin is used if not provided")
	var getUUIDFlag = flag.String("get", "", "UUID of paste to retrieve")
	var getSyntaxFlag = flag.Bool("getsyntax", false, "Retrieve supported syntax")
	var getExpiresFlag = flag.Bool("getexpires", false, "Retrieve supported expire types")

	flag.Parse()

	var err error

	if *getUUIDFlag != "" {
		paste, err := getPaste(*getUUIDFlag)
		failOnError(err, "failed to retrieve paste")
		fmt.Printf("%s\n", paste.Content)

		os.Exit(0)
	}

	if *getSyntaxFlag {
		supportedSyntax, err := getSyntax()
		failOnError(err, "failed to retrieve syntax")
		fmt.Printf("%+v\n", supportedSyntax)

		os.Exit(0)
	}

	if *getExpiresFlag {
		supportedExpires, err := getExpires()
		failOnError(err, "failed to retrieve expire types")
		fmt.Printf("%+v\n", supportedExpires)

		os.Exit(0)
	}

	var data []byte
	if *fileFlag == "" {
		data, err = ioutil.ReadAll(os.Stdin)
		failOnError(err, "Failed to read from stdin")
	} else {
		data, err = ioutil.ReadFile(*fileFlag)
		failOnError(err, "Failed to read from file")
	}

	uuid, err := addPaste(string(data), *syntaxFlag, *expiresFlag)
	failOnError(err, "Failed to add paste")

	fmt.Printf("%s/%s\n", url, uuid)
}

func failOnError(err error, message string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", message, err)
		os.Exit(1)
	}
}

func addPaste(content string, syntax string, expires string) (string, error) {

	payload := paste{
		Content: content,
		Syntax:  syntax,
		Expires: expires,
	}

	resp, err := doCall(http.MethodPost, "/v1/paste", payload)
	if err != nil {
		return "", err
	}

	s, _ := resp.(map[string]interface{})
	return s["uuid"].(string), nil
}

func getPaste(uuid string) (*paste, error) {
	resp, err := doCall(http.MethodGet, fmt.Sprintf("/v1/paste/%s", uuid), nil)
	if err != nil {
		return nil, err
	}

	s, _ := resp.(map[string]interface{})
	return &paste{
		Content: s["content"].(string),
		Syntax:  s["syntax"].(string),
		Expires: s["expires"].(string),
	}, nil
}

func getExpires() (*[]expires, error) {
	resp, err := doCall(http.MethodGet, "/v1/expires", nil)
	if err != nil {
		return nil, err
	}

	var expiresArr []expires
	for _, v := range resp.([]interface{}) {
		s, _ := v.(map[string]interface{})
		expiresArr = append(expiresArr, expires{
			Label:   s["label"].(string),
			Expires: s["expires"].(string),
			Default: s["default"].(bool),
		})
	}

	return &expiresArr, nil
}

func getSyntax() (*[]syntax, error) {
	resp, err := doCall(http.MethodGet, "/v1/syntax", nil)
	if err != nil {
		return nil, err
	}

	var syntaxArr []syntax
	for _, v := range resp.([]interface{}) {
		s, _ := v.(map[string]interface{})
		syntaxArr = append(syntaxArr, syntax{
			Label:   s["label"].(string),
			Syntax:  s["syntax"].(string),
			Default: s["default"].(bool),
		})
	}

	return &syntaxArr, nil
}

func doCall(method string, route string, payload interface{}) (interface{}, error) {
	resp := &apiResponse{}

	client := http.Client{Timeout: time.Second * 30}

	var payloadBuffer bytes.Buffer
	if payload != nil {
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, errors.New("Invalid payload")
		}
		payloadBuffer = *bytes.NewBuffer(payloadJSON)
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s/api/%s", url, strings.Trim(route, "/")), &payloadBuffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err)
	}

	r, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %s", err)
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err)
	}

	json.Unmarshal(bodyBytes, resp)

	if r.StatusCode != 200 {
		errStr := ""
		if resp.Data != nil {
			errStr, _ = resp.Data.(string)
		}
		return nil, fmt.Errorf("unsuccessful request: statuscode=%d data=%s", r.StatusCode, errStr)
	}

	if resp.Success != true {
		return nil, fmt.Errorf("api error: %s", resp.Data)
	}

	return resp.Data, nil
}
