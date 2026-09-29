package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// Custodia del PDF firmado (5.06). Tras verificar una firma, CT entrega el PDF
// firmado a Documentos, que lo guarda con custodia VEC, y enlaza la firma con
// ese documento en CT145. CT no lee tablas de Documentos: habla con él por
// este puerto y guarda solo la referencia, la versión y la huella que recibe.

var (
	// ErrCustodiaFirmadoNoDisponible: Documentos no está compuesto o no
	// responde; la firma no se registra.
	ErrCustodiaFirmadoNoDisponible = errors.New("contratacion temporal: custodia del documento firmado no disponible")
	// ErrCustodiaFirmadoDenegada: Documentos o su autorización la rechazan.
	ErrCustodiaFirmadoDenegada = errors.New("contratacion temporal: custodia del documento firmado denegada")
	// ErrCustodiaFirmadoInvalida: Documentos no admite el contenido o el tipo
	// (por ejemplo, no es un PDF o el tipo no está reservado).
	ErrCustodiaFirmadoInvalida = errors.New("contratacion temporal: documento firmado no admitido para custodia")
	// ErrCustodiaFirmadoEnConflicto: la misma operación ya custodió otro PDF.
	ErrCustodiaFirmadoEnConflicto = errors.New("contratacion temporal: custodia del documento firmado en conflicto")
)

// VersionDocumentoCustodiado es fija: cada operación de firma custodia un
// documento propio (su referencia sale de la clave de la operación), así que
// no hay versiones sucesivas de un mismo documento.
const VersionDocumentoCustodiado uint64 = 1

// OrdenCustodiaFirmado es lo que CT pide custodiar. TipoDocumental es la
// clave del catálogo de conservación (por ejemplo, la de la resolución
// firmada); FirmaOperacionRef, la operación de firma de CT.
type OrdenCustodiaFirmado struct {
	DocumentoRef         string
	ClaveIdempotencia    string
	ExpedienteRef        string
	TipoDocumental       string
	Version              uint64
	Contenido            []byte
	HuellaOriginalSHA256 string
	FirmaOperacionRef    string
}

// DocumentoCustodiado es lo que Documentos confirma: su referencia opaca, la
// versión y la huella SHA-256 de los bytes custodiados.
type DocumentoCustodiado struct {
	Ref          string
	Version      uint64
	HuellaSHA256 string
}

// CustodioDocumentoFirmado guarda el PDF firmado en Documentos. Un error
// envuelto en ErrCustodiaFirmadoNoDisponible es dependencia caída; en
// ErrCustodiaFirmadoInvalida, contenido no admitido; en
// ErrCustodiaFirmadoEnConflicto, la misma operación con otro PDF; cualquier
// otro, denegación. Repetir la misma orden tras perder la respuesta devuelve
// el documento ya custodiado.
type CustodioDocumentoFirmado interface {
	CustodiarFirmado(context.Context, OrdenCustodiaFirmado) (DocumentoCustodiado, error)
}

// referenciaCustodia deriva una referencia opaca de Documentos («ref:» y 64
// hexadecimales) de la organización, el expediente y la clave de la operación
// de firma. No es secreta: la unicidad la da CT118 (una clave por expediente)
// y el acceso lo decide la autorización.
func referenciaCustodia(uso, organizacionRef, expedienteRef, clave string) string {
	s := sha256.Sum256([]byte("vec.contratacion_temporal.custodia_firmado.v1\x00" + uso + "\x00" +
		organizacionRef + "\x00" + expedienteRef + "\x00" + clave))
	return "ref:" + hex.EncodeToString(s[:])
}

// ExpedienteDocumentalRef es la referencia con la que Documentos agrupa los
// documentos de un expediente de CT: estable y opaca, sin exponer la de CT.
func ExpedienteDocumentalRef(organizacionRef, expedienteRef string) string {
	return referenciaCustodia("expediente", organizacionRef, expedienteRef, "")
}

// DocumentoCustodiaRef y ClaveCustodiaRef son el identificador y la clave de
// idempotencia en Documentos del PDF firmado de una operación de firma.
func DocumentoCustodiaRef(organizacionRef, expedienteRef, claveFirma string) string {
	return referenciaCustodia("documento", organizacionRef, expedienteRef, claveFirma)
}

func ClaveCustodiaRef(organizacionRef, expedienteRef, claveFirma string) string {
	return referenciaCustodia("clave", organizacionRef, expedienteRef, claveFirma)
}

// OperacionFirmaRef es la referencia opaca de la operación de firma que
// Documentos guarda junto al documento.
func OperacionFirmaRef(organizacionRef, expedienteRef, claveFirma string) string {
	return referenciaCustodia("operacion", organizacionRef, expedienteRef, claveFirma)
}
