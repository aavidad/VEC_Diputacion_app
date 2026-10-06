package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type contactosQueFallanPrueba struct{ err error }

func (c contactosQueFallanPrueba) ListarContactosBolsa(context.Context, string, string, int) (puertosbolsa.PaginaContactosParticipacion, error) {
	return puertosbolsa.PaginaContactosParticipacion{}, c.err
}

// La lista de candidatos ya no responde un 503 mudo: deja en el registro la
// etapa y la causa, y responde 403 si la autorización se denegó y 504 si se
// agotó el plazo. Del error de PostgreSQL solo se registra el código.
func TestBolsasRRHHCandidatosRegistraCausaYEstadoDelFallo(t *testing.T) {
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })
	ruta := rutaBolsasRRHHDesarrollo + "/" + datosBolsasRRHHPrueba().Bolsas[0].Referencia + "/candidatos?limite=50"
	for _, caso := range []struct {
		nombre, causa string
		err           error
		estado        int
	}{
		{"denegada", "autorizacion denegada", fmt.Errorf("envoltura: %w", dominiovec.ErrAutorizacionDenegada), http.StatusForbidden},
		{"plazo", "tiempo_agotado", fmt.Errorf("lectura: %w", context.DeadlineExceeded), http.StatusGatewayTimeout},
		{"centinela", puertosbolsa.ErrContactoParticipacionNoDisponible.Error(), puertosbolsa.ErrContactoParticipacionNoDisponible, http.StatusServiceUnavailable},
		{"postgresql", "sqlstate_55P03", &pgconn.PgError{Code: "55P03", Message: "secreto-marcador"}, http.StatusServiceUnavailable},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			registro.Reset()
			manejador := manejadorBolsasRRHHPrueba()
			manejador.mutar = http.NotFoundHandler()
			manejador.contactos = contactosQueFallanPrueba{err: caso.err}
			rec := httptest.NewRecorder()
			manejador.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ruta, nil))
			linea := registro.String()
			if rec.Code != caso.estado || !strings.Contains(linea, "etapa=contactos") || !strings.Contains(linea, caso.causa) {
				t.Fatalf("estado=%d registro=%q", rec.Code, linea)
			}
			if strings.Contains(linea, "secreto-marcador") || strings.Contains(rec.Body.String(), "secreto-marcador") || strings.Contains(rec.Body.String(), "etapa") {
				t.Fatalf("detalle interno filtrado: registro=%q cuerpo=%q", linea, rec.Body.String())
			}
		})
	}
	// Carga de la bolsa fallida: misma regla, etapa propia.
	registro.Reset()
	manejador := manejadorBolsasRRHHPrueba()
	manejador.cargarBolsa = func(context.Context, string) (datasetBolsasRRHHDesarrollo, error) {
		return datasetBolsasRRHHDesarrollo{}, errors.Join(ErrComposicionDesarrolloIncompleta)
	}
	rec := httptest.NewRecorder()
	manejador.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ruta, nil))
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(registro.String(), "etapa=bolsa") || !strings.Contains(registro.String(), "composicion") {
		t.Fatalf("estado=%d registro=%q", rec.Code, registro.String())
	}
}

// Un error desconocido con valores no llega al registro: solo su tipo.
func TestCausaFalloBolsaRRHHNoRegistraTextoDeErroresDesconocidos(t *testing.T) {
	if causa := causaFalloBolsaRRHHDesarrollo(fmt.Errorf("strconv: parsing %q", "Antonio Reyes Álvarez")); strings.Contains(causa, "Antonio") {
		t.Fatalf("texto de error desconocido en el registro: %q", causa)
	}
	if causa := causaFalloBolsaRRHHDesarrollo(fmt.Errorf("capa: %w", puertosbolsa.ErrSituacionParticipacionNoEncontrada)); causa != puertosbolsa.ErrSituacionParticipacionNoEncontrada.Error() {
		t.Fatalf("centinela: %q", causa)
	}
	if causa := causaFalloBolsaRRHHDesarrollo(errBorradorNoDisponibleEn()); !strings.Contains(causa, "bolsa_rrhh_fallos_test.go:") {
		t.Fatalf("rechazo B-BACK sin fichero y línea: %q", causa)
	}
}
