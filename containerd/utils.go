package containerd

import (
	"math/rand"
	"time"
)

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "abcdefghijklmnopqrstuvwxyz" + "0123456789")

func getRandomID(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, length)
	for i := 0; i < length; i++ {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
