package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrImagenNoAutenticado    = errors.New("usuarios imagen: no autenticado")
	ErrImagenProhibido        = errors.New("usuarios imagen: prohibido")
	ErrImagenConflicto        = errors.New("usuarios imagen: conflicto")
	ErrImagenNoEncontrada     = errors.New("usuarios imagen: foto no activa o no encontrada")
	ErrImagenPeticionInvalida = errors.New("usuarios imagen: peticion invalida")
	ErrImagenNoDisponible     = errors.New("usuarios imagen: no disponible")
)

const (
	AccionImagenConsultarPropia   = "vec.imagen.consultar_propia"
	AccionImagenConsultarAjena    = "vec.imagen.consultar_ajena"
	AccionImagenActualizar        = "vec.imagen.actualizar"
	AccionImagenRecuperarReserva  = "vec.imagen.recuperar_reserva"
	FinalidadImagenPropia         = "finalidad:usuarios:imagen-propia:v1"
	FinalidadImagenInterna        = "finalidad:usuarios:imagen-directorio-interno:v1"
	TamanoMaximoOriginalImagen    = 2 * 1024 * 1024
	DimensionMaximaOriginalImagen = 8192
	PixelesMaximosOriginalImagen  = 16 * 1024 * 1024
	LadoImagenProcesada           = 256
)

type AudienciaImagen string

const (
	AudienciaImagenPersonal AudienciaImagen = "portal_personal_autenticado"
	AudienciaImagenInterna  AudienciaImagen = "portal_interno_autenticado"
)

// La frontera de identidad debe construir esta orden desde sesión y audiencia
// verificadas. Ninguno de esos valores procede del cuerpo, URL o cabecera libre.
type OrdenImagen struct {
	actor     vecdomain.ContextoActor
	audiencia AudienciaImagen
	proveedor ProveedorMaterialImagen
}

func NuevaOrdenImagen(actor vecdomain.ContextoActor, audiencia AudienciaImagen, proveedor ProveedorMaterialImagen) (OrdenImagen, error) {
	metodo := actor.Principal.AuthMethod
	metodoPersonal := metodo == vecdomain.AuthMethodCertificate || metodo == vecdomain.AuthMethodDNIe
	metodoInterno := metodoPersonal || metodo == vecdomain.AuthMethodSSO || metodo == vecdomain.AuthMethodKerberos
	if actor.Validar() != nil || proveedor == nil || (audiencia != AudienciaImagenPersonal && audiencia != AudienciaImagenInterna) || (audiencia == AudienciaImagenPersonal && !metodoPersonal) || (audiencia == AudienciaImagenInterna && !metodoInterno) {
		return OrdenImagen{}, ErrImagenNoAutenticado
	}
	copia, err := actor.Clonar()
	if err != nil {
		return OrdenImagen{}, ErrImagenNoAutenticado
	}
	return OrdenImagen{actor: copia, audiencia: audiencia, proveedor: proveedor}, nil
}
func (o OrdenImagen) ContextoActor() (vecdomain.ContextoActor, error) {
	if o.proveedor == nil || o.actor.Validar() != nil || (o.audiencia != AudienciaImagenPersonal && o.audiencia != AudienciaImagenInterna) {
		return vecdomain.ContextoActor{}, ErrImagenNoAutenticado
	}
	return o.actor.Clonar()
}
func (o OrdenImagen) Audiencia() AudienciaImagen         { return o.audiencia }
func (o OrdenImagen) Proveedor() ProveedorMaterialImagen { return o.proveedor }

