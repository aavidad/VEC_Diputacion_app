package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrCandidateDocumentInvalid = errors.New("candidate document is invalid")

type DocumentPurpose string

const (
	DocumentPurposeSolicitud    DocumentPurpose = "Solicitud"
	DocumentPurposeSubsanacion  DocumentPurpose = "Subsanacion"
	DocumentPurposeAlegacion    DocumentPurpose = "Alegacion"
	DocumentPurposeResolucion   DocumentPurpose = "Resolucion"
	DocumentPurposeNotificacion DocumentPurpose = "Notificacion"
)

func (p DocumentPurpose) IsValid() bool {
	switch p {
	case DocumentPurposeSolicitud, DocumentPurposeSubsanacion, DocumentPurposeAlegacion,
		DocumentPurposeResolucion, DocumentPurposeNotificacion:
		return true
	default:
		return false
	}
}

type CandidateDocumentInput struct {
	ID           string
	CandidateID  string
	SolicitudID  string
	ProcedureID  string
	Purpose      DocumentPurpose
	Evidence     DocumentEvidence
	RegisteredBy string
	RegisteredAt time.Time
}

type CandidateDocument struct {
	ID           string
	CandidateID  string
	SolicitudID  string
	ProcedureID  string
	Purpose      DocumentPurpose
	Evidence     DocumentEvidence
	RegisteredBy string
	RegisteredAt time.Time
}

func NewCandidateDocument(input CandidateDocumentInput) (CandidateDocument, error) {
	doc := CandidateDocument{
		ID:           strings.TrimSpace(input.ID),
		CandidateID:  strings.TrimSpace(input.CandidateID),
		SolicitudID:  strings.TrimSpace(input.SolicitudID),
		ProcedureID:  strings.TrimSpace(input.ProcedureID),
		Purpose:      input.Purpose,
		Evidence:     input.Evidence,
		RegisteredBy: strings.TrimSpace(input.RegisteredBy),
		RegisteredAt: input.RegisteredAt.UTC(),
	}
	if err := doc.Validate(); err != nil {
		return CandidateDocument{}, err
	}
	return doc, nil
}

func (d CandidateDocument) Validate() error {
	switch {
	case strings.TrimSpace(d.ID) == "":
		return fmt.Errorf("%w: id is required", ErrCandidateDocumentInvalid)
	case strings.TrimSpace(d.CandidateID) == "":
		return fmt.Errorf("%w: candidate id is required", ErrCandidateDocumentInvalid)
	case strings.TrimSpace(d.SolicitudID) == "":
		return fmt.Errorf("%w: solicitud id is required", ErrCandidateDocumentInvalid)
	case strings.TrimSpace(d.ProcedureID) == "":
		return fmt.Errorf("%w: procedure id is required", ErrCandidateDocumentInvalid)
	case !d.Purpose.IsValid():
		return fmt.Errorf("%w: purpose %q", ErrCandidateDocumentInvalid, d.Purpose)
	case strings.TrimSpace(d.RegisteredBy) == "":
		return fmt.Errorf("%w: registered by is required", ErrCandidateDocumentInvalid)
	case d.RegisteredAt.IsZero():
		return fmt.Errorf("%w: registered at is required", ErrCandidateDocumentInvalid)
	default:
		return d.Evidence.Validate()
	}
}

func (d CandidateDocument) ExportManifestItem() (DocumentManifestItem, error) {
	if err := d.EnsureExportable(); err != nil {
		return DocumentManifestItem{}, err
	}
	return documentManifestItem(d.Evidence), nil
}

func (d CandidateDocument) EnsureExportable() error {
	if err := d.Validate(); err != nil {
		return err
	}
	return d.Evidence.EnsureExportable()
}

func (d CandidateDocument) AuditPayload() []byte {
	parts := []string{
		strings.TrimSpace(d.ID),
		strings.TrimSpace(d.CandidateID),
		strings.TrimSpace(d.SolicitudID),
		strings.TrimSpace(d.ProcedureID),
		string(d.Purpose),
		string(d.Evidence.CSV),
		d.Evidence.DigestSHA256,
		string(d.Evidence.AVStatus),
	}
	return []byte(strings.Join(parts, "\x00"))
}
