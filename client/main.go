package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/common"
)

// InitConfig Function that uses viper library to parse configuration parameters.
// Viper is configured to read variables from both environment variables and the
// config file ./config.yaml. Environment variables takes precedence over parameters
// defined in the configuration file. If some of the variables cannot be parsed,
// an error is returned
func InitConfig() (*viper.Viper, error) {
	v := viper.New()

	// Configure viper to read env variables with the CLI_ prefix
	v.AutomaticEnv()
	v.SetEnvPrefix("cli")
	// Use a replacer to replace env variables underscores with points. This let us
	// use nested configurations in the config file and at the same time define
	// env variables for the nested configurations
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Add env variables supported
	v.BindEnv("id")
	v.BindEnv("server", "address")
	v.BindEnv("loop", "period")
	v.BindEnv("loop", "lapse")
	v.BindEnv("log", "level")
	v.BindEnv("bet", "name")
	v.BindEnv("bet", "surname")
	v.BindEnv("bet", "personal_id")
	v.BindEnv("bet", "birth_date")
	v.BindEnv("bet_chunk", "size")
	v.BindEnv("bet_chunk", "dir_data_path")
	v.BindEnv("bet_chunk", "file_name")

	// Try to read configuration from config file. If config file
	// does not exists then ReadInConfig will fail but configuration
	// can be loaded from the environment variables so we shouldn't
	// return an error in that case
	v.SetConfigFile("./config.yaml")
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Configuration could not be read from config file. Using env variables instead")
	}

	// Parse time.Duration variables and return an error if those variables cannot be parsed
	if _, err := time.ParseDuration(v.GetString("loop.lapse")); err != nil {
		return nil, errors.Wrapf(err, "Could not parse CLI_LOOP_LAPSE env var as time.Duration.")
	}

	if _, err := time.ParseDuration(v.GetString("loop.period")); err != nil {
		return nil, errors.Wrapf(err, "Could not parse CLI_LOOP_PERIOD env var as time.Duration.")
	}

	return v, nil
}

// InitLogger Receives the log level to be set in logrus as a string. This method
// parses the string and set the level to the logger. If the level string is not
// valid an error is returned
func InitLogger(logLevel string) error {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return err
	}

	customFormatter := &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   false,
	}
	logrus.SetFormatter(customFormatter)
	logrus.SetLevel(level)
	return nil
}

// PrintConfig Print all the configuration parameters of the program.
// For debugging purposes only
func PrintConfig(v *viper.Viper) {
	logrus.Infof("action: config | result: success | client_id: %d | server_address: %s | loop_lapse: %v | loop_period: %v | log_level: %s | bet_name: %s | bet_surname: %s | bet_personal_id: %s | bet_birth_date: %s | bet_chunk_size: %d | bet_chunk_dir_data_path: %s | bet_chunk_file_name: %s",
		v.GetInt("id"),
		v.GetString("server.address"),
		v.GetDuration("loop.lapse"),
		v.GetDuration("loop.period"),
		v.GetString("log.level"),
		v.GetString("bet.name"),
		v.GetString("bet.surname"),
		v.GetString("bet.personal_id"),
		v.GetString("bet.birth_date"),
		v.GetInt("bet_chunk.size"),
		v.GetString("bet_chunk.dir_data_path"),
		v.GetString("bet_chunk.file_name"),
	)
}

// handleSigterm Receives a channel of os.Signal and a client. It waits for a signal
// and then stops the client loop
func handleSigterm(sigs <-chan os.Signal, client *common.Client) {
	<-sigs
	client.StopClient()
}

func main() {
	v, err := InitConfig()
	if err != nil {
		log.Fatalf("%s", err)
	}

	if err := InitLogger(v.GetString("log.level")); err != nil {
		log.Fatalf("%s", err)
	}

	// Print program config with debugging purposes
	PrintConfig(v)

	clientConfig := common.ClientConfig{
		ServerAddress: v.GetString("server.address"),
		ID:            v.GetInt("id"),
		LoopLapse:     v.GetDuration("loop.lapse"),
		LoopPeriod:    v.GetDuration("loop.period"),
		BetChunkSize:  v.GetInt("bet_chunk.size"),
		DirDataPath:   v.GetString("bet_chunk.dir_data_path"),
		FileDataName:  v.GetString("bet_chunk.file_name"),
	}

	client := common.NewClient(clientConfig)

	// Create a channel to receive OS signals.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	// Handle SIGTERM signal
	go handleSigterm(sigs, client)

	client.StartClientLoop()
}
