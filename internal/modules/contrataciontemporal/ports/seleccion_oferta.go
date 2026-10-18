package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrSeleccionOfertaCTInvalida = errors.New("contratacion temporal: seleccion de oferta invalida")

const (
	EtapaSeleccionOfertaConsultar    = "consultar"
	EtapaSeleccionOfertaReservar     = "reservar"
	EtapaSeleccionOfertaAbrirVentana = "abrir_ventana"
	EtapaSeleccionOfertaConfirmar    = "confirmar"
	InicioSeleccionOfertaPublicacion = "publicacion"
	EsquemaReciboSeleccionOfertaCTV1 = "vec.contratacion-temporal.recibo-seleccion-oferta.v1"
	EstadoReciboSeleccionOfertaCTV1  = "confirmado"
)

var huellaSeleccionOfertaCT = regexp.MustCompile(`^[0-9a-f]{64}$`)
var claveBolsaSeleccionOfertaCT = regexp.MustCompile(`^ct-oferta:[0-9a-f]{64}$`)

// IntencionSeleccionOfertaCT se obtiene de la sesión y del expediente en el
// servidor. Su consulta autorizada precede cualquier nueva lectura de Bolsa.
// El vínculo y el contexto solo viajan en memoria para autorización fresca;
// nunca se congelan como sustituto del comando de oferta.
type IntencionSeleccionOfertaCT struct {
	OrganizacionRef    string
	ExpedienteRef      string
	CategoriaRef       string
	CentroRef          string
	UnidadGestoraRef   string
	ModalidadClave     string
	CausaClave         string
	VersionExpediente  uint64
	ClaveIdempotencia  string
	ActorRef           string
	PerfilRef          string
	HuellaIntencion    string
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

func (i IntencionSeleccionOfertaCT) Validar() error {
	if !domain.ReferenciaOpacaValida(i.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(i.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(i.CategoriaRef) ||
		!domain.ReferenciaOpacaValida(i.CentroRef) ||
		!domain.ReferenciaOpacaValida(i.UnidadGestoraRef) ||
		((i.ModalidadClave != "" || i.CausaClave != "") &&
			(!domain.ClaveCatalogo(i.ModalidadClave).Valida() || !domain.ClaveCatalogo(i.CausaClave).Valida())) ||
		i.VersionExpediente == 0 || i.VersionExpediente > 1<<53-1 ||
		!ClaveIdempotenciaValida(i.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(i.ActorRef) ||
		!domain.ReferenciaOpacaValida(i.PerfilRef) ||
		!huellaSeleccionOfertaCT.MatchString(i.HuellaIntencion) ||
		i.HuellaIntencion != HuellaIntencionSeleccionOfertaCT(i) ||
		i.ResultadoContexto.Validar() != nil ||
		i.Vinculo.ValidarPara(i.ResultadoContexto) != nil ||
		i.ResultadoContexto.Contexto.PersonaRef != i.ActorRef ||
		i.ResultadoContexto.Contexto.PerfilActivoRef != i.PerfilRef ||
		i.Correlacion.Validar() != nil ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(i.MotivoAutorizacion) {
		return ErrSeleccionOfertaCTInvalida
	}
	return nil
}

// HuellaIntencionSeleccionOfertaCT separa la intención de oferta de la
// selección singular. El adaptador durable sella además el ámbito con HMAC.
func HuellaIntencionSeleccionOfertaCT(i IntencionSeleccionOfertaCT) string {
	h := sha256.New()
	for _, campo := range []string{
		"vec.ct.seleccion-oferta.intencion.v1", i.OrganizacionRef, i.ExpedienteRef,
		strconv.FormatUint(i.VersionExpediente, 10), i.ClaveIdempotencia, i.ActorRef, i.PerfilRef,
	} {
		_, _ = h.Write([]byte(strconv.Itoa(len([]byte(campo))) + ":" + campo))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ClaveBolsaSeleccionOfertaCT conserva la misma operación en todos los
// reintentos y separa el ámbito CT de claves elegidas directamente en Bolsa.
func ClaveBolsaSeleccionOfertaCT(i IntencionSeleccionOfertaCT) string {
	material := strings.Join([]string{i.OrganizacionRef, i.ExpedienteRef,
		strconv.FormatUint(i.VersionExpediente, 10), i.ClaveIdempotencia}, "\x1f")
	suma := sha256.Sum256([]byte(material))
	return "ct-oferta:" + hex.EncodeToString(suma[:])
}

// DatosOfertaSeleccionCT es la proyección neutra de la necesidad RPT.
type DatosOfertaSeleccionCT struct {
	Categoria      string `json:"categoria"`
	Centro         string `json:"centro"`
	FechaInicio    string `json:"fecha_inicio"`
	FechaFin       string `json:"fecha_fin,omitempty"`
	Descripcion    string `json:"descripcion"`
	ModalidadClave string `json:"modalidad_clave,omitempty"`
	CausaClave     string `json:"causa_clave,omitempty"`
}

func (d DatosOfertaSeleccionCT) Validar() error {
	if (d.ModalidadClave != "" || d.CausaClave != "") &&
		(!domain.ClaveCatalogo(d.ModalidadClave).Valida() || !domain.ClaveCatalogo(d.CausaClave).Valida()) {
		return ErrSeleccionOfertaCTInvalida
	}
	for _, texto := range []string{d.Categoria, d.Centro, d.Descripcion} {
		if len(texto) < 2 || len(texto) > 2000 || strings.TrimSpace(texto) != texto {
			return ErrSeleccionOfertaCTInvalida
		}
	}
	inicio, err := time.Parse(time.DateOnly, d.FechaInicio)
	if err != nil || inicio.Format(time.DateOnly) != d.FechaInicio {
		return ErrSeleccionOfertaCTInvalida
	}
	if d.FechaFin != "" {
		fin, err := time.Parse(time.DateOnly, d.FechaFin)
		if err != nil || fin.Format(time.DateOnly) != d.FechaFin || fin.Before(inicio) {
			return ErrSeleccionOfertaCTInvalida
		}
	}
	return nil
}

// ComandoOfertaBolsaCT inmoviliza la captura B110 entera, incluidos corte y
// entradas. La consulta posterior devuelve exactamente este JSON durable;
// una captura nueva no vale para reintentar la operación original.
type ComandoOfertaBolsaCT struct {
	OrganizacionRef   string                 `json:"organizacion_ref"`
	ExpedienteRef     string                 `json:"expediente_ref"`
	VersionExpediente uint64                 `json:"version_expediente"`
	ActorRef          string                 `json:"actor_ref"`
	BolsaRef          string                 `json:"bolsa_ref"`
	NecesidadRef      string                 `json:"necesidad_ref"`
	NumeroPlazas      int                    `json:"numero_plazas"`
	NumeroElegibles   int                    `json:"numero_elegibles"`
	Datos             DatosOfertaSeleccionCT `json:"datos"`
	OrigenCT          json.RawMessage        `json:"origen_ct"`
	ClaveBolsa        string                 `json:"clave_bolsa"`
	InicioPlazo       string                 `json:"inicio_plazo"`
}

func (c ComandoOfertaBolsaCT) ValidarPara(i IntencionSeleccionOfertaCT) error {
	if i.Validar() != nil || c.OrganizacionRef != i.OrganizacionRef ||
		c.ExpedienteRef != i.ExpedienteRef || c.VersionExpediente != i.VersionExpediente ||
		c.ActorRef != i.ActorRef || !domain.ReferenciaOpacaValida(c.BolsaRef) ||
		!domain.ReferenciaOpacaValida(c.NecesidadRef) || c.NumeroPlazas < 2 ||
		c.NumeroPlazas > 100 || c.NumeroElegibles < c.NumeroPlazas ||
		c.Datos.Validar() != nil || len(c.OrigenCT) < 2 || len(c.OrigenCT) > 1<<20 ||
		!json.Valid(c.OrigenCT) || c.OrigenCT[0] != '{' ||
		!claveBolsaSeleccionOfertaCT.MatchString(c.ClaveBolsa) ||
		c.ClaveBolsa != ClaveBolsaSeleccionOfertaCT(i) ||
		c.InicioPlazo != InicioSeleccionOfertaPublicacion {
		return ErrSeleccionOfertaCTInvalida
	}
	// El comando histórico se recupera sin completar campos. En el formato
	// nuevo, la captura en memoria debe coincidir con los datos congelados.
	if c.Datos.ModalidadClave != "" && i.ModalidadClave != "" &&
		(c.Datos.ModalidadClave != i.ModalidadClave || c.Datos.CausaClave != i.CausaClave) {
		return ErrSeleccionOfertaCTInvalida
	}
	return nil
}

type SituacionSeleccionOfertaCT string

const (
	SeleccionOfertaAusente       SituacionSeleccionOfertaCT = "ausente"
	SeleccionOfertaReservada     SituacionSeleccionOfertaCT = "reservada"
	SeleccionOfertaOcupada       SituacionSeleccionOfertaCT = "ocupada"
	SeleccionOfertaColision      SituacionSeleccionOfertaCT = "colision"
	SeleccionOfertaIndeterminada SituacionSeleccionOfertaCT = "indeterminada"
	SeleccionOfertaConfirmada    SituacionSeleccionOfertaCT = "confirmada"
	SeleccionOfertaDescartada    SituacionSeleccionOfertaCT = "descartada"
)

type EstadoSeleccionOfertaCT struct {
	Situacion      SituacionSeleccionOfertaCT
	ReservaRef     string
	VentanaAbierta bool
	Comando        ComandoOfertaBolsaCT
	Recibo         ReciboSeleccionOfertaCT
	ReciboDescarte ReciboDescarteOfertaCT
}

// La recuperación explícita produce un solo terminal y nunca inicia una
// selección nueva ni sustituye la clave original.
type ResultadoRecuperacionOfertaCT struct {
	Oferta   *ReciboSeleccionOfertaCT
	Descarte *ReciboDescarteOfertaCT
}

type ReservaSeleccionOfertaCT struct {
	Intencion  IntencionSeleccionOfertaCT
	ReservaRef string
	Comando    ComandoOfertaBolsaCT
}

type SolicitudAutorizacionSeleccionOfertaCT struct {
	Etapa     string
	Intencion IntencionSeleccionOfertaCT
	Comando   *ComandoOfertaBolsaCT
	Oferta    *OfertaPublicadaSeleccionCT
}

type AutorizacionSeleccionOfertaCT struct {
	ActorRef, PerfilRef string
	// MaterialCanonico contiene los mismos bytes UTF-8 que el emisor ligó a
	// la decisión V3. El adaptador SQL los transmite sin reconstruir JSON.
	MaterialCanonico string
	Material         puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// CanonEtapaSeleccionOfertaCT fija los bytes que comparten el emisor V3 y el
// adaptador SQL. La base compara el JSONB completo con la reserva durable y
// verifica SHA-256 del texto exacto contra la decisión consumida.
func CanonEtapaSeleccionOfertaCT(s SolicitudAutorizacionSeleccionOfertaCT) (string, error) {
	if s.Intencion.Validar() != nil || (s.Etapa != EtapaSeleccionOfertaConsultar &&
		s.Etapa != EtapaSeleccionOfertaReservar && s.Etapa != EtapaSeleccionOfertaAbrirVentana &&
		s.Etapa != EtapaSeleccionOfertaConfirmar) ||
		(s.Etapa == EtapaSeleccionOfertaConsultar && (s.Comando != nil || s.Oferta != nil)) ||
		(s.Etapa != EtapaSeleccionOfertaConsultar && s.Comando == nil) ||
		(s.Etapa != EtapaSeleccionOfertaConfirmar && s.Oferta != nil) ||
		(s.Etapa == EtapaSeleccionOfertaConfirmar && s.Oferta == nil) {
		return "", ErrSeleccionOfertaCTInvalida
	}
	if s.Comando != nil && s.Comando.ValidarPara(s.Intencion) != nil {
		return "", ErrSeleccionOfertaCTInvalida
	}
	if s.Oferta != nil && s.Oferta.ValidarPara(*s.Comando) != nil {
		return "", ErrSeleccionOfertaCTInvalida
	}
	b, err := json.Marshal(struct {
		Esquema           string                      `json:"esquema"`
		Etapa             string                      `json:"etapa"`
		OrganizacionRef   string                      `json:"organizacion_ref"`
		ExpedienteRef     string                      `json:"expediente_ref"`
		VersionExpediente uint64                      `json:"version_expediente"`
		ClaveIdempotencia string                      `json:"clave_idempotencia"`
		HuellaIntencion   string                      `json:"huella_intencion"`
		CategoriaRef      string                      `json:"categoria_ref"`
		CentroRef         string                      `json:"centro_ref"`
		Comando           *ComandoOfertaBolsaCT       `json:"comando,omitempty"`
		Oferta            *OfertaPublicadaSeleccionCT `json:"oferta,omitempty"`
	}{"vec.ct.seleccion-oferta.etapa.v1", s.Etapa,
		s.Intencion.OrganizacionRef, s.Intencion.ExpedienteRef,
		s.Intencion.VersionExpediente, s.Intencion.ClaveIdempotencia,
		s.Intencion.HuellaIntencion, s.Intencion.CategoriaRef, s.Intencion.CentroRef,
		s.Comando, s.Oferta})
	if err != nil || len(b) > 1<<20 {
		return "", ErrSeleccionOfertaCTInvalida
	}
	return string(b), nil
}

type AutorizadorSeleccionOfertaCT interface {
	AutorizarSeleccionOfertaCT(context.Context, SolicitudAutorizacionSeleccionOfertaCT) (AutorizacionSeleccionOfertaCT, error)
}

type SolicitudAutorizacionDescarteOfertaCT struct {
	Intencion IntencionSeleccionOfertaCT
	Comando   ComandoOfertaBolsaCT
	Cierre    ReciboCierreOfertaCT
}

type AutorizacionDescarteSeleccionOfertaCT struct {
	ActorRef, PerfilRef string
	MaterialCanonico    string
	Material            puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// CanonDescarteSeleccionOfertaCT incluye el comando congelado y el recibo
// completo B121. El emisor CT V3 firma estos bytes y SQL recibe el texto
// intacto; el perfil que firma el descarte es otro del perfil vinculador.
func CanonDescarteSeleccionOfertaCT(s SolicitudAutorizacionDescarteOfertaCT) (string, error) {
	i, c, cierre := s.Intencion, s.Comando, s.Cierre
	if i.Validar() != nil || c.ValidarPara(i) != nil || cierre.ValidarPara(i, c) != nil {
		return "", ErrSeleccionOfertaCTInvalida
	}
	b, err := json.Marshal(struct {
		Esquema           string               `json:"esquema"`
		OrganizacionRef   string               `json:"organizacion_ref"`
		ExpedienteRef     string               `json:"expediente_ref"`
		VersionExpediente uint64               `json:"version_expediente"`
		ClaveIdempotencia string               `json:"clave_idempotencia"`
		HuellaIntencion   string               `json:"huella_intencion"`
		ActorRef          string               `json:"actor_ref"`
		CategoriaRef      string               `json:"categoria_ref"`
		CentroRef         string               `json:"centro_ref"`
		Comando           ComandoOfertaBolsaCT `json:"comando"`
		Cierre            ReciboCierreOfertaCT `json:"cierre"`
	}{"vec.ct.seleccion-oferta.descarte.v1", i.OrganizacionRef, i.ExpedienteRef,
		i.VersionExpediente, i.ClaveIdempotencia, i.HuellaIntencion, i.ActorRef,
		i.CategoriaRef, i.CentroRef, c, cierre})
	if err != nil || len(b) > 1<<20 {
		return "", ErrSeleccionOfertaCTInvalida
	}
	return string(b), nil
}

type AutorizadorDescarteSeleccionOfertaCT interface {
	AutorizarDescarteOfertaCT(context.Context, SolicitudAutorizacionDescarteOfertaCT) (AutorizacionDescarteSeleccionOfertaCT, error)
}

// PrepararIntencion debe seguir devolviendo identidad y organización
// autorizadas para consultar un terminal histórico aunque CT haya avanzado.
// PrepararComando solo se invoca cuando no hay ejecución anterior.
type PreparadorSeleccionOfertaCT interface {
	PrepararIntencion(context.Context, string, uint64, string) (IntencionSeleccionOfertaCT, error)
	PrepararComando(context.Context, IntencionSeleccionOfertaCT) (ComandoOfertaBolsaCT, error)
}

// RegistroSeleccionOfertaCT se monta sobre la reserva durable de selección,
// con tipo OFERTA separado del llamamiento singular. Reservar toma el mismo
// bloqueo de cabecera que cancelación; ConfirmarVinculo autentica B120 y
// confirma vínculo propio + terminal CT en una única transacción V3.
type RegistroSeleccionOfertaCT interface {
	Consultar(context.Context, IntencionSeleccionOfertaCT, AutorizacionSeleccionOfertaCT) (EstadoSeleccionOfertaCT, error)
	Reservar(context.Context, IntencionSeleccionOfertaCT, ComandoOfertaBolsaCT, AutorizacionSeleccionOfertaCT) (EstadoSeleccionOfertaCT, error)
	AbrirVentana(context.Context, ReservaSeleccionOfertaCT, AutorizacionSeleccionOfertaCT) error
	ConfirmarVinculo(context.Context, ReservaSeleccionOfertaCT, OfertaPublicadaSeleccionCT, AutorizacionSeleccionOfertaCT) (ReciboSeleccionOfertaCT, error)
}

// El descarte es otro efecto nominal y consume una autorización CT propia.
// Se mantiene fuera del registro anterior para no ampliar sus concesiones.
type RegistroDescarteSeleccionOfertaCT interface {
	Descartar(context.Context, ReservaSeleccionOfertaCT, ReciboCierreOfertaCT, AutorizacionDescarteSeleccionOfertaCT) (ReciboDescarteOfertaCT, error)
}
