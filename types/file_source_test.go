package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFileSourceFromDataNormalizesDataURLBase64(t *testing.T) {
	source := NewFileSourceFromData("base64:data:image/jpeg;base64,_9j_4AAQ SkZJRgABAQAAAQABAAD_2w", "")

	base64Source, ok := source.(*Base64Source)
	require.True(t, ok)
	require.Equal(t, "image/jpeg", base64Source.MimeType)
	require.Equal(t, "/9j/4AAQSkZJRgABAQAAAQABAAD/2w=", base64Source.Base64Data)
	require.NotContains(t, base64Source.GetIdentifier(), "data:image")
}

func TestNewFileSourceFromDataPreservesProvidedMimeType(t *testing.T) {
	source := NewFileSourceFromData("data:image/jpeg;base64,AAAA", "image/png")

	base64Source, ok := source.(*Base64Source)
	require.True(t, ok)
	require.Equal(t, "image/png", base64Source.MimeType)
	require.Equal(t, "AAAA", base64Source.Base64Data)
}
