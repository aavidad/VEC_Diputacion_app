package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

var ErrCanonCompetenciaFirmanteHistoricaV1Invalido = errors.New("vec: canon historico de competencia firmante invalido")
var ErrRecuperacionCompetenciaFirmanteHistoricaV1 = errors.New("vec: competencia firmante historica no recuperable")

const (
	EsquemaCanonCompetenciaFirmanteHistoricaV1 = "vec.competencia-firmante.historica.v1"
	limiteCanonCompetenciaFirmanteHistoricaV1  = 32 << 10
)

// ReferenciaHistoricaCompetenciaV1 contiene solamente el identificador de una
// version y su huella publicada. No transporta la fuente ni acredita su estado.
type ReferenciaHistoricaCompetenciaV1 struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaHistoricaCompetenciaV1) valida() bool {
	return textoAutorizacionSinComodinSeguro(r.Referencia, 512, false) && r.Version > 0 &&
		huellaAsignacionCompetencialV1Valida(r.HuellaSHA256)
}

type IdentidadFirmanteHistoricaV1 struct {
	CertificadoDERSHA256    string                           `json:"certificado_der_sha256"`
	PersonaRef              string                           `json:"persona_ref"`
	Persona                 ReferenciaHistoricaCompetenciaV1 `json:"persona"`
	Cuenta                  ReferenciaHistoricaCompetenciaV1 `json:"cuenta"`
	VinculoCuentaPersona    ReferenciaHistoricaCompetenciaV1 `json:"vinculo_cuenta_persona"`
	CuentaPersonaCuentaRef  string                           `json:"cuenta_persona_cuenta_ref"`
	CuentaPersonaPersonaRef string                           `json:"cuenta_persona_persona_ref"`
	VinculoCertificado      ReferenciaHistoricaCompetenciaV1 `json:"vinculo_certificado"`
	VinculoCuentaRef        string                           `json:"vinculo_cuenta_ref"`
	VinculoPersonaRef       string                           `json:"vinculo_persona_ref"`
	VinculoDERSHA256        string                           `json:"vinculo_der_sha256"`
}

// La asignacion, el rol y el control son referencias a sus versiones
// originales. Los perfiles y enlaces hacen comprobables los cruces nominales.
type AsignacionFirmanteHistoricaV1 struct {
	Asignacion            ReferenciaHistoricaCompetenciaV1 `json:"asignacion"`
	Rol                   ReferenciaHistoricaCompetenciaV1 `json:"rol"`
	RolID                 string                           `json:"rol_id"`
	ControlRol            ReferenciaHistoricaCompetenciaV1 `json:"control_rol"`
	PersonaRef            string                           `json:"persona_ref"`
	PerfilEsperadoRef     string                           `json:"perfil_esperado_ref"`
	PerfilActivoRef       string                           `json:"perfil_activo_ref"`
	ModuloID              string                           `json:"modulo_id"`
	TipoRecurso           string                           `json:"tipo_recurso"`
	RecursoRef            string                           `json:"recurso_ref"`
	AmbitoOrganizacionRef string                           `json:"ambito_organizacion_ref"`
	AmbitoUnidadRef       string                           `json:"ambito_unidad_ref"`
	AsignacionRolRef      string                           `json:"asignacion_rol_ref"`
	ControlRolRef         string                           `json:"control_rol_ref"`
	VigenteDesde          time.Time                        `json:"vigente_desde"`
	VigenteHasta          time.Time                        `json:"vigente_hasta"`
}

// La vigencia es historica, en intervalo [desde, hasta). La delegacion, si
// existe, pertenece a Personal y no cambia la asignacion propia del delegado.
type FuentePersonalFirmanteHistoricaV1 struct {
	Cargo              ReferenciaHistoricaCompetenciaV1 `json:"cargo"`
	EnlaceOcupante     ReferenciaHistoricaCompetenciaV1 `json:"enlace_ocupante"`
	OcupantePersonaRef string                           `json:"ocupante_persona_ref"`
	CargoRefEnlace     string                           `json:"cargo_ref_enlace"`
	CargoVigenteDesde  time.Time                        `json:"cargo_vigente_desde"`
	CargoVigenteHasta  time.Time                        `json:"cargo_vigente_hasta"`
	EnlaceVigenteDesde time.Time                        `json:"enlace_vigente_desde"`
	EnlaceVigenteHasta time.Time                        `json:"enlace_vigente_hasta"`
	Delegacion         *DelegacionFirmanteHistoricaV1   `json:"delegacion"`
}

