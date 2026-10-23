package generator

import "crypto/rand"

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

const maxByte = 256 - 256%len(alphabet)

func GenerateShortCode(length int) string {
	code := make([]byte, 0, length)
	buf := make([]byte, length+length/2)
	for len(code) < length {
		rand.Read(buf)
		for _, b := range buf {
			if int(b) >= maxByte {
				continue
			}
			code = append(code, alphabet[int(b)%len(alphabet)])
			if len(code) == length {
				break
			}
		}
	}
	return string(code)
}
