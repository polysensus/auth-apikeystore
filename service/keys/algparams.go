package keys

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	space         = " "
	algParts      = 3
	memSuffix     = "MB"
	maxKeyLength  = 64
	minKeyLength  = 16
	maxMem        = 64
	minMem        = 16
	maxTime       = 5
	minTime       = 1
	argon2idAlgID = "argon2id:"
)

type ParamsArgon2ID struct {
	Time   uint32
	Memory uint32
	KeyLen uint32
}

type Alg struct {
	String string
	ParamsArgon2ID
}

func ParseAlg(alg string) (Alg, error) {
	if !strings.HasPrefix(alg, argon2idAlgID) {
		return Alg{}, fmt.Errorf("missing or unsupportred algorithm name `%s'", alg)
	}

	a := Alg{String: alg}

	alg = alg[len(argon2idAlgID):]

	parts := strings.SplitN(alg, space, algParts)
	if len(parts) != 3 {
		return Alg{}, fmt.Errorf("bad alg string `%s'", alg)
	}
	u, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return Alg{}, fmt.Errorf("bad times component `%s': %v", parts[0], err)
	}
	a.Time = uint32(u)
	if a.Time > maxTime {
		return Alg{}, fmt.Errorf("time `%s' to large. max=%d", parts[0], maxTime)
	}
	if a.Time < minTime {
		return Alg{}, fmt.Errorf("time `%s' to small. min=%d", parts[0], minTime)
	}

	if !strings.HasSuffix(parts[1], memSuffix) {
		return Alg{}, fmt.Errorf("bad memory component `%s' (wrong or missing suffix)", parts[1])
	}
	u, err = strconv.ParseUint(parts[1][:len(parts[1])-len(memSuffix)], 10, 32)
	if err != nil {
		return Alg{}, fmt.Errorf("bad memory component `%s': %v", parts[1], err)
	}
	a.Memory = uint32(u)
	if a.Memory > maxMem {
		return Alg{}, fmt.Errorf("time `%s' to large. max=%d", parts[1], maxMem)
	}
	if a.Memory < minMem {
		return Alg{}, fmt.Errorf("memory `%s' to small. min=%d", parts[1], minMem)
	}

	u, err = strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return Alg{}, fmt.Errorf("bad keylength `%s': %v", parts[2], err)
	}
	if u > maxKeyLength {
		return Alg{}, fmt.Errorf("key length `%s' to large. max=%d", parts[2], maxKeyLength)
	}
	a.KeyLen = uint32(u)
	if a.KeyLen < minKeyLength {
		return Alg{}, fmt.Errorf("key length `%s' to small. min=%d", parts[2], minKeyLength)
	}

	return a, nil
}
