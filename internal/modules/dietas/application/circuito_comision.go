package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrComposicionCircuitoInvalida = errors.New("dietas: composicion de circuito invalida")
var claveCircuito = regexp.MustCompile(`^[A-Za-z0-9_-]{16,128}$`)
var cursorCircuito = regexp.MustCompile(`^dco_[A-Za-z0-9_-]{22,128}$`)
var unidadCircuito = regexp.MustCompile(`^[A-Za-z0-9:_-]{3,128}$`)
var horaCircuito = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
var numeroDocumentoCircuito = regexp.MustCompile(`^VEC-D-[0-9]{4}-[0-9]{6,18}$`)

const (
	AccionConsultarDocumentoCircuito    = "dietas.circuito.documento.consultar"
	FinalidadConsultarDocumentoCircuito = "revisar_documento_circuito_dietas"
	// La bandeja sigue consumiendo la función v1; decisión y lectura, la v2.
	esquemaEfectoBandejaV1 = "vec.dietas.circuito-operacion.v1"
)

type CasoUsoCircuitoComision interface {
	Decidir(context.Context, dietasports.IdentidadEfectivaCircuito, dietasports.SolicitudDecisionCircuito) (dietasports.ResultadoCircuitoComision, error)
	ListarPendientes(context.Context, dietasports.IdentidadEfectivaCircuito, dietasports.ConsultaBandejaCircuito) (dietasports.PaginaBandejaCircuito, error)
	ConsultarDocumento(context.Context, dietasports.IdentidadEfectivaCircuito, dietasports.SolicitudDocumentoCircuito) (dietasports.DocumentoCircuito, error)
}

type ServicioCircuitoComision struct {
	repositorio dietasports.RepositorioCircuitoComision
}

func NuevoServicioCircuitoComision(r dietasports.RepositorioCircuitoComision) (*ServicioCircuitoComision, error) {
	if interfazNula(r) {
		return nil, ErrComposicionCircuitoInvalida
	}
	return &ServicioCircuitoComision{repositorio: r}, nil
}

