package domain

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

var (
	ErrSolicitudAsignacionCompetencialV1Invalida      = errors.New("vec: solicitud de asignacion competencial v1 invalida")
	ErrEvidenciaAsignacionCompetencialV1Invalida      = errors.New("vec: evidencia de asignacion competencial v1 invalida")
	ErrSerializacionAsignacionCompetencialV1Prohibida = errors.New("vec: serializacion de asignacion competencial v1 prohibida")
)

// ReferenciaCargoCompetencialV1 identifica una version acreditada por Personal.
// No es el nombre del cargo, un rol de autenticacion ni una persona. La fuente
// conserva la relacion nominal entre esta version, la persona y su vigencia.
type ReferenciaCargoCompetencialV1 struct {
	Referencia   string
	Version      uint64
	HuellaSHA256 string
}

func (r ReferenciaCargoCompetencialV1) Validar() error {
	if !textoAutorizacionSinComodinSeguro(r.Referencia, 512, false) || r.Version == 0 ||
		!huellaAsignacionCompetencialV1Valida(r.HuellaSHA256) {
		return ErrSolicitudAsignacionCompetencialV1Invalida
	}
	return nil
}

// SolicitudAsignacionCompetencialV1 es interna. Actor y su evidencia registrada
// corresponden al consultante; PersonaRef identifica al firmante, resuelto por
// la autoridad de identidad desde el certificado verificado. No se interpreta
// un cargo, una cabecera HTTP o un nombre como identificador de persona.
// Recurso y Cargo proceden de sus autoridades, nunca de atributos libres HTTP.
type SolicitudAsignacionCompetencialV1 struct {
	bloqueoSerializacionAsignacionesCompetencialesV1
	Actor                                                     ContextoActor                      `json:"-"`
	ResultadoContexto                                         ResultadoContextoActorRegistradoV2 `json:"-"`
	Vinculo                                                   VinculoAutenticacionActorV2        `json:"-"`
	PersonaRef, PerfilFirmanteRef, CertificadoHuellaSHA256    string
	Cargo                                                     ReferenciaCargoCompetencialV1
	Recurso                                                   RecursoAutorizable
	AccionLectura, FinalidadLectura                           string
	AccionCompetencial, FinalidadCompetencial, CorrelacionRef string
	Motivo                                                    ReferenciaEntradaCatalogo
}

// ValidarEn comprueba coherencia local, no la existencia ni revocacion actual
// de las fuentes. El lector exige su autorizacion central y audita la lectura.
func (s SolicitudAsignacionCompetencialV1) ValidarEn(instante time.Time) error {
	if !instanteAutorizacionCanonico(instante) || s.Actor.Validar() != nil ||
		!s.Vinculo.VigenteEn(instante, s.ResultadoContexto) ||
		!referenciaOpacaContextoActorValida(s.PersonaRef, "per_") ||
		!textoAutorizacionSinComodinSeguro(s.PerfilFirmanteRef, 512, false) ||
		!huellaAsignacionCompetencialV1Valida(s.CertificadoHuellaSHA256) || s.Cargo.Validar() != nil ||
		s.Recurso.Validar() != nil || len(s.Recurso.Ambitos) == 0 ||
		!textoAutorizacionSinComodinSeguro(s.AccionCompetencial, 256, false) ||
		!textoAutorizacionSinComodinSeguro(s.FinalidadCompetencial, 512, false) ||
		!textoAutorizacionSinComodinSeguro(s.AccionLectura, 256, false) ||
		!textoAutorizacionSinComodinSeguro(s.FinalidadLectura, 512, false) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) ||
		!ReferenciaMotivoAutorizacionV2Valida(s.Motivo) {
		return ErrSolicitudAsignacionCompetencialV1Invalida
	}
	actorHuella, err := s.Actor.HuellaSHA256VinculadaV2()
	if err != nil || actorHuella != s.ResultadoContexto.HuellaSHA256 {
		return ErrSolicitudAsignacionCompetencialV1Invalida
	}
	return nil
}

