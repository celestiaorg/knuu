package kaniko

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTarGz(t *testing.T) {
	testDir := t.TempDir()
	expectedFiles := map[string]*struct{}{
		"file1.txt":        nil,
		"file2.txt":        nil,
		"subdir/file3.txt": nil,
	}
	for file := range expectedFiles {
		filePath := filepath.Join(testDir, file)
		err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		require.NoError(t, err, "Failed to create directory: %s", filepath.Dir(filePath))

		err = os.WriteFile(filePath, []byte(file), os.ModePerm)
		require.NoError(t, err, "Failed to create file: %s", filePath)
	}

	archiveBytes, err := createTarGz(testDir)
	require.NoError(t, err, "createTarGz failed")

	// Verify the tar.gz archive
	gzipReader, err := gzip.NewReader(bytes.NewBuffer(archiveBytes))
	require.NoError(t, err, "gzip.NewReader failed")
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	expectedFiles["."] = nil      // Expecting the root directory
	expectedFiles["subdir"] = nil // Expecting the subdirectory

	for i := 0; i < len(expectedFiles); i++ {
		header, err := tarReader.Next()
		require.NoError(t, err, "tarReader.Next failed")

		_, ok := expectedFiles[header.Name]
		require.True(t, ok, "Unexpected file: %s", header.Name)

		if header.Typeflag == tar.TypeDir {
			continue
		}

		expectedFilePath := filepath.Join(testDir, header.Name)
		expectedContent, err := os.ReadFile(expectedFilePath)
		require.NoError(t, err, "Failed to read file: %s", expectedFilePath)

		actualContent := make([]byte, header.Size)
		_, err = io.ReadFull(tarReader, actualContent)
		require.NoError(t, err, "Failed to read content from archive")

		assert.EqualValues(t, expectedContent, actualContent, "Content mismatch for file: %s", expectedFilePath)
	}
}
