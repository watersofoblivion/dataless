package amzmock

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/stretchr/testify/mock"
)

type CloudWatch struct {
	mock.Mock
	cloudwatchiface.CloudWatchAPI
}

func (mock *CloudWatch) PutMetricDataWithContext(ctx aws.Context, data *cloudwatch.PutMetricDataInput, options ...request.Option) (*cloudwatch.PutMetricDataOutput, error) {
	args := mock.Called(ctx, data)
	if output := args.Get(0); output != nil {
		return output.(*cloudwatch.PutMetricDataOutput), args.Error(1)
	}
	return nil, args.Error(1)
}
