package common

import (
	"fmt"
)

// Bet struct that represents a bet
type Bet struct {
	ID	   		int		// bet number
	Name   		string
	Surname 	string
	PersonalID  int
	BirthDate   string
}


// NewBet Initializes a new bet
func NewBet(id int, name string, surname string, personalID int, birthDate string) *Bet {
	return &Bet{
		ID: id,
		Name: name,
		Surname: surname,
		PersonalID: personalID,
		BirthDate: birthDate,
	}
}

// ToStr Returns a string representation of the bet
func (b *Bet) ToStr() string {
	return fmt.Sprintf("Bet -> [ID: %d, Name: %s, Surname: %s, PersonalID: %d, BirthDate: %s]",
						b.ID, b.Name, b.Surname, b.PersonalID, b.BirthDate)
}

// Duplicate Returns a new bet with the same values as the original but with a new ID
func (b *Bet) Duplicate() *Bet {
	return &Bet{
		ID: b.ID + 1,
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