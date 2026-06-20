package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrAuditInvalid      = errors.New("audit entry is invalid")
	ErrAuditChainInvalid = errors.New("audit chain is invalid")
)

type AuditEnvelope struct {
	Actor      string
	Action     string
	OccurredAt time.Time
	Payload    []byte
}

type AuditEntry struct {
	Sequence      int
	OccurredAt    time.Time
	Actor         string
	Action        string
	PayloadHash   string
	PrevSignature string
	Signature     string
}

func NewAuditEntry(previous *AuditEntry, envelope AuditEnvelope, signingRef string) (AuditEntry, error) {
	if strings.TrimSpace(signingRef) == "" {
		return AuditEntry{}, fmt.Errorf("%w: signing ref is required", ErrAuditInvalid)
	}
	if err := envelope.Validate(); err != nil {
		return AuditEntry{}, err
	}
	entry := AuditEntry{
		Sequence:    1,
		OccurredAt:  envelope.OccurredAt.UTC(),
		Actor:       strings.TrimSpace(envelope.Actor),
		Action:      strings.TrimSpace(envelope.Action),
		PayloadHash: HashAuditPayload(envelope.Payload),
	}
	if previous != nil {
		if err := previous.Validate(); err != nil {
			return AuditEntry{}, err
		}
		entry.Sequence = previous.Sequence + 1
		entry.PrevSignature = previous.Signature
	}
	entry.Signature = signAuditEntry(entry, signingRef)
	return entry, nil
}

func (e AuditEnvelope) Validate() error {
	switch {
	case strings.TrimSpace(e.Actor) == "":
		return fmt.Errorf("%w: actor is required", ErrAuditInvalid)
	case strings.TrimSpace(e.Action) == "":
		return fmt.Errorf("%w: action is required", ErrAuditInvalid)
	case e.OccurredAt.IsZero():
		return fmt.Errorf("%w: occurred at is required", ErrAuditInvalid)
	case len(e.Payload) == 0:
		return fmt.Errorf("%w: payload is required", ErrAuditInvalid)
	default:
		return nil
	}
}

func (e AuditEntry) Validate() error {
	switch {
	case e.Sequence < 1:
		return fmt.Errorf("%w: sequence is required", ErrAuditInvalid)
	case strings.TrimSpace(e.Actor) == "":
		return fmt.Errorf("%w: actor is required", ErrAuditInvalid)
	case strings.TrimSpace(e.Action) == "":
		return fmt.Errorf("%w: action is required", ErrAuditInvalid)
	case strings.TrimSpace(e.PayloadHash) == "":
		return fmt.Errorf("%w: payload hash is required", ErrAuditInvalid)
	case strings.TrimSpace(e.Signature) == "":
		return fmt.Errorf("%w: signature is required", ErrAuditInvalid)
	case e.OccurredAt.IsZero():
		return fmt.Errorf("%w: occurred at is required", ErrAuditInvalid)
	default:
		return nil
	}
}

func VerifyAuditChain(entries []AuditEntry, signingRef string) error {
	if strings.TrimSpace(signingRef) == "" {
		return fmt.Errorf("%w: signing ref is required", ErrAuditChainInvalid)
	}
	if len(entries) == 0 {
		return fmt.Errorf("%w: entries are required", ErrAuditChainInvalid)
	}
	var previousSignature string
	for i, entry := range entries {
		if err := entry.Validate(); err != nil {
			return err
		}
		expectedSequence := i + 1
		if entry.Sequence != expectedSequence {
			return fmt.Errorf("%w: sequence %d", ErrAuditChainInvalid, entry.Sequence)
		}
		if entry.PrevSignature != previousSignature {
			return fmt.Errorf("%w: previous signature mismatch at %d", ErrAuditChainInvalid, entry.Sequence)
		}
		if expected := signAuditEntry(entry, signingRef); entry.Signature != expected {
			return fmt.Errorf("%w: signature mismatch at %d", ErrAuditChainInvalid, entry.Sequence)
		}
		previousSignature = entry.Signature
	}
	return nil
}

func HashAuditPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func signAuditEntry(entry AuditEntry, signingRef string) string {
	parts := []string{
		strconv.Itoa(entry.Sequence),
		entry.OccurredAt.UTC().Format(time.RFC3339Nano),
		strings.TrimSpace(entry.Actor),
		strings.TrimSpace(entry.Action),
		strings.TrimSpace(entry.PayloadHash),
		strings.TrimSpace(entry.PrevSignature),
		strings.TrimSpace(signingRef),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}
