package protectorstagingdesarrollo

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	originalInfo    = "vec/bolsa/importacion-convoca/original/v1"
	esquemaOriginal = "vec.bolsa.importacion-convoca.proteccion-original.v1"
	maximoOriginal  = 1 * 1024 * 1024
)

var ErrOriginalNoConfiable = errors.New("bolsa: original CONVOCA no confiable")

// ProtectorOriginal cifra en memoria con una clave derivada distinta de las
// claves del staging. La persistencia ocurre exclusivamente en la TX B1.
type ProtectorOriginal struct {
	clave    [32]byte
	claveRef string
}

func NuevoProtectorOriginal(maestra [32]byte) (*ProtectorOriginal, error) {
	k, err := clave(maestra, originalInfo)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(maestra[:])
	return &ProtectorOriginal{clave: k, claveRef: "kms-desarrollo:" + originalInfo + ":" + hex.EncodeToString(h[:4])}, nil
}

func (p *ProtectorOriginal) Preparar(ctx context.Context, actaRef, huella, formato string, contenido []byte) (ports.OriginalProtegidoCargaConvoca, error) {
	if ctx == nil || p == nil {
		return ports.OriginalProtegidoCargaConvoca{}, ErrOriginalNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return ports.OriginalProtegidoCargaConvoca{}, err
	}
	if len(contenido) == 0 || len(contenido) > maximoOriginal || (formato != "xls" && formato != "xlsx") ||
		len(huella) != sha256.Size*2 || !strings.HasPrefix(actaRef, "acta:importacion-convoca:") ||
		len(actaRef) != len("acta:importacion-convoca:")+sha256.Size*2 {
		return ports.OriginalProtegidoCargaConvoca{}, ErrOriginalNoConfiable
	}
	suma := sha256.Sum256(contenido)
	if hex.EncodeToString(suma[:]) != huella {
		return ports.OriginalProtegidoCargaConvoca{}, ErrOriginalNoConfiable
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(actaRef, "acta:importacion-convoca:")); err != nil {
		return ports.OriginalProtegidoCargaConvoca{}, ErrOriginalNoConfiable
	}
	b, err := aes.NewCipher(p.clave[:])
	if err != nil {
		return ports.OriginalProtegidoCargaConvoca{}, ErrOriginalNoConfiable
	}
	a, err := cipher.NewGCM(b)
	if err != nil {
		return ports.OriginalProtegidoCargaConvoca{}, ErrOriginalNoConfiable
	}
	n := make([]byte, a.NonceSize())
	if _, err := io.ReadFull(rand.Reader, n); err != nil {
		return ports.OriginalProtegidoCargaConvoca{}, ErrOriginalNoConfiable
	}
	cifrado := a.Seal(nil, n, contenido, aadOriginal(actaRef, huella, formato, len(contenido)))
	huellaCifrado := sha256.Sum256(cifrado)
	return ports.OriginalProtegidoCargaConvoca{
		Referencia: "original:convoca:" + strings.TrimPrefix(actaRef, "acta:importacion-convoca:"),
		Formato:    formato, BytesOriginales: len(contenido), EsquemaProteccion: esquemaOriginal,
		ClaveRef: p.claveRef, ClaveVersion: 1, Nonce: n, ContenidoCifrado: cifrado,
		HuellaContenidoCifradoSHA256: hex.EncodeToString(huellaCifrado[:]),
	}, nil
}

func aadOriginal(actaRef, huella, formato string, tamano int) []byte {
	return []byte("bolsa.importacion.original.v1\x1f" + actaRef + "\x1f" + huella + "\x1f" + formato + "\x1f" + strconv.Itoa(tamano))
}
