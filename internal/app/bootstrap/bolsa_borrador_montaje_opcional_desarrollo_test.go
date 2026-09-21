package bootstrap

import (
	"net/http"
	"testing"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
)

func TestBorradorLlamamientoNoSeComponeSinBolsaPostgreSQL(t *testing.T) {
	if debeComponerBorradorLlamamientoDesarrollo(config.Config{}) {
		t.Fatal("B-BACK se activó sin opt-in explícito")
	}
}

func TestBorradorLlamamientoNoSeActivaPorLaConexionHistoricaDeBolsa(t *testing.T) {
	t.Setenv(config.EnvBolsaLlamamientosDatabaseURL, "postgres://bolsa-historica")
	cfg := config.Load()
	if cfg.BolsaBorradoresEnabled || debeComponerBorradorLlamamientoDesarrollo(cfg) {
		t.Fatal("la conexión histórica de Bolsa no debe activar B-BACK")
	}
}

func TestBorradorLlamamientoOptInExplicitoActivaElMontajeFailClosed(t *testing.T) {
	t.Setenv(config.EnvBolsaBorradoresEnabled, "true")
	cfg := config.Load()
	if !cfg.BolsaBorradoresEnabled || !debeComponerBorradorLlamamientoDesarrollo(cfg) {
		t.Fatal("el opt-in explícito no activó B-BACK")
	}
}

func TestRutasBorradorLlamamientoExigenPerfilBolsaYMetodosExactos(t *testing.T) {
	perfilBolsa := "prf_bolsa_bback_0123456789abcdef"
	fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(perfilBolsa)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		metodo, ruta string
		admite       bool
	}{
		{http.MethodPost, bolsahttp.RutaBorradoresLlamamiento, true},
		{http.MethodGet, bolsahttp.RutaBorradoresLlamamiento + "/borrador-llamamiento:alta:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{http.MethodGet, bolsahttp.RutaBorradoresLlamamiento, false},
		{http.MethodPost, bolsahttp.RutaBorradoresLlamamiento + "/detalle", false},
		{http.MethodHead, bolsahttp.RutaBorradoresLlamamiento + "/detalle", false},
	} {
		descriptor, admite := catalogo.resolver(caso.metodo, caso.ruta)
		if admite != caso.admite || (admite && (len(descriptor.PerfilesActivosRef) != 1 || descriptor.PerfilesActivosRef[0] != perfilBolsa)) {
			t.Fatalf("%s %s: descriptor=%+v admite=%v", caso.metodo, caso.ruta, descriptor, admite)
		}
	}
}
