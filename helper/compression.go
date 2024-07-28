package helper

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Unzip(zipname string, savePath string) error {
	zipReader, err := zip.OpenReader(zipname)
	if err != nil {
		return fmt.Errorf("error opening zip file: %w", err)
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		normalizedFileName := filepath.FromSlash(f.Name)
		newFilePath := filepath.Join(savePath, normalizedFileName)

		if f.FileInfo().IsDir() {
			err = os.MkdirAll(newFilePath, os.ModePerm)
			if err != nil {
				return fmt.Errorf("error creating directory: %w", err)
			}
			continue
		}

		if err = os.MkdirAll(filepath.Dir(newFilePath), os.ModePerm); err != nil {
			return fmt.Errorf("error creating directory for file: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("error opening file in zip: %w", err)
		}
		defer rc.Close()

		uncompressedFile, err := os.Create(newFilePath)
		if err != nil {
			return fmt.Errorf("error creating uncompressed file: %w", err)
		}
		defer uncompressedFile.Close()

		_, err = io.Copy(uncompressedFile, rc)
		if err != nil {
			return fmt.Errorf("error copying file content: %w", err)
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

		relativePath = filepath.ToSlash(relativePath)

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
