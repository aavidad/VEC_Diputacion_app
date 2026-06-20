package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrDocumentInvalid     = errors.New("document evidence is invalid")
	ErrDocumentQuarantined = errors.New("document evidence is quarantined")
	ErrFileInvalid         = errors.New("electronic file is invalid")
)

type AVStatus string

const (
	AVStatusPending AVStatus = "PENDING"
	AVStatusClean   AVStatus = "CLEAN"
	AVStatusThreat  AVStatus = "THREAT"
)

func (s AVStatus) IsValid() bool {
	switch s {
	case AVStatusPending, AVStatusClean, AVStatusThreat:
		return true
	default:
		return false
	}
}

type CSV string

func NewCSV(value string) (CSV, error) {
	csv := CSV(strings.TrimSpace(value))
	if err := csv.Validate(); err != nil {
		return "", err
	}
	return csv, nil
}

func (c CSV) Validate() error {
	if strings.TrimSpace(string(c)) == "" {
		return fmt.Errorf("%w: csv is required", ErrDocumentInvalid)
	}
	return nil
}

type ENIMetadata struct {
	DocumentID   string
	Organ        string
	Procedure    string
	DocumentType string
	Title        string
	Format       string
	Language     string
	CreatedAt    time.Time
}

func (m ENIMetadata) Validate() error {
	switch {
	case strings.TrimSpace(m.DocumentID) == "":
		return fmt.Errorf("%w: eni document id is required", ErrDocumentInvalid)
	case strings.TrimSpace(m.Organ) == "":
		return fmt.Errorf("%w: eni organ is required", ErrDocumentInvalid)
	case strings.TrimSpace(m.Procedure) == "":
		return fmt.Errorf("%w: eni procedure is required", ErrDocumentInvalid)
	case strings.TrimSpace(m.DocumentType) == "":
		return fmt.Errorf("%w: eni document type is required", ErrDocumentInvalid)
	case strings.TrimSpace(m.Title) == "":
		return fmt.Errorf("%w: eni title is required", ErrDocumentInvalid)
	case strings.TrimSpace(m.Format) == "":
		return fmt.Errorf("%w: eni format is required", ErrDocumentInvalid)
	case m.CreatedAt.IsZero():
		return fmt.Errorf("%w: eni created at is required", ErrDocumentInvalid)
	default:
		return nil
	}
}

type DocumentExternalRefs struct {
	// Opaque refs resolved by adapters such as MinIO, OpenBao or TSA clients.
	StorageObjectRef string
	VaultSecretRef   string
	TSAStampRef      string
}

func (r DocumentExternalRefs) Validate() error {
	if strings.TrimSpace(r.StorageObjectRef) == "" {
		return fmt.Errorf("%w: storage object ref is required", ErrDocumentInvalid)
	}
	return nil
}

type SignatureEvidence struct {
	SignerID     string
	SignatureRef string
	SignedAt     time.Time
}

func (s SignatureEvidence) Validate() error {
	switch {
	case strings.TrimSpace(s.SignerID) == "":
		return fmt.Errorf("%w: signer id is required", ErrDocumentInvalid)
	case strings.TrimSpace(s.SignatureRef) == "":
		return fmt.Errorf("%w: signature ref is required", ErrDocumentInvalid)
	case s.SignedAt.IsZero():
		return fmt.Errorf("%w: signed at is required", ErrDocumentInvalid)
	default:
		return nil
	}
}

type DocumentEvidenceInput struct {
	CSV          string
	DigestSHA256 string
	Refs         DocumentExternalRefs
	ENI          ENIMetadata
	AVStatus     AVStatus
	SubmittedBy  string
	SubmittedAt  time.Time
	Signatures   []SignatureEvidence
}

type DocumentEvidence struct {
	CSV          CSV
	DigestSHA256 string
	Refs         DocumentExternalRefs
	ENI          ENIMetadata
	AVStatus     AVStatus
	SubmittedBy  string
	SubmittedAt  time.Time
	Signatures   []SignatureEvidence
}

func NewDocumentEvidence(input DocumentEvidenceInput) (DocumentEvidence, error) {
	csv, err := NewCSV(input.CSV)
	if err != nil {
		return DocumentEvidence{}, err
	}
	status := input.AVStatus
	if status == "" {
		status = AVStatusPending
	}
	doc := DocumentEvidence{
		CSV:          csv,
		DigestSHA256: strings.TrimSpace(input.DigestSHA256),
		Refs:         input.Refs,
		ENI:          input.ENI,
		AVStatus:     status,
		SubmittedBy:  strings.TrimSpace(input.SubmittedBy),
		SubmittedAt:  input.SubmittedAt,
		Signatures:   append([]SignatureEvidence(nil), input.Signatures...),
	}
	if err := doc.Validate(); err != nil {
		return DocumentEvidence{}, err
	}
	return doc, nil
}

