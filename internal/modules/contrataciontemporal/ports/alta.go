package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrPreparacionAltaInvalida          = errors.New("contratacion temporal: preparacion de alta invalida")
	ErrOrdenAltaInvalida                = errors.New("contratacion temporal: orden de alta invalida")
	ErrPersistenciaNoDisponible         = errors.New("contratacion temporal: persistencia no disponible")
	ErrClaveIdempotenciaUsada           = errors.New("contratacion temporal: clave de idempotencia usada con otros datos")
	ErrMaterialConfirmacionAltaInvalido = errors.New(
		"contratacion temporal: material de confirmacion de alta invalido",
	)
	ErrProyeccionEfectoAltaInvalida = errors.New(
		"contratacion temporal: proyeccion de efecto de alta invalida",
	)
	ErrResultadoAltaIndeterminado = errors.New(
		"contratacion temporal: resultado de alta indeterminado",
	)
	ErrResultadoAltaNoConfiable = errors.New(
		"contratacion temporal: resultado de alta no confiable",
	)
)

// La clave debe generarla cada cliente con CSPRNG y conservarse solo durante
// el reintento. El formato UUIDv4 canónico descarta etiquetas humanas, formas
// no canónicas y el centinela nulo; la sintaxis no prueba por sí sola la
// calidad del generador, que se exige en cada adaptador de entrada.
var patronClaveIdempotencia = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

var patronHuellaSHA256Alta = regexp.MustCompile(`^[0-9a-f]{64}$`)

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

// CandidaturaAlta reúne referencias acuñadas por servidor antes de solicitar
// la decisión V3. No afirma reserva ni durabilidad y no concede autoridad.
type CandidaturaAlta struct {
	ReservaRef             string
	Referencias            ReferenciasAlta
	AmbitoIdempotenciaHMAC string
	HuellaPeticionHMAC     string
	OrganizacionRef        string
	ActorRef               string
	PerfilRef              string
}

