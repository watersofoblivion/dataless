package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// DateFormat of the timestamps in the data
const DateFormat = "2006-01-02 15:04:05.000"

// TimeZone the timestamps are in
var TimeZone = time.FixedZone("UTC", 0)

// Metric is a metric to publish to CloudWatch.
type Metric struct {
	Namespace  string            `json:"namespace"`
	Name       string            `json:"name"`
	At         time.Time         `json:"at"`
	Value      float64           `json:"value"`
	Dimensions map[string]string `json:"-"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.  This validates the
// metric as it unmarshals it and returns an error on an invalid metric.
//
// The "namespace", "name", "at", and "value" fields are deserialized into the
// matching fields on the object.  "at" must match the format specified in the
// DateFormat constant.  If the fields are blank or invalid, this method returns
// an error.
//
// All other fields are used as dimensions and expected to have string values.
// If there are more than MetricDimensionLimit dimensions or dimension values
// are blank, this method returns an error.
func (metric *Metric) UnmarshalJSON(bs []byte) error {
	var err error

	v := make(map[string]interface{})
	json.Unmarshal(bs, &v)

	var ok bool

	if metric.Namespace, ok = v["namespace"].(string); !ok || metric.Namespace == "" {
		return fmt.Errorf("metric namespace not given")
	}

	if metric.Name, ok = v["name"].(string); !ok || metric.Name == "" {
		return fmt.Errorf("metric name not given")
	}

	timestamp, ok := v["at"].(string)
	if !ok || timestamp == "" {
		return fmt.Errorf("metric timestamp not given")
	}

	metric.At, err = time.ParseInLocation(DateFormat, timestamp, TimeZone)
	if err != nil {
		return err
	}

	if metric.Value, ok = v["value"].(float64); !ok {
		return fmt.Errorf("metric value not given")
	}

	delete(v, "namespace")
	delete(v, "name")
	delete(v, "at")
	delete(v, "value")

	metric.Dimensions = make(map[string]string)
	for k, v := range v {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("expected dimension %q to be a string, found %T", k, v)
		}
		if s == "" {
			return fmt.Errorf("dimension %q value not given", k)
		}
		metric.Dimensions[k] = s
	}

	if numDimensions := len(metric.Dimensions); numDimensions >= MetricDimensionLimit {
		return fmt.Errorf("%d dimensions on metric %q %q is greater than the limit of %d dimensions", numDimensions, metric.Namespace, metric.Name, MetricDimensionLimit)
	}

	return nil
}
