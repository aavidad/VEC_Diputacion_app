package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/shared/telemetria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestFalloComponenteArranqueConservaCausaYEtapaInterior(t *testing.T) {
	sentinel := errors.New("fallo sintético")
	pg := &pgconn.PgError{Code: "42501", Message: "dato privado"}
	causa := errors.Join(sentinel, pg)
	interior := marcarFalloComponenteArranque("auditoria_intentos", causa)
	exterior := marcarFalloComponenteArranque("servidor_desarrollo", fmt.Errorf("composicion: %w", interior))
	var obtenido *pgconn.PgError
	if !errors.Is(exterior, sentinel) || !errors.As(exterior, &obtenido) || obtenido != pg ||
		ComponenteFalloArranque(exterior) != "auditoria_intentos" {
		t.Fatal("el wrapper perdió causa, tipo o componente interior")
	}
	if got := ComponenteFalloArranque(marcarFalloComponenteArranque("ruta/privada", sentinel)); got != "sin_etiqueta" {
		t.Fatalf("etiqueta no cerrada: %q", got)
	}
	if got := ComponenteFalloArranque(sentinel); got != "sin_etiqueta" {
		t.Fatalf("error ajeno tomó una etapa: %q", got)
	}
}

func TestFalloCompartidoDistingueDosConstructores(t *testing.T) {
	identidades := marcarFalloComponenteArranque("identidades_preferencias", ErrComposicionDesarrolloIncompleta)
	categorias := marcarFalloComponenteArranque("categorias_profesionales", ErrComposicionDesarrolloIncompleta)
	if !errors.Is(identidades, ErrComposicionDesarrolloIncompleta) ||
		!errors.Is(categorias, ErrComposicionDesarrolloIncompleta) ||
		ComponenteFalloArranque(identidades) != "identidades_preferencias" ||
		ComponenteFalloArranque(categorias) != "categorias_profesionales" {
		t.Fatal("un mismo centinela perdió el punto de composición")
	}
}

func TestConstructoresHTTPConservanComponenteYErrorOriginal(t *testing.T) {
	if _, err := NuevoServidorHTTPSupervisado(config.Config{}, nil); !errors.Is(err, ErrEmisorIncidenciasRequerido) ||
		ComponenteFalloArranque(err) != "emisor_incidencias" {
		t.Fatalf("emisor sin etiqueta exacta: %v", err)
	}
	if _, _, err := nuevoServidorDesarrollo(config.Config{}, io.Discard, nil,
		ConfiguracionIncorporacionDesarrollo{}, ConfiguracionIncorporacionDesarrollo{}); !errors.Is(err, ErrComposicionDesarrolloIncompleta) ||
		ComponenteFalloArranque(err) != "configuracion_incorporacion" {
		t.Fatalf("incorporacion sin etiqueta exacta: %v", err)
	}
	if _, err := NuevaComposicionSeguridadDesarrollo(config.Config{Address: "0.0.0.0:8443"}, io.Discard); !errors.Is(err, ErrActivacionDesarrolloInvalida) ||
		ComponenteFalloArranque(err) != "red_local" {
		t.Fatalf("red local sin etiqueta exacta: %v", err)
	}
}

func TestConstructorExternoEtiquetaRegistroYRed(t *testing.T) {
	if _, err := nuevoServidorPortalExternoDesarrollo(config.Config{}, nil, nil); !errors.Is(err, ErrRegistroArranqueDesarrollo) ||
		ComponenteFalloArranque(err) != "registro_arranque" {
		t.Fatalf("registro externo sin etiqueta exacta: %v", err)
	}
	if _, err := nuevoServidorPortalExternoDesarrollo(config.Config{Address: "0.0.0.0:8443"}, io.Discard, nil); !errors.Is(err, ErrActivacionDesarrolloInvalida) ||
		ComponenteFalloArranque(err) != "red_local" {
		t.Fatalf("red externa sin etiqueta exacta: %v", err)
	}
}

