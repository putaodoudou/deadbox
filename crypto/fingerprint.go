package crypto

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"reflect"
)

const (
	hashSeparator             = "\000\000\000"
	hashFunction              = crypto.SHA256
	fingerprintGroupSeparator = ":"
)

var fingerprintEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func FingerprintPublicKey(
	key *rsa.PublicKey,
	encryptionType string,
	challengeLevel int,
	fingerprintLengthInGroups int,
) (string, error) {
	keyBytes, err := marshalPublicKey(key)
	if err != nil {
		return "", fmt.Errorf("marshalling public key failed: %s", err)
	}

	hashSum, _, err := findChallengeSolution(keyBytes, encryptionType, hashFunction, challengeLevel)
	if err != nil {
		return "", fmt.Errorf("generating hashsum failed: %s", err)
	}

	hashSumString := fingerprintEncoding.EncodeToString(hashSum[challengeLevel:])
	fingerprint := generateGroupedFingerprint(hashSumString,
		fingerprintLengthInGroups,
		fingerprintGroupSeparator)

	return fingerprint, nil
}

func findChallengeSolution(
	keyBytes []byte,
	encryptionType string,
	hashFunction crypto.Hash,
	challengeLevel int,
) (hashSum []byte, challengeSolution int, err error) {
	for challengeSolution = 0; !isPassChallenge(hashSum, challengeLevel); challengeSolution++ {
		hashSum, err = generateHashSum(challengeSolution, keyBytes, encryptionType, hashFunction)
		if err != nil {
			return
		}
	}

	return
}

func isPassChallenge(hashInput []byte, challengeLevel int) bool {
	if hashInput == nil {
		return false
	}

	for _, b := range hashInput[:challengeLevel] {
		if b != 0 {
			return false
		}
	}

	return true
}

func generateHashSum(
	challengeSolution int,
	keyBytes []byte,
	encryptionType string,
	hashFunction crypto.Hash,
) ([]byte, error) {
	hashInput, err := generateHashInput(challengeSolution, keyBytes, encryptionType)
	if err != nil {
		return nil, fmt.Errorf("generating hash input failed: %s", err)
	}

	hash := hashFunction.New()
	_, err = hash.Write(hashInput)
	if err != nil {
		hashType := reflect.TypeOf(hash)
		return nil, fmt.Errorf("%s: %s", hashType, err)
	}

	return hash.Sum([]byte{}), nil
}

func generateHashInput(modifier int, keyBytes []byte, encryptionType string) ([]byte, error) {
	var hashInputBuffer bytes.Buffer

	hashInputBuffer.Write(keyBytes)
	hashInputBuffer.WriteString(hashSeparator)

	hashInputBuffer.WriteString(encryptionType)
	hashInputBuffer.WriteString(hashSeparator)

	err := binary.Write(&hashInputBuffer, binary.BigEndian, int64(modifier))
	if err != nil {
		return nil, fmt.Errorf("writing modifier failed: %s", err)
	}

	// @todo #27 add validity time

	return hashInputBuffer.Bytes(), nil
}

func generateGroupedFingerprint(hashSumString string, numberOfGroups int, groupSeparator string) string {
	var groupedFingerprintBuffer bytes.Buffer
	for groupIdx := 0; groupIdx < numberOfGroups; groupIdx++ {
		groupedFingerprintBuffer.WriteString(hashSumString[2*groupIdx : 2*groupIdx+2])

		if groupIdx < numberOfGroups-1 {
			groupedFingerprintBuffer.WriteString(groupSeparator)
		}
	}
	fingerprint := groupedFingerprintBuffer.String()
	return fingerprint
}
