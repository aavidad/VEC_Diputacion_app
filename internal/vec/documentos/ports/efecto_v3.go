package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Errores de dominio que el adaptador SQL distingue de la indisponibilidad.
// La frontera HTTP los traduce a 409, 422, 404 y 403; sólo
// ErrCapacidadNoDisponible significa dependencia caída (503).
var (
	ErrConflicto        = errors.New("documentos: conflicto con un registro existente")
	ErrValidacion       = errors.New("documentos: datos no válidos")
	ErrNoEncontrado     = errors.New("documentos: recurso no encontrado")
	ErrAccesoDenegado   = errors.New("documentos: acceso denegado")
	ErrEfectoV3NoValido = errors.New("documentos: efecto V3 no válido")
)

const (
	atributoPreimagenV3    = "preimagen_sha256"
	moduloRecursoV3        = "documentos"
	longitudMaximaEfectoV3 = 16384
)

// TipoRecursoV3 devuelve el tipo de recurso que AD3-60 exige para cada acción.
func TipoRecursoV3(accion string) (string, bool) {
	switch accion {
	case AccionAlta:
		return "documento_generado", true
	case AccionListar:
		return "expediente_documental", true
	case AccionDescargar:
		return "documento_original", true
	case AccionPrepararNotificacion:
		return "notificacion_preparada", true
	case AccionRegistrarExterno:
		return "documento_externo", true
	default:
		return "", false
	}
}

// RecursoV3 construye el recurso autorizable que liga una decisión V3 a la
// preimagen exacta del efecto. Su huella de contexto (ámbitos vacíos y un
// único atributo con el SHA-256 de la preimagen) es la que AD3 coteja como
// huella_efecto_sha256 y la que recalcula la fachada SQL (Documentos-4).
func RecursoV3(accion, recursoRef string, preimagen []byte) (vecdomain.RecursoAutorizable, error) {
	tipo, ok := TipoRecursoV3(accion)
	if !ok || !domain.ReferenciaOpacaValida(recursoRef) || len(preimagen) == 0 || len(preimagen) > longitudMaximaEfectoV3 {
		return vecdomain.RecursoAutorizable{}, ErrEfectoV3NoValido
	}
	suma := sha256.Sum256(preimagen)
	recurso := vecdomain.RecursoAutorizable{
		Referencia: recursoRef, ModuloID: moduloRecursoV3, Tipo: tipo,
		Ambitos:   map[string]string{},
		Atributos: map[string]string{atributoPreimagenV3: hex.EncodeToString(suma[:])},
	}
	if recurso.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ErrEfectoV3NoValido
	}
	return recurso, nil
}

// HuellaEfectoV3 es la huella del contexto de recurso para una preimagen:
// SHA-256 de {"ambitos":{},"atributos":{"preimagen_sha256":"<hex>"}}. Es
// idéntica a RecursoV3(...).HuellaContextoAutorizacionSHA256() y a la que
// calcula vec_documentos.huella_efecto_v1. Devuelve "" para una preimagen no
// admisible, que nunca coincide con una huella válida.
func HuellaEfectoV3(preimagen []byte) string {
	if len(preimagen) == 0 || len(preimagen) > longitudMaximaEfectoV3 {
		return ""
	}
	suma := sha256.Sum256(preimagen)
	contexto := `{"ambitos":{},"atributos":{"` + atributoPreimagenV3 + `":"` + hex.EncodeToString(suma[:]) + `"}}`
	huella := sha256.Sum256([]byte(contexto))
	return hex.EncodeToString(huella[:])
}
