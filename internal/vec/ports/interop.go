package ports

import (
	"context"
	"errors"
)

var ErrInteropNotEnabledV0 = errors.New("vec interop port not enabled v0")

type InteropRequest struct {
	Operation string            `json:"operation"`
	Subject   string            `json:"subject,omitempty"`
	Payload   map[string]string `json:"payload,omitempty"`
}

type InteropResult struct {
	Operation string            `json:"operation"`
	Reference string            `json:"reference,omitempty"`
	Status    string            `json:"status"`
	Payload   map[string]string `json:"payload,omitempty"`
}

type IdentityPort interface {
	Identify(context.Context, InteropRequest) (InteropResult, error)
}

type EUDIWalletPort interface {
	VerifyPresentation(context.Context, InteropRequest) (InteropResult, error)
}

type SignatureValidationPort interface {
	ValidateSignature(context.Context, InteropRequest) (InteropResult, error)
}

type TimestampPort interface {
	Timestamp(context.Context, InteropRequest) (InteropResult, error)
}

type DataIntermediationPort interface {
	QueryData(context.Context, InteropRequest) (InteropResult, error)
}

type RegistryInterconnectPort interface {
	ExchangeRegistryEntry(context.Context, InteropRequest) (InteropResult, error)
}

type CommonRegistryPort interface {
	RegisterEntry(context.Context, InteropRequest) (InteropResult, error)
}

type NotificationPort interface {
	SendNotification(context.Context, InteropRequest) (InteropResult, error)
}

type DocumentArchivePort interface {
	ArchiveDocument(context.Context, InteropRequest) (InteropResult, error)
}

type RepresentationPort interface {
	CheckRepresentation(context.Context, InteropRequest) (InteropResult, error)
}

type InvoicePort interface {
	SubmitInvoice(context.Context, InteropRequest) (InteropResult, error)
}

type OrgDirectoryPort interface {
	LookupOrgUnit(context.Context, InteropRequest) (InteropResult, error)
}

type SecureNetworkPort interface {
	CheckConnectivity(context.Context, InteropRequest) (InteropResult, error)
}
