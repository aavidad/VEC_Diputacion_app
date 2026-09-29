// Package ports define los contratos del módulo Aspirantes: autorización V3,
// registro durable, criptografía y catálogo de campos. No contiene
// implementaciones.
package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/aspirantes/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Errores nominales. La API los traduce a 401/403/404/409/422/503 sin
// revelar si un documento tiene ficha fuera de la propia.
var (
	ErrNoAutenticado  = errors.New("aspirantes: no autenticado")
	ErrProhibido      = errors.New("aspirantes: prohibido")
	ErrInvalida       = errors.New("aspirantes: peticion invalida")
	ErrConflicto      = errors.New("aspirantes: conflicto")
	ErrFichaExistente = errors.New("aspirantes: la ficha ya existe")
	ErrSinFicha       = errors.New("aspirantes: sin ficha")
	ErrNoDisponible   = errors.New("aspirantes: no disponible")
)

const (
	ModuloID            = "aspirantes"
	TipoRecursoFicha    = "ficha_aspirante_propia"
	FinalidadFicha      = "finalidad:aspirantes:ficha-propia:v1"
	AccionConsultar     = "vec.aspirantes.ficha.consultar"
	AccionAlta          = "vec.aspirantes.ficha.alta"
	AccionRectificar    = "vec.aspirantes.ficha.rectificar"
	AudienciaConsultar  = "vec_aspirantes.ficha.consultar.externa_personal.v1"
	AudienciaAlta       = "vec_aspirantes.ficha.alta.externa_personal.v1"
	AudienciaRectificar = "vec_aspirantes.ficha.rectificar.externa_personal.v1"
)

// CamposPermitidos es el contrato de campos que exigen la concesión V3 y la
// migración AD3 del consumidor. Cambiarlo exige otra migración.
func CamposPermitidos(accion string) []string {
	switch accion {
	case AccionConsultar, AccionAlta:
		return []string{"codigo_postal", "documento", "domicilio", "movil", "nombre", "primer_apellido", "segundo_apellido", "telefono", "version"}
	case AccionRectificar:
		return []string{"codigo_postal", "domicilio", "movil", "telefono", "version"}
	}
	return nil
}

// Audiencia devuelve la audiencia V3 de cada acción. Solo existe la
// superficie externa.
func Audiencia(accion string) (string, error) {
	switch accion {
	case AccionConsultar:
		return AudienciaConsultar, nil
	case AccionAlta:
		return AudienciaAlta, nil
	case AccionRectificar:
		return AudienciaRectificar, nil
	}
	return "", ErrProhibido
}

