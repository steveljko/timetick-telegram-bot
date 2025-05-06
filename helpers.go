package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// Converts comma-separated string into slice of integers.
//
// Example:
//
//	Input: "123,456,789"
//	Output: []int64{123, 456, 789}
func convertStringToIntArray(arrString string) []int64 {
	var intArr []int64

	stringSlice := strings.Split(arrString, ",")

	for _, numString := range stringSlice {
		trimmed := strings.TrimSpace(numString) // remove trailing whitespace

		intValue, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			log.Printf("Skipping invalid integer: %q - %v", trimmed, err)
			continue
		}

		intArr = append(intArr, intValue)
	}

	return intArr
}

// Generates random token using cryptographic randomness.
func GenerateToken() (string, error) {
	rb := make([]byte, 32)
	_, err := rand.Read(rb)
	if err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(rb), nil
}

// Creates SHA-256 hash of the provided input.
//
// Example:
//
// Input: asd123
// Output: 54d5cb2d332dbdb4850293caae4559ce88b65163f1ea5d4e4b3ac49d772ded14
func Hash(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	return hex.EncodeToString(hasher.Sum(nil))
}
