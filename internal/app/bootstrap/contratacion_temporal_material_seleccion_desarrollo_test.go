package bootstrap

import (
	"errors"
	"reflect"
	"testing"

	"vec-diputacion-granada/config"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
)

// seleccionMaterialCTCompletaDesarrollo enciende todos los consumidores.
// La prueba de reflexión de abajo detecta un campo nuevo que no se encienda.
func seleccionMaterialCTCompletaDesarrollo() seleccionMaterialCTDesarrollo {
	return seleccionMaterialCTDesarrollo{
		borradoresBolsa: true, miBolsa: true, portalCandidato: true,
		dietas: true, cronos: true, documentos: true, cronosResolucion: true, cronosAvisos: true,
		fichaPropiaPersonal: true, exportacionServiciosPersonal: true, historiaServiciosPersonal: true, historiaRelacionesPersonal: true, firmaDocumento: true, seguimientoCese: true, personalB2: true, cancelacion: true,
		incorporacionAcreditada: true, incorporacionB2: true, reincorporacionTitular: true, politicaOfertas: true,
		plantillasCatalogo: true, plantillasDocumental: true, ajustesReglasCT: true,
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

func TestIncorporacionB2SeleccionaNueveAudienciasNominales(t *testing.T) {
	esperadas := map[string]bool{
		bolsa.AudienciaConsultaAnclajeAceptacionCT:       true,
		personal.AudienciaPlanIncorporacionCT:            true,
		ct.AudienciaRegistrarPlanNominalB2:               true,
		ct.AudienciaLeerPlanNominalB2:                    true,
		ct.AudienciaConfirmarOrigenB2:                    true,
		bolsa.AudienciaConsultaPersonaAceptacionCT:       true,
		ct.AudienciaConsultarVinculoCategoriaRPT:         true,
		ct.AudienciaConsultarPublicacionCategoriaRPT:     true,
		"vec_catalogos_configurables.usos_categorias.v1": true,
	}
	nuevas := descriptoresMaterialIncorporacionB2()
	if len(nuevas) != len(esperadas) {
		t.Fatalf("audiencias B2: %d", len(nuevas))
	}
	for _, d := range nuevas {
		if !esperadas[d.Audiencia] || !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
			t.Fatalf("audiencia B2 inesperada o no gobernada: %s", d.Audiencia)
		}
		delete(esperadas, d.Audiencia)
	}
	if len(esperadas) != 0 || audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia("vec_contratacion_temporal.incorporacion_personal.ajena.v1") {
		t.Fatal("lista de audiencias B2 abierta o incompleta")
	}
	apagada := descriptoresMaterialSeleccionadosCTDesarrollo(seleccionMaterialCTDesarrollo{})
	encendida := descriptoresMaterialSeleccionadosCTDesarrollo(seleccionMaterialCTDesarrollo{incorporacionB2: true})
	if len(encendida) != len(apagada)+len(nuevas) {
		t.Fatal("el selector B2 no añade exactamente nueve audiencias")
	}
	for _, d := range nuevas {
		for _, anterior := range apagada {
			if anterior.Audiencia == d.Audiencia {
				t.Fatalf("B2 publicado con selector apagado: %s", d.Audiencia)
			}
		}
	}
}

func TestSeleccionMiBolsaIncluyeMaterialHistorialPropio(t *testing.T) {
	audiencia := descriptorMaterialHistorialMiBolsaDesarrollo().Audiencia
	coincidencias := 0
	for _, d := range descriptoresMaterialSeleccionadosCTDesarrollo(seleccionMaterialCTDesarrollo{miBolsa: true}) {
		if d.Audiencia == audiencia {
			coincidencias++
		}
	}
	if coincidencias != 1 {
		t.Fatalf("historial de Mi Bolsa requiere un único material nominal: %d", coincidencias)
	}
}

func TestSeleccionReincorporacionTitularIncluyeSusTresAudiencias(t *testing.T) {
	apagada := descriptoresMaterialSeleccionadosCTDesarrollo(seleccionMaterialCTDesarrollo{})
	encendida := descriptoresMaterialSeleccionadosCTDesarrollo(seleccionMaterialCTDesarrollo{reincorporacionTitular: true})
	escritura := descriptorMaterialReincorporacionTitularDesarrollo()
	lectura := descriptorMaterialLecturaReincorporacionTitularDesarrollo()
	bolsa := descriptorMaterialConsultaReincorporacionTitularBolsaDesarrollo()
	if escritura.Audiencia == lectura.Audiencia {
		t.Fatal("escritura y lectura comparten audiencia")
	}
	if len(encendida) != len(apagada)+3 {
		t.Fatalf("el selector debe añadir tres audiencias: apagada=%d encendida=%d", len(apagada), len(encendida))
	}
	for _, d := range apagada {
		if d.Audiencia == escritura.Audiencia || d.Audiencia == lectura.Audiencia || d.Audiencia == bolsa.Audiencia {
			t.Fatalf("audiencia CT130 publicada con selector apagado: %s", d.Audiencia)
		}
	}
	for _, esperada := range []descriptorMaterialConsumidorV3Desarrollo{escritura, lectura, bolsa} {
		coincidencias := 0
		for _, d := range encendida {
			if reflect.DeepEqual(d, esperada) {
				coincidencias++
			}
		}
		if coincidencias != 1 {
			t.Fatalf("descriptor de reincorporación %s publicado %d veces", esperada.Audiencia, coincidencias)
		}
	}
	restantes := make([]descriptorMaterialConsumidorV3Desarrollo, 0, len(apagada))
	for _, d := range encendida {
		if d.Audiencia != escritura.Audiencia && d.Audiencia != lectura.Audiencia && d.Audiencia != bolsa.Audiencia {
			restantes = append(restantes, d)
		}
	}
	if !reflect.DeepEqual(restantes, apagada) {
		t.Fatal("el selector de reincorporación cambió otros descriptores")
	}
}

func TestSeleccionReincorporacionPublicaUnaAudienciaDeLectura(t *testing.T) {
	audiencia := descriptorMaterialConsultaReincorporacionTitularBolsaDesarrollo().Audiencia
	contar := func(s seleccionMaterialCTDesarrollo) int {
		n := 0
		for _, d := range descriptoresMaterialSeleccionadosCTDesarrollo(s) {
			if d.Audiencia == audiencia {
				n++
			}
		}
		return n
	}
	if contar(seleccionMaterialCTDesarrollo{}) != 0 || contar(seleccionMaterialCTDesarrollo{reincorporacionTitular: true}) != 1 {
		t.Fatal("audiencia B55 no depende exactamente del selector CT130")
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
	for _, d := range append(append(descriptoresMaterialSeguimientoCeseDesarrollo(), descriptorMaterialFirmaDocumentoCTDesarrollo(),
		descriptorMaterialConfirmacionGINPIXDesarrollo()), descriptoresMaterialCancelacionCTDesarrollo()...) {
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

func TestValorSelectorPlantillasDocumentalEnRaizPublica(t *testing.T) {
	t.Setenv(envCTPlantillasDocumentalEnabled, "si")
	if err := validarValorSelectoresDespliegueBolsaCT(config.Config{}); !errors.Is(err, ErrActivacionDesarrolloInvalida) {
		t.Fatalf("selector documental ilegible admitido: %v", err)
	}
	t.Setenv(envCTPlantillasDocumentalEnabled, "true")
	if err := validarValorSelectoresDespliegueBolsaCT(config.Config{}); err != nil {
		t.Fatalf("valor legible rechazado fuera de la raiz CT: %v", err)
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
	completo := estadoMigracionesPortalCandidato{ad384: true, ad386: true, bolsa29: true, bolsa30: true, bolsa40: true, bolsa77: true}
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
		{func(e *estadoMigracionesPortalCandidato) { e.bolsa77 = false }, ErrPortalCandidatoFaltaBolsa77},
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
