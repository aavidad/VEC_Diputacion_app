package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrDependenciaBaremacionRequerida = errors.New("bolsa: dependencia de baremacion requerida")
	ErrOrdenBaremacionInvalida        = errors.New("bolsa: orden de baremacion invalida")
	ErrResultadoBaremacionNoConfiable = errors.New("bolsa: resultado de baremacion no confiable")
	ErrFirmaBaremacionNoCompletada    = errors.New("bolsa: firma de baremacion no completada")
)

const (
	duracionReservaBaremacionPredeterminada = 15 * time.Second
	duracionFirmaBaremacionPredeterminada   = 10 * time.Minute
	clasificacionBaremacionPredeterminada   = "datos_personales_alta"
	limiteDocumentoFirmadoBaremacion        = int64(64 << 20)
	hmacBaremacionPendiente                 = "hmac-sha256:pendiente_1:0000000000000000000000000000000000000000000000000000000000000000"
)

// SesionAutenticadaBaremacion es una capacidad opaca emitida por la frontera
// autoritativa de identidad. Liga la persona y el perfil resueltos con la
// sesion revalidada mediante VinculoAutenticacionActorV1; no contiene hechos
// de autenticacion copiables desde una orden o un DTO.
type SesionAutenticadaBaremacion struct {
	contextoActor dominiovec.ContextoActor
	vinculo       dominiovec.VinculoAutenticacionActorV1
}

func NuevaSesionAutenticadaBaremacion(
	contextoActor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
) (SesionAutenticadaBaremacion, error) {
	clon, err := contextoActor.Clonar()
	if err != nil || vinculo.ValidarPara(clon) != nil {
		return SesionAutenticadaBaremacion{}, dominiovec.ErrAutorizacionDenegada
	}
	return SesionAutenticadaBaremacion{contextoActor: clon, vinculo: vinculo}, nil
}

