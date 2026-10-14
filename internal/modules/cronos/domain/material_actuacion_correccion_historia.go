package domain

import "encoding/json"

type MaterialHistoriaCorreccion struct {
	ActorRef, PerfilRef, EmpleadoRef string
	SolicitudRef                     string
	Vista                            string
	Paso                             string
	ZonaHoraria                      string
}

func (m MaterialHistoriaCorreccion) Validar() error {
	if !identidadEmpleadoValida(m.ActorRef, m.PerfilRef, m.EmpleadoRef, m.ZonaHoraria) ||
		!referenciaSolicitudCorreccionValida(m.SolicitudRef) ||
		!((m.Vista == "propia" && m.Paso == "") || (m.Vista == "equipo" && PasoBandejaCorreccionValido(m.Paso))) {
		return ErrCorreccionInvalida
	}
	return nil
}

func (m MaterialHistoriaCorreccion) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrCorreccionInvalida
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"solicitud_ref": m.SolicitudRef, "vista": m.Vista, "paso": m.Paso, "zona_horaria": m.ZonaHoraria,
	})
}
