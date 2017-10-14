package main

type Config struct {
	BlobStorage BlobStorageConfig
}

type BlobStorageConfig struct {
	Port int
}
