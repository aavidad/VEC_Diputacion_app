package ejecucioncopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	aislado "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

// FicherosArchivados mide los bytes extraídos, además de la huella del tar
// autenticado. V1 aquí cubre componentes de release empaquetados como archivos;
// los almacenes externos requieren otro inspector de sus árboles completos.
type FicherosArchivados struct{ MaxComponenteBytes int64 }

func (f FicherosArchivados) MedirFicheros(ctx context.Context, e aislado.Entorno, m copias.Manifiesto) ([]copias.Artefacto, error) {
	if ctx == nil || ctx.Err() != nil || e.Archivado == nil || f.MaxComponenteBytes <= 0 || f.MaxComponenteBytes > 1<<30 || len(m.Inventario.PostgreSQL.Almacenes) > 0 {
		return nil, ErrEvidencia
	}
	esperados := append(append([]copias.Artefacto{}, m.Inventario.Release.Binarios...), m.Inventario.Release.Componentes...)
	if len(esperados) == 0 {
		return nil, ErrEvidencia
	}
	salida := make([]copias.Artefacto, 0, len(esperados))
	vistos := map[string]bool{}
	for _, a := range esperados {
		if vistos[a.ID] || a.TamanoBytes < 0 || a.TamanoBytes > f.MaxComponenteBytes {
			return nil, ErrEvidencia
		}
		vistos[a.ID] = true
		id := "fisica:" + a.ID
		encontrado := false
		for _, c := range e.Componentes {
			if c.ID == id && c.Tipo == a.Tipo && c.ContenidoSHA256 == a.SHA256 && c.ContenidoBytes == a.TamanoBytes {
				encontrado = true
				break
			}
		}
		if !encontrado {
			return nil, ErrEvidencia
		}
		b, err := e.Archivado.LeerArchivado(ctx, id, int(f.MaxComponenteBytes))
		if err != nil || int64(len(b)) != a.TamanoBytes {
			clear(b)
			return nil, ErrEvidencia
		}
		h := sha256.Sum256(b)
		clear(b)
		if hex.EncodeToString(h[:]) != a.SHA256 {
			return nil, ErrEvidencia
		}
		salida = append(salida, a)
	}
	return salida, nil
}
