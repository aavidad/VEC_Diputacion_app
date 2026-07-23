package ports

import (
	"context"
	"errors"
	"regexp"
	"strconv"
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

// DerivadorHuellaAlta usa una clave gestionada fuera del proceso. Nunca
// persiste el material en claro como sustituto de la solicitud.
type DerivadorHuellaAlta interface {
	DerivarHuellaAlta(
		context.Context,
		MaterialHuellaAlta,
	) (ColeccionSellosHMAC, error)
}

type SolicitudPrepararAlta struct {
	ClaveIdempotencia   string
	HuellasPeticionHMAC ColeccionSellosHMAC
	OrganizacionRef     string
	ActorRef            string
	PerfilRef           string
}

func (s SolicitudPrepararAlta) Validar() error {
	if !claveIdempotenciaValida(s.ClaveIdempotencia) ||
		s.HuellasPeticionHMAC.ValidarDominio(
			"vec.contratacion-temporal.huella-peticion",
		) != nil ||
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
		!SelloHMACSHA256Valido(p.AmbitoIdempotenciaHMAC) ||
		!solicitud.HuellasPeticionHMAC.Contiene(p.HuellaPeticionHMAC) ||
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
	ambitosIdempotenciaHMAC ColeccionSellosHMAC
	huellasPeticionHMAC     ColeccionSellosHMAC
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
	AmbitosIdempotenciaHMAC ColeccionSellosHMAC
	HuellasPeticionHMAC     ColeccionSellosHMAC
	Preparacion             PreparacionAlta
}

// EvidenciaOrdenConfirmarAlta propaga a persistencia la única correlación V3
// acuñada por servidor. No existe un campo de entrada que permita sustituirla.
type EvidenciaOrdenConfirmarAlta struct {
	Expediente              domain.Expediente
	SolicitudAutorizacionV3 dominiovec.SolicitudAutorizacionLigadaV3
	DecisionAutorizacionV3  dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionRegistroV3  puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	AmbitosIdempotenciaHMAC ColeccionSellosHMAC
	HuellasPeticionHMAC     ColeccionSellosHMAC
	Preparacion             PreparacionAlta
	CorrelacionV3Ref        string
}

func datosColeccionesHMACAltaAlineadas(
	ambitos ColeccionSellosHMAC,
	huellas ColeccionSellosHMAC,
) (
	DatosColeccionSellosHMAC,
	DatosColeccionSellosHMAC,
	bool,
) {
	if ambitos.ValidarDominio(
		"vec.contratacion-temporal.ambito-idempotencia",
	) != nil ||
		huellas.ValidarDominio(
			"vec.contratacion-temporal.huella-peticion",
		) != nil {
		return DatosColeccionSellosHMAC{}, DatosColeccionSellosHMAC{}, false
	}
	datosAmbitos, errAmbitos := ambitos.Datos()
	datosHuellas, errHuellas := huellas.Datos()
	if errAmbitos != nil || errHuellas != nil ||
		datosAmbitos.Activo.Generacion != datosHuellas.Activo.Generacion ||
		len(datosAmbitos.Retenidos) != len(datosHuellas.Retenidos) {
		return DatosColeccionSellosHMAC{}, DatosColeccionSellosHMAC{}, false
	}
	for indice := range datosAmbitos.Retenidos {
		if datosAmbitos.Retenidos[indice].Generacion !=
			datosHuellas.Retenidos[indice].Generacion {
			return DatosColeccionSellosHMAC{}, DatosColeccionSellosHMAC{}, false
		}
	}
	return datosAmbitos, datosHuellas, true
}

func datosColeccionesHMACAltaContienenPar(
	ambitos DatosColeccionSellosHMAC,
	huellas DatosColeccionSellosHMAC,
	ambito string,
	huella string,
) bool {
	coincide := func(
		candidatoAmbito SelloGeneracionalHMAC,
		candidataHuella SelloGeneracionalHMAC,
	) bool {
		return candidatoAmbito.Generacion == candidataHuella.Generacion &&
			sellosHMACIguales(candidatoAmbito.Valor, ambito) &&
			sellosHMACIguales(candidataHuella.Valor, huella)
	}
	encontrado := coincide(ambitos.Activo, huellas.Activo)
	for indice := range ambitos.Retenidos {
		encontrado = coincide(
			ambitos.Retenidos[indice],
			huellas.Retenidos[indice],
		) || encontrado
	}
	return encontrado
}

// ParActivoColeccionesHMACAlta proyecta el par activo solo después de
// comprobar que las dos capacidades nominales tienen las mismas generaciones.
func ParActivoColeccionesHMACAlta(
	ambitos ColeccionSellosHMAC,
	huellas ColeccionSellosHMAC,
) (string, string, error) {
	datosAmbitos, datosHuellas, validas :=
		datosColeccionesHMACAltaAlineadas(ambitos, huellas)
	if !validas {
		return "", "", ErrPreparacionAltaInvalida
	}
	return datosAmbitos.Activo.Valor, datosHuellas.Activo.Valor, nil
}

// ColeccionesHMACAltaContienenPar impide aceptar ámbito y huella válidos pero
// pertenecientes a generaciones distintas.
func ColeccionesHMACAltaContienenPar(
	ambitos ColeccionSellosHMAC,
	huellas ColeccionSellosHMAC,
	ambito string,
	huella string,
) bool {
	datosAmbitos, datosHuellas, validas :=
		datosColeccionesHMACAltaAlineadas(ambitos, huellas)
	return validas && datosColeccionesHMACAltaContienenPar(
		datosAmbitos,
		datosHuellas,
		ambito,
		huella,
	)
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
	datosAmbitosHMAC, datosHuellasHMAC, coleccionesHMACValidas :=
		datosColeccionesHMACAltaAlineadas(
			datos.AmbitosIdempotenciaHMAC,
			datos.HuellasPeticionHMAC,
		)
	if err != nil || errCorrelacion != nil || errVinculo != nil || errDecision != nil ||
		errVentana != nil || errConfirmacion != nil || errHuella != nil ||
		!coleccionesHMACValidas ||
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
			datosAmbitosHMAC.Activo.Valor,
		) ||
		!datosColeccionesHMACAltaContienenPar(
			datosAmbitosHMAC,
			datosHuellasHMAC,
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
			datosHuellasHMAC.Activo.Valor,
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
		ambitosIdempotenciaHMAC: datos.AmbitosIdempotenciaHMAC,
		huellasPeticionHMAC:     datos.HuellasPeticionHMAC,
		preparacion:             datos.Preparacion,
		correlacionV3Ref:        correlacionV3Ref,
	}}, nil
}

func (o OrdenConfirmarAlta) Datos() (EvidenciaOrdenConfirmarAlta, error) {
	if o.datos == nil {
		return EvidenciaOrdenConfirmarAlta{}, ErrOrdenAltaInvalida
	}
	entrada := DatosOrdenConfirmarAlta{
		Expediente:              o.datos.expediente.Clonar(),
		SolicitudAutorizacionV3: o.datos.solicitudAutorizacionV3,
		DecisionAutorizacionV3:  o.datos.decisionAutorizacionV3,
		ConfirmacionRegistroV3:  o.datos.confirmacionRegistroV3,
		AmbitosIdempotenciaHMAC: o.datos.ambitosIdempotenciaHMAC,
		HuellasPeticionHMAC:     o.datos.huellasPeticionHMAC,
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
		AmbitosIdempotenciaHMAC: entrada.AmbitosIdempotenciaHMAC,
		HuellasPeticionHMAC:     entrada.HuellasPeticionHMAC,
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
