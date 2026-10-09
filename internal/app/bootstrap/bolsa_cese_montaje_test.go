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
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Si Bolsa no se puede leer, la cobertura queda «no disponible», con o sin
// cese: no hay lectura anterior que servir.
func TestCeseB45CoberturaNoSirveDatosPreviosTrasFallarFuente(t *testing.T) {
	ahora := time.Now().UTC()
	for _, cese := range []bool{true, false} {
		f := &fuenteConstituidaRRHHDesarrollo{ceseActivo: cese, ahora: func() time.Time { return ahora }}
		_, err := situacionBolsaCoberturaDesarrollo{fuente: f}.SituacionBolsaCobertura(context.Background(), "categoria:rpt:prueba-01")
		if !errors.Is(err, ctports.ErrSituacionBolsaCoberturaNoDisponible) {
			t.Fatalf("cese=%v: una fuente que falla dio situación: %v", cese, err)
		}
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
