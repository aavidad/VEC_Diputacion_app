package postgres

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type objetivoJSON struct {
	CuentaRef               string    `json:"cuenta_ref"`
	CuentaVersion           uint64    `json:"cuenta_version"`
	PersonaRef              string    `json:"persona_ref"`
	PersonaVersion          uint64    `json:"persona_version"`
	PerfilRef               string    `json:"perfil_ref"`
	PerfilVersion           uint64    `json:"perfil_version"`
	VinculoRef              string    `json:"vinculo_ref"`
	VinculoVersion          uint64    `json:"vinculo_version"`
	HuellaSHA256            string    `json:"huella_sha256"`
	RevisionContinuidad     uint64    `json:"revision_continuidad"`
	ProcedenciaRef          string    `json:"procedencia_ref"`
	ProcedenciaVersion      uint64    `json:"procedencia_version"`
	ProcedenciaHuellaSHA256 string    `json:"procedencia_huella_sha256"`
	VigenteHasta            time.Time `json:"vigente_hasta"`
}

type actoJSON struct {
	Esquema         string                                 `json:"esquema"`
	OperacionRef    string                                 `json:"operacion_ref"`
	Operacion       domain.OperacionAdministracionPerfiles `json:"operacion"`
	RolVersionRef   string                                 `json:"rol_version_ref"`
	RolHuellaSHA256 string                                 `json:"rol_huella_sha256"`
	ActorPersonaRef string                                 `json:"actor_persona_ref"`
	ActorPerfilRef  string                                 `json:"actor_perfil_ref"`
	Objetivo        objetivoJSON                           `json:"objetivo"`
	UnidadRef       string                                 `json:"unidad_ref,omitempty"`
	ReferenciaActo  string                                 `json:"referencia_acto,omitempty"`
	Motivo          domain.ReferenciaEntradaCatalogo       `json:"motivo"`
	CorrelacionRef  string                                 `json:"correlacion_ref"`
}

type cierreJSON struct {
	Esquema               string                                         `json:"esquema"`
	OperacionRef          string                                         `json:"operacion_ref"`
	PropuestaRef          string                                         `json:"propuesta_ref"`
	PropuestaHuellaSHA256 string                                         `json:"propuesta_huella_sha256"`
	Decision              domain.DecisionPropuestaAdministracionPerfiles `json:"decision"`
	ActorPersonaRef       string                                         `json:"actor_persona_ref"`
	ActorPerfilRef        string                                         `json:"actor_perfil_ref"`
	Motivo                domain.ReferenciaEntradaCatalogo               `json:"motivo"`
	CorrelacionRef        string                                         `json:"correlacion_ref"`
}

func materialActo(s domain.SolicitudActoAdministracionPerfiles, rol ports.RolAdministrable, propuesta bool) (Efecto, error) {
	p := s.Objetivo
	x := actoJSON{
		Esquema: "administracion_perfiles_acto_v1", OperacionRef: s.OperacionRef,
		Operacion: s.Operacion, RolVersionRef: rol.VersionRef, RolHuellaSHA256: rol.HuellaSHA256,
		ActorPersonaRef: s.Actor.PersonaRef, ActorPerfilRef: s.Actor.PerfilActivoRef,
		Objetivo: objetivoJSON{p.CuentaRef, p.CuentaVersion, p.PersonaRef, p.PersonaVersion,
			p.PerfilRef, p.PerfilVersion, p.VinculoRef, p.VinculoVersion, p.HuellaSHA256,
			p.RevisionContinuidad, p.ProcedenciaRef, p.ProcedenciaVersion, p.ProcedenciaHuellaSHA256, p.VigenteHasta},
		UnidadRef: p.UnidadRef, ReferenciaActo: s.ReferenciaActo, Motivo: s.Motivo, CorrelacionRef: s.CorrelacionRef,
	}
	b, err := json.Marshal(x)
	if err != nil {
		return Efecto{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	accion, audiencia := "administracion.perfiles."+string(s.Operacion), "vec_autorizacion.administracion_perfiles.ordinario.v1"
	if propuesta {
		accion, audiencia = "administracion.perfiles.proponer", "vec_autorizacion.administracion_perfiles.propuesta.v1"
	}
	return Efecto{Accion: accion, Audiencia: audiencia, Referencia: s.OperacionRef, Material: b}, nil
}

func materialCierre(s domain.SolicitudCierrePropuestaAdministracionPerfiles) (Efecto, error) {
	x := cierreJSON{Esquema: "administracion_perfiles_cierre_v1", OperacionRef: s.OperacionRef,
		PropuestaRef: s.PropuestaRef, PropuestaHuellaSHA256: s.PropuestaHuellaSHA256, Decision: s.Decision,
		ActorPersonaRef: s.Aprobador.PersonaRef, ActorPerfilRef: s.Aprobador.PerfilActivoRef,
		Motivo: s.Motivo, CorrelacionRef: s.CorrelacionRef}
	b, err := json.Marshal(x)
	if err != nil {
		return Efecto{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	accion := "administracion.perfiles.aprobar"
	if s.Decision == domain.DecisionRechazarPropuestaPerfil {
		accion = "administracion.perfiles.rechazar"
	}
	return Efecto{Accion: accion, Audiencia: "vec_autorizacion.administracion_perfiles.cierre.v1", Referencia: s.OperacionRef, Material: b}, nil
}

func decodificar(b []byte, destino any) error {
	if len(b) == 0 || len(b) > 65536 {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	objeto := bytes.TrimSpace(b)
	if len(objeto) == 0 || objeto[0] != '{' {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nil
}
