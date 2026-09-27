package bootstrap

import (
	"context"
	"testing"
	"time"
)

func TestCeseB45NoSirveCacheRRHHNiB10TrasFallarFuente(t *testing.T) {
	ahora := time.Now().UTC()
	cache := datasetBolsasRRHHDesarrollo{GeneradoEn: ahora.Add(-time.Second).Format(time.RFC3339)}
	f := &fuenteConstituidaRRHHDesarrollo{
		ceseActivo: true, cacheada: true, cache: cache, hasta: ahora.Add(time.Hour), ahora: func() time.Time { return ahora },
	}
	if _, ok := f.constituidas(context.Background()); ok {
		t.Fatal("B45 activo devolvió datos previos cuando la lectura sensible falló")
	}
	f.ceseActivo = false
	if datos, ok := f.constituidas(context.Background()); !ok || datos.GeneradoEn != cache.GeneradoEn {
		t.Fatal("sin B45 se alteró el comportamiento histórico de cache")
	}
}
