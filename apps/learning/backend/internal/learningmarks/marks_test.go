package learningmarks

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	ok := Trigger{ID: "a", Word: " Rule: ", Style: "frame", Color: "blue"}

	got, err := validate(Marks{AnyLabel: true, Triggers: []Trigger{ok}})
	if err != nil || got.Triggers[0].Word != "Rule" {
		t.Fatalf("a word is trimmed of spaces and its colon: %+v, %v", got, err)
	}

	bad := map[string]Marks{
		"empty word": {Triggers: []Trigger{{ID: "a", Word: " : ", Style: "frame", Color: "blue"}}},
		"long word":  {Triggers: []Trigger{{ID: "a", Word: strings.Repeat("я", 41), Style: "frame", Color: "blue"}}},
		"two lines":  {Triggers: []Trigger{{ID: "a", Word: "a\nb", Style: "frame", Color: "blue"}}},
		"bad style":  {Triggers: []Trigger{{ID: "a", Word: "x", Style: "blink", Color: "blue"}}},
		"bad colour": {Triggers: []Trigger{{ID: "a", Word: "x", Style: "frame", Color: "pink"}}},
		"no id":      {Triggers: []Trigger{{Word: "x", Style: "frame", Color: "blue"}}},
		"duplicate":  {Triggers: []Trigger{ok, {ID: "b", Word: "rule", Style: "bar", Color: "red"}}},
		"too many":   {Triggers: make([]Trigger, maxTriggers+1)},
	}

	for name, in := range bad {
		if _, err := validate(in); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}
