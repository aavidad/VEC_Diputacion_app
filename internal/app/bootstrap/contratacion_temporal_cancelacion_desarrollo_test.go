package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

const rutaMotivosCancelacionEjemploPrueba = "../../../data/demo/reglas/ct_motivos_cancelacion.demo.json"

// Fases, motivos y canales salen de los catálogos de ejemplo: cambiar una
// fase admitida o quién puede usar un motivo es editar el catálogo.
func TestFuenteReglasCancelacionLeeLosCatalogosDeEjemplo(t *testing.T) {
	consultaReglas, err := fichero.NuevaConsultaCatalogos(rutaReglasCTEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{Consulta: consultaReglas, Metadatos: consultaReglas,
		CatalogoID: reglas.CatalogoContratacionTemporal, ModuloID: reglas.ModuloContratacionTemporal, Reloj: relojPresentacionReglasEjemplo})
	if err != nil {
		t.Fatal(err)
	}
	motivos, err := fichero.NuevaConsultaCatalogos(rutaMotivosCancelacionEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	ahora := relojPresentacionReglasEjemplo.ahora
	regla, politica, err := fuenteReglasCancelacionDesarrollo{reglas: resolutor, motivos: motivos}.ReglaCancelacion(context.Background(), ahora)
	if err != nil || !regla.Valida() || !politica.ValidaEn(ahora) || politica.DefinicionRef != "motivos_cancelacion_contratacion_temporal" {
		t.Fatalf("regla: %+v %+v %v", regla, politica, err)
	}
	if len(regla.Fases) != 3 || regla.Fases[0] != "solicitud" || regla.Fases[1] != "asignacion_unidad" || regla.Fases[2] != "informe_juridico" {
		t.Fatalf("fases: %v", regla.Fases)
	}
	var centro, rrhh int
	for _, m := range regla.Motivos {
		if m.AdmiteCanal(domain.CanalCancelacionCentro) {
			centro++
		}
		if m.AdmiteCanal(domain.CanalCancelacionRRHH) {
			rrhh++
		}
	}
	if len(regla.Motivos) != 6 || centro != 4 || rrhh != 5 {
		t.Fatalf("motivos: %d centro=%d rrhh=%d", len(regla.Motivos), centro, rrhh)
	}
	for _, f := range regla.Fases {
		if f == domain.FaseFiscalizacion || f == domain.FaseNombramiento {
			t.Fatalf("la regla de ejemplo admite una fase tras la fiscalización: %s", f)
		}
	}
}

func TestCancelacionSoloSeComponeConSelectorYSusCatalogos(t *testing.T) {
	cfg := config.Config{ReglasEjemplo: config.ConfiguracionReglasEjemplo{CTSourcePath: "ct.json"}}
	if cancelacionCTSolicitada(cfg) {
		t.Fatal("sin pedirla no se compone")
	}
	cfg.CTCancelacionEnabled = "true"
	if cancelacionCTSolicitada(cfg) {
		t.Fatal("fuera de la doble llave de desarrollo no se compone")
	}
	cfg.ExecutionProfile, cfg.AuthMode, cfg.DevelopmentGuard = config.ExecutionProfileDevelopment, config.AuthModeDevelopment, config.DevelopmentGuardAcknowledgement
	if err := validarSelectoresDespliegueBolsaCT(cfg); !errors.Is(err, config.ErrConfiguracionCTCancelacionActivacion) ||
		!strings.Contains(err.Error(), config.EnvCTMotivosCancelacionSourcePath) {
		t.Fatalf("pedida sin motivos el arranque debe fallar nombrando la variable: %v", err)
	}
	cfg.ReglasEjemplo.MotivosCancelacionSourcePath = "motivos_cancelacion.json"
	if !cancelacionCTSolicitada(cfg) {
		t.Fatal("pedida, con doble llave y con sus catálogos se compone")
	}
	if err := validarValorSelectoresDespliegueBolsaCT(config.Config{CTCancelacionEnabled: "si"}); !errors.Is(err, config.ErrConfiguracionCTCancelacionSelector) {
		t.Fatalf("un valor mal escrito lo rechazan todas las raíces: %v", err)
	}
	rutas, err := nuevasRutasCancelacionCTDesarrollo(&DependenciasCT{cfg: config.Config{}}, nil)
	if err != nil || rutas != nil {
		t.Fatalf("sin selector la conducta es la de hoy: %v %v", rutas, err)
	}
	if _, err := nuevasRutasCancelacionCTDesarrollo(&DependenciasCT{cfg: cfg}, nil); err == nil {
		t.Fatal("pedida sin dependencias debe detener el arranque")
	}
	for _, o := range operacionesCancelacionCTDesarrollo() {
		if !rutaCancelacionCTDesarrollo(o.ruta) || motivoCancelacionCTDesarrollo(o.ruta).Validar() != nil {
			t.Fatalf("operación mal declarada: %+v", o)
		}
	}
	if err := comprobarMigracionesCancelacionCTDesarrollo(context.Background(), nil); !errors.Is(err, ErrCancelacionCTMigracionesNoDisponibles) {
		t.Fatalf("sin pool la comprobación falla cerrada: %v", err)
	}
}

// La instantánea solo admite el recurso exacto: RRHH, en curso, con fase
// válida y sin ámbitos añadidos.
func TestAmbitosCancelacionExigenElRecursoExacto(t *testing.T) {
	s := &soporteAltaContratacionTemporalDesarrollo{}
	recurso := func() vecdomain.RecursoAutorizable {
		return vecdomain.RecursoAutorizable{Referencia: "expediente:ct:prueba", ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoCancelacion,
			Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo, "expediente_ref": "expediente:ct:prueba",
				"fase_previa": "asignacion_unidad", "estado_previo": "en_curso"},
			Atributos: map[string]string{"canal": "rrhh"}}
	}
	datos := func(r vecdomain.RecursoAutorizable) vecdomain.DatosSolicitudAutorizacionLigadaV3 {
		return vecdomain.DatosSolicitudAutorizacionLigadaV3{Accion: string(domain.AccionCancelarExpediente), Finalidad: ports.FinalidadCancelarExpediente,
			ReferenciaMotivo: motivoCancelacionCTDesarrollo(httpinterno.RutaCancelacionesExpediente), Recurso: r}
	}
	if ambitos, ok := s.ambitosCancelacionCT(httpinterno.RutaCancelacionesExpediente, datos(recurso())); !ok || len(ambitos) != 4 {
		t.Fatalf("recurso exacto: %v %v", ambitos, ok)
	}
	for nombre, alterar := range map[string]func(*vecdomain.RecursoAutorizable){
		"estado":        func(r *vecdomain.RecursoAutorizable) { r.Ambitos["estado_previo"] = "cancelado" },
		"canal centro":  func(r *vecdomain.RecursoAutorizable) { r.Atributos["canal"] = "centro" },
		"ámbito extra":  func(r *vecdomain.RecursoAutorizable) { r.Ambitos["centro_ref"] = "centro:x" },
		"organización":  func(r *vecdomain.RecursoAutorizable) { r.Ambitos["organizacion_ref"] = "organizacion:otra" },
		"expediente":    func(r *vecdomain.RecursoAutorizable) { r.Ambitos["expediente_ref"] = "expediente:ct:otro" },
		"tipo recurso":  func(r *vecdomain.RecursoAutorizable) { r.Tipo = ports.TipoRecursoCese },
		"fase inválida": func(r *vecdomain.RecursoAutorizable) { r.Ambitos["fase_previa"] = "Fase X" },
	} {
		r := recurso()
		alterar(&r)
		if _, ok := s.ambitosCancelacionCT(httpinterno.RutaCancelacionesExpediente, datos(r)); ok {
			t.Fatalf("%s: se esperaba rechazo", nombre)
		}
	}
}
