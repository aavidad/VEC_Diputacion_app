// Package ports declara los contratos mínimos del módulo Selección con sus
// adaptadores: persistencia, protección de datos personales, autorización
// V3, identificadores y servicios externos. No contiene implementaciones.
package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/shared/baremacion"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Autorización V3 de Selección. AD3-89: la persona sobre su recurso propio
// 'mis-solicitudes:<persona>'. AD3-90: RRHH sobre el listado de una
// convocatoria o la ficha de una solicitud.
const (
	ModuloSeleccion = "seleccion"

	AccionConsultarPropias        = "seleccion.solicitudes_propias.consultar"
	AccionGuardarBorrador         = "seleccion.solicitudes_propias.guardar_borrador"
	AccionPresentar               = "seleccion.solicitudes_propias.presentar"
	AudienciaConsultarPropias     = "vec_seleccion.solicitudes_propias.consultar.v1"
	AudienciaGuardarBorrador      = "vec_seleccion.solicitudes_propias.guardar_borrador.v1"
	AudienciaPresentar            = "vec_seleccion.solicitudes_propias.presentar.v1"
	FinalidadSolicitudesPropias   = "gestion_solicitudes_propias"
	TipoRecursoSolicitudesPropias = "solicitudes_propias_seleccion"
	PrefijoRecursoPropio          = "mis-solicitudes:"

	AccionConsultarSolicitudes    = "seleccion.solicitudes.consultar"
	AccionConsultarDetalle        = "seleccion.solicitudes.consultar_detalle"
	AudienciaConsultarSolicitudes = "vec_seleccion.solicitudes.consultar.v1"
	AudienciaConsultarDetalle     = "vec_seleccion.solicitudes.consultar_detalle.v1"
	FinalidadConsultaSolicitudes  = "consulta_solicitudes_seleccion"
	TipoRecursoSolicitudes        = "solicitudes_seleccion"
	PrefijoRecursoConvocatoria    = "solicitudes-convocatoria:"
	PrefijoRecursoSolicitud       = "solicitud-seleccion:"
)

// AccionesPropias enumera las acciones de AD3-89 con su audiencia.
func AccionesPropias() [][2]string {
	return [][2]string{
		{AccionConsultarPropias, AudienciaConsultarPropias},
		{AccionGuardarBorrador, AudienciaGuardarBorrador},
		{AccionPresentar, AudienciaPresentar},
	}
}

// AccionesRRHH enumera las acciones de AD3-90 con su audiencia.
func AccionesRRHH() [][2]string {
	return [][2]string{
		{AccionConsultarSolicitudes, AudienciaConsultarSolicitudes},
		{AccionConsultarDetalle, AudienciaConsultarDetalle},
	}
}

// Errores del caso de uso. Los adaptadores de transporte los traducen a
// códigos estables (claves i18n), nunca a texto.
var (
	ErrNoDisponible                = errors.New("seleccion: servicio no disponible")
	ErrDatosNoValidos              = errors.New("seleccion: datos no validos")
	ErrDeclaracionRequerida        = errors.New("seleccion: declaracion responsable requerida")
	ErrFueraDePlazo                = errors.New("seleccion: fuera de plazo")
	ErrClaveReutilizada            = errors.New("seleccion: clave reutilizada con otro contenido")
	ErrVersionObsoleta             = errors.New("seleccion: version obsoleta")
	ErrYaPresentada                = errors.New("seleccion: solicitud ya presentada")
	ErrRequisitoNoCumplido         = errors.New("seleccion: requisito no cumplido")
	ErrConvocatoriaActualizada     = errors.New("seleccion: convocatoria actualizada")
	ErrConvocatoriaNoDisponible    = errors.New("seleccion: convocatoria no disponible")
	ErrSolicitudNoEncontrada       = errors.New("seleccion: solicitud no encontrada")
	ErrDatosIncompletos            = errors.New("seleccion: solicitud incompleta")
	ErrSinBorrador                 = errors.New("seleccion: sin borrador")
	ErrServicioExternoNoDisponible = errors.New("seleccion: servicio externo no disponible")
)

