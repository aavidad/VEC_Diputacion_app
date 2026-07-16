package ports

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	// VersionIndiceIdempotenciaBaremacionV1 identifica la derivacion HMAC
	// adoptada por DEC-045. Una rotacion de clave cambia ClaveHMACRef, no esta
	// version del esquema.
	VersionIndiceIdempotenciaBaremacionV1 uint16 = 1
	VersionPrincipalEstableBaremacionV1   uint16 = 1
	VersionSeudonimoSujetoBaremacionV1    uint16 = 1

	// VersionIntencionCambioBaremacionV1 cubre la incorporacion de una decision
	// tecnica ya firmada, custodiada y retenida. El alta inicial necesitara una
	// proyeccion distinta: no se admite rellenar sus campos con valores ficticios.
	VersionIntencionCambioBaremacionV1 uint16 = 1

	// VersionHMACIntencionCambioBaremacionV1 versiona el sobre criptografico,
	// no la clave. ClaveHMACRef identifica la clave concreta del llavero y
	// permite verificar sellos anteriores despues de una rotacion.
	VersionHMACIntencionCambioBaremacionV1 uint16 = 1

	// VersionHMACMotivoBaremacionV1 pertenece a un dominio criptografico
	// separado de indice, sujeto e intencion.
	VersionHMACMotivoBaremacionV1 uint16 = 1

	// VersionHMACManifiestoMaterialBaremacionV2 identifica el sello nominal
	// del manifiesto estable V2; no admite el sobre textual generico de V1.
	VersionHMACManifiestoMaterialBaremacionV2 uint16 = 2
)

const (
	// EsquemaCanonicoPrincipalEstableBaremacionV1 y
	// EsquemaCanonicoIndiceIdempotenciaBaremacionV1 fijan las dos formulas de
	// DEC-045. PoliticaDerivacionIdempotenciaBaremacionDEC045V1 forma parte de
	// las preimagenes y del material atestado; no es un texto declarativo libre.
	EsquemaCanonicoSeudonimoSujetoBaremacionV1       = "vec.bolsa.seudonimo-sujeto.v1"
	EsquemaCanonicoPrincipalEstableBaremacionV1      = "vec.bolsa.principal-estable.v1"
	EsquemaCanonicoIndiceIdempotenciaBaremacionV1    = "vec.bolsa.indice-idempotencia.v1"
	PoliticaDerivacionIdempotenciaBaremacionDEC045V1 = "vec.bolsa.politica-derivacion.dec-045.v1"

	esquemaCanonicoIndiceIdempotenciaBaremacionV1    = EsquemaCanonicoIndiceIdempotenciaBaremacionV1
	esquemaCanonicoIntencionCambioBaremacionV1       = "vec.bolsa.intencion-cambio-baremacion.v1"
	esquemaFingerprintSemanticoIntencionBaremacionV1 = "vec.bolsa.fingerprint-semantico-intencion.v1"
	esquemaResolucionIdentidadInternaBaremacionV1    = "vec.bolsa.resolucion-identidad-interna.v1"
	esquemaHuellaSnapshotIdentidadBaremacionV1       = "vec.bolsa.huella-snapshot-identidad.v1"
	longitudIdentidadInternaEstableBaremacion        = 32
)

// EsquemaMaterialEstableBaremacion cierra los artefactos admitidos por la
// intencion. Las constantes V2 son reservas de contrato: los productores y
// verificadores V2 aun deben existir antes de abrir este flujo. Manifiesto V1,
// que incorpora autorizaciones efimeras, queda explicitamente excluido.
type EsquemaMaterialEstableBaremacion string

const (
	EsquemaManifiestoMaterialEstableBaremacionV2 EsquemaMaterialEstableBaremacion = "vec.bolsa.manifiesto-material-estable.v2"
	EsquemaPlanFirmaDurableBaremacionV2          EsquemaMaterialEstableBaremacion = "vec.bolsa.plan-firma-durable-material.v2"
	EsquemaReciboRecuperacionBaremacionV2        EsquemaMaterialEstableBaremacion = "vec.bolsa.recibo-recuperacion-firmado.v2"
	EsquemaReciboCustodiaBaremacionV2            EsquemaMaterialEstableBaremacion = "vec.bolsa.recibo-custodia-firmado.v2"
	EsquemaReciboRetencionBaremacionV2           EsquemaMaterialEstableBaremacion = "vec.bolsa.recibo-retencion-firmado.v2"

	VersionManifiestoMaterialEstableBaremacionV2 uint16 = 2
	VersionPlanFirmaDurableBaremacionV2          uint16 = 2
	VersionReciboRecuperacionBaremacionV2        uint16 = 2
	VersionReciboCustodiaBaremacionV2            uint16 = 2
	VersionReciboRetencionBaremacionV2           uint16 = 2
)

type EstadoPlanFirmaDurableBaremacion string

const EstadoPlanFirmaDurableCompletado EstadoPlanFirmaDurableBaremacion = "completado"

func (e EstadoPlanFirmaDurableBaremacion) Valido() bool {
	return e == EstadoPlanFirmaDurableCompletado
}

// EstadoInmovilizacionObjetoBaremacion evita que el valor cero se interprete
// como false. Ambos estados son materiales y deben proceder del recibo V2.
type EstadoInmovilizacionObjetoBaremacion string

const (
	EstadoInmovilizacionNoAplicada EstadoInmovilizacionObjetoBaremacion = "no_aplicada"
	EstadoInmovilizacionAplicada   EstadoInmovilizacionObjetoBaremacion = "aplicada"
)

func (e EstadoInmovilizacionObjetoBaremacion) Valido() bool {
	return e == EstadoInmovilizacionNoAplicada || e == EstadoInmovilizacionAplicada
}

// ClaveClasificacionDocumentoBaremacion referencia un catalogo administrable;
// no fija clasificaciones en codigo ni acepta texto libre en la intencion.
type ClaveClasificacionDocumentoBaremacion string

func (c ClaveClasificacionDocumentoBaremacion) Valida() bool {
	return claveCatalogoConfiguracionBaremacionValida(string(c))
}

// ClaveFormatoDocumentoBaremacion selecciona una entrada administrable del
// catalogo historico de formatos, no un tipo fijado al compilar el nucleo.
type ClaveFormatoDocumentoBaremacion string

func (c ClaveFormatoDocumentoBaremacion) Valida() bool {
	return claveCatalogoConfiguracionBaremacionValida(string(c))
}

// MIMECanonicoDocumentoBaremacion conserva el media type concreto resuelto
// por el catalogo. No admite parametros, comodines ni mayusculas canonicas.
type MIMECanonicoDocumentoBaremacion string

func (m MIMECanonicoDocumentoBaremacion) Valido() bool {
	return mimeCanonicoDocumentoBaremacionValido(string(m))
}

// InstantaneaCatalogoFormatoDocumentoBaremacion fija la entrada historica que
// eligio el flujo. Su forma no prueba que exista: el verificador material de
// la TCB debe resolver CatalogoRef+Version+Huella y cotejar clave y MIME.
type InstantaneaCatalogoFormatoDocumentoBaremacion struct {
	CatalogoRef          string
	CatalogoVersion      uint32
	HuellaCatalogoSHA256 string
	FormatoClave         ClaveFormatoDocumentoBaremacion
	MIMECanonico         MIMECanonicoDocumentoBaremacion
}

