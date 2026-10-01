package registrocopias

import (
	"context"
	"errors"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
)

var ErrAbandonoNoAutorizado = errors.New("registro_abandono_no_autorizado")

type SolicitudAbandono struct {
	Operacion       string `json:"operacion"`
	Clave           string `json:"clave"`
	VersionEsperada uint64 `json:"version_esperada"`
	SolicitudSHA256 string `json:"solicitud_sha256"`
	Destino         string `json:"destino"`
	FalloReferencia string `json:"fallo_referencia"`
	FalloSHA256     string `json:"fallo_sha256"`
}

// ObservadorAbandono must revalidate the real current actor/authority and observe
// the executor's stopped effect, cancelled lease and absence of pending platform
// writes, maintenance or restoration effects, including on a replay.
// It is wired by trusted composition, never chosen by a JSON caller or CLI.
type ObservadorAbandono interface {
	ConfirmarAbandono(context.Context, Declaracion, SolicitudAbandono) (operacionescopias.ObservacionAbandono, error)
}

type AbandonadorCaptura interface {
	AbandonarCaptura(context.Context, Declaracion, SolicitudAbandono) (Resultado, error)
}
