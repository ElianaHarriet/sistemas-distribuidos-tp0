package common

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"
)

// sendMessage Sends a message to the server
// In case of failure, error is returned
// This method avoids short-write
func (c *Client) sendMessage(msg string) error {
    msgBytes := []byte(fmt.Sprintf("%s\n", msg))

    totalSent := 0
    writeErr := make(chan error, 1)
    go func() {
        for totalSent < len(msgBytes) {
            sent, err := c.conn.Write(msgBytes[totalSent:])
            if err != nil {
                log.Fatalf("action: send_message | result: fail | client_id: %v | error: %v",
                    c.config.ID,
                    err,
                )
                c.StopClient()
                writeErr <- err
                return
            }
            totalSent += sent
        }
        writeErr <- nil
    }()

    select {
    case <-c.stop_chan:
        return fmt.Errorf("stopped by user")
    case err := <-writeErr:
        if err != nil {
            return err
        }
    }

    return nil
}

// receiveMessage Receives a message from the server
// In case of failure, error is returned
// This method avoids short-reads
func (c *Client) receiveMessage() (string, error) {
    var message bytes.Buffer
    data := make([]byte, 1024)

    readErr := make(chan error, 1)
    go func() {
        for {
            n, err := c.conn.Read(data)
            if err != nil {
                if err == io.EOF {
                    break
                }
                log.Fatalf("action: receive_message | result: fail | client_id: %v | error: %v",
                    c.config.ID,
                    err,
                )
                c.StopClient()
                readErr <- err
                return
            }
            message.Write(data[:n])
            if bytes.HasSuffix(message.Bytes(), []byte{'\n'}) {
                break
            }
        }
        readErr <- nil
    }()

    select {
    case <-c.stop_chan:
        return "", fmt.Errorf("stopped by user")
    case err := <-readErr:
        if err != nil {
            return "", err
        }
    }

    return strings.TrimSuffix(message.String(), "\n"), nil
}

// sendBets Sends a list of bets to the server
// In case of failure, true is returned
func sendBets(c *Client, bets []*Bet) bool {
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
	err := c.sendMessage(message)

	if err != nil {
		logBets(bets, "fail")
		c.StopClient()
		return true
	}

	logBets(bets, "success")
	return false
}

// readBets Reads a chunk of bets from the file
// In case of failure, true is returned
func (c *Client) readBets(reader *csv.Reader) ([]*Bet, bool, bool) {
	bets := make([]*Bet, 0, c.config.BetChunkSize)
	end := false
	for i := 0; i < c.config.BetChunkSize; i++ {
		bet, err := readBet(c.config.ID, reader)
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
			return nil, false, true
		}
		bets = append(bets, bet)
		c.personal_ids[bet.GetPersonalID()] = true
	}
	return bets, end, false
}

// getWinners Gets the winners from the server
// In case of failure, true is returned
func (c *Client) getWinners(result string) {
	winners := strings.Split(strings.Split(result, ":")[2], ",")
	log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v",
		len(winners),
	)
}
