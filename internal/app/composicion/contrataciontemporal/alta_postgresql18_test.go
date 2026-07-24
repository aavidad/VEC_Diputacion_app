//go:build o207postgresql

package contrataciontemporal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestComposicionAltaPostgreSQL18Real(t *testing.T) {
	dsnRuntime := os.Getenv("VEC_O207_DSN_RUNTIME")
	dsnAdmin := os.Getenv("VEC_O207_DSN_ADMIN")
	if dsnRuntime == "" || dsnAdmin == "" {
		t.Skip("runner PostgreSQL O2-07 no activado")
	}
	ctx := context.Background()
	runtime, err := pgxpool.New(ctx, dsnRuntime)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	doble := &dependenciasInertes{}
	dependencias := DependenciasAlta{
		AutoridadCanal: doble,
		Contextos:      doble,
		Flujos:         doble,
		Huellas:        doble,
		Ambitos:        doble,
		Motivos:        doble,
		Correlaciones:  doble,
		Referencias:    doble,
		Autorizador:    doble,
		Material:       doble,
		Reloj:          doble,
		PoolAltas:      runtime,
	}
	api, err := NuevaAPIAlta(ctx, dependencias)
	if err != nil || api == nil {
		t.Fatalf("runtime nominal rechazado = (%T, %v)", api, err)
	}
	peticion := httptest.NewRequest(
		http.MethodGet,
		"/api/interno/v1/contratacion-temporal/solicitudes",
		nil,
	)
	respuesta := httptest.NewRecorder()
	api.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusMethodNotAllowed {
		t.Fatalf("ruta compuesta = %d", respuesta.Code)
	}

	admin, err := pgxpool.New(ctx, dsnAdmin)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	dependencias.PoolAltas = admin
	api, err = NuevaAPIAlta(ctx, dependencias)
	if api != nil || !errors.Is(err, ErrComposicionAltaNoDisponible) {
		t.Fatalf("pool administrador aceptado = (%T, %v)", api, err)
	}
}
