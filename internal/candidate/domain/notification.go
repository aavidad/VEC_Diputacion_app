package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotificationInvalid    = errors.New("notification is invalid")
	ErrNotificationTransition = errors.New("notification transition is invalid")
)

type NotificationState string

const (
	NotificationStateCreada    NotificationState = "Creada"
	NotificationStateEnviada   NotificationState = "Enviada"
	NotificationStateLeida     NotificationState = "Leida"
	NotificationStateFallida   NotificationState = "Fallida"
	NotificationStateCancelada NotificationState = "Cancelada"
)

func (s NotificationState) IsValid() bool {
	switch s {
	case NotificationStateCreada, NotificationStateEnviada, NotificationStateLeida,
		NotificationStateFallida, NotificationStateCancelada:
		return true
	default:
		return false
	}
}

type NotificationReceipt struct {
	CSV         CSV
	RecipientID string
	Channel     string
	IssuedAt    time.Time
	PayloadHash string
}

func NewNotificationReceipt(csv, recipientID, channel string, issuedAt time.Time, payload []byte) (NotificationReceipt, error) {
	receiptCSV, err := NewCSV(csv)
	if err != nil {
		return NotificationReceipt{}, err
	}
	receipt := NotificationReceipt{
		CSV: receiptCSV, RecipientID: strings.TrimSpace(recipientID),
		Channel: strings.TrimSpace(channel), IssuedAt: issuedAt.UTC(),
		PayloadHash: HashAuditPayload(payload),
	}
	if err := receipt.Validate(); err != nil {
		return NotificationReceipt{}, err
	}
	return receipt, nil
}

func (r NotificationReceipt) Validate() error {
	switch {
	case r.CSV.Validate() != nil:
		return r.CSV.Validate()
	case strings.TrimSpace(r.RecipientID) == "":
		return fmt.Errorf("%w: recipient id is required", ErrNotificationInvalid)
	case strings.TrimSpace(r.Channel) == "":
		return fmt.Errorf("%w: channel is required", ErrNotificationInvalid)
	case r.IssuedAt.IsZero():
		return fmt.Errorf("%w: issued at is required", ErrNotificationInvalid)
	case strings.TrimSpace(r.PayloadHash) == "":
		return fmt.Errorf("%w: payload hash is required", ErrNotificationInvalid)
	default:
		return nil
	}
}

type NotificationInput struct {
	ID          string
	CandidateID string
	SolicitudID string
	Type        string
	Subject     string
	Body        string
	CreatedBy   string
	CreatedAt   time.Time
}

type Notification struct {
	ID          string
	CandidateID string
	SolicitudID string
	Type        string
	Subject     string
	Body        string
	State       NotificationState
	CreatedBy   string
	CreatedAt   time.Time
	Receipts    []NotificationReceipt
}

func NewNotification(input NotificationInput) (Notification, error) {
	notification := Notification{
		ID: strings.TrimSpace(input.ID), CandidateID: strings.TrimSpace(input.CandidateID),
		SolicitudID: strings.TrimSpace(input.SolicitudID), Type: strings.TrimSpace(input.Type),
		Subject: strings.TrimSpace(input.Subject), Body: strings.TrimSpace(input.Body),
		State: NotificationStateCreada, CreatedBy: strings.TrimSpace(input.CreatedBy),
		CreatedAt: input.CreatedAt.UTC(),
	}
	if err := notification.Validate(); err != nil {
		return Notification{}, err
	}
	return notification, nil
}

func (n Notification) Validate() error {
	switch {
	case strings.TrimSpace(n.ID) == "":
		return fmt.Errorf("%w: id is required", ErrNotificationInvalid)
	case strings.TrimSpace(n.CandidateID) == "":
		return fmt.Errorf("%w: candidate id is required", ErrNotificationInvalid)
	case strings.TrimSpace(n.SolicitudID) == "":
		return fmt.Errorf("%w: solicitud id is required", ErrNotificationInvalid)
	case strings.TrimSpace(n.Type) == "":
		return fmt.Errorf("%w: type is required", ErrNotificationInvalid)
	case strings.TrimSpace(n.Subject) == "":
		return fmt.Errorf("%w: subject is required", ErrNotificationInvalid)
	case strings.TrimSpace(n.Body) == "":
		return fmt.Errorf("%w: body is required", ErrNotificationInvalid)
	case !n.State.IsValid():
		return fmt.Errorf("%w: state %q", ErrNotificationInvalid, n.State)
	case strings.TrimSpace(n.CreatedBy) == "":
		return fmt.Errorf("%w: created by is required", ErrNotificationInvalid)
	case n.CreatedAt.IsZero():
		return fmt.Errorf("%w: created at is required", ErrNotificationInvalid)
	}
	for _, receipt := range n.Receipts {
		if err := receipt.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (n *Notification) Send(receipt NotificationReceipt) error {
	return n.transitionWithReceipt(NotificationStateEnviada, receipt)
}

func (n *Notification) MarkRead(receipt NotificationReceipt) error {
	return n.transitionWithReceipt(NotificationStateLeida, receipt)
}

func (n *Notification) transitionWithReceipt(to NotificationState, receipt NotificationReceipt) error {
	if n == nil {
		return fmt.Errorf("%w: nil notification", ErrNotificationTransition)
	}
	if err := receipt.Validate(); err != nil {
		return err
	}
	if !n.canTransition(to) {
		return fmt.Errorf("%w: %s -> %s", ErrNotificationTransition, n.State, to)
	}
	n.State = to
	n.Receipts = append(n.Receipts, receipt)
	return n.Validate()
}

func (n Notification) canTransition(to NotificationState) bool {
	for _, allowed := range allowedNotificationTransitions[n.State] {
		if allowed == to {
			return true
		}
	}
	return false
}

func (n Notification) AuditPayload() []byte {
	parts := []string{n.ID, n.CandidateID, n.SolicitudID, n.Type, n.Subject, string(n.State)}
	for _, receipt := range n.Receipts {
		parts = append(parts, string(receipt.CSV), receipt.PayloadHash)
	}
	return []byte(strings.Join(parts, "\x00"))
}

var allowedNotificationTransitions = map[NotificationState][]NotificationState{
	NotificationStateCreada:  {NotificationStateEnviada, NotificationStateFallida, NotificationStateCancelada},
	NotificationStateEnviada: {NotificationStateLeida, NotificationStateFallida},
}
