package parser

import (
	"bufio"
	"errors"
	"strings"
)

func parseLine(s *bufio.Scanner, skip int, limit int) ([]string, int, error) {
	if !s.Scan() {
		return nil, 0, errors.New("Error while reading")
	}
	line := s.Text()
	if len(line) < 1 {
		return []string{}, 0, errors.New("Empty content")
	}
	tmp := strings.Split(line, separator)
	lgt := skip + limit
	if len(tmp) < skip || len(tmp) < lgt {
		return nil, 0, errors.New("Array index overflow")
	}
	if limit < 1 {
		lgt = len(tmp)
	}
	return tmp[skip:lgt], len([]byte(line)), nil
}
