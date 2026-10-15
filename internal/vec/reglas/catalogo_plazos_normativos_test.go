package reglas

import (
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
)

func TestPlazosCTPublicadosV3RespetanLaNorma(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogoProvisionable("../../../data/catalogos/contratacion-temporal/reglas.v3.json")
	if err != nil {
		t.Fatal(err)
	}
	instante := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	resolutor, err := NuevoResolutor(Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: CatalogoContratacionTemporal,
		ModuloID: ModuloContratacionTemporal, Reloj: relojFijo(instante), MunicipioSede: MunicipioSedeDiputacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	catalogo, _, _, err := resolutor.CatalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if catalogo.Version != 3 {
		t.Fatalf("se esperaba la nueva publicación, llegó %d", catalogo.Version)
	}
	if err := ValidarCatalogoBaseReglas(catalogo); err != nil {
		t.Fatalf("el catálogo publicado no es consumible: %v", err)
	}
	for clave, cantidad := range map[string]int{CTPlazoFiscalizacion: 10, CTPlazoSubsanacion: 15} {
		regla, err := resolutor.Regla(t.Context(), clave)
		if err != nil || regla.Cantidad != cantidad || regla.Unidad != UnidadDiasHabiles || regla.Origen != OrigenReglamento || regla.Edicion != nil {
			t.Fatalf("plazo normativo inesperado: %+v, %v", regla, err)
		}
		if regla.Articulo == "" || regla.Norma == "" {
			t.Fatalf("falta procedencia normativa: %+v", regla)
		}
		for _, cambio := range []SolicitudCambioAjuste{
			{ReglaClave: clave, Campo: CampoCantidad, Nuevo: "9"},
			{ReglaClave: clave, Campo: CampoCantidadUrgente, Nuevo: "4"},
			{ReglaClave: clave, Campo: CampoUnidad, Nuevo: string(UnidadDiasNaturales)},
		} {
			_, err := PrepararCambioAjustes(catalogo, instante, 0, VersionAjustes{}, false, []SolicitudCambioAjuste{cambio})
			if !errors.Is(err, ErrAjusteInvalido) {
				t.Fatalf("se admitió alterar un plazo normativo con %+v: %v", cambio, err)
			}
		}
	}
	fiscalizacion, err := resolutor.Regla(t.Context(), CTPlazoFiscalizacion)
	if err != nil || fiscalizacion.Atributos[AtributoCantidadUrgente] != "5" {
		t.Fatalf("se perdió el plazo urgente de cinco días: %+v, %v", fiscalizacion, err)
	}
	// Las reglas de organización que no fija esta norma conservan su editor.
	analisis, err := resolutor.Regla(t.Context(), CTPlazoAnalisis)
	if err != nil || !analisis.Edicion.Admite(CampoCantidad) {
		t.Fatalf("se bloqueó un plazo ajeno a esta norma: %+v, %v", analisis, err)
	}
}
