package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

type validadorMotivoAlegacionPrueba struct{ catalogo string }

func (v validadorMotivoAlegacionPrueba) ValidarReferenciaMotivoAutorizacionV2(_ context.Context, m core.ReferenciaEntradaCatalogo, _ time.Time) error {
	if m.CatalogoID != v.catalogo {
		return core.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

func TestMotivoAlegacionesSeparadoYPrivado(t *testing.T) {
	directorio := t.TempDir()
	if err := os.Mkdir(filepath.Join(directorio, "identidad"), 0o700); err != nil {
		t.Fatal(err)
	}
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_alegaciones_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_" + strings.Repeat("b", 32)}
	contenido, err := json.Marshal(configuracionMotivoAlegacionesBolsa{Version: 1, MotivoPresentacion: motivo})
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(directorio, rutaMotivoAlegacionesBolsa)
	if err = os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	obtenido, err := leerMotivoAlegacionesBolsa(directorio)
	if err != nil || obtenido != motivo {
		t.Fatalf("motivo propio no cargado: %v", err)
	}
	despachador := validadoresMotivoInscripcionAlegaciones{
		inscripcion:         validadorMotivoAlegacionPrueba{motivoPresentarInscripcionBolsaDesarrollo().CatalogoID},
		alegaciones:         validadorMotivoAlegacionPrueba{motivo.CatalogoID},
		catalogoInscripcion: motivoPresentarInscripcionBolsaDesarrollo().CatalogoID,
		catalogoAlegaciones: motivo.CatalogoID,
	}
	if err := despachador.ValidarReferenciaMotivoAutorizacionV2(context.Background(), motivo, time.Now()); err != nil {
		t.Fatal(err)
	}
	motivo.CatalogoID = "motivos_ajenos"
	if err := despachador.ValidarReferenciaMotivoAutorizacionV2(context.Background(), motivo, time.Now()); !errors.Is(err, core.ErrSolicitudAutorizacionInvalida) {
		t.Fatal("catálogo lateral admitido")
	}
	if err := os.Chmod(ruta, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := leerMotivoAlegacionesBolsa(directorio); err == nil {
		t.Fatal("material legible por terceros admitido")
	}
}
