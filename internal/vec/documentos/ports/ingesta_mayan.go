package ports

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionReservarRecursoMayan  = "documentos.ingesta_mayan.reservar_recurso"
	AccionConsultarReservaMayan = "documentos.ingesta_mayan.consultar_reserva"
	AccionAbandonarRecursoMayan = "documentos.ingesta_mayan.abandonar_recurso"
	AccionPrepararIngestaMayan  = "documentos.ingesta_mayan.preparar"
	AccionDespacharIngestaMayan = "documentos.ingesta_mayan.despachar"
	AccionResolverIngestaMayan  = "documentos.ingesta_mayan.resolver"
	AccionConciliarIngestaMayan = "documentos.ingesta_mayan.conciliar"
	AccionConsultarIngestaMayan = "documentos.ingesta_mayan.consultar"
	FinalidadIngestaMayan       = "custodiar_evidencia_meritos"
	AccionConsultaCargaCustodia = "documentos.carga_admitida.consultar_para_custodia"
	FaseConsultaCargaCustodia   = "consulta"
	FaseAperturaCargaCustodia   = "abrir"
	EstadoIngestaPreparada      = "preparado"
	EstadoIngestaDespachada     = "despachado"
	EstadoIngestaConfirmada     = "confirmado"
	EstadoIngestaIndeterminada  = "indeterminado"
	EstadoIngestaRechazada      = "rechazado"
	EstadoRecursoReservado      = "reservada"
	EstadoRecursoVinculado      = "vinculada"
	EstadoRecursoAbandonado     = "abandonada"
	EstadoRecursoCaducado       = "caducada"
)

var ErrIngestaIndeterminada = errors.New("documentos: ingesta externa pendiente de conciliacion")
var ErrIngestaNoReenviable = errors.New("documentos: despacho externo ya registrado")

// ReservaRecursoDocumental no representa un hecho declarado en RUM. Fija el
// recurso opaco y su persona antes de la primera carga S3. ClaveIdempotencia
// identifica la operación; un replay devuelve la fila ganadora.
type ReservaRecursoDocumental struct {
	RecursoRef            string    `json:"recurso_ref"`
	PersonaRef            string    `json:"persona_ref"`
	RegistroContextoRef   string    `json:"registro_contexto_ref"`
	HuellaContextoSHA256  string    `json:"huella_contexto_sha256"`
	ClaveIdempotencia     string    `json:"clave_idempotencia"`
	HuellaSolicitudSHA256 string    `json:"huella_solicitud_sha256"`
	Estado                string    `json:"estado"`
	Version               int64     `json:"version"`
	CreadaEn              time.Time `json:"creada_en"`
	ExpiraEn              time.Time `json:"expira_en"`
}

func (r ReservaRecursoDocumental) ValidarCandidata() error {
	if !strings.HasPrefix(r.RecursoRef, "ref:") || !huellaHexIngesta(strings.TrimPrefix(r.RecursoRef, "ref:")) ||
		!domain.ReferenciaOpacaValida(r.RecursoRef) || !vecdomain.ReferenciaPersonaDenominacionValida(r.PersonaRef) ||
		r.RegistroContextoRef == "" || !huellaHexIngesta(r.HuellaContextoSHA256) ||
		!domain.ReferenciaOpacaValida(r.ClaveIdempotencia) || !huellaHexIngesta(r.HuellaSolicitudSHA256) ||
		r.Estado != "" || r.Version != 0 || !r.CreadaEn.IsZero() || !r.ExpiraEn.IsZero() {
		return ErrSolicitudInvalida
	}
	return nil
}

func (r ReservaRecursoDocumental) ValidarPersistida() error {
	copia := r
	copia.Estado = ""
	copia.Version = 0
	copia.CreadaEn = time.Time{}
	copia.ExpiraEn = time.Time{}
	if copia.ValidarCandidata() != nil || r.Version < 1 || r.CreadaEn.IsZero() || r.ExpiraEn.IsZero() || !r.ExpiraEn.After(r.CreadaEn) ||
		(r.Estado != EstadoRecursoReservado && r.Estado != EstadoRecursoVinculado && r.Estado != EstadoRecursoAbandonado && r.Estado != EstadoRecursoCaducado) {
		return ErrSolicitudInvalida
	}
	return nil
}

type OrdenReservarRecursoMayan struct {
	Reserva      ReservaRecursoDocumental
	Autorizacion AutorizacionV3
}
type OrdenConsultaReservaMayan struct {
	ClaveIdempotencia string
	Autorizacion      AutorizacionV3
}
type OrdenAbandonarRecursoMayan struct {
	ClaveIdempotencia string
	Autorizacion      AutorizacionV3
}

