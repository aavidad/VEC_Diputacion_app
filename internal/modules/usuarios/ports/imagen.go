package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Errores nominales de «Mi imagen». La API los traduce a 401/403/409/413/
// 422/503. Ninguno lleva bytes, metadatos ni nombres de fichero.
var (
	ErrImagenNoAutenticado    = errors.New("usuarios imagen: no autenticado")
	ErrImagenProhibido        = errors.New("usuarios imagen: prohibido")
	ErrImagenConflicto        = errors.New("usuarios imagen: conflicto")
	ErrImagenPeticionInvalida = errors.New("usuarios imagen: peticion invalida")
	ErrImagenFotoNoAdmitida   = errors.New("usuarios imagen: foto no admitida")
	ErrImagenFotoGrande       = errors.New("usuarios imagen: foto demasiado grande")
	ErrImagenNoDisponible     = errors.New("usuarios imagen: no disponible")
)

const (
	FinalidadImagenPropia            = "finalidad:usuarios:imagen-propia:v1"
	AccionConsultarImagen            = "vec.imagen.consultar"
	AccionActualizarImagen           = "vec.imagen.actualizar"
	TipoRecursoImagen                = "imagen_persona"
	AudienciaConsultarImagenInterna  = "vec_usuarios.imagen.consultar.interna_corporativa.v1"
	AudienciaActualizarImagenInterna = "vec_usuarios.imagen.actualizar.interna_corporativa.v1"
	AudienciaConsultarImagenExterna  = "vec_usuarios.imagen.consultar.externa_personal.v1"
	AudienciaActualizarImagenExterna = "vec_usuarios.imagen.actualizar.externa_personal.v1"
)

// Límites de la foto. Se comprueban antes de decodificar: tamaño del fichero,
// ancho, alto y número de píxeles de la cabecera. La salida es un JPEG
// cuadrado de LadoFotoImagen px, recodificado y sin metadatos.
const (
	TamanoMaximoFotoImagen    = 5 << 20
	DimensionMaximaFotoImagen = 8000
	PixelesMaximosFotoImagen  = 16_000_000
	LadoFotoImagen            = 256
	TamanoMaximoFotoCustodia  = 256 << 10
	TipoFotoImagen            = "image/jpeg"
)

// CamposPermitidosImagen es el contrato de campos que la concesión V3 y
// AD3-108 exigen por acción. Un cambio aquí exige otra migración AD3.
func CamposPermitidosImagen(accion string) []string {
	switch accion {
	case AccionConsultarImagen, AccionActualizarImagen:
		return []string{"foto", "icono", "modo", "paleta", "version"}
	}
	return nil
}

type ProveedorMaterialImagen interface {
	ProveerMaterialImagen(context.Context, vecdomain.VinculoAutenticacionActorV2, MaterialImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenImagen transporta la identidad validada en dominio y el puerto V3.
// Solo la construye la frontera; nunca un campo HTTP.
type OrdenImagen struct {
	Identidad domain.IdentidadImagen
	Proveedor ProveedorMaterialImagen
}

// MaterialImagen es la preimagen V3/SQL. Solo lleva códigos cerrados, la
// huella de la petición y la huella del JPEG ya recodificado; nunca bytes.
type MaterialImagen struct {
	Superficie         vecdomain.SuperficieAutenticacionActorV1 `json:"superficie"`
	PersonaRef         string                                   `json:"persona_ref"`
	PerfilRef          string                                   `json:"perfil_ref"`
	Accion             string                                   `json:"accion"`
	FinalidadRef       string                                   `json:"finalidad_ref"`
	CatalogoVersionRef string                                   `json:"catalogo_version_ref"`
	VersionEsperada    uint64                                   `json:"version_esperada"`
	ClaveOperacion     string                                   `json:"clave_operacion"`
	HuellaPeticion     string                                   `json:"huella_peticion"`
	Eleccion           domain.EleccionImagen                    `json:"eleccion"`
	FotoSHA256         string                                   `json:"foto_sha256"`
}

// Operaciones de la API: «elegir» fija iniciales o icono (o conserva la foto
// ya guardada cambiando la paleta); «subir_foto» sustituye la foto.
const (
	OperacionElegirImagen    = "elegir"
	OperacionSubirFotoImagen = "subir_foto"
)

type PeticionImagen struct {
	Operacion          string
	VersionEsperada    uint64
	CatalogoVersionRef string
	ClaveOperacion     string
	Eleccion           domain.EleccionImagen
	// Foto es el fichero tal como llega; solo vive en memoria hasta
	// recodificarse y nunca se guarda ni se registra.
	Foto []byte
}

type EstadoImagen struct {
	PersonaRef         string                `json:"-"`
	Version            uint64                `json:"version"`
	CatalogoVersionRef string                `json:"catalogo_version_ref"`
	Eleccion           domain.EleccionImagen `json:"eleccion"`
}

// FotoImagen es el JPEG recodificado que se entrega a su titular.
type FotoImagen struct {
	Tipo  string `json:"tipo"`
	Datos []byte `json:"datos"`
}

type VistaImagen struct {
	Catalogo domain.CatalogoImagen `json:"catalogo"`
	Estado   EstadoImagen          `json:"estado"`
	Foto     *FotoImagen           `json:"foto"`
}

type ReciboImagen struct {
	ReciboRef          string                `json:"recibo_ref"`
	PersonaRef         string                `json:"-"`
	Version            uint64                `json:"version"`
	CatalogoVersionRef string                `json:"catalogo_version_ref"`
	Eleccion           domain.EleccionImagen `json:"eleccion"`
	FotoNueva          bool                  `json:"foto_nueva"`
	FotoRetirada       bool                  `json:"foto_retirada"`
	FechaUTC           time.Time             `json:"fecha_utc"`
	Replay             bool                  `json:"replay"`
}

// FotoProcesada es la salida del transformador: JPEG cuadrado, recodificado
// desde los píxeles, sin EXIF, GPS ni perfiles. Bytes es propiedad del caso de uso.
type FotoProcesada struct {
	Bytes  []byte
	SHA256 string
}

// TransformadorFoto verifica el formato por su contenido (JPEG, PNG o WebP),
// lee ancho y alto de la cabecera y rechaza antes de decodificar si superan
// los límites; después orienta, recorta, reescala y recodifica.
type TransformadorFoto interface {
	Procesar(context.Context, []byte) (FotoProcesada, error)
}

// RegistroImagen ejecuta en una transacción la V3 fresca, el CAS, la custodia
// de la foto en Documentos, el estado, la historia y el recibo. Recuperar
// consume V3 aun cuando la clave no existe.
type RegistroImagen interface {
	CatalogoVigente(context.Context, OrdenImagen) (domain.CatalogoImagen, error)
	Consultar(context.Context, OrdenImagen, MaterialImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (EstadoImagen, bool, *FotoImagen, error)
	RecuperarOperacion(context.Context, OrdenImagen, MaterialImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboImagen, bool, error)
	Guardar(context.Context, OrdenImagen, MaterialImagen, []byte, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboImagen, error)
}