type DelegacionFirmanteHistoricaV1 struct {
	Acto                ReferenciaHistoricaCompetenciaV1 `json:"acto"`
	DelegantePersonaRef string                           `json:"delegante_persona_ref"`
	DelegadoPersonaRef  string                           `json:"delegado_persona_ref"`
	CargoRef            string                           `json:"cargo_ref"`
	VigenteDesde        time.Time                        `json:"vigente_desde"`
	VigenteHasta        time.Time                        `json:"vigente_hasta"`
}

// RelacionCTFirmanteHistoricaV1 conserva las referencias reales de CT174.
// PruebaSnapshotSHA256 es la huella de la prueba completa de origen; no se
// presenta como una huella especifica de la relacion expediente/unidad.
type RelacionCTFirmanteHistoricaV1 struct {
	ExpedienteRef        string    `json:"expediente_ref"`
	UnidadRef            string    `json:"unidad_ref"`
	OrigenRef            string    `json:"origen_ref"`
	OrigenVersion        uint64    `json:"origen_version"`
	PruebaSnapshotSHA256 string    `json:"prueba_snapshot_sha256"`
	EventoRef            string    `json:"evento_ref"`
	EventoHuellaSHA256   string    `json:"evento_huella_sha256"`
	ConfirmadaEn         time.Time `json:"confirmada_en"`
}

type RecursoFirmaHistoricaV1 struct {
	OrganizacionRef       string                            `json:"organizacion_ref"`
	UnidadRef             string                            `json:"unidad_ref"`
	ExpedienteRef         string                            `json:"expediente_ref"`
	DocumentoRef          string                            `json:"documento_ref"`
	RecursoAutorizableRef string                            `json:"recurso_autorizable_ref"`
	ModuloID              string                            `json:"modulo_id"`
	TipoRecurso           string                            `json:"tipo_recurso"`
	RecursoContextoSHA256 string                            `json:"recurso_contexto_sha256"`
	Original              ReferenciaHistoricaCompetenciaV1  `json:"original"`
	PDFRaizSHA256         string                            `json:"pdf_raiz_sha256"`
	Firmado               ReferenciaHistoricaCompetenciaV1  `json:"firmado"`
	PDFFirmadoSHA256      string                            `json:"pdf_firmado_sha256"`
	NumeroFirmas          uint64                            `json:"numero_firmas"`
	EntradaRevision       *ReferenciaHistoricaCompetenciaV1 `json:"entrada_revision"`
}

// CanonCompetenciaFirmanteHistoricaV1 es un registro de procedencia. Sus
// referencias se contrastan con fuentes propietarias al crear y recuperar el
// efecto; nunca concede competencia o acceso por si mismo. Se construye desde
// valores minimos verificados, sin serializar SolicitudAsignacionCompetencialV1,
// EvidenciaAsignacionCompetencialV1 ni acreditaciones internas.
type CanonCompetenciaFirmanteHistoricaV1 struct {
	Esquema        string                            `json:"esquema"`
	Identidad      IdentidadFirmanteHistoricaV1      `json:"identidad"`
	Competencia    AsignacionFirmanteHistoricaV1     `json:"competencia"`
	Personal       FuentePersonalFirmanteHistoricaV1 `json:"personal"`
	Recurso        RecursoFirmaHistoricaV1           `json:"recurso"`
	RelacionCT     RelacionCTFirmanteHistoricaV1     `json:"relacion_ct"`
	Accion         string                            `json:"accion"`
	Finalidad      string                            `json:"finalidad"`
	Motivo         ReferenciaEntradaCatalogo         `json:"motivo"`
	Circuito       ReferenciaHistoricaCompetenciaV1  `json:"circuito"`
	PasoRef        string                            `json:"paso_ref"`
	PasoOrden      uint64                            `json:"paso_orden"`
	FechaHistorica time.Time                         `json:"fecha_historica"`
}