// PlanIngestaMayan conserva metadatos y referencias, nunca contenido. CargaID
// identifica una carga previamente admitida por Documentos y no concede por si
// solo acceso al original.
type PlanIngestaMayan struct {
	PlanRef               string    `json:"plan_ref"`
	CargaID               string    `json:"carga_id"`
	PersonaRef            string    `json:"persona_ref"`
	RegistroContextoRef   string    `json:"registro_contexto_ref"`
	HuellaContextoSHA256  string    `json:"huella_contexto_sha256"`
	ExpedienteRef         string    `json:"expediente_ref"`
	EvidenciaRef          string    `json:"evidencia_ref"`
	TipoRef               string    `json:"tipo_ref"`
	ClaveIdempotencia     string    `json:"clave_idempotencia"`
	ClaveReserva          string    `json:"clave_reserva"`
	HuellaSolicitudSHA256 string    `json:"huella_solicitud_sha256"`
	HuellaSHA256          string    `json:"huella_sha256"`
	MIME                  string    `json:"mime"`
	PoliticaRef           string    `json:"politica_ref"`
	VersionPolitica       int64     `json:"version_politica"`
	HuellaPoliticaSHA256  string    `json:"huella_politica_sha256"`
	ConservacionHasta     time.Time `json:"conservacion_hasta"`
	EstadoPolitica        string    `json:"estado_politica"`
	Proteccion            string    `json:"proteccion"`
	Tamano                int64     `json:"tamano"`
	CargaVersion          int64     `json:"carga_version"`
	Version               int64     `json:"version"`
	Estado                string    `json:"estado"`
	DocumentoID           string    `json:"documento_id"`
	ArchivoID             string    `json:"archivo_id"`
	VersionID             string    `json:"version_id"`
	CreadoEn              time.Time `json:"creado_en"`
	ActualizadoEn         time.Time `json:"actualizado_en"`
}

func (p PlanIngestaMayan) ValidarBase() error {
	if (p.PlanRef != "" && !domain.ReferenciaOpacaValida(p.PlanRef)) || !domain.ReferenciaOpacaValida(p.CargaID) ||
		!vecdomain.ReferenciaPersonaDenominacionValida(p.PersonaRef) ||
		p.RegistroContextoRef == "" || !huellaHexIngesta(p.HuellaContextoSHA256) ||
		!strings.HasPrefix(p.ExpedienteRef, "ref:") || !huellaHexIngesta(strings.TrimPrefix(p.ExpedienteRef, "ref:")) ||
		!domain.ReferenciaExpedienteModuloValida("meritos", p.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(p.EvidenciaRef) || !domain.ReferenciaOpacaValida(p.TipoRef) ||
		!domain.ReferenciaOpacaValida(p.PoliticaRef) || p.VersionPolitica < 1 ||
		!huellaHexIngesta(p.HuellaPoliticaSHA256) || p.ConservacionHasta.IsZero() ||
		(p.EstadoPolitica != "aprobada" && p.EstadoPolitica != "provisional") ||
		(p.Proteccion != "conservacion" && p.Proteccion != "bloqueo") ||
		(p.EstadoPolitica == "provisional" && p.Proteccion != "conservacion") ||
		!domain.ReferenciaOpacaValida(p.ClaveIdempotencia) || !domain.ReferenciaOpacaValida(p.ClaveReserva) ||
		!huellaHexIngesta(p.HuellaSolicitudSHA256) || !huellaHexIngesta(p.HuellaSHA256) ||
		p.Tamano < 1 || p.Tamano > 64<<20 || p.CargaVersion != 5 ||
		p.MIME == "" || len(p.MIME) > 255 {
		return ErrSolicitudInvalida
	}
	return nil
}

func (p PlanIngestaMayan) ValidarPersistido() error {
	if p.ValidarBase() != nil || !domain.ReferenciaOpacaValida(p.PlanRef) || p.Version < 1 ||
		p.Estado == "" || p.CreadoEn.IsZero() || p.ActualizadoEn.IsZero() {
		return ErrSolicitudInvalida
	}
	return nil
}

func huellaHexIngesta(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return false
		}
	}
	return true
}

type DocumentoMayan struct {
	DocumentoID  string `json:"documento_id"`
	ArchivoID    string `json:"archivo_id"`
	VersionID    string `json:"version_id"`
	HuellaSHA256 string `json:"huella_sha256"`
	MIME         string `json:"mime"`
	Tamano       int64  `json:"tamano"`
}

