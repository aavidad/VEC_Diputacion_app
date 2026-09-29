package ports

import (
	"context"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Lectura del correo activo de «Mis correos» para avisar a una persona
// candidata de un llamamiento de Bolsa. Es la única lectura del correo de una
// persona por otra: devuelve, como mucho, el correo activo y verificado que la
// persona añadió y confirmó desde el área personal externa, y sólo mientras
// RRHH emite ese llamamiento.
//
// El permiso es el mismo que ya tiene quien emite el llamamiento: acción
// llamamiento.emitir.v1 sobre la bolsa constituida, con la finalidad de
// gestión de llamamientos. La capacidad V3 lleva su propia audiencia, que sólo
// consume Usuarios (AD3-109). La bolsa, el llamamiento y la persona candidata
// quedan fijados en la huella auditada del recurso; que la candidata sea de
// ese llamamiento lo comprueba Bolsa antes de preguntar.
const (
	EsquemaMaterialCorreoAvisos             = "vec.usuarios.correo-avisos-llamamiento.v1"
	AudienciaCorreoAvisosLlamamientoInterna = "vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1"
	AccionV3CorreoAvisosLlamamiento         = "llamamiento.emitir.v1"
	ModuloV3CorreoAvisosLlamamiento         = "bolsa"
	TipoRecursoV3CorreoAvisosLlamamiento    = "bolsa_constituida"
	FinalidadCorreoAvisosLlamamiento        = "gestion_llamamientos_bolsa"
)

// SolicitudCorreoAvisos contiene sólo referencias opacas que Bolsa ya conoce.
// Ninguna identifica a la persona fuera de VEC.
type SolicitudCorreoAvisos struct {
	BolsaRef       string
	UnidadRef      string
	AmbitoRef      string
	LlamamientoRef string
	CandidatoRef   string
}

// MaterialCorreoAvisos es la preimagen que firma la V3 (su SHA-256 va en el
// recurso) y que PostgreSQL vuelve a comprobar. No lleva datos personales.
type MaterialCorreoAvisos struct {
	Esquema        string `json:"esquema"`
	Superficie     string `json:"superficie"`
	FinalidadRef   string `json:"finalidad_ref"`
	BolsaRef       string `json:"bolsa_ref"`
	UnidadRef      string `json:"unidad_ref"`
	AmbitoRef      string `json:"ambito_ref"`
	LlamamientoRef string `json:"llamamiento_ref"`
	CandidatoRef   string `json:"candidato_ref"`
}

// ProveedorMaterialCorreoAvisos emite una V3 fresca para el material exacto
// con la identidad de quien emite el llamamiento. La composición lo liga a la
// petición en curso; Usuarios no conoce la política que concede el permiso.
type ProveedorMaterialCorreoAvisos interface {
	ProveerMaterialCorreoAvisos(context.Context, MaterialCorreoAvisos, []byte) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenCorreoAvisos transporta el emisor V3 de la petición en curso.
type OrdenCorreoAvisos struct {
	Proveedor ProveedorMaterialCorreoAvisos
}

// LecturaCorreoAvisos es la respuesta de PostgreSQL. PersonaRef sólo sirve
// para abrir el sobre (va en sus datos asociados) y no sale de Usuarios.
type LecturaCorreoAvisos struct {
	Encontrado   bool
	PersonaRef   string
	CorreoRef    string
	AuditoriaRef string
	Sobre        SobreDireccionCorreo
}

// RegistroCorreoAvisos consume la V3 y lee en la misma transacción.
type RegistroCorreoAvisos interface {
	LeerCorreoActivoAvisos(context.Context, []byte, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (LecturaCorreoAvisos, error)
}

// ResultadoCorreoAvisos es lo único que vuelve a quien pidió el aviso: si
// había correo activo y su referencia opaca. La dirección sólo existe dentro
// de la llamada que la usa.
type ResultadoCorreoAvisos struct {
	Encontrado bool
	CorreoRef  string
}
