package api

import "fmt"

type BatchWriteResponse struct {
	FailedCount int                 `json:"failed_count"`
	Records     []*BatchWriteRecord `json:"records"`
}

type BatchWriteRecord struct {
	Failed       bool   `json:"failed"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func (record *BatchWriteRecord) Error() string {
	if record.Failed {
		return fmt.Sprintf("%s: %s", record.ErrorCode, record.ErrorMessage)
	}

	return ""
}
