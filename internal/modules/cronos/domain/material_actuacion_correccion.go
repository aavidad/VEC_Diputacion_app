package domain

import (
	"encoding/json"
	"strconv"
)

// MaterialBandejaCorrecciones liga una lectura al empleado propio del actor y
// al paso concreto. El empleado de cada solicitud se obtiene sólo en SQL.
type MaterialBandejaCorrecciones struct {
	ActorRef, PerfilRef, EmpleadoRef string
	Paso                             string
	ZonaHoraria                      string
}

func PasoBandejaCorreccionValido(p string) bool {
	return p == "responsable" || p == "administracion"
}

func (m MaterialBandejaCorrecciones) Validar() error {
	if !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) || !PasoBandejaCorreccionValido(m.Paso) {
		return ErrCorreccionInvalida
	}
	return nil
}

func (m MaterialBandejaCorrecciones) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrCorreccionInvalida
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"paso": m.Paso, "zona_horaria": m.ZonaHoraria,
	})
}

// MaterialActuacionCorreccion incluye motivo y versión en la huella V3. La
// referencia del empleado afectado nunca se recibe del cliente.
type MaterialActuacionCorreccion struct {
	ActorRef, PerfilRef, EmpleadoRef string
	SolicitudRef, ClaveOperacion     string
	Paso                             PasoCorreccion
	Resultado                        ResultadoCorreccion
	Motivo                           string
	VersionEsperada                  uint64
	ZonaHoraria                      string
}

func (m MaterialActuacionCorreccion) Validar() error {
	if !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) ||
		!referenciaSolicitudCorreccionValida(m.SolicitudRef) || !claveOperacionMarcaje(m.ClaveOperacion) ||
		!MotivoResolucionValido(m.Motivo) {
		return ErrCorreccionInvalida
	}
	switch m.Paso {
	case PasoDecisionResponsable:
		if m.VersionEsperada != 1 || !resultadoCorreccionValido(m.Resultado) {
			return ErrCorreccionInvalida
		}
	case PasoResolucionRRHH:
		if m.VersionEsperada != 2 || !resultadoCorreccionValido(m.Resultado) {
			return ErrCorreccionInvalida
		}
	case PasoAplicacion:
		if m.VersionEsperada != 3 || m.Resultado != "" {
			return ErrCorreccionInvalida
		}
	default:
		return ErrCorreccionInvalida
	}
	if m.Motivo == "" {
		return ErrCorreccionInvalida
	}
	return nil
}

func (m MaterialActuacionCorreccion) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrCorreccionInvalida
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"solicitud_ref": m.SolicitudRef, "clave_operacion": m.ClaveOperacion,
		"paso": string(m.Paso), "resultado": string(m.Resultado), "motivo": m.Motivo,
		"version_esperada": strconv.FormatUint(m.VersionEsperada, 10), "zona_horaria": m.ZonaHoraria,
	})
}

func (m MaterialActuacionCorreccion) PasoAmbito() string {
	if m.Paso == PasoDecisionResponsable {
		return "responsable"
	}
	return "administracion"
}
