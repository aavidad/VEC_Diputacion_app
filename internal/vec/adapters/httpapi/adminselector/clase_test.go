package adminselector

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
)

// Cada 503 del selector deja una línea técnica con la clase cerrada del fallo
// (con el SQLSTATE cuando lo hay) y nunca la causa; los demás estados no.
func TestSelector503RegistraClaseSinCausa(t *testing.T) {
	sql := adminperfiles.ConClaseSelector("selector_sql_P0001",
		fmt.Errorf("envoltura: %w", &pgconn.PgError{Code: "P0001", Message: "DETALLE_PRIVADO"}))
	for _, caso := range []struct {
		nombre, clase string
		err           error
		estado        int
	}{
		{"sql", "selector_sql_P0001", sql, 503},
		{"validacion", "selector_revision_fuera_de_rango",
			adminperfiles.ConClaseSelector("selector_revision_fuera_de_rango", api.ErrConfiguracionIncompleta), 503},
		{"sin_clase", "sin_clase", errors.New("DETALLE_PRIVADO"), 503},
		{"conflicto", "", api.ErrConflictoEstado, 409},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			h, _, s, _ := escenario(t)
			var buf bytes.Buffer
			h.registro = slog.New(slog.NewTextHandler(&buf, nil))
			s.err = caso.err
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticion(http.MethodPost, RutaSeleccion, cuerpoValido()))
			linea := buf.String()
			if w.Code != caso.estado || strings.Contains(linea, "DETALLE_PRIVADO") || strings.Contains(w.Body.String(), "selector_") {
				t.Fatalf("estado=%d linea=%q cuerpo=%s", w.Code, linea, w.Body)
			}
			if caso.clase == "" {
				if linea != "" {
					t.Fatalf("un %d no debe registrar clase: %q", w.Code, linea)
				}
				return
			}
			if !strings.Contains(linea, "vec_admin_seleccion_no_disponible") || !strings.Contains(linea, "clase="+caso.clase) ||
				!strings.Contains(linea, "ruta=seleccion") {
				t.Fatalf("línea sin clase %q: %q", caso.clase, linea)
			}
		})
	}
}

func TestClaseSelectorConservaLaMasInternaYLaCausa(t *testing.T) {
	interna := adminperfiles.ConClaseSelector("selector_instante", api.ErrConfiguracionIncompleta)
	externa := adminperfiles.ConClaseSelector("selector_resultado_invalido", interna)
	if adminperfiles.ClaseFalloSelector(externa) != "selector_instante" || !errors.Is(externa, api.ErrConfiguracionIncompleta) {
		t.Fatal("se perdió la clase interna o la causa")
	}
	for _, clase := range []string{"selector_sql_p0001", "selector_sql_P00011", "otra", "selector_sql_"} {
		if adminperfiles.ClaseSelectorAdmitida(clase) || adminperfiles.ConClaseSelector(clase, api.ErrConfiguracionIncompleta) != api.ErrConfiguracionIncompleta {
			t.Fatalf("clase fuera de la lista admitida: %q", clase)
		}
	}
}
