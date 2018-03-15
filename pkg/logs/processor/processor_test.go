// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package processor

import (
	"regexp"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
	"github.com/stretchr/testify/assert"
)

func buildTestLogsConfig(ruleType, replacePlaceholder, pattern string) *config.LogsConfig {
	rule := config.LogsProcessingRule{
		Type:                    ruleType,
		Name:                    "test",
		ReplacePlaceholder:      replacePlaceholder,
		ReplacePlaceholderBytes: []byte(replacePlaceholder),
		Pattern:                 pattern,
		Reg:                     regexp.MustCompile(pattern),
	}
	return &config.LogsConfig{ProcessingRules: []config.LogsProcessingRule{rule}}
}

func newNetworkMessage(content []byte, cfg *config.LogsConfig) message.Message {
	msg := message.NewNetworkMessage(content)
	msgOrigin := message.NewOrigin()
	msgOrigin.LogsConfig = cfg
	msg.SetOrigin(msgOrigin)
	return msg
}

func TestExclusion(t *testing.T) {
	var shouldProcess bool
	var redactedMessage []byte

	logsConfig := buildTestLogsConfig("exclude_at_match", "", "world")
	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("hello"), logsConfig))
	assert.Equal(t, true, shouldProcess)
	assert.Equal(t, []byte("hello"), redactedMessage)

	shouldProcess, _ = applyRedactingRules(newNetworkMessage([]byte("world"), logsConfig))
	assert.Equal(t, false, shouldProcess)

	shouldProcess, _ = applyRedactingRules(newNetworkMessage([]byte("a brand new world"), logsConfig))
	assert.Equal(t, false, shouldProcess)

	logsConfig = buildTestLogsConfig("exclude_at_match", "", "$world")
	shouldProcess, _ = applyRedactingRules(newNetworkMessage([]byte("a brand new world"), logsConfig))
	assert.Equal(t, true, shouldProcess)
}

func TestInclusion(t *testing.T) {
	var shouldProcess bool
	var redactedMessage []byte

	logsConfig := buildTestLogsConfig("include_at_match", "", "world")
	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("hello"), logsConfig))
	assert.Equal(t, false, shouldProcess)
	assert.Nil(t, redactedMessage)

	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("world"), logsConfig))
	assert.Equal(t, true, shouldProcess)
	assert.Equal(t, []byte("world"), redactedMessage)

	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("a brand new world"), logsConfig))
	assert.Equal(t, true, shouldProcess)
	assert.Equal(t, []byte("a brand new world"), redactedMessage)

	logsConfig = buildTestLogsConfig("include_at_match", "", "^world")
	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("a brand new world"), logsConfig))
	assert.Equal(t, false, shouldProcess)
	assert.Nil(t, redactedMessage)
}

func TestExclusionWithInclusion(t *testing.T) {
	var shouldProcess bool
	var redactedMessage []byte

	ePattern := "^bob"
	eRule := config.LogsProcessingRule{
		Type:    "exclude_at_match",
		Name:    "exclude_bob",
		Pattern: ePattern,
		Reg:     regexp.MustCompile(ePattern),
	}
	iPattern := ".*@datadoghq.com$"
	iRule := config.LogsProcessingRule{
		Type:    "include_at_match",
		Name:    "include_datadoghq",
		Pattern: iPattern,
		Reg:     regexp.MustCompile(iPattern),
	}
	logsConfig := &config.LogsConfig{ProcessingRules: []config.LogsProcessingRule{eRule, iRule}}

	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("bob@datadoghq.com"), logsConfig))
	assert.Equal(t, false, shouldProcess)
	assert.Nil(t, redactedMessage)

	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("bill@datadoghq.com"), logsConfig))
	assert.Equal(t, true, shouldProcess)
	assert.Equal(t, []byte("bill@datadoghq.com"), redactedMessage)

	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("bob@amail.com"), logsConfig))
	assert.Equal(t, false, shouldProcess)
	assert.Nil(t, redactedMessage)

	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("bill@amail.com"), logsConfig))
	assert.Equal(t, false, shouldProcess)
	assert.Nil(t, redactedMessage)
}

func TestMask(t *testing.T) {
	var shouldProcess bool
	var redactedMessage []byte

	logsConfig := buildTestLogsConfig("mask_sequences", "[masked_world]", "world")
	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("hello"), logsConfig))
	assert.Equal(t, true, shouldProcess)
	assert.Equal(t, []byte("hello"), redactedMessage)

	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("hello world!"), logsConfig))
	assert.Equal(t, true, shouldProcess)
	assert.Equal(t, []byte("hello [masked_world]!"), redactedMessage)

	logsConfig = buildTestLogsConfig("mask_sequences", "[masked_user]", "User=\\w+@datadoghq.com")
	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("new test launched by User=beats@datadoghq.com on localhost"), logsConfig))
	assert.Equal(t, true, shouldProcess)
	assert.Equal(t, []byte("new test launched by [masked_user] on localhost"), redactedMessage)

	logsConfig = buildTestLogsConfig("mask_sequences", "[masked_credit_card]", "(?:4[0-9]{12}(?:[0-9]{3})?|[25][1-7][0-9]{14}|6(?:011|5[0-9][0-9])[0-9]{12}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|(?:2131|1800|35\\d{3})\\d{11})")
	shouldProcess, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("The credit card 4323124312341234 was used to buy some time"), logsConfig))
	assert.Equal(t, true, shouldProcess)
	assert.Equal(t, []byte("The credit card [masked_credit_card] was used to buy some time"), redactedMessage)
}

func TestTruncate(t *testing.T) {
	logsConfig := &config.LogsConfig{}
	var redactedMessage []byte

	_, redactedMessage = applyRedactingRules(newNetworkMessage([]byte("hello"), logsConfig))
	assert.Equal(t, []byte("hello"), redactedMessage)
}
