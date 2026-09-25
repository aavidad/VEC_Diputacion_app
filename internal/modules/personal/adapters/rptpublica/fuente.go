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
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

const HuellaRPT2026 = "b0685beb5c02b8a30d5e0d6d3d9bceca11ddf76ad4987f4bcb1aa60ac7ebe9a8"
const maximoBytesRPTPublica = 8 << 20

// Fuente lee la RPT publicada, inmovilizada por su huella. El catálogo ya
// validado se conserva en memoria: mientras el fichero conserve tamaño y fecha
// de modificación no se vuelve a leer, resumir ni decodificar (8 MiB por
// consulta). Cualquier cambio en el fichero obliga a validarlo de nuevo, de
// modo que una fuente sustituida o retirada sigue fallando cerrada.
type Fuente struct {
	ruta string

	mu      sync.Mutex
	memoria *catalogoValidado
}

type catalogoValidado struct {
	tamano     int64
	modificado time.Time
	catalogo   domain.CatalogoRPTPublica
}

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
	if catalogo, ok := f.enMemoria(infoRuta); ok {
		return catalogo, nil
	}
	catalogo, info, err := f.leer()
	if err != nil {
		return domain.CatalogoRPTPublica{}, err
	}
	if err := ctx.Err(); err != nil {
		return domain.CatalogoRPTPublica{}, err
	}
	f.mu.Lock()
	f.memoria = &catalogoValidado{tamano: info.Size(), modificado: info.ModTime(), catalogo: catalogo.Clonar()}
	f.mu.Unlock()
	return catalogo, nil
}

// enMemoria devuelve una copia del catálogo validado si el fichero no ha
// cambiado desde que se validó.
func (f *Fuente) enMemoria(info os.FileInfo) (domain.CatalogoRPTPublica, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.memoria == nil || f.memoria.tamano != info.Size() || !f.memoria.modificado.Equal(info.ModTime()) {
		return domain.CatalogoRPTPublica{}, false
	}
	return f.memoria.catalogo.Clonar(), true
}

// leer lee, comprueba la huella, decodifica y valida el fichero.
func (f *Fuente) leer() (domain.CatalogoRPTPublica, os.FileInfo, error) {
	archivo, err := os.Open(f.ruta)
	if err != nil {
		return domain.CatalogoRPTPublica{}, nil, domain.ErrRPTPublicaNoDisponible
	}
	defer archivo.Close()
	info, err := archivo.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maximoBytesRPTPublica {
		return domain.CatalogoRPTPublica{}, nil, domain.ErrRPTPublicaNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(archivo, maximoBytesRPTPublica+1))
	if err != nil || len(contenido) < 1 || len(contenido) > maximoBytesRPTPublica || int64(len(contenido)) != info.Size() {
		return domain.CatalogoRPTPublica{}, nil, domain.ErrRPTPublicaNoDisponible
	}
	suma := sha256.Sum256(contenido)
	if !constanteIgual(hex.EncodeToString(suma[:]), HuellaRPT2026) {
		return domain.CatalogoRPTPublica{}, nil, domain.ErrRPTPublicaNoDisponible
	}
	var catalogo domain.CatalogoRPTPublica
	if json.Unmarshal(contenido, &catalogo) != nil {
		return domain.CatalogoRPTPublica{}, nil, domain.ErrRPTPublicaNoDisponible
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
		return domain.CatalogoRPTPublica{}, nil, domain.ErrRPTPublicaNoDisponible
	}
	return catalogo, info, nil
}
func constanteIgual(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

var _ ports.ConsultaRPTPublica = (*Fuente)(nil)
