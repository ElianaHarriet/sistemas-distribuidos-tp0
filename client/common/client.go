package common

import (
	"fmt"
	"net"
	"time"
	"os"
	"encoding/csv"
	"io"
	"strings"

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
	config ClientConfig
	conn   net.Conn
	data_file *os.File
	stop_chan chan bool
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
		conn:   nil,
		data_file: nil,
		stop_chan: make(chan bool),
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Fatalf(
	        "action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

// CloseClientSocket Closes the client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
// it forces the connection to be closed no matter what
func (c *Client) closeClientSocket() error {
	if c.conn != nil {
		err := c.conn.Close()
		if err != nil {
			log.Fatalf(
				"action: close_connection | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
		}
		c.conn = nil
	}
	return nil
}

// sendMessage Sends a message to the server
// In case of failure, error is returned
// This method avoids short-write
func (c *Client) sendMessage(msg string) error {
    msgBytes := []byte(fmt.Sprintf("%s\n", msg))

    totalSent := 0
    for totalSent < len(msgBytes) {
        sent, err := c.conn.Write(msgBytes[totalSent:])
        if err != nil {
            return err
        }
        totalSent += sent
    }

    return nil
}

// openFile Opens the file in read mode (it does not create the file if it does not exist)
func (c *Client) openFile() error {
	filePath := fmt.Sprintf("%s/%s%d.csv", c.config.DirDataPath, c.config.FileDataName, c.config.ID)
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("action: open_file | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}
	c.data_file = file
	return nil
}

// closeFile Closes the file
func (c *Client) closeFile() error {
	if c.data_file != nil {
		err := c.data_file.Close()
		if err != nil {
			log.Fatalf(
				"action: close_file | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return err
		}
		c.data_file = nil
	}
	return nil
}

// StopClientLoop Stops the client loop
func (c *Client) StopClientLoop() {
    defer func() {
		c.stop_chan <- true
        log.Infof("action: stop_loop | result: success | client_id: %v",
            c.config.ID,
        )
        c.closeClientSocket()
		c.closeFile()
    }()
}

func LogBets(bets []*Bet, result string) {
    for _, bet := range bets {
        log.Infof("action: apuesta_enviada | result: %s | dni: %v | numero: %v",
            result,
            bet.GetPersonalID(),
            bet.GetBetID(),
        )
    }
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

	for  {
		bets := make([]*Bet, 0, c.config.BetChunkSize)
		end := false
		for i := 0; i < c.config.BetChunkSize; i++ {
			bet, err := ReadBet(c.config.ID, reader)
			if err == io.EOF {
				end = true
				break
			}
			if err != nil {
				log.Infof("action: read_bet | result: fail | client_id: %v | error: %v",
					c.config.ID,
					err,
				)
				c.StopClientLoop()
				return
			}
			bets = append(bets, bet)
		}

		// Create the connection the server in every loop iteration
		err = c.createClientSocket()
		defer c.conn.Close()
		if err != nil {
			log.Fatalf("action: connect | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			c.StopClientLoop()
			return
		}

		betStrings := make([]string, len(bets))
		for i, bet := range bets {
			betStrings[i] = bet.ToStr()
		}
		joinedBets := strings.Join(betStrings, "")

		message := fmt.Sprintf(
			"[CLIENT %v] Bets -> %s",
			c.config.ID,
			joinedBets,
		)
		err = c.sendMessage(message)

		if err != nil {
			LogBets(bets, "fail")
			c.StopClientLoop()
			return
		}

		err = c.closeClientSocket()

		if err != nil {
			log.Fatalf("action: close_connection | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			c.StopClientLoop()
			return
		}
		LogBets(bets, "success")

		if end {
			break
		}

		select {
		case <-time.After(c.config.LoopPeriod):
			// Wait a time between sending one message and the next one
		case <-c.stop_chan:
			log.Warnf("action: loop_finished | result: aborted | client_id: %v", c.config.ID)
			return
		}
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
