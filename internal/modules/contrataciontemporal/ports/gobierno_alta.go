package ports

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const redaccionGobiernoAlta = "[GOBIERNO-ALTA-REDACTADO]"

var (
	ErrGobiernoAltaNoDisponible = errors.New(
		"contratacion temporal: gobierno de alta no disponible",
	)
	ErrSerializacionGobiernoAltaProhibida = errors.New(
		"contratacion temporal: serializacion de gobierno de alta prohibida",
	)
)

// DatosSolicitudGobiernoAlta contiene solo hechos aportados al gobierno por
// el servidor. Flujo, fase, unidad, accion y motivo no forman parte de esta
// entrada y solo pueden proceder de AutoridadGobiernoAlta.
type DatosSolicitudGobiernoAlta struct {
	OrganizacionRef  string
	Solicitud        domain.SolicitudCentro
	InstanteGobierno time.Time
}

// SolicitudGobiernoAlta liga una evaluacion nominal a la organizacion y a la
// solicitud completa del centro sin compartir su coleccion de documentos.
type SolicitudGobiernoAlta struct {
	datos *DatosSolicitudGobiernoAlta
}

func NuevaSolicitudGobiernoAlta(
	organizacionRef string,
	solicitud domain.SolicitudCentro,
	instanteGobierno time.Time,
) (SolicitudGobiernoAlta, error) {
	datos := DatosSolicitudGobiernoAlta{
		OrganizacionRef:  organizacionRef,
		Solicitud:        solicitud,
		InstanteGobierno: instanteGobierno,
	}
	copia, err := clonarDatosSolicitudGobiernoAlta(datos)
	if err != nil {
		return SolicitudGobiernoAlta{}, ErrGobiernoAltaNoDisponible
	}
	return SolicitudGobiernoAlta{datos: &copia}, nil
}

// Datos devuelve una copia defensiva para que una autoridad externa pueda
// consultar la entrada sin adquirir propiedad sobre ella.
func (s SolicitudGobiernoAlta) Datos() (DatosSolicitudGobiernoAlta, error) {
	if s.datos == nil {
		return DatosSolicitudGobiernoAlta{}, ErrGobiernoAltaNoDisponible
	}
	return clonarDatosSolicitudGobiernoAlta(*s.datos)
}

func clonarDatosSolicitudGobiernoAlta(
	datos DatosSolicitudGobiernoAlta,
) (DatosSolicitudGobiernoAlta, error) {
	solicitud, err := datos.Solicitud.Clonar()
	if err != nil || !domain.ReferenciaOpacaValida(datos.OrganizacionRef) ||
		!domain.InstanteUTCCanonico(datos.InstanteGobierno) {
		return DatosSolicitudGobiernoAlta{}, ErrGobiernoAltaNoDisponible
	}
	datos.Solicitud = solicitud
	return datos, nil
}

// PublicacionGobiernoAlta es un DTO de interoperabilidad, no una autoridad.
// Solo ResolverGobiernoAltaSeguro puede cotejarlo y sellar una instantanea.
type PublicacionGobiernoAlta struct {
	OrganizacionRef    string
	Solicitud          domain.SolicitudCentro
	Configuracion      ConfiguracionAltaFlujo
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
	PublicadaEn        time.Time
	VigenteDesde       time.Time
	VigenteHasta       time.Time
}

// AutoridadGobiernoAlta resuelve conjuntamente todas las decisiones del alta.
// Una resolucion segura invoca exactamente una vez una sola implementacion.
type AutoridadGobiernoAlta interface {
	ResolverGobiernoAlta(
		context.Context,
		SolicitudGobiernoAlta,
	) (PublicacionGobiernoAlta, error)
}

// DatosInstantaneaGobiernoAlta es una copia verificable de la instantanea. El
// bloqueo embebido impide convertir incluso esta proyeccion en un DTO.
type DatosInstantaneaGobiernoAlta struct {
	bloqueoSerializacionGobiernoAlta
	OrganizacionRef    string
	Solicitud          domain.SolicitudCentro
	Configuracion      ConfiguracionAltaFlujo
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
	InstanteGobierno   time.Time
	PublicadaEn        time.Time
	VigenteDesde       time.Time
	VigenteHasta       time.Time
}

