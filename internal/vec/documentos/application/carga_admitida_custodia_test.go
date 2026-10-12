package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"testing"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestFuenteCargaAdmitidaCustodiaSinAutoridadesFallaCerrada(t *testing.T) {
	if _, err := NuevaFuenteCargaAdmitidaCustodia(nil, nil, nil, nil, nil, nil,
		vecdomain.ReferenciaEntradaCatalogo{}, nil); err == nil {
		t.Fatal("se construyó fuente sin configuración ni PDP")
	}
}

func TestLectorCustodiaExigeEOFYHuellaExacta(t *testing.T) {
	contenido := []byte("contenido-admitido")
	suma := sha256.Sum256(contenido)
	huella := hex.EncodeToString(suma[:])
	nuevo := func(datos []byte, esperado int64) *lectorCustodiaCompleta {
		return &lectorCustodiaCompleta{origen: io.NopCloser(bytes.NewReader(datos)),
			esperado: esperado, huella: huella, hash: sha256.New()}
	}
	exacto := nuevo(contenido, int64(len(contenido)))
	leidos, err := io.ReadAll(exacto)
	if err != nil || !bytes.Equal(leidos, contenido) || exacto.Close() != nil {
		t.Fatalf("lectura exacta: %v", err)
	}
	temprano := nuevo(contenido, int64(len(contenido)))
	if _, err := temprano.Read(make([]byte, 3)); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(temprano.Close(), vecports.ErrIntegridadObjetoAlmacen) {
		t.Fatal("cierre antes de EOF acreditó bytes no leídos")
	}
	truncado := nuevo(contenido[:len(contenido)-1], int64(len(contenido)))
	if _, err := io.ReadAll(truncado); !errors.Is(err, vecports.ErrIntegridadObjetoAlmacen) {
		t.Fatalf("truncado aceptado: %v", err)
	}
	extra := nuevo(append(append([]byte(nil), contenido...), 'x'), int64(len(contenido)))
	if _, err := io.ReadAll(extra); !errors.Is(err, vecports.ErrIntegridadObjetoAlmacen) {
		t.Fatalf("byte extra aceptado: %v", err)
	}
	alterado := append([]byte(nil), contenido...)
	alterado[0] ^= 1
	if _, err := io.ReadAll(nuevo(alterado, int64(len(contenido)))); !errors.Is(err, vecports.ErrIntegridadObjetoAlmacen) {
		t.Fatalf("mismo tamaño con SHA ajeno aceptado: %v", err)
	}
}
