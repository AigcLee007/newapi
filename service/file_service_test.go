package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeBase64FileDataStripsBase64AndDataURLPrefixes(t *testing.T) {
	const jpegBase64 = "/9j/4AAQSkZJRgABAQAAAQABAAD/2w=="

	tests := []struct {
		name     string
		input    string
		wantData string
		wantMime string
	}{
		{
			name:     "plain base64",
			input:    jpegBase64,
			wantData: jpegBase64,
		},
		{
			name:     "data url",
			input:    "data:image/jpeg;base64," + jpegBase64,
			wantData: jpegBase64,
			wantMime: "image/jpeg",
		},
		{
			name:     "file source identifier style prefix",
			input:    "base64:data:image/jpeg;base64," + jpegBase64,
			wantData: jpegBase64,
			wantMime: "image/jpeg",
		},
		{
			name:     "nested base64 prefix and whitespace",
			input:    " base64:base64:data:image/jpeg;base64,/9j/4AAQ\nSkZJRgABAQAAAQABAAD/2w== ",
			wantData: jpegBase64,
			wantMime: "image/jpeg",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotData, gotMime := normalizeBase64FileData(tc.input)
			require.Equal(t, tc.wantData, gotData)
			require.Equal(t, tc.wantMime, gotMime)
		})
	}
}

func TestLoadFromBase64AcceptsFileSourceIdentifierStyleDataURL(t *testing.T) {
	const pngBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAAAAAA6fptVAAAACklEQVR4nGNgAAAAAgABSK+kcQAAAABJRU5ErkJggg=="

	cached, err := loadFromBase64("base64:data:image/png;base64,"+pngBase64, "")
	require.NoError(t, err)
	require.Equal(t, "image/png", cached.MimeType)

	got, err := cached.GetBase64Data()
	require.NoError(t, err)
	require.Equal(t, pngBase64, got)
}

func TestLoadFromBase64AcceptsURLSafeImageBase64(t *testing.T) {
	const standardJPEGBase64 = "/9j/4AAQSkZJRgABAQAAAQABAAD/2w=="
	const urlSafeJPEGBase64 = "_9j_4AAQSkZJRgABAQAAAQABAAD_2w=="

	cached, err := loadFromBase64("base64:data:image/jpeg;base64,"+urlSafeJPEGBase64, "")
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", cached.MimeType)

	got, err := cached.GetBase64Data()
	require.NoError(t, err)
	require.Equal(t, standardJPEGBase64, got)
}
