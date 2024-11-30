package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/google/uuid"
)

func getStorageAccountDetails() (*azblob.Client, error) {
	accountName, ok := os.LookupEnv("AZURE_STORAGE_ACCOUNT_NAME")
	if !ok {
		panic("AZURE_STORAGE_ACCOUNT_NAME could not be found")
	}

	accountKey, ok := os.LookupEnv("AZURE_STORAGE_PRIMARY_ACCOUNT_KEY")
	if !ok {
		panic("AZURE_STORAGE_PRIMARY_ACCOUNT_KEY could not be found")
	}

	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return &azblob.Client{}, err
	}

	storageAccountUrl := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)

	client, err := azblob.NewClientWithSharedKeyCredential(storageAccountUrl, cred, nil)
	if err != nil {
		return &azblob.Client{}, err
	}

	return client, err
}

func writeToExternalStorage(uploadedFileBytes []byte, originalFilename string, contentType string) (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	filename := uuid.String() + filepath.Ext(originalFilename)
	savedPath := filepath.Join("uploads", filename)

	// err = os.WriteFile(savedPath, uploadedFileBytes, 0644)
	// if err != nil {
	// 	return err
	// }

	containerName, ok := os.LookupEnv("AZURE_STORAGE_CONTAINER_NAME")
	if !ok {
		panic("AZURE_STORAGE_CONTAINER_NAME could not be found")
	}

	client, err := getStorageAccountDetails()
	if err != nil {
		return "", err
	}

	_, err = client.UploadBuffer(context.TODO(), containerName, savedPath, uploadedFileBytes, &azblob.UploadBufferOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: &contentType,
		},
	})
	if err != nil {
		return "", err
	}

	return savedPath, nil
}

func readFromExternalStorage(savedPath string) (bytes.Buffer, error) {
	downloadedData := bytes.Buffer{}
	containerName, ok := os.LookupEnv("AZURE_STORAGE_CONTAINER_NAME")
	if !ok {
		panic("AZURE_STORAGE_CONTAINER_NAME could not be found")
	}

	client, err := getStorageAccountDetails()
	if err != nil {
		return downloadedData, err
	}

	get, err := client.DownloadStream(context.TODO(), containerName, savedPath, nil)
	if err != nil {
		return downloadedData, err
	}

	retryReader := get.NewRetryReader(context.TODO(), &azblob.RetryReaderOptions{})
	_, err = downloadedData.ReadFrom(retryReader)
	if err != nil {
		return downloadedData, err
	}

	err = retryReader.Close()
	if err != nil {
		return downloadedData, err
	}

	return downloadedData, nil
}
