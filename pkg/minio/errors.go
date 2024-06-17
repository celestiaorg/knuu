package minio

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrMinioFailedToStart                       = errors.New("MinioFailedToStart", "failed to create or update Minio deployment")
	ErrMinioFailedToBeReady                     = errors.New("MinioFailedToBeReady", "failed waiting for Minio to be ready")
	ErrMinioFailedToCreateOrUpdateService       = errors.New("MinioFailedToCreateOrUpdateService", "failed to create or update Minio service")
	ErrMinioFailedToBeReadyService              = errors.New("MinioFailedToBeReadyService", "failed waiting for Minio service to be ready")
	ErrMinioFailedToCreatePVC                   = errors.New("MinioFailedToCreatePVC", "failed to create PVC")
	ErrMinioFailedToCreateDeployment            = errors.New("MinioFailedToCreateDeployment", "failed to create Minio deployment")
	ErrMinioFailedToGetDeployment               = errors.New("MinioFailedToGetDeployment", "failed to get Minio deployment")
	ErrMinioFailedToUpdateDeployment            = errors.New("MinioFailedToUpdateDeployment", "failed to update Minio deployment")
	ErrMinioFailedToGetService                  = errors.New("MinioFailedToGetService", "failed to get Minio deployment")
	ErrMinioFailedToGetEndpoint                 = errors.New("MinioFailedToGetEndpoint", "failed to get Minio endpoint")
	ErrMinioFailedToInitializeClient            = errors.New("MinioFailedToInitializeClient", "failed to initialize Minio client")
	ErrMinioFailedToCreateBucket                = errors.New("MinioFailedToCreateBucket", "failed to create bucket")
	ErrMinioFailedToUploadData                  = errors.New("MinioFailedToUploadData", "failed to upload data to Minio")
	ErrMinioFailedToGetPresignedURL             = errors.New("MinioFailedToGetPresignedURL", "failed to generate presigned URL for Minio object")
	ErrMinioFailedToUpdateService               = errors.New("MinioFailedToUpdateService", "failed to update Minio service")
	ErrMinioFailedToFindFileBeforeDeletion      = errors.New("MinioFailedToFindFileBeforeDeletion", "failed to find file in Minio before deletion")
	ErrMinioFailedToDeleteFile                  = errors.New("MinioFailedToDeleteFile", "failed to delete file from Minio")
	ErrMinioFailedToGetMinioEndpoint            = errors.New("MinioFailedToGetMinioEndpoint", "failed to get Minio endpoint")
	ErrMinioFailedToGeneratePresignedURL        = errors.New("MinioFailedToGeneratePresignedURL", "failed to generate presigned URL for Minio object")
	ErrMinioFailedToCreateService               = errors.New("MinioFailedToCreateService", "failed to create Minio service")
	ErrMinioFailedToCheckBucket                 = errors.New("MinioFailedToCheckBucket", "failed to check if bucket exists")
	ErrMinioLoadBalancerIPNotAvailable          = errors.New("MinioLoadBalancerIPNotAvailable", "LoadBalancer IP not available yet")
	ErrMinioFailedToGetNodes                    = errors.New("MinioFailedToGetNodes", "failed to get nodes")
	ErrMinioNoNodesFound                        = errors.New("MinioNoNodesFound", "no nodes found")
	ErrMinioTimeoutWaitingForReady              = errors.New("MinioTimeoutWaitingForReady", "timeout waiting for Minio to be ready")
	ErrMinioNodePortNotSet                      = errors.New("MinioNodePortNotSet", "NodePort for minio service is not set")
	ErrMinioExternalIPsNotSet                   = errors.New("MinioExternalIPsNotSet", "external IPs for minio service are not set")
	ErrMinioTimeoutWaitingForServiceReady       = errors.New("MinioTimeoutWaitingForServiceReady", "timeout waiting for Minio service to be ready")
	ErrMinioFailedToConnect                     = errors.New("MinioFailedToConnect", "failed to connect to %s")
	ErrMinioFailedToParseStorageSize            = errors.New("MinioFailedToParseStorageSize", "failed to parse storage size")
	ErrMinioFailedToListPersistentVolumes       = errors.New("MinioFailedToListPersistentVolumes", "failed to list PersistentVolumes")
	ErrMinioFailedToCreatePersistentVolume      = errors.New("MinioFailedToCreatePersistentVolume", "failed to create PersistentVolume")
	ErrMinioFailedToCreatePersistentVolumeClaim = errors.New("MinioFailedToCreatePersistentVolumeClaim", "failed to create PersistentVolumeClaim")
	ErrMinioClientNotInitialized                = errors.New("MinioClientNotInitialized", "Minio client not initialized")
)
