package pinner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
)

const baseURL = "https://api.pinata.cloud/pinning/pinFileToIPFS"

type Response struct {
	IpfsHash  string `json:"IpfsHash"`
	PinSize   int64  `json:"PinSize"`
	Timestamp string `json:"Timestamp"`
}

type Error struct {
	Error struct {
		Reason  string `json:"reason"`
		Details string `json:"details"`
	} `json:"error"`
}

type Pinata struct {
	jwt    string
	client *http.Client
}

func NewPinataPinner(jwt string) IPinner {
	return &Pinata{jwt: jwt, client: &http.Client{}}
}

func (p *Pinata) Pin(fileName string, file io.Reader) (string, error) {
	method := "POST"
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	part, err := writer.CreateFormFile("file", filepath.Base("./"+fileName))
	if err != nil {
		return "", fmt.Errorf("failed to create for file: %w", err)
	}

	if _, err = io.Copy(part, file); err != nil {
		return "", fmt.Errorf("copy file: %w", err)
	}

	if err := writer.WriteField("pinataOptions", "{\"cidVersion\": 1}"); err != nil {
		return "", fmt.Errorf("failed to write pinataOptions field: %w", err)
	}

	if err := writer.WriteField("pinataMetadata", fmt.Sprintf("{\"name\": \"%s\"}", fileName)); err != nil {
		return "", fmt.Errorf("failed to write pinataMetadata field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	req, err := http.NewRequest(method, baseURL, payload)
	if err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+p.jwt)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("do pin request: %w", err)
	}

	if res != nil {
		defer res.Body.Close()
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	response := &Response{}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.IpfsHash, nil
}
