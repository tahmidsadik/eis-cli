package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// ECRClient wraps the AWS ECR client with helper methods
type ECRClient struct {
	client    *ecr.Client
	accountID string
	region    string
	profile   string
}

// NewECRClient creates a new ECR client using the specified AWS profile
func NewECRClient(ctx context.Context, profile, region string) (*ECRClient, error) {
	// Load AWS configuration with the specified profile
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Get account ID using STS
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS account ID: %w", err)
	}

	return &ECRClient{
		client:    ecr.NewFromConfig(cfg),
		accountID: *identity.Account,
		region:    region,
		profile:   profile,
	}, nil
}

// RepositoryExists checks if an ECR repository exists
func (c *ECRClient) RepositoryExists(ctx context.Context, repositoryName string) (bool, *types.Repository, error) {
	input := &ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{repositoryName},
	}

	output, err := c.client.DescribeRepositories(ctx, input)
	if err != nil {
		// Check if error is RepositoryNotFoundException
		var notFoundErr *types.RepositoryNotFoundException
		if errors.As(err, &notFoundErr) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("failed to describe repository: %w", err)
	}

	if len(output.Repositories) > 0 {
		return true, &output.Repositories[0], nil
	}

	return false, nil, nil
}

// CreateRepository creates a new ECR repository
func (c *ECRClient) CreateRepository(ctx context.Context, repositoryName string) (*types.Repository, error) {
	input := &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(repositoryName),
		ImageScanningConfiguration: &types.ImageScanningConfiguration{
			ScanOnPush: true,
		},
		ImageTagMutability: types.ImageTagMutabilityMutable,
	}

	output, err := c.client.CreateRepository(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	return output.Repository, nil
}

// GetRepositoryURI returns the full ECR repository URI
func (c *ECRClient) GetRepositoryURI(repositoryName string) string {
	return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s",
		c.accountID, c.region, repositoryName)
}

// GetConsoleURL returns the AWS Console URL for the repository
func (c *ECRClient) GetConsoleURL(repositoryName string) string {
	return fmt.Sprintf("https://%s.console.aws.amazon.com/ecr/repositories/private/%s/%s",
		c.region, c.accountID, repositoryName)
}

// GetAccountID returns the AWS account ID
func (c *ECRClient) GetAccountID() string {
	return c.accountID
}

// GetRegion returns the AWS region
func (c *ECRClient) GetRegion() string {
	return c.region
}

// GetProfile returns the AWS profile name
func (c *ECRClient) GetProfile() string {
	return c.profile
}
