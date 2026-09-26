package bootstrap

import (
	"errors"
	"reflect"
	"testing"

	"vec-diputacion-granada/config"
)

// seleccionMaterialCTCompletaDesarrollo enciende todos los consumidores.
// La prueba de reflexión de abajo detecta un campo nuevo que no se encienda.
func seleccionMaterialCTCompletaDesarrollo() seleccionMaterialCTDesarrollo {
	return seleccionMaterialCTDesarrollo{
		borradoresBolsa: true, miBolsa: true, portalCandidato: true,
		dietas: true, cronos: true, documentos: true, cronosResolucion: true, cronosAvisos: true,
		fichaPropiaPersonal: true, firmaDocumento: true, seguimientoCese: true, personalB2: true, cancelacion: true,
	}
}

func TestSeleccionCompletaEnciendeTodosLosConsumidores(t *testing.T) {
	v := reflect.ValueOf(seleccionMaterialCTCompletaDesarrollo())
	for i := range v.NumField() {
		if !v.Field(i).Bool() {
			t.Fatalf("la selección completa no enciende %s", v.Type().Field(i).Name)
		}
	}
}

// Con todos los selectores encendidos (portal del candidato y seguimiento de
// cese incluidos) cada audiencia compuesta debe ser publicable por el
// gobierno de CT; si no, el arranque caería al publicar su clave.
func TestAudienciasSeleccionCompletaPublicablesPorElGobiernoCT(t *testing.T) {
	descriptores := descriptoresMaterialSeleccionadosCTDesarrollo(seleccionMaterialCTCompletaDesarrollo())
	for _, d := range descriptores {
		if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
			t.Errorf("audiencia compuesta no publicable por CT: %s", d.Audiencia)
		}
	}
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptores); err != nil {
		t.Fatal("la selección completa colisiona en el catálogo común", err)
	}
}

// El portal publica un proveedor por acción propia: pausa, reactivación,
// respuesta, disposición y confirmación del contacto.
func TestAudienciasPortalCandidatoPublicablesPorElGobiernoCT(t *testing.T) {
	for _, par := range accionesPropiasPortalDesarrollo() {
		if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(par[1]) {
			t.Errorf("acción propia %s con audiencia no publicable por CT: %s", par[0], par[1])
		}
	}
	for _, d := range append(descriptoresMaterialPortalCandidatoDesarrollo(), descriptoresMaterialContactoPropioDesarrollo()...) {
		if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
			t.Errorf("audiencia del portal no publicable por CT: %s", d.Audiencia)
		}
	}
}

func TestAudienciasSeguimientoCeseYFirmaPublicablesPorElGobiernoCT(t *testing.T) {
	for _, d := range append(append(descriptoresMaterialSeguimientoCeseDesarrollo(), descriptorMaterialFirmaDocumentoCTDesarrollo()), descriptoresMaterialCancelacionCTDesarrollo()...) {
		if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
			t.Errorf("audiencia de cese o firma no publicable por CT: %s", d.Audiencia)
		}
	}
}

// Un "true" fuera de la doble llave solo lo rechaza la raíz que compone la
// capacidad; un valor mal escrito lo rechazan todas.
func TestValidacionSelectoresDespliegueBolsaCT(t *testing.T) {
	encendido := config.Config{BolsaPortalCandidatoEnabled: "true", CTSeguimientoCeseEnabled: "true"}
	if err := validarValorSelectoresDespliegueBolsaCT(encendido); err != nil {
		t.Fatalf("el valor válido no debe fallar fuera de vec-server: %v", err)
	}
	if err := validarSelectoresDespliegueBolsaCT(encendido); !errors.Is(err, config.ErrConfiguracionBolsaPortalCandidatoActivacion) {
		t.Fatalf("encendido sin doble llave = %v", err)
	}
	if err := validarSelectoresDespliegueBolsaCT(config.Config{CTSeguimientoCeseEnabled: "true"}); !errors.Is(err, config.ErrConfiguracionCTSeguimientoCeseActivacion) {
		t.Fatalf("cese sin doble llave = %v", err)
	}
	for _, cfg := range []config.Config{{BolsaPortalCandidatoEnabled: "TRUE"}, {CTSeguimientoCeseEnabled: "1"}} {
		if validarValorSelectoresDespliegueBolsaCT(cfg) == nil || validarSelectoresDespliegueBolsaCT(cfg) == nil {
			t.Fatalf("valor inválido admitido: %+v", cfg)
		}
	}
	if err := validarSelectoresDespliegueBolsaCT(config.Config{}); err != nil {
		t.Fatalf("la ausencia equivale a apagado: %v", err)
	}
}

// Pedir el portal del candidato sin su catálogo de reglas o sin poder componer
// «Mi bolsa» detiene el arranque en lugar de no montar sus rutas en silencio.
func TestPortalCandidatoPedidoSinSusRequisitosDetieneElArranque(t *testing.T) {
	cfg := config.Config{BolsaPortalCandidatoEnabled: "true", ExecutionProfile: config.ExecutionProfileDevelopment,
		AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
	if err := validarSelectoresDespliegueBolsaCT(cfg); !errors.Is(err, config.ErrConfiguracionBolsaPortalCandidatoActivacion) {
		t.Fatalf("sin reglas de Bolsa = %v", err)
	}
	cfg.ReglasEjemplo.BolsaSourcePath = "bolsa.json"
	if err := validarSelectoresDespliegueBolsaCT(cfg); err != nil {
		t.Fatalf("con reglas de Bolsa el selector es válido: %v", err)
	}
	if _, err := seleccionMaterialCTDesarrolloDesdeConfig(cfg); !errors.Is(err, config.ErrConfiguracionBolsaPortalCandidatoActivacion) {
		t.Fatalf("sin Mi bolsa = %v", err)
	}
}

func TestDiagnosticoMigracionesPortalCandidato(t *testing.T) {
	completo := estadoMigracionesPortalCandidato{ad384: true, ad386: true, bolsa29: true, bolsa30: true, bolsa40: true}
	if err := completo.diagnostico(); err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		quitar func(*estadoMigracionesPortalCandidato)
		error  error
	}{
		{func(e *estadoMigracionesPortalCandidato) { e.ad384 = false }, ErrPortalCandidatoFaltaAD384},
		{func(e *estadoMigracionesPortalCandidato) { e.bolsa29 = false }, ErrPortalCandidatoFaltaBolsa29},
		{func(e *estadoMigracionesPortalCandidato) { e.bolsa30 = false }, ErrPortalCandidatoFaltaBolsa30},
		{func(e *estadoMigracionesPortalCandidato) { e.ad386 = false }, ErrPortalCandidatoFaltaAD386},
		{func(e *estadoMigracionesPortalCandidato) { e.bolsa40 = false }, ErrPortalCandidatoFaltaBolsa40},
	}
	for _, c := range casos {
		e := completo
		c.quitar(&e)
		err := e.diagnostico()
		if !errors.Is(err, c.error) || !errors.Is(err, ErrPortalCandidatoMigracionesNoDisponibles) {
			t.Fatalf("diagnóstico = %v; se esperaba %v", err, c.error)
		}
	}
	if err := comprobarMigracionesPortalCandidatoDesarrollo(t.Context(), nil); !errors.Is(err, ErrPortalCandidatoMigracionesNoDisponibles) {
		t.Fatalf("sin pool = %v", err)
	}
}
