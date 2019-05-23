package api

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
)

type BatchWriteResponse struct {
	FailedCount int                 `json:"failed_count"`
	Records     []*BatchWriteRecord `json:"records"`
}

func NewBatchWriteResponse(n int, output *firehose.PutRecordBatchOutput) *BatchWriteResponse {
	batch := &BatchWriteResponse{Records: make([]*BatchWriteRecord, n)}

	for i, record := range output.RequestResponses {
		batch.Records[i] = new(BatchWriteRecord)

		if record.ErrorCode != nil && record.ErrorMessage != nil {
			batch.FailedCount++

			batch.Records[i] = &BatchWriteRecord{
				Failed:       true,
				ErrorCode:    aws.StringValue(record.ErrorCode),
				ErrorMessage: aws.StringValue(record.ErrorMessage),
			}
		}
	}

	return batch
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