// InstantaneaGobiernoAlta conserva estado privado y solo puede nacer de la
// frontera segura. El valor cero no representa ninguna decision.
type InstantaneaGobiernoAlta struct {
	bloqueoSerializacionGobiernoAlta
	datos *DatosInstantaneaGobiernoAlta
}

// ResolverGobiernoAltaSeguro es la unica conversion de una publicacion a una
// instantanea. Valida antes de llamar, no reintenta, sanea errores y coteja la
// respuesta completa contra una copia congelada de la solicitud nominal.
func ResolverGobiernoAltaSeguro(
	ctx context.Context,
	autoridad AutoridadGobiernoAlta,
	solicitud SolicitudGobiernoAlta,
) (InstantaneaGobiernoAlta, error) {
	if dependenciaGobiernoAltaNula(ctx) ||
		dependenciaGobiernoAltaNula(autoridad) {
		return InstantaneaGobiernoAlta{}, errorGobiernoAlta()
	}
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		return InstantaneaGobiernoAlta{}, errorGobiernoAlta(err)
	}
	if err := ctx.Err(); err != nil {
		return InstantaneaGobiernoAlta{}, errorGobiernoAlta(err)
	}
	copiaParaAutoridad, err := nuevaSolicitudGobiernoAltaDesdeDatos(datosSolicitud)
	if err != nil {
		return InstantaneaGobiernoAlta{}, errorGobiernoAlta(err)
	}
	publicacion, errAutoridad := autoridad.ResolverGobiernoAlta(
		ctx,
		copiaParaAutoridad,
	)
	if errContexto := ctx.Err(); errContexto != nil {
		return InstantaneaGobiernoAlta{}, errorGobiernoAlta(errContexto)
	}
	if errAutoridad != nil {
		return InstantaneaGobiernoAlta{}, errorGobiernoAlta(errAutoridad)
	}
	instantanea, err := sellarInstantaneaGobiernoAlta(datosSolicitud, publicacion)
	if err != nil {
		return InstantaneaGobiernoAlta{}, errorGobiernoAlta(err)
	}
	return instantanea, nil
}

func nuevaSolicitudGobiernoAltaDesdeDatos(
	datos DatosSolicitudGobiernoAlta,
) (SolicitudGobiernoAlta, error) {
	copia, err := clonarDatosSolicitudGobiernoAlta(datos)
	if err != nil {
		return SolicitudGobiernoAlta{}, err
	}
	return SolicitudGobiernoAlta{datos: &copia}, nil
}

func sellarInstantaneaGobiernoAlta(
	solicitud DatosSolicitudGobiernoAlta,
	publicacion PublicacionGobiernoAlta,
) (InstantaneaGobiernoAlta, error) {
	if !publicacionGobiernoAltaValidaPara(publicacion, solicitud) {
		return InstantaneaGobiernoAlta{}, ErrGobiernoAltaNoDisponible
	}
	solicitudPublicada, err := publicacion.Solicitud.Clonar()
	if err != nil {
		return InstantaneaGobiernoAlta{}, ErrGobiernoAltaNoDisponible
	}
	datos := DatosInstantaneaGobiernoAlta{
		OrganizacionRef:    publicacion.OrganizacionRef,
		Solicitud:          solicitudPublicada,
		Configuracion:      publicacion.Configuracion,
		MotivoAutorizacion: publicacion.MotivoAutorizacion,
		InstanteGobierno:   solicitud.InstanteGobierno,
		PublicadaEn:        publicacion.PublicadaEn,
		VigenteDesde:       publicacion.VigenteDesde,
		VigenteHasta:       publicacion.VigenteHasta,
	}
	if !datosInstantaneaGobiernoAltaValidos(datos) {
		return InstantaneaGobiernoAlta{}, ErrGobiernoAltaNoDisponible
	}
	return InstantaneaGobiernoAlta{datos: &datos}, nil
}

