package registrocopias

import (
	"context"
	"errors"
)

var (
	ErrOrdenConflicto = errors.New("registro_orden_conflicto")
	ErrFence          = errors.New("registro_fence_obsoleto")
)

// RecepcionOrden contains bindings from a trusted upstream verifier. The FS
// adapter stores them; it does not verify signatures, grants, validity or epoch.
type RecepcionOrden struct {
	Orden           string `json:"orden"`
	Operacion       string `json:"operacion"`
	SHA256          string `json:"sha256"`
	SolicitudSHA256 string `json:"solicitud_sha256"`
	Destino         string `json:"destino"`
	Epoca           string `json:"epoca"`
	Fence           uint64 `json:"fence"`
}
type Aceptacion struct {
	Orden     RecepcionOrden `json:"orden"`
	Recibo    Recibo         `json:"recibo"`
	Replay    bool           `json:"replay"`
	Auditoria Auditoria      `json:"auditoria"`
}
type AceptadorOrden interface {
	AceptarOrden(context.Context, Declaracion, RecepcionOrden) (Aceptacion, error)
}