func TestFuentesConfiguradasDistinguenOrganizacionYRPTSinPublicarRutas(t *testing.T) {
	rutaPrivada := filepath.Join(t.TempDir(), "fuente-sintetica-ausente.json")
	for _, caso := range []struct {
		nombre, organizacion, rpt, componente string
		causa                                 error
	}{
		{"organizacion", rutaPrivada, rutaRPTRepositorioPrueba, "fuente_organizacion", vecdomain.ErrCatalogoConfigurableInvalido},
		{"rpt", rutaEstructuraRepositorioPrueba, rutaPrivada, "catalogo_rpt", errCatalogoRPTDesarrolloInvalido},
		{"ambas_ausentes_primero_organizacion", rutaPrivada, rutaPrivada, "fuente_organizacion", vecdomain.ErrCatalogoConfigurableInvalido},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := nuevoCatalogoDesarrollo(caso.organizacion, caso.rpt)
			if err == nil || !errors.Is(err, caso.causa) {
				t.Fatalf("la causa original no se conserva: %v", err)
			}
			envuelto := marcarFalloComponenteArranque("contratacion_temporal", fmt.Errorf("composicion: %w", err))
			if ComponenteFalloArranque(envuelto) != caso.componente ||
				telemetria.ClaseErrorArranque(envuelto) != "configuracion" ||
				telemetria.MensajeErrorArranque(envuelto) != "composicion: "+mensajeFuenteConfiguracionArranque ||
				!errors.Is(envuelto, caso.causa) {
				t.Fatalf("diagnostico o causa no conservados: componente=%s clase=%s mensaje=%s",
					ComponenteFalloArranque(envuelto), telemetria.ClaseErrorArranque(envuelto), telemetria.MensajeErrorArranque(envuelto))
			}
			if strings.Contains(err.Error(), rutaPrivada) || strings.Contains(envuelto.Error(), rutaPrivada) ||
				strings.Contains(telemetria.MensajeErrorArranque(envuelto), rutaPrivada) {
				t.Fatal("la ruta configurada aparece en el diagnostico")
			}
		})
	}
}

func TestFuenteOrganizacionConCentroNoAdmitidoPorCTConservaDiagnosticoDeFuente(t *testing.T) {
	contenido, err := os.ReadFile(rutaEstructuraRepositorioPrueba)
	if err != nil {
		t.Fatal(err)
	}
	var paquete map[string]any
	if err := json.Unmarshal(contenido, &paquete); err != nil {
		t.Fatal(err)
	}
	catalogo, ok := paquete["catalogo"].(map[string]any)
	if !ok {
		t.Fatal("fixture sin catalogo")
	}
	entradas, ok := catalogo["entradas"].([]any)
	if !ok {
		t.Fatal("fixture sin entradas")
	}
	cambiado := false
	for _, valor := range entradas {
		entrada, ok := valor.(map[string]any)
		if !ok {
			t.Fatal("entrada de fixture invalida")
		}
		atributos, ok := entrada["atributos"].(map[string]any)
		if ok && atributos["tipo"] == "centro" {
			entrada["clave"] = "ab" // Válida para Personal; demasiado corta para la referencia CT.
			cambiado = true
			break
		}
	}
	if !cambiado {
		t.Fatal("fixture sin centro")
	}
	contenido, err = json.Marshal(paquete)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "organizacion-sintetica.json")
	if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = nuevoCatalogoDesarrollo(ruta, rutaRPTRepositorioPrueba)
	if !errors.Is(err, errCatalogosAltaContratacionTemporalDesarrolloNoDisponibles) ||
		ComponenteFalloArranque(err) != "fuente_organizacion" ||
		telemetria.ClaseErrorArranque(err) != "configuracion" ||
		telemetria.MensajeErrorArranque(err) != mensajeFuenteConfiguracionArranque ||
		strings.Contains(err.Error(), ruta) {
		t.Fatalf("diagnostico de fuente no conservado: componente=%s clase=%s mensaje=%s",
			ComponenteFalloArranque(err), telemetria.ClaseErrorArranque(err), telemetria.MensajeErrorArranque(err))
	}
}