func (s *ServicioCircuitoComision) Decidir(ctx context.Context, identidad dietasports.IdentidadEfectivaCircuito, solicitud dietasports.SolicitudDecisionCircuito) (dietasports.ResultadoCircuitoComision, error) {
	var cero dietasports.ResultadoCircuitoComision
	if s == nil || interfazNula(s.repositorio) || ctx == nil || ValidarSolicitudDecisionCircuito(solicitud) != nil {
		return cero, domain.ErrDecisionCircuitoInvalida
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	accion, recurso, finalidad := ContratoCircuito(dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionDecidirCircuito, Decision: solicitud})
	if !identidadCircuitoValida(identidad, accion, recurso, finalidad, solicitud.UnidadRef) {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	return s.repositorio.Decidir(ctx, identidad, solicitud)
}

func (s *ServicioCircuitoComision) ListarPendientes(ctx context.Context, identidad dietasports.IdentidadEfectivaCircuito, consulta dietasports.ConsultaBandejaCircuito) (dietasports.PaginaBandejaCircuito, error) {
	var cero dietasports.PaginaBandejaCircuito
	if s == nil || interfazNula(s.repositorio) || ctx == nil {
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if consulta.Limite == 0 {
		consulta.Limite = 20
	}
	if ValidarConsultaBandejaCircuito(consulta) != nil {
		return cero, domain.ErrDecisionCircuitoInvalida
	}
	accion, recurso, finalidad := ContratoCircuito(dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionListarBandeja, Consulta: consulta})
	if !identidadCircuitoValida(identidad, accion, recurso, finalidad, consulta.UnidadRef) {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	return s.repositorio.ListarPendientes(ctx, identidad, consulta)
}

// ConsultarDocumento entrega a quien revisa el documento pendiente en su
// etapa. SQL repite competencia y separación antes de mostrarlo.
func (s *ServicioCircuitoComision) ConsultarDocumento(ctx context.Context, identidad dietasports.IdentidadEfectivaCircuito, solicitud dietasports.SolicitudDocumentoCircuito) (dietasports.DocumentoCircuito, error) {
	var cero dietasports.DocumentoCircuito
	if s == nil || interfazNula(s.repositorio) || ctx == nil {
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if ValidarSolicitudDocumentoCircuito(solicitud) != nil {
		return cero, domain.ErrDecisionCircuitoInvalida
	}
	accion, recurso, finalidad := ContratoCircuito(dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionConsultarDocumentoCircuito, Documento: solicitud})
	if !identidadCircuitoValida(identidad, accion, recurso, finalidad, solicitud.UnidadRef) {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	documento, err := s.repositorio.ConsultarDocumento(ctx, identidad, solicitud)
	if err != nil {
		return cero, err
	}
	if ValidarDocumentoCircuito(documento, solicitud) != nil {
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	return documento, nil
}

func ValidarSolicitudDecisionCircuito(s dietasports.SolicitudDecisionCircuito) error {
	if !referenciaComision.MatchString(s.Referencia) || !unidadCircuito.MatchString(s.UnidadRef) || s.Etapa.EstadoPendiente() == "" ||
		(s.Decision != domain.DecisionAprobar && s.Decision != domain.DecisionDevolver) ||
		!claveCircuito.MatchString(s.ClaveIdempotencia) || s.VersionEsperada == 0 || s.VersionEsperada > 999999999999999999 ||
		len(s.Motivo) > 600 || !domain.TextoSinBordes(s.Motivo) ||
		!textoCircuitoValido(s.Motivo) || (s.Decision == domain.DecisionDevolver && len(s.Motivo) < 3) {
		return domain.ErrDecisionCircuitoInvalida
	}
	return nil
}

func ValidarConsultaBandejaCircuito(q dietasports.ConsultaBandejaCircuito) error {
	if q.Etapa.EstadoPendiente() == "" || !unidadCircuito.MatchString(q.UnidadRef) || q.Limite < 1 || q.Limite > 50 ||
		(q.FechaDesde != "" && !domain.FechaCircuitoValida(q.FechaDesde)) ||
		(q.FechaHasta != "" && !domain.FechaCircuitoValida(q.FechaHasta)) ||
		(q.FechaDesde != "" && q.FechaHasta != "" && q.FechaDesde > q.FechaHasta) ||
		(q.Cursor != "" && !cursorCircuito.MatchString(q.Cursor)) {
		return domain.ErrDecisionCircuitoInvalida
	}
	return nil
}

func ValidarSolicitudDocumentoCircuito(s dietasports.SolicitudDocumentoCircuito) error {
	if !referenciaComision.MatchString(s.Referencia) || s.Etapa.EstadoPendiente() == "" || !unidadCircuito.MatchString(s.UnidadRef) {
		return domain.ErrDecisionCircuitoInvalida
	}
	return nil
}

// ValidarDocumentoCircuito comprueba la forma del documento devuelto por SQL
// antes de entregarlo: referencia y etapa pedidas, fechas y horas civiles y
// objetos JSON para cálculo y documento.
func ValidarDocumentoCircuito(d dietasports.DocumentoCircuito, s dietasports.SolicitudDocumentoCircuito) error {
	objeto := func(v json.RawMessage) bool {
		v = bytes.TrimSpace(v)
		return len(v) > 1 && v[0] == '{' && json.Valid(v)
	}
	rutas := bytes.TrimSpace(d.Rutas)
	if d.Referencia != s.Referencia || d.Estado != s.Etapa.EstadoPendiente() || d.Version < 2 ||
		!numeroDocumentoCircuito.MatchString(d.NumeroDocumento) || d.FechaApertura == "" ||
		!domain.FechaCircuitoValida(d.FechaInicio) || !domain.FechaCircuitoValida(d.FechaFin) || d.FechaInicio > d.FechaFin ||
		!horaCircuito.MatchString(d.HoraInicio) || !horaCircuito.MatchString(d.HoraFin) ||
		len(d.Motivo) < 3 || len(d.Motivo) > 600 || len(d.CodigosRuta) > 12 ||
		!objeto(d.Calculo) || !objeto(d.Documento) ||
		(d.VehiculoPropio == nil) != (len(rutas) == 0) ||
		(len(rutas) > 0 && (rutas[0] != '[' || !json.Valid(rutas))) ||
		(d.Devolucion != nil && d.Devolucion.ValidarAnteriorA(d.Version) != nil) {
		return dietasports.ErrCircuitoNoDisponible
	}
	return nil
}

func ContratoCircuito(s dietasports.SolicitudOperacionCircuito) (accion, recurso, finalidad string) {
	switch s.Operacion {
	case dietasports.OperacionDecidirCircuito:
		if ValidarSolicitudDecisionCircuito(s.Decision) != nil {
			return "", "", ""
		}
		switch s.Decision.Etapa {
		case domain.EtapaRevision:
			return "dietas.documento.revisar", s.Decision.Referencia, "revisar_documento_dietas"
		case domain.EtapaAutorizacion:
			return "dietas.documento.autorizar", s.Decision.Referencia, "autorizar_documento_dietas"
		case domain.EtapaLiquidacion:
			return "dietas.documento.liquidar", s.Decision.Referencia, "liquidar_documento_dietas"
		case domain.EtapaFiscalizacion:
			return "dietas.documento.fiscalizar", s.Decision.Referencia, "fiscalizar_documento_dietas"
		}
	case dietasports.OperacionListarBandeja:
		if ValidarConsultaBandejaCircuito(s.Consulta) == nil {
			etapa := string(s.Consulta.Etapa)
			return "dietas.bandeja." + etapa + ".consultar", "dietas:bandeja:" + etapa, "consultar_bandeja_" + etapa + "_dietas"
		}
	case dietasports.OperacionConsultarDocumentoCircuito:
		if ValidarSolicitudDocumentoCircuito(s.Documento) == nil {
			return AccionConsultarDocumentoCircuito, s.Documento.Referencia, FinalidadConsultarDocumentoCircuito
		}
	}
	return "", "", ""
}

func identidadCircuitoValida(i dietasports.IdentidadEfectivaCircuito, accion, recurso, finalidad, unidad string) bool {
	if accion == "" || recurso == "" || finalidad == "" || !unidadCircuito.MatchString(unidad) || i.UnidadCompetenciaRef != unidad || i.ContextoRegistrado.Validar() != nil || i.Vinculo.ValidarPara(i.ContextoRegistrado) != nil {
		return false
	}
	a := i.Autorizacion
	return a.Material.ValidarEstructura() == nil && a.Accion == accion && a.RecursoRef == recurso && a.Finalidad == finalidad &&
		a.Material.PersonaVersion() == i.ContextoRegistrado.Contexto.Instantanea.PersonaVersion && a.Material.PerfilVersion() == i.ContextoRegistrado.Contexto.Instantanea.PerfilVersion
}

func textoCircuitoValido(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

type materialCircuito struct {
	Esquema         string                                 `json:"esquema"`
	Operacion       string                                 `json:"operacion"`
	RecursoRef      string                                 `json:"recurso_ref"`
	UnidadRef       string                                 `json:"unidad_ref"`
	Etapa           domain.EtapaCircuito                   `json:"etapa,omitempty"`
	Identidad       identidadCircuito                      `json:"identidad"`
	HuellaSemantica string                                 `json:"huella_semantica,omitempty"`
	Comando         *dietasports.SolicitudDecisionCircuito `json:"comando,omitempty"`
	Consulta        *dietasports.ConsultaBandejaCircuito   `json:"consulta,omitempty"`
}
type identidadCircuito struct {
	ActorRef         string `json:"actor_ref"`
	PerfilRef        string `json:"perfil_ref"`
	PersonaRef       string `json:"persona_ref"`
	ContextoActorRef string `json:"contexto_actor_ref"`
	ContextoVersion  uint64 `json:"contexto_version"`
	CuentaRef        string `json:"cuenta_ref"`
	CuentaVersion    uint64 `json:"cuenta_version"`
	PersonaVersion   uint64 `json:"persona_version"`
	PerfilVersion    uint64 `json:"perfil_version"`
}

// ConstruirEfectoAutorizacionCircuito liga la operación exacta al contexto V2
// y a la unidad que acredita la fuente de competencia, para que la autoridad
// común emita V3 y SQL lo consuma en la misma transacción que el efecto. La
// decisión y la lectura del documento no llevan asignación del cliente: SQL
// usa la de la versión enviada.
func ConstruirEfectoAutorizacionCircuito(contexto vecdomain.ResultadoContextoActorRegistradoV2, unidadCompetenciaRef string, solicitud dietasports.SolicitudOperacionCircuito) (dietasports.EfectoAutorizacionCircuito, error) {
	var cero dietasports.EfectoAutorizacionCircuito
	if contexto.Validar() != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	accion, recursoRef, _ := ContratoCircuito(solicitud)
	if accion == "" {
		return cero, domain.ErrDecisionCircuitoInvalida
	}
	var unidad string
	switch solicitud.Operacion {
	case dietasports.OperacionDecidirCircuito:
		unidad = solicitud.Decision.UnidadRef
	case dietasports.OperacionListarBandeja:
		unidad = solicitud.Consulta.UnidadRef
	default:
		unidad = solicitud.Documento.UnidadRef
	}
	if unidadCompetenciaRef != unidad || !unidadCircuito.MatchString(unidad) {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	actor := contexto.Contexto
	i := actor.Instantanea
	m := materialCircuito{Esquema: dietasports.EsquemaEfectoCircuitoV2, Operacion: string(solicitud.Operacion), RecursoRef: recursoRef, UnidadRef: unidad,
		Identidad: identidadCircuito{ActorRef: actor.Principal.ID, PerfilRef: actor.PerfilActivoRef, PersonaRef: actor.PersonaRef, ContextoActorRef: i.VinculoRef, ContextoVersion: i.VinculoVersion, CuentaRef: i.CuentaRef, CuentaVersion: i.CuentaVersion, PersonaVersion: i.PersonaVersion, PerfilVersion: i.PerfilVersion}}
	tipo := dietasports.TipoRecursoDocumentoDietas
	switch solicitud.Operacion {
	case dietasports.OperacionDecidirCircuito:
		m.Comando = &solicitud.Decision
		d := solicitud.Decision
		canon := strings.Join([]string{actor.PersonaRef, recursoRef, d.UnidadRef, string(d.Etapa), string(d.Decision), d.Motivo, d.ClaveIdempotencia, strconv.FormatUint(d.VersionEsperada, 10)}, "\x1f")
		h := sha256.Sum256([]byte(canon))
		m.HuellaSemantica = hex.EncodeToString(h[:])
	case dietasports.OperacionListarBandeja:
		m.Esquema = esquemaEfectoBandejaV1
		m.Consulta = &solicitud.Consulta
		tipo = dietasports.TipoRecursoBandejaDietas
	default:
		m.Etapa = solicitud.Documento.Etapa
	}
	material, err := json.Marshal(m)
	if err != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	h := sha256.Sum256(material)
	atributos := map[string]string{"operacion": m.Operacion, "recurso_ref": recursoRef, "material_sha256": hex.EncodeToString(h[:]),
		"contexto_actor_ref": i.VinculoRef, "contexto_version": strconv.FormatUint(i.VinculoVersion, 10), "cuenta_ref": i.CuentaRef,
		"cuenta_version": strconv.FormatUint(i.CuentaVersion, 10), "persona_version": strconv.FormatUint(i.PersonaVersion, 10), "perfil_version": strconv.FormatUint(i.PerfilVersion, 10)}
	if m.Etapa != "" {
		atributos["etapa"] = string(m.Etapa)
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: recursoRef, ModuloID: dietasports.ModuloDietas, Tipo: tipo, Ambitos: map[string]string{"persona_ref": actor.PersonaRef, "unidad_ref": unidad}, Atributos: atributos}
	if recurso.Validar() != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	return dietasports.EfectoAutorizacionCircuito{Material: material, Recurso: recurso}, nil
}

var _ CasoUsoCircuitoComision = (*ServicioCircuitoComision)(nil)
