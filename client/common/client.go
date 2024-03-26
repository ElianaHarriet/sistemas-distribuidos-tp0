package common

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
	"os"
	"strconv"
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

// receiveMessage Receives a message from the server
// In case of failure, error is returned
// This method avoids short-reads
func (c *Client) receiveMessage() (string, error) {
	var message bytes.Buffer
	data := make([]byte, 1024)

	for {
		n, err := c.conn.Read(data)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		message.Write(data[:n])
		if bytes.HasSuffix(message.Bytes(), []byte{'\n'}) {
			break
		}
	}

	return strings.TrimSuffix(message.String(), "\n"), nil
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
                c.StopClient()
                return
            }
            bets = append(bets, bet)
			c.personal_ids[bet.GetPersonalID()] = true
        }

		if len(bets) == 0 {
			break
		}

        // Create the connection the server in every loop iteration
        err = c.createClientSocket()
		defer c.conn.Close()
        if err != nil {
            log.Fatalf("action: connect | result: fail | client_id: %v | error: %v",
                c.config.ID,
                err,
            )
            c.StopClient()
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
            c.StopClient()
            return
        }

        err = c.closeClientSocket()

        if err != nil {
            log.Fatalf("action: close_connection | result: fail | client_id: %v | error: %v",
                c.config.ID,
                err,
            )
            c.StopClient()
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

	wait := true
	for wait {
		wait, err = c.getWinners()
		if err != nil {
			return
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

func (c *Client) getWinners() (bool, error) {
	err := c.createClientSocket()
	defer c.conn.Close()
	if err != nil {
		log.Fatalf("action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		c.StopClient()
		return false, err
	}

	message := fmt.Sprintf(
		"[CLIENT %v] Awaiting results",
		c.config.ID,
	)
	err = c.sendMessage(message)

	if err != nil {
		log.Fatalf("action: send_message | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		c.StopClient()
		return false, err
	}

	result, err := c.receiveMessage()
	if err != nil {
		log.Fatalf("action: receive_message | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		c.StopClient()
		return false, err
	}

	log.Infof("action: receive_message | result: success | client_id: %v | message: %v",
		c.config.ID,
		result,
	)

	wait := true
	if strings.HasPrefix(result, "OK") {
		shouldReturn, returnValue1 := c.logWinners(result)
		if shouldReturn {
			return false, returnValue1
		}
		wait = false
	} else if strings.HasPrefix(result, "ERROR") {
		log.Fatalf("action: consulta_ganadores | result: fail | %v",
			result,
		)
		c.StopClient()
		err = errors.New(result)
		return false, err
	} else if strings.HasPrefix(result, "WAIT") {
		wait = true
		log.Infof("action: consulta_ganadores | result: wait | %v",
			result,
		)
	}

	err = c.closeClientSocket()
	if err != nil {
		log.Fatalf("action: close_connection | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		c.StopClient()
		return false, err
	}
	return wait, nil
}

func (c *Client) logWinners(result string) (bool, error) {
	winners := strings.Split(strings.Split(result, ":")[2], ",")
	num_winners := 0
	for _, winner := range winners {
		winner_id, err := strconv.Atoi(winner)
		if err != nil {
			log.Fatalf("action: convert_winner | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			c.StopClient()
			return true, err
		}
		if _, ok := c.personal_ids[winner_id]; ok {
			num_winners++
		}
	}
	log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v",
		num_winners,
	)
	return false, nil
}
