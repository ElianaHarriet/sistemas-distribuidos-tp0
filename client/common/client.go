package common

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopLapse     time.Duration
	LoopPeriod    time.Duration
	Bet		   	  *Bet
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
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

// StopClientLoop Stops the client loop
func (c *Client) StopClientLoop() {
    defer func() {
        log.Infof("action: stop_loop | result: success | client_id: %v",
            c.config.ID,
        )
        c.closeClientSocket()
    }()
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {

loop:
	// Send messages if the loopLapse threshold has not been surpassed
	for timeout := time.After(c.config.LoopLapse); ; {
		select {
		case <-timeout:
	        log.Infof("action: timeout_detected | result: success | client_id: %v",
                c.config.ID,
            )
			break loop
		default:
		}

		// Create the connection the server in every loop iteration
		c.createClientSocket()
		defer c.conn.Close()

		message := fmt.Sprintf(
			"[CLIENT %v] %s",
			c.config.ID,
			c.config.Bet.ToStr(),
		)
		err := c.sendMessage(message)

		c.closeClientSocket()

		if err != nil {
			log.Infof("action: apuesta_enviada | result: fail | dni: %v | numero: %v",
				c.config.Bet.GetPersonalID(),
				c.config.Bet.GetBetID(),
			)
			return
		}
		log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v",
            c.config.Bet.GetPersonalID(),
			c.config.Bet.GetBetID(),
        )

		// Duplicate the bet
		c.config.Bet = c.config.Bet.Duplicate()

		// Wait a time between sending one message and the next one
		time.Sleep(c.config.LoopPeriod)
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
