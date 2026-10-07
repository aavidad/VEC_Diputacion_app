package bootstrap

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/config"
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