type MaterialImagen struct {
	ActorPersonaRef    string
	TitularPersonaRef  string
	PerfilRef          string
	Accion             string
	FinalidadRef       string
	Audiencia          AudienciaImagen
	VersionEsperada    uint64
	CatalogoVersionRef string
	ClaveOperacion     string
	HuellaPeticion     string
	Eleccion           domain.EleccionImagen
}
type ProveedorMaterialImagen interface {
	ProveerMaterialImagen(context.Context, MaterialImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type EstadoImagen struct {
	PersonaRef         string                `json:"persona_ref"`
	Version            uint64                `json:"version"`
	CatalogoVersionRef string                `json:"catalogo_version_ref"`
	Eleccion           domain.EleccionImagen `json:"eleccion"`
}
type VistaImagen struct {
	Catalogo domain.CatalogoImagen `json:"catalogo"`
	Estado   EstadoImagen          `json:"estado"`
	// FotoDisponible sólo se consulta cuando la elección persistida es foto.
	// Si es false, la presentación usa iniciales sin ocultar la elección real.
	FotoDisponible bool `json:"foto_disponible"`
	// NombreAutorizado sólo lo aporta una fuente de identidad autorizada para
	// esa lectura. Nunca se deriva de la referencia opaca ni del cliente.
	NombreAutorizado string `json:"nombre_autorizado,omitempty"`
}
type PeticionElegirImagen struct {
	VersionEsperada    uint64            `json:"version_esperada"`
	CatalogoVersionRef string            `json:"catalogo_version_ref"`
	ClaveOperacion     string            `json:"clave_operacion"`
	Modo               domain.ModoImagen `json:"modo"`
	Paleta             string            `json:"paleta"`
	Icono              string            `json:"icono,omitempty"`
}
type PeticionSubirImagen struct {
	VersionEsperada    uint64 `json:"version_esperada"`
	CatalogoVersionRef string `json:"catalogo_version_ref"`
	ClaveOperacion     string `json:"clave_operacion"`
	Paleta             string `json:"paleta"`
	// TipoDeclarado es orientativo; TransformadorImagen detecta el tipo real.
	TipoDeclarado string `json:"-"`
	Original      []byte `json:"-"`
}
type ReciboImagen struct {
	ReciboRef          string                `json:"recibo_ref"`
	PersonaRef         string                `json:"persona_ref"`
	Version            uint64                `json:"version"`
	CatalogoVersionRef string                `json:"catalogo_version_ref"`
	Eleccion           domain.EleccionImagen `json:"eleccion"`
	FechaUTC           time.Time             `json:"fecha_utc"`
	Replay             bool                  `json:"replay"`
}

// RegistroImagen ejecuta consumo V3, CAS, historia inmutable, auditoría y
// recibo en una sola transacción. Recuperar reautoriza aun en replay. Leer
// audita tanto éxitos como denegaciones según el contrato central.
type RegistroImagen interface {
	CatalogoVigente(context.Context) (domain.CatalogoImagen, error)
	Leer(context.Context, OrdenImagen, MaterialImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (EstadoImagen, bool, error)
	Recuperar(context.Context, OrdenImagen, MaterialImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboImagen, bool, error)
	Guardar(context.Context, OrdenImagen, MaterialImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboImagen, error)
}

type LimitesTransformacionImagen struct {
	MaxBytes     int
	MaxDimension int
	MaxPixeles   int
	LadoSalida   int
}
type ImagenProcesada struct {
	Bytes               []byte
	TipoReal            string
	TipoSalida          string
	AnchoOriginal       int
	AltoOriginal        int
	Ancho               int
	Alto                int
	OrientacionAplicada bool
	MetadatosEliminados bool
}

// El transformador detecta firma y decodifica con límites antes de asignar
// píxeles, aplica orientación, recorta cuadrado, reescala y recodifica. No
// conserva el original temporal ni datos EXIF/ubicación en la salida.
type TransformadorImagen interface {
	Procesar(context.Context, []byte, LimitesTransformacionImagen) (ImagenProcesada, error)
}
type ReservaImagen struct {
	ActorPersonaRef    string
	PerfilRef          string
	Audiencia          AudienciaImagen
	FinalidadRef       string
	PersonaRef         string
	ClaveOperacion     string
	HuellaPeticion     string
	OriginalSHA        string
	VersionEsperada    uint64
	CatalogoVersionRef string
	Paleta             string
	DocumentoRef       string
	HuellaContenido    string
}

// ContenidoImagen es la proyección mínima que HTTP podrá entregar tras una
// lectura autorizada. El adaptador devuelve bytes propios ya recodificados,
// verifica la huella y versión del objeto y jamás entrega el original. El
// adaptador limita la lectura antes de asignar memoria. El consumidor libera
// estos bytes tras responder; cabeceras de caché y tipo seguro pertenecen a
// HTTP, no al dominio.
type ContenidoImagen struct {
	DocumentoRef string
	Tipo         string
	Bytes        []byte
}

// Documentos comunes custodia sólo la salida recodificada. Cada método
// reautoriza actor, titular, acción, audiencia y finalidad con su autoridad
// propia; el V3 estructural obtenido por Usuarios no basta para Documentos.
// La reserva es durable e idempotente por persona+clave+huella y permanece
// reconciliable hasta confirmación/abandono explícito. No hay transacción
// entre módulos. El adaptador verifica las dos huellas y clona los bytes.
type CustodiaImagen interface {
	Reservar(context.Context, OrdenImagen, MaterialImagen, ReservaImagen, ImagenProcesada) (ReservaImagen, error)
	RecuperarReserva(context.Context, OrdenImagen, MaterialImagen, string, string) (ReservaImagen, bool, error)
	ConfirmarReserva(context.Context, OrdenImagen, MaterialImagen, ReservaImagen) error
	Disponible(context.Context, OrdenImagen, MaterialImagen, string) (bool, error)
	// Abrir reautoriza en Documentos el actor, titular, finalidad, audiencia,
	// versión de elección activa y la referencia exacta; nunca un recibo
	// histórico ni la mera existencia del objeto concede acceso.
	Abrir(context.Context, OrdenImagen, MaterialImagen, string) (ContenidoImagen, error)
}

// Documentos consume este puerto justo antes de leer el objeto. La respuesta
// procede del estado actual de Usuarios con V3 nuevo para actor, titular y
// audiencia; no de una reserva, recibo ni referencia histórica. false niega
// la apertura. El puente de composición puede adaptar esta interfaz sin que
// Documentos importe tablas o tipos privados de Usuarios.
type ComprobadorReferenciaImagenActiva interface {
	ReferenciaActiva(context.Context, OrdenImagen, string, string) (bool, error)
}

// Una fuente de identidad separada devuelve el nombre visible autorizado sólo
// después de la autorización nominal de la lectura; vacío significa fallback.
type NombreImagenAutorizado interface {
	NombreVisible(context.Context, OrdenImagen, MaterialImagen, string) (string, error)
}