func (d DocumentEvidence) Validate() error {
	switch {
	case d.CSV.Validate() != nil:
		return d.CSV.Validate()
	case strings.TrimSpace(d.DigestSHA256) == "":
		return fmt.Errorf("%w: sha256 digest is required", ErrDocumentInvalid)
	case !d.AVStatus.IsValid():
		return fmt.Errorf("%w: av status %q", ErrDocumentInvalid, d.AVStatus)
	case strings.TrimSpace(d.SubmittedBy) == "":
		return fmt.Errorf("%w: submitted by is required", ErrDocumentInvalid)
	case d.SubmittedAt.IsZero():
		return fmt.Errorf("%w: submitted at is required", ErrDocumentInvalid)
	}
	if err := d.Refs.Validate(); err != nil {
		return err
	}
	if err := d.ENI.Validate(); err != nil {
		return err
	}
	for _, signature := range d.Signatures {
		if err := signature.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (d DocumentEvidence) IsQuarantined() bool {
	return d.AVStatus != AVStatusClean
}

func (d DocumentEvidence) EnsureExportable() error {
	if err := d.Validate(); err != nil {
		return err
	}
	if d.IsQuarantined() {
		return fmt.Errorf("%w: av status %s", ErrDocumentQuarantined, d.AVStatus)
	}
	return nil
}

func (d *DocumentEvidence) MarkAVStatus(status AVStatus) error {
	if d == nil {
		return fmt.Errorf("%w: nil document", ErrDocumentInvalid)
	}
	if !status.IsValid() {
		return fmt.Errorf("%w: av status %q", ErrDocumentInvalid, status)
	}
	d.AVStatus = status
	return nil
}

type ElectronicFile struct {
	ID          string
	CandidateID string
	ProcedureID string
	CreatedBy   string
	CreatedAt   time.Time
	Documents   []DocumentEvidence
}

type ElectronicFileManifest struct {
	FileID      string
	CandidateID string
	ProcedureID string
	Items       []DocumentManifestItem
}

type DocumentManifestItem struct {
	CSV           CSV
	Who           string
	What          string
	When          time.Time
	SignatureRefs []string
	TSAStampRef   string
}

func (f ElectronicFile) ExportManifest() (ElectronicFileManifest, error) {
	if err := f.Validate(); err != nil {
		return ElectronicFileManifest{}, err
	}
	items := make([]DocumentManifestItem, 0, len(f.Documents))
	for _, doc := range f.Documents {
		if err := doc.EnsureExportable(); err != nil {
			return ElectronicFileManifest{}, err
		}
		items = append(items, documentManifestItem(doc))
	}
	return ElectronicFileManifest{
		FileID:      strings.TrimSpace(f.ID),
		CandidateID: strings.TrimSpace(f.CandidateID),
		ProcedureID: strings.TrimSpace(f.ProcedureID),
		Items:       items,
	}, nil
}

func (f ElectronicFile) Validate() error {
	switch {
	case strings.TrimSpace(f.ID) == "":
		return fmt.Errorf("%w: file id is required", ErrFileInvalid)
	case strings.TrimSpace(f.CandidateID) == "":
		return fmt.Errorf("%w: candidate id is required", ErrFileInvalid)
	case strings.TrimSpace(f.ProcedureID) == "":
		return fmt.Errorf("%w: procedure id is required", ErrFileInvalid)
	case strings.TrimSpace(f.CreatedBy) == "":
		return fmt.Errorf("%w: created by is required", ErrFileInvalid)
	case f.CreatedAt.IsZero():
		return fmt.Errorf("%w: created at is required", ErrFileInvalid)
	case len(f.Documents) == 0:
		return fmt.Errorf("%w: documents are required", ErrFileInvalid)
	default:
		return nil
	}
}

func documentManifestItem(doc DocumentEvidence) DocumentManifestItem {
	signatureRefs := make([]string, 0, len(doc.Signatures))
	for _, signature := range doc.Signatures {
		signatureRefs = append(signatureRefs, signature.SignatureRef)
	}
	return DocumentManifestItem{
		CSV:           doc.CSV,
		Who:           doc.SubmittedBy,
		What:          doc.ENI.Title,
		When:          doc.SubmittedAt,
		SignatureRefs: signatureRefs,
		TSAStampRef:   doc.Refs.TSAStampRef,
	}
}