// MaterialConsumoV3 es el material atestado que PostgreSQL consume en la
// misma transacción que el efecto. Lo satisface
// puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3.
type MaterialConsumoV3 interface {
	CapacidadCanonica() []byte
	DecisionCanonica() []byte
	MotivoCanonico() []byte
	ContextoActorCanonico() []byte
	PersonaVersion() uint64
	PerfilVersion() uint64
	PayloadVECAD3() []byte
	SobreCOSESign1() []byte
	EvidenciaVerificacion() []byte
	RaizPublicaSPKI() []byte
}

// EmisorMaterialV3 decide y emite el material atestado de una solicitud de
// autorización ligada. El adaptador aporta el PDP y la atestación.
type EmisorMaterialV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

// Reloj de la aplicación. El plazo que decide es el de la base.
type Reloj interface {
	Ahora() time.Time
}

// GeneradorReferencias crea referencias opacas nuevas ('sol_…').
type GeneradorReferencias interface {
	NuevaReferenciaSolicitud(context.Context) (string, error)
}

// AsociacionDatos liga el sobre a la persona, la convocatoria y la versión:
// un sobre copiado a otra fila no se descifra.
type AsociacionDatos struct {
	PersonaRef      string
	ConvocatoriaRef string
	Version         int
}

// SobreDatos son los datos personales cifrados fuera de la base.
type SobreDatos struct {
	ClaveRef string
	Nonce    []byte
	Cifrado  []byte
}

// ProtectorDatosSolicitud cifra los datos personales y calcula huellas con
// clave (documento de identidad, material de idempotencia) para no guardar
// ni comparar datos en claro. El claro solo existe dentro del callback.
type ProtectorDatosSolicitud interface {
	CifrarDatosSolicitud(context.Context, AsociacionDatos, []byte) (SobreDatos, error)
	ConDatosSolicitudDescifrados(context.Context, AsociacionDatos, SobreDatos, func([]byte) error) error
	HuellaConClave(ctx context.Context, dominio string, datos []byte) (string, error)
}

// ComandoGuardarBorrador llega a la persistencia con el material ya emitido.
type ComandoGuardarBorrador struct {
	PersonaRef, ConvocatoriaRef          string
	ConvocatoriaVersion, VersionEsperada int
	Clave, HuellaMaterial                string
	SolicitudRefNueva, Turno             string
	Sobre                                SobreDatos
	// DocumentoHuella y DocumentoParcial van vacíos si el borrador aún no
	// trae documento; DatosCompletos dice si puede presentarse.
	DocumentoHuella, DocumentoParcial string
	DatosCompletos                    bool
	Requisitos                        []domain.RequisitoDeclarado
	Meritos                           []domain.MeritoDeclarado
	Puntuacion                        baremacion.Puntos
	Material                          MaterialConsumoV3
}

// ResultadoGuardado de una versión del borrador.
type ResultadoGuardado struct {
	Reutilizada    bool
	SolicitudRef   string
	Version        int
	Puntuacion     baremacion.Puntos
	DatosCompletos bool
}

// ComandoPresentar presenta la versión que la persona vio.
type ComandoPresentar struct {
	PersonaRef, SolicitudRef         string
	VersionEsperada                  int
	Clave, HuellaMaterial, ReciboRef string
	Material                         MaterialConsumoV3
}

// ResultadoPresentacion con el justificante interno (no es un asiento de
// registro administrativo ni una firma).
type ResultadoPresentacion struct {
	Reutilizada        bool
	SolicitudRef       string
	NumeroJustificante string
	ReciboRef          string
	PresentadaEn       time.Time
	Puntuacion         baremacion.Puntos
}

// ResumenSolicitudPropia es una fila de «Mis solicitudes».
type ResumenSolicitudPropia struct {
	SolicitudRef, ConvocatoriaRef, ConvocatoriaTitulo string
	Estado                                            domain.EstadoSolicitud
	Version                                           int
	PresentadaEn                                      *time.Time
	NumeroJustificante                                string
	Puntuacion                                        baremacion.Puntos
}

