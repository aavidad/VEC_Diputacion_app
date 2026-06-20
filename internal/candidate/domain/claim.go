package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrClaimInvalid    = errors.New("claim is invalid")
	ErrClaimTransition = errors.New("claim transition is invalid")
)

type ClaimState string

const (
	ClaimStatePresentada  ClaimState = "Presentada"
	ClaimStateEnRevision  ClaimState = "EnRevision"
	ClaimStateEstimada    ClaimState = "Estimada"
	ClaimStateDesestimada ClaimState = "Desestimada"
	ClaimStateArchivada   ClaimState = "Archivada"
)

func (s ClaimState) IsValid() bool {
	switch s {
	case ClaimStatePresentada, ClaimStateEnRevision, ClaimStateEstimada,
		ClaimStateDesestimada, ClaimStateArchivada:
		return true
	default:
		return false
	}
}

type ClaimReceipt struct {
	CSV         CSV
	Actor       string
	IssuedAt    time.Time
	PayloadHash string
}

func (r ClaimReceipt) Validate() error {
	switch {
	case r.CSV.Validate() != nil:
		return r.CSV.Validate()
	case strings.TrimSpace(r.Actor) == "":
		return fmt.Errorf("%w: receipt actor is required", ErrClaimInvalid)
	case r.IssuedAt.IsZero():
		return fmt.Errorf("%w: receipt issued at is required", ErrClaimInvalid)
	case strings.TrimSpace(r.PayloadHash) == "":
		return fmt.Errorf("%w: receipt payload hash is required", ErrClaimInvalid)
	default:
		return nil
	}
}

type ClaimInput struct {
	ID          string
	CandidateID string
	SolicitudID string
	Text        string
	Documents   []CandidateDocument
	PresentedBy string
	PresentedAt time.Time
	ReceiptCSV  string
}

type Claim struct {
	ID          string
	CandidateID string
	SolicitudID string
	Text        string
	Documents   []CandidateDocument
	State       ClaimState
	PresentedBy string
	PresentedAt time.Time
	Receipt     ClaimReceipt
}

func NewClaim(input ClaimInput) (Claim, error) {
	csv, err := NewCSV(input.ReceiptCSV)
	if err != nil {
		return Claim{}, err
	}
	claim := Claim{
		ID: strings.TrimSpace(input.ID), CandidateID: strings.TrimSpace(input.CandidateID),
		SolicitudID: strings.TrimSpace(input.SolicitudID), Text: strings.TrimSpace(input.Text),
		Documents: append([]CandidateDocument(nil), input.Documents...), State: ClaimStatePresentada,
		PresentedBy: strings.TrimSpace(input.PresentedBy), PresentedAt: input.PresentedAt.UTC(),
	}
	claim.Receipt = ClaimReceipt{
		CSV: csv, Actor: claim.PresentedBy, IssuedAt: claim.PresentedAt,
		PayloadHash: HashAuditPayload(claim.AuditPayload()),
	}
	if err := claim.Validate(); err != nil {
		return Claim{}, err
	}
	return claim, nil
}

func (c Claim) Validate() error {
	switch {
	case strings.TrimSpace(c.ID) == "":
		return fmt.Errorf("%w: id is required", ErrClaimInvalid)
	case strings.TrimSpace(c.CandidateID) == "":
		return fmt.Errorf("%w: candidate id is required", ErrClaimInvalid)
	case strings.TrimSpace(c.SolicitudID) == "":
		return fmt.Errorf("%w: solicitud id is required", ErrClaimInvalid)
	case strings.TrimSpace(c.Text) == "":
		return fmt.Errorf("%w: text is required", ErrClaimInvalid)
	case !c.State.IsValid():
		return fmt.Errorf("%w: state %q", ErrClaimInvalid, c.State)
	case strings.TrimSpace(c.PresentedBy) == "":
		return fmt.Errorf("%w: presented by is required", ErrClaimInvalid)
	case c.PresentedAt.IsZero():
		return fmt.Errorf("%w: presented at is required", ErrClaimInvalid)
	}
	for _, document := range c.Documents {
		if err := document.EnsureExportable(); err != nil {
			return err
		}
	}
	return c.Receipt.Validate()
}

func (c Claim) CanTransition(to ClaimState) bool {
	for _, allowed := range allowedClaimTransitions[c.State] {
		if allowed == to {
			return true
		}
	}
	return false
}

func (c *Claim) Transition(to ClaimState) error {
	if c == nil {
		return fmt.Errorf("%w: nil claim", ErrClaimTransition)
	}
	if !to.IsValid() {
		return fmt.Errorf("%w: target state %q", ErrClaimInvalid, to)
	}
	if !c.CanTransition(to) {
		return fmt.Errorf("%w: %s -> %s", ErrClaimTransition, c.State, to)
	}
	c.State = to
	return c.Validate()
}

func (c Claim) AuditPayload() []byte {
	parts := []string{c.ID, c.CandidateID, c.SolicitudID, c.Text, string(c.State), string(c.Receipt.CSV)}
	for _, document := range c.Documents {
		parts = append(parts, string(document.Evidence.CSV), document.Evidence.DigestSHA256)
	}
	return []byte(strings.Join(parts, "\x00"))
}

var allowedClaimTransitions = map[ClaimState][]ClaimState{
	ClaimStatePresentada: {ClaimStateEnRevision, ClaimStateArchivada},
	ClaimStateEnRevision: {ClaimStateEstimada, ClaimStateDesestimada, ClaimStateArchivada},
}
