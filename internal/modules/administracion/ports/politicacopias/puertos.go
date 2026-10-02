// Package politicacopias provides transport-neutral contracts. Production
// adapters must reuse central ADMIN authorization and CS07's operation register.
package politicacopias

import (
	"context"
	"time"
	d "vec-diputacion-granada/internal/modules/administracion/domain/politicacopias"
)

type Registro struct {
	Version        uint64     `json:"version"`
	Politica       d.Politica `json:"politica"`
	SHA256         string     `json:"sha256"`
	Actor          string     `json:"actor"`
	Revisor        string     `json:"revisor,omitempty"`
	Correlacion    string     `json:"correlacion"`
	Instante       time.Time  `json:"instante"`
	AnteriorSHA256 string     `json:"anterior_sha256"`
	RegistroSHA256 string     `json:"registro_sha256"`
}

// Revalidacion is called under the repository's exclusive lock, immediately
// before persisting policy plus its append-only history. It must consume/recheck
// current central authority; a previous grant alone cannot authorize this write.
type Revalidacion func(context.Context) error
type Repositorio interface {
	Actual(context.Context) (Registro, error)
	Historia(context.Context) ([]Registro, error)
	Guardar(context.Context, uint64, d.Politica, Atribucion, time.Time, Revalidacion) (Registro, error)
}

// Versionador keeps the active policy unchanged while its scheduled reservation
// and execution are delegated. No platform call may reenter the policy repository.
type Versionador interface {
	ConVersion(context.Context, uint64, func(context.Context) error) error
}

type Intencion struct {
	Accion          string
	Politica        string
	Destino         string
	PoliticaSHA256  string
	VersionEsperada uint64
	DobleControl    bool
	ClaveEjecucion  string
}
type Atribucion struct {
	Actor       string
	Revisor     string
	Correlacion string
}

// Autoridad resolves trusted identities, exact scope and current obligations.
// Its production implementation must use the existing authorization service,
// not infer permissions from Politica, caller strings or configuration flags.
type Autoridad interface {
	Autorizar(context.Context, Intencion) (Atribucion, error)
	Revalidar(context.Context, Intencion, Atribucion) error
}
type Reloj interface{ Ahora() time.Time }
type Reserva struct {
	Clave          string         `json:"clave"`
	Politica       string         `json:"politica"`
	Version        uint64         `json:"version"`
	PoliticaSHA256 string         `json:"politica_sha256"`
	Destino        string         `json:"destino"`
	Evento         d.EventoAgenda `json:"evento"`
}
type ReciboReserva struct {
	Operacion  string `json:"operacion"`
	Replay     bool   `json:"replay"`
	AvisoFallo string `json:"aviso_fallo,omitempty"`
}

// Reservador is CS07's unique durable register through a bridge. It must also
// prevent overlapping captures per destination and policy across scheduled keys.
type Reservador interface {
	Reservar(context.Context, Reserva, Atribucion) (ReciboReserva, error)
}

// Ejecutor delegates to the real authorized backup application, with the same
// CS07 operation and idempotent key on retry. A successful reservation is never
// described as a verified backup. Reconciliation must not repeat capture blindly.
type Ejecutor interface {
	Ejecutar(context.Context, Reserva, ReciboReserva, Atribucion) error
}

// FalloAgenda contains opaque links only. It never carries provider error text.
type FalloAgenda struct {
	Clave, Operacion, Politica, Destino string
	Version                             uint64
	Motivo                              string
}

// NotificadorFallo forwards an operational incident. A nil error confirms only
// invocation/acceptance, never delivery. The operation owner must retain its
// failure and notification state in CS07's external audit/reconciliation register.
type NotificadorFallo interface {
	NotificarFallo(context.Context, FalloAgenda) error
}
