package deck

import (
	"fmt"
	"math/rand"
	"strconv"
)

type Suit int

func (s Suit) String() string {
	switch s {
	case Spades:
		return "SPADES"
	case Harts:
		return "HARTS"
	case Diamonds:
		return "DIAMONDS"
	case Clubs:
		return "CLUBS"
	default:
		panic("invalid card suit")
	}
}

const (
	Spades Suit = iota
	Harts
	Diamonds
	Clubs
)

type Card struct {
	Suit  Suit
	Value int
}

func (c Card) String() string {
	var value string
	switch c.Value {
	case 1:
		value = "A"
	case 11:
		value = "J"
	case 12:
		value = "Q"
	case 13:
		value = "K"
	default:
		value = strconv.Itoa(c.Value)
	}
	return fmt.Sprintf("%s %s", value, suitToUnicode(c.Suit))
}
func NewCard(s Suit, v int) Card {
	if v > 13 {
		panic("the value of the card cannot be more than 13")
	}
	return Card{
		Suit:  s,
		Value: v,
	}
}

func suitToUnicode(s Suit) string {
	switch s {
	case Spades:
		return "♠"
	case Harts:
		return "♥"
	case Diamonds:
		return "♦"
	case Clubs:
		return "♣"
	default:
		panic("invalid card suit")
	}
}

type Deck [52]Card

func New() Deck {
	var (
		nSuits  = 4
		nValues = 13
		d       = [52]Card{}
	)
	for i := 0; i < nSuits; i++ {
		for j := 0; j < nValues; j++ {
			d[i*nValues+j] = NewCard(Suit(i), j+1)
		}
	}
	return shuffle(d)
}
func shuffle(d Deck) Deck {
	for i := 0; i < len(d); i++ {
		r := rand.Intn(i + 1)
		if i != r {
			d[i], d[r] = d[r], d[i]
		}
	}
	return d
}