// EvidenciaAsignacionCompetencialV1 es una instantanea de fuentes nominales,
// nunca una concesion ejecutable. El consumidor debe revalidarla junto a la
// autorizacion, en la transaccion final del efecto, tambien en recuperacion.
// Asignacion, rol y control reutilizan las autoridades y canonicos de VEC.
type EvidenciaAsignacionCompetencialV1 struct {
	bloqueoSerializacionAsignacionesCompetencialesV1
	PersonaRef, PerfilFirmanteRef, PerfilActivoFirmanteRef, CertificadoHuellaSHA256 string
	UnidadRef, PuestoRef, AmbitoRef                                                 string
	Cargo                                                                           ReferenciaCargoCompetencialV1
	CargoVigenteDesde, CargoVigenteHasta                                            time.Time
	EnlaceOcupante                                                                  EnlaceOcupanteCompetencialV1
	Delegacion                                                                      *DelegacionCompetencialV1
	RecursoHuellaSHA256, AccionCompetencial, FinalidadCompetencial                  string
	RecursoRef, ModuloID, TipoRecurso                                               string
	Asignacion                                                                      AsignacionPerfil
	AsignacionHuellaSHA256                                                          string
	VersionRol                                                                      VersionRol
	VersionRolHuellaSHA256                                                          string
	ControlVigencia                                                                 ControlVigenciaVersionRol
	ControlVigenciaHuellaSHA256                                                     string
	ActoCompetenciaRef                                                              string
	ComprobadaEn                                                                    time.Time
}

// ValidarParaEn rechaza cruces de persona, cargo, perfil, ambito y versiones,
// asi como huellas alteradas o vigencias agotadas. No evalua ABAC ni sustituye
// la revalidacion de certificado, cargo o asignacion desde fuentes actuales.
func (e EvidenciaAsignacionCompetencialV1) ValidarParaEn(s SolicitudAsignacionCompetencialV1, instante time.Time) error {
	ocupanteRef := s.PersonaRef
	if e.Delegacion != nil {
		ocupanteRef = e.Delegacion.DelegantePersonaRef
	}
	if s.ValidarEn(instante) != nil || !instanteAutorizacionCanonico(e.ComprobadaEn) ||
		e.ComprobadaEn.After(instante) || e.PersonaRef != s.PersonaRef ||
		e.PerfilFirmanteRef != s.PerfilFirmanteRef ||
		!referenciaOpacaContextoActorValida(e.PerfilActivoFirmanteRef, "prf_") ||
		e.CertificadoHuellaSHA256 != s.CertificadoHuellaSHA256 ||
		!textoAutorizacionSinComodinSeguro(e.UnidadRef, 512, false) ||
		(e.PuestoRef != "" && !textoAutorizacionSinComodinSeguro(e.PuestoRef, 512, false)) ||
		(e.AmbitoRef != "" && !textoAutorizacionSinComodinSeguro(e.AmbitoRef, 512, false)) ||
		e.Cargo != s.Cargo || e.Cargo.Validar() != nil ||
		e.EnlaceOcupante.ValidarParaEn(ocupanteRef, s.Cargo.Referencia, e.ComprobadaEn) != nil ||
		e.EnlaceOcupante.ValidarParaEn(ocupanteRef, s.Cargo.Referencia, instante) != nil ||
		(e.Delegacion != nil && (e.Delegacion.ValidarParaEn(s.PersonaRef, s.Cargo.Referencia, e.ComprobadaEn) != nil ||
			e.Delegacion.ValidarParaEn(s.PersonaRef, s.Cargo.Referencia, instante) != nil)) ||
		!instanteAutorizacionCanonico(e.CargoVigenteDesde) || !instanteAutorizacionCanonico(e.CargoVigenteHasta) ||
		!e.CargoVigenteHasta.After(e.CargoVigenteDesde) ||
		e.ComprobadaEn.Before(e.CargoVigenteDesde) || !instante.Before(e.CargoVigenteHasta) ||
		e.AccionCompetencial != s.AccionCompetencial || e.FinalidadCompetencial != s.FinalidadCompetencial ||
		e.RecursoRef != s.Recurso.Referencia || e.ModuloID != s.Recurso.ModuloID || e.TipoRecurso != s.Recurso.Tipo ||
		e.Asignacion.Validar() != nil || e.Asignacion.PrincipalID != s.PersonaRef ||
		e.Asignacion.PerfilActivoRef != e.PerfilActivoFirmanteRef ||
		!e.Asignacion.VigenteEn(e.ComprobadaEn) || !e.Asignacion.VigenteEn(instante) || !e.Asignacion.Cubre(s.Recurso) ||
		e.VersionRol.Validar() != nil || e.VersionRol.Estado != EstadoVersionRolPublicada ||
		e.VersionRol.PublicadaEn.After(e.ComprobadaEn) || e.Asignacion.VersionRolRef != e.VersionRol.Referencia() ||
		!concesionCompetencialV1Presente(e.VersionRol, s) ||
		e.ControlVigencia.Validar() != nil || e.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		e.ControlVigencia.VersionRolRef != e.VersionRol.Referencia() || e.ControlVigencia.ActualizadoEn.After(e.ComprobadaEn) ||
		!textoAutorizacionSinComodinSeguro(e.ActoCompetenciaRef, 512, false) {
		return ErrEvidenciaAsignacionCompetencialV1Invalida
	}
	if !huellaCompetencialCoincide(s.Recurso.HuellaContextoAutorizacionSHA256, e.RecursoHuellaSHA256) ||
		!huellaCompetencialCoincide(e.Asignacion.HuellaSHA256, e.AsignacionHuellaSHA256) ||
		!huellaCompetencialCoincide(e.VersionRol.HuellaSHA256, e.VersionRolHuellaSHA256) ||
		!huellaCompetencialCoincide(e.ControlVigencia.HuellaSHA256, e.ControlVigenciaHuellaSHA256) {
		return ErrEvidenciaAsignacionCompetencialV1Invalida
	}
	return nil
}

