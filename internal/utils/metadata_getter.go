package utils

import (
	"fmt"
	"io"
	"lunabox/internal/models"
	"net/http"
	"time"
)

// Getter 获取元数据
type Getter interface {
	FetchMetadata(id string, token string) (models.Game, error)

	FetchMetadataByName(name string, token string) (models.Game, error)
}

func getResp(client http.Client, url string, authorization string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	req.Header.Set("User-Agent", "Saramanda9988/LunaBox/1.3.2 (desktop) (https://github.com/Saramanda9988/LunaBox)")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	// defer func(Body io.ReadCloser) {
	// 	err := Body.Close()
	// 	if err != nil {
	// 		log.Warnf("Error closing response body: %v", err)
	// 	}
	// }(resp.Body)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bangumi API returned status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}
	return resp, nil
}

func NewVNDBInfoGetter() *VNDBInfoGetter {
	return &VNDBInfoGetter{
		client:  &http.Client{},
		timeout: 10 * time.Second,
	}
}

func NewDmmInfoGetter() *DmmInfoGetter {
	return &DmmInfoGetter{
		client:  &http.Client{},
		timeout: 10 * time.Second,
	}
}

func NewDlsiteInfoGetter() *DlsiteInfoGetter {
	return &DlsiteInfoGetter{
		client:  &http.Client{},
		timeout: 10 * time.Second,
	}
}

func NewGetchuInfoGetter() *GetchuInfoGetter {
	return &GetchuInfoGetter{
		client:  &http.Client{},
		timeout: 10 * time.Second,
	}
}
