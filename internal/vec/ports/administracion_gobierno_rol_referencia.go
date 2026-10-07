package ports

import (
	"context"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// ErrGobiernoRolIntentoAuditado sólo marca un fallo cuya auditoría de intento
// AUT60 ya confirmó en la misma transacción. La frontera HTTP registra los
// demás fallos; no infiere auditoría de un SQLSTATE ni del PDP en memoria.
var ErrGobiernoRolIntentoAuditado = errors.New("gobierno_rol_intento_auditado")

// ResultadoPropuestaGobiernoRolNuevo separa la caducidad para un primer acto
// de la recuperación de una propuesta histórica. Sólo la autoridad SQL puede
// declarar Replay tras cotejar operación/material y auditar el acceso actual.
type ResultadoPropuestaGobiernoRolNuevo struct {
	Propuesta          domain.PropuestaGobiernoPerfil
	Replay             bool
	AuditoriaAccesoRef string
}

func (r ResultadoPropuestaGobiernoRolNuevo) ValidarPara(
	o domain.OrdenPropuestaGobiernoPerfil, ahora time.Time) error {
	if r.Propuesta.ValidarPara(o) != nil || !referenciaAuditoriaGobiernoRol(r.AuditoriaAccesoRef) ||
		(!r.Replay && !r.Propuesta.CaducaEn.After(ahora)) {
		return domain.ErrPlanGobiernoPerfilInvalido
	}
	return nil
}

func referenciaAuditoriaGobiernoRol(ref string) bool {
	const prefijo = "aud_v3_"
	if !strings.HasPrefix(ref, prefijo) || len(ref) != len(prefijo)+32 {
		return false
	}
	for _, c := range ref[len(prefijo):] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

type AutoridadPropuestaGobiernoRolNuevoRecuperable interface {
	ProponerGobiernoRolNuevoRecuperable(context.Context,
		domain.OrdenPropuestaGobiernoPerfil) (ResultadoPropuestaGobiernoRolNuevo, error)
}

// AutoridadCierreGobiernoRolPorReferencia conserva el cierre en la autoridad
// central. Debe recuperar la propuesta original dentro de la misma transacción
// que consume V3, comprueba dos ADMIN distintos y publica el rol. Ninguna
// consulta previa ni campo del cliente sustituye esa recuperación.
type AutoridadCierreGobiernoRolPorReferencia interface {
	CerrarGobiernoRolPorReferencia(context.Context,
		domain.SolicitudCierreGobiernoRolPorReferencia) (domain.CierreGobiernoPerfil, error)
}
