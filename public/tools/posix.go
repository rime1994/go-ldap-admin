package tools

import (
	"crypto/md5"
	"encoding/binary"
)

const posixIDMin = 10000
const posixIDMax = 60000

// GeneratePosixID maps seed deterministically into [10000, 60000),
// incrementing on collision. Marks the chosen id in taken.
// Use for uidNumber where the population is large (100+ users).
func GeneratePosixID(seed string, taken map[int]bool) int {
	id := HashToPosixID(seed)
	for taken[id] {
		id++
		if id >= posixIDMax {
			id = posixIDMin
		}
	}
	taken[id] = true
	return id
}

// HashToPosixID maps seed deterministically into [10000, 60000) with no
// collision tracking. Safe for small populations (e.g. ≤20 departments).
func HashToPosixID(seed string) int {
	sum := md5.Sum([]byte(seed))
	n := int(binary.BigEndian.Uint32(sum[:4]))
	return posixIDMin + n%(posixIDMax-posixIDMin)
}
