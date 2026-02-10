package github

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"savetool/config"
	"savetool/helper"
	"strings"
	"time"

	"github.com/atjoao/dialog"
)

const (
	baseAPIURL = "https://api.github.com"
)

var (
	token       string
	repo        string
	branch      string
	gameName    string
	savePath    string
	keepSaves   bool
	hostnameStr string
)

func init() {
	hostname, err := os.Hostname()
	if err != nil {
		panic("Error getting hostname: " + err.Error())
	}
	hostnameStr = hostname
}

type GitHubFileResponse struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	SHA     string `json:"sha"`
	Content string `json:"content"`
	Message string `json:"message"`
}

type GitHubCommitRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	Branch  string `json:"branch"`
	SHA     string `json:"sha,omitempty"`
}

func apiRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	url := baseAPIURL + endpoint
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	return client.Do(req)
}

func getFileSHA(filePath string) (string, error) {
	encodedPath := url.PathEscape(filePath)
	endpoint := fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, encodedPath, branch)
	resp, err := apiRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", nil // File doesn't exist
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var fileResp GitHubFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
		return "", err
	}

	return fileResp.SHA, nil
}

func uploadFile(filePath string, content []byte, message string) error {
	sha, err := getFileSHA(filePath)
	if err != nil {
		fmt.Println("Warning: Could not get SHA for file:", err)
	}

	commitReq := GitHubCommitRequest{
		Message: message,
		Content: base64.StdEncoding.EncodeToString(content),
		Branch:  branch,
	}

	if sha != "" {
		commitReq.SHA = sha
	}

	jsonBody, err := json.Marshal(commitReq)
	if err != nil {
		return fmt.Errorf("error marshaling request: %w", err)
	}

	encodedPath := url.PathEscape(filePath)
	endpoint := fmt.Sprintf("/repos/%s/contents/%s", repo, encodedPath)
	fmt.Println("Uploading file:", filePath)
	resp, err := apiRequest("PUT", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func downloadFile(filePath string) ([]byte, error) {
	encodedPath := url.PathEscape(filePath)
	endpoint := fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, encodedPath, branch)
	resp, err := apiRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil // File doesn't exist
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	var fileResp GitHubFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
		return nil, err
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(fileResp.Content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("error decoding content: %w", err)
	}

	return decoded, nil
}

func getGamePath() string {
	return gameName
}

func UploadLastFile(uploaded string) {
	content := fmt.Sprintf("%s+%s+%d", hostnameStr, uploaded, time.Now().Unix())
	filePath := getGamePath() + "/.lastopened"

	err := uploadFile(filePath, []byte(content), fmt.Sprintf("Update .lastopened for %s", gameName))
	if err != nil {
		fmt.Println("Error uploading .lastopened:", err)
		return
	}

	fmt.Println("Uploaded .lastopened marker")
}

func Retrieve(cfg *config.GitHubConfig) int {
	token = cfg.Token
	repo = cfg.Repo
	branch = cfg.Branch
	gameName = cfg.GameName
	savePath = cfg.SavePath
	keepSaves = cfg.KeepSaves

	fmt.Printf("GitHub Service initialized for game: %s\n", gameName)
	fmt.Printf("Repository: %s, Branch: %s\n", repo, branch)

	if keepSaves {
		fmt.Println("Creating backup...")
		backupPath := "saves/"
		err := os.MkdirAll(filepath.Dir(backupPath), 0755)
		if err != nil {
			fmt.Println("Error creating backup directory:", err)
		}

		err = helper.Compress(fmt.Sprintf("saves/%d.zip", time.Now().Unix()), savePath, keepSaves)
		if err != nil {
			fmt.Println("Error compressing files:", err)
		}
	}

	lastOpenedPath := getGamePath() + "/.lastopened"
	lastOpenedContent, err := downloadFile(lastOpenedPath)
	if err != nil {
		fmt.Println("Error checking .lastopened:", err)
	}

	if lastOpenedContent != nil {
		fmt.Println("Last time executed:", string(lastOpenedContent))

		parts := strings.Split(string(lastOpenedContent), "+")
		if len(parts) >= 2 {
			remoteHostname := parts[0]
			isUploaded := parts[1]

			fmt.Println("Hostname:", remoteHostname)
			fmt.Println("Is uploaded:", isUploaded)

			if remoteHostname == hostnameStr {
				UploadLastFile("false")
				return 1
			}

			if remoteHostname != hostnameStr && isUploaded == "false" {
				choice := dialog.Message("%s", fmt.Sprintf("Files from %s weren't uploaded\nAre you okay with that?", remoteHostname)).Title("Warning").YesNo()
				if choice {
					UploadLastFile("false")
				} else {
					fmt.Println("Closing...")
					os.Exit(0)
				}
			}

			if remoteHostname != hostnameStr && isUploaded == "true" {
				choice := dialog.Message("%s", fmt.Sprintf("Files from %s were uploaded\nDo you want to download them?\n\nYES = DOWNLOAD CLOUD SAVE\nNO = USE LOCAL SAVES\nCANCEL = CLOSE", remoteHostname)).Title("Warning").YesNoCancel()
				/* if choice {
					DownloadSaveZip()
					UploadLastFile("false")
				} else {
					fmt.Println("Closing...")
					os.Exit(0)
				} */

				switch choice {
				case dialog.YesNoCancelYes:
					DownloadSaveZip()
					UploadLastFile("false")
					break
				case dialog.YesNoCancelNo:
					UploadLastFile("false")
					break
				case dialog.YesNoCancelCancel:
					fmt.Println("Closing...")
					os.Exit(0)
					break
				}
			}
		}
	} else {
		UploadLastFile("false")
	}

	return 1
}

func DownloadSaveZip() {
	zipPath := getGamePath() + "/latest_save.zip"
	content, err := downloadFile(zipPath)
	if err != nil {
		fmt.Println("Error downloading save zip:", err)
		return
	}

	if content == nil {
		fmt.Println("No save file found in repository")
		return
	}

	fmt.Println("Downloaded save zip from GitHub")

	err = os.WriteFile("latest_save.zip", content, os.ModePerm)
	if err != nil {
		fmt.Println("Error writing zip file:", err)
		return
	}

	fmt.Println("Extracting zip file...")
	err = helper.Unzip("latest_save.zip", savePath)
	if err != nil {
		fmt.Println("Error decompressing zip file:", err)
		dialog.Message("%s", "I couldn't decompress the zip file with success\nCancelling game launch!").Title("Warning").Error()
		os.Exit(1)
	}

	os.Remove("latest_save.zip")
	fmt.Println("Save files extracted successfully")
}

func CompressAndUpload() {
	fmt.Println("Compressing files...")
	err := helper.Compress("latest_save.zip", savePath, keepSaves)
	if err != nil {
		fmt.Println("Error compressing files:", err)
		return
	}

	fmt.Println("Uploading zip file to GitHub...")
	err = Upload()
	if err != nil {
		fmt.Println("Error uploading zip file:", err)
		return
	}

	fmt.Println("Zip file uploaded successfully")
}

func Upload() error {
	content, err := os.ReadFile("latest_save.zip")
	if err != nil {
		return fmt.Errorf("error reading zip file: %w", err)
	}

	zipPath := getGamePath() + "/latest_save.zip"
	err = uploadFile(zipPath, content, fmt.Sprintf("Update save for %s", gameName))
	if err != nil {
		return err
	}

	// Clean up local temp file
	os.Remove("latest_save.zip")

	return nil
}