func (d DocumentoMayan) ValidarContra(p PlanIngestaMayan) error {
	if !idMayanValido(d.DocumentoID) || !idMayanValido(d.ArchivoID) ||
		!idMayanValido(d.VersionID) || d.HuellaSHA256 != p.HuellaSHA256 ||
		d.Tamano != p.Tamano || d.MIME != p.MIME {
		return ErrSolicitudInvalida
	}
	return nil
}

func idMayanValido(s string) bool {
	n, err := strconv.ParseUint(s, 10, 64)
	return err == nil && n > 0 && strconv.FormatUint(n, 10) == s
}

type OrdenPrepararIngestaMayan struct {
	Plan         PlanIngestaMayan
	Autorizacion AutorizacionV3
}

type OrdenCambiarIngestaMayan struct {
	PlanRef, ExpedienteRef, Estado string
	VersionEsperada                int64
	Documento                      DocumentoMayan
	Autorizacion                   AutorizacionV3
}

func (o OrdenCambiarIngestaMayan) Preimagen(accion string) ([]byte, error) {
	return PreimagenIngestaMayan(accion, struct {
		PlanRef         string         `json:"plan_ref"`
		ExpedienteRef   string         `json:"expediente_ref"`
		VersionEsperada int64          `json:"version_esperada"`
		Estado          string         `json:"estado"`
		Documento       DocumentoMayan `json:"documento"`
	}{o.PlanRef, o.ExpedienteRef, o.VersionEsperada, o.Estado, o.Documento})
}

type OrdenConsultarIngestaMayan struct {
	PlanRef, ExpedienteRef string
	Autorizacion           AutorizacionV3
}

func (o OrdenConsultarIngestaMayan) Preimagen() ([]byte, error) {
	return PreimagenIngestaMayan(AccionConsultarIngestaMayan, struct {
		PlanRef       string `json:"plan_ref"`
		ExpedienteRef string `json:"expediente_ref"`
	}{o.PlanRef, o.ExpedienteRef})
}

// PreimagenIngestaMayan usa campos ordenados y exactos. SQL vuelve a cotejar
// accion, recurso y ambito contra el material V3 antes de cada efecto.
func PreimagenIngestaMayan(accion string, dato any) ([]byte, error) {
	if accion != AccionReservarRecursoMayan && accion != AccionConsultarReservaMayan && accion != AccionAbandonarRecursoMayan &&
		accion != AccionPrepararIngestaMayan && accion != AccionDespacharIngestaMayan &&
		accion != AccionResolverIngestaMayan && accion != AccionConciliarIngestaMayan &&
		accion != AccionConsultarIngestaMayan {
		return nil, ErrSolicitudInvalida
	}
	b, err := json.Marshal(struct {
		Accion string `json:"accion"`
		Dato   any    `json:"dato"`
	}{accion, dato})
	if err != nil {
		return nil, ErrSolicitudInvalida
	}
	return b, nil
}

func (a AutorizacionV3) ValidarParaIngestaMayan(accion, planRef, expedienteRef string, ahora time.Time) error {
	switch accion {
	case AccionReservarRecursoMayan, AccionConsultarReservaMayan, AccionAbandonarRecursoMayan,
		AccionPrepararIngestaMayan, AccionDespacharIngestaMayan, AccionResolverIngestaMayan,
		AccionConciliarIngestaMayan, AccionConsultarIngestaMayan:
	default:
		return ErrSolicitudInvalida
	}
	if !domain.ReferenciaOpacaValida(planRef) || !domain.ReferenciaExpedienteModuloValida("meritos", expedienteRef) ||
		a.Accion != accion || a.Finalidad != FinalidadIngestaMayan || a.RecursoRef != planRef ||
		a.AmbitoRef != expedienteRef || !domain.ReferenciaValida(a.PrincipalID) ||
		!domain.ReferenciaValida(a.PerfilActivoRef) || !domain.ReferenciaValida(a.CorrelacionRef) ||
		a.Material.ValidarEstructura() != nil || ahora.IsZero() {
		return ErrSolicitudInvalida
	}
	r := a.Material.ResumenCapacidad()
	if r.ValidarEstructura() != nil || r.AudienciaConsumo() != AudienciaV3 ||
		r.Operacion() != accion || r.EfectoRef() != planRef ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) {
		return ErrSolicitudInvalida
	}
	return nil
}

