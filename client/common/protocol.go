package common

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// sendMessage Sends a message to the server
// In case of failure, error is returned
// This method avoids short-write
func (c *Client) sendMessage(msg string) error {
	msgBytes := []byte(fmt.Sprintf("%s\n", msg))

	totalSent := 0
	for totalSent < len(msgBytes) {
		sent, err := c.conn.Write(msgBytes[totalSent:])
		if err != nil {
			log.Fatalf("action: send_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			c.StopClient()
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
			log.Fatalf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			c.StopClient()
			return "", err
		}
		message.Write(data[:n])
		if bytes.HasSuffix(message.Bytes(), []byte{'\n'}) {
			break
		}
	}

	return strings.TrimSuffix(message.String(), "\n"), nil
}

// sendBets Sends a list of bets to the server
// In case of failure, true is returned
func sendBets(c *Client, bets []*Bet) bool {
	err := c.createClientSocket()
	defer c.conn.Close()
	if err != nil {
		return true
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
		logBets(bets, "fail")
		c.StopClient()
		return true
	}

	err = c.closeClientSocket()
	if err != nil {
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
func (c *Client) getWinners(result string) (bool, error) {
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