func (i InstantaneaCatalogoFormatoDocumentoBaremacion) Validar() error {
	if !claveCatalogoConfiguracionBaremacionValida(i.CatalogoRef) ||
		i.CatalogoVersion < 1 || i.CatalogoVersion > maximoVersionCatalogoIntencion ||
		!huellaSHA256Valida(i.HuellaCatalogoSHA256) || !i.FormatoClave.Valida() ||
		!i.MIMECanonico.Valido() {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (InstantaneaCatalogoFormatoDocumentoBaremacion) String() string {
	return "[INSTANTANEA-CATALOGO-FORMATO-DOCUMENTO-BAREMACION-PROTEGIDA]"
}
func (InstantaneaCatalogoFormatoDocumentoBaremacion) GoString() string {
	return "ports.InstantaneaCatalogoFormatoDocumentoBaremacion{[PROTEGIDA]}"
}
func (i InstantaneaCatalogoFormatoDocumentoBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, i.String())
}
func (InstantaneaCatalogoFormatoDocumentoBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (InstantaneaCatalogoFormatoDocumentoBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (InstantaneaCatalogoFormatoDocumentoBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (i InstantaneaCatalogoFormatoDocumentoBaremacion) LogValue() slog.Value {
	return slog.StringValue(i.String())
}

// InstantaneaCatalogoClasificacionDocumentoBaremacion fija por referencia,
// version y huella la clasificacion aplicada al documento custodiado.
type InstantaneaCatalogoClasificacionDocumentoBaremacion struct {
	CatalogoRef          string
	CatalogoVersion      uint32
	HuellaCatalogoSHA256 string
	ClasificacionClave   ClaveClasificacionDocumentoBaremacion
}

func (i InstantaneaCatalogoClasificacionDocumentoBaremacion) Validar() error {
	if !claveCatalogoConfiguracionBaremacionValida(i.CatalogoRef) ||
		i.CatalogoVersion < 1 || i.CatalogoVersion > maximoVersionCatalogoIntencion ||
		!huellaSHA256Valida(i.HuellaCatalogoSHA256) || !i.ClasificacionClave.Valida() {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (InstantaneaCatalogoClasificacionDocumentoBaremacion) String() string {
	return "[INSTANTANEA-CATALOGO-CLASIFICACION-DOCUMENTO-BAREMACION-PROTEGIDA]"
}
func (InstantaneaCatalogoClasificacionDocumentoBaremacion) GoString() string {
	return "ports.InstantaneaCatalogoClasificacionDocumentoBaremacion{[PROTEGIDA]}"
}
func (i InstantaneaCatalogoClasificacionDocumentoBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, i.String())
}
func (InstantaneaCatalogoClasificacionDocumentoBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (InstantaneaCatalogoClasificacionDocumentoBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (InstantaneaCatalogoClasificacionDocumentoBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (i InstantaneaCatalogoClasificacionDocumentoBaremacion) LogValue() slog.Value {
	return slog.StringValue(i.String())
}

// EstadoDisponibilidadObjetoBaremacion prueba que el objeto incorporado sigue
// disponible. El valor cero, eliminado, pendiente o desconocido falla cerrado.
type EstadoDisponibilidadObjetoBaremacion string

const EstadoDisponibilidadObjetoActivoNoEliminado EstadoDisponibilidadObjetoBaremacion = "activo_no_eliminado"

func (e EstadoDisponibilidadObjetoBaremacion) Valido() bool {
	return e == EstadoDisponibilidadObjetoActivoNoEliminado
}

const (
	maximoVersionBaremacionIntencion    uint64 = 1<<63 - 1
	maximoVersionPoliticaFirmaIntencion uint32 = 1<<31 - 1
	maximoVersionPoliticaRetencion      uint32 = 1<<31 - 1
	maximoVersionCatalogoIntencion      uint32 = 1<<31 - 1

	// El limite mantiene acotada la busqueda durante una rotacion. Con ocho
	// generaciones de principal y ocho de indice, la aplicacion nunca prueba
	// mas de 64 combinaciones antes de declarar la operacion ausente.
	maximoGeneracionesHMACIdempotenciaBaremacion = 8
)

var (
	ErrSeudonimoSujetoBaremacionInvalido        = errors.New("bolsa: seudonimo HMAC de sujeto invalido")
	ErrPrincipalEstableBaremacionInvalido       = errors.New("bolsa: principal estable HMAC invalido")
	ErrHMACIntencionCambioBaremacionInvalido    = errors.New("bolsa: HMAC de intencion de cambio invalido")
	ErrHMACMotivoBaremacionInvalido             = errors.New("bolsa: HMAC de motivo de baremacion invalido")
	ErrHMACManifiestoMaterialBaremacionInvalido = errors.New("bolsa: HMAC de manifiesto material invalido")
	ErrSerializacionIdempotenciaBaremacion      = errors.New("bolsa: serializacion de idempotencia semantica prohibida")
	ErrCoincidenciaIdempotenciaAmbigua          = errors.New("bolsa: coincidencia de idempotencia ausente, multiple o ajena")
	ErrSeparacionDominiosClaveBaremacion        = errors.New("bolsa: separacion de dominios de clave no acreditada")
)

// indiceIdempotenciaBaremacion es material interno del testimonio combinado.
// No existe tipo, constructor, selector ni getter publico que permita promover
// una celda aislada o persistirla fuera del producto completo.
type indiceIdempotenciaBaremacion struct {
	Version         uint16
	GeneracionClave uint32
	ClaveHMACRef    string
	ValorHMAC       string
}

func (i indiceIdempotenciaBaremacion) validar() error {
	if i.Version != VersionIndiceIdempotenciaBaremacionV1 || i.GeneracionClave < 1 ||
		!claveCatalogoConfiguracionBaremacionValida(i.ClaveHMACRef) || !huellaSHA256Valida(i.ValorHMAC) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return nil
}

func (i indiceIdempotenciaBaremacion) igualConstante(otro indiceIdempotenciaBaremacion) bool {
	if i.validar() != nil || otro.validar() != nil {
		return false
	}
	return subtle.ConstantTimeEq(int32(i.GeneracionClave), int32(otro.GeneracionClave)) == 1 &&
		hmacVersionadoIgualConstante(
			i.Version, i.ClaveHMACRef, i.ValorHMAC,
			otro.Version, otro.ClaveHMACRef, otro.ValorHMAC,
		)
}

func (indiceIdempotenciaBaremacion) String() string {
	return "[INDICE-IDEMPOTENCIA-BAREMACION-INTERNO-PROTEGIDO]"
}
func (indiceIdempotenciaBaremacion) GoString() string {
	return "ports.indiceIdempotenciaBaremacion{[PROTEGIDO]}"
}
func (i indiceIdempotenciaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, i.String())
}
func (indiceIdempotenciaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (indiceIdempotenciaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (indiceIdempotenciaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (i indiceIdempotenciaBaremacion) LogValue() slog.Value { return slog.StringValue(i.String()) }

// principalEstableBaremacionHMAC solo puede nacer dentro del receptor efimero
// del lote combinado. No identifica al sujeto del expediente ni sale del
// producto nominal.
type principalEstableBaremacionHMAC struct {
	Version         uint16
	GeneracionClave uint32
	ClaveHMACRef    string
	ValorHMAC       string
}

func (p principalEstableBaremacionHMAC) validar() error {
	if p.Version != VersionPrincipalEstableBaremacionV1 || p.GeneracionClave < 1 ||
		!claveCatalogoConfiguracionBaremacionValida(p.ClaveHMACRef) || !huellaSHA256Valida(p.ValorHMAC) {
		return ErrPrincipalEstableBaremacionInvalido
	}
	return nil
}

func (p principalEstableBaremacionHMAC) igualConstante(otro principalEstableBaremacionHMAC) bool {
	if p.validar() != nil || otro.validar() != nil {
		return false
	}
	return subtle.ConstantTimeEq(int32(p.GeneracionClave), int32(otro.GeneracionClave)) == 1 &&
		hmacVersionadoIgualConstante(
			p.Version, p.ClaveHMACRef, p.ValorHMAC,
			otro.Version, otro.ClaveHMACRef, otro.ValorHMAC,
		)
}

func (principalEstableBaremacionHMAC) String() string {
	return "[PRINCIPAL-ESTABLE-BAREMACION-INTERNO-PROTEGIDO]"
}
func (principalEstableBaremacionHMAC) GoString() string {
	return "ports.principalEstableBaremacionHMAC{[PROTEGIDO]}"
}
func (p principalEstableBaremacionHMAC) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (principalEstableBaremacionHMAC) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (principalEstableBaremacionHMAC) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (principalEstableBaremacionHMAC) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (p principalEstableBaremacionHMAC) LogValue() slog.Value { return slog.StringValue(p.String()) }

func validarPrincipalesInternosBaremacion(principales []principalEstableBaremacionHMAC) error {
	if len(principales) < 1 || len(principales) > maximoGeneracionesHMACIdempotenciaBaremacion {
		return ErrPrincipalEstableBaremacionInvalido
	}
	referencias := make(map[string]struct{}, len(principales))
	valores := make(map[string]struct{}, len(principales))
	var generacionAnterior uint32
	for posicion, principal := range principales {
		if principal.validar() != nil ||
			(posicion > 0 && principal.GeneracionClave >= generacionAnterior) {
			return ErrPrincipalEstableBaremacionInvalido
		}
		if _, repetida := referencias[principal.ClaveHMACRef]; repetida {
			return ErrPrincipalEstableBaremacionInvalido
		}
		if _, repetido := valores[principal.ValorHMAC]; repetido {
			return ErrPrincipalEstableBaremacionInvalido
		}
		referencias[principal.ClaveHMACRef] = struct{}{}
		valores[principal.ValorHMAC] = struct{}{}
		generacionAnterior = principal.GeneracionClave
	}
	return nil
}

func validarIndicesInternosBaremacion(indices []indiceIdempotenciaBaremacion) error {
	if len(indices) < 1 || len(indices) > maximoGeneracionesHMACIdempotenciaBaremacion {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	referencias := make(map[string]struct{}, len(indices))
	valores := make(map[string]struct{}, len(indices))
	var generacionAnterior uint32
	for posicion, indice := range indices {
		if indice.validar() != nil || (posicion > 0 && indice.GeneracionClave >= generacionAnterior) {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
		if _, repetida := referencias[indice.ClaveHMACRef]; repetida {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
		if _, repetido := valores[indice.ValorHMAC]; repetido {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
		referencias[indice.ClaveHMACRef] = struct{}{}
		valores[indice.ValorHMAC] = struct{}{}
		generacionAnterior = indice.GeneracionClave
	}
	return nil
}

// ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion es
// el unico camino publico del testimonio nominal. Resuelve la identidad una
// sola vez y usa copias internas del mismo lote inmutable para productor,
// verificador independiente y raiz. El consumidor recibe solo una vista
// completa y efimera; no existe producto persistible ni puente reutilizable.
//
// La clave cliente tiene propietario compartido y queda revocada al salir,
// tambien ante error o panico. Como todas las dependencias siguen siendo
// argumentos del llamador, un retorno nil NO concede autoridad, CAS ni efectos.
// Se exige diversidad de tipo concreto entre productor, verificador y raiz;
// es solo una barrera nominal y no acredita procesos, operadores ni claves
// fisicas independientes. Esa acreditacion de DEC-048/049 sigue pendiente.
// El flujo permanece NO-GO hasta que el servicio privado de aplicacion fije la
// composicion, verifique historicamente motivo/material y persista de forma
// atomica.
func ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
	ctx context.Context,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	fronteraIdentidad FronteraIdentidadInternaEstableIdempotenciaBaremacion,
	productor ProductorTestimonioAtomicoIdempotenciaBaremacion,
	verificador VerificadorIndependienteTestimonioIdempotenciaBaremacion,
	raiz VerificadorIndependienteTestimonioIdempotenciaBaremacion,
	consumidor ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	reclamacion := solicitud.claveCliente.reclamarUsoExclusivo()
	if reclamacion == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	solicitud.reclamacion = reclamacion
	defer solicitud.claveCliente.finalizarUsoYDestruir(reclamacion)
	if dependenciaNulaBaremacion(ctx) || dependenciaNulaBaremacion(fronteraIdentidad) ||
		dependenciaNulaBaremacion(productor) || dependenciaNulaBaremacion(verificador) ||
		dependenciaNulaBaremacion(raiz) || dependenciaNulaBaremacion(consumidor) ||
		solicitud.validar() != nil || mismaDependenciaConcretaBaremacion(productor, verificador) ||
		mismaDependenciaConcretaBaremacion(productor, raiz) ||
		mismaDependenciaConcretaBaremacion(verificador, raiz) ||
		mismaDependenciaConcretaBaremacion(raiz, consumidor) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	loteIdentidad, err := resolverIdentidadInternaEstableBaremacion(ctx, solicitud, fronteraIdentidad)
	if err != nil {
		return err
	}
	defer loteIdentidad.destruir()
	receptor := nuevoReceptorEfimeroTestimonioIdempotenciaBaremacion(
		solicitud, loteIdentidad.instantanea,
	)
	defer receptor.destruir()
	fuenteIdentidad := nuevaFuenteEfimeraIdentidadInternaEstableBaremacion(loteIdentidad.ancla)
	defer fuenteIdentidad.destruir()
	clave := solicitud.claveCliente.materialParaLote(solicitud.reclamacion)
	defer borrarBytesBaremacion(clave)
	fuente := nuevaFuenteEfimeraClaveClienteIdempotenciaBaremacion(clave)
	defer fuente.destruir()
	errProduccion := productor.ProducirTestimonioAtomicoIdempotenciaBaremacion(
		ctx, solicitud, fuenteIdentidad, fuente, receptor,
	)
	if err := ctx.Err(); err != nil {
		return err
	}
	identidadProduccionCompleta := fuenteIdentidad.cerrarYComprobarCompletada()
	claveProduccionCompleta := fuente.cerrarYComprobarCompletada()
	if errProduccion != nil || !identidadProduccionCompleta || !claveProduccionCompleta {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	testimonio, err := receptor.finalizar()
	if err != nil {
		return err
	}
	defer destruirTestimonioAtomicoIdempotenciaBaremacion(&testimonio)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := verificarTestimonioNominalConLoteIdentidadBaremacion(
		ctx, solicitud, loteIdentidad.ancla, testimonio, verificador,
	); err != nil {
		if errContexto := ctx.Err(); errContexto != nil {
			return errContexto
		}
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	if err := verificarTestimonioNominalConLoteIdentidadBaremacion(
		ctx, solicitud, loteIdentidad.ancla, testimonio, raiz,
	); err != nil {
		if errContexto := ctx.Err(); errContexto != nil {
			return errContexto
		}
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return consumirTestimonioNominalCompletoBaremacion(ctx, solicitud, testimonio, consumidor)
}

func verificarTestimonioNominalConLoteIdentidadBaremacion(
	ctx context.Context,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	ancla []byte,
	testimonio testimonioAtomicoIdempotenciaBaremacion,
	verificador VerificadorIndependienteTestimonioIdempotenciaBaremacion,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	clon, err := testimonio.clonar()
	if err != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	vista := nuevaVistaEfimeraTestimonioIdempotenciaBaremacion(clon)
	fuenteIdentidad := nuevaFuenteEfimeraIdentidadInternaEstableBaremacion(ancla)
	defer fuenteIdentidad.destruir()
	clave := solicitud.claveCliente.materialParaLote(solicitud.reclamacion)
	defer borrarBytesBaremacion(clave)
	if len(clave) == 0 {
		vista.cerrarYComprobarSinActividad()
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	fuenteClave := nuevaFuenteEfimeraClaveClienteIdempotenciaBaremacion(clave)
	defer fuenteClave.destruir()
	vistaCerradaSinActividad := false
	errVerificacion := func() error {
		defer func() { vistaCerradaSinActividad = vista.cerrarYComprobarSinActividad() }()
		return verificador.VerificarTestimonioAtomicoIdempotenciaBaremacion(
			ctx, solicitud, fuenteIdentidad, fuenteClave, vista,
		)
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	identidadCompleta := fuenteIdentidad.cerrarYComprobarCompletada()
	claveCompleta := fuenteClave.cerrarYComprobarCompletada()
	if errVerificacion != nil || !vistaCerradaSinActividad || !identidadCompleta || !claveCompleta {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return nil
}

func consumirTestimonioNominalCompletoBaremacion(
	ctx context.Context,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	testimonio testimonioAtomicoIdempotenciaBaremacion,
	consumidor ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	clon, err := testimonio.clonar()
	if err != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	vistaConsumidor := nuevaVistaEfimeraTestimonioIdempotenciaBaremacion(clon)
	vistaConsumidorCerradaSinActividad := false
	errConsumo := func() error {
		defer func() { vistaConsumidorCerradaSinActividad = vistaConsumidor.cerrarYComprobarSinActividad() }()
		return consumidor.ConsumirProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion(
			ctx, solicitud, vistaConsumidor,
		)
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	if errConsumo != nil || !vistaConsumidorCerradaSinActividad {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return nil
}

// SeudonimoSujetoBaremacionHMAC sustituye cualquier DNI, nombre o referencia
// libre en la intencion persistible. Su clave tiene dominio exclusivo y debe
// conservarse en el llavero mientras existan operaciones recuperables.
type SeudonimoSujetoBaremacionHMAC struct {
	Version      uint16
	ClaveHMACRef string
	ValorHMAC    string
}

func (s SeudonimoSujetoBaremacionHMAC) Validar() error {
	if s.Version != VersionSeudonimoSujetoBaremacionV1 ||
		!claveCatalogoConfiguracionBaremacionValida(s.ClaveHMACRef) || !huellaSHA256Valida(s.ValorHMAC) {
		return ErrSeudonimoSujetoBaremacionInvalido
	}
	return nil
}

func (s SeudonimoSujetoBaremacionHMAC) Clonar() (SeudonimoSujetoBaremacionHMAC, error) {
	if err := s.Validar(); err != nil {
		return SeudonimoSujetoBaremacionHMAC{}, err
	}
	return s, nil
}

func (s SeudonimoSujetoBaremacionHMAC) IgualConstante(otro SeudonimoSujetoBaremacionHMAC) bool {
	if s.Validar() != nil || otro.Validar() != nil {
		return false
	}
	return hmacVersionadoIgualConstante(
		s.Version, s.ClaveHMACRef, s.ValorHMAC,
		otro.Version, otro.ClaveHMACRef, otro.ValorHMAC,
	)
}

func (SeudonimoSujetoBaremacionHMAC) String() string {
	return "[SEUDONIMO-SUJETO-BAREMACION-PROTEGIDO]"
}
func (SeudonimoSujetoBaremacionHMAC) GoString() string {
	return "ports.SeudonimoSujetoBaremacionHMAC{[PROTEGIDO]}"
}
func (s SeudonimoSujetoBaremacionHMAC) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (SeudonimoSujetoBaremacionHMAC) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (SeudonimoSujetoBaremacionHMAC) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (SeudonimoSujetoBaremacionHMAC) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (s SeudonimoSujetoBaremacionHMAC) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// HMACIntencionCambioBaremacion es el sello persistible del fingerprint
// semantico estable, no del sobre probatorio rotatorio. Solo puede calcularse
// despues de verificar el MotivoHMAC historico contra el motivo efimero. Los
// sobres exactos de seudonimo, manifiesto y motivo se conservan en la auditoria
// del intento, pero nunca deciden igualdad semantica.
type HMACIntencionCambioBaremacion struct {
	Version      uint16
	ClaveHMACRef string
	ValorHMAC    string
}

func (h HMACIntencionCambioBaremacion) Validar() error {
	if h.Version != VersionHMACIntencionCambioBaremacionV1 ||
		!claveCatalogoConfiguracionBaremacionValida(h.ClaveHMACRef) || !huellaSHA256Valida(h.ValorHMAC) {
		return ErrHMACIntencionCambioBaremacionInvalido
	}
	return nil
}

func (h HMACIntencionCambioBaremacion) Clonar() (HMACIntencionCambioBaremacion, error) {
	if err := h.Validar(); err != nil {
		return HMACIntencionCambioBaremacion{}, err
	}
	return h, nil
}

func (h HMACIntencionCambioBaremacion) IgualConstante(otro HMACIntencionCambioBaremacion) bool {
	if h.Validar() != nil || otro.Validar() != nil {
		return false
	}
	return hmacVersionadoIgualConstante(
		h.Version, h.ClaveHMACRef, h.ValorHMAC,
		otro.Version, otro.ClaveHMACRef, otro.ValorHMAC,
	)
}

func (HMACIntencionCambioBaremacion) String() string {
	return "[HMAC-INTENCION-CAMBIO-BAREMACION-PROTEGIDO]"
}
func (HMACIntencionCambioBaremacion) GoString() string {
	return "ports.HMACIntencionCambioBaremacion{[PROTEGIDO]}"
}
func (h HMACIntencionCambioBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, h.String())
}
func (HMACIntencionCambioBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (HMACIntencionCambioBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (HMACIntencionCambioBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (h HMACIntencionCambioBaremacion) LogValue() slog.Value {
	return slog.StringValue(h.String())
}

// HMACMotivoBaremacion compromete el motivo exacto sin persistirlo ni exponer
// una huella SHA-256 susceptible a ataques de diccionario. Su clave debe ser
// exclusiva de este dominio y mantenerse en el llavero historico.
type HMACMotivoBaremacion struct {
	Version      uint16
	ClaveHMACRef string
	ValorHMAC    string
}

func (h HMACMotivoBaremacion) Validar() error {
	if h.Version != VersionHMACMotivoBaremacionV1 ||
		!claveCatalogoConfiguracionBaremacionValida(h.ClaveHMACRef) || !huellaSHA256Valida(h.ValorHMAC) {
		return ErrHMACMotivoBaremacionInvalido
	}
	return nil
}

func (h HMACMotivoBaremacion) Clonar() (HMACMotivoBaremacion, error) {
	if err := h.Validar(); err != nil {
		return HMACMotivoBaremacion{}, err
	}
	return h, nil
}

func (h HMACMotivoBaremacion) IgualConstante(otro HMACMotivoBaremacion) bool {
	if h.Validar() != nil || otro.Validar() != nil {
		return false
	}
	return hmacVersionadoIgualConstante(
		h.Version, h.ClaveHMACRef, h.ValorHMAC,
		otro.Version, otro.ClaveHMACRef, otro.ValorHMAC,
	)
}

func (HMACMotivoBaremacion) String() string {
	return "[HMAC-MOTIVO-BAREMACION-PROTEGIDO]"
}
func (HMACMotivoBaremacion) GoString() string {
	return "ports.HMACMotivoBaremacion{[PROTEGIDO]}"
}
func (h HMACMotivoBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, h.String())
}
func (HMACMotivoBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (HMACMotivoBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (HMACMotivoBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (h HMACMotivoBaremacion) LogValue() slog.Value {
	return slog.StringValue(h.String())
}

// HMACManifiestoMaterialBaremacionV2 sustituye el sobre textual generico. La
// forma del valor no acredita autenticidad. Solo la composicion TCB homologada
// puede confiar en la aceptacion conjunta del verificador de dominios y del
// verificador material V2 que mantiene como dependencias privadas.
type HMACManifiestoMaterialBaremacionV2 struct {
	Version      uint16
	ClaveHMACRef string
	ValorHMAC    string
}

func (h HMACManifiestoMaterialBaremacionV2) Validar() error {
	if h.Version != VersionHMACManifiestoMaterialBaremacionV2 ||
		!claveCatalogoConfiguracionBaremacionValida(h.ClaveHMACRef) || !huellaSHA256Valida(h.ValorHMAC) {
		return ErrHMACManifiestoMaterialBaremacionInvalido
	}
	return nil
}

func (h HMACManifiestoMaterialBaremacionV2) IgualConstante(
	otro HMACManifiestoMaterialBaremacionV2,
) bool {
	if h.Validar() != nil || otro.Validar() != nil {
		return false
	}
	return hmacVersionadoIgualConstante(
		h.Version, h.ClaveHMACRef, h.ValorHMAC,
		otro.Version, otro.ClaveHMACRef, otro.ValorHMAC,
	)
}

func (HMACManifiestoMaterialBaremacionV2) String() string {
	return "[HMAC-MANIFIESTO-MATERIAL-BAREMACION-V2-PROTEGIDO]"
}
func (HMACManifiestoMaterialBaremacionV2) GoString() string {
	return "ports.HMACManifiestoMaterialBaremacionV2{[PROTEGIDO]}"
}
func (h HMACManifiestoMaterialBaremacionV2) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, h.String())
}
func (HMACManifiestoMaterialBaremacionV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (HMACManifiestoMaterialBaremacionV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (HMACManifiestoMaterialBaremacionV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (h HMACManifiestoMaterialBaremacionV2) LogValue() slog.Value {
	return slog.StringValue(h.String())
}

// DominioClaveHMACBaremacion es un catalogo cerrado de usos criptograficos.
// Una referencia distinta no demuestra una clave distinta: el verificador
// autoritativo debe resolver cada alias contra HSM/KMS y acreditar claves
// fisicas y politicas de uso separadas.
type DominioClaveHMACBaremacion string

const (
	DominioClavePrincipalBaremacion  DominioClaveHMACBaremacion = "principal"
	DominioClaveIndiceBaremacion     DominioClaveHMACBaremacion = "indice"
	DominioClaveSujetoBaremacion     DominioClaveHMACBaremacion = "sujeto"
	DominioClaveMotivoBaremacion     DominioClaveHMACBaremacion = "motivo"
	DominioClaveManifiestoBaremacion DominioClaveHMACBaremacion = "manifiesto"
	DominioClaveIntencionBaremacion  DominioClaveHMACBaremacion = "intencion"
)

func (d DominioClaveHMACBaremacion) Valido() bool {
	switch d {
	case DominioClavePrincipalBaremacion, DominioClaveIndiceBaremacion,
		DominioClaveSujetoBaremacion, DominioClaveMotivoBaremacion,
		DominioClaveManifiestoBaremacion, DominioClaveIntencionBaremacion:
		return true
	default:
		return false
	}
}

// ReferenciaGeneracionClaveHMACNominalBaremacion es un descriptor opaco para
// solicitar al KMS la comprobacion conjunta. La referencia no acredita por si
// sola una clave fisica distinta ni concede autoridad.
type ReferenciaGeneracionClaveHMACNominalBaremacion struct {
	dominio      DominioClaveHMACBaremacion
	generacion   uint32
	claveHMACRef string
}

func NuevaReferenciaGeneracionClaveHMACNominalBaremacion(
	dominio DominioClaveHMACBaremacion,
	generacion uint32,
	claveHMACRef string,
) (ReferenciaGeneracionClaveHMACNominalBaremacion, error) {
	referencia := ReferenciaGeneracionClaveHMACNominalBaremacion{
		dominio: dominio, generacion: generacion, claveHMACRef: claveHMACRef,
	}
	if referencia.validar() != nil {
		return ReferenciaGeneracionClaveHMACNominalBaremacion{}, ErrSeparacionDominiosClaveBaremacion
	}
	return referencia, nil
}

func (u ReferenciaGeneracionClaveHMACNominalBaremacion) validar() error {
	if !u.dominio.Valido() || u.generacion == 0 ||
		!claveCatalogoConfiguracionBaremacionValida(u.claveHMACRef) {
		return ErrSeparacionDominiosClaveBaremacion
	}
	return nil
}

func (ReferenciaGeneracionClaveHMACNominalBaremacion) String() string {
	return "[REFERENCIA-GENERACION-CLAVE-HMAC-NOMINAL-BAREMACION-PROTEGIDA]"
}
func (ReferenciaGeneracionClaveHMACNominalBaremacion) GoString() string {
	return "ports.ReferenciaGeneracionClaveHMACNominalBaremacion{[PROTEGIDA]}"
}
func (u ReferenciaGeneracionClaveHMACNominalBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, u.String())
}
func (ReferenciaGeneracionClaveHMACNominalBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (ReferenciaGeneracionClaveHMACNominalBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (ReferenciaGeneracionClaveHMACNominalBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (u ReferenciaGeneracionClaveHMACNominalBaremacion) LogValue() slog.Value {
	return slog.StringValue(u.String())
}

// SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion representa
// los seis dominios completos y todas sus generaciones recuperables (1..8 por
// dominio, hasta 48). El KMS debe resolver cada alias y acreditar separacion de
// clave fisica y politica; esta forma solo elimina omisiones y alias logicos.
type SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion struct {
	referencias []ReferenciaGeneracionClaveHMACNominalBaremacion
}

func ConstruirSolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion(
	referencias []ReferenciaGeneracionClaveHMACNominalBaremacion,
) (SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion, error) {
	solicitud := SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion{
		referencias: append([]ReferenciaGeneracionClaveHMACNominalBaremacion(nil), referencias...),
	}
	sort.Slice(solicitud.referencias, func(izquierda, derecha int) bool {
		a := solicitud.referencias[izquierda]
		b := solicitud.referencias[derecha]
		if ordenDominioClaveHMACBaremacion(a.dominio) != ordenDominioClaveHMACBaremacion(b.dominio) {
			return ordenDominioClaveHMACBaremacion(a.dominio) < ordenDominioClaveHMACBaremacion(b.dominio)
		}
		if a.generacion != b.generacion {
			return a.generacion > b.generacion
		}
		return a.claveHMACRef < b.claveHMACRef
	})
	if err := solicitud.validar(); err != nil {
		return SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion{}, err
	}
	return solicitud, nil
}

func ordenDominioClaveHMACBaremacion(dominio DominioClaveHMACBaremacion) int {
	switch dominio {
	case DominioClavePrincipalBaremacion:
		return 0
	case DominioClaveIndiceBaremacion:
		return 1
	case DominioClaveSujetoBaremacion:
		return 2
	case DominioClaveMotivoBaremacion:
		return 3
	case DominioClaveManifiestoBaremacion:
		return 4
	case DominioClaveIntencionBaremacion:
		return 5
	default:
		return 6
	}
}

func (s SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) validar() error {
	if len(s.referencias) < 6 || len(s.referencias) > 48 {
		return ErrSeparacionDominiosClaveBaremacion
	}
	cantidades := make(map[DominioClaveHMACBaremacion]int, 6)
	generaciones := make(map[DominioClaveHMACBaremacion]map[uint32]struct{}, 6)
	alias := make(map[string]struct{}, len(s.referencias))
	ordenAnterior := -1
	var generacionAnterior uint32
	for posicion, referencia := range s.referencias {
		if referencia.validar() != nil {
			return ErrSeparacionDominiosClaveBaremacion
		}
		orden := ordenDominioClaveHMACBaremacion(referencia.dominio)
		if orden < ordenAnterior || (posicion > 0 && orden == ordenAnterior && referencia.generacion >= generacionAnterior) {
			return ErrSeparacionDominiosClaveBaremacion
		}
		if _, repetida := alias[referencia.claveHMACRef]; repetida {
			return ErrSeparacionDominiosClaveBaremacion
		}
		if generaciones[referencia.dominio] == nil {
			generaciones[referencia.dominio] = make(map[uint32]struct{})
		}
		if _, repetida := generaciones[referencia.dominio][referencia.generacion]; repetida {
			return ErrSeparacionDominiosClaveBaremacion
		}
		generaciones[referencia.dominio][referencia.generacion] = struct{}{}
		alias[referencia.claveHMACRef] = struct{}{}
		cantidades[referencia.dominio]++
		if cantidades[referencia.dominio] > 8 {
			return ErrSeparacionDominiosClaveBaremacion
		}
		ordenAnterior = orden
		generacionAnterior = referencia.generacion
	}
	for _, dominio := range []DominioClaveHMACBaremacion{
		DominioClavePrincipalBaremacion, DominioClaveIndiceBaremacion,
		DominioClaveSujetoBaremacion, DominioClaveMotivoBaremacion,
		DominioClaveManifiestoBaremacion, DominioClaveIntencionBaremacion,
	} {
		if cantidades[dominio] < 1 || cantidades[dominio] > 8 {
			return ErrSeparacionDominiosClaveBaremacion
		}
	}
	return nil
}

// VisitarReferencias entrega el lote canonico solo durante la llamada.
func (s SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) VisitarReferencias(
	visita func(DominioClaveHMACBaremacion, uint32, string) error,
) error {
	if visita == nil || s.validar() != nil {
		return ErrSeparacionDominiosClaveBaremacion
	}
	for _, referencia := range s.referencias {
		if err := visita(referencia.dominio, referencia.generacion, referencia.claveHMACRef); err != nil {
			return err
		}
	}
	return nil
}

func (SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) String() string {
	return "[SOLICITUD-NOMINAL-NO-AUTORITATIVA-SEPARACION-DOMINIOS-CLAVE-BAREMACION-PROTEGIDA]"
}
func (SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) GoString() string {
	return "ports.SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion{[PROTEGIDA]}"
}
func (s SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (s SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

type VerificadorSeparacionDominiosClaveBaremacion interface {
	VerificarSeparacionDominiosClaveBaremacion(
		context.Context,
		SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion,
	) error
}

// ClaveClienteIdempotenciaBaremacion admite solo UUIDv4 canonico lowercase o
// base64url sin relleno que decodifique 32..64 bytes no textuales y con
// diversidad minima. La forma reduce DNI/NIE, correo, rutas y texto humano;
// la entropia real debe generarla el cliente con CSPRNG.
type ClaveClienteIdempotenciaBaremacion struct {
	propietario *propietarioClaveClienteIdempotenciaBaremacion
}

type propietarioClaveClienteIdempotenciaBaremacion struct {
	mu          sync.Mutex
	formato     formatoClaveClienteIdempotenciaBaremacion
	valor       []byte
	destruido   bool
	reclamacion *reclamacionClaveClienteIdempotenciaBaremacion
}

type reclamacionClaveClienteIdempotenciaBaremacion struct{ marcador byte }

type formatoClaveClienteIdempotenciaBaremacion uint8

const (
	formatoClaveClienteUUIDv4 formatoClaveClienteIdempotenciaBaremacion = iota + 1
	formatoClaveClienteBase64URL
)

func NuevaClaveClienteIdempotenciaBaremacion(valor string) (ClaveClienteIdempotenciaBaremacion, error) {
	if bytesUUID, valida := decodificarUUIDv4CanonicoBaremacion(valor); valida {
		return ClaveClienteIdempotenciaBaremacion{
			propietario: &propietarioClaveClienteIdempotenciaBaremacion{
				formato: formatoClaveClienteUUIDv4, valor: bytesUUID,
			},
		}, nil
	}
	decodificado, err := base64.RawURLEncoding.DecodeString(valor)
	defer borrarBytesBaremacion(decodificado)
	if err != nil || base64.RawURLEncoding.EncodeToString(decodificado) != valor ||
		!materialBase64URLClienteBaremacionValido(decodificado) {
		return ClaveClienteIdempotenciaBaremacion{}, ErrClaveIdempotenciaBaremacionInvalida
	}
	return ClaveClienteIdempotenciaBaremacion{
		propietario: &propietarioClaveClienteIdempotenciaBaremacion{
			formato: formatoClaveClienteBase64URL,
			valor:   append([]byte(nil), decodificado...),
		},
	}, nil
}

func (c ClaveClienteIdempotenciaBaremacion) validar() error {
	return c.validarConReclamacion(nil)
}

func (c ClaveClienteIdempotenciaBaremacion) validarConReclamacion(
	reclamacion *reclamacionClaveClienteIdempotenciaBaremacion,
) error {
	if c.propietario == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	c.propietario.mu.Lock()
	defer c.propietario.mu.Unlock()
	if c.propietario.destruido ||
		(c.propietario.reclamacion != nil && c.propietario.reclamacion != reclamacion) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return validarMaterialClaveClienteIdempotenciaBaremacion(
		c.propietario.formato, c.propietario.valor,
	)
}

func validarMaterialClaveClienteIdempotenciaBaremacion(
	formato formatoClaveClienteIdempotenciaBaremacion,
	valor []byte,
) error {
	switch formato {
	case formatoClaveClienteUUIDv4:
		if len(valor) != 16 || valor[6]>>4 != 4 || valor[8]>>6 != 2 {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
	case formatoClaveClienteBase64URL:
		if !materialBase64URLClienteBaremacionValido(valor) {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
	default:
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return nil
}

func (c ClaveClienteIdempotenciaBaremacion) materialParaLote(
	reclamacion *reclamacionClaveClienteIdempotenciaBaremacion,
) []byte {
	_, material := c.formatoYMaterialParaLote(reclamacion)
	return material
}

func (c ClaveClienteIdempotenciaBaremacion) formatoYMaterialParaLote(
	reclamacion *reclamacionClaveClienteIdempotenciaBaremacion,
) (
	formatoClaveClienteIdempotenciaBaremacion,
	[]byte,
) {
	if c.propietario == nil {
		return 0, nil
	}
	c.propietario.mu.Lock()
	defer c.propietario.mu.Unlock()
	if c.propietario.destruido ||
		(c.propietario.reclamacion != nil && c.propietario.reclamacion != reclamacion) ||
		validarMaterialClaveClienteIdempotenciaBaremacion(
			c.propietario.formato, c.propietario.valor,
		) != nil {
		return 0, nil
	}
	return c.propietario.formato, append([]byte(nil), c.propietario.valor...)
}

func (c ClaveClienteIdempotenciaBaremacion) reclamarUsoExclusivo() *reclamacionClaveClienteIdempotenciaBaremacion {
	if c.propietario == nil {
		return nil
	}
	c.propietario.mu.Lock()
	defer c.propietario.mu.Unlock()
	if c.propietario.destruido || c.propietario.reclamacion != nil ||
		validarMaterialClaveClienteIdempotenciaBaremacion(
			c.propietario.formato, c.propietario.valor,
		) != nil {
		return nil
	}
	reclamacion := &reclamacionClaveClienteIdempotenciaBaremacion{marcador: 1}
	c.propietario.reclamacion = reclamacion
	return reclamacion
}

func (c ClaveClienteIdempotenciaBaremacion) finalizarUsoYDestruir(
	reclamacion *reclamacionClaveClienteIdempotenciaBaremacion,
) {
	if c.propietario == nil {
		return
	}
	c.propietario.mu.Lock()
	if reclamacion == nil || c.propietario.reclamacion != reclamacion {
		c.propietario.mu.Unlock()
		return
	}
	c.propietario.destruido = true
	borrarBytesBaremacion(c.propietario.valor)
	c.propietario.valor = nil
	c.propietario.formato = 0
	c.propietario.reclamacion = nil
	c.propietario.mu.Unlock()
}

func (ClaveClienteIdempotenciaBaremacion) String() string {
	return "[CLAVE-CLIENTE-IDEMPOTENCIA-BAREMACION-PROTEGIDA]"
}
func (ClaveClienteIdempotenciaBaremacion) GoString() string {
	return "ports.ClaveClienteIdempotenciaBaremacion{[PROTEGIDA]}"
}
func (c ClaveClienteIdempotenciaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (ClaveClienteIdempotenciaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (ClaveClienteIdempotenciaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (ClaveClienteIdempotenciaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (c ClaveClienteIdempotenciaBaremacion) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

type ModuloIdempotenciaBaremacion string

const ModuloIdempotenciaBolsa ModuloIdempotenciaBaremacion = "bolsa"

func (m ModuloIdempotenciaBaremacion) Valido() bool { return m == ModuloIdempotenciaBolsa }

type ReferenciaDespliegueIdempotenciaBaremacion string

func (r ReferenciaDespliegueIdempotenciaBaremacion) Valida() bool {
	return claveCatalogoConfiguracionBaremacionValida(string(r))
}

// SolicitudTestimonioAtomicoIdempotenciaBaremacion posee campos privados y no
// expone la clave. El productor solo recibe el contexto estable por getters y
// una fuente efimera creada dentro de la fabrica nominal.
type SolicitudTestimonioAtomicoIdempotenciaBaremacion struct {
	despliegueRef ReferenciaDespliegueIdempotenciaBaremacion
	modulo        ModuloIdempotenciaBaremacion
	clase         ClaseCambioBaremacion
	ambitoSujeto  SolicitudResolverSeudonimoSujetoBaremacion
	seudonimo     SeudonimoSujetoBaremacionHMAC
	claveCliente  ClaveClienteIdempotenciaBaremacion
	reclamacion   *reclamacionClaveClienteIdempotenciaBaremacion
	vinculo       [32]byte
}

func NuevaSolicitudTestimonioAtomicoIdempotenciaBaremacion(
	despliegue ReferenciaDespliegueIdempotenciaBaremacion,
	modulo ModuloIdempotenciaBaremacion,
	clase ClaseCambioBaremacion,
	ambitoSujeto SolicitudResolverSeudonimoSujetoBaremacion,
	seudonimo SeudonimoSujetoBaremacionHMAC,
	clave ClaveClienteIdempotenciaBaremacion,
) (SolicitudTestimonioAtomicoIdempotenciaBaremacion, error) {
	if !despliegue.Valida() || !modulo.Valido() || clase != ClaseCambioIncorporarDecision ||
		ambitoSujeto.validar() != nil || seudonimo.Validar() != nil || clave.validar() != nil {
		return SolicitudTestimonioAtomicoIdempotenciaBaremacion{}, ErrClaveIdempotenciaBaremacionInvalida
	}
	solicitud := SolicitudTestimonioAtomicoIdempotenciaBaremacion{
		despliegueRef: despliegue, modulo: modulo, clase: clase,
		ambitoSujeto: ambitoSujeto,
		seudonimo:    seudonimo,
		claveCliente: clave,
	}
	solicitud.vinculo = vinculoSolicitudTestimonioIdempotenciaBaremacion(solicitud)
	if err := solicitud.validar(); err != nil {
		return SolicitudTestimonioAtomicoIdempotenciaBaremacion{}, err
	}
	return solicitud, nil
}

func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) validar() error {
	if !s.despliegueRef.Valida() || !s.modulo.Valido() ||
		s.clase != ClaseCambioIncorporarDecision || s.ambitoSujeto.validar() != nil ||
		s.seudonimo.Validar() != nil || s.claveCliente.validarConReclamacion(s.reclamacion) != nil ||
		s.vinculo == ([32]byte{}) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	esperado := vinculoSolicitudTestimonioIdempotenciaBaremacion(s)
	if subtle.ConstantTimeCompare(esperado[:], s.vinculo[:]) != 1 {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return nil
}

func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) DespliegueRef() ReferenciaDespliegueIdempotenciaBaremacion {
	return s.despliegueRef
}
func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) Modulo() ModuloIdempotenciaBaremacion {
	return s.modulo
}
func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) Clase() ClaseCambioBaremacion {
	return s.clase
}

// VisitarAmbitoSujetoBaremacion entrega el expediente opaco y el seudonimo ya
// resuelto exclusivamente para ligar solicitud y atestacion. El seudonimo es
// rotatorio y, por tanto, nunca forma la preimagen del principal estable. No
// expone DNI, nombre ni un getter reutilizable. En este puerto nominal la forma
// no acredita origen: el futuro servicio privado debe resolver y verificar el
// seudonimo con FronteraIdentidadesEstablesBaremacion antes de construirla.
func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) VisitarAmbitoSujetoBaremacion(
	visita func(
		procesoRef, solicitudRef, baremacionMeritoRef string,
		versionSeudonimo uint16, claveSeudonimoRef, valorSeudonimoHMAC string,
	) error,
) error {
	if visita == nil || s.validar() != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return visita(
		s.ambitoSujeto.procesoRef, s.ambitoSujeto.solicitudRef,
		s.ambitoSujeto.baremacionMeritoRef, s.seudonimo.Version,
		s.seudonimo.ClaveHMACRef, s.seudonimo.ValorHMAC,
	)
}

// VisitarMaterialCanonicoParaDerivarSeudonimoSujetoBaremacion fija la
// formula seudonimo = HMAC(clave_sujeto_actual,
// esquema+politica+version+identidad_interna_estable). Seudonimo y principal
// parten de la misma identidad efimera pero de dominios de clave independientes;
// rotar la clave de sujeto cambia solo el seudonimo. El propietario interno se
// borra al volver del callback, tambien ante error o panico; el adaptador no
// recibe un getter ni una carga reutilizable.
func VisitarMaterialCanonicoParaDerivarSeudonimoSujetoBaremacion(
	s SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	identidadInternaEstableEfimera []byte,
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) error {
	if visita == nil || s.validar() != nil ||
		len(identidadInternaEstableEfimera) != longitudIdentidadInternaEstableBaremacion {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	material := make([]byte, 0, 192)
	defer func() { borrarBytesBaremacion(material) }()
	material = anexarCampoCanonicoTextoIntencion(material, 0, EsquemaCanonicoSeudonimoSujetoBaremacionV1)
	material = anexarCampoCanonicoTextoIntencion(material, 1, PoliticaDerivacionIdempotenciaBaremacionDEC045V1)
	material = anexarCampoCanonicoIntencion(material, 2, canonicoUint16Intencion(s.seudonimo.Version))
	material = anexarCampoCanonicoIntencion(material, 3, identidadInternaEstableEfimera)
	carga, err := NuevaCargaProtegida(material)
	if err != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return visitarCargaProtegidaEfimeraBaremacion(carga, visita)
}

// VisitarMaterialCanonicoParaDerivarPrincipalEstableBaremacion fija la
// formula principal = HMAC(llavero_principal[g],
// esquema+politica+identidad_interna_estable). La identidad es un identificador
// binario opaco de 32 bytes que la frontera privada entrega solo durante la
// llamada: nunca es DNI, seudonimo rotatorio, referencia libre, DTO, contexto,
// log ni dato persistible. La clave cliente tampoco forma esta preimagen.
//
// El material visitado sigue siendo sensible. Su propietario interno se borra
// al terminar el callback. El futuro aislamiento reforzado de DEC-047 debe
// ejecutar este limite en otro proceso si un adaptador del mismo proceso se
// considera malicioso.
func VisitarMaterialCanonicoParaDerivarPrincipalEstableBaremacion(
	s SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	identidadInternaEstableEfimera []byte,
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) error {
	if visita == nil || s.validar() != nil ||
		len(identidadInternaEstableEfimera) != longitudIdentidadInternaEstableBaremacion {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	material := make([]byte, 0, 256)
	defer func() { borrarBytesBaremacion(material) }()
	material = anexarCampoCanonicoTextoIntencion(material, 0, EsquemaCanonicoPrincipalEstableBaremacionV1)
	material = anexarCampoCanonicoTextoIntencion(material, 1, PoliticaDerivacionIdempotenciaBaremacionDEC045V1)
	material = anexarCampoCanonicoIntencion(material, 2, identidadInternaEstableEfimera)
	carga, err := NuevaCargaProtegida(material)
	if err != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return visitarCargaProtegidaEfimeraBaremacion(carga, visita)
}

// VisitarMaterialCanonicoParaDerivarIndiceIdempotenciaBaremacion fija
// indice = HMAC(llavero_indice[g], esquema+politica+despliegue+modulo+accion+
// principal_estable+clave_cliente). La clave solo puede proceder de la fuente
// efimera del lote y debe coincidir con la solicitud; nunca se obtiene por un
// getter publico. La preimagen propietaria se borra al volver del callback.
func VisitarMaterialCanonicoParaDerivarIndiceIdempotenciaBaremacion(
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	versionPrincipal uint16,
	generacionPrincipal uint32,
	clavePrincipalRef, valorPrincipalHMAC string,
	claveClienteEfimera []byte,
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) error {
	principal := principalEstableBaremacionHMAC{
		Version: versionPrincipal, GeneracionClave: generacionPrincipal,
		ClaveHMACRef: clavePrincipalRef, ValorHMAC: valorPrincipalHMAC,
	}
	claveEsperada := solicitud.claveCliente.materialParaLote(solicitud.reclamacion)
	defer borrarBytesBaremacion(claveEsperada)
	if visita == nil || solicitud.validar() != nil || principal.validar() != nil ||
		len(claveClienteEfimera) != len(claveEsperada) ||
		subtle.ConstantTimeCompare(claveClienteEfimera, claveEsperada) != 1 {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	material := make([]byte, 0, 512)
	defer func() { borrarBytesBaremacion(material) }()
	material = anexarCampoCanonicoTextoIntencion(material, 0, EsquemaCanonicoIndiceIdempotenciaBaremacionV1)
	material = anexarCampoCanonicoTextoIntencion(material, 1, PoliticaDerivacionIdempotenciaBaremacionDEC045V1)
	material = anexarCampoCanonicoTextoIntencion(material, 2, string(solicitud.despliegueRef))
	material = anexarCampoCanonicoTextoIntencion(material, 3, string(solicitud.modulo))
	material = anexarCampoCanonicoTextoIntencion(material, 4, string(solicitud.clase))
	material = anexarCampoCanonicoIntencion(material, 5, canonicoUint16Intencion(principal.Version))
	material = anexarCampoCanonicoIntencion(material, 6, canonicoUint32Intencion(principal.GeneracionClave))
	material = anexarCampoCanonicoTextoIntencion(material, 7, principal.ClaveHMACRef)
	valorPrincipal := decodificarHexCanonicoIntencion(principal.ValorHMAC)
	defer borrarBytesBaremacion(valorPrincipal)
	material = anexarCampoCanonicoIntencion(material, 8, valorPrincipal)
	material = anexarCampoCanonicoIntencion(material, 9, claveClienteEfimera)
	carga, err := NuevaCargaProtegida(material)
	if err != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return visitarCargaProtegidaEfimeraBaremacion(carga, visita)
}
func (SolicitudTestimonioAtomicoIdempotenciaBaremacion) String() string {
	return "[SOLICITUD-TESTIMONIO-ATOMICO-IDEMPOTENCIA-BAREMACION-PROTEGIDA]"
}
func (SolicitudTestimonioAtomicoIdempotenciaBaremacion) GoString() string {
	return "ports.SolicitudTestimonioAtomicoIdempotenciaBaremacion{[PROTEGIDA]}"
}
func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (SolicitudTestimonioAtomicoIdempotenciaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (SolicitudTestimonioAtomicoIdempotenciaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (SolicitudTestimonioAtomicoIdempotenciaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (s SolicitudTestimonioAtomicoIdempotenciaBaremacion) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// ConsumidorIdentidadInternaEstableIdempotenciaBaremacion solo puede recibir
// el identificador interno opaco dentro de un callback sincrono. El valor no
// debe copiarse a DTO, contexto, registro ni persistencia.
type ConsumidorIdentidadInternaEstableIdempotenciaBaremacion interface {
	ConsumirIdentidadInternaEstableIdempotenciaBaremacion(context.Context, []byte) error
}

// FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion la crea la
// frontera privada de identidad/HSM despues de resolver el sujeto desde el
// expediente y cotejar su seudonimo actual. Entrega exactamente una identidad
// binaria estable de 32 bytes. La fabrica nominal crea entregas internas
// separadas y de un solo uso para productor, verificador y raiz, y borra el
// valor. El futuro servicio autoritativo debe fijar la implementacion como
// dependencia privada; una fuente elegida por el llamador nunca concede
// autoridad.
type FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion interface {
	EntregarIdentidadInternaEstableIdempotenciaBaremacion(
		context.Context,
		ConsumidorIdentidadInternaEstableIdempotenciaBaremacion,
	) error
}

// ReceptorEfimeroResolucionIdentidadInternaEstableBaremacion solo vive durante
// una llamada a la frontera. La instantanea identifica la version inmutable de
// la relacion expediente+seudonimo esperado -> ancla interna; la atestacion
// permite que la raiz independiente compruebe esa resolucion sin ver el ancla.
type ReceptorEfimeroResolucionIdentidadInternaEstableBaremacion interface {
	RegistrarResolucionIdentidadInternaEstableBaremacion(
		context.Context,
		[]byte,
		string,
		uint64,
		string,
		string,
		string,
		string,
		[]byte,
	) error
}

// FronteraIdentidadInternaEstableIdempotenciaBaremacion debe resolver de forma
// atomica las referencias opacas y el seudonimo esperado exacto. Una
// implementacion no puede aceptar referencias de A junto al seudonimo de B.
// Debe entregar una unica instantanea y una ancla CSPRNG inmutable de 256 bits.
// Al seguir siendo inyectable aqui, esta frontera solo produce un testimonio
// nominal; el servicio de aplicacion pendiente debe fijarla en privado.
type FronteraIdentidadInternaEstableIdempotenciaBaremacion interface {
	ResolverYEntregarIdentidadInternaEstableIdempotenciaBaremacion(
		context.Context,
		SolicitudResolverSeudonimoSujetoBaremacion,
		SeudonimoSujetoBaremacionHMAC,
		ReceptorEfimeroResolucionIdentidadInternaEstableBaremacion,
	) error
}

type instantaneaResolucionIdentidadInternaEstableBaremacion struct {
	SnapshotRef         string
	Revision            uint64
	HuellaSHA256        string
	FormatoAtestacion   string
	EmisorAtestacionRef string
	ClaveAtestacionRef  string
	ValorAtestacion     []byte
}

func (i instantaneaResolucionIdentidadInternaEstableBaremacion) validar() error {
	if !claveCatalogoConfiguracionBaremacionValida(i.SnapshotRef) || i.Revision == 0 ||
		!huellaSHA256Valida(i.HuellaSHA256) ||
		!claveCatalogoConfiguracionBaremacionValida(i.FormatoAtestacion) ||
		!claveCatalogoConfiguracionBaremacionValida(i.EmisorAtestacionRef) ||
		!claveCatalogoConfiguracionBaremacionValida(i.ClaveAtestacionRef) ||
		len(i.ValorAtestacion) < 16 || len(i.ValorAtestacion) > 8192 {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return nil
}

func (i instantaneaResolucionIdentidadInternaEstableBaremacion) representacionCanonica() []byte {
	material := make([]byte, 0, 512+len(i.ValorAtestacion))
	material = anexarCampoCanonicoTextoIntencion(material, 0, esquemaResolucionIdentidadInternaBaremacionV1)
	material = anexarCampoCanonicoTextoIntencion(material, 1, i.SnapshotRef)
	material = anexarCampoCanonicoIntencion(material, 2, canonicoUint64Intencion(i.Revision))
	material = anexarCampoCanonicoTextoIntencion(material, 3, i.HuellaSHA256)
	material = anexarCampoCanonicoTextoIntencion(material, 4, i.FormatoAtestacion)
	material = anexarCampoCanonicoTextoIntencion(material, 5, i.EmisorAtestacionRef)
	material = anexarCampoCanonicoTextoIntencion(material, 6, i.ClaveAtestacionRef)
	material = anexarCampoCanonicoIntencion(material, 7, i.ValorAtestacion)
	return material
}

func (i instantaneaResolucionIdentidadInternaEstableBaremacion) clonar() instantaneaResolucionIdentidadInternaEstableBaremacion {
	clon := i
	clon.ValorAtestacion = append([]byte(nil), i.ValorAtestacion...)
	return clon
}

func (i instantaneaResolucionIdentidadInternaEstableBaremacion) igual(
	otra instantaneaResolucionIdentidadInternaEstableBaremacion,
) bool {
	if i.validar() != nil || otra.validar() != nil {
		return false
	}
	izquierda := i.representacionCanonica()
	derecha := otra.representacionCanonica()
	defer borrarBytesBaremacion(izquierda)
	defer borrarBytesBaremacion(derecha)
	return len(izquierda) == len(derecha) && subtle.ConstantTimeCompare(izquierda, derecha) == 1
}

func (instantaneaResolucionIdentidadInternaEstableBaremacion) String() string {
	return "[INSTANTANEA-RESOLUCION-IDENTIDAD-INTERNA-ESTABLE-BAREMACION-PROTEGIDA]"
}
func (instantaneaResolucionIdentidadInternaEstableBaremacion) GoString() string {
	return "ports.instantaneaResolucionIdentidadInternaEstableBaremacion{[PROTEGIDA]}"
}
func (i instantaneaResolucionIdentidadInternaEstableBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, i.String())
}
func (instantaneaResolucionIdentidadInternaEstableBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (instantaneaResolucionIdentidadInternaEstableBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (instantaneaResolucionIdentidadInternaEstableBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (i instantaneaResolucionIdentidadInternaEstableBaremacion) LogValue() slog.Value {
	return slog.StringValue(i.String())
}

// VisitarMaterialCanonicoAtestacionResolucionIdentidadInternaEstableBaremacion
// liga la evidencia de la frontera al ambito, seudonimo esperado y snapshot.
// El hash del snapshot compromete la relacion interna; el ancla clara nunca
// forma parte del testimonio ni de este material compartible. La carga
// propietaria se borra al volver de la visita.
func VisitarMaterialCanonicoAtestacionResolucionIdentidadInternaEstableBaremacion(
	ambito SolicitudResolverSeudonimoSujetoBaremacion,
	seudonimo SeudonimoSujetoBaremacionHMAC,
	snapshotRef string,
	revision uint64,
	huellaSHA256 string,
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) error {
	if visita == nil || ambito.validar() != nil || seudonimo.Validar() != nil ||
		!claveCatalogoConfiguracionBaremacionValida(snapshotRef) || revision == 0 ||
		!huellaSHA256Valida(huellaSHA256) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	material := make([]byte, 0, 512)
	defer func() { borrarBytesBaremacion(material) }()
	material = anexarCampoCanonicoTextoIntencion(material, 0, esquemaResolucionIdentidadInternaBaremacionV1)
	material = anexarCampoCanonicoTextoIntencion(material, 1, ambito.procesoRef)
	material = anexarCampoCanonicoTextoIntencion(material, 2, ambito.solicitudRef)
	material = anexarCampoCanonicoTextoIntencion(material, 3, ambito.baremacionMeritoRef)
	material = anexarCampoCanonicoIntencion(material, 4, canonicoUint16Intencion(seudonimo.Version))
	material = anexarCampoCanonicoTextoIntencion(material, 5, seudonimo.ClaveHMACRef)
	valorSeudonimo := decodificarHexCanonicoIntencion(seudonimo.ValorHMAC)
	defer borrarBytesBaremacion(valorSeudonimo)
	material = anexarCampoCanonicoIntencion(material, 6, valorSeudonimo)
	material = anexarCampoCanonicoTextoIntencion(material, 7, snapshotRef)
	material = anexarCampoCanonicoIntencion(material, 8, canonicoUint64Intencion(revision))
	material = anexarCampoCanonicoTextoIntencion(material, 9, huellaSHA256)
	carga, err := NuevaCargaProtegida(material)
	if err != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return visitarCargaProtegidaEfimeraBaremacion(carga, visita)
}

// CalcularHuellaSnapshotResolucionIdentidadInternaEstableBaremacion fija
// HuellaSHA256 = SHA-256(esquema + ambito + seudonimo esperado + snapshotRef +
// revision + ancla binaria). La huella compromete el ancla sin publicarla.
func CalcularHuellaSnapshotResolucionIdentidadInternaEstableBaremacion(
	ambito SolicitudResolverSeudonimoSujetoBaremacion,
	seudonimo SeudonimoSujetoBaremacionHMAC,
	snapshotRef string,
	revision uint64,
	ancla []byte,
) (string, error) {
	if ambito.validar() != nil || seudonimo.Validar() != nil ||
		!claveCatalogoConfiguracionBaremacionValida(snapshotRef) || revision == 0 ||
		!anclaIdentidadInternaEstableBaremacionValida(ancla) {
		return "", ErrClaveIdempotenciaBaremacionInvalida
	}
	material := make([]byte, 0, 512)
	defer func() { borrarBytesBaremacion(material) }()
	material = anexarCampoCanonicoTextoIntencion(material, 0, esquemaHuellaSnapshotIdentidadBaremacionV1)
	material = anexarCampoCanonicoTextoIntencion(material, 1, ambito.procesoRef)
	material = anexarCampoCanonicoTextoIntencion(material, 2, ambito.solicitudRef)
	material = anexarCampoCanonicoTextoIntencion(material, 3, ambito.baremacionMeritoRef)
	material = anexarCampoCanonicoIntencion(material, 4, canonicoUint16Intencion(seudonimo.Version))
	material = anexarCampoCanonicoTextoIntencion(material, 5, seudonimo.ClaveHMACRef)
	valorSeudonimo := decodificarHexCanonicoIntencion(seudonimo.ValorHMAC)
	defer borrarBytesBaremacion(valorSeudonimo)
	material = anexarCampoCanonicoIntencion(material, 6, valorSeudonimo)
	material = anexarCampoCanonicoTextoIntencion(material, 7, snapshotRef)
	material = anexarCampoCanonicoIntencion(material, 8, canonicoUint64Intencion(revision))
	material = anexarCampoCanonicoIntencion(material, 9, ancla)
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

type loteResolucionIdentidadInternaEstableBaremacion struct {
	ancla       []byte
	instantanea instantaneaResolucionIdentidadInternaEstableBaremacion
}

func anclaIdentidadInternaEstableBaremacionValida(ancla []byte) bool {
	if len(ancla) != longitudIdentidadInternaEstableBaremacion || utf8.Valid(ancla) {
		return false
	}
	distintos := make(map[byte]struct{}, len(ancla))
	noCero := false
	for _, valor := range ancla {
		distintos[valor] = struct{}{}
		if valor != 0 {
			noCero = true
		}
	}
	// Es solo una barrera estructural contra cero, DNI/texto y baja diversidad;
	// no demuestra entropia. La politica operativa exige CSPRNG servidor,
	// cifrado en el servicio de identidad e inmutabilidad de la relacion.
	return noCero && len(distintos) >= 16
}

func (l *loteResolucionIdentidadInternaEstableBaremacion) destruir() {
	if l == nil {
		return
	}
	borrarBytesBaremacion(l.ancla)
	borrarBytesBaremacion(l.instantanea.ValorAtestacion)
	*l = loteResolucionIdentidadInternaEstableBaremacion{}
}

type receptorResolucionIdentidadInternaEstableBaremacion struct {
	mu        sync.Mutex
	lote      loteResolucionIdentidadInternaEstableBaremacion
	cerrado   bool
	enVuelo   bool
	llamadas  uint8
	violacion bool
}

func (*receptorResolucionIdentidadInternaEstableBaremacion) String() string {
	return "[RECEPTOR-EFIMERO-RESOLUCION-IDENTIDAD-INTERNA-ESTABLE-BAREMACION-PROTEGIDO]"
}
func (*receptorResolucionIdentidadInternaEstableBaremacion) GoString() string {
	return "ports.receptorResolucionIdentidadInternaEstableBaremacion{[PROTEGIDO]}"
}
func (r *receptorResolucionIdentidadInternaEstableBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (*receptorResolucionIdentidadInternaEstableBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (*receptorResolucionIdentidadInternaEstableBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (*receptorResolucionIdentidadInternaEstableBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (r *receptorResolucionIdentidadInternaEstableBaremacion) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func (r *receptorResolucionIdentidadInternaEstableBaremacion) RegistrarResolucionIdentidadInternaEstableBaremacion(
	ctx context.Context,
	ancla []byte,
	snapshotRef string,
	revision uint64,
	huellaSHA256 string,
	formatoAtestacion string,
	emisorAtestacionRef string,
	claveAtestacionRef string,
	valorAtestacion []byte,
) error {
	if dependenciaNulaBaremacion(ctx) || len(ancla) != longitudIdentidadInternaEstableBaremacion ||
		len(valorAtestacion) < 16 || len(valorAtestacion) > 8192 {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	copiaAncla := append([]byte(nil), ancla...)
	copiaValorAtestacion := append([]byte(nil), valorAtestacion...)
	instantanea := instantaneaResolucionIdentidadInternaEstableBaremacion{
		SnapshotRef: snapshotRef, Revision: revision, HuellaSHA256: huellaSHA256,
		FormatoAtestacion: formatoAtestacion, EmisorAtestacionRef: emisorAtestacionRef,
		ClaveAtestacionRef: claveAtestacionRef, ValorAtestacion: copiaValorAtestacion,
	}
	if !anclaIdentidadInternaEstableBaremacionValida(copiaAncla) ||
		instantanea.validar() != nil {
		borrarBytesBaremacion(copiaAncla)
		borrarBytesBaremacion(copiaValorAtestacion)
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	r.mu.Lock()
	if r.cerrado || r.enVuelo || r.llamadas != 0 {
		r.violacion = true
		r.mu.Unlock()
		borrarBytesBaremacion(copiaAncla)
		borrarBytesBaremacion(copiaValorAtestacion)
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	if err := ctx.Err(); err != nil {
		r.mu.Unlock()
		borrarBytesBaremacion(copiaAncla)
		borrarBytesBaremacion(copiaValorAtestacion)
		return err
	}
	r.llamadas++
	r.enVuelo = true
	r.mu.Unlock()

	r.mu.Lock()
	if r.cerrado {
		r.violacion = true
		borrarBytesBaremacion(copiaAncla)
		borrarBytesBaremacion(copiaValorAtestacion)
		r.enVuelo = false
		r.mu.Unlock()
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	r.lote = loteResolucionIdentidadInternaEstableBaremacion{
		ancla: copiaAncla, instantanea: instantanea,
	}
	r.enVuelo = false
	r.mu.Unlock()
	return nil
}

func (r *receptorResolucionIdentidadInternaEstableBaremacion) destruir() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.cerrado = true
	r.enVuelo = false
	r.lote.destruir()
	r.mu.Unlock()
}

func (r *receptorResolucionIdentidadInternaEstableBaremacion) finalizar() (
	loteResolucionIdentidadInternaEstableBaremacion,
	bool,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cerrado {
		return loteResolucionIdentidadInternaEstableBaremacion{}, false
	}
	r.cerrado = true
	valida := !r.enVuelo && !r.violacion && r.llamadas == 1 &&
		anclaIdentidadInternaEstableBaremacionValida(r.lote.ancla) && r.lote.instantanea.validar() == nil
	if !valida {
		r.lote.destruir()
		return loteResolucionIdentidadInternaEstableBaremacion{}, false
	}
	lote := loteResolucionIdentidadInternaEstableBaremacion{
		ancla: append([]byte(nil), r.lote.ancla...), instantanea: r.lote.instantanea.clonar(),
	}
	r.lote.destruir()
	return lote, true
}

func resolverIdentidadInternaEstableBaremacion(
	ctx context.Context,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	frontera FronteraIdentidadInternaEstableIdempotenciaBaremacion,
) (loteResolucionIdentidadInternaEstableBaremacion, error) {
	if dependenciaNulaBaremacion(ctx) || dependenciaNulaBaremacion(frontera) || solicitud.validar() != nil {
		return loteResolucionIdentidadInternaEstableBaremacion{}, ErrClaveIdempotenciaBaremacionInvalida
	}
	if err := ctx.Err(); err != nil {
		return loteResolucionIdentidadInternaEstableBaremacion{}, err
	}
	receptor := &receptorResolucionIdentidadInternaEstableBaremacion{}
	defer receptor.destruir()
	errEntrega := frontera.ResolverYEntregarIdentidadInternaEstableIdempotenciaBaremacion(
		ctx, solicitud.ambitoSujeto, solicitud.seudonimo, receptor,
	)
	lote, completa := receptor.finalizar()
	if err := ctx.Err(); err != nil {
		lote.destruir()
		return loteResolucionIdentidadInternaEstableBaremacion{}, err
	}
	if errEntrega != nil || !completa {
		lote.destruir()
		return loteResolucionIdentidadInternaEstableBaremacion{}, ErrClaveIdempotenciaBaremacionInvalida
	}
	huellaEsperada, err := CalcularHuellaSnapshotResolucionIdentidadInternaEstableBaremacion(
		solicitud.ambitoSujeto, solicitud.seudonimo, lote.instantanea.SnapshotRef,
		lote.instantanea.Revision, lote.ancla,
	)
	if err != nil || !textoIgualConstanteBaremacion(
		huellaEsperada, lote.instantanea.HuellaSHA256,
	) {
		lote.destruir()
		return loteResolucionIdentidadInternaEstableBaremacion{}, ErrClaveIdempotenciaBaremacionInvalida
	}
	return lote, nil
}

type fuenteEfimeraIdentidadInternaEstableBaremacion struct {
	mu           sync.Mutex
	valor        []byte
	iniciada     bool
	enVuelo      bool
	finalizadaOK bool
	destruida    bool
}

func nuevaFuenteEfimeraIdentidadInternaEstableBaremacion(
	identidad []byte,
) *fuenteEfimeraIdentidadInternaEstableBaremacion {
	return &fuenteEfimeraIdentidadInternaEstableBaremacion{valor: append([]byte(nil), identidad...)}
}

func (*fuenteEfimeraIdentidadInternaEstableBaremacion) String() string {
	return "[FUENTE-EFIMERA-IDENTIDAD-INTERNA-ESTABLE-BAREMACION-PROTEGIDA]"
}
func (*fuenteEfimeraIdentidadInternaEstableBaremacion) GoString() string {
	return "ports.fuenteEfimeraIdentidadInternaEstableBaremacion{[PROTEGIDA]}"
}
func (f *fuenteEfimeraIdentidadInternaEstableBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, f.String())
}
func (*fuenteEfimeraIdentidadInternaEstableBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (*fuenteEfimeraIdentidadInternaEstableBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (*fuenteEfimeraIdentidadInternaEstableBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (f *fuenteEfimeraIdentidadInternaEstableBaremacion) LogValue() slog.Value {
	return slog.StringValue(f.String())
}

func (f *fuenteEfimeraIdentidadInternaEstableBaremacion) EntregarIdentidadInternaEstableIdempotenciaBaremacion(
	ctx context.Context,
	consumidor ConsumidorIdentidadInternaEstableIdempotenciaBaremacion,
) error {
	if dependenciaNulaBaremacion(ctx) || dependenciaNulaBaremacion(consumidor) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	f.mu.Lock()
	if f.destruida || f.iniciada || !anclaIdentidadInternaEstableBaremacionValida(f.valor) {
		f.mu.Unlock()
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	if err := ctx.Err(); err != nil {
		f.mu.Unlock()
		return err
	}
	f.iniciada = true
	f.enVuelo = true
	copia := append([]byte(nil), f.valor...)
	f.mu.Unlock()

	termino := false
	var errRetorno error
	defer func() {
		borrarBytesBaremacion(copia)
		f.mu.Lock()
		f.enVuelo = false
		if termino && errRetorno == nil && !f.destruida {
			f.finalizadaOK = true
		}
		f.mu.Unlock()
	}()
	errRetorno = consumidor.ConsumirIdentidadInternaEstableIdempotenciaBaremacion(ctx, copia)
	if err := ctx.Err(); err != nil {
		errRetorno = err
	}
	termino = true
	return errRetorno
}

func (f *fuenteEfimeraIdentidadInternaEstableBaremacion) cerrarYComprobarCompletada() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.destruida {
		return false
	}
	f.destruida = true
	borrarBytesBaremacion(f.valor)
	f.valor = nil
	return f.iniciada && !f.enVuelo && f.finalizadaOK
}

func (f *fuenteEfimeraIdentidadInternaEstableBaremacion) destruir() {
	f.mu.Lock()
	f.destruida = true
	borrarBytesBaremacion(f.valor)
	f.valor = nil
	f.mu.Unlock()
}

// ConsumidorClaveClienteLoteIdempotenciaBaremacion pertenece exclusivamente al
// adaptador HSM/KMS. La fuente se crea dentro de la fabrica, permite una sola
// entrega y se destruye al terminar; el llamador nunca recibe la fuente.
type ConsumidorClaveClienteLoteIdempotenciaBaremacion interface {
	ConsumirClaveClienteLoteIdempotenciaBaremacion(context.Context, []byte) error
}

type FuenteEfimeraClaveClienteIdempotenciaBaremacion interface {
	EntregarClaveClienteLoteIdempotenciaBaremacion(
		context.Context,
		ConsumidorClaveClienteLoteIdempotenciaBaremacion,
	) error
}

type fuenteEfimeraClaveClienteIdempotenciaBaremacion struct {
	mu                             sync.Mutex
	valor                          []byte
	entregaIniciada                bool
	entregasEnVuelo                int
	entregaFinalizadaCorrectamente bool
	destruida                      bool
}

func (*fuenteEfimeraClaveClienteIdempotenciaBaremacion) String() string {
	return "[FUENTE-EFIMERA-CLAVE-CLIENTE-BAREMACION-PROTEGIDA]"
}
func (*fuenteEfimeraClaveClienteIdempotenciaBaremacion) GoString() string {
	return "ports.fuenteEfimeraClaveClienteIdempotenciaBaremacion{[PROTEGIDA]}"
}
func (f *fuenteEfimeraClaveClienteIdempotenciaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, f.String())
}
func (*fuenteEfimeraClaveClienteIdempotenciaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (*fuenteEfimeraClaveClienteIdempotenciaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (*fuenteEfimeraClaveClienteIdempotenciaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (f *fuenteEfimeraClaveClienteIdempotenciaBaremacion) LogValue() slog.Value {
	return slog.StringValue(f.String())
}

func nuevaFuenteEfimeraClaveClienteIdempotenciaBaremacion(
	valor []byte,
) *fuenteEfimeraClaveClienteIdempotenciaBaremacion {
	return &fuenteEfimeraClaveClienteIdempotenciaBaremacion{valor: append([]byte(nil), valor...)}
}

func (f *fuenteEfimeraClaveClienteIdempotenciaBaremacion) EntregarClaveClienteLoteIdempotenciaBaremacion(
	ctx context.Context,
	consumidor ConsumidorClaveClienteLoteIdempotenciaBaremacion,
) (errRetorno error) {
	if dependenciaNulaBaremacion(ctx) || dependenciaNulaBaremacion(consumidor) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	f.mu.Lock()
	if f.destruida || f.entregaIniciada || len(f.valor) == 0 {
		f.mu.Unlock()
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	if err := ctx.Err(); err != nil {
		f.mu.Unlock()
		return err
	}
	f.entregaIniciada = true
	f.entregasEnVuelo++
	copia := append([]byte(nil), f.valor...)
	f.mu.Unlock()
	termino := false
	defer func() {
		borrarBytesBaremacion(copia)
		f.mu.Lock()
		f.entregasEnVuelo--
		if termino && errRetorno == nil && !f.destruida {
			f.entregaFinalizadaCorrectamente = true
		}
		f.mu.Unlock()
	}()
	errConsumo := consumidor.ConsumirClaveClienteLoteIdempotenciaBaremacion(ctx, copia)
	if err := ctx.Err(); err != nil {
		errRetorno = err
		termino = true
		return errRetorno
	}
	errRetorno = errConsumo
	termino = true
	return errRetorno
}

func (f *fuenteEfimeraClaveClienteIdempotenciaBaremacion) destruir() {
	f.mu.Lock()
	f.destruida = true
	borrarBytesBaremacion(f.valor)
	f.valor = nil
	f.mu.Unlock()
}

func (f *fuenteEfimeraClaveClienteIdempotenciaBaremacion) cerrarYComprobarCompletada() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.destruida {
		return false
	}
	f.destruida = true
	borrarBytesBaremacion(f.valor)
	f.valor = nil
	return f.entregaIniciada && f.entregasEnVuelo == 0 && f.entregaFinalizadaCorrectamente
}

const esquemaTestimonioAtomicoIdempotenciaBaremacionV1 = "vec.bolsa.testimonio-atomico-idempotencia.v1"

// topologiaClaveHMACBaremacion es privada: una entrada aislada no puede
// convertirse en autoridad ni usarse para seleccionar una clave historica.
type topologiaClaveHMACBaremacion struct {
	Version         uint16
	GeneracionClave uint32
	ClaveHMACRef    string
}

func (t topologiaClaveHMACBaremacion) validar(dominio DominioClaveHMACBaremacion) error {
	versionEsperada := VersionIndiceIdempotenciaBaremacionV1
	if dominio == DominioClavePrincipalBaremacion {
		versionEsperada = VersionPrincipalEstableBaremacionV1
	}
	if (dominio != DominioClavePrincipalBaremacion && dominio != DominioClaveIndiceBaremacion) ||
		t.Version != versionEsperada || t.GeneracionClave < 1 ||
		!claveCatalogoConfiguracionBaremacionValida(t.ClaveHMACRef) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return nil
}

func (topologiaClaveHMACBaremacion) String() string {
	return "[TOPOLOGIA-CLAVE-HMAC-BAREMACION-INTERNA-PROTEGIDA]"
}
func (topologiaClaveHMACBaremacion) GoString() string {
	return "ports.topologiaClaveHMACBaremacion{[PROTEGIDA]}"
}
func (t topologiaClaveHMACBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, t.String())
}
func (topologiaClaveHMACBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (topologiaClaveHMACBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (topologiaClaveHMACBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (t topologiaClaveHMACBaremacion) LogValue() slog.Value { return slog.StringValue(t.String()) }

type instantaneaLlaveroHMACBaremacion struct {
	Dominio      DominioClaveHMACBaremacion
	LlaveroRef   string
	Revision     uint64
	Cantidad     uint8
	HuellaSHA256 string
	Topologia    []topologiaClaveHMACBaremacion
}

func (i instantaneaLlaveroHMACBaremacion) validar() error {
	if (i.Dominio != DominioClavePrincipalBaremacion && i.Dominio != DominioClaveIndiceBaremacion) ||
		!claveCatalogoConfiguracionBaremacionValida(i.LlaveroRef) || i.Revision == 0 ||
		i.Cantidad < 1 || int(i.Cantidad) > maximoGeneracionesHMACIdempotenciaBaremacion ||
		int(i.Cantidad) != len(i.Topologia) || !huellaSHA256Valida(i.HuellaSHA256) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	referencias := make(map[string]struct{}, len(i.Topologia))
	var generacionAnterior uint32
	for posicion, entrada := range i.Topologia {
		if entrada.validar(i.Dominio) != nil ||
			(posicion > 0 && entrada.GeneracionClave >= generacionAnterior) {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
		if _, repetida := referencias[entrada.ClaveHMACRef]; repetida {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
		referencias[entrada.ClaveHMACRef] = struct{}{}
		generacionAnterior = entrada.GeneracionClave
	}
	material := i.representacionCanonicaSinHuella()
	defer func() { borrarBytesBaremacion(material) }()
	huella := sha256.Sum256(material)
	if !textoIgualConstanteBaremacion(hex.EncodeToString(huella[:]), i.HuellaSHA256) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return nil
}

func (i instantaneaLlaveroHMACBaremacion) representacionCanonicaSinHuella() []byte {
	material := make([]byte, 0, 256)
	material = anexarCampoCanonicoTextoIntencion(material, 1, string(i.Dominio))
	material = anexarCampoCanonicoTextoIntencion(material, 2, i.LlaveroRef)
	material = anexarCampoCanonicoIntencion(material, 3, canonicoUint64Intencion(i.Revision))
	material = anexarCampoCanonicoIntencion(material, 4, []byte{i.Cantidad})
	for posicion, entrada := range i.Topologia {
		base := uint16(100 + posicion*4)
		material = anexarCampoCanonicoIntencion(material, base, canonicoUint16Intencion(entrada.Version))
		material = anexarCampoCanonicoIntencion(material, base+1, canonicoUint32Intencion(entrada.GeneracionClave))
		material = anexarCampoCanonicoTextoIntencion(material, base+2, entrada.ClaveHMACRef)
	}
	return material
}

func (i instantaneaLlaveroHMACBaremacion) clonar() instantaneaLlaveroHMACBaremacion {
	clon := i
	clon.Topologia = append([]topologiaClaveHMACBaremacion(nil), i.Topologia...)
	return clon
}

func (instantaneaLlaveroHMACBaremacion) String() string {
	return "[INSTANTANEA-LLAVERO-HMAC-BAREMACION-INTERNA-PROTEGIDA]"
}
func (instantaneaLlaveroHMACBaremacion) GoString() string {
	return "ports.instantaneaLlaveroHMACBaremacion{[PROTEGIDA]}"
}
func (i instantaneaLlaveroHMACBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, i.String())
}
func (instantaneaLlaveroHMACBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (instantaneaLlaveroHMACBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (instantaneaLlaveroHMACBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (i instantaneaLlaveroHMACBaremacion) LogValue() slog.Value {
	return slog.StringValue(i.String())
}

type evidenciaAtestacionIdempotenciaBaremacion struct {
	Formato               string
	EmisorRef             string
	ClaveAtestacionRef    string
	Revision              uint64
	HuellaContenidoSHA256 string
	Valor                 []byte
}

func (e evidenciaAtestacionIdempotenciaBaremacion) validar() error {
	if !claveCatalogoConfiguracionBaremacionValida(e.Formato) ||
		!claveCatalogoConfiguracionBaremacionValida(e.EmisorRef) ||
		!claveCatalogoConfiguracionBaremacionValida(e.ClaveAtestacionRef) ||
		e.Revision == 0 || !huellaSHA256Valida(e.HuellaContenidoSHA256) ||
		len(e.Valor) < 16 || len(e.Valor) > 8192 {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return nil
}

func (e evidenciaAtestacionIdempotenciaBaremacion) clonar() evidenciaAtestacionIdempotenciaBaremacion {
	clon := e
	clon.Valor = append([]byte(nil), e.Valor...)
	return clon
}

func (evidenciaAtestacionIdempotenciaBaremacion) String() string {
	return "[EVIDENCIA-ATESTACION-IDEMPOTENCIA-BAREMACION-INTERNA-PROTEGIDA]"
}
func (evidenciaAtestacionIdempotenciaBaremacion) GoString() string {
	return "ports.evidenciaAtestacionIdempotenciaBaremacion{[PROTEGIDA]}"
}
func (e evidenciaAtestacionIdempotenciaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}
func (evidenciaAtestacionIdempotenciaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (evidenciaAtestacionIdempotenciaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (evidenciaAtestacionIdempotenciaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (e evidenciaAtestacionIdempotenciaBaremacion) LogValue() slog.Value {
	return slog.StringValue(e.String())
}

type testimonioAtomicoIdempotenciaBaremacion struct {
	vinculoSolicitud    [32]byte
	resolucionIdentidad instantaneaResolucionIdentidadInternaEstableBaremacion
	identidades         instantaneaLlaveroHMACBaremacion
	indices             instantaneaLlaveroHMACBaremacion
	principales         []principalEstableBaremacionHMAC
	matriz              [][]indiceIdempotenciaBaremacion
	evidencia           evidenciaAtestacionIdempotenciaBaremacion
}

func (t testimonioAtomicoIdempotenciaBaremacion) validarCuerpo() error {
	if t.vinculoSolicitud == ([32]byte{}) || t.resolucionIdentidad.validar() != nil ||
		t.identidades.validar() != nil ||
		t.indices.validar() != nil || t.identidades.Dominio != DominioClavePrincipalBaremacion ||
		t.indices.Dominio != DominioClaveIndiceBaremacion ||
		len(t.principales) != int(t.identidades.Cantidad) ||
		len(t.matriz) != int(t.identidades.Cantidad) ||
		validarPrincipalesInternosBaremacion(t.principales) != nil ||
		t.identidades.LlaveroRef == t.indices.LlaveroRef {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	referenciasClaves := make(map[string]struct{}, len(t.identidades.Topologia)+len(t.indices.Topologia))
	for _, entrada := range t.identidades.Topologia {
		referenciasClaves[entrada.ClaveHMACRef] = struct{}{}
	}
	for _, entrada := range t.indices.Topologia {
		if _, repetida := referenciasClaves[entrada.ClaveHMACRef]; repetida {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
		referenciasClaves[entrada.ClaveHMACRef] = struct{}{}
	}
	valores := make(map[string]struct{}, len(t.principales)*(int(t.indices.Cantidad)+1))
	for posicion, principal := range t.principales {
		topologia := t.identidades.Topologia[posicion]
		if principal.Version != topologia.Version ||
			principal.GeneracionClave != topologia.GeneracionClave ||
			principal.ClaveHMACRef != topologia.ClaveHMACRef {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
		if _, repetido := valores[principal.ValorHMAC]; repetido {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
		valores[principal.ValorHMAC] = struct{}{}
		fila := t.matriz[posicion]
		if len(fila) != int(t.indices.Cantidad) || validarIndicesInternosBaremacion(fila) != nil {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
		for columna, indice := range fila {
			topologiaIndice := t.indices.Topologia[columna]
			if indice.Version != topologiaIndice.Version ||
				indice.GeneracionClave != topologiaIndice.GeneracionClave ||
				indice.ClaveHMACRef != topologiaIndice.ClaveHMACRef {
				return ErrClaveIdempotenciaBaremacionInvalida
			}
			if _, repetido := valores[indice.ValorHMAC]; repetido {
				return ErrClaveIdempotenciaBaremacionInvalida
			}
			valores[indice.ValorHMAC] = struct{}{}
		}
	}
	return nil
}

func (t testimonioAtomicoIdempotenciaBaremacion) validarEstructura() error {
	if t.validarCuerpo() != nil || t.evidencia.validar() != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	material, err := t.representacionCanonicaSinEvidencia()
	if err != nil {
		return err
	}
	defer destruirCargaProtegidaBaremacion(&material)
	huella := sumaSHA256CargaProtegidaBaremacion(material)
	if !textoIgualConstanteBaremacion(hex.EncodeToString(huella[:]), t.evidencia.HuellaContenidoSHA256) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return nil
}

func (t testimonioAtomicoIdempotenciaBaremacion) representacionCanonicaSinEvidencia() (CargaProtegida, error) {
	if t.validarCuerpo() != nil {
		return CargaProtegida{}, ErrClaveIdempotenciaBaremacionInvalida
	}
	material := make([]byte, 0, 4096)
	defer func() { borrarBytesBaremacion(material) }()
	material = anexarCampoCanonicoTextoIntencion(material, 0, esquemaTestimonioAtomicoIdempotenciaBaremacionV1)
	material = anexarCampoCanonicoIntencion(material, 1, t.vinculoSolicitud[:])
	material = anexarCampoCanonicoTextoIntencion(material, 2, EsquemaCanonicoSeudonimoSujetoBaremacionV1)
	material = anexarCampoCanonicoTextoIntencion(material, 3, EsquemaCanonicoPrincipalEstableBaremacionV1)
	material = anexarCampoCanonicoTextoIntencion(material, 4, EsquemaCanonicoIndiceIdempotenciaBaremacionV1)
	material = anexarCampoCanonicoTextoIntencion(material, 5, PoliticaDerivacionIdempotenciaBaremacionDEC045V1)
	resolucionCanonica := t.resolucionIdentidad.representacionCanonica()
	defer borrarBytesBaremacion(resolucionCanonica)
	identidadesCanonicas := t.identidades.representacionCanonicaSinHuella()
	defer borrarBytesBaremacion(identidadesCanonicas)
	indicesCanonicos := t.indices.representacionCanonicaSinHuella()
	defer borrarBytesBaremacion(indicesCanonicos)
	material = anexarCampoCanonicoIntencion(material, 6, resolucionCanonica)
	material = anexarCampoCanonicoIntencion(material, 7, identidadesCanonicas)
	material = anexarCampoCanonicoTextoIntencion(material, 8, t.identidades.HuellaSHA256)
	material = anexarCampoCanonicoIntencion(material, 9, indicesCanonicos)
	material = anexarCampoCanonicoTextoIntencion(material, 10, t.indices.HuellaSHA256)
	etiqueta := uint16(100)
	for _, principal := range t.principales {
		material = anexarCampoCanonicoIntencion(material, etiqueta, canonicoUint16Intencion(principal.Version))
		material = anexarCampoCanonicoIntencion(material, etiqueta+1, canonicoUint32Intencion(principal.GeneracionClave))
		material = anexarCampoCanonicoTextoIntencion(material, etiqueta+2, principal.ClaveHMACRef)
		material = anexarCampoCanonicoTextoIntencion(material, etiqueta+3, principal.ValorHMAC)
		etiqueta += 4
	}
	for _, fila := range t.matriz {
		for _, indice := range fila {
			material = anexarCampoCanonicoIntencion(material, etiqueta, canonicoUint16Intencion(indice.Version))
			material = anexarCampoCanonicoIntencion(material, etiqueta+1, canonicoUint32Intencion(indice.GeneracionClave))
			material = anexarCampoCanonicoTextoIntencion(material, etiqueta+2, indice.ClaveHMACRef)
			material = anexarCampoCanonicoTextoIntencion(material, etiqueta+3, indice.ValorHMAC)
			etiqueta += 4
		}
	}
	return NuevaCargaProtegida(material)
}

func (t testimonioAtomicoIdempotenciaBaremacion) representacionCanonicaAtestada() (CargaProtegida, error) {
	if t.validarEstructura() != nil {
		return CargaProtegida{}, ErrClaveIdempotenciaBaremacionInvalida
	}
	cuerpo, err := t.representacionCanonicaSinEvidencia()
	if err != nil {
		return CargaProtegida{}, err
	}
	defer destruirCargaProtegidaBaremacion(&cuerpo)
	material := cuerpo.Revelar()
	defer func() { borrarBytesBaremacion(material) }()
	material = anexarCampoCanonicoTextoIntencion(material, 60000, t.evidencia.Formato)
	material = anexarCampoCanonicoTextoIntencion(material, 60001, t.evidencia.EmisorRef)
	material = anexarCampoCanonicoTextoIntencion(material, 60002, t.evidencia.ClaveAtestacionRef)
	material = anexarCampoCanonicoIntencion(material, 60003, canonicoUint64Intencion(t.evidencia.Revision))
	material = anexarCampoCanonicoTextoIntencion(material, 60004, t.evidencia.HuellaContenidoSHA256)
	material = anexarCampoCanonicoIntencion(material, 60005, t.evidencia.Valor)
	return NuevaCargaProtegida(material)
}

func (t testimonioAtomicoIdempotenciaBaremacion) clonar() (testimonioAtomicoIdempotenciaBaremacion, error) {
	if t.validarEstructura() != nil {
		return testimonioAtomicoIdempotenciaBaremacion{}, ErrClaveIdempotenciaBaremacionInvalida
	}
	clon := t
	clon.resolucionIdentidad = t.resolucionIdentidad.clonar()
	clon.identidades = t.identidades.clonar()
	clon.indices = t.indices.clonar()
	clon.principales = append([]principalEstableBaremacionHMAC(nil), t.principales...)
	clon.matriz = make([][]indiceIdempotenciaBaremacion, len(t.matriz))
	for fila := range t.matriz {
		clon.matriz[fila] = append([]indiceIdempotenciaBaremacion(nil), t.matriz[fila]...)
	}
	clon.evidencia = t.evidencia.clonar()
	return clon, nil
}

func destruirTestimonioAtomicoIdempotenciaBaremacion(
	testimonio *testimonioAtomicoIdempotenciaBaremacion,
) {
	if testimonio == nil {
		return
	}
	borrarBytesBaremacion(testimonio.resolucionIdentidad.ValorAtestacion)
	for posicion := range testimonio.identidades.Topologia {
		testimonio.identidades.Topologia[posicion] = topologiaClaveHMACBaremacion{}
	}
	for posicion := range testimonio.indices.Topologia {
		testimonio.indices.Topologia[posicion] = topologiaClaveHMACBaremacion{}
	}
	for posicion := range testimonio.principales {
		testimonio.principales[posicion] = principalEstableBaremacionHMAC{}
	}
	for fila := range testimonio.matriz {
		for columna := range testimonio.matriz[fila] {
			testimonio.matriz[fila][columna] = indiceIdempotenciaBaremacion{}
		}
		testimonio.matriz[fila] = nil
	}
	borrarBytesBaremacion(testimonio.evidencia.Valor)
	*testimonio = testimonioAtomicoIdempotenciaBaremacion{}
}

func (testimonioAtomicoIdempotenciaBaremacion) String() string {
	return "[TESTIMONIO-ATOMICO-IDEMPOTENCIA-BAREMACION-INTERNO-PROTEGIDO]"
}
func (testimonioAtomicoIdempotenciaBaremacion) GoString() string {
	return "ports.testimonioAtomicoIdempotenciaBaremacion{[PROTEGIDO]}"
}
func (t testimonioAtomicoIdempotenciaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, t.String())
}
func (testimonioAtomicoIdempotenciaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (testimonioAtomicoIdempotenciaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (testimonioAtomicoIdempotenciaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (t testimonioAtomicoIdempotenciaBaremacion) LogValue() slog.Value {
	return slog.StringValue(t.String())
}

// ReceptorEfimeroTestimonioAtomicoIdempotenciaBaremacion es la unica via de
// construccion para un adaptador externo. El receptor lo crea la fabrica, se
// cierra tras una sola llamada al productor y no devuelve tipos individuales.
type ReceptorEfimeroTestimonioAtomicoIdempotenciaBaremacion interface {
	InmovilizarLlaveroIdentidadesBaremacion(string, uint64, uint8, string) error
	RegistrarPrincipalEstableBaremacion(int, uint16, uint32, string, string) error
	InmovilizarLlaveroIndicesBaremacion(string, uint64, uint8, string) error
	RegistrarIndiceIdempotenciaBaremacion(int, int, uint16, uint32, string, string) error
	VisitarMaterialCanonicoParaAtestacionBaremacion(func(MaterialCanonicoEfimeroBaremacion) error) error
	RegistrarEvidenciaAtestacionBaremacion(string, string, string, uint64, string, []byte) error
}

type receptorEfimeroTestimonioIdempotenciaBaremacion struct {
	mu                  sync.Mutex
	testimonio          testimonioAtomicoIdempotenciaBaremacion
	inmovilizoIdentidad bool
	inmovilizoIndices   bool
	materialIniciado    bool
	materialEnVuelo     bool
	materialCompletado  bool
	registroEvidencia   bool
	cerrado             bool
}

func (*receptorEfimeroTestimonioIdempotenciaBaremacion) String() string {
	return "[RECEPTOR-EFIMERO-TESTIMONIO-IDEMPOTENCIA-BAREMACION-PROTEGIDO]"
}
func (*receptorEfimeroTestimonioIdempotenciaBaremacion) GoString() string {
	return "ports.receptorEfimeroTestimonioIdempotenciaBaremacion{[PROTEGIDO]}"
}
func (r *receptorEfimeroTestimonioIdempotenciaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (*receptorEfimeroTestimonioIdempotenciaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (*receptorEfimeroTestimonioIdempotenciaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (*receptorEfimeroTestimonioIdempotenciaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (r *receptorEfimeroTestimonioIdempotenciaBaremacion) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func nuevoReceptorEfimeroTestimonioIdempotenciaBaremacion(
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	resolucion instantaneaResolucionIdentidadInternaEstableBaremacion,
) *receptorEfimeroTestimonioIdempotenciaBaremacion {
	return &receptorEfimeroTestimonioIdempotenciaBaremacion{
		testimonio: testimonioAtomicoIdempotenciaBaremacion{
			vinculoSolicitud:    solicitud.vinculo,
			resolucionIdentidad: resolucion.clonar(),
		},
	}
}

func (r *receptorEfimeroTestimonioIdempotenciaBaremacion) destruir() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.cerrado = true
	destruirTestimonioAtomicoIdempotenciaBaremacion(&r.testimonio)
	r.mu.Unlock()
}

func (r *receptorEfimeroTestimonioIdempotenciaBaremacion) InmovilizarLlaveroIdentidadesBaremacion(
	referencia string, revision uint64, cantidad uint8, huella string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cerrado || r.inmovilizoIdentidad || r.inmovilizoIndices || cantidad < 1 ||
		int(cantidad) > maximoGeneracionesHMACIdempotenciaBaremacion {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	r.testimonio.identidades = instantaneaLlaveroHMACBaremacion{
		Dominio: DominioClavePrincipalBaremacion, LlaveroRef: referencia,
		Revision: revision, Cantidad: cantidad, HuellaSHA256: huella,
		Topologia: make([]topologiaClaveHMACBaremacion, int(cantidad)),
	}
	r.testimonio.principales = make([]principalEstableBaremacionHMAC, int(cantidad))
	r.inmovilizoIdentidad = true
	return nil
}

func (r *receptorEfimeroTestimonioIdempotenciaBaremacion) RegistrarPrincipalEstableBaremacion(
	posicion int, version uint16, generacion uint32, referencia, valor string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cerrado || !r.inmovilizoIdentidad || r.inmovilizoIndices ||
		posicion < 0 || posicion >= len(r.testimonio.principales) ||
		r.testimonio.principales[posicion] != (principalEstableBaremacionHMAC{}) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	principal := principalEstableBaremacionHMAC{
		Version: version, GeneracionClave: generacion, ClaveHMACRef: referencia, ValorHMAC: valor,
	}
	if principal.validar() != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	r.testimonio.principales[posicion] = principal
	r.testimonio.identidades.Topologia[posicion] = topologiaClaveHMACBaremacion{
		Version: version, GeneracionClave: generacion, ClaveHMACRef: referencia,
	}
	return nil
}

func (r *receptorEfimeroTestimonioIdempotenciaBaremacion) InmovilizarLlaveroIndicesBaremacion(
	referencia string, revision uint64, cantidad uint8, huella string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cerrado || !r.inmovilizoIdentidad || r.inmovilizoIndices || cantidad < 1 ||
		int(cantidad) > maximoGeneracionesHMACIdempotenciaBaremacion ||
		validarPrincipalesInternosBaremacion(r.testimonio.principales) != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	r.testimonio.indices = instantaneaLlaveroHMACBaremacion{
		Dominio: DominioClaveIndiceBaremacion, LlaveroRef: referencia,
		Revision: revision, Cantidad: cantidad, HuellaSHA256: huella,
		Topologia: make([]topologiaClaveHMACBaremacion, int(cantidad)),
	}
	r.testimonio.matriz = make([][]indiceIdempotenciaBaremacion, len(r.testimonio.principales))
	for fila := range r.testimonio.matriz {
		r.testimonio.matriz[fila] = make([]indiceIdempotenciaBaremacion, int(cantidad))
	}
	r.inmovilizoIndices = true
	return nil
}

func (r *receptorEfimeroTestimonioIdempotenciaBaremacion) RegistrarIndiceIdempotenciaBaremacion(
	fila, columna int, version uint16, generacion uint32, referencia, valor string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cerrado || !r.inmovilizoIndices || fila < 0 || fila >= len(r.testimonio.matriz) ||
		columna < 0 || columna >= len(r.testimonio.matriz[fila]) ||
		r.testimonio.matriz[fila][columna] != (indiceIdempotenciaBaremacion{}) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	indice := indiceIdempotenciaBaremacion{
		Version: version, GeneracionClave: generacion, ClaveHMACRef: referencia, ValorHMAC: valor,
	}
	if indice.validar() != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	if fila > 0 {
		esperada := r.testimonio.indices.Topologia[columna]
		if esperada == (topologiaClaveHMACBaremacion{}) || esperada.Version != version ||
			esperada.GeneracionClave != generacion || esperada.ClaveHMACRef != referencia {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
	} else {
		r.testimonio.indices.Topologia[columna] = topologiaClaveHMACBaremacion{
			Version: version, GeneracionClave: generacion, ClaveHMACRef: referencia,
		}
	}
	r.testimonio.matriz[fila][columna] = indice
	return nil
}

func (r *receptorEfimeroTestimonioIdempotenciaBaremacion) VisitarMaterialCanonicoParaAtestacionBaremacion(
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) (errRetorno error) {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	r.mu.Lock()
	if r.cerrado || !r.inmovilizoIdentidad || !r.inmovilizoIndices ||
		r.materialIniciado || r.materialEnVuelo || r.materialCompletado || r.registroEvidencia {
		r.mu.Unlock()
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	material, err := r.testimonio.representacionCanonicaSinEvidencia()
	if err != nil {
		r.mu.Unlock()
		return err
	}
	r.materialIniciado = true
	r.materialEnVuelo = true
	r.mu.Unlock()
	completo := false
	defer func() {
		r.mu.Lock()
		r.materialEnVuelo = false
		if completo && errRetorno == nil && !r.cerrado {
			r.materialCompletado = true
		}
		r.mu.Unlock()
	}()
	errRetorno = visitarCargaProtegidaEfimeraBaremacion(material, visita)
	completo = true
	return errRetorno
}

func (r *receptorEfimeroTestimonioIdempotenciaBaremacion) RegistrarEvidenciaAtestacionBaremacion(
	formato, emisor, clave string, revision uint64, huella string, evidencia []byte,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cerrado || !r.inmovilizoIndices || !r.materialCompletado ||
		r.materialEnVuelo || r.registroEvidencia {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	material, err := r.testimonio.representacionCanonicaSinEvidencia()
	if err != nil {
		return err
	}
	defer destruirCargaProtegidaBaremacion(&material)
	suma := sumaSHA256CargaProtegidaBaremacion(material)
	if !textoIgualConstanteBaremacion(hex.EncodeToString(suma[:]), huella) {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	r.testimonio.evidencia = evidenciaAtestacionIdempotenciaBaremacion{
		Formato: formato, EmisorRef: emisor, ClaveAtestacionRef: clave,
		Revision: revision, HuellaContenidoSHA256: huella, Valor: append([]byte(nil), evidencia...),
	}
	if r.testimonio.evidencia.validar() != nil {
		borrarBytesBaremacion(r.testimonio.evidencia.Valor)
		r.testimonio.evidencia = evidenciaAtestacionIdempotenciaBaremacion{}
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	r.registroEvidencia = true
	return nil
}

func (r *receptorEfimeroTestimonioIdempotenciaBaremacion) finalizar() (
	testimonioAtomicoIdempotenciaBaremacion, error,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cerrado || !r.registroEvidencia {
		r.cerrado = true
		destruirTestimonioAtomicoIdempotenciaBaremacion(&r.testimonio)
		return testimonioAtomicoIdempotenciaBaremacion{}, ErrClaveIdempotenciaBaremacionInvalida
	}
	r.cerrado = true
	if r.testimonio.validarEstructura() != nil {
		destruirTestimonioAtomicoIdempotenciaBaremacion(&r.testimonio)
		return testimonioAtomicoIdempotenciaBaremacion{}, ErrClaveIdempotenciaBaremacionInvalida
	}
	clon, err := r.testimonio.clonar()
	if err != nil {
		destruirTestimonioAtomicoIdempotenciaBaremacion(&r.testimonio)
		return testimonioAtomicoIdempotenciaBaremacion{}, err
	}
	return clon, nil
}

// ProductorTestimonioAtomicoIdempotenciaBaremacion debe hacer, en una sola
// operacion del HSM/KMS, las dos instantaneas, la matriz completa y la
// atestacion ligada al contenido. No se admite resolver/verificar por sujeto.
type ProductorTestimonioAtomicoIdempotenciaBaremacion interface {
	ProducirTestimonioAtomicoIdempotenciaBaremacion(
		context.Context,
		SolicitudTestimonioAtomicoIdempotenciaBaremacion,
		FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
		FuenteEfimeraClaveClienteIdempotenciaBaremacion,
		ReceptorEfimeroTestimonioAtomicoIdempotenciaBaremacion,
	) error
}

// VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion permite a una raiz
// independiente cotejar la topologia esperada y verificar la evidencia. Solo
// vive durante la llamada; no ofrece selectores ni devuelve el producto. La
// efimeridad limita el ciclo de vida, pero codigo malicioso en el mismo proceso
// puede copiar bytes o cadenas dentro del callback. Si ese actor forma parte
// del modelo de amenaza, DEC-047 exige adaptador aislado en otro proceso.
type VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion interface {
	VisitarMaterialCanonicoAtestadoBaremacion(func(MaterialCanonicoEfimeroBaremacion) error) error
	VisitarResolucionIdentidadInternaEstableBaremacion(
		func(string, uint64, string, string, string, string, []byte) error,
	) error
	ResumenLlaveroIdentidadesBaremacion() (string, uint64, uint8, string, error)
	VisitarTopologiaIdentidadesBaremacion(func(int, uint16, uint32, string) error) error
	ResumenLlaveroIndicesBaremacion() (string, uint64, uint8, string, error)
	VisitarTopologiaIndicesBaremacion(func(int, uint16, uint32, string) error) error
	VisitarPrincipalesBaremacion(func(int, uint16, uint32, string, string) error) error
	VisitarMatrizIndicesBaremacion(func(int, int, uint16, uint32, string, string) error) error
	VisitarRepresentacionesCanonicasIntencionBaremacion(
		SolicitudTestimonioAtomicoIdempotenciaBaremacion,
		IntencionCambioBaremacion,
		[]byte,
		func(int, int, MaterialCanonicoEfimeroBaremacion, MaterialCanonicoEfimeroBaremacion) error,
	) error
	VisitarEvidenciaAtestacionBaremacion(func(string, string, string, uint64, string, []byte) error) error
}

type VerificadorIndependienteTestimonioIdempotenciaBaremacion interface {
	VerificarTestimonioAtomicoIdempotenciaBaremacion(
		context.Context,
		SolicitudTestimonioAtomicoIdempotenciaBaremacion,
		FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
		FuenteEfimeraClaveClienteIdempotenciaBaremacion,
		VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
	) error
}

// ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion
// es un punto de copia estructural, no un repositorio, CAS ni puerto de efecto.
// El servicio de aplicacion pendiente debe implementarlo con un tipo privado y
// limitarlo a crear su capacidad privada tras la reverificacion independiente.
type ConsumidorEfimeroProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion interface {
	ConsumirProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion(
		context.Context,
		SolicitudTestimonioAtomicoIdempotenciaBaremacion,
		VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
	) error
}

type vistaEfimeraTestimonioIdempotenciaBaremacion struct {
	mu             sync.Mutex
	testimonio     testimonioAtomicoIdempotenciaBaremacion
	visitasEnVuelo int
	cobertura      uint16
	cerrada        bool
}

const (
	coberturaMaterialAtestadoBaremacion uint16 = 1 << iota
	coberturaResolucionIdentidadBaremacion
	coberturaResumenIdentidadesBaremacion
	coberturaTopologiaIdentidadesBaremacion
	coberturaResumenIndicesBaremacion
	coberturaTopologiaIndicesBaremacion
	coberturaPrincipalesBaremacion
	coberturaMatrizBaremacion
	coberturaEvidenciaBaremacion
	coberturaCompletaVistaBaremacion = coberturaMaterialAtestadoBaremacion |
		coberturaResolucionIdentidadBaremacion |
		coberturaResumenIdentidadesBaremacion |
		coberturaTopologiaIdentidadesBaremacion |
		coberturaResumenIndicesBaremacion |
		coberturaTopologiaIndicesBaremacion |
		coberturaPrincipalesBaremacion |
		coberturaMatrizBaremacion |
		coberturaEvidenciaBaremacion
)

func nuevaVistaEfimeraTestimonioIdempotenciaBaremacion(
	testimonio testimonioAtomicoIdempotenciaBaremacion,
) *vistaEfimeraTestimonioIdempotenciaBaremacion {
	return &vistaEfimeraTestimonioIdempotenciaBaremacion{testimonio: testimonio}
}

func (*vistaEfimeraTestimonioIdempotenciaBaremacion) String() string {
	return "[VISTA-EFIMERA-TESTIMONIO-IDEMPOTENCIA-BAREMACION-PROTEGIDA]"
}
func (*vistaEfimeraTestimonioIdempotenciaBaremacion) GoString() string {
	return "ports.vistaEfimeraTestimonioIdempotenciaBaremacion{[PROTEGIDA]}"
}
func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, v.String())
}
func (*vistaEfimeraTestimonioIdempotenciaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (*vistaEfimeraTestimonioIdempotenciaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (*vistaEfimeraTestimonioIdempotenciaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) LogValue() slog.Value {
	return slog.StringValue(v.String())
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) conTestimonio(
	visita func(testimonioAtomicoIdempotenciaBaremacion) error,
) error {
	v.mu.Lock()
	if v.cerrada || v.testimonio.validarEstructura() != nil {
		v.mu.Unlock()
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	v.visitasEnVuelo++
	clon, err := v.testimonio.clonar()
	v.mu.Unlock()
	if err != nil {
		v.mu.Lock()
		v.visitasEnVuelo--
		v.mu.Unlock()
		return err
	}
	defer func() {
		destruirTestimonioAtomicoIdempotenciaBaremacion(&clon)
		v.mu.Lock()
		v.visitasEnVuelo--
		v.mu.Unlock()
	}()
	return visita(clon)
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) marcarCobertura(marca uint16) {
	v.mu.Lock()
	if !v.cerrada {
		v.cobertura |= marca
	}
	v.mu.Unlock()
}

// cerrarYComprobarSinActividad nunca espera: marca la vista cerrada, destruye
// su copia original y devuelve false si el adaptador dejo un callback activo.
// Cada callback activo posee otra copia y la destruye en su defer al terminar.
func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) cerrarYComprobarSinActividad() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.cerrada {
		return false
	}
	v.cerrada = true
	destruirTestimonioAtomicoIdempotenciaBaremacion(&v.testimonio)
	return v.visitasEnVuelo == 0 && v.cobertura == coberturaCompletaVistaBaremacion
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarMaterialCanonicoAtestadoBaremacion(
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		material, err := t.representacionCanonicaAtestada()
		if err != nil {
			return err
		}
		return visitarCargaProtegidaEfimeraBaremacion(material, visita)
	})
	if err == nil {
		v.marcarCobertura(coberturaMaterialAtestadoBaremacion)
	}
	return err
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarResolucionIdentidadInternaEstableBaremacion(
	visita func(string, uint64, string, string, string, string, []byte) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		i := t.resolucionIdentidad
		atestacion := append([]byte(nil), i.ValorAtestacion...)
		defer borrarBytesBaremacion(atestacion)
		return visita(
			i.SnapshotRef, i.Revision, i.HuellaSHA256, i.FormatoAtestacion,
			i.EmisorAtestacionRef, i.ClaveAtestacionRef, atestacion,
		)
	})
	if err == nil {
		v.marcarCobertura(coberturaResolucionIdentidadBaremacion)
	}
	return err
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) ResumenLlaveroIdentidadesBaremacion() (
	referencia string, revision uint64, cantidad uint8, huella string, err error,
) {
	err = v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		referencia, revision, cantidad, huella = t.identidades.LlaveroRef,
			t.identidades.Revision, t.identidades.Cantidad, t.identidades.HuellaSHA256
		return nil
	})
	if err == nil {
		v.marcarCobertura(coberturaResumenIdentidadesBaremacion)
	}
	return
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarTopologiaIdentidadesBaremacion(
	visita func(int, uint16, uint32, string) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		for posicion, entrada := range t.identidades.Topologia {
			if err := visita(posicion, entrada.Version, entrada.GeneracionClave, entrada.ClaveHMACRef); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		v.marcarCobertura(coberturaTopologiaIdentidadesBaremacion)
	}
	return err
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) ResumenLlaveroIndicesBaremacion() (
	referencia string, revision uint64, cantidad uint8, huella string, err error,
) {
	err = v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		referencia, revision, cantidad, huella = t.indices.LlaveroRef,
			t.indices.Revision, t.indices.Cantidad, t.indices.HuellaSHA256
		return nil
	})
	if err == nil {
		v.marcarCobertura(coberturaResumenIndicesBaremacion)
	}
	return
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarTopologiaIndicesBaremacion(
	visita func(int, uint16, uint32, string) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		for posicion, entrada := range t.indices.Topologia {
			if err := visita(posicion, entrada.Version, entrada.GeneracionClave, entrada.ClaveHMACRef); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		v.marcarCobertura(coberturaTopologiaIndicesBaremacion)
	}
	return err
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarPrincipalesBaremacion(
	visita func(int, uint16, uint32, string, string) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		for posicion, principal := range t.principales {
			if err := visita(posicion, principal.Version, principal.GeneracionClave,
				principal.ClaveHMACRef, principal.ValorHMAC); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		v.marcarCobertura(coberturaPrincipalesBaremacion)
	}
	return err
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarMatrizIndicesBaremacion(
	visita func(int, int, uint16, uint32, string, string) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		for fila, indices := range t.matriz {
			for columna, indice := range indices {
				if err := visita(fila, columna, indice.Version, indice.GeneracionClave,
					indice.ClaveHMACRef, indice.ValorHMAC); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err == nil {
		v.marcarCobertura(coberturaMatrizBaremacion)
	}
	return err
}

// VisitarRepresentacionesCanonicasIntencionBaremacion es el puente nominal
// para que el futuro servicio privado pueda sellar, por cada celda de la
// matriz completa, el fingerprint semantico estable y conservar el sobre
// probatorio exacto. No devuelve indices, celdas ni capacidades seleccionables:
// las dos cargas solo viven dentro del callback y se borran al terminar.
//
// motivoEfimeroYaVerificado debe haber sido cotejado antes contra MotivoHMAC
// con la clave historica por la composicion privada. Este puerto no acredita
// esa verificacion, no ejecuta CAS ni concede efectos; mientras no exista
// internal/modules/bolsa/application/servicio_idempotencia_baremacion.go el
// flujo permanece expresamente NO-GO.
func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarRepresentacionesCanonicasIntencionBaremacion(
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	intencion IntencionCambioBaremacion,
	motivoEfimeroYaVerificado []byte,
	visita func(
		filaPrincipal, columnaIndice int,
		fingerprintSemantico, sobreProbatorio MaterialCanonicoEfimeroBaremacion,
	) error,
) error {
	if visita == nil || !intencionCambioBaremacionVinculadaSolicitud(intencion, solicitud) ||
		len(motivoEfimeroYaVerificado) == 0 || len(motivoEfimeroYaVerificado) > 8000 {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	bytesMotivo := append([]byte(nil), motivoEfimeroYaVerificado...)
	motivoCopia, err := NuevaCargaProtegida(bytesMotivo)
	borrarBytesBaremacion(bytesMotivo)
	if err != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	defer destruirCargaProtegidaBaremacion(&motivoCopia)

	err = v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		if subtle.ConstantTimeCompare(t.vinculoSolicitud[:], solicitud.vinculo[:]) != 1 {
			return ErrClaveIdempotenciaBaremacionInvalida
		}
		for fila, indices := range t.matriz {
			for columna, indice := range indices {
				fingerprint, err := intencion.representacionCanonicaFingerprintSemanticoParaHMAC(
					indice, motivoCopia,
				)
				if err != nil {
					return ErrClaveIdempotenciaBaremacionInvalida
				}
				sobre, err := intencion.representacionCanonicaSobreProbatorioParaHMAC(indice)
				if err != nil {
					destruirCargaProtegidaBaremacion(&fingerprint)
					return ErrClaveIdempotenciaBaremacionInvalida
				}
				errVisita := visitarCargaProtegidaEfimeraBaremacion(
					fingerprint,
					func(fingerprintEfimero MaterialCanonicoEfimeroBaremacion) error {
						return visitarCargaProtegidaEfimeraBaremacion(
							sobre,
							func(sobreEfimero MaterialCanonicoEfimeroBaremacion) error {
								return visita(fila, columna, fingerprintEfimero, sobreEfimero)
							},
						)
					},
				)
				if errVisita != nil {
					return errVisita
				}
			}
		}
		return nil
	})
	if err == nil {
		// La visita recorre la misma matriz completa, aunque entrega unicamente
		// sus representaciones efimeras y no los indices aislados.
		v.marcarCobertura(coberturaMatrizBaremacion)
	}
	return err
}

func (v *vistaEfimeraTestimonioIdempotenciaBaremacion) VisitarEvidenciaAtestacionBaremacion(
	visita func(string, string, string, uint64, string, []byte) error,
) error {
	if visita == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	err := v.conTestimonio(func(t testimonioAtomicoIdempotenciaBaremacion) error {
		e := t.evidencia
		copia := append([]byte(nil), e.Valor...)
		defer borrarBytesBaremacion(copia)
		return visita(e.Formato, e.EmisorRef, e.ClaveAtestacionRef, e.Revision,
			e.HuellaContenidoSHA256, copia)
	})
	if err == nil {
		v.marcarCobertura(coberturaEvidenciaBaremacion)
	}
	return err
}

// SolicitudResolverSeudonimoSujetoBaremacion solo identifica el expediente.
// La frontera resuelve el sujeto autoritativo desde datos internos; el cliente
// no puede presentar DNI, nombre ni una referencia de persona.
type SolicitudResolverSeudonimoSujetoBaremacion struct {
	procesoRef          string
	solicitudRef        string
	baremacionMeritoRef string
}

func NuevaSolicitudResolverSeudonimoSujetoBaremacion(
	procesoRef, solicitudRef, baremacionMeritoRef string,
) (SolicitudResolverSeudonimoSujetoBaremacion, error) {
	solicitud := SolicitudResolverSeudonimoSujetoBaremacion{
		procesoRef: procesoRef, solicitudRef: solicitudRef, baremacionMeritoRef: baremacionMeritoRef,
	}
	if solicitud.validar() != nil {
		return SolicitudResolverSeudonimoSujetoBaremacion{}, ErrSeudonimoSujetoBaremacionInvalido
	}
	return solicitud, nil
}

func (s SolicitudResolverSeudonimoSujetoBaremacion) validar() error {
	if !referenciaMaterialOpacaBaremacionValida(s.procesoRef, 512) ||
		!referenciaMaterialOpacaBaremacionValida(s.solicitudRef, 512) ||
		!referenciaMaterialOpacaBaremacionValida(s.baremacionMeritoRef, 512) {
		return ErrSeudonimoSujetoBaremacionInvalido
	}
	return nil
}

// VisitarReferencias entrega las referencias opacas solo durante la llamada al
// adaptador de identidad; evita getters y colecciones reutilizables.
func (s SolicitudResolverSeudonimoSujetoBaremacion) VisitarReferencias(
	visita func(procesoRef, solicitudRef, baremacionMeritoRef string) error,
) error {
	if visita == nil || s.validar() != nil {
		return ErrSeudonimoSujetoBaremacionInvalido
	}
	return visita(s.procesoRef, s.solicitudRef, s.baremacionMeritoRef)
}

func (SolicitudResolverSeudonimoSujetoBaremacion) String() string {
	return "[SOLICITUD-RESOLVER-SEUDONIMO-SUJETO-BAREMACION-PROTEGIDA]"
}
func (SolicitudResolverSeudonimoSujetoBaremacion) GoString() string {
	return "ports.SolicitudResolverSeudonimoSujetoBaremacion{[PROTEGIDA]}"
}
func (s SolicitudResolverSeudonimoSujetoBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (SolicitudResolverSeudonimoSujetoBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (SolicitudResolverSeudonimoSujetoBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (SolicitudResolverSeudonimoSujetoBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (s SolicitudResolverSeudonimoSujetoBaremacion) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// FronteraIdentidadesEstablesBaremacion queda limitada al seudonimo del sujeto.
// Los principales de idempotencia solo nacen en la operacion atomica combinada;
// no existe resolucion ni verificacion individual susceptible de TOCTOU.
type FronteraIdentidadesEstablesBaremacion interface {
	ResolverSeudonimoSujetoBaremacion(
		context.Context,
		SolicitudResolverSeudonimoSujetoBaremacion,
	) (SeudonimoSujetoBaremacionHMAC, error)
	VerificarSeudonimoSujetoBaremacion(
		context.Context,
		SolicitudResolverSeudonimoSujetoBaremacion,
		SeudonimoSujetoBaremacionHMAC,
	) error
}

type SolicitudDerivarHMACMotivoBaremacion struct {
	motivoClave string
	material    MaterialCanonicoEfimeroBaremacion
}

func (s SolicitudDerivarHMACMotivoBaremacion) Validar() error {
	if !claveCatalogoConfiguracionBaremacionValida(s.motivoClave) || s.material.Validar() != nil {
		return ErrHMACMotivoBaremacionInvalido
	}
	return nil
}

func (s SolicitudDerivarHMACMotivoBaremacion) MotivoClave() (string, error) {
	if s.Validar() != nil {
		return "", ErrHMACMotivoBaremacionInvalido
	}
	return s.motivoClave, nil
}

func (s SolicitudDerivarHMACMotivoBaremacion) VisitarMotivo(
	visita func([]byte) error,
) error {
	if s.Validar() != nil {
		return ErrHMACMotivoBaremacionInvalido
	}
	return s.material.VisitarBytes(visita)
}

// VisitarSolicitudDerivarHMACMotivoBaremacion evita almacenar el motivo en
// una solicitud copiable. La solicitud y todas sus copias quedan revocadas al
// volver; el adaptador criptografico debe consumir el motivo sincronicamente.
func VisitarSolicitudDerivarHMACMotivoBaremacion(
	motivoClave string,
	motivo []byte,
	visita func(SolicitudDerivarHMACMotivoBaremacion) error,
) error {
	if visita == nil || !claveCatalogoConfiguracionBaremacionValida(motivoClave) ||
		len(motivo) == 0 || len(motivo) > 8000 {
		return ErrHMACMotivoBaremacionInvalido
	}
	copia := append([]byte(nil), motivo...)
	defer borrarBytesBaremacion(copia)
	carga, err := NuevaCargaProtegida(copia)
	if err != nil {
		return ErrHMACMotivoBaremacionInvalido
	}
	return visitarCargaProtegidaEfimeraBaremacion(
		carga,
		func(material MaterialCanonicoEfimeroBaremacion) error {
			return visita(SolicitudDerivarHMACMotivoBaremacion{
				motivoClave: motivoClave, material: material,
			})
		},
	)
}

func (SolicitudDerivarHMACMotivoBaremacion) String() string {
	return "[SOLICITUD-DERIVAR-HMAC-MOTIVO-BAREMACION-PROTEGIDA]"
}
func (SolicitudDerivarHMACMotivoBaremacion) GoString() string {
	return "ports.SolicitudDerivarHMACMotivoBaremacion{[PROTEGIDA]}"
}
func (s SolicitudDerivarHMACMotivoBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (SolicitudDerivarHMACMotivoBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (SolicitudDerivarHMACMotivoBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (SolicitudDerivarHMACMotivoBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (s SolicitudDerivarHMACMotivoBaremacion) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

type SolicitudVerificarHMACMotivoBaremacion struct {
	Solicitud SolicitudDerivarHMACMotivoBaremacion
	Sello     HMACMotivoBaremacion
}

func (s SolicitudVerificarHMACMotivoBaremacion) Validar() error {
	if s.Solicitud.Validar() != nil || s.Sello.Validar() != nil {
		return ErrHMACMotivoBaremacionInvalido
	}
	return nil
}

func VisitarSolicitudVerificarHMACMotivoBaremacion(
	motivoClave string,
	motivo []byte,
	sello HMACMotivoBaremacion,
	visita func(SolicitudVerificarHMACMotivoBaremacion) error,
) error {
	if visita == nil || sello.Validar() != nil {
		return ErrHMACMotivoBaremacionInvalido
	}
	return VisitarSolicitudDerivarHMACMotivoBaremacion(
		motivoClave, motivo,
		func(solicitud SolicitudDerivarHMACMotivoBaremacion) error {
			return visita(SolicitudVerificarHMACMotivoBaremacion{
				Solicitud: solicitud, Sello: sello,
			})
		},
	)
}

func (SolicitudVerificarHMACMotivoBaremacion) String() string {
	return "[SOLICITUD-VERIFICAR-HMAC-MOTIVO-BAREMACION-PROTEGIDA]"
}
func (SolicitudVerificarHMACMotivoBaremacion) GoString() string {
	return "ports.SolicitudVerificarHMACMotivoBaremacion{[PROTEGIDA]}"
}
func (s SolicitudVerificarHMACMotivoBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (SolicitudVerificarHMACMotivoBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (SolicitudVerificarHMACMotivoBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (SolicitudVerificarHMACMotivoBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (s SolicitudVerificarHMACMotivoBaremacion) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// CriptografiaMotivoBaremacion impide aceptar una cadena hexadecimal como
// prueba del motivo. La derivacion y verificacion usan un dominio y llavero
// distintos de sujeto, principal, indice e intencion.
type CriptografiaMotivoBaremacion interface {
	DerivarHMACMotivoBaremacion(
		context.Context,
		SolicitudDerivarHMACMotivoBaremacion,
	) (HMACMotivoBaremacion, error)
	VerificarHMACMotivoBaremacion(
		context.Context,
		SolicitudVerificarHMACMotivoBaremacion,
	) error
}

// El sellado, la verificacion, la reserva y cualquier efecto quedan
// deliberadamente fuera de este puerto nominal. Solo el futuro servicio
// internal/modules/bolsa/application/servicio_idempotencia_baremacion.go puede
// recibir el producto completo, promoverlo con dependencias privadas y crear
// solicitudes de efecto no construibles desde handlers o adaptadores.

// VerificadorMaterialEstableIntencionBaremacion, cuando esta fijado como
// dependencia privada de la composicion TCB homologada, debe recuperar y
// comprobar la representacion V2 del plan durable de firma, manifiesto y
// recibos. Tambien debe resolver historicamente ambas instantaneas de catalogo
// por referencia+version+huella y acreditar:
//   - formato clave <-> MIME canonico <-> perfil/politica de firma <-> conector;
//   - clasificacion <-> reglas de custodia y politica de retencion.
//
// La interfaz es sustituible y no es autoridad por si sola. No existe aun el
// servicio de aplicacion con dependencias privadas ni productor V2 homologado:
// hasta implementarlos y superar sus pruebas, el flujo permanece en NO-GO.
type VerificadorMaterialEstableIntencionBaremacion interface {
	VerificarMaterialEstableIntencionBaremacion(
		context.Context,
		IntencionCambioBaremacion,
	) error
}

// EstadoOperacionIdempotenteBaremacion es un catalogo cerrado. Ausente no es
// una fila persistida: es la proyeccion fail-closed de una busqueda sin
// resultado. En curso nunca autoriza a repetir efectos ya iniciados.
type EstadoOperacionIdempotenteBaremacion string

const (
	EstadoOperacionIdempotenteAusente    EstadoOperacionIdempotenteBaremacion = "ausente"
	EstadoOperacionIdempotenteEnCurso    EstadoOperacionIdempotenteBaremacion = "en_curso"
	EstadoOperacionIdempotenteConfirmada EstadoOperacionIdempotenteBaremacion = "confirmada"
)

func (e EstadoOperacionIdempotenteBaremacion) Valido() bool {
	switch e {
	case EstadoOperacionIdempotenteAusente, EstadoOperacionIdempotenteEnCurso,
		EstadoOperacionIdempotenteConfirmada:
		return true
	default:
		return false
	}
}

// IntencionCambioBaremacion es la proyeccion material estable de una
// incorporacion firmada. Excluye deliberadamente contexto, autenticacion,
// sesion, autorizacion del intento, tokens, correlaciones, auditoria, outbox y
// tiempos del intento. Motivo se compromete mediante HMAC de dominio propio
// para no duplicar texto que pueda contener datos personales. Los campos V2
// de manifiesto y recibos son puertas cerradas: Validar comprueba forma, pero
// solo la composicion TCB con su verificador material privado puede confiar en
// el contenido y en las instantaneas historicas de catalogo.
//
// No contiene mapas, punteros ni colecciones: una copia del valor no comparte
// memoria mutable con el original.
type IntencionCambioBaremacion struct {
	Version uint16
	Clase   ClaseCambioBaremacion

	ProcesoRef          string
	SolicitudRef        string
	SujetoSeudonimoHMAC SeudonimoSujetoBaremacionHMAC
	BaremacionMeritoRef string
	VersionBase         ReferenciaVersionBaremacion
	VersionObjetivo     uint64

	DecisionRef                          string
	NumeroDecision                       uint64
	ClaseDecision                        dominiobolsa.ClaseDecisionTecnica
	ResultadoDecision                    dominiobolsa.ResultadoDecisionTecnica
	HuellaContenidoDecisionSHA256        string
	HuellaEstadoResultanteDecisionSHA256 string

	PoliticaFirmaRef             string
	PoliticaFirmaVersion         uint32
	HuellaPoliticaFirmaSHA256    string
	EsquemaPlanFirmaDurable      EsquemaMaterialEstableBaremacion
	VersionPlanFirmaDurable      uint16
	PlanFirmaDurableRef          string
	HuellaPlanFirmaDurableSHA256 string
	EstadoPlanFirmaDurable       EstadoPlanFirmaDurableBaremacion

	DocumentoFirmableRef          string
	VersionDocumentoFirmable      string
	HuellaDocumentoFirmableSHA256 string
	FirmaRef                      string
	HuellaFirmaSHA256             string
	DocumentoFirmadoRef           string
	HuellaDocumentoFirmadoSHA256  string

	EsquemaManifiestoProbatorio      EsquemaMaterialEstableBaremacion
	VersionManifiestoProbatorio      uint16
	ManifiestoProbatorioRef          string
	HuellaManifiestoProbatorioSHA256 string
	SelloManifiestoProbatorioHMAC    HMACManifiestoMaterialBaremacionV2

	ObjetoCustodiadoRef               string
	VersionObjetoCustodiado           string
	ConectorCustodiaID                string
	ZonaCustodia                      puertosvec.ZonaAlmacen
	HuellaObjetoCustodiadoSHA256      string
	FormatoDocumento                  InstantaneaCatalogoFormatoDocumentoBaremacion
	ClasificacionDocumento            InstantaneaCatalogoClasificacionDocumentoBaremacion
	TamanoDocumentoFirmado            uint64
	EstadoInmovilizacionObjeto        EstadoInmovilizacionObjetoBaremacion
	EstadoDisponibilidadObjeto        EstadoDisponibilidadObjetoBaremacion
	EsquemaEvidenciaRecuperacion      EsquemaMaterialEstableBaremacion
	VersionEvidenciaRecuperacion      uint16
	EvidenciaRecuperacionFirmadoRef   string
	HuellaEvidenciaRecuperacionSHA256 string
	EsquemaEvidenciaCustodia          EsquemaMaterialEstableBaremacion
	VersionEvidenciaCustodia          uint16
	EvidenciaCustodiaFirmadoRef       string
	HuellaEvidenciaCustodiaSHA256     string
	EsquemaEvidenciaRetencion         EsquemaMaterialEstableBaremacion
	VersionEvidenciaRetencion         uint16
	EvidenciaRetencionFirmadoRef      string
	HuellaEvidenciaRetencionSHA256    string
	PoliticaRetencionRef              string
	PoliticaRetencionVersion          uint32
	HuellaPoliticaRetencionSHA256     string
	RetenidoHasta                     time.Time

	HuellaAgregadoObjetivoSHA256 string
	MotivoClave                  string
	MotivoHMAC                   HMACMotivoBaremacion
}

func (i IntencionCambioBaremacion) Validar() error {
	if i.Version != VersionIntencionCambioBaremacionV1 ||
		i.Clase != ClaseCambioIncorporarDecision ||
		!referenciaMaterialOpacaBaremacionValida(i.ProcesoRef, 512) ||
		!referenciaMaterialOpacaBaremacionValida(i.SolicitudRef, 512) ||
		i.SujetoSeudonimoHMAC.Validar() != nil ||
		!referenciaMaterialOpacaBaremacionValida(i.BaremacionMeritoRef, 512) ||
		i.VersionBase.Validar() != nil || i.VersionBase.BaremacionMeritoRef != i.BaremacionMeritoRef ||
		!referenciaMaterialOpacaBaremacionValida(i.VersionBase.BaremacionMeritoRef, 512) ||
		i.VersionBase.Numero >= maximoVersionBaremacionIntencion ||
		i.VersionObjetivo != i.VersionBase.Numero+1 || i.VersionObjetivo > maximoVersionBaremacionIntencion ||
		!referenciaMaterialOpacaBaremacionValida(i.DecisionRef, 512) || i.NumeroDecision < 1 ||
		i.NumeroDecision != i.VersionBase.Numero || !i.ClaseDecision.Valida() ||
		!i.ResultadoDecision.Valido() || !huellaSHA256Valida(i.HuellaContenidoDecisionSHA256) ||
		!huellaSHA256Valida(i.HuellaEstadoResultanteDecisionSHA256) ||
		!claveCatalogoConfiguracionBaremacionValida(i.PoliticaFirmaRef) || i.PoliticaFirmaVersion < 1 ||
		i.PoliticaFirmaVersion > maximoVersionPoliticaFirmaIntencion ||
		!huellaSHA256Valida(i.HuellaPoliticaFirmaSHA256) ||
		i.EsquemaPlanFirmaDurable != EsquemaPlanFirmaDurableBaremacionV2 ||
		i.VersionPlanFirmaDurable != VersionPlanFirmaDurableBaremacionV2 ||
		!referenciaMaterialOpacaBaremacionValida(i.PlanFirmaDurableRef, 512) ||
		!huellaSHA256Valida(i.HuellaPlanFirmaDurableSHA256) || !i.EstadoPlanFirmaDurable.Valido() ||
		!referenciaMaterialOpacaBaremacionValida(i.DocumentoFirmableRef, 512) ||
		!versionLogicaMaterialBaremacionValida(i.VersionDocumentoFirmable) ||
		!huellaSHA256Valida(i.HuellaDocumentoFirmableSHA256) ||
		!referenciaMaterialOpacaBaremacionValida(i.FirmaRef, 512) || !huellaSHA256Valida(i.HuellaFirmaSHA256) ||
		!referenciaMaterialOpacaBaremacionValida(i.DocumentoFirmadoRef, 512) ||
		!huellaSHA256Valida(i.HuellaDocumentoFirmadoSHA256) ||
		i.EsquemaManifiestoProbatorio != EsquemaManifiestoMaterialEstableBaremacionV2 ||
		i.VersionManifiestoProbatorio != VersionManifiestoMaterialEstableBaremacionV2 ||
		!referenciaMaterialOpacaBaremacionValida(i.ManifiestoProbatorioRef, 512) ||
		!huellaSHA256Valida(i.HuellaManifiestoProbatorioSHA256) ||
		i.SelloManifiestoProbatorioHMAC.Validar() != nil ||
		!referenciaMaterialOpacaBaremacionValida(i.ObjetoCustodiadoRef, 512) ||
		!versionLogicaMaterialBaremacionValida(i.VersionObjetoCustodiado) ||
		!claveCatalogoConfiguracionBaremacionValida(i.ConectorCustodiaID) ||
		i.ZonaCustodia != puertosvec.ZonaAlmacenAdmitida ||
		!huellaSHA256Valida(i.HuellaObjetoCustodiadoSHA256) ||
		i.FormatoDocumento.Validar() != nil || i.ClasificacionDocumento.Validar() != nil ||
		i.TamanoDocumentoFirmado < 1 || i.TamanoDocumentoFirmado > uint64(maximoCargaProtegida) ||
		!i.EstadoInmovilizacionObjeto.Valido() || !i.EstadoDisponibilidadObjeto.Valido() ||
		i.EsquemaEvidenciaRecuperacion != EsquemaReciboRecuperacionBaremacionV2 ||
		i.VersionEvidenciaRecuperacion != VersionReciboRecuperacionBaremacionV2 ||
		!referenciaMaterialOpacaBaremacionValida(i.EvidenciaRecuperacionFirmadoRef, 512) ||
		!huellaSHA256Valida(i.HuellaEvidenciaRecuperacionSHA256) ||
		i.EsquemaEvidenciaCustodia != EsquemaReciboCustodiaBaremacionV2 ||
		i.VersionEvidenciaCustodia != VersionReciboCustodiaBaremacionV2 ||
		!referenciaMaterialOpacaBaremacionValida(i.EvidenciaCustodiaFirmadoRef, 512) ||
		!huellaSHA256Valida(i.HuellaEvidenciaCustodiaSHA256) ||
		i.EsquemaEvidenciaRetencion != EsquemaReciboRetencionBaremacionV2 ||
		i.VersionEvidenciaRetencion != VersionReciboRetencionBaremacionV2 ||
		!referenciaMaterialOpacaBaremacionValida(i.EvidenciaRetencionFirmadoRef, 512) ||
		!huellaSHA256Valida(i.HuellaEvidenciaRetencionSHA256) ||
		!claveCatalogoConfiguracionBaremacionValida(i.PoliticaRetencionRef) || i.PoliticaRetencionVersion < 1 ||
		i.PoliticaRetencionVersion > maximoVersionPoliticaRetencion ||
		!huellaSHA256Valida(i.HuellaPoliticaRetencionSHA256) || i.RetenidoHasta.IsZero() ||
		i.RetenidoHasta.Location() != time.UTC || i.RetenidoHasta.Year() < 1970 || i.RetenidoHasta.Year() > 9999 ||
		i.RetenidoHasta.Nanosecond()%1000 != 0 ||
		!huellaSHA256Valida(i.HuellaAgregadoObjetivoSHA256) ||
		i.HuellaObjetoCustodiadoSHA256 != i.HuellaDocumentoFirmadoSHA256 ||
		i.HuellaEvidenciaRecuperacionSHA256 == i.HuellaDocumentoFirmadoSHA256 ||
		i.HuellaEvidenciaCustodiaSHA256 == i.HuellaDocumentoFirmadoSHA256 ||
		i.HuellaEvidenciaRetencionSHA256 == i.HuellaDocumentoFirmadoSHA256 ||
		i.HuellaEvidenciaRecuperacionSHA256 == i.HuellaEvidenciaCustodiaSHA256 ||
		i.HuellaEvidenciaRecuperacionSHA256 == i.HuellaEvidenciaRetencionSHA256 ||
		i.HuellaEvidenciaCustodiaSHA256 == i.HuellaEvidenciaRetencionSHA256 ||
		!claveCatalogoConfiguracionBaremacionValida(i.MotivoClave) || i.MotivoHMAC.Validar() != nil ||
		i.SujetoSeudonimoHMAC.ClaveHMACRef == i.MotivoHMAC.ClaveHMACRef {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (i IntencionCambioBaremacion) Clonar() (IntencionCambioBaremacion, error) {
	if err := i.Validar(); err != nil {
		return IntencionCambioBaremacion{}, err
	}
	clon := i
	clon.RetenidoHasta = i.RetenidoHasta.UTC()
	return clon, nil
}

func intencionCambioBaremacionVinculadaSolicitud(
	intencion IntencionCambioBaremacion,
	solicitud SolicitudTestimonioAtomicoIdempotenciaBaremacion,
) bool {
	return intencion.Validar() == nil && solicitud.validar() == nil &&
		intencion.Clase == solicitud.clase &&
		intencion.ProcesoRef == solicitud.ambitoSujeto.procesoRef &&
		intencion.SolicitudRef == solicitud.ambitoSujeto.solicitudRef &&
		intencion.BaremacionMeritoRef == solicitud.ambitoSujeto.baremacionMeritoRef &&
		intencion.SujetoSeudonimoHMAC.IgualConstante(solicitud.seudonimo)
}

func (IntencionCambioBaremacion) String() string {
	return "[INTENCION-CAMBIO-BAREMACION-PROTEGIDA]"
}
func (IntencionCambioBaremacion) GoString() string {
	return "ports.IntencionCambioBaremacion{[PROTEGIDA]}"
}
func (i IntencionCambioBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, i.String())
}
func (IntencionCambioBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (IntencionCambioBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (IntencionCambioBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (i IntencionCambioBaremacion) LogValue() slog.Value {
	return slog.StringValue(i.String())
}

// representacionCanonicaSobreProbatorioParaHMAC conserva todos los sobres
// exactos para firma/auditoria del intento. Nunca debe emplearse como
// fingerprint de conflicto: sus HMAC cambian al rotar sujeto, manifiesto o
// motivo aunque el contenido de negocio sea identico.
func (i IntencionCambioBaremacion) representacionCanonicaSobreProbatorioParaHMAC(
	indice indiceIdempotenciaBaremacion,
) (CargaProtegida, error) {
	if err := i.Validar(); err != nil {
		return CargaProtegida{}, err
	}
	if err := indice.validar(); err != nil {
		return CargaProtegida{}, err
	}
	if indice.ClaveHMACRef == i.SujetoSeudonimoHMAC.ClaveHMACRef ||
		indice.ClaveHMACRef == i.MotivoHMAC.ClaveHMACRef {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}

	material := make([]byte, 0, len(esquemaCanonicoIntencionCambioBaremacionV1)+2048)
	defer func() { borrarBytesBaremacion(material) }()
	material = anexarCampoCanonicoTextoIntencion(
		material, 0, esquemaCanonicoIntencionCambioBaremacionV1,
	)
	etiqueta := uint16(1)
	anexar := func(valor []byte) {
		material = anexarCampoCanonicoIntencion(material, etiqueta, valor)
		borrarBytesBaremacion(valor)
		etiqueta++
	}
	anexarTexto := func(valor string) {
		material = anexarCampoCanonicoTextoIntencion(material, etiqueta, valor)
		etiqueta++
	}
	anexarHuella := func(valor string) {
		temporal := decodificarHexCanonicoIntencion(valor)
		material = anexarCampoCanonicoIntencion(material, etiqueta, temporal)
		borrarBytesBaremacion(temporal)
		etiqueta++
	}

	// Etiquetas 1-4: sobre versionado del indice estable. No son campos de la
	// intencion y permiten localizar una operacion creada con una clave rotada.
	anexar(canonicoUint16Intencion(indice.Version))
	anexar(canonicoUint32Intencion(indice.GeneracionClave))
	anexarTexto(indice.ClaveHMACRef)
	anexarHuella(indice.ValorHMAC)
	// Etiqueta 5: version del esquema de IntencionCambioBaremacion. Aunque hoy
	// ambas versiones valgan 1, sus dominios y posiciones son independientes.
	anexar(canonicoUint16Intencion(i.Version))
	anexarTexto(string(i.Clase))
	anexarTexto(i.ProcesoRef)
	anexarTexto(i.SolicitudRef)
	anexar(canonicoUint16Intencion(i.SujetoSeudonimoHMAC.Version))
	anexarTexto(i.SujetoSeudonimoHMAC.ClaveHMACRef)
	anexarHuella(i.SujetoSeudonimoHMAC.ValorHMAC)
	anexarTexto(i.BaremacionMeritoRef)
	anexarTexto(i.VersionBase.BaremacionMeritoRef)
	anexar(canonicoUint64Intencion(i.VersionBase.Numero))
	anexarHuella(i.VersionBase.HuellaEstadoSHA256)
	anexar(canonicoUint64Intencion(i.VersionObjetivo))
	anexarTexto(i.DecisionRef)
	anexar(canonicoUint64Intencion(i.NumeroDecision))
	anexarTexto(string(i.ClaseDecision))
	anexarTexto(string(i.ResultadoDecision))
	anexarHuella(i.HuellaContenidoDecisionSHA256)
	anexarHuella(i.HuellaEstadoResultanteDecisionSHA256)
	anexarTexto(i.PoliticaFirmaRef)
	anexar(canonicoUint32Intencion(i.PoliticaFirmaVersion))
	anexarHuella(i.HuellaPoliticaFirmaSHA256)
	anexarTexto(string(i.EsquemaPlanFirmaDurable))
	anexar(canonicoUint16Intencion(i.VersionPlanFirmaDurable))
	anexarTexto(i.PlanFirmaDurableRef)
	anexarHuella(i.HuellaPlanFirmaDurableSHA256)
	anexarTexto(string(i.EstadoPlanFirmaDurable))
	anexarTexto(i.DocumentoFirmableRef)
	anexarTexto(i.VersionDocumentoFirmable)
	anexarHuella(i.HuellaDocumentoFirmableSHA256)
	anexarTexto(i.FirmaRef)
	anexarHuella(i.HuellaFirmaSHA256)
	anexarTexto(i.DocumentoFirmadoRef)
	anexarHuella(i.HuellaDocumentoFirmadoSHA256)
	anexarTexto(string(i.EsquemaManifiestoProbatorio))
	anexar(canonicoUint16Intencion(i.VersionManifiestoProbatorio))
	anexarTexto(i.ManifiestoProbatorioRef)
	anexarHuella(i.HuellaManifiestoProbatorioSHA256)
	anexar(canonicoUint16Intencion(i.SelloManifiestoProbatorioHMAC.Version))
	anexarTexto(i.SelloManifiestoProbatorioHMAC.ClaveHMACRef)
	anexarHuella(i.SelloManifiestoProbatorioHMAC.ValorHMAC)
	anexarTexto(i.ObjetoCustodiadoRef)
	anexarTexto(i.VersionObjetoCustodiado)
	anexarTexto(i.ConectorCustodiaID)
	anexarTexto(string(i.ZonaCustodia))
	anexarHuella(i.HuellaObjetoCustodiadoSHA256)
	anexarTexto(i.FormatoDocumento.CatalogoRef)
	anexar(canonicoUint32Intencion(i.FormatoDocumento.CatalogoVersion))
	anexarHuella(i.FormatoDocumento.HuellaCatalogoSHA256)
	anexarTexto(string(i.FormatoDocumento.FormatoClave))
	anexarTexto(string(i.FormatoDocumento.MIMECanonico))
	anexarTexto(i.ClasificacionDocumento.CatalogoRef)
	anexar(canonicoUint32Intencion(i.ClasificacionDocumento.CatalogoVersion))
	anexarHuella(i.ClasificacionDocumento.HuellaCatalogoSHA256)
	anexarTexto(string(i.ClasificacionDocumento.ClasificacionClave))
	anexar(canonicoUint64Intencion(i.TamanoDocumentoFirmado))
	anexarTexto(string(i.EstadoInmovilizacionObjeto))
	anexarTexto(string(i.EstadoDisponibilidadObjeto))
	anexarTexto(string(i.EsquemaEvidenciaRecuperacion))
	anexar(canonicoUint16Intencion(i.VersionEvidenciaRecuperacion))
	anexarTexto(i.EvidenciaRecuperacionFirmadoRef)
	anexarHuella(i.HuellaEvidenciaRecuperacionSHA256)
	anexarTexto(string(i.EsquemaEvidenciaCustodia))
	anexar(canonicoUint16Intencion(i.VersionEvidenciaCustodia))
	anexarTexto(i.EvidenciaCustodiaFirmadoRef)
	anexarHuella(i.HuellaEvidenciaCustodiaSHA256)
	anexarTexto(string(i.EsquemaEvidenciaRetencion))
	anexar(canonicoUint16Intencion(i.VersionEvidenciaRetencion))
	anexarTexto(i.EvidenciaRetencionFirmadoRef)
	anexarHuella(i.HuellaEvidenciaRetencionSHA256)
	anexarTexto(i.PoliticaRetencionRef)
	anexar(canonicoUint32Intencion(i.PoliticaRetencionVersion))
	anexarHuella(i.HuellaPoliticaRetencionSHA256)
	anexar(canonicoTiempoIntencion(i.RetenidoHasta))
	anexarHuella(i.HuellaAgregadoObjetivoSHA256)
	anexarTexto(i.MotivoClave)
	anexar(canonicoUint16Intencion(i.MotivoHMAC.Version))
	anexarTexto(i.MotivoHMAC.ClaveHMACRef)
	anexarHuella(i.MotivoHMAC.ValorHMAC)
	if etiqueta != 83 {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	return NuevaCargaProtegida(material)
}

// representacionCanonicaFingerprintSemanticoParaHMAC produce la unica
// preimagen apta para comparar reintentos. Excluye los sobres rotatorios de
// seudonimo, manifiesto y motivo, pero conserva sus referencias y huellas de
// contenido estables. El motivo claro se incorpora solo de forma efimera dentro
// de esta preimagen; antes de llamar, la composicion privada debe verificarlo
// contra MotivoHMAC usando su clave historica. Nunca se persiste ni se publica.
//
// Mientras no exista servicio_idempotencia_baremacion.go que imponga esa
// verificacion y almacene por separado el sobre probatorio exacto, el flujo de
// efectos sigue NO-GO.
func (i IntencionCambioBaremacion) representacionCanonicaFingerprintSemanticoParaHMAC(
	indice indiceIdempotenciaBaremacion,
	motivoEfimeroYaVerificado CargaProtegida,
) (CargaProtegida, error) {
	if motivoEfimeroYaVerificado.Validar() != nil || motivoEfimeroYaVerificado.Tamano() > 8000 {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	const referenciaExcluida = "sobre-rotatorio-excluido-fingerprint"
	huellaExcluida := strings.Repeat("0", sha256.Size*2)
	normalizada := i
	normalizada.SujetoSeudonimoHMAC = SeudonimoSujetoBaremacionHMAC{
		Version:      VersionSeudonimoSujetoBaremacionV1,
		ClaveHMACRef: referenciaExcluida + "-sujeto",
		ValorHMAC:    huellaExcluida,
	}
	normalizada.SelloManifiestoProbatorioHMAC = HMACManifiestoMaterialBaremacionV2{
		Version:      VersionHMACManifiestoMaterialBaremacionV2,
		ClaveHMACRef: referenciaExcluida + "-manifiesto",
		ValorHMAC:    huellaExcluida,
	}
	normalizada.MotivoHMAC = HMACMotivoBaremacion{
		Version:      VersionHMACMotivoBaremacionV1,
		ClaveHMACRef: referenciaExcluida + "-motivo",
		ValorHMAC:    huellaExcluida,
	}
	sobreNormalizado, err := normalizada.representacionCanonicaSobreProbatorioParaHMAC(indice)
	if err != nil {
		return CargaProtegida{}, err
	}
	defer destruirCargaProtegidaBaremacion(&sobreNormalizado)
	bytesSobre := sobreNormalizado.Revelar()
	defer borrarBytesBaremacion(bytesSobre)
	if len(bytesSobre) < 10 {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	longitudEsquema := binary.BigEndian.Uint64(bytesSobre[2:10])
	if binary.BigEndian.Uint16(bytesSobre[:2]) != 0 ||
		longitudEsquema > uint64(len(bytesSobre)-10) {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	desplazamiento := 10 + int(longitudEsquema)
	motivo := motivoEfimeroYaVerificado.Revelar()
	defer borrarBytesBaremacion(motivo)
	material := make([]byte, 0, len(bytesSobre)+len(motivo)+64)
	defer func() { borrarBytesBaremacion(material) }()
	material = anexarCampoCanonicoTextoIntencion(
		material, 0, esquemaFingerprintSemanticoIntencionBaremacionV1,
	)
	material = anexarBytesAPropietarioCanonicoIntencion(
		material, bytesSobre[desplazamiento:],
	)
	material = anexarCampoCanonicoIntencion(material, 83, motivo)
	return NuevaCargaProtegida(material)
}

// anexarCampoCanonicoIntencion solo admite un destino local propietario. El
// valor puede ser prestado e incluso apuntar al backing del propio destino.
func anexarCampoCanonicoIntencion(destino []byte, etiqueta uint16, valor []byte) []byte {
	var cabecera [10]byte
	binary.BigEndian.PutUint16(cabecera[:2], etiqueta)
	binary.BigEndian.PutUint64(cabecera[2:], uint64(len(valor)))
	if len(valor) > int(^uint(0)>>1)-len(cabecera) {
		panic("bolsa: desbordamiento de material canonico")
	}
	adicional := len(cabecera) + len(valor)
	if adicional > int(^uint(0)>>1)-len(destino) {
		panic("bolsa: desbordamiento de material canonico")
	}
	necesaria := len(destino) + adicional
	if necesaria <= cap(destino) {
		inicio := len(destino)
		destino = destino[:necesaria]
		copy(destino[inicio:inicio+len(cabecera)], cabecera[:])
		copy(destino[inicio+len(cabecera):], valor)
		return destino
	}
	nuevaCapacidad := cap(destino) * 2
	if nuevaCapacidad < necesaria || nuevaCapacidad < cap(destino) {
		nuevaCapacidad = necesaria
	}
	inicio := len(destino)
	nuevo := make([]byte, necesaria, nuevaCapacidad)
	copy(nuevo[:inicio], destino)
	copy(nuevo[inicio:inicio+len(cabecera)], cabecera[:])
	// valor puede ser una vista del backing de destino. Debe copiarse antes de
	// borrar ese backing; este orden forma parte del contrato de crecimiento.
	copy(nuevo[inicio+len(cabecera):], valor)
	if cap(destino) > 0 {
		borrarBytesBaremacion(destino[:cap(destino)])
	}
	return nuevo
}

// anexarCampoCanonicoTextoIntencion copia directamente desde string al
// backing canonico: evita crear otro []byte propietario que quedaria hasta GC.
func anexarCampoCanonicoTextoIntencion(destino []byte, etiqueta uint16, valor string) []byte {
	var cabecera [10]byte
	binary.BigEndian.PutUint16(cabecera[:2], etiqueta)
	binary.BigEndian.PutUint64(cabecera[2:], uint64(len(valor)))
	if len(valor) > int(^uint(0)>>1)-len(cabecera) {
		panic("bolsa: desbordamiento de material canonico")
	}
	adicional := len(cabecera) + len(valor)
	if adicional > int(^uint(0)>>1)-len(destino) {
		panic("bolsa: desbordamiento de material canonico")
	}
	necesaria := len(destino) + adicional
	if necesaria <= cap(destino) {
		inicio := len(destino)
		destino = destino[:necesaria]
		copy(destino[inicio:inicio+len(cabecera)], cabecera[:])
		copy(destino[inicio+len(cabecera):], valor)
		return destino
	}
	nuevaCapacidad := cap(destino) * 2
	if nuevaCapacidad < necesaria || nuevaCapacidad < cap(destino) {
		nuevaCapacidad = necesaria
	}
	inicio := len(destino)
	nuevo := make([]byte, necesaria, nuevaCapacidad)
	copy(nuevo[:inicio], destino)
	copy(nuevo[inicio:inicio+len(cabecera)], cabecera[:])
	copy(nuevo[inicio+len(cabecera):], valor)
	if cap(destino) > 0 {
		borrarBytesBaremacion(destino[:cap(destino)])
	}
	return nuevo
}

// anexarBytesAPropietarioCanonicoIntencion solo admite un destino local
// propietario. En crecimiento copia tambien valor antes de borrar el backing
// anterior, porque valor puede ser una vista de ese mismo backing.
func anexarBytesAPropietarioCanonicoIntencion(destino, valor []byte) []byte {
	if len(valor) > int(^uint(0)>>1)-len(destino) {
		panic("bolsa: desbordamiento de material canonico")
	}
	necesaria := len(destino) + len(valor)
	if necesaria <= cap(destino) {
		inicio := len(destino)
		destino = destino[:necesaria]
		copy(destino[inicio:], valor)
		return destino
	}
	nuevaCapacidad := cap(destino) * 2
	if nuevaCapacidad < necesaria || nuevaCapacidad < cap(destino) {
		nuevaCapacidad = necesaria
	}
	nuevo := make([]byte, necesaria, nuevaCapacidad)
	copy(nuevo[:len(destino)], destino)
	copy(nuevo[len(destino):], valor)
	if cap(destino) > 0 {
		borrarBytesBaremacion(destino[:cap(destino)])
	}
	return nuevo
}

func canonicoUint16Intencion(valor uint16) []byte {
	resultado := make([]byte, 2)
	binary.BigEndian.PutUint16(resultado, valor)
	return resultado
}

func canonicoUint32Intencion(valor uint32) []byte {
	resultado := make([]byte, 4)
	binary.BigEndian.PutUint32(resultado, valor)
	return resultado
}

func canonicoUint64Intencion(valor uint64) []byte {
	resultado := make([]byte, 8)
	binary.BigEndian.PutUint64(resultado, valor)
	return resultado
}

func canonicoTiempoIntencion(instante time.Time) []byte {
	resultado := make([]byte, 8)
	binary.BigEndian.PutUint64(resultado, uint64(instante.UnixMicro()))
	return resultado
}

func decodificarHexCanonicoIntencion(valor string) []byte {
	resultado, _ := hex.DecodeString(valor)
	return resultado
}

// Las claves de catalogo/configuracion usan una gramatica nominal cerrada del
// nucleo. No se pretende inferir PII por heuristicas: estos campos nunca deben
// transportar referencias de entidad ni texto aportado por una persona.
func claveCatalogoConfiguracionBaremacionValida(valor string) bool {
	return claveValida(valor)
}

// Las versiones logicas son claves nominales, no referencias de entidad.
func versionLogicaMaterialBaremacionValida(valor string) bool {
	return len(valor) <= 64 && claveCatalogoConfiguracionBaremacionValida(valor)
}

// mimeCanonicoDocumentoBaremacionValido implementa la gramatica positiva de
// restricted-name de RFC 6838 para type/subtype. Cada componente queda
// acotado a 127 octetos, empieza y termina en alfanumerico lowercase y solo
// admite los signos registrados en posiciones interiores.
func mimeCanonicoDocumentoBaremacionValido(valor string) bool {
	if len(valor) < 3 || len(valor) > 255 {
		return false
	}
	separador := -1
	for posicion := 0; posicion < len(valor); posicion++ {
		caracter := valor[posicion]
		if caracter == '/' {
			if separador >= 0 {
				return false
			}
			separador = posicion
		}
	}
	if separador < 1 || separador >= len(valor)-1 {
		return false
	}
	return componenteMIMECanonicoBaremacionValido(valor[:separador]) &&
		componenteMIMECanonicoBaremacionValido(valor[separador+1:])
}

func componenteMIMECanonicoBaremacionValido(componente string) bool {
	if len(componente) < 1 || len(componente) > 127 ||
		!alfanumericoMIMEMinusculaBaremacion(componente[0]) ||
		!alfanumericoMIMEMinusculaBaremacion(componente[len(componente)-1]) {
		return false
	}
	for posicion := 0; posicion < len(componente); posicion++ {
		caracter := componente[posicion]
		if alfanumericoMIMEMinusculaBaremacion(caracter) {
			continue
		}
		switch caracter {
		case '!', '#', '$', '&', '-', '^', '_', '.', '+':
			continue
		default:
			return false
		}
	}
	return true
}

func alfanumericoMIMEMinusculaBaremacion(caracter byte) bool {
	return (caracter >= 'a' && caracter <= 'z') || (caracter >= '0' && caracter <= '9')
}

// Las referencias materiales solo aceptan UUIDv4 canonico lowercase. Esta
// forma reduce el canal accidental de PII pero no acredita origen ni entropia;
// esa garantia corresponde a la composicion TCB y su verificador privado.
func referenciaMaterialOpacaBaremacionValida(valor string, maximo int) bool {
	if len(valor) != 36 || len(valor) > maximo || valor[8] != '-' || valor[13] != '-' ||
		valor[18] != '-' || valor[23] != '-' || valor[14] != '4' {
		return false
	}
	variante := valor[19]
	if variante != '8' && variante != '9' && variante != 'a' && variante != 'b' {
		return false
	}
	for posicion := 0; posicion < len(valor); posicion++ {
		caracter := valor[posicion]
		if posicion == 8 || posicion == 13 || posicion == 18 || posicion == 23 {
			continue
		}
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

func decodificarUUIDv4CanonicoBaremacion(valor string) ([]byte, bool) {
	if !referenciaMaterialOpacaBaremacionValida(valor, 36) {
		return nil, false
	}
	compacto := make([]byte, 0, 32)
	defer borrarBytesBaremacion(compacto)
	for posicion := 0; posicion < len(valor); posicion++ {
		if valor[posicion] != '-' {
			compacto = append(compacto, valor[posicion])
		}
	}
	if len(compacto) != 32 {
		return nil, false
	}
	decodificado := make([]byte, 16)
	_, err := hex.Decode(decodificado, compacto)
	if err != nil || decodificado[6]>>4 != 4 || decodificado[8]>>6 != 2 {
		borrarBytesBaremacion(decodificado)
		return nil, false
	}
	return decodificado, true
}

func materialBase64URLClienteBaremacionValido(material []byte) bool {
	if len(material) < 32 || len(material) > 64 || utf8.Valid(material) {
		return false
	}
	distintos := make(map[byte]struct{}, len(material))
	for _, valor := range material {
		distintos[valor] = struct{}{}
	}
	return len(distintos) >= 16
}

func vinculoSolicitudTestimonioIdempotenciaBaremacion(
	s SolicitudTestimonioAtomicoIdempotenciaBaremacion,
) [32]byte {
	formatoClave, materialClave := s.claveCliente.formatoYMaterialParaLote(s.reclamacion)
	defer borrarBytesBaremacion(materialClave)
	if formatoClave == 0 || len(materialClave) == 0 {
		return [32]byte{}
	}
	material := make([]byte, 0, 160)
	defer func() { borrarBytesBaremacion(material) }()
	material = anexarCampoCanonicoTextoIntencion(material, 1, esquemaTestimonioAtomicoIdempotenciaBaremacionV1)
	material = anexarCampoCanonicoTextoIntencion(material, 2, string(s.despliegueRef))
	material = anexarCampoCanonicoTextoIntencion(material, 3, string(s.modulo))
	material = anexarCampoCanonicoTextoIntencion(material, 4, string(s.clase))
	material = anexarCampoCanonicoTextoIntencion(material, 5, s.ambitoSujeto.procesoRef)
	material = anexarCampoCanonicoTextoIntencion(material, 6, s.ambitoSujeto.solicitudRef)
	material = anexarCampoCanonicoTextoIntencion(material, 7, s.ambitoSujeto.baremacionMeritoRef)
	material = anexarCampoCanonicoIntencion(material, 8, canonicoUint16Intencion(s.seudonimo.Version))
	material = anexarCampoCanonicoTextoIntencion(material, 9, s.seudonimo.ClaveHMACRef)
	valorSeudonimo := decodificarHexCanonicoIntencion(s.seudonimo.ValorHMAC)
	defer borrarBytesBaremacion(valorSeudonimo)
	material = anexarCampoCanonicoIntencion(
		material, 10, valorSeudonimo,
	)
	material = anexarCampoCanonicoIntencion(material, 11, []byte{byte(formatoClave)})
	material = anexarCampoCanonicoIntencion(material, 12, materialClave)
	return sha256.Sum256(material)
}

func mismaDependenciaConcretaBaremacion(izquierda, derecha any) bool {
	if dependenciaNulaBaremacion(izquierda) || dependenciaNulaBaremacion(derecha) {
		return true
	}
	// Compartir tipo concreto permite que una misma implementacion represente
	// simultaneamente al productor y a la supuesta raiz independiente, incluso
	// usando dos punteros. Se exige tambien diversidad de implementacion.
	return reflect.TypeOf(izquierda) == reflect.TypeOf(derecha)
}

// MaterialCanonicoEfimeroBaremacion es una vista revocable de una preimagen
// sensible. Todas sus copias comparten propietario: al terminar el callback
// exterior quedan invalidas y VisitarBytes falla. VisitarBytes entrega una
// unica copia sincrona y la borra al volver, incluso ante error o panico.
//
// Esto elimina las copias propias controlables, pero Go no permite garantizar
// zeroization absoluta frente a copias deliberadas del adaptador o del runtime.
// Si ese actor esta en el modelo de amenaza, DEC-047 exige otro proceso.
type MaterialCanonicoEfimeroBaremacion struct {
	propietario *propietarioMaterialCanonicoEfimeroBaremacion
}

type propietarioMaterialCanonicoEfimeroBaremacion struct {
	mu               sync.Mutex
	carga            CargaProtegida
	revocado         bool
	visitaIniciada   bool
	visitasEnVuelo   int
	visitaCompletada bool
}

func nuevoMaterialCanonicoEfimeroBaremacion(
	carga CargaProtegida,
) MaterialCanonicoEfimeroBaremacion {
	return MaterialCanonicoEfimeroBaremacion{
		propietario: &propietarioMaterialCanonicoEfimeroBaremacion{carga: carga},
	}
}

func (m MaterialCanonicoEfimeroBaremacion) Validar() error {
	if m.propietario == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	m.propietario.mu.Lock()
	defer m.propietario.mu.Unlock()
	if m.propietario.revocado || m.propietario.visitaIniciada ||
		m.propietario.visitaCompletada || m.propietario.carga.Validar() != nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return nil
}

func (m MaterialCanonicoEfimeroBaremacion) VisitarBytes(
	visita func([]byte) error,
) (errRetorno error) {
	if visita == nil || m.propietario == nil {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	m.propietario.mu.Lock()
	if m.propietario.revocado || m.propietario.visitaIniciada ||
		m.propietario.carga.Validar() != nil {
		m.propietario.mu.Unlock()
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	m.propietario.visitaIniciada = true
	m.propietario.visitasEnVuelo++
	copia := m.propietario.carga.Revelar()
	m.propietario.mu.Unlock()
	termino := false
	defer func() {
		borrarBytesBaremacion(copia)
		m.propietario.mu.Lock()
		m.propietario.visitasEnVuelo--
		if termino && errRetorno == nil && !m.propietario.revocado {
			m.propietario.visitaCompletada = true
		}
		m.propietario.mu.Unlock()
	}()
	errRetorno = visita(copia)
	termino = true
	return errRetorno
}

func (m MaterialCanonicoEfimeroBaremacion) revocarYComprobarConsumido() bool {
	if m.propietario == nil {
		return false
	}
	m.propietario.mu.Lock()
	defer m.propietario.mu.Unlock()
	if m.propietario.revocado {
		return false
	}
	m.propietario.revocado = true
	destruirCargaProtegidaBaremacion(&m.propietario.carga)
	return m.propietario.visitaIniciada && m.propietario.visitasEnVuelo == 0 &&
		m.propietario.visitaCompletada
}

func (MaterialCanonicoEfimeroBaremacion) String() string {
	return "[MATERIAL-CANONICO-EFIMERO-BAREMACION-PROTEGIDO]"
}
func (MaterialCanonicoEfimeroBaremacion) GoString() string {
	return "ports.MaterialCanonicoEfimeroBaremacion{[PROTEGIDO]}"
}
func (m MaterialCanonicoEfimeroBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, m.String())
}
func (MaterialCanonicoEfimeroBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (MaterialCanonicoEfimeroBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (MaterialCanonicoEfimeroBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionIdempotenciaBaremacion
}
func (m MaterialCanonicoEfimeroBaremacion) LogValue() slog.Value {
	return slog.StringValue(m.String())
}

// destruirCargaProtegidaBaremacion elimina las copias propietarias que este
// paquete si controla. Go no permite prometer zeroization absoluta frente a
// copias realizadas por el runtime o por un callback malicioso; ese limite se
// cubre con aislamiento de proceso en DEC-047, no con afirmaciones falsas.
func destruirCargaProtegidaBaremacion(carga *CargaProtegida) {
	if carga == nil {
		return
	}
	borrarBytesBaremacion(carga.valor)
	*carga = CargaProtegida{}
}

func visitarCargaProtegidaEfimeraBaremacion(
	carga CargaProtegida,
	visita func(MaterialCanonicoEfimeroBaremacion) error,
) error {
	if visita == nil || carga.Validar() != nil {
		destruirCargaProtegidaBaremacion(&carga)
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	material := nuevoMaterialCanonicoEfimeroBaremacion(carga)
	consumido := false
	defer func() {
		if !consumido {
			material.revocarYComprobarConsumido()
		}
	}()
	err := visita(material)
	consumido = material.revocarYComprobarConsumido()
	if err != nil {
		return err
	}
	if !consumido {
		return ErrClaveIdempotenciaBaremacionInvalida
	}
	return nil
}

func borrarBytesBaremacion(valor []byte) {
	for posicion := range valor {
		valor[posicion] = 0
	}
}

func sumaSHA256CargaProtegidaBaremacion(carga CargaProtegida) [32]byte {
	material := carga.Revelar()
	defer func() { borrarBytesBaremacion(material) }()
	return sha256.Sum256(material)
}

func dependenciaNulaBaremacion(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}

func textoIgualConstanteBaremacion(izquierda, derecha string) bool {
	if len(izquierda) != len(derecha) {
		return false
	}
	coincide := 1
	for posicion := 0; posicion < len(izquierda); posicion++ {
		coincide &= subtle.ConstantTimeByteEq(izquierda[posicion], derecha[posicion])
	}
	return coincide == 1
}

func hmacVersionadoIgualConstante(
	versionIzquierda uint16,
	claveIzquierdaRef, valorIzquierda string,
	versionDerecha uint16,
	claveDerechaRef, valorDerecha string,
) bool {
	var claveIzquierda, claveDerecha [128]byte
	copy(claveIzquierda[:], claveIzquierdaRef)
	copy(claveDerecha[:], claveDerechaRef)
	macIzquierda, _ := hex.DecodeString(valorIzquierda)
	macDerecha, _ := hex.DecodeString(valorDerecha)
	defer borrarBytesBaremacion(macIzquierda)
	defer borrarBytesBaremacion(macDerecha)

	coincide := subtle.ConstantTimeEq(int32(versionIzquierda), int32(versionDerecha))
	coincide &= subtle.ConstantTimeEq(int32(len(claveIzquierdaRef)), int32(len(claveDerechaRef)))
	coincide &= subtle.ConstantTimeCompare(claveIzquierda[:], claveDerecha[:])
	coincide &= subtle.ConstantTimeCompare(macIzquierda, macDerecha)
	return coincide == 1
}