// VersionSolicitud es la última versión guardada, con su sobre cifrado.
type VersionSolicitud struct {
	SolicitudRef, PersonaRef, ConvocatoriaRef string
	ConvocatoriaVersion, Version              int
	Estado                                    domain.EstadoSolicitud
	Turno                                     string
	DatosCompletos                            bool
	Sobre                                     SobreDatos
	Requisitos                                []domain.RequisitoDeclarado
	Meritos                                   []domain.MeritoDeclarado
	Puntuacion                                baremacion.Puntos
	ActualizadaEn                             time.Time
}

// FilaSolicitudRRHH es una solicitud presentada en el listado de RRHH.
type FilaSolicitudRRHH struct {
	PresentacionID                            int64
	SolicitudRef, PersonaRef, ConvocatoriaRef string
	Version                                   int
	NumeroJustificante, Turno                 string
	Sobre                                     SobreDatos
	DocumentoParcial                          string
	PresentadaEn                              time.Time
	Puntuacion                                baremacion.Puntos
}

// EventoHistoria de la solicitud (solo adición).
type EventoHistoria struct {
	Tipo    string
	Version int
	En      time.Time
}

// FichaSolicitudRRHH es la solicitud presentada completa, con la versión de
// la convocatoria con la que se presentó.
type FichaSolicitudRRHH struct {
	Fila         FilaSolicitudRRHH
	Convocatoria domain.ConvocatoriaPublicada
	Requisitos   []domain.RequisitoDeclarado
	Meritos      []domain.MeritoDeclarado
	Historia     []EventoHistoria
}

// RepositorioSolicitudes persiste solicitudes y consume la decisión V3 en
// la misma transacción que cada lectura o escritura.
type RepositorioSolicitudes interface {
	GuardarBorrador(context.Context, ComandoGuardarBorrador) (ResultadoGuardado, error)
	Presentar(context.Context, ComandoPresentar) (ResultadoPresentacion, error)
	ListarPropias(ctx context.Context, personaRef string, material MaterialConsumoV3) ([]ResumenSolicitudPropia, error)
	// LeerBorrador devuelve ErrSinBorrador si no hay solicitud.
	LeerBorrador(ctx context.Context, personaRef, convocatoriaRef string, material MaterialConsumoV3) (VersionSolicitud, error)
	ListarPresentadas(ctx context.Context, convocatoriaRef string, cursor int64, limite int, material MaterialConsumoV3) ([]FilaSolicitudRRHH, error)
	LeerFicha(ctx context.Context, solicitudRef string, material MaterialConsumoV3) (FichaSolicitudRRHH, error)
}

// RegistroConvocatorias publica y lee las convocatorias gobernadas.
type RegistroConvocatorias interface {
	// Publicar reutiliza la versión vigente si el contenido coincide.
	PublicarConvocatoria(context.Context, domain.Convocatoria) (version int, nueva bool, err error)
	ConvocatoriasVigentes(context.Context) ([]domain.ConvocatoriaPublicada, error)
}

// FuenteConvocatorias es el catálogo versionado del que se publican.
type FuenteConvocatorias interface {
	Convocatorias(context.Context) ([]domain.Convocatoria, error)
}

// PresentacionExterna identifica una presentación ante los servicios
// externos, sin datos personales.
type PresentacionExterna struct {
	SolicitudRef, ConvocatoriaRef, NumeroJustificante, ReciboRef string
	PresentadaEn                                                 time.Time
}

// Servicios externos declarados desde la fase 1 sin proveedor real. Un
// adaptador ausente responde ErrServicioExternoNoDisponible: la aplicación lo
// dice expresamente y nunca lo presenta como hecho.
type (
	FirmaSolicitud interface {
		FirmarPresentacion(context.Context, PresentacionExterna) error
	}
	RegistroSede interface {
		RegistrarPresentacion(context.Context, PresentacionExterna) error
	}
	PasarelaTasas interface {
		ComprobarTasa(context.Context, PresentacionExterna) error
	}
	NotificadorSolicitudes interface {
		NotificarPresentacion(context.Context, PresentacionExterna) error
	}
)

// EstadoServicioExterno tras la presentación.
type EstadoServicioExterno string

const (
	ServicioNoDisponible EstadoServicioExterno = "no_disponible"
	ServicioRealizado    EstadoServicioExterno = "realizado"
	ServicioFallido      EstadoServicioExterno = "error"
)
