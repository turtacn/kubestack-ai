// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package utils provides common helper functions used across the application.
package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileExists checks if a file or directory exists at the given path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	// os.IsNotExist is the most reliable way to check for non-existence.
	return !os.IsNotExist(err)
}

// WriteFileSafe writes data to a file in a safe, atomic manner. It first writes to a
// temporary file in the same directory and then renames it to the final destination.
// This prevents file corruption if the write operation is interrupted.
func WriteFileSafe(path string, data []byte, perm os.FileMode) error {
	// Ensure the target directory exists.
	dir := filepath.Dir(path)
	if !FileExists(dir) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create a temporary file.
	tmpFile, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	// Ensure the temp file is removed if anything goes wrong.
	defer os.Remove(tmpFile.Name())

	// Write data to the temporary file.
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Set the desired permissions on the temporary file.
	if err := os.Chmod(tmpFile.Name(), perm); err != nil {
		return fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	// Atomically rename the temporary file to the final destination.
	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return fmt.Errorf("failed to rename temporary file to destination '%s': %w", path, err)
	}

	return nil
}

// CalculateSHA256 computes the SHA256 checksum of a file, returning it as a hex string.
// This is useful for verifying file integrity after download or transfer.
func CalculateSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file '%s' for checksum: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to copy file content for checksum: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// TODO: Implement compression/decompression utilities using `compress/gzip` or `archive/zip`.
// TODO: Implement file monitoring using a library like `fsnotify`.

//Personal.AI order the ending
