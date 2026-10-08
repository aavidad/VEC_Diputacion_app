package reglas

import (
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
)

func TestProyeccionUsaCabezaAutorizadaSinVolverALeerAjustes(t *testing.T) {
	resolutor := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, nil)
	base, _, instante, err := resolutor.CatalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	vigente := versionAjustes(t, 1, instante.Add(-time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "7"},
	})
	proyectadas, err := ProyectarReglasConAjustes(base, instante, &vigente)
	if err != nil {
		t.Fatal(err)
	}
	for _, regla := range proyectadas {
		if regla.Clave != CTPlazoFiscalizacion {
			continue
		}
		if regla.Cantidad != 7 || regla.Ajuste == nil || regla.Ajuste.Version != 1 ||
			regla.AjusteNoAplicable || regla.ReferenciaEntrada.CatalogoID != vigente.CatalogoID {
			t.Fatalf("proyección no aplicó cabeza CT148: %+v", regla)
		}
		return
	}
	t.Fatal("falta c03")
}

func TestProyeccionAisladaMarcaAjusteIncompatible(t *testing.T) {
	resolutor := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, nil)
	base, _, instante, err := resolutor.CatalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	vigente := versionAjustes(t, 2, instante.Add(-time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "61"},
	})
	proyectadas, err := ProyectarReglasConAjustes(base, instante, &vigente)
	if err != nil {
		t.Fatal(err)
	}
	marcada, otra := false, false
	for _, regla := range proyectadas {
		switch regla.Clave {
		case CTPlazoFiscalizacion:
			marcada = regla.AjusteNoAplicable && regla.Ajuste == nil
		case CTPlazoSubsanacion:
			otra = !regla.AjusteNoAplicable && regla.Cantidad == 10
		}
	}
	if !marcada || !otra {
		t.Fatalf("ajuste inválido contaminó catálogo: marcada=%v otra=%v", marcada, otra)
	}
}

func TestProyeccionRechazaCabezaFuturaYUsaVersionEfectiva(t *testing.T) {
	resolutor := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, nil)
	base, _, instante, err := resolutor.CatalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	vigente := versionAjustes(t, 1, instante.Add(-time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "7"},
	})
	futura := versionAjustes(t, 2, instante.Add(time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "9"},
	})
	if _, err := ProyectarReglasConAjustes(base, instante, &futura); !errors.Is(err, ErrAjustesNoDisponibles) {
		t.Fatalf("cabeza futura aplicada antes del efecto: %v", err)
	}
	reglas, err := ProyectarReglasConAjustes(base, instante, &vigente)
	if err != nil {
		t.Fatal(err)
	}
	for _, regla := range reglas {
		if regla.Clave == CTPlazoFiscalizacion {
			if regla.Cantidad != 7 || regla.Ajuste == nil || regla.Ajuste.Version != 1 {
				t.Fatalf("versión efectiva omitida: %+v", regla)
			}
			return
		}
	}
	t.Fatal("falta la regla proyectada")
}

func TestProyeccionBolsaV6AplicaSoloAjusteAdmitido(t *testing.T) {
	// El catálogo v6 pertenece al corte de inventario Bolsa. Esta prueba
	// conjunta verifica que la proyección compartida acepta su b05 editable.
	consulta, err := fichero.NuevaConsultaCatalogos("../../../data/demo/reglas/bolsa_reglas.rrhh-20261008.v6.json")
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := NuevoResolutor(Configuracion{Consulta: consulta, Metadatos: consulta,
		CatalogoID: CatalogoBolsa, ModuloID: ModuloBolsa,
		Reloj:         relojFijo(time.Date(2026, 10, 8, 8, 0, 0, 0, time.UTC)),
		MunicipioSede: MunicipioSedeDiputacion})
	if err != nil {
		t.Fatal(err)
	}
	base, _, instante, err := resolutor.CatalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	ajustes := map[string]map[string]string{
		BolsaPlazoRespuesta: {CampoCantidad: "2", CampoUnidad: string(UnidadDiasNaturales), CampoComputo: string(ComputoCivil)},
	}
	huella, err := HuellaAjustes(ajustes)
	if err != nil {
		t.Fatal(err)
	}
	vigente := VersionAjustes{CatalogoID: CatalogoAjustesDe(CatalogoBolsa), Version: 1,
		HuellaSHA256: huella, VigenteDesde: instante.Add(-time.Hour), Ajustes: ajustes}
	reglas, err := ProyectarReglasConAjustes(base, instante, &vigente)
	if err != nil {
		t.Fatal(err)
	}
	var encontrada bool
	for _, regla := range reglas {
		if regla.Clave != BolsaPlazoRespuesta {
			if regla.Ajuste != nil || regla.AjusteNoAplicable {
				t.Fatalf("ajuste b05 contaminó otra regla: %s", regla.Clave)
			}
			continue
		}
		encontrada = true
		if regla.Cantidad != 2 || regla.Unidad != UnidadDiasNaturales || regla.Computo != ComputoCivil ||
			regla.Ajuste == nil || regla.Ajuste.Version != 1 || regla.ReferenciaEntrada.CatalogoID != vigente.CatalogoID {
			t.Fatalf("b05 no recibió la versión efectiva: %+v", regla)
		}
	}
	if !encontrada {
		t.Fatal("falta b05 en el catálogo de Bolsa")
	}
	// Las opciones del catálogo admiten cada campo por separado; el motor
	// revalida además la regla completa después de combinarlos.
	ajustes[BolsaPlazoRespuesta][CampoUnidad] = string(UnidadHoras)
	huella, _ = HuellaAjustes(ajustes)
	vigente.HuellaSHA256, vigente.Ajustes = huella, ajustes
	reglas, err = ProyectarReglasConAjustes(base, instante, &vigente)
	if err != nil {
		t.Fatal(err)
	}
	for _, regla := range reglas {
		if regla.Clave == BolsaPlazoRespuesta && (!regla.AjusteNoAplicable || regla.Ajuste != nil) {
			t.Fatalf("combinación no permitida aplicada: %+v", regla)
		}
	}
}
