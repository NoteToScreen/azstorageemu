# azstorageemu
Cross-platform Azure Blob Storage emulator, written in Go. Tries to be somewhat compatible with the [official emulator](https://docs.microsoft.com/en-us/azure/storage/common/storage-use-emulator).

Portions of this code are based off of the [Azure SDK for Go](https://github.com/Azure/azure-sdk-for-go), which is licensed under the [Apache License](https://github.com/Azure/azure-sdk-for-go/blob/master/LICENSE).

## Warning
This is intended to only be an emulator, for development purposes. You shouldn't actually use it in production, as it is not very secure/scalable. (has a fixed secret key, doesn't have any way of setting permissions on blobs/containers, no metadata, and more)

It's also not a very accurate emulator (the response bodies are not the same as the XML that real Azure storage responds with, mainly because it's not documented very well by Microsoft)

## Supported things
* GET a blob (but all blobs are assumed to be private)
* PUT a blob, block, or block list (except block lists support only includes `Uncommitted` blocks)
* Authentication with an `Authorization` header or a [Service SAS](https://docs.microsoft.com/en-us/rest/api/storageservices/constructing-a-service-sas)

## Unsupported things
* Basically every other Blob API call
* Using an [Account SAS](https://docs.microsoft.com/en-us/rest/api/storageservices/constructing-an-account-sas)
* Queue and Table storage