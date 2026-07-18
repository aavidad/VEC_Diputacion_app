package memory

import (
	"context"
	"errors"
	"sync"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

var ErrLoteConvocaInvalido = errors.New("bolsa memoria: lote Convoca invalido")

// RepositorioImportacionesConvoca es el adaptador transaccional de memoria
// usado por el corte vertical y sus pruebas. GuardarSiAusente representa el
// mismo CAS por SHA-256 que debera cumplir cualquier persistencia durable.
type RepositorioImportacionesConvoca struct {
	mu    sync.RWMutex
	lotes map[string]dominio.LoteValidado
}

func NuevoRepositorioImportacionesConvoca() *RepositorioImportacionesConvoca {
	return &RepositorioImportacionesConvoca{lotes: make(map[string]dominio.LoteValidado)}
}

func (r *RepositorioImportacionesConvoca) GuardarSiAusente(
	ctx context.Context,
	lote dominio.LoteValidado,
) (dominio.LoteValidado, bool, error) {
	if err := ctx.Err(); err != nil {
		return dominio.LoteValidado{}, false, err
	}
	if lote.Validar() != nil {
		return dominio.LoteValidado{}, false, ErrLoteConvocaInvalido
	}
	huella := lote.Acta.HuellaFicheroSHA256
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return dominio.LoteValidado{}, false, err
	}
	if existente, ok := r.lotes[huella]; ok {
		return dominio.ClonarLote(existente), true, nil
	}
	guardado := dominio.ClonarLote(lote)
	r.lotes[huella] = guardado
	return dominio.ClonarLote(guardado), false, nil
}

func (r *RepositorioImportacionesConvoca) ObtenerPorHuella(
	ctx context.Context,
	huella string,
) (dominio.LoteValidado, bool, error) {
	if err := ctx.Err(); err != nil {
		return dominio.LoteValidado{}, false, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	lote, ok := r.lotes[huella]
	if !ok {
		return dominio.LoteValidado{}, false, nil
	}
	return dominio.ClonarLote(lote), true, nil
}

func (r *RepositorioImportacionesConvoca) NumeroLotes() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.lotes)
}
