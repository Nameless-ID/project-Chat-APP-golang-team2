package helper

import (
	"math/rand"
	"time"
)

func GenerateOTP() string {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	const digits = "0123456789"
	otp := make([]byte, 4)
	for i := range otp {
		otp[i] = digits[random.Intn(len(digits))]
	}
	return string(otp)
}
