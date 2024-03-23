package common

import (
	"fmt"
	"strconv"
	"encoding/csv"
)

// Bet struct that represents a bet
type Bet struct {
	AgencyID	int		// agency number
	ID	   		int		// bet number
	Name   		string
	Surname 	string
	PersonalID  int
	BirthDate   string
}


// NewBet Initializes a new bet
func NewBet(AgencyID int, id int, name string, surname string, personalID int, birthDate string) *Bet {
	return &Bet{
		ID: id,
		Name: name,
		Surname: surname,
		PersonalID: personalID,
		BirthDate: birthDate,
	}
}

// FromCSV Initializes a new bet from a CSV record
func FromCSV(agencyID int, record []string) (*Bet, error) {
	if len(record) < 5 {
		return nil, fmt.Errorf("record does not have enough fields")
	}
	personalID, err := strconv.Atoi(record[2])
	if err != nil {
		return nil, fmt.Errorf("error converting personalID to int: %v", err)
	}
	id, err := strconv.Atoi(record[4])
	if err != nil {
		return nil, fmt.Errorf("error converting id to int: %v", err)
	}
	return &Bet{
		AgencyID:   agencyID,
		ID:         id,
		Name:       record[0],
		Surname:    record[1],
		PersonalID: personalID,
		BirthDate:  record[3],
	}, nil
}


// ToStr Returns a string representation of the bet
func (b *Bet) ToStr() string {
	return fmt.Sprintf("[AgencyID:%d,ID:%d,Name:%s,Surname:%s,PersonalID:%d,BirthDate:%s]",
						b.AgencyID, b.ID, b.Name, b.Surname, b.PersonalID, b.BirthDate)
}

// Duplicate Returns a new bet with the same values as the original but with a new ID
func (b *Bet) Duplicate() *Bet {
	return &Bet{
		AgencyID: b.AgencyID,
		ID: (b.ID % 9999) + 1,
		Name: b.Name,
		Surname: b.Surname,
		PersonalID: b.PersonalID,
		BirthDate: b.BirthDate,
	}
}

// GetBetID Returns the bet ID
func (b *Bet) GetBetID() int {
	return b.ID
}

// GetPesonalID Returns the personal ID
func (b *Bet) GetPersonalID() int {
	return b.PersonalID
}

// ReadBet Reads a bet from a CSV file
func ReadBet(agencyID int, reader *csv.Reader) (*Bet, error) {
    record, err := reader.Read()
    if err != nil {
        return nil, err
    }

    bet, err := FromCSV(agencyID, record)
    if err != nil {
        return nil, err
    }

    return bet, nil
}