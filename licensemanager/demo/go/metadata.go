package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type metadata struct {
	ID     string `json:"id"`
	Vendor struct {
		FolderID string `json:"folderId"`
	}
}

func fetchParamsFromMetadata() (*params, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://169.254.169.254/computeMetadata/v1/instance/?recursive=true", nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var md metadata
	err = json.Unmarshal(body, &md)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &params{
		resourceID: md.ID,
		folderID:   md.Vendor.FolderID,
	}, nil
}