// ProveedorMaterialFicha emite la V3 fresca ligada al material exacto.
type ProveedorMaterialFicha interface {
	ProveerMaterialFicha(context.Context, vecdomain.VinculoAutenticacionActorV2, MaterialFicha) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenFicha transporta la sesión acreditada, la identidad del certificado y
// el puerto V3. La aplicación vuelve a validar ambas en cada uso.
type OrdenFicha struct {
	Sesion    domain.SesionAspirante
	Identidad domain.IdentidadAcreditada
	Proveedor ProveedorMaterialFicha
}

// HuellaSemantica es un HMAC versionado de la preimagen de una petición. Las
// retenidas permiten reconocer una repetición tras rotar la clave.
type HuellaSemantica struct {
	ClaveRef string `json:"clave_ref"`
	Valor    string `json:"valor"`
}

type HuellasSemanticas struct {
	Activa    HuellaSemantica   `json:"activa"`
	Retenidas []HuellaSemantica `json:"retenidas"`
}

// IndiceDocumento es el índice ciego del documento: HMAC con clave propia.
// Permite encontrar la ficha y detectar duplicados sin descifrar.
type IndiceDocumento struct {
	ClaveRef string `json:"clave_ref"`
	Valor    string `json:"valor"`
}

// MaterialFicha es la preimagen V3/SQL. Solo lleva referencias opacas, HMAC
// y el índice ciego; nunca nombre, documento ni contacto en claro.
type MaterialFicha struct {
	Superficie      vecdomain.SuperficieAutenticacionActorV1 `json:"superficie"`
	PersonaRef      string                                   `json:"persona_ref"`
	PerfilRef       string                                   `json:"perfil_ref"`
	Accion          string                                   `json:"accion"`
	FinalidadRef    string                                   `json:"finalidad_ref"`
	VersionEsperada uint64                                   `json:"version_esperada"`
	ClaveOperacion  string                                   `json:"clave_operacion"`
	HuellasPeticion HuellasSemanticas                        `json:"huellas_peticion"`
	IndiceDocumento IndiceDocumento                          `json:"indice_documento"`
}

// SobreCifrado es un valor cifrado con AEAD. El registro lo guarda tal cual.
type SobreCifrado struct {
	ClaveRef string
	Nonce    []byte
	Cifrado  []byte
}

// ValorCifrado es una versión de un campo. Sobre nil significa «retirado».
type ValorCifrado struct {
	Campo   domain.CampoFicha
	Version uint64
	Origen  domain.OrigenValor
	Sobre   *SobreCifrado
}

// DocumentoCifrado es una entrada del historial de documentos de identidad.
type DocumentoCifrado struct {
	DocumentoRef string
	Tipo         domain.TipoDocumento
	Pais         string
	Sobre        SobreCifrado
	Indice       IndiceDocumento
}

// FichaNueva es todo lo que escribe un alta, ya cifrado.
type FichaNueva struct {
	AspiranteRef string
	Documento    DocumentoCifrado
	Valores      []ValorCifrado
	CatalogoRef  string
}

// FichaCifrada es la ficha vigente tal como la devuelve el registro: el
// documento encontrado por el índice y el último valor de cada campo.
type FichaCifrada struct {
	AspiranteRef string
	Version      uint64
	Documento    DocumentoCifrado
	Valores      []ValorCifrado
	AccesoRef    string
}

// EstadoParaCambio es lo que el registro revela dentro de la transacción de
// una rectificación, después de consumir la V3 y bloquear la ficha.
type EstadoParaCambio struct {
	AspiranteRef string
	Version      uint64
	Presentes    []domain.CampoFicha
}

// CifradorCambios recibe el estado bloqueado y devuelve los valores ya
// cifrados con la ficha y la versión que van a escribirse. El claro nunca
// pasa por el registro.
type CifradorCambios interface {
	CifrarCambios(context.Context, EstadoParaCambio) ([]ValorCifrado, error)
}

type ReciboFicha struct {
	ReciboRef string
	Accion    string
	Version   uint64
	FechaUTC  time.Time
	Replay    bool
}

// RegistroFichas es la única puerta durable. Cada llamada consume la V3
// antes de mirar ninguna fila y escribe estado, historia, recibo, acceso y
// evento de salida en la misma transacción. Una clave de operación ya usada
// devuelve el recibo original; con otra petición, ErrConflicto.
type RegistroFichas interface {
	// ConsultarPropia devuelve la ficha o false si el documento no tiene
	// ficha. Cada lectura con ficha queda anotada con su finalidad.
	ConsultarPropia(context.Context, OrdenFicha, MaterialFicha, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (FichaCifrada, bool, error)
	// Alta devuelve ErrFichaExistente si el índice ya pertenece a una ficha.
	Alta(context.Context, OrdenFicha, MaterialFicha, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, FichaNueva) (ReciboFicha, error)
	// Rectificar bloquea la ficha, comprueba la versión, pide los valores
	// cifrados al CifradorCambios y los escribe con historia y motivo.
	Rectificar(context.Context, OrdenFicha, MaterialFicha, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, domain.MotivoCambio, string, CifradorCambios) (ReciboFicha, error)
}

// ProtectorFicha cifra y descifra con AEAD ligado a ficha, campo y versión,
// y calcula el índice ciego. Las claves son del portal externo.
type ProtectorFicha interface {
	CifrarValor(ctx context.Context, aspiranteRef string, campo domain.CampoFicha, version uint64, claro []byte) (SobreCifrado, error)
	DescifrarValor(ctx context.Context, aspiranteRef string, campo domain.CampoFicha, version uint64, sobre SobreCifrado) ([]byte, error)
	CifrarDocumento(ctx context.Context, aspiranteRef, documentoRef string, claro []byte) (SobreCifrado, error)
	DescifrarDocumento(ctx context.Context, aspiranteRef, documentoRef string, sobre SobreCifrado) ([]byte, error)
	IndiceDocumento(ctx context.Context, documento domain.DocumentoIdentidad) (IndiceDocumento, error)
}

// SelladorHuella calcula el HMAC semántico de una petición. La preimagen
// solo vive en memoria durante la llamada.
type SelladorHuella interface {
	SellarHuella(context.Context, []byte) (HuellasSemanticas, error)
}

// ExigenciasContacto dice qué datos de contacto pide el catálogo de datos
// personales y qué versión lo justifica (CatalogoRef queda en la historia).
// Sin campos no se pide ninguno. Ejemplo es cierto si sale de un paquete de
// ejemplo pendiente de RRHH y del DPD.
type ExigenciasContacto struct {
	CatalogoRef string
	Campos      []domain.ExigenciaCampo
	Ejemplo     bool
}

// CatalogoExigenciasFicha es el contrato de lectura con el catálogo de datos
// personales (F1.2, `internal/vec/datospersonales`). El adaptador resuelve
// los tipos de convocatoria que cuentan para la persona en el momento de la
// inscripción: mientras Bolsa no use `asp_`, los que se configuren para el
// área personal. Un error significa «no se puede pedir nada», nunca «todo».
type CatalogoExigenciasFicha interface {
	ExigenciasContactoFichaPropia(context.Context) (ExigenciasContacto, error)
}
