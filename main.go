package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	cdn "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdn/v20180606"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tencentCloudSDKErrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"gopkg.in/yaml.v2"
)

// Config represents the structure of the configuration file
type Config struct {
	TencentCloud struct {
		SecretID  string `yaml:"secret_id"`
		SecretKey string `yaml:"secret_key"`
		Region    string `yaml:"region"`
	} `yaml:"tencent_cloud"`
	PurgeConfig struct {
		Paths     []string `yaml:"paths"`
		FlushType string   `yaml:"flush_type"`
		UrlEncode bool     `yaml:"url_encode"`
		Area      string   `yaml:"area"`
	} `yaml:"purge_config"`
}

// loadConfig reads and parses the YAML configuration file
func loadConfig(configPath string) (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", configPath)
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse YAML
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %v", err)
	}

	return &config, nil
}

// validateConfig checks if required configuration fields are present
func validateConfig(config *Config) error {
	if config.TencentCloud.SecretID == "" {
		return errors.New("secret_id is required in configuration")
	}
	if config.TencentCloud.SecretKey == "" {
		return errors.New("secret_key is required in configuration")
	}
	if len(config.PurgeConfig.Paths) == 0 {
		return errors.New("at least one path is required in purge_config.paths")
	}
	if config.PurgeConfig.FlushType == "" {
		return errors.New("flush_type is required in purge_config")
	}
	return nil
}

func main() {
	// Define command line flag for config file path
	var configPath string
	flag.StringVar(&configPath, "c", "config.yaml", "Path to the configuration file")
	flag.Parse()

	// Load configuration from YAML file
	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Validate required configuration fields
	if err := validateConfig(config); err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	// Create credential using values from configuration file
	// Using configuration file approach provides better security than hardcoding credentials
	// and allows for easier environment-specific configurations
	credential := common.NewCredential(
		config.TencentCloud.SecretID,
		config.TencentCloud.SecretKey,
	)

	// Initialize client profile with optional settings
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cdn.tencentcloudapi.com"

	// Create client instance for CDN service
	// Region is now read from configuration file instead of being hardcoded
	client, err := cdn.NewClient(credential, config.TencentCloud.Region, cpf)
	if err != nil {
		fmt.Printf("Error creating CDN client: %v\n", err)
		os.Exit(1)
	}

	// Create request object for path cache purging
	request := cdn.NewPurgePathCacheRequest()

	// Configure request parameters from YAML configuration
	// Paths must include protocol header (http:// or https://)
	request.Paths = common.StringPtrs(config.PurgeConfig.Paths)
	request.FlushType = common.StringPtr(config.PurgeConfig.FlushType)
	request.UrlEncode = common.BoolPtr(config.PurgeConfig.UrlEncode)

	// Area parameter is optional, only set if specified in config
	if config.PurgeConfig.Area != "" {
		request.Area = common.StringPtr(config.PurgeConfig.Area)
	}

	// Execute the API call to purge path cache
	response, err := client.PurgePathCache(request)

	// Handle Tencent Cloud SDK specific errors
	var tencentCloudSDKError *tencentCloudSDKErrors.TencentCloudSDKError
	if errors.As(err, &tencentCloudSDKError) {
		fmt.Printf("API error returned: %s\n", err)
		os.Exit(1)
	}

	// Handle general errors
	if err != nil {
		fmt.Printf("Unexpected error: %v\n", err)
		os.Exit(1)
	}

	// Output response in JSON format
	fmt.Printf("Purge operation completed successfully: %s\n", response.ToJsonString())
}
