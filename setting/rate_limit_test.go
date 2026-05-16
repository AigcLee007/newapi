package setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateModelRequestRateLimitGroupKeepsExistingOnInvalidJSON(t *testing.T) {
	original := ModelRequestRateLimitGroup2JSONString()
	t.Cleanup(func() {
		if err := UpdateModelRequestRateLimitGroupByJSONString(original); err != nil {
			t.Errorf("restore original rate limit group: %v", err)
		}
	})

	require.NoError(t, UpdateModelRequestRateLimitGroupByJSONString(`{"vip":[10,5]}`))

	err := UpdateModelRequestRateLimitGroupByJSONString(`{"vip":[`)
	require.Error(t, err)

	total, success, found := GetGroupRateLimit("vip")
	require.True(t, found)
	require.Equal(t, 10, total)
	require.Equal(t, 5, success)
}
