package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNotificationRecordsStateAndReceipts(t *testing.T) {
	at := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	notification := mustNotification(t, at)
	sent := mustNotificationReceipt(t, "CSV-NOT-1", at, notification.AuditPayload())

	if err := notification.Send(sent); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if notification.State != NotificationStateEnviada || len(notification.Receipts) != 1 {
		t.Fatalf("sent notification = %+v", notification)
	}

	read := mustNotificationReceipt(t, "CSV-NOT-2", at.Add(time.Hour), notification.AuditPayload())
	if err := notification.MarkRead(read); err != nil {
		t.Fatalf("MarkRead() error = %v", err)
	}
	if notification.State != NotificationStateLeida || len(notification.Receipts) != 2 {
		t.Fatalf("read notification = %+v", notification)
	}
}

func TestNotificationRejectsReadBeforeSend(t *testing.T) {
	at := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	notification := mustNotification(t, at)
	receipt := mustNotificationReceipt(t, "CSV-NOT-1", at, notification.AuditPayload())

	err := notification.MarkRead(receipt)
	if !errors.Is(err, ErrNotificationTransition) {
		t.Fatalf("MarkRead() error = %v, want %v", err, ErrNotificationTransition)
	}
}

func mustNotification(t *testing.T, at time.Time) Notification {
	t.Helper()
	notification, err := NewNotification(NotificationInput{
		ID: "not-1", CandidateID: "cand-1", SolicitudID: "sol-1",
		Type: "subsanacion", Subject: "Subsanacion requerida", Body: "Aporte documento.",
		CreatedBy: "gestor-1", CreatedAt: at,
	})
	if err != nil {
		t.Fatalf("NewNotification() error = %v", err)
	}
	return notification
}

func mustNotificationReceipt(t *testing.T, csv string, at time.Time, payload []byte) NotificationReceipt {
	t.Helper()
	receipt, err := NewNotificationReceipt(csv, "cand-1", "vec", at, payload)
	if err != nil {
		t.Fatalf("NewNotificationReceipt() error = %v", err)
	}
	return receipt
}
