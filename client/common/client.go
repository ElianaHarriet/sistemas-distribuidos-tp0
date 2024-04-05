package common

import (
	"encoding/csv"
	"net"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            int
	ServerAddress string
	LoopLapse     time.Duration
	LoopPeriod    time.Duration
	BetChunkSize  int
	DirDataPath   string
	FileDataName  string
}

// Client Entity that encapsulates how
type Client struct {
	config       ClientConfig
	conn         net.Conn
	data_file    *os.File
	stop_chan    chan bool
	personal_ids map[int]bool
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config:       config,
		conn:         nil,
		data_file:    nil,
		stop_chan:    make(chan bool),
		personal_ids: make(map[int]bool),
	}
	return client
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	err := c.openFile()
	defer c.data_file.Close()
	if err != nil {
		log.Fatalf("action: open_file | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return
	}

	reader := csv.NewReader(c.data_file)

	err = c.createClientSocket()
	defer c.conn.Close()
	if err != nil {
		return
	}

	for {
		bets, end, shouldReturn1 := c.readBets(reader)
		if shouldReturn1 {
			return
		}
		if len(bets) == 0 {
			break
		}
		shouldReturn2 := sendBets(c, bets)
		if shouldReturn2 {
			return
		}
		if end {
			break
		}

		result, err := c.receiveMessage()
		if err != nil {
			return
		}
		if len(result) == 0 {
			log.Warnf("action: receive_message | result: fail | client_id: %v | error: empty message",
				c.config.ID,
			)
			c.StopClient()
			return
		}

		_, shouldReturn, _ := c.manageServerResponse(result)
		if shouldReturn {
			c.StopClient()
			return
		}

		shouldReturn = c.waitOrStop()
		if shouldReturn {
			return
		}
	}

	wait := true
	for wait {
		wait, err = c.askResults()
		if err != nil {
			return
		}
		// Wait a time between sending one message and the next one
		shouldReturn := c.waitOrStop()
		if shouldReturn {
			return
		}
	}

	err = c.closeClientSocket()
	if err != nil {
		return
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}

// StopClient Stops the client loop
func (c *Client) StopClient() {
	defer func() {
		c.stop_chan <- true
		log.Infof("action: stop_loop | result: success | client_id: %v",
			c.config.ID,
		)
		c.closeClientSocket()
		c.closeFile()
	}()
}
