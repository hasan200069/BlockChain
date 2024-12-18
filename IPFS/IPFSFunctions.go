package IPFS

import (
	"fmt"
	"io"
	"os"

	shell "github.com/ipfs/go-ipfs-api"
)

func UploadFileToIPFS(sh *shell.Shell, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	cid, err := sh.Add(file)
	if err != nil {
		return "", fmt.Errorf("failed to add file to IPFS: %w", err)
	}

	return cid, nil
}
func DownloadFileFromIPFS(sh *shell.Shell, cid string, outputPath string) error {
	reader, err := sh.Cat(cid)
	if err != nil {
		return fmt.Errorf("failed to retrieve file from IPFS: %w", err)
	}
	defer func(reader io.ReadCloser) {
		err := reader.Close()
		if err != nil {

		}
	}(reader)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func(outputFile *os.File) {
		err := outputFile.Close()
		if err != nil {

		}
	}(outputFile)

	_, err = io.Copy(outputFile, reader)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	return nil
}
