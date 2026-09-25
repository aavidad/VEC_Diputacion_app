package domain

import (
	"encoding/json"
	"time"
)

// Zonas civiles admitidas por el libro de saldo (cronos_v1 000004).
const (
	ZonaSaldoPeninsula = "Europe/Madrid"
	ZonaSaldoCanarias  = "Atlantic/Canary"
)

// MaterialConsultaSaldoPropio es el material de LECTURA que V3 autoriza y
// que la función durable vuelve a comprobar: persona, perfil, empleado y el
// periodo civil exacto. Lo construye el servidor; nunca procede del cliente.
type MaterialConsultaSaldoPropio struct {
	ActorRef, PerfilRef, EmpleadoRef string
	Desde, Hasta, ZonaHoraria        string
}

func (m MaterialConsultaSaldoPropio) Validar() error {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") || !referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") ||
		(m.ZonaHoraria != ZonaSaldoPeninsula && m.ZonaHoraria != ZonaSaldoCanarias) {
		return ErrMarcajeInvalido
	}
	desde, err1 := time.Parse(time.DateOnly, m.Desde)
	hasta, err2 := time.Parse(time.DateOnly, m.Hasta)
	if err1 != nil || err2 != nil || desde.Format(time.DateOnly) != m.Desde || hasta.Format(time.DateOnly) != m.Hasta ||
		hasta.Before(desde) || hasta.Sub(desde) > 366*24*time.Hour {
		return ErrMarcajeInvalido
	}
	return nil
}

// Canonico produce los bytes que se firman y que SQL resume: JSON de claves
// ordenadas, igual que el resto del material de Cronos.
func (m MaterialConsultaSaldoPropio) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrMarcajeInvalido
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"desde": m.Desde, "hasta": m.Hasta, "zona_horaria": m.ZonaHoraria,
	})
}

// MaterialDisponibilidadMarcajeRemoto es LECTURA del estado propio del
// fichaje remoto en el instante del servidor. ClaveOperacion vacía consulta
// la disponibilidad; con clave concilia una operación ya registrada.
type MaterialDisponibilidadMarcajeRemoto struct {
	ActorRef, PerfilRef, EmpleadoRef, ClaveOperacion string
	InstanteUTC                                      time.Time
	Canal                                            AcreditacionCanalMarcaje
}

func (m MaterialDisponibilidadMarcajeRemoto) Validar() error {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") || !referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") ||
		(m.ClaveOperacion != "" && !claveOperacionMarcaje(m.ClaveOperacion)) ||
		m.InstanteUTC.IsZero() || m.InstanteUTC.Location() != time.UTC || m.InstanteUTC.Nanosecond()%1000 != 0 ||
		m.Canal.Validar() != nil || m.Canal.OrigenRef() != OrigenMarcajeRemoto {
		return ErrMarcajeInvalido
	}
	return nil
}

func (m MaterialDisponibilidadMarcajeRemoto) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrMarcajeInvalido
	}
	return json.Marshal(map[string]any{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"clave_operacion": m.ClaveOperacion, "instante_utc": m.InstanteUTC.Format("2006-01-02T15:04:05.000000Z07:00"),
		"canal": map[string]string{
			"politica_version_ref": m.Canal.PoliticaVersionRef(), "canal_ref": m.Canal.CanalRef(),
			"origen_ref": m.Canal.OrigenRef(), "calidad_ref": m.Canal.CalidadRef(),
		},
	})
}
