package advertising

import (
	"testing"
)

func TestController(t *testing.T) {
	t.Run("Impressions", func(t *testing.T) {
	})

	t.Run("Clicks", func(t *testing.T) {
	})

	t.Run("Publish To CloudWatch", func(t *testing.T) {

	})

	t.Run("Advertising Info", func(t *testing.T) {
		t.Run("returns", func(t *testing.T) {
			t.Run("Bad Request", func(t *testing.T) {
				t.Run("on invalid", func(t *testing.T) {
					t.Run("ad ID", func(t *testing.T) {})
					t.Run("start date", func(t *testing.T) {})
					t.Run("end date", func(t *testing.T) {})
					t.Run("page", func(t *testing.T) {})
					t.Run("limit", func(t *testing.T) {})
				})
			})

			t.Run("Internal Server Error", func(t *testing.T) {
				t.Run("on SDK error", func(t *testing.T) {})
			})
		})
	})
}