func vigenciaHistoricaCompetenciaV1(desde, hasta, en time.Time) bool {
	return instanteAutorizacionCanonico(desde) && instanteAutorizacionCanonico(hasta) &&
		hasta.After(desde) && !en.Before(desde) && en.Before(hasta)
}

func (c CanonCompetenciaFirmanteHistoricaV1) Validar() error {
	en, i, a, p, r, ct := c.FechaHistorica, c.Identidad, c.Competencia, c.Personal, c.Recurso, c.RelacionCT
	if c.Esquema != EsquemaCanonCompetenciaFirmanteHistoricaV1 || !instanteAutorizacionCanonico(en) ||
		!huellaAsignacionCompetencialV1Valida(i.CertificadoDERSHA256) ||
		!referenciaOpacaContextoActorValida(i.PersonaRef, "per_") ||
		!i.Persona.valida() || i.Persona.Referencia != i.PersonaRef ||
		!i.Cuenta.valida() || !i.VinculoCuentaPersona.valida() || !i.VinculoCertificado.valida() ||
		i.CuentaPersonaCuentaRef != i.Cuenta.Referencia || i.CuentaPersonaPersonaRef != i.PersonaRef ||
		i.VinculoCuentaRef != i.Cuenta.Referencia || i.VinculoPersonaRef != i.PersonaRef ||
		i.VinculoDERSHA256 != i.CertificadoDERSHA256 ||
		!a.Asignacion.valida() || !a.Rol.valida() || !a.ControlRol.valida() ||
		!textoAutorizacionSinComodinSeguro(a.RolID, 128, false) ||
		a.Rol.Version > uint64(^uint(0)>>1) ||
		a.Rol.Referencia != (VersionRol{RolID: a.RolID, Version: int(a.Rol.Version)}).Referencia() ||
		a.PersonaRef != i.PersonaRef || a.AsignacionRolRef != a.Rol.Referencia || a.ControlRolRef != a.Rol.Referencia ||
		a.ModuloID != r.ModuloID || a.TipoRecurso != r.TipoRecurso || a.RecursoRef != r.RecursoAutorizableRef ||
		a.AmbitoOrganizacionRef != r.OrganizacionRef || a.AmbitoUnidadRef != r.UnidadRef ||
		!textoAutorizacionSinComodinSeguro(a.PerfilEsperadoRef, 512, false) ||
		!referenciaOpacaContextoActorValida(a.PerfilActivoRef, "prf_") ||
		!vigenciaHistoricaCompetenciaV1(a.VigenteDesde, a.VigenteHasta, en) ||
		!p.Cargo.valida() || !p.EnlaceOcupante.valida() || p.CargoRefEnlace != p.Cargo.Referencia ||
		!vigenciaHistoricaCompetenciaV1(p.CargoVigenteDesde, p.CargoVigenteHasta, en) ||
		!vigenciaHistoricaCompetenciaV1(p.EnlaceVigenteDesde, p.EnlaceVigenteHasta, en) ||
		!referenciaOpacaContextoActorValida(p.OcupantePersonaRef, "per_") ||
		!textoAutorizacionSinComodinSeguro(r.OrganizacionRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(r.UnidadRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(r.ExpedienteRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(r.DocumentoRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(r.RecursoAutorizableRef, 512, false) ||
		r.RecursoAutorizableRef != r.DocumentoRef ||
		!textoAutorizacionSinComodinSeguro(r.ModuloID, 128, false) ||
		!textoAutorizacionSinComodinSeguro(r.TipoRecurso, 128, false) ||
		!huellaAsignacionCompetencialV1Valida(r.RecursoContextoSHA256) ||
		!r.Original.valida() || r.Original.HuellaSHA256 != r.PDFRaizSHA256 ||
		!r.Firmado.valida() || r.Firmado.HuellaSHA256 != r.PDFFirmadoSHA256 || r.NumeroFirmas == 0 ||
		!textoAutorizacionSinComodinSeguro(ct.OrigenRef, 512, false) || ct.OrigenVersion == 0 ||
		!huellaAsignacionCompetencialV1Valida(ct.PruebaSnapshotSHA256) ||
		!textoAutorizacionSinComodinSeguro(ct.EventoRef, 512, false) ||
		!huellaAsignacionCompetencialV1Valida(ct.EventoHuellaSHA256) ||
		!instanteAutorizacionCanonico(ct.ConfirmadaEn) || ct.ConfirmadaEn.After(en) ||
		ct.ExpedienteRef != r.ExpedienteRef || ct.UnidadRef != r.UnidadRef ||
		!textoAutorizacionSinComodinSeguro(c.Accion, 256, false) ||
		!textoAutorizacionSinComodinSeguro(c.Finalidad, 512, false) ||
		c.Motivo.Validar() != nil || !c.Circuito.valida() ||
		!textoAutorizacionSinComodinSeguro(c.PasoRef, 512, false) || c.PasoOrden == 0 {
		return ErrCanonCompetenciaFirmanteHistoricaV1Invalido
	}
	if p.Delegacion == nil {
		if p.OcupantePersonaRef != i.PersonaRef {
			return ErrCanonCompetenciaFirmanteHistoricaV1Invalido
		}
	} else {
		d := p.Delegacion
		if !d.Acto.valida() || d.DelegantePersonaRef != p.OcupantePersonaRef ||
			d.DelegadoPersonaRef != i.PersonaRef || d.DelegantePersonaRef == d.DelegadoPersonaRef ||
			d.CargoRef != p.Cargo.Referencia ||
			!vigenciaHistoricaCompetenciaV1(d.VigenteDesde, d.VigenteHasta, en) {
			return ErrCanonCompetenciaFirmanteHistoricaV1Invalido
		}
	}
	if (r.NumeroFirmas > 1 && r.EntradaRevision == nil) ||
		(r.NumeroFirmas == 1 && r.EntradaRevision != nil) ||
		(r.EntradaRevision != nil && !r.EntradaRevision.valida()) {
		return ErrCanonCompetenciaFirmanteHistoricaV1Invalido
	}
	return nil
}

// SelectorHistoricoCompetenciaFirmanteV1 identifica el efecto original. El
// propietario lo resuelve desde RegistroRef; un valor recibido del canal no
// prueba esta relacion.
type SelectorHistoricoCompetenciaFirmanteV1 struct {
	OrganizacionRef, UnidadRef, ExpedienteRef, DocumentoRef string
	ModuloID, TipoRecurso, RecursoRef                       string
	RecursoContextoSHA256                                   string
}

func (s SelectorHistoricoCompetenciaFirmanteV1) Validar() error {
	if textoAutorizacionSinComodinSeguro(s.OrganizacionRef, 512, false) &&
		textoAutorizacionSinComodinSeguro(s.UnidadRef, 512, false) &&
		textoAutorizacionSinComodinSeguro(s.ExpedienteRef, 512, false) &&
		textoAutorizacionSinComodinSeguro(s.DocumentoRef, 512, false) &&
		textoAutorizacionSinComodinSeguro(s.ModuloID, 128, false) &&
		textoAutorizacionSinComodinSeguro(s.TipoRecurso, 128, false) &&
		s.RecursoRef == s.DocumentoRef &&
		huellaAsignacionCompetencialV1Valida(s.RecursoContextoSHA256) {
		return nil
	}
	return ErrCanonCompetenciaFirmanteHistoricaV1Invalido
}

// DescriptorLecturaCompetenciaFirmanteHistoricaV1 pertenece al catalogo
// publicado de consultas. La fuente debe comprobar su version/huella y que
// RegistroRef procede del efecto propietario antes de usarlo con el PDP.
type DescriptorLecturaCompetenciaFirmanteHistoricaV1 struct {
	Catalogo                                        ReferenciaEntradaCatalogo
	RegistroRef, RecursoRef, ModuloID, TipoRecurso  string
	ClaveOrganizacion, ClaveUnidad, ClaveExpediente string
}

func (d DescriptorLecturaCompetenciaFirmanteHistoricaV1) Validar() error {
	if d.Catalogo.Validar() == nil &&
		textoAutorizacionSinComodinSeguro(d.RegistroRef, 512, false) &&
		textoAutorizacionSinComodinSeguro(d.RecursoRef, 512, false) &&
		textoAutorizacionSinComodinSeguro(d.ModuloID, 128, false) &&
		textoAutorizacionSinComodinSeguro(d.TipoRecurso, 128, false) &&
		textoAutorizacionSinComodinSeguro(d.ClaveOrganizacion, 128, false) &&
		textoAutorizacionSinComodinSeguro(d.ClaveUnidad, 128, false) &&
		textoAutorizacionSinComodinSeguro(d.ClaveExpediente, 128, false) &&
		d.ClaveOrganizacion != d.ClaveUnidad && d.ClaveOrganizacion != d.ClaveExpediente &&
		d.ClaveUnidad != d.ClaveExpediente {
		return nil
	}
	return ErrCanonCompetenciaFirmanteHistoricaV1Invalido
}

// ValidarLecturaCompetenciaFirmanteHistoricaV1 liga la lectura autorizada
// actual al efecto historico sin identificar sus dos huellas de contexto.
// La autorizacion V3 del consultante se consume aparte sobre recursoActual.
func ValidarLecturaCompetenciaFirmanteHistoricaV1(
	c CanonCompetenciaFirmanteHistoricaV1, registroRef string,
	s SelectorHistoricoCompetenciaFirmanteV1,
	d DescriptorLecturaCompetenciaFirmanteHistoricaV1,
	recursoActual RecursoAutorizable,
) error {
	r := c.Recurso
	if c.Validar() != nil || s.Validar() != nil || d.Validar() != nil ||
		recursoActual.Validar() != nil || d.RegistroRef != registroRef ||
		recursoActual.Referencia != d.RecursoRef || recursoActual.ModuloID != d.ModuloID ||
		recursoActual.Tipo != d.TipoRecurso || len(recursoActual.Ambitos) != 3 ||
		recursoActual.Ambitos[d.ClaveOrganizacion] != r.OrganizacionRef ||
		recursoActual.Ambitos[d.ClaveUnidad] != r.UnidadRef ||
		recursoActual.Ambitos[d.ClaveExpediente] != r.ExpedienteRef ||
		s.OrganizacionRef != r.OrganizacionRef || s.UnidadRef != r.UnidadRef ||
		s.ExpedienteRef != r.ExpedienteRef || s.DocumentoRef != r.DocumentoRef ||
		s.ModuloID != r.ModuloID || s.TipoRecurso != r.TipoRecurso ||
		s.RecursoRef != r.RecursoAutorizableRef ||
		s.RecursoContextoSHA256 != r.RecursoContextoSHA256 {
		return ErrCanonCompetenciaFirmanteHistoricaV1Invalido
	}
	return nil
}

// SolicitudRecuperacionCompetenciaFirmanteHistoricaV1 separa la autorización
// vigente del consultante del selector del efecto original. La fuente resuelve
// ambos desde sus propietarios antes de conceder una lectura.
type SolicitudRecuperacionCompetenciaFirmanteHistoricaV1 struct {
	RegistroRef       string
	CanonHuellaSHA256 string
	Actor             ContextoActor
	ResultadoContexto ResultadoContextoActorRegistradoV2
	Vinculo           VinculoAutenticacionActorV2
	RecursoActual     RecursoAutorizable
	SelectorHistorico SelectorHistoricoCompetenciaFirmanteV1
	DescriptorLectura DescriptorLecturaCompetenciaFirmanteHistoricaV1
	Accion            string
	Finalidad         string
	Motivo            ReferenciaEntradaCatalogo
	CorrelacionRef    string
}

func (s SolicitudRecuperacionCompetenciaFirmanteHistoricaV1) ValidarEn(en time.Time) error {
	if instanteAutorizacionCanonico(en) &&
		s.Actor.Validar() == nil && s.ResultadoContexto.Validar() == nil &&
		s.Vinculo.VigenteEn(en, s.ResultadoContexto) &&
		s.RecursoActual.Validar() == nil && len(s.RecursoActual.Ambitos) > 0 &&
		s.SelectorHistorico.Validar() == nil &&
		s.DescriptorLectura.Validar() == nil &&
		ReferenciaMotivoAutorizacionV2Valida(s.Motivo) &&
		textoAutorizacionSinComodinSeguro(s.RegistroRef, 512, false) &&
		huellaAsignacionCompetencialV1Valida(s.CanonHuellaSHA256) &&
		textoAutorizacionSinComodinSeguro(s.Accion, 256, false) &&
		textoAutorizacionSinComodinSeguro(s.Finalidad, 512, false) &&
		ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) {
		huella, err := s.Actor.HuellaSHA256VinculadaV2()
		if err == nil && huella == s.ResultadoContexto.HuellaSHA256 {
			return nil
		}
	}
	return ErrRecuperacionCompetenciaFirmanteHistoricaV1
}

// ValidarResultado liga el canon al recurso y a su selector exactos. La fuente
// consume la autorización actual y audita incluso una recuperación repetida.
func (s SolicitudRecuperacionCompetenciaFirmanteHistoricaV1) ValidarResultado(
	c CanonCompetenciaFirmanteHistoricaV1, en time.Time,
) error {
	if s.ValidarEn(en) != nil {
		return ErrRecuperacionCompetenciaFirmanteHistoricaV1
	}
	huella, err := c.HuellaSHA256()
	if err != nil || huella != s.CanonHuellaSHA256 ||
		ValidarLecturaCompetenciaFirmanteHistoricaV1(c, s.RegistroRef,
			s.SelectorHistorico, s.DescriptorLectura, s.RecursoActual) != nil {
		return ErrRecuperacionCompetenciaFirmanteHistoricaV1
	}
	return nil
}

type datosCanonCompetenciaFirmanteHistoricaV1 CanonCompetenciaFirmanteHistoricaV1

// Canonico devuelve un esquema cerrado y versionado para persistencia. No
// incluye hora de consulta, correlacion ni concesion efimera del consultante.
func (c CanonCompetenciaFirmanteHistoricaV1) Canonico() ([]byte, error) {
	if c.Validar() != nil {
		return nil, ErrCanonCompetenciaFirmanteHistoricaV1Invalido
	}
	b, err := json.Marshal(datosCanonCompetenciaFirmanteHistoricaV1(c))
	if err != nil || len(b) > limiteCanonCompetenciaFirmanteHistoricaV1 {
		return nil, ErrCanonCompetenciaFirmanteHistoricaV1Invalido
	}
	return b, nil
}

func (c CanonCompetenciaFirmanteHistoricaV1) HuellaSHA256() (string, error) {
	b, err := c.Canonico()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// Recuperar exige bytes canonicos exactos y una huella externa conservada.
// No sustituye la autorizacion vigente de lectura ni la verificacion de las
// fuentes propietarias y de la firma.
func RecuperarCanonCompetenciaFirmanteHistoricaV1(b []byte, huella string) (CanonCompetenciaFirmanteHistoricaV1, error) {
	var vacio CanonCompetenciaFirmanteHistoricaV1
	if len(b) == 0 || len(b) > limiteCanonCompetenciaFirmanteHistoricaV1 ||
		!huellaAsignacionCompetencialV1Valida(huella) {
		return vacio, ErrCanonCompetenciaFirmanteHistoricaV1Invalido
	}
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != huella {
		return vacio, ErrCanonCompetenciaFirmanteHistoricaV1Invalido
	}
	var d datosCanonCompetenciaFirmanteHistoricaV1
	if json.Unmarshal(b, &d) != nil {
		return vacio, ErrCanonCompetenciaFirmanteHistoricaV1Invalido
	}
	c := CanonCompetenciaFirmanteHistoricaV1(d)
	canonico, err := c.Canonico()
	if err != nil || !bytes.Equal(canonico, b) {
		return vacio, ErrCanonCompetenciaFirmanteHistoricaV1Invalido
	}
	return c, nil
}

// Las proyecciones de canal se definen aparte. La serializacion generica de
// este registro sensible no es una via de exportacion.
func (CanonCompetenciaFirmanteHistoricaV1) MarshalJSON() ([]byte, error) {
	return nil, ErrCanonCompetenciaFirmanteHistoricaV1Invalido
}

func (*CanonCompetenciaFirmanteHistoricaV1) UnmarshalJSON([]byte) error {
	return ErrCanonCompetenciaFirmanteHistoricaV1Invalido
}