func publicacionGobiernoAltaValidaPara(
	publicacion PublicacionGobiernoAlta,
	solicitud DatosSolicitudGobiernoAlta,
) bool {
	return publicacion.OrganizacionRef == solicitud.OrganizacionRef &&
		solicitudesCentroGobiernoAltaIguales(
			publicacion.Solicitud,
			solicitud.Solicitud,
		) &&
		publicacion.Configuracion.Validar() == nil &&
		publicacion.MotivoAutorizacion.Validar() == nil &&
		publicacion.MotivoAutorizacion.CatalogoHuellaSHA256 !=
			strings.Repeat("0", 64) &&
		uint64(publicacion.MotivoAutorizacion.CatalogoVersion) <=
			MaximoEnteroSeguroOperacionAnalisis &&
		ventanaGobiernoAltaValida(
			publicacion.PublicadaEn,
			publicacion.VigenteDesde,
			publicacion.VigenteHasta,
			solicitud.InstanteGobierno,
		)
}

func ventanaGobiernoAltaValida(
	publicadaEn, vigenteDesde, vigenteHasta, instante time.Time,
) bool {
	return domain.InstanteUTCCanonico(publicadaEn) &&
		domain.InstanteUTCCanonico(vigenteDesde) &&
		domain.InstanteUTCCanonico(vigenteHasta) &&
		!publicadaEn.After(vigenteDesde) &&
		vigenteHasta.After(vigenteDesde) &&
		!instante.Before(vigenteDesde) && instante.Before(vigenteHasta)
}

func datosInstantaneaGobiernoAltaValidos(
	datos DatosInstantaneaGobiernoAlta,
) bool {
	solicitud := DatosSolicitudGobiernoAlta{
		OrganizacionRef:  datos.OrganizacionRef,
		Solicitud:        datos.Solicitud,
		InstanteGobierno: datos.InstanteGobierno,
	}
	_, err := clonarDatosSolicitudGobiernoAlta(solicitud)
	return err == nil && publicacionGobiernoAltaValidaPara(
		PublicacionGobiernoAlta{
			OrganizacionRef:    datos.OrganizacionRef,
			Solicitud:          datos.Solicitud,
			Configuracion:      datos.Configuracion,
			MotivoAutorizacion: datos.MotivoAutorizacion,
			PublicadaEn:        datos.PublicadaEn,
			VigenteDesde:       datos.VigenteDesde,
			VigenteHasta:       datos.VigenteHasta,
		},
		solicitud,
	)
}

// Datos devuelve un clon y reacredita el estado privado antes de exponerlo.
func (i InstantaneaGobiernoAlta) Datos() (
	DatosInstantaneaGobiernoAlta,
	error,
) {
	if i.datos == nil || !datosInstantaneaGobiernoAltaValidos(*i.datos) {
		return DatosInstantaneaGobiernoAlta{}, ErrGobiernoAltaNoDisponible
	}
	copia := *i.datos
	solicitud, err := i.datos.Solicitud.Clonar()
	if err != nil {
		return DatosInstantaneaGobiernoAlta{}, ErrGobiernoAltaNoDisponible
	}
	copia.Solicitud = solicitud
	return copia, nil
}

// ValidarPara exige la misma solicitud nominal y una consulta dentro de la
// ventana gobernada: inicio inclusivo y fin exclusivo.
func (i InstantaneaGobiernoAlta) ValidarPara(
	solicitud SolicitudGobiernoAlta,
	instante time.Time,
) error {
	datos, err := i.Datos()
	datosSolicitud, errSolicitud := solicitud.Datos()
	if err != nil || errSolicitud != nil ||
		!domain.InstanteUTCCanonico(instante) ||
		datos.OrganizacionRef != datosSolicitud.OrganizacionRef ||
		!datos.InstanteGobierno.Equal(datosSolicitud.InstanteGobierno) ||
		!solicitudesCentroGobiernoAltaIguales(
			datos.Solicitud,
			datosSolicitud.Solicitud,
		) ||
		!ventanaGobiernoAltaValida(
			datos.PublicadaEn,
			datos.VigenteDesde,
			datos.VigenteHasta,
			instante,
		) {
		return ErrGobiernoAltaNoDisponible
	}
	return nil
}

