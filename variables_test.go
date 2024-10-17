package main

import (
	"testing"

	"github.com/fr3h4g/mjau/cmd/mjau"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func TestInsertVariables(t *testing.T) {
	config := mjau.Config{
		StoredVariables: []mjau.KeyValue{
			{Key: "key1", Value: "value1"},
			{Key: "key2", Value: "value2"},
		},
	}
	assert.Equal(t, "value1", config.InsertVariables("{{key1}}"), "Should return value1")
	assert.Equal(
		t,
		"value1value2",
		config.InsertVariables("{{key1}}{{key2}}"),
		"Should return value1value2",
	)
	assert.Equal(
		t,
		"value10",
		config.InsertVariables("{{key1}}{{$random(1)}}"),
		"Should not return value10",
	)
	assert.True(t, IsValidUUID(config.InsertVariables("{{$uuid()}}")), "Should return a valid UUID")
}
