package basicutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func FindProjectRoot() (string, error) {
	currentPath, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not get current working directory: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(currentPath, "go.mod")); err == nil {
			return currentPath, nil
		}

		nextPath := filepath.Dir(currentPath)
		if nextPath == currentPath {
			return "", errors.New("reached / without finding go.mod")
		}
		currentPath = nextPath
	}
}

func FindInParentDirectories(target, stopFile string) ([]string, error) {
	currentPath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get current working directory: %w", err)
	}

	ret := make([]string, 0, 1)
	for {
		path := filepath.Join(currentPath, target)
		if _, err := os.Stat(path); err == nil {
			ret = append(ret, path)
		}

		if _, err := os.Stat(filepath.Join(currentPath, stopFile)); err == nil {
			if len(ret) == 0 {
				return nil, fmt.Errorf("found stopfile %q without finding %q", stopFile, target)
			}
			return ret, nil
		}

		nextPath := filepath.Dir(currentPath)
		if nextPath == currentPath {
			if len(ret) == 0 {
				return nil, fmt.Errorf("reached / without finding %q", target)
			}
			return ret, nil
		}
		
		currentPath = nextPath
	}
}