func solicitudesCentroGobiernoAltaIguales(
	primera, segunda domain.SolicitudCentro,
) bool {
	if primera.CentroRef != segunda.CentroRef ||
		primera.ContactoRef != segunda.ContactoRef ||
		primera.CategoriaRef != segunda.CategoriaRef ||
		primera.GrupoSubgrupo != segunda.GrupoSubgrupo ||
		primera.MotivoClave != segunda.MotivoClave ||
		primera.Detalle != segunda.Detalle ||
		!primera.Periodo.Inicio.Equal(segunda.Periodo.Inicio) ||
		!primera.Periodo.Fin.Equal(segunda.Periodo.Fin) ||
		primera.RC.Existe != segunda.RC.Existe ||
		primera.RC.Numero != segunda.RC.Numero ||
		!primera.RC.Fecha.Equal(segunda.RC.Fecha) ||
		primera.RC.Importe != segunda.RC.Importe ||
		primera.RC.DocumentoRef != segunda.RC.DocumentoRef ||
		primera.Observaciones != segunda.Observaciones ||
		len(primera.DocumentosAdjuntos) != len(segunda.DocumentosAdjuntos) {
		return false
	}
	for indice := range primera.DocumentosAdjuntos {
		if primera.DocumentosAdjuntos[indice] != segunda.DocumentosAdjuntos[indice] {
			return false
		}
	}
	return true
}

func dependenciaGobiernoAltaNula(dependencia any) bool {
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

func errorGobiernoAlta(causas ...error) error {
	publicos := []error{ErrGobiernoAltaNoDisponible}
	for _, causa := range causas {
		switch {
		case errors.Is(causa, context.Canceled):
			publicos = append(publicos, context.Canceled)
		case errors.Is(causa, context.DeadlineExceeded):
			publicos = append(publicos, context.DeadlineExceeded)
		}
	}
	return errors.Join(publicos...)
}

type bloqueoSerializacionGobiernoAlta struct{}

func (bloqueoSerializacionGobiernoAlta) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionGobiernoAltaProhibida
}
func (*bloqueoSerializacionGobiernoAlta) UnmarshalJSON([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (bloqueoSerializacionGobiernoAlta) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (*bloqueoSerializacionGobiernoAlta) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (bloqueoSerializacionGobiernoAlta) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionGobiernoAltaProhibida
}
func (*bloqueoSerializacionGobiernoAlta) UnmarshalText([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (bloqueoSerializacionGobiernoAlta) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionGobiernoAltaProhibida
}
func (*bloqueoSerializacionGobiernoAlta) UnmarshalBinary([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (bloqueoSerializacionGobiernoAlta) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionGobiernoAltaProhibida
}
func (*bloqueoSerializacionGobiernoAlta) GobDecode([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (bloqueoSerializacionGobiernoAlta) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionGobiernoAltaProhibida
}
func (*bloqueoSerializacionGobiernoAlta) UnmarshalCBOR([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (bloqueoSerializacionGobiernoAlta) MarshalYAML() (any, error) {
	return nil, ErrSerializacionGobiernoAltaProhibida
}
func (*bloqueoSerializacionGobiernoAlta) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (bloqueoSerializacionGobiernoAlta) String() string { return redaccionGobiernoAlta }
func (b bloqueoSerializacionGobiernoAlta) GoString() string {
	return b.String()
}
func (b bloqueoSerializacionGobiernoAlta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionGobiernoAlta) LogValue() slog.Value {
	return slog.StringValue(b.String())
}

// Los decodificadores directos evitan que un receptor, incluso nil tipado,
// pueda reconstruir estado opaco mediante la promocion de metodos.
func (*DatosInstantaneaGobiernoAlta) UnmarshalJSON([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (*DatosInstantaneaGobiernoAlta) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (*DatosInstantaneaGobiernoAlta) UnmarshalText([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (*DatosInstantaneaGobiernoAlta) UnmarshalBinary([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (*DatosInstantaneaGobiernoAlta) GobDecode([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (*InstantaneaGobiernoAlta) UnmarshalJSON([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (*InstantaneaGobiernoAlta) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (*InstantaneaGobiernoAlta) UnmarshalText([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (*InstantaneaGobiernoAlta) UnmarshalBinary([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
func (*InstantaneaGobiernoAlta) GobDecode([]byte) error {
	return ErrSerializacionGobiernoAltaProhibida
}
