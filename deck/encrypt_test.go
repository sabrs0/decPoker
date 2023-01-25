package deck

import (
	"fmt"
	"reflect"
	"testing"
)

func TestEncryptCard(t *testing.T) {
	key := []byte("foobarbaz")
	card := Card{
		Suit:  Spades,
		Value: 1,
	}
	encOutput, err := EncryptCard(key, card)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("enc card : ", encOutput)

	decCard, err := DecryptCard(key, encOutput)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("dec card : ", decCard)
	if !reflect.DeepEqual(decCard, card) {
		t.Error("Does not matched")
	}
}
