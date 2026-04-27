package pipeline

import (
	"fmt"
	"os"
)

// savePNG writes PNG bytes to the given file path.
func savePNG(data []byte, path string) error {
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
