package helper

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func Unzip(zipname string, savePath string) error {
	zip, err := zip.OpenReader(zipname)
	if err != nil {
		fmt.Println("Error opening zip file:", err)
		return err
	}

	defer zip.Close()

	for k, f := range zip.File {
		fmt.Printf("Unzipping %s:\n", f.Name)
		rc, err := f.Open()
		if err != nil {
			fmt.Printf("Impossible to open file n°%d in archive: %s\n", k, err)
			return err
		}

		newFilePath := fmt.Sprintf("%s/%s", savePath, f.Name)

		if f.FileInfo().IsDir() {
			err = os.MkdirAll(newFilePath, os.ModeAppend)
			if err != nil {
				fmt.Printf("Impossible to MkdirAll: %s\n", err)
				return err
			}
			continue
		}

		uncompressedFile, err := os.Create(newFilePath)
		if err != nil {
			fmt.Printf("Impossible to create uncompressed file: %s\n", err)
			return err
		}

		defer uncompressedFile.Close()

		_, err = io.Copy(uncompressedFile, rc)
		if err != nil {
			fmt.Printf("Impossible to copy file n°%d: %s\n", k, err)
			return err
		}
	}

	return nil
}

func Compress(zipname string, savePath string, keepSave bool) error {
	zipfile, err := os.Create(zipname)
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

	if keepSave {
		backupPath := "gamesaves/"
		err = os.MkdirAll(filepath.Dir(backupPath), 0755)
		if err != nil {
			return fmt.Errorf("error creating backup directory: %w", err)
		}

		sourceFile, err := os.Open(zipname)
		if err != nil {
			return fmt.Errorf("error opening source zip file: %w", err)
		}

		defer sourceFile.Close()

		destFile, err := os.Create(fmt.Sprintf("gamesaves/%d.zip", time.Now().Unix()))
		if err != nil {
			return fmt.Errorf("error creating backup zip file: %w", err)
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, sourceFile)
		if err != nil {
			return fmt.Errorf("error copying zip file to backup: %w", err)
		}
	}

	if err != nil {
		return err
	}

	return nil
}
