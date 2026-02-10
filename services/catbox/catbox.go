package catbox

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"savetool/config"
	"savetool/helper"
	"strings"
	"time"

	"github.com/atjoao/dialog"
)

const (
	baseURL    = "https://catbox.moe/c/"
	baseURLAPI = "https://catbox.moe/user/api.php"
)

var (
	userhash           string
	albumID            string
	hostnameStr        string
	downloadZip        string
	downloadLastOpened string
	savePath           string
	keepSaves          bool
)

func init() {
	hostname, err := os.Hostname()
	if err != nil {
		panic("Error getting hostname: " + err.Error())
	}

	hostnameStr = hostname
}

func requestHandler(req *http.Request) ([]byte, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error doing request(%s) of type %s with error %s\n", resp.Request.URL, resp.Request.Method, err)
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	return body, nil
}

func UploadLastFile(uploaded string) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	part, err := writer.CreateFormFile("fileToUpload", "file.lastopened")
	if err != nil {
		fmt.Println("Error creating form field:", err)
		return
	}

	_, err = io.Copy(part, strings.NewReader(hostnameStr+"+"+uploaded+"+"+fmt.Sprintf("%d", time.Now().Unix())))
	if err != nil {
		fmt.Println("Error copying file content:", err)
		return
	}

	writer.WriteField("reqtype", "fileupload")
	writer.WriteField("userhash", userhash)

	err = writer.Close()
	if err != nil {
		fmt.Println("Error closing writer:", err)
		return
	}

	req, err := http.NewRequest("POST", baseURLAPI, buf)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	body, err := requestHandler(req)
	if err != nil {
		fmt.Println("Error processing request", err)
	}

	if downloadLastOpened != "" {
		Delete(downloadLastOpened)
	}

	downloadLastOpened = string(body)

	addToAlbum(string(body))
}

func Retrieve(cfg *config.CatboxConfig) int {
	userhash = cfg.Userhash
	albumID = cfg.AlbumID
	savePath = cfg.SavePath
	keepSaves = cfg.KeepSaves

	if keepSaves {
		fmt.Println("Creating backup...")
		backupPath := "saves/"
		err := os.MkdirAll(filepath.Dir(backupPath), 0755)
		if err != nil {
			fmt.Println("Error creating backup directory: %w", err)
		}

		err = helper.Compress(fmt.Sprintf("saves/%d.zip", time.Now().Unix()), savePath, keepSaves)
		if err != nil {
			fmt.Println("Error compressing files:", err)
		}
	}

	client := &http.Client{}
	fmt.Println("Retrieving from Catbox...")

	resp, err := http.Get(baseURL + albumID)
	if err != nil {
		fmt.Println("Error retrieving from Catbox:", err)
		return 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return 0
	}

	formattedBody := strings.ReplaceAll(string(body), "\n", "")

	regex := regexp.MustCompile(`<div class="imagelist" style="display: none;">(.*)<\/div>`)
	aRegex := regexp.MustCompile(`href=[\'"](.*?)[\'"]`)
	matches := regex.FindStringSubmatch(formattedBody)
	if len(matches) < 2 {
		fmt.Println("No matches found.")
	} else {
		links := aRegex.FindAllStringSubmatch(matches[1], -1)

		fmt.Println("Links:", links)

		for _, link := range links {
			if strings.Contains(link[1], ".lastopened") {
				fmt.Println("Found link:", link[1])
				downloadLastOpened = link[1]
			} else if strings.Contains(link[1], ".zip") {
				fmt.Println("Found zip link:", link[1])
				downloadZip = link[1]
			}
		}
	}

	if downloadLastOpened != "" {
		fmt.Println("Downloading last opened file...")

		req, err := http.NewRequest("GET", downloadLastOpened, nil)
		if err != nil {
			fmt.Println("Error creating request:", err)
			return 0
		}
		req.Header.Set("User-Agent", "a")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error downloading last opened file:", err)
			return 0
		}
		defer resp.Body.Close()

		out, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return 0
		}

		fmt.Println("Last time executed:", string(out))

		outStr := strings.Split(string(out), "+")
		fmt.Println("Hostname:", outStr[0])
		fmt.Println("Is uploaded: ", outStr[1])

		if outStr[0] == hostnameStr {
			Delete(downloadLastOpened)
			UploadLastFile("false")
			return 1
		}

		if outStr[0] != hostnameStr && outStr[1] == "false" {
			choice := dialog.Message("%s", fmt.Sprintf("Files from %s weren't uploaded\nDo you want to continue?", outStr[0])).Title("Warning").YesNo()
			if choice {
				Delete(downloadLastOpened)
				UploadLastFile("false")
			} else {
				fmt.Println("Closing...")
				os.Exit(0)
			}
		}

		if outStr[0] != hostnameStr && outStr[1] == "true" {
			choice := dialog.Message("%s", fmt.Sprintf("Files from %s were uploaded\nDo you want to download them?\n\nYES = DOWNLOAD CLOUD SAVE\nNO = USE LOCAL SAVES\nCANCEL = CLOSE", outStr[0])).Title("Warning").YesNoCancel()
			/* if choice {
				DownloadSaveZip()
				Delete(downloadLastOpened)
				UploadLastFile("false")
			} else {
				fmt.Println("Closing...")
				os.Exit(0)
			} */

			switch choice {
			case dialog.YesNoCancelYes:
				DownloadSaveZip()
				Delete(downloadLastOpened)
				UploadLastFile("false")
				break
			case dialog.YesNoCancelNo:
				Delete(downloadLastOpened)
				UploadLastFile("false")
				break
			case dialog.YesNoCancelCancel:
				fmt.Println("Closing...")
				os.Exit(0)
				break
			}
		}

	}

	return 1
}

