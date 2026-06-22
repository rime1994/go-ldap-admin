package tools

import (
	"crypto/md5"
	"encoding/binary"
)

const posixIDMin = 10000
const posixIDMax = 60000

// GeneratePosixID maps seed deterministically into [10000, 60000),
// incrementing on collision. Marks the chosen id in taken.
func GeneratePosixID(seed string, taken map[int]bool) int {
	sum := md5.Sum([]byte(seed))
	n := int(binary.BigEndian.Uint32(sum[:4]))
	id := posixIDMin + n%(posixIDMax-posixIDMin)
	for taken[id] {
		id++
		if id >= posixIDMax {
			id = posixIDMin
		}
	}
	taken[id] = true
	return id
}
