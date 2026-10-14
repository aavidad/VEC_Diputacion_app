package domain

import "encoding/json"

// MaterialRecuperacionCorreccionPropia identifica el recibo inicial propio.
// El actor, perfil y empleado los aporta el servidor.
type MaterialRecuperacionCorreccionPropia struct {
	ActorRef, PerfilRef, EmpleadoRef string
	SolicitudRef, ClaveOperacion     string
}

func (m MaterialRecuperacionCorreccionPropia) Validar() error {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") ||
		!referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") ||
		!claveOperacionMarcaje(m.ClaveOperacion) ||
		m.SolicitudRef != "correccion:cronos:"+m.ClaveOperacion {
		return ErrCorreccionInvalida
	}
	return nil
}

func (m MaterialRecuperacionCorreccionPropia) Canonico() ([]byte, error) {
	if err := m.Validar(); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef,
		"empleado_ref": m.EmpleadoRef, "clave_operacion": m.ClaveOperacion,
		"solicitud_ref": m.SolicitudRef, "paso": string(PasoSolicitudCorreccion),
		"version_recibo": "1",
	})
}
