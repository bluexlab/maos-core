package suitestore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess"
	"gitlab.com/navyx/ai/maos/maos-core/dbaccess/dbsqlc"
)

type S3ClientInterface interface {
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// S3SuiteStore implements the SuiteStore interface using AWS S3
type S3SuiteStore struct {
	logger       *slog.Logger
	s3Client     S3ClientInterface
	bucketName   string
	keyPrefix    string
	displayName  string
	db           dbaccess.Accessor
	scanInterval time.Duration
	stopChan     chan struct{}
	stoppedChan  chan struct{} // New field to signal when the scanner has stopped
	lastUpdated  map[string]time.Time
}

// NewS3SuiteStore creates a new S3SuiteStore
func NewS3SuiteStore(logger *slog.Logger, s3Client S3ClientInterface, bucketName, keyPrefix, displayName string, db dbaccess.Accessor, scanInterval time.Duration) *S3SuiteStore {
	return &S3SuiteStore{
		logger:       logger,
		s3Client:     s3Client,
		bucketName:   bucketName,
		keyPrefix:    keyPrefix,
		displayName:  displayName,
		db:           db,
		scanInterval: scanInterval,
		stopChan:     make(chan struct{}),
		stoppedChan:  make(chan struct{}),
		lastUpdated:  make(map[string]time.Time),
	}
}

// StartBackgroundScanner starts the background daemon to scan S3 and update the database
func (s *S3SuiteStore) StartBackgroundScanner(ctx context.Context) {
	go func() {
		s.logger.Info("Starting reference config suite scanner", "scanInterval", s.scanInterval)

		ticker := time.NewTicker(s.scanInterval)
		defer ticker.Stop()
		defer close(s.stoppedChan) // Signal that the scanner has stopped

		for {
			s.logger.Info("Waiting for next scan interval")
			select {
			case <-ticker.C:
				if err := s.scanAndUpdateDatabase(ctx); err != nil {
					s.logger.Error("Failed to scan and update database", "error", err)
				}
			case <-s.stopChan:
				s.logger.Info("Stopping reference config suite scanner due to stop signal")
				return
			case <-ctx.Done():
				s.logger.Info("Stopping reference config suite scanner due to context cancellation")
				return
			}
		}
	}()
}

// StopAndWaitForScannerToStop stops the background scanner and waits for it to fully stop
func (s *S3SuiteStore) StopAndWaitForScannerToStop(timeout time.Duration) error {
	close(s.stopChan)

	select {
	case <-s.stoppedChan:
		s.logger.Info("Reference config suite scanner stopped")
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for scanner to stop after %v", timeout)
	}
}

func (s *S3SuiteStore) scanAndUpdateDatabase(ctx context.Context) error {
	s.logger.Info("Scanning and updating reference config suites", "bucketName", s.bucketName, "keyPrefix", s.keyPrefix, "displayName", s.displayName)

	paginator := s3.NewListObjectsV2Paginator(s.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(s.keyPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			s.logger.Error("Failed to get next page", "error", err)
			return err
		}

		for _, object := range page.Contents {
			lastUpdated, exists := s.lastUpdated[*object.Key]

			if exists && lastUpdated.Equal(object.LastModified.Local()) {
				s.logger.Debug("Skipping unchanged suite", "key", *object.Key)
				continue
			}

			suite, err := s.readSingleSuite(ctx, *object.Key)
			if err != nil {
				s.logger.Error("Failed to read suite from S3", "error", err, "key", *object.Key)
				return err
			}

			if suite.SuiteName == s.displayName {
				s.logger.Debug("Skipping self suite")
				continue
			}

			ConfigSuitesBytes, err := json.Marshal(suite.ConfigSuites)
			if err != nil {
				s.logger.Error("Failed to marshal suite", "error", err, "suiteName", suite.SuiteName)
				return err
			}

			s.logger.Info("Upserting suite in database", "suiteName", suite.SuiteName)
			_, err = s.db.Querier().ReferenceConfigSuiteUpsert(ctx, s.db.Source(), &dbsqlc.ReferenceConfigSuiteUpsertParams{
				Name:              suite.SuiteName,
				ConfigSuitesBytes: ConfigSuitesBytes,
			})
			if err != nil {
				s.logger.Error("Failed to upsert suite in database", "error", err, "suiteName", suite.SuiteName)
				return err
			}

			s.lastUpdated[*object.Key] = object.LastModified.Local()
		}
	}

	return nil
}

// readSingleSuite reads a single config suite from S3
func (s *S3SuiteStore) readSingleSuite(ctx context.Context, key string) (ReferenceConfigSuite, error) {
	result, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		s.logger.Error("Failed to read suite from S3", "error", err)
		return ReferenceConfigSuite{}, err
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		s.logger.Error("Failed to read suite from S3", "error", err)
		return ReferenceConfigSuite{}, err
	}

	var suite ReferenceConfigSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		s.logger.Error("Failed to unmarshal suite", "error", err)
		return ReferenceConfigSuite{}, err
	}

	return suite, nil
}

// WriteSuite writes the given config suite to S3
func (s *S3SuiteStore) WriteSuite(ctx context.Context, suite []AgentConfig) error {
	s.logger.Info("Writing suite to S3", "bucketName", s.bucketName, "keyPrefix", s.keyPrefix, "displayName", s.displayName, "suite", suite)

	output := ReferenceConfigSuite{
		SuiteName:    s.displayName,
		ConfigSuites: suite,
	}
	data, err := json.Marshal(output)
	if err != nil {
		s.logger.Error("Failed to marshal suite", "error", err)
		return err
	}

	if s.displayName == "" {
		s.logger.Error("displayName is required")
		return fmt.Errorf("displayName is required")
	}

	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(filepath.Join(s.keyPrefix, s.displayName+".json")),
		Body:   bytes.NewReader(data),
	})

	if err != nil {
		s.logger.Error("Failed to write suite to S3", "error", err)
	}

	return err
}
