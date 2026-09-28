package reglas

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	vecreglas "vec-diputacion-granada/internal/vec/reglas"
)

func TestFuentePoliticaContactosNoPromueveDemoEnAviso(t *testing.T) {
	fuente := NuevaFuentePoliticaContactos(NuevosIntentosContacto(resolutorIntentosPrueba(t, rutaCatalogoIntentosPrueba)), vecreglas.MunicipioSedeDiputacion)
	if _, err := fuente.ObtenerPublicada(t.Context(), "bolsa:administrativo:2026"); !errors.Is(err, ErrFuentePoliticaContactosNoDisponible) {
		t.Fatalf("politica de aviso publicada = %v", err)
	}
}

func TestFuentePoliticaContactosPreparaSoloCatalogoEstrictoCoherente(t *testing.T) {
	contenido, err := os.ReadFile(rutaCatalogoIntentosPrueba)
	if err != nil {
		t.Fatal(err)
	}
	marcador := `"zona_horaria": "Europe/Madrid",
          "control": "advertir"`
	if strings.Count(string(contenido), marcador) != 1 {
		t.Fatal("el ejemplo cambio; no alterar otra regla")
	}
	ruta := filepath.Join(t.TempDir(), "reglas.json")
	if err := os.WriteFile(ruta, []byte(strings.Replace(string(contenido), marcador,
		`"zona_horaria": "Europe/Madrid",
          "control": "impedir"`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	fuente := NuevaFuentePoliticaContactos(NuevosIntentosContacto(resolutorIntentosPrueba(t, ruta)), vecreglas.MunicipioSedeDiputacion)
	resultado, err := fuente.ObtenerPublicada(t.Context(), "bolsa:administrativo:2026")
	if err != nil || resultado.DesdeMinuto != 540 || resultado.HastaMinuto != 840 ||
		resultado.ControlFranja != dominiobolsa.ControlReglaImpedir ||
		resultado.HuellaFuenteSHA256 == "" || resultado.CatalogoHuellaSHA256 == "" {
		t.Fatalf("politica estricta = %+v, %v", resultado, err)
	}
}
