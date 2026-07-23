package ports

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrPreparacionAltaInvalida  = errors.New("contratacion temporal: preparacion de alta invalida")
	ErrOrdenAltaInvalida        = errors.New("contratacion temporal: orden de alta invalida")
	ErrPersistenciaNoDisponible = errors.New("contratacion temporal: persistencia no disponible")
	ErrClaveIdempotenciaUsada   = errors.New("contratacion temporal: clave de idempotencia usada con otros datos")
)

const MaximoGeneracionesHMACAlta = 4

// La clave debe generarla cada cliente con CSPRNG y conservarse solo durante
// el reintento. El formato UUIDv4 canónico descarta etiquetas humanas, formas
// no canónicas y el centinela nulo; la sintaxis no prueba por sí sola la
// calidad del generador, que se exige en cada adaptador de entrada.
var patronClaveIdempotencia = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

func claveIdempotenciaValida(valor string) bool {
	return patronClaveIdempotencia.MatchString(valor) &&
		valor != "00000000-0000-4000-8000-000000000000"
}

// ClaveIdempotenciaValida expone la misma gramática a los orquestadores que
// deben fallar antes de invocar selladores o persistencia.
func ClaveIdempotenciaValida(valor string) bool {
	return claveIdempotenciaValida(valor)
}

type ReferenciasAlta struct {
	ExpedienteRef string
	NumeroVisible string
	ReciboRef     string
}

