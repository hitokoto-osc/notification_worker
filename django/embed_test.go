package django

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEmbeddedFS(t *testing.T) {
	assert.NotNil(t, embeddedFS, "embeddedFS should not be nil")
	files, err := getAllFiles(embeddedFS)
	assert.NoError(t, err, "getAllFiles should not return error")
	assert.NotEmpty(t, files, "getAllFiles should not return empty")
	t.Log(files)
}
