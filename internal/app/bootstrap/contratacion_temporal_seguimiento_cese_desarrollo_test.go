package bootstrap

import (
	"context"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/reglas"
)

const rutaCausasCeseEjemploPrueba = "../../../data/demo/reglas/ct_causas_cese.demo.json"

// Las opciones de cese, cierre y modificación salen de los catálogos de
// ejemplo: cambiar una causa, un justificante, las condiciones del cierre o
// la fase de vuelta es editar el catálogo, no el código.
func TestFuenteReglasSeguimientoLeeLosCatalogosDeEjemplo(t *testing.T) {
	consultaReglas, err := fichero.NuevaConsultaCatalogos(rutaReglasCTEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{Consulta: consultaReglas, Metadatos: consultaReglas,
		CatalogoID: reglas.CatalogoContratacionTemporal, ModuloID: reglas.ModuloContratacionTemporal, Reloj: relojPresentacionReglasEjemplo})
	if err != nil {
		t.Fatal(err)
	}
	causas, err := fichero.NuevaConsultaCatalogos(rutaCausasCeseEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	motivos, err := fichero.NuevaConsultaCatalogos(rutaMotivosCTEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	f := fuenteReglasSeguimientoDesarrollo{reglas: resolutor, causas: causas, motivos: motivos}
	ahora := relojPresentacionReglasEjemplo.ahora
	ctx := context.Background()

	lista, politica, err := f.CausasCese(ctx, ahora)
	if err != nil || len(lista) != 7 || !politica.ValidaEn(ahora) || politica.DefinicionRef != "causas_cese_contratacion_temporal" {
		t.Fatalf("causas: %d %+v %v", len(lista), politica, err)
	}
	if lista[0].Clave != "fin_sustitucion" || lista[0].JustificanteTipo != "comunicacion_reincorporacion" {
		t.Fatalf("primera causa: %+v", lista[0])
	}
	cierre, politica, err := f.ReglaCierre(ctx, ahora)
	if err != nil || !politica.ValidaEn(ahora) || !domain.CondicionesCierreValidas(cierre.Condiciones) || len(cierre.Condiciones) != 2 {
		t.Fatalf("cierre: %+v %v", cierre, err)
	}
	modificacion, politica, err := f.ReglaModificacion(ctx, ahora)
	if err != nil || !politica.ValidaEn(ahora) || modificacion.FaseRetorno != domain.FaseFiscalizacion || len(modificacion.Motivos) != 2 {
		t.Fatalf("modificación: %+v %v", modificacion, err)
	}
	opciones := ports.OpcionesSeguimiento{Causas: lista, Condiciones: cierre.Condiciones, FaseRetorno: modificacion.FaseRetorno, Motivos: modificacion.Motivos}
	if !opciones.Validas() {
		t.Fatal("las opciones de los catálogos de ejemplo no son válidas")
	}
}

func TestSeguimientoCeseSoloSeComponeConSelectorYLosTresCatalogos(t *testing.T) {
	cfg := config.Config{ReglasEjemplo: config.ConfiguracionReglasEjemplo{CTSourcePath: "ct.json", CausasCeseSourcePath: "causas.json"}}
	if seguimientoCeseSolicitado(cfg) {
		t.Fatal("sin motivos de rectificación no se compone")
	}
	cfg.CTAnalisisMotivosSourcePath = "motivos.json"
	if seguimientoCeseSolicitado(cfg) {
		t.Fatal("sin pedirlo expresamente no se compone aunque estén los catálogos")
	}
	cfg.CTSeguimientoCeseEnabled = "true"
	if seguimientoCeseSolicitado(cfg) {
		t.Fatal("fuera de la doble llave de desarrollo no se compone")
	}
	cfg.ExecutionProfile, cfg.AuthMode, cfg.DevelopmentGuard = config.ExecutionProfileDevelopment, config.AuthModeDevelopment, config.DevelopmentGuardAcknowledgement
	if !seguimientoCeseSolicitado(cfg) {
		t.Fatal("pedido, con doble llave y con los tres catálogos se compone")
	}
	rutas, err := nuevasRutasSeguimientoCeseDesarrollo(&DependenciasCT{cfg: config.Config{}}, nil)
	if err != nil || rutas != nil {
		t.Fatalf("sin catálogos la conducta es la de hoy: %v %v", rutas, err)
	}
	for _, o := range operacionesSeguimientoCeseDesarrollo() {
		if !rutaSeguimientoCeseDesarrollo(o.ruta) || motivoSeguimientoCeseDesarrollo(o.ruta).Validar() != nil {
			t.Fatalf("operación mal declarada: %+v", o)
		}
	}
}
