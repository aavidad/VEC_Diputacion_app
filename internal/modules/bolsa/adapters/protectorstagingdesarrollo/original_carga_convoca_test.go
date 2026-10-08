package protectorstagingdesarrollo

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOriginalCargaConvocaCifradoLigadoAlActa(t *testing.T) {
	contenido, err := os.ReadFile(filepath.Join("..", "..", "application", "testdata", "carga_convoca", "carga_convoca_ejemplo.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
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
	segundo, err := p.Preparar(context.Background(), acta, huella, "xlsx", contenido)
	if err != nil || bytes.Equal(sobre.Nonce, segundo.Nonce) || bytes.Equal(sobre.ContenidoCifrado, segundo.ContenidoCifrado) {
		t.Fatalf("el cifrado repitió nonce o texto cifrado: %v", err)
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
	if _, err := p.Preparar(context.Background(), acta, huella, "xls", contenido); !errors.Is(err, ErrOriginalNoConfiable) {
		t.Fatalf("XLSX renombrado .xls admitido: %v", err)
	}
	xls, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "data", "demo", "bolsa", "convoca", "psicologo", "resumen.xls"))
	if err != nil {
		t.Fatal(err)
	}
	hXLS := sha256.Sum256(xls)
	huellaXLS := hex.EncodeToString(hXLS[:])
	if _, err := p.Preparar(context.Background(), acta, huellaXLS, "xls", xls); err != nil {
		t.Fatalf("XLS válido rechazado: %v", err)
	}
	if _, err := p.Preparar(context.Background(), acta, huellaXLS, "xlsx", xls); !errors.Is(err, ErrOriginalNoConfiable) {
		t.Fatalf("XLS renombrado .xlsx admitido: %v", err)
	}
	tamper := append([]byte(nil), contenido...)
	tamper[len(tamper)-1] ^= 1
	if _, err := p.Preparar(context.Background(), acta, huella, "xlsx", tamper); !errors.Is(err, ErrOriginalNoConfiable) {
		t.Fatalf("original alterado admitido: %v", err)
	}
}
