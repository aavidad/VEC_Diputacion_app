package domain

import (
	"encoding/json"
	"time"
)

// OrigenMarcajeRemoto identifica el origen que debe acreditar la política de canal.
// No se acepta como dato enviado por el navegador.
const OrigenMarcajeRemoto = "remoto"

// PeriodoTeletrabajo es una vigencia de instantes UTC [desde, hasta).
// La autorización de escritura se revalida dentro de la transacción durable.
type PeriodoTeletrabajo struct {
	DesdeUTC time.Time
	HastaUTC time.Time
}

func (p PeriodoTeletrabajo) Validar() error {
	if p.DesdeUTC.IsZero() || p.HastaUTC.IsZero() ||
		p.DesdeUTC.Location() != time.UTC || p.HastaUTC.Location() != time.UTC ||
		p.DesdeUTC.Nanosecond()%1000 != 0 || p.HastaUTC.Nanosecond()%1000 != 0 ||
		!p.DesdeUTC.Before(p.HastaUTC) {
		return ErrMarcajeInvalido
	}
	return nil
}

func (p PeriodoTeletrabajo) Contiene(instanteUTC time.Time) bool {
	return p.Validar() == nil && instanteUTC.Location() == time.UTC &&
		!instanteUTC.Before(p.DesdeUTC) && instanteUTC.Before(p.HastaUTC)
}

// ValidarMovimientosRemotosPermitidos valida una proyección nominal del
// repositorio. La secuencia real se revalida al registrar bajo el mismo lock.
func ValidarMovimientosRemotosPermitidos(movimientos []PunchKind) error {
	if len(movimientos) > 4 {
		return ErrMarcajeInvalido
	}
	vistos := make(map[PunchKind]bool, len(movimientos))
	for _, movimiento := range movimientos {
		switch movimiento {
		case PunchEntry, PunchExit, PunchPauseStart, PunchPauseEnd:
		default:
			return ErrMarcajeInvalido
		}
		if vistos[movimiento] {
			return ErrMarcajeInvalido
		}
		vistos[movimiento] = true
	}
	return nil
}

// MaterialRecuperacionMarcajeRemoto es material de LECTURA. PerfilRef y canal
// son los actuales para la concesión; el repositorio compara con el original
// actor, empleado, clave, movimiento y origen remoto, no su hora de registro.
type MaterialRecuperacionMarcajeRemoto struct {
	ActorRef, PerfilRef, EmpleadoRef, ClaveOperacion string
	Movimiento                                       PunchKind
	Canal                                            AcreditacionCanalMarcaje
}

func (m MaterialRecuperacionMarcajeRemoto) Validar() error {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") ||
		!referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") ||
		!claveOperacionMarcaje(m.ClaveOperacion) ||
		m.Canal.Validar() != nil || m.Canal.OrigenRef() != OrigenMarcajeRemoto {
		return ErrMarcajeInvalido
	}
	switch m.Movimiento {
	case PunchEntry, PunchExit, PunchPauseStart, PunchPauseEnd:
		return nil
	default:
		return ErrMarcajeInvalido
	}
}

func (m MaterialRecuperacionMarcajeRemoto) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrMarcajeInvalido
	}
	return json.Marshal(map[string]any{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"clave_operacion": m.ClaveOperacion, "movimiento": m.Movimiento,
		"canal": map[string]string{
			"politica_version_ref": m.Canal.PoliticaVersionRef(), "canal_ref": m.Canal.CanalRef(),
			"origen_ref": m.Canal.OrigenRef(), "calidad_ref": m.Canal.CalidadRef(),
		},
	})
}
