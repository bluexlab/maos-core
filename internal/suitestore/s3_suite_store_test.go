package suitestore_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
	"gitlab.com/navyx/ai/maos/maos-core/internal/suitestore"
	"gitlab.com/navyx/ai/maos/maos-core/internal/testhelper"
)

// MockS3Client is a mock for the S3 client
type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

func (m *MockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

// MockDBAccessor is a mock for the database accessor
type MockDBAccessor struct {
	mock.Mock
}

func (m *MockDBAccessor) Querier() dbsqlc.Querier {
	args := m.Called()
	return args.Get(0).(dbsqlc.Querier)
}

func (m *MockDBAccessor) Source() string {
	args := m.Called()
	return args.String(0)
}

// MockQuerier is a mock for the database querier
type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) ReferenceConfigSuiteUpsert(ctx context.Context, source string, arg *dbsqlc.ReferenceConfigSuiteUpsertParams) (int64, error) {
	args := m.Called(ctx, source, arg)
	return args.Get(0).(int64), args.Error(1)
}

func TestS3SuiteStore_StartBackgroundScanner(t *testing.T) {
	ctx := context.Background()
	dbPool := testhelper.TestDB(ctx, t)
	accessor := dbaccess.New(dbPool)

	mockS3Client := new(MockS3Client)

	store := suitestore.NewS3SuiteStore(
		slog.Default(),
		mockS3Client,
		"test-bucket",
		"test-prefix",
		"test-cluster",
		accessor,
		600*time.Millisecond,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Mock ListObjectsV2 to return two pages of results
	mockS3Client.On("ListObjectsV2", mock.Anything, &s3.ListObjectsV2Input{
		Bucket: aws.String("test-bucket"),
		Prefix: aws.String("test-prefix"),
	}, mock.Anything).Return(&s3.ListObjectsV2Output{
		Contents: []types.Object{
			{Key: aws.String("test-prefix/object1.json"), LastModified: aws.Time(time.Now())},
		},
		IsTruncated:           aws.Bool(true),
		NextContinuationToken: aws.String("token"),
	}, nil).Once()

	mockS3Client.On("ListObjectsV2", mock.Anything, &s3.ListObjectsV2Input{
		Bucket:            aws.String("test-bucket"),
		Prefix:            aws.String("test-prefix"),
		ContinuationToken: aws.String("token"),
	}, mock.Anything).Return(&s3.ListObjectsV2Output{
		Contents: []types.Object{
			{Key: aws.String("test-prefix/object2.json"), LastModified: aws.Time(time.Now())},
		},
		IsTruncated: aws.Bool(false),
	}, nil).Once()

	// Mock GetObject for both objects
	suite1 := suitestore.ReferenceConfigSuite{
		SuiteName: "suite1",
		ConfigSuites: []suitestore.AgentConfig{
			{AgentName: "agent1", Configs: map[string]string{"key": "value1"}},
		},
	}
	suite1Bytes, _ := json.Marshal(suite1)
	mockS3Client.On("GetObject", mock.Anything, &s3.GetObjectInput{
		Bucket: aws.String("test-bucket"),
		Key:    aws.String("test-prefix/object1.json"),
	}, mock.Anything).Return(&s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader(suite1Bytes)),
	}, nil).Once()

	suite2 := suitestore.ReferenceConfigSuite{
		SuiteName: "suite2",
		ConfigSuites: []suitestore.AgentConfig{
			{AgentName: "agent2", Configs: map[string]string{"key": "value2"}},
		},
	}
	suite2Bytes, _ := json.Marshal(suite2)
	mockS3Client.On("GetObject", mock.Anything, &s3.GetObjectInput{
		Bucket: aws.String("test-bucket"),
		Key:    aws.String("test-prefix/object2.json"),
	}, mock.Anything).Return(&s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader(suite2Bytes)),
	}, nil).Once()

	store.StartBackgroundScanner(ctx)

	time.Sleep(1 * time.Second)

	err := store.StopAndWaitForScannerToStop(1 * time.Second)
	require.NoError(t, err)

	mockS3Client.AssertExpectations(t)
	suites, err := accessor.Querier().ReferenceConfigSuiteList(ctx, accessor.Source())
	require.NoError(t, err)
	require.Len(t, suites, 2)
	require.Equal(t, "suite1", suites[0].Name)
	require.Equal(t, "suite2", suites[1].Name)
}

func TestS3SuiteStore_WriteSuite(t *testing.T) {
	ctx := context.Background()
	dbPool := testhelper.TestDB(ctx, t)
	accessor := dbaccess.New(dbPool)
	mockS3Client := new(MockS3Client)

	store := suitestore.NewS3SuiteStore(
		slog.Default(),
		mockS3Client,
		"test-bucket",
		"test-prefix",
		"test-cluster",
		accessor,
		1*time.Second,
	)

	suite := []suitestore.AgentConfig{
		{AgentName: "test-agent", Configs: map[string]string{"key": "value"}},
	}

	expectedContent := suitestore.ReferenceConfigSuite{
		SuiteName:    "test-cluster",
		ConfigSuites: suite,
	}
	expectedJSON, err := json.Marshal(expectedContent)
	require.NoError(t, err)

	mockS3Client.On("PutObject", mock.Anything, &s3.PutObjectInput{
		Bucket: aws.String("test-bucket"),
		Key:    aws.String("test-prefix/test-cluster.json"),
		Body:   bytes.NewReader(expectedJSON),
	}, mock.Anything).Return(&s3.PutObjectOutput{}, nil)

	err = store.WriteSuite(context.Background(), suite)
	require.NoError(t, err)

	mockS3Client.AssertExpectations(t)
}

func TestS3SuiteStore_WriteSuite_Error(t *testing.T) {
	ctx := context.Background()
	dbPool := testhelper.TestDB(ctx, t)
	accessor := dbaccess.New(dbPool)
	mockS3Client := new(MockS3Client)

	store := suitestore.NewS3SuiteStore(
		slog.Default(),
		mockS3Client,
		"test-bucket",
		"test-prefix",
		"", // Empty cluster name to trigger error
		accessor,
		1*time.Second,
	)

	suite := []suitestore.AgentConfig{
		{AgentName: "test-agent", Configs: map[string]string{"key": "value"}},
	}

	err := store.WriteSuite(context.Background(), suite)
	require.Error(t, err)
	require.Contains(t, err.Error(), "clusterName is required")

	mockS3Client.AssertNotCalled(t, "PutObject")
}
