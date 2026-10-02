package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// La fuente resuelve empleado_ref; la petición sólo identifica la solicitud.
type MaterialConsultaJustificacion struct {
	ActorRef     string `json:"actor_ref"`
	PerfilRef    string `json:"perfil_ref"`
	EmpleadoRef  string `json:"empleado_ref"`
	SolicitudRef string `json:"solicitud_ref"`
}

func (m MaterialConsultaJustificacion) Canonico() ([]byte, error) {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") || !referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") || !SolicitudPermisoRefValida(m.SolicitudRef) {
		return nil, ErrJustificacionInvalida
	}
	return json.Marshal(m)
}

// Recuperar usa la clave y huella de la operación original, con actor actual.
type MaterialReciboJustificacion struct {
	ActorRef       string `json:"actor_ref"`
	PerfilRef      string `json:"perfil_ref"`
	EmpleadoRef    string `json:"empleado_ref"`
	SolicitudRef   string `json:"solicitud_ref"`
	ClaveOperacion string `json:"clave_operacion"`
	HuellaMaterial string `json:"huella_material"`
}

// La consulta por clave recupera el material original de una revisión. El
// servidor coteja después los campos del POST; la clave sola no autoriza nada.
type MaterialReciboPorClaveJustificacion struct {
	ActorRef       string `json:"actor_ref"`
	PerfilRef      string `json:"perfil_ref"`
	EmpleadoRef    string `json:"empleado_ref"`
	SolicitudRef   string `json:"solicitud_ref"`
	ClaveOperacion string `json:"clave_operacion"`
}

func (m MaterialReciboPorClaveJustificacion) Canonico() ([]byte, error) {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") || !referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") || !SolicitudPermisoRefValida(m.SolicitudRef) ||
		!RefDocumentoJustificacionValida(m.ClaveOperacion) {
		return nil, ErrJustificacionInvalida
	}
	return json.Marshal(m)
}

func (m MaterialReciboJustificacion) Canonico() ([]byte, error) {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") || !referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") || !SolicitudPermisoRefValida(m.SolicitudRef) ||
		!RefDocumentoJustificacionValida(m.ClaveOperacion) || !HuellaEfectosValida(m.HuellaMaterial) {
		return nil, ErrJustificacionInvalida
	}
	return json.Marshal(m)
}

func HuellaMaterialLecturaJustificacion(canonico []byte) string {
	h := sha256.Sum256(canonico)
	return hex.EncodeToString(h[:])
}
