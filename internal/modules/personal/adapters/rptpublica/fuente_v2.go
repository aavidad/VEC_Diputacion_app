package rptpublica

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"strings"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

const maximoBytesRPTPublicaV2 = 8 << 20

// FuenteV2 lee una instantánea completa en cada consulta. El hash fijado en la
// composición identifica los bytes admitidos; el material V3 y el recibo se
// ligan a esa misma instantánea, aunque el fichero cambie después de leerlo.
type FuenteV2 struct {
	ruta           string
	publicacionRef string
	corte          string
	huellaSHA256   string
}

func NuevaFuenteV2(ruta, publicacionRef, corte, huellaSHA256 string) (*FuenteV2, error) {
	if ruta == "" || ruta != strings.TrimSpace(ruta) || publicacionRef == "" || corte == "" || len(huellaSHA256) != 64 {
		return nil, domain.ErrRPTPublicaV2NoDisponible
	}
	for _, c := range huellaSHA256 {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return nil, domain.ErrRPTPublicaV2NoDisponible
		}
	}
	return &FuenteV2{ruta: ruta, publicacionRef: publicacionRef, corte: corte, huellaSHA256: huellaSHA256}, nil
}

func (f *FuenteV2) ObtenerRPTPublicaV2(ctx context.Context) (domain.SnapshotRPTPublicaV2, error) {
	var vacia domain.SnapshotRPTPublicaV2
	if f == nil || ctx == nil || ctx.Err() != nil {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	archivo, err := os.Open(f.ruta)
	if err != nil {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	defer archivo.Close()
	info, err := archivo.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maximoBytesRPTPublicaV2 {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	bruto, err := io.ReadAll(io.LimitReader(archivo, maximoBytesRPTPublicaV2+1))
	if err != nil || len(bruto) < 1 || len(bruto) > maximoBytesRPTPublicaV2 || int64(len(bruto)) != info.Size() || ctx.Err() != nil {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	suma := sha256.Sum256(bruto)
	actual := hex.EncodeToString(suma[:])
	if subtle.ConstantTimeCompare([]byte(actual), []byte(f.huellaSHA256)) != 1 {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	var catalogo domain.CatalogoRPTPublicaV2
	lector := json.NewDecoder(bytes.NewReader(bruto))
	lector.DisallowUnknownFields()
	if lector.Decode(&catalogo) != nil || lector.Decode(new(any)) != io.EOF {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	snapshot := domain.SnapshotRPTPublicaV2{PublicacionRef: f.publicacionRef, Corte: f.corte, HuellaSHA256: actual, Catalogo: catalogo}
	if snapshot.Validar() != nil || ctx.Err() != nil {
		return vacia, domain.ErrRPTPublicaV2NoDisponible
	}
	return snapshot, nil
}

var _ ports.FuenteRPTPublicaV2 = (*FuenteV2)(nil)
