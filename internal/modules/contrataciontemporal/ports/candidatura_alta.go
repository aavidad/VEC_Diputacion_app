package ports

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	dominioAmbitoCandidaturaAlta = "vec.contratacion-temporal.ambito-idempotencia"
	dominioHuellaCandidaturaAlta = "vec.contratacion-temporal.huella-peticion"
	redaccionCandidaturaAlta     = "[CANDIDATURA-ALTA-REDACTADA]"
)

var ErrSerializacionCandidaturaAltaProhibida = errors.New(
	"contratacion temporal: serializacion de candidatura de alta prohibida",
)

// bloqueoSerializacionCandidaturaAlta impide usar una candidatura técnica o
// sus entradas nominales como DTO, recibo o material reconstruible.
type bloqueoSerializacionCandidaturaAlta struct{}

func (bloqueoSerializacionCandidaturaAlta) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCandidaturaAltaProhibida
}
func (*bloqueoSerializacionCandidaturaAlta) UnmarshalJSON([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (bloqueoSerializacionCandidaturaAlta) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*bloqueoSerializacionCandidaturaAlta) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (bloqueoSerializacionCandidaturaAlta) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionCandidaturaAltaProhibida
}
func (*bloqueoSerializacionCandidaturaAlta) UnmarshalText([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (bloqueoSerializacionCandidaturaAlta) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionCandidaturaAltaProhibida
}
func (*bloqueoSerializacionCandidaturaAlta) UnmarshalBinary([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (bloqueoSerializacionCandidaturaAlta) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionCandidaturaAltaProhibida
}
func (*bloqueoSerializacionCandidaturaAlta) GobDecode([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (bloqueoSerializacionCandidaturaAlta) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionCandidaturaAltaProhibida
}
func (*bloqueoSerializacionCandidaturaAlta) UnmarshalCBOR([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (bloqueoSerializacionCandidaturaAlta) MarshalYAML() (any, error) {
	return nil, ErrSerializacionCandidaturaAltaProhibida
}
func (*bloqueoSerializacionCandidaturaAlta) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (bloqueoSerializacionCandidaturaAlta) String() string {
	return redaccionCandidaturaAlta
}
func (b bloqueoSerializacionCandidaturaAlta) GoString() string {
	return b.String()
}
func (b bloqueoSerializacionCandidaturaAlta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionCandidaturaAlta) LogValue() slog.Value {
	return slog.StringValue(b.String())
}

// DatosCandidaturaAlta contiene solo coordenadas técnicas no autoritativas.
// No representa reserva administrativa, expediente, autorización ni recibo.
type DatosCandidaturaAlta struct {
	bloqueoSerializacionCandidaturaAlta
	ReservaRef             string
	Referencias            ReferenciasAlta
	AmbitoIdempotenciaHMAC string
	HuellaPeticionHMAC     string
	OrganizacionRef        string
	ActorRef               string
	PerfilRef              string
	InstanteEfecto         time.Time
}

// CandidaturaAlta estabiliza las coordenadas del efecto antes de cualquier
// autorización o transacción. Es inmutable, opaca y recuperable repetidamente.
type CandidaturaAlta struct {
	bloqueoSerializacionCandidaturaAlta
	datos *DatosCandidaturaAlta
}

func NuevaCandidaturaAlta(datos DatosCandidaturaAlta) (CandidaturaAlta, error) {
	if !datosCandidaturaAltaValidos(datos) {
		return CandidaturaAlta{}, ErrPreparacionAltaInvalida
	}
	copia := datos
	return CandidaturaAlta{datos: &copia}, nil
}

// Datos entrega una copia defensiva y vuelve a acreditar todas las
// coordenadas para que un valor cero o alterado dentro del paquete falle.
func (c CandidaturaAlta) Datos() (DatosCandidaturaAlta, error) {
	if c.datos == nil || !datosCandidaturaAltaValidos(*c.datos) {
		return DatosCandidaturaAlta{}, ErrPreparacionAltaInvalida
	}
	return *c.datos, nil
}

func datosCandidaturaAltaValidos(datos DatosCandidaturaAlta) bool {
	dominioAmbito, generacionAmbito, ambitoValido := descomponerSelloHMAC(
		datos.AmbitoIdempotenciaHMAC,
	)
	dominioHuella, generacionHuella, huellaValida := descomponerSelloHMAC(
		datos.HuellaPeticionHMAC,
	)
	return domain.ReferenciaOpacaValida(datos.ReservaRef) &&
		datos.Referencias.Validar() == nil &&
		ambitoValido && huellaValida &&
		dominioAmbito == dominioAmbitoCandidaturaAlta &&
		dominioHuella == dominioHuellaCandidaturaAlta &&
		generacionAmbito == generacionHuella &&
		domain.ReferenciaOpacaValida(datos.OrganizacionRef) &&
		domain.ReferenciaOpacaValida(datos.ActorRef) &&
		domain.ReferenciaOpacaValida(datos.PerfilRef) &&
		domain.InstanteUTCCanonico(datos.InstanteEfecto)
}

// DatosSolicitudResolverCandidaturaAlta transporta las matrices gobernadas y
// una propuesta nueva. La propuesta debe usar exclusivamente el par activo.
type DatosSolicitudResolverCandidaturaAlta struct {
	bloqueoSerializacionCandidaturaAlta
	AmbitosIdempotenciaHMAC ColeccionSellosHMAC
	HuellasPeticionHMAC     ColeccionSellosHMAC
	OrganizacionRef         string
	ActorRef                string
	PerfilRef               string
	Propuesta               CandidaturaAlta
}

// SolicitudResolverCandidaturaAlta es la entrada opaca y repetible del puerto
// de estabilización. No concede autoridad de efecto.
type SolicitudResolverCandidaturaAlta struct {
	bloqueoSerializacionCandidaturaAlta
	datos *DatosSolicitudResolverCandidaturaAlta
}

func NuevaSolicitudResolverCandidaturaAlta(
	datos DatosSolicitudResolverCandidaturaAlta,
) (SolicitudResolverCandidaturaAlta, error) {
	copia, err := clonarDatosSolicitudResolverCandidaturaAlta(datos)
	if err != nil {
		return SolicitudResolverCandidaturaAlta{}, ErrPreparacionAltaInvalida
	}
	return SolicitudResolverCandidaturaAlta{datos: &copia}, nil
}

// Datos devuelve clones independientes de ambas colecciones y de la
// candidatura propuesta.
func (s SolicitudResolverCandidaturaAlta) Datos() (
	DatosSolicitudResolverCandidaturaAlta,
	error,
) {
	if s.datos == nil {
		return DatosSolicitudResolverCandidaturaAlta{},
			ErrPreparacionAltaInvalida
	}
	return clonarDatosSolicitudResolverCandidaturaAlta(*s.datos)
}

// ValidarResultado acepta el par activo o un par retenido alineado. Las
// referencias y el instante pueden pertenecer a la candidatura histórica:
// solo la identidad estable y la generación HMAC deben conservarse.
func (s SolicitudResolverCandidaturaAlta) ValidarResultado(
	resultado CandidaturaAlta,
) error {
	datosSolicitud, err := s.Datos()
	if err != nil {
		return ErrPreparacionAltaInvalida
	}
	datosResultado, err := resultado.Datos()
	if err != nil ||
		datosResultado.OrganizacionRef != datosSolicitud.OrganizacionRef ||
		datosResultado.ActorRef != datosSolicitud.ActorRef ||
		datosResultado.PerfilRef != datosSolicitud.PerfilRef ||
		!ColeccionesHMACAltaContienenPar(
			datosSolicitud.AmbitosIdempotenciaHMAC,
			datosSolicitud.HuellasPeticionHMAC,
			datosResultado.AmbitoIdempotenciaHMAC,
			datosResultado.HuellaPeticionHMAC,
		) {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

func clonarDatosSolicitudResolverCandidaturaAlta(
	datos DatosSolicitudResolverCandidaturaAlta,
) (DatosSolicitudResolverCandidaturaAlta, error) {
	datosAmbitos, datosHuellas, alineadas :=
		datosColeccionesHMACAltaAlineadas(
			datos.AmbitosIdempotenciaHMAC,
			datos.HuellasPeticionHMAC,
		)
	datosPropuesta, err := datos.Propuesta.Datos()
	if !alineadas || err != nil ||
		!domain.ReferenciaOpacaValida(datos.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(datos.ActorRef) ||
		!domain.ReferenciaOpacaValida(datos.PerfilRef) ||
		datosPropuesta.OrganizacionRef != datos.OrganizacionRef ||
		datosPropuesta.ActorRef != datos.ActorRef ||
		datosPropuesta.PerfilRef != datos.PerfilRef ||
		!sellosHMACIguales(
			datosPropuesta.AmbitoIdempotenciaHMAC,
			datosAmbitos.Activo.Valor,
		) ||
		!sellosHMACIguales(
			datosPropuesta.HuellaPeticionHMAC,
			datosHuellas.Activo.Valor,
		) {
		return DatosSolicitudResolverCandidaturaAlta{},
			ErrPreparacionAltaInvalida
	}
	ambitos, errAmbitos := reconstruirColeccionSellosHMAC(datosAmbitos)
	huellas, errHuellas := reconstruirColeccionSellosHMAC(datosHuellas)
	propuesta, errPropuesta := NuevaCandidaturaAlta(datosPropuesta)
	if errAmbitos != nil || errHuellas != nil || errPropuesta != nil {
		return DatosSolicitudResolverCandidaturaAlta{},
			ErrPreparacionAltaInvalida
	}
	datos.AmbitosIdempotenciaHMAC = ambitos
	datos.HuellasPeticionHMAC = huellas
	datos.Propuesta = propuesta
	return datos, nil
}

func reconstruirColeccionSellosHMAC(
	datos DatosColeccionSellosHMAC,
) (ColeccionSellosHMAC, error) {
	retenidos := make([]string, 0, len(datos.Retenidos))
	for _, retenido := range datos.Retenidos {
		retenidos = append(retenidos, retenido.Valor)
	}
	return NuevaColeccionSellosHMAC(datos.Activo.Valor, retenidos)
}

// ResolutorCandidaturaAlta estabiliza o recupera coordenadas técnicas. El
// efecto administrativo pertenece a TransaccionAltas y a cortes posteriores.
type ResolutorCandidaturaAlta interface {
	ResolverCandidaturaAlta(
		context.Context,
		SolicitudResolverCandidaturaAlta,
	) (CandidaturaAlta, error)
}

// Los métodos directos de reconstrucción mantienen seguro un receptor nil
// tipado. La emisión y la redacción de valores se heredan del bloqueo.
func (*DatosCandidaturaAlta) UnmarshalJSON([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*DatosCandidaturaAlta) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*DatosCandidaturaAlta) UnmarshalText([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*DatosCandidaturaAlta) UnmarshalBinary([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*DatosCandidaturaAlta) GobDecode([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*DatosCandidaturaAlta) UnmarshalCBOR([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*DatosCandidaturaAlta) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionCandidaturaAltaProhibida
}

func (*CandidaturaAlta) UnmarshalJSON([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*CandidaturaAlta) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*CandidaturaAlta) UnmarshalText([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*CandidaturaAlta) UnmarshalBinary([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*CandidaturaAlta) GobDecode([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*CandidaturaAlta) UnmarshalCBOR([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*CandidaturaAlta) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionCandidaturaAltaProhibida
}

func (*DatosSolicitudResolverCandidaturaAlta) UnmarshalJSON([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*DatosSolicitudResolverCandidaturaAlta) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*DatosSolicitudResolverCandidaturaAlta) UnmarshalText([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*DatosSolicitudResolverCandidaturaAlta) UnmarshalBinary([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*DatosSolicitudResolverCandidaturaAlta) GobDecode([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*DatosSolicitudResolverCandidaturaAlta) UnmarshalCBOR([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*DatosSolicitudResolverCandidaturaAlta) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionCandidaturaAltaProhibida
}

func (*SolicitudResolverCandidaturaAlta) UnmarshalJSON([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*SolicitudResolverCandidaturaAlta) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*SolicitudResolverCandidaturaAlta) UnmarshalText([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*SolicitudResolverCandidaturaAlta) UnmarshalBinary([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*SolicitudResolverCandidaturaAlta) GobDecode([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*SolicitudResolverCandidaturaAlta) UnmarshalCBOR([]byte) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
func (*SolicitudResolverCandidaturaAlta) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionCandidaturaAltaProhibida
}
