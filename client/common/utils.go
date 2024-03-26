package common

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

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
		c.StopClient()
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
			c.StopClient()
		}
		c.conn = nil
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

// waitOrStop Waits for the loop period or stops the client if stop_chan is flagged
// Returns true if the client should stop
func (c *Client) waitOrStop() bool {
	select {
	case <-time.After(c.config.LoopPeriod):

	case <-c.stop_chan:
		log.Warnf("action: loop_finished | result: aborted | client_id: %v", c.config.ID)
		return true
	}
	return false
}

// askResults Sends a message to the server to ask for the results
// Returns true if the client should wait for the results and keep
// asking for them
func (c *Client) askResults() (bool, error) {
	err := c.createClientSocket()
	defer c.conn.Close()
	if err != nil {
		return false, err
	}

	message := fmt.Sprintf(
		"[CLIENT %v] Awaiting results",
		c.config.ID,
	)
	err = c.sendMessage(message)

	if err != nil {
		return false, err
	}

	result, err := c.receiveMessage()
	if err != nil {
		return false, err
	}

	log.Infof("action: receive_message | result: success | client_id: %v | message: %v",
		c.config.ID,
		result,
	)

	wait, shouldReturn, returnValue1 := c.manageServerResponse(result)
	if shouldReturn {
		return false, returnValue1
	}

	err = c.closeClientSocket()
	if err != nil {
		return false, err
	}
	return wait, nil
}

// manageServerResponse Manages the server response
// Returns true if the client should wait for the results and keep
// asking for them
func (c *Client) manageServerResponse(result string) (bool, bool, error) {
	wait := true
	if strings.HasPrefix(result, "OK") {
		shouldReturn, returnValue1 := c.getWinners(result)
		if shouldReturn {
			return false, true, returnValue1
		}
		wait = false
	} else if strings.HasPrefix(result, "ERROR") {
		log.Fatalf("action: consulta_ganadores | result: fail | %v",
			result,
		)
		c.StopClient()
		err := errors.New(result)
		return false, true, err
	} else if strings.HasPrefix(result, "WAIT") {
		wait = true
		log.Infof("action: consulta_ganadores | result: wait | %v",
			result,
		)
	}
	return wait, false, nil
}