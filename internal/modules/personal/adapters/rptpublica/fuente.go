// Package rptpublica adapta exclusivamente la proyeccion publica aprobada de RPT.
package rptpublica

import (
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

const HuellaRPT2026 = "b0685beb5c02b8a30d5e0d6d3d9bceca11ddf76ad4987f4bcb1aa60ac7ebe9a8"
const maximoBytesRPTPublica = 8 << 20

type Fuente struct{ ruta string }

func NuevaFuente(ruta string) (*Fuente, error) {
	if strings.TrimSpace(ruta) == "" {
		return nil, domain.ErrConsultaRPTPublicaInvalida
	}
	return &Fuente{ruta: ruta}, nil
}
func (f *Fuente) ObtenerRPTPublica(ctx context.Context) (domain.CatalogoRPTPublica, error) {
	if ctx == nil || f == nil || strings.TrimSpace(f.ruta) == "" {
		return domain.CatalogoRPTPublica{}, domain.ErrConsultaRPTPublicaInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.CatalogoRPTPublica{}, err
	}
	infoRuta, err := os.Stat(f.ruta)
	if err != nil || !infoRuta.Mode().IsRegular() || infoRuta.Size() < 1 || infoRuta.Size() > maximoBytesRPTPublica {
		return domain.CatalogoRPTPublica{}, domain.ErrRPTPublicaNoDisponible
	}
	archivo, err := os.Open(f.ruta)
	if err != nil {
		return domain.CatalogoRPTPublica{}, domain.ErrRPTPublicaNoDisponible
	}
	defer archivo.Close()
	info, err := archivo.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maximoBytesRPTPublica {
		return domain.CatalogoRPTPublica{}, domain.ErrRPTPublicaNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(archivo, maximoBytesRPTPublica+1))
	if err != nil || len(contenido) < 1 || len(contenido) > maximoBytesRPTPublica || int64(len(contenido)) != info.Size() {
		return domain.CatalogoRPTPublica{}, domain.ErrRPTPublicaNoDisponible
	}
	suma := sha256.Sum256(contenido)
	if !constanteIgual(hex.EncodeToString(suma[:]), HuellaRPT2026) {
		return domain.CatalogoRPTPublica{}, domain.ErrRPTPublicaNoDisponible
	}
	var catalogo domain.CatalogoRPTPublica
	if json.Unmarshal(contenido, &catalogo) != nil {
		return domain.CatalogoRPTPublica{}, domain.ErrRPTPublicaNoDisponible
	}
	catalogo.Fuente.HuellaSHA256 = HuellaRPT2026
	for i := range catalogo.Categorias {
		catalogo.Categorias[i] = catalogo.Categorias[i].Clonar()
	}
	for i := range catalogo.Puestos {
		catalogo.Puestos[i] = catalogo.Puestos[i].Clonar()
	}
	domain.OrdenarCategoriasRPTPublica(catalogo.Categorias)
	domain.OrdenarPuestosRPTPublica(catalogo.Puestos)
	if catalogo.Validar() != nil {
		return domain.CatalogoRPTPublica{}, domain.ErrRPTPublicaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return domain.CatalogoRPTPublica{}, err
	}
	return catalogo, nil
}
func constanteIgual(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

var _ ports.ConsultaRPTPublica = (*Fuente)(nil)
