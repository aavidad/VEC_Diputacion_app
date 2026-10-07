package almacen

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"regexp"
	"strings"
)

var (
	referenciaFuenteCT = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$`)
	claveDocumentoCT   = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	referenciaHash     = regexp.MustCompile(`^ref:[0-9a-f]{64}$`)
	referenciaUUID     = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	huellaSHA256       = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// IdentidadOriginalCT admite la gramática del expediente CT. Sus valores solo
// se usan para derivar referencias opacas; nunca se convierten en rutas.
type IdentidadOriginalCT struct {
	OrganizacionRef, ExpedienteRef, Documento string
	Version                                   uint64
}

func (i IdentidadOriginalCT) Valida() bool {
	return referenciaFuenteCT.MatchString(i.OrganizacionRef) &&
		referenciaFuenteCT.MatchString(i.ExpedienteRef) &&
		claveDocumentoCT.MatchString(i.Documento) && i.Version > 0 && i.Version <= 9007199254740991
}

func (i IdentidadOriginalCT) Referencia() string {
	if !i.Valida() {
		return ""
	}
	h := sha256.New()
	_, _ = h.Write([]byte("vec:documentos:original-firmable-ct:v1"))
	for _, campo := range []string{i.OrganizacionRef, i.ExpedienteRef, i.Documento} {
		var longitud [8]byte
		binary.BigEndian.PutUint64(longitud[:], uint64(len(campo)))
		_, _ = h.Write(longitud[:])
		_, _ = h.Write([]byte(campo))
	}
	var version [8]byte
	binary.BigEndian.PutUint64(version[:], i.Version)
	_, _ = h.Write(version[:])
	return "ref:" + hex.EncodeToString(h.Sum(nil))
}

func (i IdentidadOriginalCT) ClaveLogica() string {
	if !i.Valida() {
		return ""
	}
	suma := sha256.Sum256([]byte("vec:documentos:clave-original-firmable-ct:v1:" + i.Referencia()))
	return "ref:" + hex.EncodeToString(suma[:])
}

func ReferenciaDocumentoValida(s string) bool {
	return (referenciaHash.MatchString(s) && s != "ref:"+strings.Repeat("0", 64)) || referenciaUUID.MatchString(s)
}

func HuellaSHA256Valida(s string) bool { return huellaSHA256.MatchString(s) }