func addToAlbum(url string) {
	urlSplit := strings.Split(url, "/")[3]

	req, err := http.NewRequest("POST", baseURLAPI, strings.NewReader("reqtype=addtoalbum&userhash="+userhash+"&short="+albumID+"&files="+urlSplit))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp []byte
	if resp, err = requestHandler(req); err != nil {
		return
	}

	fmt.Println(string(resp))
}

func Delete(url string) {
	urlSplit := strings.Split(url, "/")[3]

	req, err := http.NewRequest("POST", baseURLAPI, strings.NewReader("reqtype=deletefiles&userhash="+userhash+"&short="+albumID+"&files="+urlSplit))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var resp []byte
	if resp, err = requestHandler(req); err != nil {
		return
	}

	fmt.Println(string(resp))
}

func DownloadSaveZip() {
	if downloadZip == "" {
		fmt.Println("No zip file found.")
		return
	}

	fmt.Println("Downloading zip file...")
	req, err := http.NewRequest("GET", downloadZip, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("User-Agent", "a")

	var resp []byte
	if resp, err = requestHandler(req); err != nil {
		return
	}

	err = os.WriteFile("latest_save.zip", resp, os.ModePerm)
	if err != nil {
		fmt.Println("Error writing zip file:", err)
		return
	}

	fmt.Println("Zip file downloaded.")
	fmt.Println("Extracting zip file...")

	err = helper.Unzip("latest_save.zip", savePath)
	if err != nil {
		fmt.Println("Error decompressing zip file: ", err)
		dialog.Message("%s", "I coudn't decompress the zip file with success\nCancelling game launch!").Title("Warning").Error()
		os.Exit(1)
	}

	os.Remove("latest_save.zip")
}

func CompressAndUpload() {
	fmt.Println("Compressing files...")
	err := helper.Compress("latest_save.zip", savePath, keepSaves)
	if err != nil {
		fmt.Println("Error compressing files:", err)
		return
	}

	fmt.Println("Uploading zip file...")
	err = Upload()
	if err != nil {
		fmt.Println("Error uploading zip file:", err)
		return
	}

	fmt.Println("Zip file uploaded.")
}

func Upload() error {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	part, err := writer.CreateFormFile("fileToUpload", "latest_save.zip")
	if err != nil {
		return fmt.Errorf("error creating form field: %w", err)
	}

	file, err := os.Open("latest_save.zip")
	if err != nil {
		return fmt.Errorf("error opening zip file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("error copying file content: %w", err)
	}

	writer.WriteField("reqtype", "fileupload")
	writer.WriteField("userhash", userhash)

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("error closing writer: %w", err)
	}

	req, err := http.NewRequest("POST", baseURLAPI, buf)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	body, err := requestHandler(req)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	fmt.Println("Upload response:", string(body))
	addToAlbum(string(body))
	if downloadZip != "" {
		if downloadZip != string(body) {
			Delete(downloadZip)
		}
	}

	return nil
}
