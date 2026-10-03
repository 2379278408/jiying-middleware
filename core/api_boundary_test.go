package core

import "testing"

func TestValidConversationID(t *testing.T) {
	valid := []string{"conv-1", "topic_2", "abc.def", "A1-b2_c3"}
	for _, id := range valid {
		if !validConversationID(id) {
			t.Fatalf("expected %q to be valid", id)
		}
	}
	invalid := []string{
		"",
		".",
		"..",
		"../messages",
		"conv/1",
		"conv 1",
		"conv?x=1",
		"conv#frag",
		"conv%2f",
		"conv\\1",
		"conv\n1",
	}
	for _, id := range invalid {
		if validConversationID(id) {
			t.Fatalf("expected %q to be rejected", id)
		}
	}
}
