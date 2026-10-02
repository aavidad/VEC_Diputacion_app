package ports

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
)

var (
	ErrOriginalFirmableCTInvalido     = errors.New("vec: original firmable CT invalido")
	ErrOriginalFirmableCTConflicto    = errors.New("vec: original firmable CT en conflicto")
	ErrOriginalFirmableCTNoDisponible = errors.New("vec: original firmable CT no disponible")
)

const LimiteOriginalFirmableCT = 1 << 20

var (
	referenciaOriginalCTHash = regexp.MustCompile(`^ref:[0-9a-f]{64}$`)
	referenciaOriginalCTUUID = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	claveDocumentoOriginalCT = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	huellaOriginalCT         = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func ReferenciaOriginalCTValida(s string) bool {
	return (referenciaOriginalCTHash.MatchString(s) && s != "ref:"+strings.Repeat("0", 64)) ||
		referenciaOriginalCTUUID.MatchString(s)
}

func HuellaOriginalCTValida(s string) bool { return huellaOriginalCT.MatchString(s) }

// SolicitudOriginalFirmableCT contiene solo identificadores de la version CT.
// OriginalRef es una expectativa que se coteja, nunca una ruta ni una fuente
// de autoridad. Los bytes y la huella no proceden del solicitante.
type SolicitudOriginalFirmableCT struct {
	OrganizacionRef, ExpedienteRef, Documento, OriginalRef string
	OriginalVersion                                        uint64
}

func (s SolicitudOriginalFirmableCT) Validar() error {
	if !ReferenciaOriginalCTValida(s.OrganizacionRef) ||
		!ReferenciaOriginalCTValida(s.ExpedienteRef) ||
		!claveDocumentoOriginalCT.MatchString(s.Documento) ||
		s.OriginalVersion == 0 || s.OriginalVersion > 9007199254740991 {
		return ErrOriginalFirmableCTInvalido
	}
	ref := ReferenciaOriginalFirmableCT(s)
	if s.OriginalRef != "" && s.OriginalRef != ref {
		return ErrOriginalFirmableCTInvalido
	}
	return nil
}

// ReferenciaOriginalFirmableCT separa dominios y codifica longitudes para que
// ninguna combinacion de campos comparta preimagen. La referencia no contiene
// informacion legible del expediente y es estable entre procesos.
func ReferenciaOriginalFirmableCT(s SolicitudOriginalFirmableCT) string {
	h := sha256.New()
	_, _ = h.Write([]byte("vec:documentos:original-firmable-ct:v1"))
	for _, campo := range []string{s.OrganizacionRef, s.ExpedienteRef, s.Documento} {
		var longitud [8]byte
		binary.BigEndian.PutUint64(longitud[:], uint64(len(campo)))
		_, _ = h.Write(longitud[:])
		_, _ = h.Write([]byte(campo))
	}
	var version [8]byte
	binary.BigEndian.PutUint64(version[:], s.OriginalVersion)
	_, _ = h.Write(version[:])
	return "ref:" + hex.EncodeToString(h.Sum(nil))
}

// ClaveAltaOriginalFirmableCT permite al almacén aplicar idempotencia a un
// único original sin compartir la clave con otras operaciones documentales.
func ClaveAltaOriginalFirmableCT(s SolicitudOriginalFirmableCT) string {
	suma := sha256.Sum256([]byte("vec:documentos:clave-original-firmable-ct:v1:" + ReferenciaOriginalFirmableCT(s)))
	return "ref:" + hex.EncodeToString(suma[:])
}

// PDFOriginalCT procede de una fuente CT autorizada para esta version exacta;
// TipoRef es la referencia del tipo documental gobernado por Documentos.
type PDFOriginalCT struct {
	TipoRef   string
	Contenido []byte
}

type FuentePDFOriginalCT interface {
	ObtenerPDFOriginalCT(context.Context, SolicitudOriginalFirmableCT) (PDFOriginalCT, error)
}

// OriginalFirmableCT es el original custodiado. Cada lectura requiere una
// concesion PDP nueva dentro de CustodiaOriginalFirmableCT.
type OriginalFirmableCT struct {
	Referencia, TipoRef, HuellaSHA256 string
	Version                           uint64
	Contenido                         []byte
}

// CustodiaOriginalFirmableCT la implementa un puente al servicio Documentos,
// con su registro SQL, almacén y autorizaciones V3. GuardarUnaVez debe cotejar
// bytes, huella y metadatos de un original existente y rechazar diferencias;
// LeerOriginal debe consumir autorización de lectura vigente para cada llamada.
type CustodiaOriginalFirmableCT interface {
	GuardarUnaVez(context.Context, SolicitudOriginalFirmableCT, PDFOriginalCT, string, string) (OriginalFirmableCT, error)
	LeerOriginal(context.Context, SolicitudOriginalFirmableCT, string) (OriginalFirmableCT, error)
}
