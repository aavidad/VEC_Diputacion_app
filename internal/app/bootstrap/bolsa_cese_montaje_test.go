package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
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

func TestRelevoCeseNoExponeDSNEnRegistroNiError(t *testing.T) {
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })
	err := falloRelevoCeseBolsaDesarrollo(errors.New("postgres://usuario:secreto-marcador@host-privado/vec"))
	if !errors.Is(err, puertosbolsa.ErrContratosParticipacionNoDisponible) {
		t.Fatalf("error público del relevo: %v", err)
	}
	for _, secreto := range []string{"secreto-marcador", "host-privado", "postgres://"} {
		if strings.Contains(registro.String(), secreto) || strings.Contains(err.Error(), secreto) {
			t.Fatalf("el relevo filtró material de conexión: %s", secreto)
		}
	}
}
