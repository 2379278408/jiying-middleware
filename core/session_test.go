package core

import (
	"context"
	"testing"
	"time"
)

func TestSenderIsolatedPerGoroutine(t *testing.T) {
	const n = 16
	start := make(chan struct{})
	done := make(chan string, n)
	for i := 0; i < n; i++ {
		id := string(rune('a' + i))
		go func() {
			<-start
			withSender(&Faker{UserID: id}, func() {
				time.Sleep(10 * time.Millisecond)
				if got := CurrentSender().(*Faker).UserID; got != id {
					done <- got
					return
				}
				withSender(&Faker{UserID: id + "2"}, func() {
					time.Sleep(5 * time.Millisecond)
					if got := CurrentSender().(*Faker).UserID; got != id+"2" {
						done <- got
						return
					}
				})
				if got := CurrentSender().(*Faker).UserID; got != id {
					done <- got
					return
				}
			})
			done <- ""
		}()
	}
	close(start)
	for i := 0; i < n; i++ {
		if got := <-done; got != "" {
			t.Fatalf("发送者串栈: got %q", got)
		}
	}
	if CurrentSender() != nil {
		t.Fatal("协程结束后不应残留发送者")
	}
}

func TestResumeChoiceRejectsOtherUser(t *testing.T) {
	task, err := EnqueueTask(Task{Feature: "帮助", UserID: "u1", ConversationID: "c1", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	defer DeleteTask(task.ID)
	bindChoice("choice-owner", task)
	if err := resumeChoice(context.Background(), "choice-owner", "u2", "c1", []string{"yes"}); err == nil {
		t.Fatal("非发起人不应能确认操作")
	}
	if err := resumeChoice(context.Background(), "choice-owner", "u1", "c2", []string{"yes"}); err == nil {
		t.Fatal("跨会话确认应被拒绝")
	}
	if err := resumeChoice(context.Background(), "choice-owner", "u1", "c1", []string{"no"}); err != nil {
		t.Fatal(err)
	}
}