func (r ReferenciasAlta) Validar() error {
	if !domain.ReferenciaOpacaValida(r.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.NumeroExpedienteValido(r.NumeroVisible) {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

type MaterialHuellaAlta struct {
	OrganizacionRef string
	ActorRef        string
	PerfilRef       string
	Flujo           domain.ReferenciaFlujo
	Solicitud       domain.SolicitudCentro
}

func (m MaterialHuellaAlta) Validar() error {
	if !domain.ReferenciaOpacaValida(m.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(m.ActorRef) ||
		!domain.ReferenciaOpacaValida(m.PerfilRef) ||
		m.Flujo.Validar() != nil || m.Solicitud.Validar() != nil {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

// DatosIdentidadHMACAlta es una proyección defensiva, no una capacidad. Solo
// ColeccionIdentidadesHMACAlta puede transportar estos pares al caso de uso.
type DatosIdentidadHMACAlta struct {
	Generacion             uint32
	AmbitoIdempotenciaHMAC string
	HuellaPeticionHMAC     string
}

func validarDatosIdentidadHMACAlta(i DatosIdentidadHMACAlta) error {
	generacionAmbito, ambitoValido := generacionSelloHMACAlta(
		i.AmbitoIdempotenciaHMAC,
		"vec.contratacion-temporal.ambito-idempotencia/v",
	)
	generacionHuella, huellaValida := generacionSelloHMACAlta(
		i.HuellaPeticionHMAC,
		"vec.contratacion-temporal.huella-peticion/v",
	)
	if !ambitoValido || !huellaValida ||
		generacionAmbito != generacionHuella ||
		i.Generacion != generacionAmbito {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

// DatosColeccionIdentidadesHMACAlta expone una copia para cotejo interno.
// Los retenidos están ordenados de generación mayor a menor.
type DatosColeccionIdentidadesHMACAlta struct {
	Activa    DatosIdentidadHMACAlta
	Retenidas []DatosIdentidadHMACAlta
}

type datosColeccionIdentidadesHMACAlta struct {
	activa    DatosIdentidadHMACAlta
	retenidas []DatosIdentidadHMACAlta
}

// ColeccionIdentidadesHMACAlta es el resultado nominal opaco del llavero. El
// cliente no puede aportarlo en SolicitudRegistrarExpediente.
type ColeccionIdentidadesHMACAlta struct {
	datos *datosColeccionIdentidadesHMACAlta
}

func NuevaColeccionIdentidadesHMACAlta(
	activa DatosIdentidadHMACAlta,
	retenidas []DatosIdentidadHMACAlta,
) (ColeccionIdentidadesHMACAlta, error) {
	if validarDatosIdentidadHMACAlta(activa) != nil ||
		len(retenidas)+1 > MaximoGeneracionesHMACAlta {
		return ColeccionIdentidadesHMACAlta{}, ErrPreparacionAltaInvalida
	}
	copia := make([]DatosIdentidadHMACAlta, len(retenidas))
	anterior := activa.Generacion
	for _, retenida := range retenidas {
		if validarDatosIdentidadHMACAlta(retenida) != nil ||
			retenida.Generacion >= anterior {
			return ColeccionIdentidadesHMACAlta{}, ErrPreparacionAltaInvalida
		}
		anterior = retenida.Generacion
	}
	copy(copia, retenidas)
	return ColeccionIdentidadesHMACAlta{
		datos: &datosColeccionIdentidadesHMACAlta{
			activa: activa, retenidas: copia,
		},
	}, nil
}

func (c ColeccionIdentidadesHMACAlta) Datos() (
	DatosColeccionIdentidadesHMACAlta,
	error,
) {
	if c.datos == nil {
		return DatosColeccionIdentidadesHMACAlta{}, ErrPreparacionAltaInvalida
	}
	datos := DatosColeccionIdentidadesHMACAlta{
		Activa: c.datos.activa,
		Retenidas: append(
			[]DatosIdentidadHMACAlta(nil),
			c.datos.retenidas...,
		),
	}
	reconstruida, err := NuevaColeccionIdentidadesHMACAlta(
		datos.Activa,
		datos.Retenidas,
	)
	if err != nil || reconstruida.datos.activa != datos.Activa {
		return DatosColeccionIdentidadesHMACAlta{}, ErrPreparacionAltaInvalida
	}
	return datos, nil
}

func (c ColeccionIdentidadesHMACAlta) Validar() error {
	_, err := c.Datos()
	return err
}

func (c ColeccionIdentidadesHMACAlta) Clonar() (ColeccionIdentidadesHMACAlta, error) {
	datos, err := c.Datos()
	if err != nil {
		return ColeccionIdentidadesHMACAlta{}, err
	}
	return NuevaColeccionIdentidadesHMACAlta(datos.Activa, datos.Retenidas)
}

// ContienePar exige que ámbito y huella pertenezcan juntos a una generación
// autorizada. No admite combinar dos generaciones válidas por separado.
func (c ColeccionIdentidadesHMACAlta) ContienePar(ambito, huella string) bool {
	datos, err := c.Datos()
	if err != nil {
		return false
	}
	coincide := func(identidad DatosIdentidadHMACAlta) bool {
		return sellosHMACIguales(identidad.AmbitoIdempotenciaHMAC, ambito) &&
			sellosHMACIguales(identidad.HuellaPeticionHMAC, huella)
	}
	encontrado := coincide(datos.Activa)
	for _, retenida := range datos.Retenidas {
		encontrado = coincide(retenida) || encontrado
	}
	return encontrado
}

type SolicitudDerivarIdentidadesHMACAlta struct {
	ClaveIdempotencia string
	Material          MaterialHuellaAlta
}

func (s SolicitudDerivarIdentidadesHMACAlta) Validar() error {
	if !claveIdempotenciaValida(s.ClaveIdempotencia) ||
		s.Material.Validar() != nil {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

// DerivadorIdentidadesHMACAlta resuelve el llavero una sola vez para impedir
// que ámbito y huella procedan de generaciones o configuraciones distintas.
type DerivadorIdentidadesHMACAlta interface {
	DerivarIdentidadesHMACAlta(
		context.Context,
		SolicitudDerivarIdentidadesHMACAlta,
	) (ColeccionIdentidadesHMACAlta, error)
}

// DerivadorHuellaAlta usa una clave gestionada fuera del proceso. Nunca
// persiste el material en claro como sustituto de la solicitud.
type DerivadorHuellaAlta interface {
	DerivarHuellaAlta(context.Context, MaterialHuellaAlta) (string, error)
}

type SolicitudPrepararAlta struct {
	ClaveIdempotencia string
	IdentidadesHMAC   ColeccionIdentidadesHMACAlta
	OrganizacionRef   string
	ActorRef          string
	PerfilRef         string
}

func (s SolicitudPrepararAlta) Validar() error {
	if !claveIdempotenciaValida(s.ClaveIdempotencia) ||
		s.IdentidadesHMAC.Validar() != nil ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

type EstadoPreparacionAlta string

const (
	PreparacionReservada  EstadoPreparacionAlta = "reservada"
	PreparacionConfirmada EstadoPreparacionAlta = "confirmada"
)

type PreparacionAlta struct {
	ReservaRef             string
	Referencias            ReferenciasAlta
	AmbitoIdempotenciaHMAC string
	HuellaPeticionHMAC     string
	OrganizacionRef        string
	ActorRef               string
	PerfilRef              string
	Estado                 EstadoPreparacionAlta
	ReciboConfirmado       *ReciboAlta
}

func (p PreparacionAlta) ValidarPara(solicitud SolicitudPrepararAlta) error {
	if solicitud.Validar() != nil || !domain.ReferenciaOpacaValida(p.ReservaRef) ||
		p.Referencias.Validar() != nil ||
		!solicitud.IdentidadesHMAC.ContienePar(
			p.AmbitoIdempotenciaHMAC,
			p.HuellaPeticionHMAC,
		) ||
		p.OrganizacionRef != solicitud.OrganizacionRef ||
		p.ActorRef != solicitud.ActorRef || p.PerfilRef != solicitud.PerfilRef ||
		(p.Estado != PreparacionReservada && p.Estado != PreparacionConfirmada) {
		return ErrPreparacionAltaInvalida
	}
	if p.Estado == PreparacionReservada && p.ReciboConfirmado != nil {
		return ErrPreparacionAltaInvalida
	}
	if p.Estado == PreparacionConfirmada {
		if p.ReciboConfirmado == nil ||
			p.ReciboConfirmado.ValidarEstructura() != nil ||
			p.ReciboConfirmado.ExpedienteRef != p.Referencias.ExpedienteRef ||
			p.ReciboConfirmado.NumeroVisible != p.Referencias.NumeroVisible ||
			p.ReciboConfirmado.ReciboRef != p.Referencias.ReciboRef {
			return ErrPreparacionAltaInvalida
		}
	}
	return nil
}

func generacionSelloHMACAlta(
	sello string,
	prefijoReferencia string,
) (uint32, bool) {
	if !SelloHMACSHA256Valido(sello) {
		return 0, false
	}
	partes := strings.Split(sello, ":")
	if len(partes) != 3 || !strings.HasPrefix(partes[1], prefijoReferencia) {
		return 0, false
	}
	texto := strings.TrimPrefix(partes[1], prefijoReferencia)
	if texto == "" || texto[0] == '0' {
		return 0, false
	}
	generacion, err := strconv.ParseUint(texto, 10, 32)
	return uint32(generacion), err == nil && generacion > 0
}

type PreparadorAltaIdempotente interface {
	PrepararAlta(context.Context, SolicitudPrepararAlta) (PreparacionAlta, error)
}

type Reloj interface {
	Ahora() time.Time
}

type OrdenConfirmarAlta struct {
	datos *datosOrdenConfirmarAlta
}

type datosOrdenConfirmarAlta struct {
	expediente              domain.Expediente
	solicitudAutorizacionV3 dominiovec.SolicitudAutorizacionLigadaV3
	decisionAutorizacionV3  dominiovec.DecisionAutorizacionLigadaV3
	confirmacionRegistroV3  puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	identidadesHMAC         ColeccionIdentidadesHMACAlta
	preparacion             PreparacionAlta
	correlacionV3Ref        string
}

// DatosOrdenConfirmarAlta es exclusivamente la entrada nominal del
// constructor. La correlación no se acepta como texto: se deriva de la
// SolicitudAutorizacionV3 generada por el servidor.
type DatosOrdenConfirmarAlta struct {
	Expediente              domain.Expediente
	SolicitudAutorizacionV3 dominiovec.SolicitudAutorizacionLigadaV3
	DecisionAutorizacionV3  dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionRegistroV3  puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	IdentidadesHMAC         ColeccionIdentidadesHMACAlta
	Preparacion             PreparacionAlta
}

// EvidenciaOrdenConfirmarAlta propaga a persistencia la única correlación V3
// acuñada por servidor. No existe un campo de entrada que permita sustituirla.
type EvidenciaOrdenConfirmarAlta struct {
	Expediente              domain.Expediente
	SolicitudAutorizacionV3 dominiovec.SolicitudAutorizacionLigadaV3
	DecisionAutorizacionV3  dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionRegistroV3  puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	IdentidadesHMAC         ColeccionIdentidadesHMACAlta
	Preparacion             PreparacionAlta
	CorrelacionV3Ref        string
}

func NuevaOrdenConfirmarAlta(datos DatosOrdenConfirmarAlta) (OrdenConfirmarAlta, error) {
	if datos.Expediente.Validar() != nil {
		return OrdenConfirmarAlta{}, ErrOrdenAltaInvalida
	}
	solicitudV3, err := datos.SolicitudAutorizacionV3.Datos()
	correlacionV3Ref, errCorrelacion := solicitudV3.Correlacion.ValorCanonico()
	vinculo, errVinculo := solicitudV3.VinculoAutenticacionActor.Datos()
	concedida, _, errDecision := datos.DecisionAutorizacionV3.Resultado()
	emitidaEn, validaHasta, errVentana := datos.DecisionAutorizacionV3.VentanaValidez()
	confirmacionV3, errConfirmacion := datos.ConfirmacionRegistroV3.Datos()
	huellaDecision, errHuella := dominiovec.HuellaSHA256DecisionAutorizacionV3(
		datos.DecisionAutorizacionV3,
	)
	identidadesHMAC, errIdentidades := datos.IdentidadesHMAC.Clonar()
	datosIdentidadesHMAC, errDatosIdentidades := identidadesHMAC.Datos()
	if err != nil || errCorrelacion != nil || errVinculo != nil || errDecision != nil ||
		errVentana != nil || errConfirmacion != nil || errHuella != nil ||
		errIdentidades != nil || errDatosIdentidades != nil ||
		!concedida ||
		datos.DecisionAutorizacionV3.ValidarPara(datos.SolicitudAutorizacionV3) != nil ||
		datos.Preparacion.Estado != PreparacionReservada ||
		!domain.ReferenciaOpacaValida(datos.Preparacion.ReservaRef) ||
		datos.Preparacion.Referencias.Validar() != nil ||
		!SelloHMACSHA256Valido(datos.Preparacion.AmbitoIdempotenciaHMAC) ||
		!SelloHMACSHA256Valido(datos.Preparacion.HuellaPeticionHMAC) ||
		datos.Preparacion.OrganizacionRef != datos.Expediente.OrganizacionRef ||
		datos.Preparacion.ActorRef != vinculo.PrincipalID ||
		datos.Preparacion.PerfilRef != vinculo.PerfilActivoRef ||
		datos.Expediente.Actuaciones[0].ActorRef != vinculo.PrincipalID ||
		datos.Preparacion.ReciboConfirmado != nil ||
		datos.Preparacion.Referencias.ExpedienteRef != datos.Expediente.Referencia ||
		datos.Preparacion.Referencias.NumeroVisible != datos.Expediente.NumeroVisible ||
		datos.Preparacion.Referencias.ReciboRef != datos.Expediente.Actuaciones[0].ReciboRef ||
		solicitudV3.Accion != AccionCrearSolicitud ||
		solicitudV3.Finalidad != FinalidadCrearSolicitud ||
		solicitudV3.Recurso.ModuloID != ModuloContratacion ||
		solicitudV3.Recurso.Tipo != TipoRecursoExpediente ||
		len(solicitudV3.Recurso.Ambitos) != 3 ||
		len(solicitudV3.Recurso.Atributos) != 4 ||
		!sellosHMACIguales(
			solicitudV3.Recurso.Referencia,
			datosIdentidadesHMAC.Activa.AmbitoIdempotenciaHMAC,
		) ||
		!identidadesHMAC.ContienePar(
			datos.Preparacion.AmbitoIdempotenciaHMAC,
			datos.Preparacion.HuellaPeticionHMAC,
		) ||
		solicitudV3.Recurso.Ambitos["organizacion_ref"] != datos.Expediente.OrganizacionRef ||
		solicitudV3.Recurso.Ambitos["centro_ref"] != datos.Expediente.Solicitud.CentroRef ||
		solicitudV3.Recurso.Ambitos["categoria_ref"] != datos.Expediente.Solicitud.CategoriaRef ||
		solicitudV3.Recurso.Atributos["flujo_ref"] != datos.Expediente.Flujo.DefinicionRef ||
		solicitudV3.Recurso.Atributos["flujo_version"] !=
			formatearVersionFlujo(datos.Expediente.Flujo.Version) ||
		solicitudV3.Recurso.Atributos["flujo_huella_sha256"] !=
			datos.Expediente.Flujo.HuellaSHA256 ||
		!sellosHMACIguales(
			solicitudV3.Recurso.Atributos[AtributoHuellaPeticionHMACActiva],
			datosIdentidadesHMAC.Activa.HuellaPeticionHMAC,
		) ||
		confirmacionV3.DecisionRef == "" ||
		confirmacionV3.DecisionHuellaSHA256 != huellaDecision ||
		!confirmacionV3.EmitidaEn.Equal(emitidaEn) ||
		!confirmacionV3.ValidaHasta.Equal(validaHasta) ||
		!datos.ConfirmacionRegistroV3.DentroDeVentanaEn(datos.Expediente.CreadoEn) {
		return OrdenConfirmarAlta{}, ErrOrdenAltaInvalida
	}
	return OrdenConfirmarAlta{datos: &datosOrdenConfirmarAlta{
		expediente:              datos.Expediente.Clonar(),
		solicitudAutorizacionV3: datos.SolicitudAutorizacionV3,
		decisionAutorizacionV3:  datos.DecisionAutorizacionV3,
		confirmacionRegistroV3:  datos.ConfirmacionRegistroV3,
		identidadesHMAC:         identidadesHMAC,
		preparacion:             datos.Preparacion,
		correlacionV3Ref:        correlacionV3Ref,
	}}, nil
}

func (o OrdenConfirmarAlta) Datos() (EvidenciaOrdenConfirmarAlta, error) {
	if o.datos == nil {
		return EvidenciaOrdenConfirmarAlta{}, ErrOrdenAltaInvalida
	}
	identidadesHMAC, err := o.datos.identidadesHMAC.Clonar()
	if err != nil {
		return EvidenciaOrdenConfirmarAlta{}, ErrOrdenAltaInvalida
	}
	entrada := DatosOrdenConfirmarAlta{
		Expediente:              o.datos.expediente.Clonar(),
		SolicitudAutorizacionV3: o.datos.solicitudAutorizacionV3,
		DecisionAutorizacionV3:  o.datos.decisionAutorizacionV3,
		ConfirmacionRegistroV3:  o.datos.confirmacionRegistroV3,
		IdentidadesHMAC:         identidadesHMAC,
		Preparacion:             o.datos.preparacion,
	}
	reconstruida, err := NuevaOrdenConfirmarAlta(entrada)
	if err != nil || reconstruida.datos.correlacionV3Ref != o.datos.correlacionV3Ref {
		return EvidenciaOrdenConfirmarAlta{}, ErrOrdenAltaInvalida
	}
	return EvidenciaOrdenConfirmarAlta{
		Expediente:              entrada.Expediente,
		SolicitudAutorizacionV3: entrada.SolicitudAutorizacionV3,
		DecisionAutorizacionV3:  entrada.DecisionAutorizacionV3,
		ConfirmacionRegistroV3:  entrada.ConfirmacionRegistroV3,
		IdentidadesHMAC:         identidadesHMAC,
		Preparacion:             entrada.Preparacion,
		CorrelacionV3Ref:        o.datos.correlacionV3Ref,
	}, nil
}

func formatearVersionFlujo(version uint64) string {
	return strconv.FormatUint(version, 10)
}

type ReciboAlta struct {
	ExpedienteRef string    `json:"expediente_ref"`
	NumeroVisible string    `json:"numero_visible"`
	Version       uint64    `json:"version"`
	ReciboRef     string    `json:"recibo_ref"`
	AuditoriaRef  string    `json:"auditoria_ref"`
	EventoRef     string    `json:"evento_ref"`
	ConfirmadaEn  time.Time `json:"confirmada_en"`
}

func (r ReciboAlta) ValidarEstructura() error {
	if !domain.ReferenciaOpacaValida(r.ExpedienteRef) ||
		!domain.NumeroExpedienteValido(r.NumeroVisible) || r.Version == 0 ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.EventoRef) ||
		!domain.InstanteUTCCanonico(r.ConfirmadaEn) {
		return ErrPersistenciaNoDisponible
	}
	return nil
}

func (r ReciboAlta) ValidarPara(expediente domain.Expediente) error {
	if expediente.Validar() != nil || r.ExpedienteRef != expediente.Referencia ||
		r.NumeroVisible != expediente.NumeroVisible || r.Version != expediente.Version ||
		r.ValidarEstructura() != nil ||
		r.ConfirmadaEn.Before(expediente.ActualizadoEn) ||
		len(expediente.Actuaciones) == 0 ||
		r.ReciboRef != expediente.Actuaciones[0].ReciboRef {
		return ErrPersistenciaNoDisponible
	}
	return nil
}

// TransaccionAltas debe cotejar y consumir la autorización, confirmar la
// reserva, el expediente, la auditoría y el outbox en un único COMMIT.
type TransaccionAltas interface {
	ConfirmarAlta(context.Context, OrdenConfirmarAlta) (ReciboAlta, error)
}
