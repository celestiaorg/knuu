package minio

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrMinioFailedToStart                       = &Error{Code: "MinioFailedToStart", Message: "failed to create or update Minio deployment"}
	ErrMinioFailedToBeReady                     = &Error{Code: "MinioFailedToBeReady", Message: "failed waiting for Minio to be ready"}
	ErrMinioFailedToCreateOrUpdateService       = &Error{Code: "MinioFailedToCreateOrUpdateService", Message: "failed to create or update Minio service"}
	ErrMinioFailedToBeReadyService              = &Error{Code: "MinioFailedToBeReadyService", Message: "failed waiting for Minio service to be ready"}
	ErrMinioFailedToCreatePVC                   = &Error{Code: "MinioFailedToCreatePVC", Message: "failed to create PVC"}
	ErrMinioFailedToCreateDeployment            = &Error{Code: "MinioFailedToCreateDeployment", Message: "failed to create Minio deployment"}
	ErrMinioFailedToGetDeployment               = &Error{Code: "MinioFailedToGetDeployment", Message: "failed to get Minio deployment"}
	ErrMinioFailedToUpdateDeployment            = &Error{Code: "MinioFailedToUpdateDeployment", Message: "failed to update Minio deployment"}
	ErrMinioFailedToGetService                  = &Error{Code: "MinioFailedToGetService", Message: "failed to get Minio deployment"}
	ErrMinioFailedToGetEndpoint                 = &Error{Code: "MinioFailedToGetEndpoint", Message: "failed to get Minio endpoint"}
	ErrMinioFailedToInitializeClient            = &Error{Code: "MinioFailedToInitializeClient", Message: "failed to initialize Minio client"}
	ErrMinioFailedToCreateBucket                = &Error{Code: "MinioFailedToCreateBucket", Message: "failed to create bucket"}
	ErrMinioFailedToUploadData                  = &Error{Code: "MinioFailedToUploadData", Message: "failed to upload data to Minio"}
	ErrMinioFailedToGetPresignedURL             = &Error{Code: "MinioFailedToGetPresignedURL", Message: "failed to generate presigned URL for Minio object"}
	ErrMinioFailedToUpdateService               = &Error{Code: "MinioFailedToUpdateService", Message: "failed to update Minio service"}
	ErrMinioFailedToFindFileBeforeDeletion      = &Error{Code: "MinioFailedToFindFileBeforeDeletion", Message: "failed to find file in Minio before deletion"}
	ErrMinioFailedToDeleteFile                  = &Error{Code: "MinioFailedToDeleteFile", Message: "failed to delete file from Minio"}
	ErrMinioFailedToGetMinioEndpoint            = &Error{Code: "MinioFailedToGetMinioEndpoint", Message: "failed to get Minio endpoint"}
	ErrMinioFailedToGeneratePresignedURL        = &Error{Code: "MinioFailedToGeneratePresignedURL", Message: "failed to generate presigned URL for Minio object"}
	ErrMinioFailedToCreateService               = &Error{Code: "MinioFailedToCreateService", Message: "failed to create Minio service"}
	ErrMinioFailedToCheckBucket                 = &Error{Code: "MinioFailedToCheckBucket", Message: "failed to check if bucket exists"}
	ErrMinioLoadBalancerIPNotAvailable          = &Error{Code: "MinioLoadBalancerIPNotAvailable", Message: "LoadBalancer IP not available yet"}
	ErrMinioFailedToGetNodes                    = &Error{Code: "MinioFailedToGetNodes", Message: "failed to get nodes"}
	ErrMinioNoNodesFound                        = &Error{Code: "MinioNoNodesFound", Message: "no nodes found"}
	ErrMinioTimeoutWaitingForReady              = &Error{Code: "MinioTimeoutWaitingForReady", Message: "timeout waiting for Minio to be ready"}
	ErrMinioNodePortNotSet                      = &Error{Code: "MinioNodePortNotSet", Message: "NodePort for minio service is not set"}
	ErrMinioExternalIPsNotSet                   = &Error{Code: "MinioExternalIPsNotSet", Message: "external IPs for minio service are not set"}
	ErrMinioTimeoutWaitingForServiceReady       = &Error{Code: "MinioTimeoutWaitingForServiceReady", Message: "timeout waiting for Minio service to be ready"}
	ErrMinioFailedToConnect                     = &Error{Code: "MinioFailedToConnect", Message: "failed to connect to %s"}
	ErrMinioFailedToParseStorageSize            = &Error{Code: "MinioFailedToParseStorageSize", Message: "failed to parse storage size"}
	ErrMinioFailedToListPersistentVolumes       = &Error{Code: "MinioFailedToListPersistentVolumes", Message: "failed to list PersistentVolumes"}
	ErrMinioFailedToCreatePersistentVolume      = &Error{Code: "MinioFailedToCreatePersistentVolume", Message: "failed to create PersistentVolume"}
	ErrMinioFailedToCreatePersistentVolumeClaim = &Error{Code: "MinioFailedToCreatePersistentVolumeClaim", Message: "failed to create PersistentVolumeClaim"}
)
