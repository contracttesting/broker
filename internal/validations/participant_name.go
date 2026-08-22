package validations

import (
	"errors"
	"strings"
)

func ParticipantName(value string) error {
	for _, word := range strings.Split(value, "_") {
		if word == "" {
			return errors.New("must be snake_case")
		}

		for _, character := range word {
			if (character < 'a' || character > 'z') && (character < '0' || character > '9') {
				return errors.New("must be snake_case")
			}
		}
	}

	return nil
}