func (c CandidaturaAlta) Validar() error {
	if !domain.ReferenciaOpacaValida(c.ReservaRef) ||
		c.Referencias.Validar() != nil ||
		!SelloHMACSHA256Valido(c.AmbitoIdempotenciaHMAC) ||
		!SelloHMACSHA256Valido(c.HuellaPeticionHMAC) ||
		!domain.ReferenciaOpacaValida(c.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(c.ActorRef) ||
		!domain.ReferenciaOpacaValida(c.PerfilRef) {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

type SolicitudProyectarEfectoAlta struct {
	Expediente  domain.Expediente
	Candidatura CandidaturaAlta
}

func (s SolicitudProyectarEfectoAlta) Validar() error {
	if s.Expediente.Validar() != nil || s.Candidatura.Validar() != nil ||
		len(s.Expediente.Actuaciones) != 1 ||
		s.Candidatura.Referencias.ExpedienteRef != s.Expediente.Referencia ||
		s.Candidatura.Referencias.NumeroVisible != s.Expediente.NumeroVisible ||
		s.Candidatura.Referencias.ReciboRef !=
			s.Expediente.Actuaciones[0].ReciboRef ||
		s.Candidatura.OrganizacionRef != s.Expediente.OrganizacionRef ||
		s.Candidatura.ActorRef != s.Expediente.Actuaciones[0].ActorRef {
		return ErrProyeccionEfectoAltaInvalida
	}
	return nil
}

type datosProyeccionEfectoAlta struct {
	contenido    []byte
	huellaSHA256 string
}

// ProyeccionEfectoAlta conserva los mismos bytes que se comprometieron en la
// decisión V3. La función SQL vuelve a validar forma, canon y semántica.
type ProyeccionEfectoAlta struct {
	datos *datosProyeccionEfectoAlta
}

func NuevaProyeccionEfectoAlta(
	contenido []byte,
) (ProyeccionEfectoAlta, error) {
	if len(contenido) < 256 || len(contenido) > 32*1024 {
		return ProyeccionEfectoAlta{}, ErrProyeccionEfectoAltaInvalida
	}
	suma := sha256.Sum256(contenido)
	return ProyeccionEfectoAlta{datos: &datosProyeccionEfectoAlta{
		contenido:    append([]byte(nil), contenido...),
		huellaSHA256: hex.EncodeToString(suma[:]),
	}}, nil
}

func (p ProyeccionEfectoAlta) Datos() ([]byte, string, error) {
	if p.datos == nil || len(p.datos.contenido) < 256 ||
		len(p.datos.contenido) > 32*1024 ||
		!patronHuellaSHA256Alta.MatchString(p.datos.huellaSHA256) {
		return nil, "", ErrProyeccionEfectoAltaInvalida
	}
	suma := sha256.Sum256(p.datos.contenido)
	if hex.EncodeToString(suma[:]) != p.datos.huellaSHA256 {
		return nil, "", ErrProyeccionEfectoAltaInvalida
	}
	return append([]byte(nil), p.datos.contenido...),
		p.datos.huellaSHA256, nil
}

type ProyectorEfectoAlta interface {
	ProyectarEfectoAlta(
		SolicitudProyectarEfectoAlta,
	) (ProyeccionEfectoAlta, error)
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
	resultadoContextoV2     dominiovec.ResultadoContextoActorRegistradoV2
	ambitosIdempotenciaHMAC ColeccionSellosHMAC
	huellasPeticionHMAC     ColeccionSellosHMAC
	candidatura             CandidaturaAlta
	proyeccionEfecto        ProyeccionEfectoAlta
	instanteConfirmacion    time.Time
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
	ResultadoContextoV2     dominiovec.ResultadoContextoActorRegistradoV2
	AmbitosIdempotenciaHMAC ColeccionSellosHMAC
	HuellasPeticionHMAC     ColeccionSellosHMAC
	Candidatura             CandidaturaAlta
	ProyeccionEfecto        ProyeccionEfectoAlta
	InstanteConfirmacion    time.Time
}

// EvidenciaOrdenConfirmarAlta propaga a persistencia la única correlación V3
// acuñada por servidor. No existe un campo de entrada que permita sustituirla.
type EvidenciaOrdenConfirmarAlta struct {
	Expediente              domain.Expediente
	SolicitudAutorizacionV3 dominiovec.SolicitudAutorizacionLigadaV3
	DecisionAutorizacionV3  dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionRegistroV3  puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	ResultadoContextoV2     dominiovec.ResultadoContextoActorRegistradoV2
	AmbitosIdempotenciaHMAC ColeccionSellosHMAC
	HuellasPeticionHMAC     ColeccionSellosHMAC
	Candidatura             CandidaturaAlta
	ProyeccionEfecto        ProyeccionEfectoAlta
	InstanteConfirmacion    time.Time
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
	resultadoContexto, errContexto := datos.ResultadoContextoV2.Clonar()
	concedida, _, errDecision := datos.DecisionAutorizacionV3.Resultado()
	emitidaEn, validaHasta, errVentana := datos.DecisionAutorizacionV3.VentanaValidez()
	confirmacionV3, errConfirmacion := datos.ConfirmacionRegistroV3.Datos()
	_, huellaEfecto, errEfecto := datos.ProyeccionEfecto.Datos()
	huellaDecision, errHuella := dominiovec.HuellaSHA256DecisionAutorizacionV3(
		datos.DecisionAutorizacionV3,
	)
	datosAmbitosHMAC, datosHuellasHMAC, coleccionesHMACValidas :=
		datosColeccionesHMACAltaAlineadas(
			datos.AmbitosIdempotenciaHMAC,
			datos.HuellasPeticionHMAC,
		)
	if err != nil || errCorrelacion != nil || errVinculo != nil ||
		errContexto != nil || errDecision != nil ||
		errVentana != nil || errConfirmacion != nil || errEfecto != nil ||
		errHuella != nil ||
		!coleccionesHMACValidas ||
		!concedida ||
		!domain.InstanteUTCCanonico(datos.InstanteConfirmacion) ||
		datos.InstanteConfirmacion.Before(datos.Expediente.CreadoEn) ||
		solicitudV3.VinculoAutenticacionActor.ValidarPara(resultadoContexto) != nil ||
		datos.DecisionAutorizacionV3.ValidarPara(datos.SolicitudAutorizacionV3) != nil ||
		datos.Candidatura.Validar() != nil ||
		datos.Candidatura.OrganizacionRef != datos.Expediente.OrganizacionRef ||
		datos.Candidatura.ActorRef != vinculo.PrincipalID ||
		datos.Candidatura.PerfilRef != vinculo.PerfilActivoRef ||
		datos.Expediente.Actuaciones[0].ActorRef != vinculo.PrincipalID ||
		datos.Candidatura.Referencias.ExpedienteRef != datos.Expediente.Referencia ||
		datos.Candidatura.Referencias.NumeroVisible != datos.Expediente.NumeroVisible ||
		datos.Candidatura.Referencias.ReciboRef != datos.Expediente.Actuaciones[0].ReciboRef ||
		solicitudV3.Accion != AccionCrearSolicitud ||
		solicitudV3.Finalidad != FinalidadCrearSolicitud ||
		solicitudV3.Recurso.ModuloID != ModuloContratacion ||
		solicitudV3.Recurso.Tipo != TipoRecursoExpediente ||
		len(solicitudV3.Recurso.Ambitos) != 3 ||
		len(solicitudV3.Recurso.Atributos) != 5 ||
		!patronHuellaSHA256Alta.MatchString(
			solicitudV3.Recurso.Atributos[AtributoHuellaEfectoAltaSHA256],
		) ||
		solicitudV3.Recurso.Atributos[AtributoHuellaEfectoAltaSHA256] !=
			huellaEfecto ||
		!sellosHMACIguales(
			solicitudV3.Recurso.Referencia,
			datosAmbitosHMAC.Activo.Valor,
		) ||
		!datosColeccionesHMACAltaContienenPar(
			datosAmbitosHMAC,
			datosHuellasHMAC,
			datos.Candidatura.AmbitoIdempotenciaHMAC,
			datos.Candidatura.HuellaPeticionHMAC,
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
		!datos.ConfirmacionRegistroV3.DentroDeVentanaEn(
			datos.InstanteConfirmacion,
		) {
		return OrdenConfirmarAlta{}, ErrOrdenAltaInvalida
	}
	return OrdenConfirmarAlta{datos: &datosOrdenConfirmarAlta{
		expediente:              datos.Expediente.Clonar(),
		solicitudAutorizacionV3: datos.SolicitudAutorizacionV3,
		decisionAutorizacionV3:  datos.DecisionAutorizacionV3,
		confirmacionRegistroV3:  datos.ConfirmacionRegistroV3,
		resultadoContextoV2:     resultadoContexto,
		ambitosIdempotenciaHMAC: datos.AmbitosIdempotenciaHMAC,
		huellasPeticionHMAC:     datos.HuellasPeticionHMAC,
		candidatura:             datos.Candidatura,
		proyeccionEfecto:        datos.ProyeccionEfecto,
		instanteConfirmacion:    datos.InstanteConfirmacion,
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
		ResultadoContextoV2:     o.datos.resultadoContextoV2,
		AmbitosIdempotenciaHMAC: o.datos.ambitosIdempotenciaHMAC,
		HuellasPeticionHMAC:     o.datos.huellasPeticionHMAC,
		Candidatura:             o.datos.candidatura,
		ProyeccionEfecto:        o.datos.proyeccionEfecto,
		InstanteConfirmacion:    o.datos.instanteConfirmacion,
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
		ResultadoContextoV2:     entrada.ResultadoContextoV2,
		AmbitosIdempotenciaHMAC: entrada.AmbitosIdempotenciaHMAC,
		HuellasPeticionHMAC:     entrada.HuellasPeticionHMAC,
		Candidatura:             entrada.Candidatura,
		ProyeccionEfecto:        entrada.ProyeccionEfecto,
		InstanteConfirmacion:    entrada.InstanteConfirmacion,
		CorrelacionV3Ref:        o.datos.correlacionV3Ref,
	}, nil
}

func formatearVersionFlujo(version uint64) string {
	return strconv.FormatUint(version, 10)
}

type ReciboAlta struct {
	ExpedienteRef      string    `json:"expediente_ref"`
	NumeroVisible      string    `json:"numero_visible"`
	Version            uint64    `json:"version"`
	ReciboRef          string    `json:"recibo_ref"`
	AuditoriaRef       string    `json:"auditoria_ref"`
	EventoRef          string    `json:"evento_ref"`
	ConfirmadaEn       time.Time `json:"confirmada_en"`
	ReciboHuellaSHA256 string    `json:"recibo_huella_sha256"`
}

func (r ReciboAlta) ValidarEstructura() error {
	if !domain.ReferenciaOpacaValida(r.ExpedienteRef) ||
		!domain.NumeroExpedienteValido(r.NumeroVisible) || r.Version == 0 ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.EventoRef) ||
		!domain.InstanteUTCCanonico(r.ConfirmadaEn) ||
		!huellaSHA256ReciboAltaValida(r.ReciboHuellaSHA256) {
		return ErrPersistenciaNoDisponible
	}
	esperada, err := CalcularHuellaReciboAlta(r)
	if err != nil || esperada != r.ReciboHuellaSHA256 {
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

// CalcularHuellaReciboAlta reproduce el framing congelado por O2-05. La
// propia huella queda fuera de la preimagen.
func CalcularHuellaReciboAlta(r ReciboAlta) (string, error) {
	if !domain.ReferenciaOpacaValida(r.ExpedienteRef) ||
		!domain.NumeroExpedienteValido(r.NumeroVisible) || r.Version == 0 ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.EventoRef) ||
		!domain.InstanteUTCCanonico(r.ConfirmadaEn) {
		return "", ErrResultadoAltaNoConfiable
	}
	valores := []string{
		r.ExpedienteRef,
		r.NumeroVisible,
		strconv.FormatUint(r.Version, 10),
		r.ReciboRef,
		r.AuditoriaRef,
		r.EventoRef,
		r.ConfirmadaEn.UTC().Format("2006-01-02T15:04:05.000000Z"),
	}
	preimagen := make([]byte, 0, 512)
	for _, valor := range valores {
		preimagen = strconv.AppendInt(preimagen, int64(len([]byte(valor))), 10)
		preimagen = append(preimagen, ':')
		preimagen = append(preimagen, valor...)
		preimagen = append(preimagen, '\n')
	}
	suma := sha256.Sum256(preimagen)
	return hex.EncodeToString(suma[:]), nil
}

func huellaSHA256ReciboAltaValida(valor string) bool {
	if !patronHuellaSHA256Alta.MatchString(valor) ||
		valor == strings.Repeat("0", sha256.Size*2) {
		return false
	}
	return true
}

type DatosMaterialConfirmacionAlta struct {
	CapacidadVECAD3       puertosvec.ExportadorCapacidadAtestacionAutorizacionV3
	PayloadVECAD3         []byte
	SobreCOSESign1        []byte
	EvidenciaVerificacion []byte
	RaizPublicaSPKI       []byte
}

// MaterialConfirmacionAlta mantiene opaco el material que O2-05 vuelve a
// verificar. Su tipo no concede autoridad.
type MaterialConfirmacionAlta struct {
	datos *DatosMaterialConfirmacionAlta
}

func NuevoMaterialConfirmacionAlta(
	datos DatosMaterialConfirmacionAlta,
) (MaterialConfirmacionAlta, error) {
	if dependenciaMaterialNula(datos.CapacidadVECAD3) ||
		len(datos.PayloadVECAD3) == 0 || len(datos.PayloadVECAD3) > 1024*1024 ||
		len(datos.SobreCOSESign1) == 0 ||
		len(datos.SobreCOSESign1) > 1024*1024 ||
		len(datos.EvidenciaVerificacion) == 0 ||
		len(datos.EvidenciaVerificacion) > 256*1024 ||
		len(datos.RaizPublicaSPKI) != 44 {
		return MaterialConfirmacionAlta{}, ErrMaterialConfirmacionAltaInvalido
	}
	copia := datos
	copia.PayloadVECAD3 = append([]byte(nil), datos.PayloadVECAD3...)
	copia.SobreCOSESign1 = append([]byte(nil), datos.SobreCOSESign1...)
	copia.EvidenciaVerificacion = append(
		[]byte(nil), datos.EvidenciaVerificacion...,
	)
	copia.RaizPublicaSPKI = append([]byte(nil), datos.RaizPublicaSPKI...)
	return MaterialConfirmacionAlta{datos: &copia}, nil
}

func (m MaterialConfirmacionAlta) Datos() (DatosMaterialConfirmacionAlta, error) {
	if m.datos == nil {
		return DatosMaterialConfirmacionAlta{}, ErrMaterialConfirmacionAltaInvalido
	}
	return DatosMaterialConfirmacionAlta{
		CapacidadVECAD3: m.datos.CapacidadVECAD3,
		PayloadVECAD3:   append([]byte(nil), m.datos.PayloadVECAD3...),
		SobreCOSESign1:  append([]byte(nil), m.datos.SobreCOSESign1...),
		EvidenciaVerificacion: append(
			[]byte(nil), m.datos.EvidenciaVerificacion...,
		),
		RaizPublicaSPKI: append([]byte(nil), m.datos.RaizPublicaSPKI...),
	}, nil
}

type ProveedorMaterialConfirmacionAlta interface {
	ObtenerMaterialConfirmacionAlta(
		context.Context,
		OrdenConfirmarAlta,
	) (MaterialConfirmacionAlta, error)
}

func dependenciaMaterialNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

// TransaccionAltas debe cotejar y consumir la autorización, confirmar la
// reserva, el expediente, la auditoría y el outbox en un único COMMIT.
type TransaccionAltas interface {
	ConfirmarAlta(context.Context, OrdenConfirmarAlta) (ReciboAlta, error)
}
