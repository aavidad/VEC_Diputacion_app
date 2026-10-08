package postgres

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

func prepararCapturaLecturaInscripcion(actor inscripcion.Actor, accion, recurso string, filtro inscripcion.Filtro) ([]byte, []byte, []byte, error) {
	if !actor.LecturaValida(accion, recurso, filtro) {
		return nil, nil, nil, inscripcion.ErrAccesoDenegado
	}
	vinculo, err := actor.Vinculo.Datos()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if err != nil || !actor.Vinculo.VigenteEn(ahora, actor.ResultadoContexto) ||
		actor.Lectura.Canal != string(vinculo.Superficie) ||
		actor.Lectura.AutenticacionRef != vinculo.AutenticacionRef ||
		actor.Lectura.SesionRef != vinculo.SesionRef ||
		actor.Lectura.CuentaRef != vinculo.CuentaRef ||
		actor.Lectura.Finalidad != finalidadLecturaInscripcion(accion) ||
		!canalLecturaInscripcion(accion, actor.Lectura.Canal) {
		return nil, nil, nil, inscripcion.ErrAccesoDenegado
	}
	contexto := bytes.Clone(actor.ResultadoContexto.RepresentacionCanonica)
	if len(contexto) == 0 {
		return nil, nil, nil, inscripcion.ErrAccesoDenegado
	}
	vinculoJSON, err := json.Marshal(actor.Vinculo)
	if err != nil {
		return nil, nil, nil, inscripcion.ErrNoDisponible
	}
	var intento [16]byte
	if _, err := rand.Read(intento[:]); err != nil {
		return nil, nil, nil, inscripcion.ErrNoDisponible
	}
	c := actor.Lectura
	capturaJSON, err := json.Marshal(struct {
		IntentoRef              string `json:"intento_ref"`
		PersonaRef              string `json:"persona_ref"`
		PerfilRef               string `json:"perfil_ref"`
		CuentaRef               string `json:"cuenta_ref"`
		SesionRef               string `json:"sesion_ref"`
		AutenticacionRef        string `json:"autenticacion_ref"`
		CertificadoHuellaSHA256 string `json:"certificado_huella_sha256"`
		Canal                   string `json:"canal"`
		Accion                  string `json:"accion"`
		RecursoRef              string `json:"recurso_ref"`
		Finalidad               string `json:"finalidad"`
		CorrelacionRef          string `json:"correlacion_ref"`
		RevisionPermisos        uint64 `json:"revision_permisos"`
		HuellaInstantaneaSHA256 string `json:"huella_instantanea_sha256"`
		Filtro                  struct {
			Estado          string `json:"estado"`
			ConvocatoriaRef string `json:"convocatoria_ref"`
			Limite          int    `json:"limite"`
			Cursor          string `json:"cursor"`
		} `json:"filtro"`
		Idioma      string    `json:"idioma"`
		EmitidaEn   time.Time `json:"emitida_en"`
		ValidaHasta time.Time `json:"valida_hasta"`
	}{
		IntentoRef: "lectura_" + hex.EncodeToString(intento[:]),
		PersonaRef: c.PersonaRef, PerfilRef: c.PerfilRef, CuentaRef: c.CuentaRef,
		SesionRef: c.SesionRef, AutenticacionRef: c.AutenticacionRef,
		CertificadoHuellaSHA256: c.CertificadoHuellaSHA256, Canal: c.Canal,
		Accion: c.Accion, RecursoRef: c.RecursoRef, Finalidad: c.Finalidad,
		CorrelacionRef: c.CorrelacionRef, RevisionPermisos: c.RevisionPermisos,
		HuellaInstantaneaSHA256: c.HuellaInstantaneaSHA256,
		Filtro: struct {
			Estado          string `json:"estado"`
			ConvocatoriaRef string `json:"convocatoria_ref"`
			Limite          int    `json:"limite"`
			Cursor          string `json:"cursor"`
		}{c.Filtro.Estado, c.Filtro.ConvocatoriaRef, c.Filtro.Limite, c.Filtro.Cursor},
		Idioma: actor.Idioma, EmitidaEn: c.EmitidaEn, ValidaHasta: c.ValidaHasta,
	})
	if err != nil {
		return nil, nil, nil, inscripcion.ErrNoDisponible
	}
	return contexto, vinculoJSON, capturaJSON, nil
}

func finalidadLecturaInscripcion(accion string) string {
	switch accion {
	case inscripcion.AccionListarAbiertas, inscripcion.AccionDetalleAbierta:
		return "consulta_convocatoria_abierta"
	case inscripcion.AccionListarPropias, inscripcion.AccionDetallePropia:
		return "consulta_inscripcion_propia"
	case inscripcion.AccionListarRRHH, inscripcion.AccionDetalleRRHH:
		return "consulta_inscripcion_rrhh"
	case inscripcion.AccionMotivosRRHH:
		return "consulta_motivos_inscripcion_rrhh"
	default:
		return ""
	}
}

func canalLecturaInscripcion(accion, canal string) bool {
	switch accion {
	case inscripcion.AccionListarAbiertas, inscripcion.AccionDetalleAbierta,
		inscripcion.AccionListarPropias, inscripcion.AccionDetallePropia:
		return canal == "externa_personal" || canal == "interna_corporativa"
	case inscripcion.AccionListarRRHH, inscripcion.AccionDetalleRRHH, inscripcion.AccionMotivosRRHH:
		return canal == "interna_corporativa"
	default:
		return false
	}
}
