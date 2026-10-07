package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

// El listado público solo cuenta personas: no descifra ninguna acta. La
// lista de una bolsa lee solo esa bolsa.
func TestBolsasPublicasServidorLeeLoJusto(t *testing.T) {
	fuente, c := fuenteVariasBolsasRRHHPrueba(t, 4, 30, true)
	publica := &fuenteBolsasPublicasDesarrollo{fuente: fuente}
	bolsas, _, err := publica.BolsasPublicas(context.Background())
	if err != nil || len(bolsas) != 4 {
		t.Fatalf("bolsas=%d err=%v", len(bolsas), err)
	}
	for _, b := range bolsas {
		if b.Total != 30 {
			t.Fatalf("total %s=%d", b.BolsaRef, b.Total)
		}
	}
	if c.recuperar.Load() != 0 {
		t.Fatalf("el listado público descifró %d actas", c.recuperar.Load())
	}
	ordenAntes := c.orden.Load()
	bolsa, posiciones, _, err := publica.ListaPublica(context.Background(), "bolsa:prueba:02")
	if err != nil || bolsa.BolsaRef != "bolsa:prueba:02" || len(posiciones) != 30 || posiciones[0].DocumentoEnmascarado == "" {
		t.Fatalf("lista: %+v %d %v", bolsa, len(posiciones), err)
	}
	if c.recuperar.Load() != 1 || c.orden.Load()-ordenAntes != 1 {
		t.Fatalf("la lista de una bolsa leyó de más: actas=%d órdenes=%d", c.recuperar.Load(), c.orden.Load()-ordenAntes)
	}
}

// Un plazo agotado se devuelve como tal (el manejador público responde 504)
// y deja su causa en el registro, en lugar de un 503 sin explicación.
func TestBolsasPublicasServidorConservaPlazoAgotadoYRegistraCausa(t *testing.T) {
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })
	fuente, _ := fuenteVariasBolsasRRHHPrueba(t, 1, 1, true)
	publica := &fuenteBolsasPublicasDesarrollo{fuente: fuente}
	_, _, err := publica.datos(context.Background(), "", func(context.Context) (datasetBolsasRRHHDesarrollo, error) {
		return datasetBolsasRRHHDesarrollo{}, fmt.Errorf("orden vigente: %w", context.DeadlineExceeded)
	})
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(registro.String(), "causa=tiempo_agotado") {
		t.Fatalf("err=%v registro=%q", err, registro.String())
	}
}