func (SesionAutenticadaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (*SesionAutenticadaBaremacion) UnmarshalJSON([]byte) error {
	return ErrOrdenBaremacionInvalida
}
func (SesionAutenticadaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (SesionAutenticadaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (SesionAutenticadaBaremacion) String() string     { return "[SESION-BAREMACION-OPACA]" }
func (s SesionAutenticadaBaremacion) GoString() string { return s.String() }
func (s SesionAutenticadaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SesionAutenticadaBaremacion) LogValue() slog.Value { return slog.StringValue(s.String()) }

func (s SesionAutenticadaBaremacion) capacidades() (
	dominiovec.ContextoActor,
	dominiovec.VinculoAutenticacionActorV1,
	error,
) {
	contextoActor, err := s.contextoActor.Clonar()
	if err != nil || s.vinculo.ValidarPara(contextoActor) != nil {
		return dominiovec.ContextoActor{}, dominiovec.VinculoAutenticacionActorV1{}, dominiovec.ErrAutorizacionDenegada
	}
	return contextoActor, s.vinculo, nil
}

// FuenteSesionAutenticadaBaremacion debe resolver la sesion a partir del
// contexto confiable del adaptador de entrada, no de cabeceras aportadas por
// el cliente. Devuelve todas las coincidencias para que la aplicacion pueda
// denegar tanto la ausencia como la ambiguedad sin elegir la primera.
type FuenteSesionAutenticadaBaremacion interface {
	BuscarSesionesAutenticadasBaremacion(context.Context) ([]SesionAutenticadaBaremacion, error)
}

// OpcionesServicioBaremacion fija exclusivamente limites tecnicos y el
// conector de almacenamiento admitido por el despliegue.
type OpcionesServicioBaremacion struct {
	DuracionReserva          time.Duration
	DuracionFirma            time.Duration
	ClasificacionDocumental  string
	ConectorAlmacenPermitido string
	PoliticaRetencionRef     string
	DuracionRetencion        time.Duration
}

// ServicioBaremacion coordina el corte probatorio de una revision tecnica.
// Cada paso obtiene una decision de autorizacion nueva, exacta y no
// serializable; ningun contexto de autorizacion llega en las ordenes.
type ServicioBaremacion struct {
	repositorio        puertosbolsa.RepositorioBaremaciones
	fuenteDatos        puertosbolsa.FuenteDatosBaremacion
	calculador         puertosbolsa.CalculadorOficialBaremacion
	catalogoFirma      puertosbolsa.CatalogoPoliticasFirmaBaremacion
	codificador        puertosbolsa.CodificadorCanonicoDecision
	almacen            puertosbolsa.AlmacenDocumentosFirmables
	firmador           puertosbolsa.FirmadorInteractivo
	recuperadorBinario puertosbolsa.RecuperadorBinarioFirmado
	validadorFirma     puertosbolsa.ValidadorFirmaServidor
	selladorTiempo     puertosbolsa.SelladorTiempoFirma
	aumentadorFirma    puertosbolsa.AumentadorFirmaLongeva
	selladorSolicitud  puertosbolsa.SelladorSolicitudBaremacion
	seudonimizador     puertosvec.SeudonimizadorSujetoAlmacen
	generador          puertosbolsa.GeneradorReferenciasOpacasBaremacion
	autorizador        puertosvec.Autorizador
	sesiones           FuenteSesionAutenticadaBaremacion
	reloj              puertosbolsa.Reloj
	duracionReserva    time.Duration
	duracionFirma      time.Duration
	clasificacion      string
	conectorAlmacen    string
	politicaRetencion  string
	duracionRetencion  time.Duration
}

// NuevoServicioBaremacion exige todos los conectores de seguridad al arrancar;
// una composicion parcial no crea un servicio degradado.
func NuevoServicioBaremacion(
	repositorio puertosbolsa.RepositorioBaremaciones,
	fuenteDatos puertosbolsa.FuenteDatosBaremacion,
	calculador puertosbolsa.CalculadorOficialBaremacion,
	catalogoFirma puertosbolsa.CatalogoPoliticasFirmaBaremacion,
	codificador puertosbolsa.CodificadorCanonicoDecision,
	almacen puertosbolsa.AlmacenDocumentosFirmables,
	firmador puertosbolsa.FirmadorInteractivo,
	recuperadorBinario puertosbolsa.RecuperadorBinarioFirmado,
	validadorFirma puertosbolsa.ValidadorFirmaServidor,
	selladorTiempo puertosbolsa.SelladorTiempoFirma,
	aumentadorFirma puertosbolsa.AumentadorFirmaLongeva,
	selladorSolicitud puertosbolsa.SelladorSolicitudBaremacion,
	seudonimizador puertosvec.SeudonimizadorSujetoAlmacen,
	generador puertosbolsa.GeneradorReferenciasOpacasBaremacion,
	autorizador puertosvec.Autorizador,
	sesiones FuenteSesionAutenticadaBaremacion,
	reloj puertosbolsa.Reloj,
	opciones OpcionesServicioBaremacion,
) (*ServicioBaremacion, error) {
	if dependenciaBaremacionNula(repositorio) || dependenciaBaremacionNula(fuenteDatos) ||
		dependenciaBaremacionNula(calculador) || dependenciaBaremacionNula(catalogoFirma) ||
		dependenciaBaremacionNula(codificador) || dependenciaBaremacionNula(almacen) ||
		dependenciaBaremacionNula(firmador) || dependenciaBaremacionNula(recuperadorBinario) ||
		dependenciaBaremacionNula(validadorFirma) || dependenciaBaremacionNula(selladorTiempo) ||
		dependenciaBaremacionNula(aumentadorFirma) || dependenciaBaremacionNula(selladorSolicitud) ||
		dependenciaBaremacionNula(seudonimizador) || dependenciaBaremacionNula(generador) ||
		dependenciaBaremacionNula(autorizador) || dependenciaBaremacionNula(sesiones) ||
		dependenciaBaremacionNula(reloj) {
		return nil, ErrDependenciaBaremacionRequerida
	}
	duracionReserva := opciones.DuracionReserva
	if duracionReserva == 0 {
		duracionReserva = duracionReservaBaremacionPredeterminada
	}
	duracionFirma := opciones.DuracionFirma
	if duracionFirma == 0 {
		duracionFirma = duracionFirmaBaremacionPredeterminada
	}
	clasificacion := strings.TrimSpace(opciones.ClasificacionDocumental)
	if clasificacion == "" {
		clasificacion = clasificacionBaremacionPredeterminada
	}
	conector := strings.TrimSpace(opciones.ConectorAlmacenPermitido)
	politicaRetencion := strings.TrimSpace(opciones.PoliticaRetencionRef)
	if duracionReserva < time.Second || duracionReserva > puertosbolsa.VentanaMaximaUsoAutorizacionBaremacion ||
		duracionFirma < time.Second || duracionFirma > puertosbolsa.VentanaMaximaSesionFirmaInteractiva ||
		clasificacion != opciones.ClasificacionDocumental && opciones.ClasificacionDocumental != "" ||
		conector == "" || conector != opciones.ConectorAlmacenPermitido ||
		politicaRetencion == "" || politicaRetencion != opciones.PoliticaRetencionRef ||
		opciones.DuracionRetencion < 24*time.Hour {
		return nil, ErrDependenciaBaremacionRequerida
	}
	return &ServicioBaremacion{
		repositorio: repositorio, fuenteDatos: fuenteDatos, calculador: calculador,
		catalogoFirma: catalogoFirma, codificador: codificador, almacen: almacen,
		firmador: firmador, recuperadorBinario: recuperadorBinario, validadorFirma: validadorFirma, selladorTiempo: selladorTiempo,
		aumentadorFirma: aumentadorFirma, selladorSolicitud: selladorSolicitud,
		seudonimizador: seudonimizador, generador: generador, autorizador: autorizador,
		sesiones: sesiones, reloj: reloj, duracionReserva: duracionReserva,
		duracionFirma: duracionFirma, clasificacion: clasificacion, conectorAlmacen: conector,
		politicaRetencion: politicaRetencion, duracionRetencion: opciones.DuracionRetencion,
	}, nil
}
