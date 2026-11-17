package polls

import (
	"errors"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name  string
		input CreateInput
		valid bool
	}{
		{"trim and preserve case", CreateInput{"  Lunch?  ", []string{" Dosa ", "Idli"}}, true},
		{"missing question", CreateInput{" ", []string{"Dosa", "Idli"}}, false},
		{"too few choices", CreateInput{"Lunch?", []string{"Dosa"}}, false},
		{"empty choice", CreateInput{"Lunch?", []string{"Dosa", " "}}, false},
		{"duplicate choice", CreateInput{"Lunch?", []string{"Masala Dosa", "masala  dosa"}}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			question, options, err := validate(test.input)
			if (err == nil) != test.valid {
				t.Fatalf("unexpected validation result: %v", err)
			}
			if test.valid && (question != "Lunch?" || options[0].Label != "Dosa" || options[0].ID != "1") {
				t.Fatalf("unexpected normalized payload: %q, %+v", question, options)
			}
		})
	}
}

func TestCheckChoice(t *testing.T) {
	poll := Poll{Options: []Option{{ID: "1"}}}
	if err := checkChoice(poll, "1"); err != nil {
		t.Fatal(err)
	}
	if err := checkChoice(poll, "2"); !errors.Is(err, errChoice) {
		t.Fatalf("foreign option: %v", err)
	}
	poll.Closed = true
	if err := checkChoice(poll, "1"); !errors.Is(err, errClosed) {
		t.Fatalf("closed poll: %v", err)
	}
}
