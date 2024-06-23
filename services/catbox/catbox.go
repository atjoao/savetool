package catbox

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"savetool/config"
	"strings"
	"time"

	"github.com/sqweek/dialog"
)

var (
	baseURL            = "https://catbox.moe/c/"
	baseURLAPI         = "https://catbox.moe/user/api.php"
	userhash           string
	albumID            string
	hostnameStr        string
	downloadZip        string
	downloadLastOpened string
	savePath           string
)

func init() {
	hostname, err := os.Hostname()
	if err != nil {
		panic("Error getting hostname: " + err.Error())
	}

	hostnameStr = hostname
}

func UploadLastFile(uploaded string) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	part, err := writer.CreateFormFile("fileToUpload", "file.lastopened")
	if err != nil {
		fmt.Println("Error creating form field:", err)
		return
	}

	file, err := os.Create("file.lastopened")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	now := time.Now()
	sec := now.Unix()
	secStr := fmt.Sprintf("%d", sec)

	_, err = file.Write([]byte(hostnameStr + "+" + uploaded + "+" + secStr))
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	file.Seek(0, 0)

	_, err = io.Copy(part, file)
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error uploading file:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
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
			choice := dialog.Message("%s", fmt.Sprintf("Files from %s weren't uploaded\nAre u okay with that?", outStr[0])).Title("Warning").YesNo()
			if choice {
				Delete(downloadLastOpened)
				UploadLastFile("false")
			} else {
				fmt.Println("Closing...")
				os.Exit(0)
			}
		}

		if outStr[0] != hostnameStr && outStr[1] == "true" {
			choice := dialog.Message("%s", fmt.Sprintf("Files from %s were uploaded\nDo you want to download them?", outStr[0])).Title("Warning").YesNo()
			if choice {
				DownloadSaveZip()
				Delete(downloadLastOpened)
				UploadLastFile("false")
			} else {
				fmt.Println("Closing...")
				os.Exit(0)
			}
		}

	}

	return 1
}

func addToAlbum(url string) {
	client := &http.Client{}

	urlSplit := strings.Split(url, "/")[3]

	req, err := http.NewRequest("POST", baseURLAPI, strings.NewReader("reqtype=addtoalbum&userhash="+userhash+"&short="+albumID+"&files="+urlSplit))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error adding to album:", err)
		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	fmt.Println(string(body))

}

func Delete(url string) {
	client := &http.Client{}

	urlSplit := strings.Split(url, "/")[3]

	req, err := http.NewRequest("POST", baseURLAPI, strings.NewReader("reqtype=deletefiles&userhash="+userhash+"&short="+albumID+"&files="+urlSplit))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error adding to album:", err)
		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	fmt.Println(string(body))

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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error downloading zip file:", err)
		return
	}

	defer resp.Body.Close()

	out, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	err = os.WriteFile("latest_save.zip", out, 0644)
	if err != nil {
		fmt.Println("Error writing zip file:", err)
		return
	}

	fmt.Println("Zip file downloaded.")
	fmt.Println("Extracting zip file...")

	zip, err := zip.OpenReader("latest_save.zip")
	if err != nil {
		fmt.Println("Error opening zip file:", err)
		return
	}
	defer zip.Close()

	for k, f := range zip.File {
		fmt.Printf("Unzipping %s:\n", f.Name)
		rc, err := f.Open()
		if err != nil {
			fmt.Printf("Impossible to open file n°%d in archive: %s\n", k, err)
			os.Exit(1)
		}

		newFilePath := fmt.Sprintf("%s/%s", savePath, f.Name)

		if f.FileInfo().IsDir() {
			err = os.MkdirAll(newFilePath, 0777)
			if err != nil {
				fmt.Printf("Impossible to MkdirAll: %s\n", err)
				os.Exit(1)
			}
			continue
		}

		uncompressedFile, err := os.Create(newFilePath)
		if err != nil {
			fmt.Printf("Impossible to create uncompressed file: %s\n", err)
			os.Exit(1)
		}
		defer uncompressedFile.Close()

		_, err = io.Copy(uncompressedFile, rc)
		if err != nil {
			fmt.Printf("Impossible to copy file n°%d: %s\n", k, err)
			os.Exit(1)
		}
	}

	os.Remove("latest_save.zip")
}

func Compress() error {
	zipfile, err := os.Create("latest_save.zip")
	if err != nil {
		return fmt.Errorf("error creating zip file: %w", err)
	}
	defer zipfile.Close()

	zipWriter := zip.NewWriter(zipfile)
	defer zipWriter.Close()

	err = filepath.Walk(savePath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking file path: %w", err)
		}

		if filePath == savePath {
			return nil
		}

		relativePath, err := filepath.Rel(savePath, filePath)
		if err != nil {
			return fmt.Errorf("error getting relative path: %w", err)
		}

		if info.IsDir() {
			_, err := zipWriter.Create(relativePath + "/")
			if err != nil {
				return fmt.Errorf("error creating directory in zip: %w", err)
			}
			return nil
		}

		fileToZip, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("error opening file to zip: %w", err)
		}
		defer fileToZip.Close()

		w1, err := zipWriter.Create(relativePath)
		if err != nil {
			return fmt.Errorf("error creating zip writer: %w", err)
		}

		if _, err := io.Copy(w1, fileToZip); err != nil {
			return fmt.Errorf("error copying file content to zip: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func CompressAndUpload() {
	fmt.Println("Compressing files...")
	err := Compress()
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error uploading file: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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
