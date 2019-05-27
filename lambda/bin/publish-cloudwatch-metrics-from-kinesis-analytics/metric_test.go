package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetric(t *testing.T) {
	namespace := "test-namespace"
	name := "test-name"
	at := time.Now()
	value := 4.2
	dimensionOne := "one"
	dimensionTwo := "two"

	t.Run("Unmarshal JSON", func(t *testing.T) {
		v := map[string]interface{}{
			"namespace":    namespace,
			"name":         name,
			"at":           at.Format(DateFormat),
			"value":        value,
			"dimensionOne": dimensionOne,
			"dimensionTwo": dimensionTwo,
		}

		bs, err := json.Marshal(v)
		require.NoError(t, err)

		metric := new(Metric)
		err = json.Unmarshal(bs, metric)
		require.NoError(t, err)

		assert.Equal(t, namespace, metric.Namespace)
		assert.Equal(t, name, metric.Name)
		assert.Equal(t, at.Format(DateFormat), metric.At.Format(DateFormat))
		assert.Equal(t, value, metric.Value)

		require.Len(t, metric.Dimensions, 2)
		assert.Contains(t, metric.Dimensions, "dimensionOne")
		assert.Equal(t, dimensionOne, metric.Dimensions["dimensionOne"])
		assert.Contains(t, metric.Dimensions, "dimensionTwo")
		assert.Equal(t, dimensionTwo, metric.Dimensions["dimensionTwo"])

		t.Run("returns error", func(t *testing.T) {
			t.Run("on invalid JSON", func(t *testing.T) {
				metric := new(Metric)
				err := json.Unmarshal([]byte(`{"invalid":"json`), metric)
				assert.Error(t, err)
			})
			t.Run("on missing namespace", func(t *testing.T) {
				delete(v, "namespace")
				defer func() { v["namespace"] = namespace }()

				err := marshalMetric(t, v)
				assert.Error(t, err)
			})
			t.Run("on missing name", func(t *testing.T) {
				delete(v, "name")
				defer func() { v["name"] = name }()

				err := marshalMetric(t, v)
				assert.Error(t, err)
			})
			t.Run("on missing timestamp", func(t *testing.T) {
				delete(v, "at")
				defer func() { v["at"] = at.Format(DateFormat) }()

				err := marshalMetric(t, v)
				assert.Error(t, err)
			})
			t.Run("on invalid date format", func(t *testing.T) {
				v["at"] = "invalid"
				defer func() { v["at"] = at.Format(DateFormat) }()

				err := marshalMetric(t, v)
				assert.Error(t, err)
			})
			t.Run("on missing value", func(t *testing.T) {
				delete(v, "value")
				defer func() { v["value"] = value }()

				err := marshalMetric(t, v)
				assert.Error(t, err)
			})
			t.Run("on invalid value", func(t *testing.T) {
				v["value"] = "not a float"
				defer func() { v["value"] = value }()

				err := marshalMetric(t, v)
				assert.Error(t, err)
			})
			t.Run("on invalid dimension", func(t *testing.T) {
				v["dimensionOne"] = 4.2
				defer func() { v["dimensionOne"] = dimensionOne }()

				err := marshalMetric(t, v)
				assert.Error(t, err)
			})
			t.Run("on excess dimensions", func(t *testing.T) {
				for i := 0; i < MetricDimensionLimit+1; i++ {
					dimName := fmt.Sprintf("dimension-%d", i)
					v[dimName] = fmt.Sprintf("%d", i)
					defer delete(v, dimName)
				}

				err := marshalMetric(t, v)
				assert.Error(t, err)
			})
		})
	})
}

func marshalMetric(t *testing.T, v map[string]interface{}) error {
	bs, err := json.Marshal(v)
	require.NoError(t, err)

	metric := new(Metric)
	return json.Unmarshal(bs, metric)
}