// RepositorioIngestaMayan consume V3 y escribe estado, CAS, auditoria y outbox
// en la misma transaccion. Despachar devuelve nuevo=false en cualquier replay:
// la aplicacion solo puede llamar al proveedor cuando nuevo=true.
type RepositorioIngestaMayan interface {
	ReservarRecurso(context.Context, OrdenReservarRecursoMayan) (ReservaRecursoDocumental, error)
	ConsultarReserva(context.Context, OrdenConsultaReservaMayan) (ReservaRecursoDocumental, error)
	AbandonarRecurso(context.Context, OrdenAbandonarRecursoMayan) (ReservaRecursoDocumental, error)
	PrepararIngesta(context.Context, OrdenPrepararIngestaMayan) (PlanIngestaMayan, error)
	DespacharIngesta(context.Context, OrdenCambiarIngestaMayan) (PlanIngestaMayan, bool, error)
	ResolverIngesta(context.Context, OrdenCambiarIngestaMayan) (PlanIngestaMayan, error)
	ConsultarIngesta(context.Context, OrdenConsultarIngestaMayan) (PlanIngestaMayan, error)
}

// CustodioMayan se implementa en el adaptador del proveedor. Reconciliar solo
// recibe un ID conocido y nunca repite POST.
type CustodioMayan interface {
	IngerirOriginal(context.Context, PlanIngestaMayan, io.Reader) (DocumentoMayan, error)
	ReconciliarOriginal(context.Context, DocumentoMayan) (DocumentoMayan, error)
}

// SolicitudCargaAdmitidaCustodia procede de la frontera autenticada. E emite
// material V3 nuevo para cada fase y lo consume antes del SELECT PostgreSQL;
// abrir exige además una concesión nativa de almacén para la versión derivada
// de ese SELECT. Ni el cliente ni este DTO proponen key, bucket o URL.
type SolicitudCargaAdmitidaCustodia struct {
	Accion, Fase, Finalidad                  string
	CargaID, RecursoRef, PersonaRef, PlanRef string
	VersionEsperada                          int
	ActorRegistradoV2                        vecdomain.ResultadoContextoActorRegistradoV2
	VinculoV2                                vecdomain.VinculoAutenticacionActorV2
}

func (s SolicitudCargaAdmitidaCustodia) Validar() error {
	if s.Accion != AccionConsultaCargaCustodia ||
		(s.Fase != FaseConsultaCargaCustodia && s.Fase != FaseAperturaCargaCustodia) ||
		s.Finalidad != FinalidadIngestaMayan || !domain.ReferenciaOpacaValida(s.CargaID) ||
		!domain.ReferenciaExpedienteModuloValida("meritos", s.RecursoRef) ||
		!vecdomain.ReferenciaPersonaDenominacionValida(s.PersonaRef) || s.VersionEsperada != 5 ||
		(s.Fase == FaseAperturaCargaCustodia && !domain.ReferenciaOpacaValida(s.PlanRef)) ||
		s.ActorRegistradoV2.Validar() != nil || s.VinculoV2.ValidarPara(s.ActorRegistradoV2) != nil ||
		s.ActorRegistradoV2.Contexto.PersonaRef != s.PersonaRef {
		return ErrSolicitudInvalida
	}
	return nil
}

// Solo metadatos del agregado autoritativo y su huella canónica, sin bytes.
type DatosCargaAdmitidaCustodia struct {
	Carga             vecdomain.CargaDocumental
	HuellaCargaSHA256 string
}

type LecturaCargaAdmitidaCustodia struct {
	Datos     DatosCargaAdmitidaCustodia
	Contenido io.ReadCloser
}

// AbrirOriginalExacto reconsulta PG con otra decisión V3 y abre en S3 solo la
// versión derivada de esa fila, cotejando la huella esperada del primer SELECT.
type FuenteCargaAdmitidaCustodia interface {
	ConsultarAdmitida(context.Context, SolicitudCargaAdmitidaCustodia) (DatosCargaAdmitidaCustodia, error)
	AbrirOriginalExacto(context.Context, SolicitudCargaAdmitidaCustodia, string) (LecturaCargaAdmitidaCustodia, error)
}

// AutoridadIngestaMayan obtiene una decisión nueva para la preimagen exacta.
// El repositorio vuelve a consumir y revalidar el material en su transacción.
type AutoridadIngestaMayan interface {
	AutorizarIngestaMayan(context.Context, string, []byte, string, string) (AutorizacionV3, error)
}
