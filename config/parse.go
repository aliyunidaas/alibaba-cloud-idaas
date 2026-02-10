package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
	"github.com/pkg/errors"
)

func CreateCloudCredentialConfig(configFilename string) error {
	if configFilename == "" {
		return errors.New("configFilename is empty")
	}
	if _, err := os.Stat(configFilename); err == nil {
		return errors.New("config file already exists")
	}
	config := &CloudCredentialConfig{
		Version: Version1,
		Profile: map[string]*CloudStsConfig{},
	}
	configBytes, marshalErr := json.Marshal(config)
	if marshalErr != nil {
		return errors.Wrap(marshalErr, "failed to marshal config")
	}
	configFile, createErr := os.Create(configFilename)
	if createErr != nil {
		return errors.Wrap(createErr, "failed to create config file")
	}
	defer configFile.Close()
	_, writeErr := configFile.Write(configBytes)
	if writeErr != nil {
		return errors.Wrap(writeErr, "failed to write config file")
	}
	return nil
}

func LoadCloudCredentialConfig(configFilename string) (*CloudCredentialConfig, error) {
	if configFilename == "" {
		var err error
		configFilename, err = GetDefaultCloudCredentialConfigFile()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get default config file")
		}
	}
	config, err := ReadCloudCredentialConfig(configFilename)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read config file: %s", configFilename)
	}
	if config == nil {
		return nil, fmt.Errorf("config file: %s not found", configFilename)
	}
	return config, nil
}

func ReadCloudCredentialConfig(configFilename string) (*CloudCredentialConfig, error) {
	if configFilename == "" {
		return nil, errors.New("configFilename is empty")
	}
	if _, err := os.Stat(configFilename); os.IsNotExist(err) {
		// config file not exists
		idaaslog.Debug.PrintfLn("Config file does not exist: %s, error: %s", configFilename, err.Error())
		return nil, nil
	}
	configFile, openErr := os.Open(configFilename)
	if openErr != nil {
		return nil, errors.Wrapf(openErr, "failed to open config file: %s", configFilename)
	}
	defer configFile.Close()

	configContent, readAllErr := io.ReadAll(configFile)
	if readAllErr != nil {
		return nil, errors.Wrapf(readAllErr, "failed to read config file: %s", configFilename)
	}

	var config CloudCredentialConfig
	configUnmarshalErr := json.Unmarshal(configContent, &config)
	if configUnmarshalErr != nil {
		return nil, errors.Wrapf(configUnmarshalErr, "failed to unmarshal config file: %s", configFilename)
	}

	// current we only know version1
	if config.Version != Version1 {
		return nil, errors.Errorf("config file version %s is not supported, "+
			"please consider upgrade alibaba-cloud-idaas, get latest version from: %s",
			config.Version, constants.UrlIdaasProduct)
	}

	return &config, nil
}

func GetDefaultCloudCredentialConfigFile() (string, error) {
	defaultConfigFileFromEnv := os.Getenv(constants.EnvConfigFile)
	if defaultConfigFileFromEnv != "" {
		return defaultConfigFileFromEnv, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "failed to get home dir")
	}
	idaaslog.Debug.PrintfLn("Home dir: %s", homeDir)
	configFilename := filepath.Join(homeDir, constants.ConfigRootDir, constants.ConfigFilename)
	idaaslog.Debug.PrintfLn("Load default cloud credential config from: %s", configFilename)
	return configFilename, nil
}
