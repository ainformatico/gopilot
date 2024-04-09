package main

import (
	"io"
	"strconv"
	"testing"
)

type MockReadCloser struct {
	data string
}

func (m *MockReadCloser) Read(p []byte) (n int, err error) {
	copy(p, []byte(m.data))

	return len(m.data), io.EOF
}

func (m *MockReadCloser) Close() error {
	return nil
}

func TestRemoveDataUntil(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`data: {"choices":[{"delta":{"content":"hello"}}]}`, `{"choices":[{"delta":{"content":"hello"}}]}`},
		{`{"choices":[{"delta":{"content":"hello"}}]}`, `{"choices":[{"delta":{"content":"hello"}}]}`},
		{"foo", "foo"},
		{"", ""},
	}

	for _, tt := range tests {
		got := removeUntilData(tt.input)

		if got != tt.want {
			t.Errorf("got %s want %s", got, tt.want)
		}
	}
}

func TestParseResponse(t *testing.T) {
	tests := []struct {
		input               string
		want                string
		callbackCalledTimes int
	}{
		{`
		data: {"choices":[{"delta":{"content":"hello"}}]}
		data: {"choices":[{"delta":{"content":" "}}]}
		data: {"choices":[{"delta":{"content":"from"}}]}
		\n
		data: {"choices":[{"delta":{"content":" "}}]}
		data: {"choices":[{"delta":{"content":"here."}}]}
		data: {"choices":[{"delta":{"content":"\n"}}]}
		data: {"choices":[{"delta":{"content":"Bye!"}}]}

		[DONE]
		`, "hello from here.\nBye!",
			8},
		{`
		data: {"choices":[{"delta":{"content":"hello"}}]}
		[DONE]
		data: {"choices":[{"delta":{"content":" "}}]}
		data: {"choices":[{"delta":{"content":"from"}}]}
		`, "hello",
			2},
		{"", "", 1},
	}

	for _, tt := range tests {
		mockReader := &MockReadCloser{data: tt.input}
		totalCalls := 0
		finished := false

		got := parseResponse(mockReader, func(s string, b bool) {
			totalCalls++

			finished = b
		})

		if totalCalls != tt.callbackCalledTimes {
			t.Errorf("got %d want %d", totalCalls, tt.callbackCalledTimes)
		}

		if !finished {
			t.Errorf("Not finished")
		}

		if got != tt.want {
			t.Errorf("got %s want %s", got, tt.want)
		}
	}
}

func TestExtractExpiration(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"tid=00000000000000000000000000000000;exp=1714329795;sku=yearly_subscriber;st=dotcom;chat=1;8kp=1:1111111111111111111111111111111111111111111111111111111111111111", 1714329795},
		{"tid=00000000000000000000000000000000;sku=yearly_subscriber;st=dotcom;chat=1;8kp=1:1111111111111111111111111111111111111111111111111111111111111111", 0},
		{"", 0},
	}

	for _, tt := range tests {
		got := extractExpiration(tt.input)

		if got != tt.want {
			t.Errorf("got %d want %d", got, tt.want)
		}
	}
}

func TestIsExpired(t *testing.T) {
	tests := []struct {
		input int64
		want  bool
	}{
		{1714329795, true},
		{2714329795, false},
		{0, true},
	}

	for _, tt := range tests {
		got := isExpired(tt.input)

		if got != tt.want {
			t.Errorf("got %s want %s", strconv.FormatBool(got), strconv.FormatBool(tt.want))
		}
	}
}
