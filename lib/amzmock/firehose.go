package amzmock

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
	"github.com/stretchr/testify/mock"
)

type Firehose struct {
	mock.Mock
	firehoseiface.FirehoseAPI
}

func (mock *Firehose) PutRecordBatchWithContext(ctx aws.Context, input *firehose.PutRecordBatchInput, options ...request.Option) (*firehose.PutRecordBatchOutput, error) {
	args := mock.Called(ctx, input)
	if output := args.Get(0); output != nil {
		return output.(*firehose.PutRecordBatchOutput), args.Error(1)
	}
	return nil, args.Error(1)
}
