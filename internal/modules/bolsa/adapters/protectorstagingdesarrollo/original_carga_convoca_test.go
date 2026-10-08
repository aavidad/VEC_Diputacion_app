package protectorstagingdesarrollo

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestOriginalCargaConvocaCifradoLigadoAlActa(t *testing.T) {
	contenido := []byte("xlsx sintetico para prueba")
	h := sha256.Sum256(contenido)
	huella := hex.EncodeToString(h[:])
	acta := "acta:importacion-convoca:" + strings.Repeat("a", 64)
	p, err := NuevoProtectorOriginal([32]byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	sobre, err := p.Preparar(context.Background(), acta, huella, "xlsx", contenido)
	if err != nil {
		t.Fatal(err)
	}
	if sobre.Referencia != "original:convoca:"+strings.Repeat("a", 64) || bytes.Contains(sobre.ContenidoCifrado, contenido) || sobre.ClaveVersion != 1 {
		t.Fatalf("sobre inseguro: ref=%q version=%d", sobre.Referencia, sobre.ClaveVersion)
	}
	b, err := aes.NewCipher(p.clave[:])
	if err != nil {
		t.Fatal(err)
	}
	a, err := cipher.NewGCM(b)
	if err != nil {
		t.Fatal(err)
	}
	claro, err := a.Open(nil, sobre.Nonce, sobre.ContenidoCifrado, aadOriginal(acta, huella, "xlsx", len(contenido)))
	if err != nil || !bytes.Equal(claro, contenido) {
		t.Fatalf("cifrado no recuperable: %v", err)
	}
	if _, err := a.Open(nil, sobre.Nonce, sobre.ContenidoCifrado, aadOriginal(acta, huella, "xls", len(contenido))); err == nil {
		t.Fatal("AAD distinto aceptado")
	}
	if _, err := p.Preparar(context.Background(), acta, strings.Repeat("0", 64), "xlsx", contenido); !errors.Is(err, ErrOriginalNoConfiable) {
		t.Fatalf("huella falsa admitida: %v", err)
	}
}
