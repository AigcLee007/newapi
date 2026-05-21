package gemini

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseInlineImageURLForGeminiNormalizesDataURL(t *testing.T) {
	base64Data, mimeType, ok, err := parseInlineImageURLForGemini("base64:data:image/jpeg;base64,_9j_4AAQ SkZJRgABAQAAAQABAAD_2w", "")

	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "image/jpeg", mimeType)
	require.Equal(t, "/9j/4AAQSkZJRgABAQAAAQABAAD/2w==", base64Data)
}

func TestParseInlineImageURLForGeminiIgnoresRemoteURL(t *testing.T) {
	_, _, ok, err := parseInlineImageURLForGemini("https://example.com/image.jpg", "")

	require.NoError(t, err)
	require.False(t, ok)
}