// Una misma concesion debe declarar la competencia completa. Este cotejo
// estructural no suma concesiones ni evalua ABAC, campos u obligaciones.
func concesionCompetencialV1Presente(rol VersionRol, s SolicitudAsignacionCompetencialV1) bool {
	for _, concesion := range rol.Concesiones {
		if concesion.Accion == s.AccionCompetencial && concesion.ModuloID == s.Recurso.ModuloID &&
			concesion.TipoRecurso == s.Recurso.Tipo && concesion.AdmiteFinalidad(s.FinalidadCompetencial) {
			return true
		}
	}
	return false
}

func huellaCompetencialCoincide(calcular func() (string, error), esperada string) bool {
	actual, err := calcular()
	return err == nil && huellaAsignacionCompetencialV1Valida(esperada) && actual == esperada
}

func huellaAsignacionCompetencialV1Valida(v string) bool {
	return huellaSHA256AutorizacionValida(v) && v != strings.Repeat("0", 64)
}

// El resultado completo solo viaja entre componentes internos. Los canales
// exponen proyecciones propias autorizadas, nunca estas fuentes nominales.
type bloqueoSerializacionAsignacionesCompetencialesV1 struct{}

func (bloqueoSerializacionAsignacionesCompetencialesV1) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAsignacionCompetencialV1Prohibida
}
func (*bloqueoSerializacionAsignacionesCompetencialesV1) UnmarshalJSON([]byte) error {
	return ErrSerializacionAsignacionCompetencialV1Prohibida
}
func (bloqueoSerializacionAsignacionesCompetencialesV1) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAsignacionCompetencialV1Prohibida
}
func (*bloqueoSerializacionAsignacionesCompetencialesV1) UnmarshalText([]byte) error {
	return ErrSerializacionAsignacionCompetencialV1Prohibida
}
func (bloqueoSerializacionAsignacionesCompetencialesV1) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionAsignacionCompetencialV1Prohibida
}
func (*bloqueoSerializacionAsignacionesCompetencialesV1) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionAsignacionCompetencialV1Prohibida
}
func (bloqueoSerializacionAsignacionesCompetencialesV1) String() string {
	return "[ASIGNACION-COMPETENCIAL-INTERNA]"
}
func (b bloqueoSerializacionAsignacionesCompetencialesV1) GoString() string { return b.String() }
func (b bloqueoSerializacionAsignacionesCompetencialesV1) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionAsignacionesCompetencialesV1) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
