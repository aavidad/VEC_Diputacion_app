package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	principalAutorizacionPrueba               = "per_persona_autorizacion_prueba_0001"
	principalAjenoAutorizacionPrueba          = "per_persona_ajena_autorizacion_0001"
	perfilBolsaAutorizacionPrueba             = "prf_perfil_bolsa_autorizacion_prueba_0001"
	perfilConsultaAutorizacionPrueba          = "prf_perfil_consulta_autorizacion_prueba_0001"
	perfilModificacionAutorizacionPrueba      = "prf_perfil_modificacion_autorizacion_prueba_0001"
	claveMotivoAutorizacionV2Prueba           = "motivo_33333333333333333333333333333333"
	referenciaCorrelacionAutorizacionV2Prueba = "correlacion_33333333333333333333333333333333"
)

type validadorMotivoAutorizacionMemoriaPrueba struct {
	referencia domain.ReferenciaEntradaCatalogo
}

func (v validadorMotivoAutorizacionMemoriaPrueba) ValidarReferenciaMotivoAutorizacionV2(
	_ context.Context,
	referencia domain.ReferenciaEntradaCatalogo,
	_ time.Time,
) error {
	if referencia != v.referencia {
		return domain.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

func TestAutorizadorNoSumaPermisosDePerfiles(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	almacen := NuevoAlmacenAutorizacionMemoria()
	consulta := versionRolAutorizacionPrueba("consulta_bolsa", "bolsa.expediente.leer", domain.AuthAssuranceSubstantial, []string{"gestion_bolsa"})
	modificacion := versionRolAutorizacionPrueba("modificacion_bolsa", "bolsa.expediente.modificar", domain.AuthAssuranceSubstantial, []string{"gestion_bolsa"})
	sembrarVersionAutorizacion(t, almacen, consulta)
	sembrarVersionAutorizacion(t, almacen, modificacion)
	sembrarAsignacionAutorizacion(t, almacen, asignacionAutorizacionPrueba(perfilConsultaAutorizacionPrueba, "asig-consulta", consulta.Referencia(), "seleccion"))
	sembrarAsignacionAutorizacion(t, almacen, asignacionAutorizacionPrueba(perfilModificacionAutorizacionPrueba, "asig-modificacion", modificacion.Referencia(), "seleccion"))
	autorizador := nuevoAutorizadorPrueba(t, almacen, almacen, ahora)

	solicitud := solicitudAutorizacionPerfilPrueba(perfilConsultaAutorizacionPrueba, domain.AuthAssuranceHigh)
	solicitud.Accion = "bolsa.expediente.modificar"
	// Estos datos proceden del principal y se ignoran como autoridad.
	solicitud.Principal.Roles = []string{"administrador"}
	solicitud.Principal.Permissions = []string{"bolsa.expediente.modificar"}
	decision, err := autorizador.Exigir(context.Background(), solicitud)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Concedida {
		t.Fatalf("el otro perfil no debe sumar permisos: decision=%+v err=%v", decision, err)
	}
	if decision.Codigo != "accion_no_concedida" {
		t.Fatalf("codigo inesperado: %q", decision.Codigo)
	}

	solicitud = solicitudAutorizacionPerfilPrueba(perfilModificacionAutorizacionPrueba, domain.AuthAssuranceHigh)
	solicitud.Accion = "bolsa.expediente.modificar"
	decision, err = autorizador.Exigir(context.Background(), solicitud)
	if err != nil || !decision.Concedida {
		t.Fatalf("el perfil seleccionado si debe conceder: decision=%+v err=%v", decision, err)
	}
}

func TestAlmacenAutorizacionMemoriaSeparaRegistrosV1YV2(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	servicioV1, almacen := autorizadorBasePrueba(t, ahora)
	solicitud := solicitudAutorizacionPrueba()

	decisionV1, err := servicioV1.Exigir(context.Background(), solicitud)
	if err != nil || decisionV1.TieneSolicitudLigadaV2() {
		t.Fatalf("el servicio V1 no produjo una decision historica valida: decision=%+v err=%v", decisionV1, err)
	}
	referenciaMotivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
		CatalogoHuellaSHA256: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		EntradaClave:         claveMotivoAutorizacionV2Prueba,
	}
	servicioV2, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV2(
		almacen,
		almacen,
		almacen,
		validadorMotivoAutorizacionMemoriaPrueba{referencia: referenciaMotivo},
		relojAutorizacionPrueba{ahora: ahora},
		generadorReferenciaAutorizacionPrueba("decision:registro-memoria:v2"),
		vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatalf("crear servicio V2: %v", err)
	}
	solicitudV2, err := domain.NuevaSolicitudAutorizacionLigadaV2(
		domain.DatosSolicitudAutorizacionLigadaV2{
			ContextoActor: solicitud.ContextoActor, VinculoAutenticacionActor: solicitud.VinculoAutenticacionActor,
			ReferenciaMotivo: referenciaMotivo, Accion: solicitud.Accion,
			Recurso: solicitud.Recurso, Finalidad: solicitud.Finalidad,
			CorrelacionRef: referenciaCorrelacionAutorizacionV2Prueba,
		},
	)
	if err != nil {
		t.Fatalf("crear solicitud V2: %v", err)
	}
	decisionV2, err := servicioV2.ExigirSolicitudLigadaV2(context.Background(), solicitudV2)
	if err != nil || !decisionV2.TieneSolicitudLigadaV2() {
		t.Fatalf("el servicio V2 no produjo una decision ligada: decision=%+v err=%v", decisionV2, err)
	}

	if err := almacen.RegistrarDecisionSiInstantaneaVigente(context.Background(), decisionV2); !errors.Is(err, domain.ErrDecisionAutorizacionInvalida) {
		t.Fatalf("el registro V1 acepto una decision V2: %v", err)
	}
	if _, err := ports.NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(decisionV1, referenciaMotivo); !errors.Is(err, ports.ErrOrdenRegistroAutorizacionSolicitudLigadaV2Invalida) {
		t.Fatalf("se pudo construir una orden V2 desde decision V1: %v", err)
	}
	if len(almacen.decisiones) != 2 {
		t.Fatalf("numero inesperado de decisiones registradas: %d", len(almacen.decisiones))
	}
	if almacen.referenciasMotivoV2[decisionV2.DecisionRef] != referenciaMotivo {
		t.Fatal("el registro V2 no conservo la referencia exacta de motivo")
	}
}

func TestAutorizadorRestringeAmbitoDeUnidad(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	autorizador, _ := autorizadorBasePrueba(t, ahora)
	solicitud := solicitudAutorizacionPrueba()
	solicitud.Recurso.Ambitos["unidad"] = "nominas"

	decision, err := autorizador.Exigir(context.Background(), solicitud)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Concedida {
		t.Fatalf("una unidad ajena debe denegarse: decision=%+v err=%v", decision, err)
	}
	if decision.Codigo != "ambito_no_autorizado" {
		t.Fatalf("codigo inesperado: %q", decision.Codigo)
	}
}

func TestAutorizadorNoCruzaConcesionEntreModulos(t *testing.T) {
	autorizador, _ := autorizadorBasePrueba(t, instanteAutorizacionPrueba())
	solicitud := solicitudAutorizacionPrueba()
	solicitud.Recurso.ModuloID = "cronos"

	decision, err := autorizador.Exigir(context.Background(), solicitud)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Concedida {
		t.Fatalf("una concesion de bolsa cruzo a cronos: decision=%+v err=%v", decision, err)
	}
	if decision.Codigo != "accion_no_concedida" {
		t.Fatalf("codigo inesperado: %q", decision.Codigo)
	}
}

func TestAutorizadorDeniegaAsignacionCaducadaORevocada(t *testing.T) {
	ahora := instanteAutorizacionPrueba()

	t.Run("caducada", func(t *testing.T) {
		almacen := NuevoAlmacenAutorizacionMemoria()
		rol := versionRolAutorizacionPrueba("tecnico_bolsa", "bolsa.expediente.leer", domain.AuthAssuranceSubstantial, []string{"gestion_bolsa"})
		sembrarVersionAutorizacion(t, almacen, rol)
		asignacion := asignacionAutorizacionPrueba(perfilBolsaAutorizacionPrueba, "asig-bolsa", rol.Referencia(), "seleccion")
		asignacion.VigenteDesde = ahora.Add(-2 * time.Hour)
		asignacion.VigenteHasta = ahora.Add(-time.Hour)
		sembrarAsignacionAutorizacion(t, almacen, asignacion)
		autorizador := nuevoAutorizadorPrueba(t, almacen, almacen, ahora)

		decision, err := autorizador.Exigir(context.Background(), solicitudAutorizacionPrueba())
		if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Codigo != "perfil_no_vigente" {
			t.Fatalf("asignacion caducada aceptada: decision=%+v err=%v", decision, err)
		}
	})

	t.Run("revocada_por_version_posterior", func(t *testing.T) {
		autorizador, almacen := autorizadorBasePrueba(t, ahora)
		revocada := asignacionAutorizacionPrueba(perfilBolsaAutorizacionPrueba, "asig-bolsa", versionRolAutorizacionPrueba("tecnico_bolsa", "bolsa.expediente.leer", domain.AuthAssuranceSubstantial, []string{"gestion_bolsa"}).Referencia(), "seleccion")
		revocada.Version = 2
		revocada.Estado = domain.EstadoAsignacionPerfilRevocada
		revocada.RevocadaPor = "responsable-seguridad"
		revocada.RevocadaEn = ahora.Add(-time.Minute)
		revocada.RevocacionRef = "revocacion:2026-001"
		sembrarAsignacionAutorizacion(t, almacen, revocada)

		decision, err := autorizador.Exigir(context.Background(), solicitudAutorizacionPrueba())
		if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Codigo != "perfil_no_vigente" {
			t.Fatalf("asignacion revocada aceptada: decision=%+v err=%v", decision, err)
		}
	})
}

func TestGarantiaDePoliticaNoRebajaLaDelRol(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	almacen := NuevoAlmacenAutorizacionMemoria()
	rol := versionRolAutorizacionPrueba("responsable_bolsa", "bolsa.expediente.leer", domain.AuthAssuranceHigh, []string{"gestion_bolsa"})
	sembrarVersionAutorizacion(t, almacen, rol)
	sembrarAsignacionAutorizacion(t, almacen, asignacionAutorizacionPrueba(perfilBolsaAutorizacionPrueba, "asig-bolsa", rol.Referencia(), "seleccion"))
	politica := politicaAutorizacionPrueba("garantia_baja", domain.EfectoPoliticaRestringir)
	politica.GarantiaMinima = domain.AuthAssuranceLow
	sembrarPoliticaAutorizacion(t, almacen, politica)
	autorizador := nuevoAutorizadorPrueba(t, almacen, almacen, ahora)
	solicitud := solicitudAutorizacionPerfilPrueba(perfilBolsaAutorizacionPrueba, domain.AuthAssuranceSubstantial)

	decision, err := autorizador.Exigir(context.Background(), solicitud)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Codigo != "garantia_insuficiente" {
		t.Fatalf("la politica rebajo la garantia del rol: decision=%+v err=%v", decision, err)
	}
	if decision.GarantiaMinima != domain.AuthAssuranceHigh {
		t.Fatalf("garantia efectiva inesperada: %q", decision.GarantiaMinima)
	}
}

func TestPoliticaRestringeFinalidadSinConceder(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	almacen := NuevoAlmacenAutorizacionMemoria()
	rol := versionRolAutorizacionPrueba(
		"tecnico_bolsa",
		"bolsa.expediente.leer",
		domain.AuthAssuranceSubstantial,
		[]string{"gestion_bolsa", "analitica_comercial"},
	)
	sembrarVersionAutorizacion(t, almacen, rol)
	sembrarAsignacionAutorizacion(t, almacen, asignacionAutorizacionPrueba(perfilBolsaAutorizacionPrueba, "asig-bolsa", rol.Referencia(), "seleccion"))
	politica := politicaAutorizacionPrueba("finalidad_bolsa", domain.EfectoPoliticaRestringir)
	politica.FinalidadesPermitidas = []string{"gestion_bolsa"}
	politica.RestringeCampos = true
	politica.CamposPermitidos = []string{"estado"}
	politica.Obligaciones = []string{"registrar_consulta_reforzada"}
	sembrarPoliticaAutorizacion(t, almacen, politica)
	autorizador := nuevoAutorizadorPrueba(t, almacen, almacen, ahora)

	solicitud := solicitudAutorizacionPrueba()
	solicitud.Finalidad = "analitica_comercial"
	decision, err := autorizador.Exigir(context.Background(), solicitud)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Codigo != "restriccion_abac_incumplida" {
		t.Fatalf("finalidad no autorizada aceptada: decision=%+v err=%v", decision, err)
	}

	solicitud.Finalidad = "gestion_bolsa"
	decision, err = autorizador.Exigir(context.Background(), solicitud)
	if err != nil || !decision.Concedida {
		t.Fatalf("finalidad administrativa denegada: decision=%+v err=%v", decision, err)
	}
	if len(decision.CamposPermitidos) != 1 || decision.CamposPermitidos[0] != "estado" {
		t.Fatalf("campos no restringidos: %v", decision.CamposPermitidos)
	}
	if len(decision.PoliticasRefs) != 1 || decision.PoliticasHuellasSHA256[decision.PoliticasRefs[0]] == "" {
		t.Fatalf("falta evidencia de politica: %+v", decision)
	}
}

func TestPoliticaDeDenegacionVence(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	autorizador, almacen := autorizadorBasePrueba(t, ahora)
	restriccion := politicaAutorizacionPrueba("a_restriccion", domain.EfectoPoliticaRestringir)
	restriccion.Restricciones = []domain.RestriccionAtributoRecurso{{Clave: "clasificacion", ValoresPermitidos: []string{"interno"}}}
	denegacion := politicaAutorizacionPrueba("z_denegacion", domain.EfectoPoliticaDenegar)
	sembrarPoliticaAutorizacion(t, almacen, restriccion)
	sembrarPoliticaAutorizacion(t, almacen, denegacion)

	solicitud := solicitudAutorizacionPrueba()
	solicitud.Recurso.Atributos["clasificacion"] = "interno"
	decision, err := autorizador.Exigir(context.Background(), solicitud)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Concedida || decision.Codigo != "denegada_por_politica" {
		t.Fatalf("la denegacion no vencio: decision=%+v err=%v", decision, err)
	}
	if _, err := almacen.ObtenerDecision(context.Background(), decision.DecisionRef); !errors.Is(err, ports.ErrDecisionAutorizacionNoEncontrada) {
		t.Fatalf("una denegacion aparecio como capacidad ejecutable: %v", err)
	}
	registrada, err := almacen.ObtenerDenegacion(context.Background(), decision.DecisionRef)
	if err != nil || registrada.Concedida || registrada.Codigo != decision.Codigo {
		t.Fatalf("traza negativa no separada: decision=%+v err=%v", registrada, err)
	}
}

func TestFalloDeRegistroConvierteConcesionEnDenegacion(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	_, almacen := autorizadorBasePrueba(t, ahora)
	registro := registroDecisionesConFallo{err: errors.New("almacenamiento no disponible")}
	autorizador := nuevoAutorizadorPrueba(t, almacen, registro, ahora)

	decision, err := autorizador.Exigir(context.Background(), solicitudAutorizacionPrueba())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || !errors.Is(err, ports.ErrRegistroDecisionNoDisponible) {
		t.Fatalf("error de registro no cerro el acceso: decision=%+v err=%v", decision, err)
	}
	if decision.Concedida || decision.Codigo != "registro_no_disponible" {
		t.Fatalf("decision insegura tras fallo de registro: %+v", decision)
	}
}

func TestRegistrosDeConcesionYDenegacionNoIntercambianSemantica(t *testing.T) {
	almacen := NuevoAlmacenAutorizacionMemoria()
	if err := almacen.RegistrarDecisionSiInstantaneaVigente(
		context.Background(), domain.DecisionAutorizacion{Concedida: false, Codigo: "accion_no_concedida"},
	); !errors.Is(err, ports.ErrRegistroDecisionNoDisponible) {
		t.Fatalf("el registro de concesiones acepto una denegacion: %v", err)
	}
	if err := almacen.RegistrarDenegacionAutorizacion(
		context.Background(), domain.DecisionAutorizacion{Concedida: true, Codigo: "concedida"},
	); !errors.Is(err, ports.ErrRegistroDenegacionNoDisponible) {
		t.Fatalf("el registro de denegaciones acepto una concesion: %v", err)
	}
}

func TestDecisionRefLaGeneraElNucleoYQuedaRegistrada(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	autorizador, almacen := autorizadorBasePrueba(t, ahora)
	solicitud := solicitudAutorizacionPrueba()
	solicitud.CorrelacionRef = "decision:valor-controlado-por-cliente"

	decision, err := autorizador.Exigir(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("autorizar: %v", err)
	}
	if !strings.HasPrefix(decision.DecisionRef, "decision:") || decision.DecisionRef == solicitud.CorrelacionRef {
		t.Fatalf("referencia no interna: %q", decision.DecisionRef)
	}
	registrada, err := almacen.ObtenerDecision(context.Background(), decision.DecisionRef)
	if err != nil {
		t.Fatalf("obtener decision registrada: %v", err)
	}
	if !registrada.Concedida || registrada.AsignacionHuellaSHA256 == "" || registrada.VersionRolHuellaSHA256 == "" {
		t.Fatalf("evidencia incompleta: %+v", registrada)
	}
	if registrada.ValidaHasta.Sub(registrada.EmitidaEn) != 90*time.Second {
		t.Fatalf("vigencia inesperada: %v", registrada.ValidaHasta.Sub(registrada.EmitidaEn))
	}
}

func TestDecisionConservaCatalogoCompletoSeparadoDelSubconjuntoAplicado(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	autorizador, almacen := autorizadorBasePrueba(t, ahora)
	aplicada := politicaAutorizacionPrueba("minimizacion", domain.EfectoPoliticaRestringir)
	noAplicada := politicaAutorizacionPrueba("solo_cronos", domain.EfectoPoliticaRestringir)
	noAplicada.Modulos = []string{"cronos"}
	sembrarPoliticaAutorizacion(t, almacen, aplicada)
	sembrarPoliticaAutorizacion(t, almacen, noAplicada)

	decision, err := autorizador.Exigir(context.Background(), solicitudAutorizacionPrueba())
	if err != nil {
		t.Fatalf("autorizar: %v", err)
	}
	if len(decision.PoliticasEvaluadasRefs) != 2 || len(decision.PoliticasEvaluadasHuellasSHA256) != 2 {
		t.Fatalf("catalogo evaluado incompleto: %+v", decision)
	}
	if len(decision.PoliticasRefs) != 1 || decision.PoliticasRefs[0] != aplicada.Referencia() {
		t.Fatalf("subconjunto aplicado incorrecto: %v", decision.PoliticasRefs)
	}
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion([]domain.PoliticaRestrictiva{aplicada, noAplicada})
	if err != nil {
		t.Fatalf("huella esperada: %v", err)
	}
	if decision.CatalogoPoliticasHuellaSHA256 != huella || decision.RevisionCatalogoPoliticas != 3 {
		t.Fatalf("control de catalogo incorrecto: revision=%d huella=%q", decision.RevisionCatalogoPoliticas, decision.CatalogoPoliticasHuellaSHA256)
	}
}

func TestAlmacenAutorizacionCopiaProfundamenteInstantaneasYDecisiones(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	almacen := NuevoAlmacenAutorizacionMemoria()
	rol := versionRolAutorizacionPrueba("tecnico_bolsa", "bolsa.expediente.leer", domain.AuthAssuranceSubstantial, []string{"gestion_bolsa"})
	asignacion := asignacionAutorizacionPrueba(perfilBolsaAutorizacionPrueba, "asig-bolsa", rol.Referencia(), "seleccion")
	politica := politicaAutorizacionPrueba("clasificacion", domain.EfectoPoliticaRestringir)
	politica.Restricciones = []domain.RestriccionAtributoRecurso{{Clave: "clasificacion", ValoresPermitidos: []string{"interno"}}}
	sembrarVersionAutorizacion(t, almacen, rol)
	sembrarAsignacionAutorizacion(t, almacen, asignacion)
	sembrarPoliticaAutorizacion(t, almacen, politica)

	// Los argumentos de siembra no conservan alias mutables con el almacen.
	rol.Concesiones[0].Finalidades[0] = "alterada"
	asignacion.Ambitos[0].Valores[0] = "nominas"
	politica.Restricciones[0].ValoresPermitidos[0] = "publico"
	primera, err := almacen.ObtenerInstantaneaAutorizacion(context.Background(), principalAutorizacionPrueba, perfilBolsaAutorizacionPrueba)
	if err != nil {
		t.Fatalf("obtener primera instantanea: %v", err)
	}
	if primera.VersionRol.Concesiones[0].Finalidades[0] != "gestion_bolsa" ||
		primera.AsignacionPerfil.Ambitos[0].Valores[0] != "seleccion" ||
		primera.Politicas[0].Restricciones[0].ValoresPermitidos[0] != "interno" {
		t.Fatalf("la siembra conservo alias mutables: %+v", primera)
	}

	// Tampoco la lectura puede modificar el estado conservado.
	primera.VersionRol.Concesiones[0].Finalidades[0] = "otra"
	primera.AsignacionPerfil.Ambitos[0].Valores[0] = "otra"
	primera.Politicas[0].Restricciones[0].ValoresPermitidos[0] = "otro"
	segunda, err := almacen.ObtenerInstantaneaAutorizacion(context.Background(), principalAutorizacionPrueba, perfilBolsaAutorizacionPrueba)
	if err != nil || segunda.Validar() != nil {
		t.Fatalf("obtener segunda instantanea: instantanea=%+v err=%v", segunda, err)
	}
	if segunda.VersionRol.Concesiones[0].Finalidades[0] != "gestion_bolsa" ||
		segunda.AsignacionPerfil.Ambitos[0].Valores[0] != "seleccion" ||
		segunda.Politicas[0].Restricciones[0].ValoresPermitidos[0] != "interno" {
		t.Fatalf("la lectura modifico el almacen: %+v", segunda)
	}

	autorizador := nuevoAutorizadorPrueba(t, almacen, almacen, ahora)
	solicitud := solicitudAutorizacionPrueba()
	solicitud.Recurso.Atributos["clasificacion"] = "interno"
	decision, err := autorizador.Exigir(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("autorizar: %v", err)
	}
	if len(decision.PoliticasRefs) == 0 || len(decision.PoliticasEvaluadasRefs) == 0 {
		t.Fatalf("la prueba requiere mapas no vacios: %+v", decision)
	}
	refAplicada := decision.PoliticasRefs[0]
	refEvaluada := decision.PoliticasEvaluadasRefs[0]
	huellaAplicada := decision.PoliticasHuellasSHA256[refAplicada]
	huellaEvaluada := decision.PoliticasEvaluadasHuellasSHA256[refEvaluada]
	decision.PoliticasRefs[0] = "politica:alterada:v1"
	decision.PoliticasEvaluadasRefs[0] = "politica:alterada:v1"
	decision.PoliticasHuellasSHA256[refAplicada] = strings.Repeat("0", 64)
	decision.PoliticasEvaluadasHuellasSHA256[refEvaluada] = strings.Repeat("0", 64)

	registrada, err := almacen.ObtenerDecision(context.Background(), decision.DecisionRef)
	if err != nil {
		t.Fatalf("obtener decision: %v", err)
	}
	if registrada.PoliticasRefs[0] != refAplicada || registrada.PoliticasEvaluadasRefs[0] != refEvaluada ||
		registrada.PoliticasHuellasSHA256[refAplicada] != huellaAplicada ||
		registrada.PoliticasEvaluadasHuellasSHA256[refEvaluada] != huellaEvaluada {
		t.Fatalf("la decision registrada compartio mapas o listas: %+v", registrada)
	}
	registrada.PoliticasHuellasSHA256[refAplicada] = strings.Repeat("1", 64)
	registrada.PoliticasEvaluadasHuellasSHA256[refEvaluada] = strings.Repeat("1", 64)
	otraLectura, err := almacen.ObtenerDecision(context.Background(), decision.DecisionRef)
	if err != nil || otraLectura.PoliticasHuellasSHA256[refAplicada] != huellaAplicada ||
		otraLectura.PoliticasEvaluadasHuellasSHA256[refEvaluada] != huellaEvaluada {
		t.Fatalf("la decision leida compartio mapas: decision=%+v err=%v", otraLectura, err)
	}
}

func TestFuenteAutorizacionNoRevelaPerfilDeOtroSujeto(t *testing.T) {
	_, almacen := autorizadorBasePrueba(t, instanteAutorizacionPrueba())
	ctx := context.Background()
	instantaneaAjena, errAjeno := almacen.ObtenerInstantaneaAutorizacion(ctx, principalAjenoAutorizacionPrueba, perfilBolsaAutorizacionPrueba)
	instantaneaAusente, errAusente := almacen.ObtenerInstantaneaAutorizacion(ctx, principalAjenoAutorizacionPrueba, "prf_perfil_inexistente_autorizacion_0001")
	if !errors.Is(errAjeno, ports.ErrAsignacionPerfilNoEncontrada) || !errors.Is(errAusente, ports.ErrAsignacionPerfilNoEncontrada) ||
		errAjeno.Error() != errAusente.Error() {
		t.Fatalf("el perfil ajeno se distinguio del ausente: ajeno=%v ausente=%v", errAjeno, errAusente)
	}
	if len(instantaneaAjena.Politicas) != 0 || len(instantaneaAusente.Politicas) != 0 ||
		instantaneaAjena.CatalogoPoliticasHuellaSHA256 != "" || instantaneaAusente.CatalogoPoliticasHuellaSHA256 != "" {
		t.Fatalf("la respuesta revelo catalogo o configuracion: ajena=%+v ausente=%+v", instantaneaAjena, instantaneaAusente)
	}
}

func TestAutorizadorNoNormalizaReferenciaInternaParaConceder(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	_, almacen := autorizadorBasePrueba(t, ahora)
	autorizador, err := vecapp.NuevoServicioAutorizacion(
		almacen,
		almacen,
		almacen,
		relojAutorizacionPrueba{ahora: ahora},
		generadorReferenciaAutorizacionPrueba(" decision:no-canonica "),
		vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatalf("crear autorizador: %v", err)
	}
	decision, err := autorizador.Exigir(context.Background(), solicitudAutorizacionPrueba())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Concedida {
		t.Fatalf("referencia no canonica aceptada: decision=%+v err=%v", decision, err)
	}
	if _, err := almacen.ObtenerDecision(context.Background(), "decision:no-canonica"); !errors.Is(err, ports.ErrDecisionAutorizacionNoEncontrada) {
		t.Fatalf("la referencia normalizada no debio persistirse: %v", err)
	}
}

func TestRepositorioAutorizacionNoNormalizaClavesDeConsulta(t *testing.T) {
	_, almacen := autorizadorBasePrueba(t, instanteAutorizacionPrueba())
	ctx := context.Background()

	if _, err := almacen.ObtenerInstantaneaAutorizacion(ctx, principalAutorizacionPrueba, " "+perfilBolsaAutorizacionPrueba); !errors.Is(err, ports.ErrAsignacionPerfilNoEncontrada) {
		t.Fatalf("el perfil no canonico encontro una asignacion: %v", err)
	}
	if _, err := almacen.ObtenerInstantaneaAutorizacion(ctx, principalAutorizacionPrueba+" ", perfilBolsaAutorizacionPrueba); !errors.Is(err, ports.ErrAsignacionPerfilNoEncontrada) {
		t.Fatalf("el principal no canonico encontro una asignacion: %v", err)
	}

	decision, err := nuevoAutorizadorPrueba(t, almacen, almacen, instanteAutorizacionPrueba()).Exigir(ctx, solicitudAutorizacionPrueba())
	if err != nil {
		t.Fatalf("crear decision de prueba: %v", err)
	}
	if _, err := almacen.ObtenerDecision(ctx, " "+decision.DecisionRef); !errors.Is(err, ports.ErrDecisionAutorizacionNoEncontrada) {
		t.Fatalf("la referencia de decision no canonica encontro evidencia: %v", err)
	}
}

func TestAutorizadorMemoriaRegistraDecisionesConcurrentes(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	autorizador, almacen := autorizadorBasePrueba(t, ahora)
	const total = 24
	referencias := make(chan string, total)
	errores := make(chan error, total)
	var grupo sync.WaitGroup
	for indice := 0; indice < total; indice++ {
		grupo.Add(1)
		go func(indice int) {
			defer grupo.Done()
			solicitud := solicitudAutorizacionPrueba()
			solicitud.CorrelacionRef = fmt.Sprintf("corr-concurrente-%d", indice)
			decision, err := autorizador.Exigir(context.Background(), solicitud)
			if err != nil {
				errores <- err
				return
			}
			referencias <- decision.DecisionRef
		}(indice)
	}
	grupo.Wait()
	close(referencias)
	close(errores)
	for err := range errores {
		t.Fatalf("autorizacion concurrente: %v", err)
	}
	vistas := make(map[string]struct{}, total)
	for referencia := range referencias {
		if _, repetida := vistas[referencia]; repetida {
			t.Fatalf("referencia repetida: %q", referencia)
		}
		vistas[referencia] = struct{}{}
		if _, err := almacen.ObtenerDecision(context.Background(), referencia); err != nil {
			t.Fatalf("decision concurrente no registrada: %v", err)
		}
	}
	if len(vistas) != total {
		t.Fatalf("decisiones registradas=%d, se esperaban %d", len(vistas), total)
	}
}

func TestCASImpideConcesionSiLaInstantaneaCambiaAntesDelRegistro(t *testing.T) {
	tipoCambio := []struct {
		nombre  string
		aplicar func(*testing.T, *AlmacenAutorizacionMemoria)
	}{
		{
			nombre: "publicacion de politica",
			aplicar: func(t *testing.T, almacen *AlmacenAutorizacionMemoria) {
				sembrarPoliticaAutorizacion(t, almacen, politicaAutorizacionPrueba("nueva_restriccion", domain.EfectoPoliticaRestringir))
			},
		},
		{
			nombre: "revocacion de asignacion",
			aplicar: func(t *testing.T, almacen *AlmacenAutorizacionMemoria) {
				revocada := asignacionAutorizacionPrueba(
					perfilBolsaAutorizacionPrueba, "asig-bolsa",
					versionRolAutorizacionPrueba("tecnico_bolsa", "bolsa.expediente.leer", domain.AuthAssuranceSubstantial, []string{"gestion_bolsa"}).Referencia(),
					"seleccion",
				)
				revocada.Version = 2
				revocada.Estado = domain.EstadoAsignacionPerfilRevocada
				revocada.RevocadaPor = "responsable-seguridad"
				revocada.RevocadaEn = instanteAutorizacionPrueba().Add(-time.Minute)
				revocada.RevocacionRef = "acto:revocacion-asignacion:1"
				sembrarAsignacionAutorizacion(t, almacen, revocada)
			},
		},
		{
			nombre: "retirada global de version de rol",
			aplicar: func(t *testing.T, almacen *AlmacenAutorizacionMemoria) {
				instantanea, err := almacen.ObtenerInstantaneaAutorizacion(
					context.Background(), principalAutorizacionPrueba, perfilBolsaAutorizacionPrueba,
				)
				if err != nil {
					t.Fatalf("obtener control de rol: %v", err)
				}
				control := instantanea.ControlVigenciaVersionRol
				control.Revision++
				control.Estado = domain.EstadoControlVigenciaVersionRolRetirada
				control.ActualizadoPor = "responsable-seguridad"
				control.ActualizadoEn = instanteAutorizacionPrueba().Add(-time.Minute)
				control.ActoRef = "acto:retirada-rol:1"
				control.MotivoCodigo = "incidente_seguridad"
				if err := almacen.SembrarControlVigenciaVersionRol(control); err != nil {
					t.Fatalf("retirar version de rol: %v", err)
				}
			},
		},
	}
	for _, cambio := range tipoCambio {
		t.Run(cambio.nombre, func(t *testing.T) {
			ahora := instanteAutorizacionPrueba()
			_, almacen := autorizadorBasePrueba(t, ahora)
			registro := &registroAutorizacionConBarrera{
				destino: almacen, iniciada: make(chan struct{}), continuar: make(chan struct{}),
			}
			autorizador := nuevoAutorizadorPrueba(t, almacen, registro, ahora)
			type resultadoCAS struct {
				decision domain.DecisionAutorizacion
				err      error
			}
			resultados := make(chan resultadoCAS, 1)
			go func() {
				decision, err := autorizador.Exigir(context.Background(), solicitudAutorizacionPrueba())
				resultados <- resultadoCAS{decision: decision, err: err}
			}()
			select {
			case <-registro.iniciada:
			case <-time.After(2 * time.Second):
				close(registro.continuar)
				t.Fatal("el registro CAS no alcanzo la barrera")
			}
			cambio.aplicar(t, almacen)
			close(registro.continuar)
			var obtenido resultadoCAS
			select {
			case obtenido = <-resultados:
			case <-time.After(2 * time.Second):
				t.Fatal("el registro CAS no termino tras liberar la barrera")
			}
			if !errors.Is(obtenido.err, domain.ErrAutorizacionDenegada) ||
				!errors.Is(obtenido.err, ports.ErrInstantaneaAutorizacionObsoleta) ||
				obtenido.decision.Concedida || obtenido.decision.Codigo != "instantanea_obsoleta" {
				t.Fatalf("el cambio concurrente termino en concesion: decision=%+v err=%v", obtenido.decision, obtenido.err)
			}
			if _, err := almacen.ObtenerDecision(context.Background(), obtenido.decision.DecisionRef); !errors.Is(err, ports.ErrDecisionAutorizacionNoEncontrada) {
				t.Fatalf("se inserto la concesion pese al CAS: %v", err)
			}
		})
	}
}

func TestRetiradaDeOtraVersionNoRevocaLaVersionAsignada(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	autorizador, almacen := autorizadorBasePrueba(t, ahora)
	retiradaV2 := versionRolAutorizacionPrueba(
		"tecnico_bolsa", "bolsa.expediente.leer", domain.AuthAssuranceSubstantial, []string{"gestion_bolsa"},
	)
	retiradaV2.Version = 2
	retiradaV2.Estado = domain.EstadoVersionRolRetirada
	retiradaV2.RetiradaPor = "responsable-seguridad"
	retiradaV2.RetiradaEn = ahora.Add(-time.Minute)
	retiradaV2.RetiradaRef = "acto:retirada-rol:v2"
	retiradaV2.MotivoRetiradaCodigo = "version_no_utilizable"
	sembrarVersionAutorizacion(t, almacen, retiradaV2)

	decision, err := autorizador.Exigir(context.Background(), solicitudAutorizacionPrueba())
	if err != nil || !decision.Concedida || decision.VersionRolRef != "rol:tecnico_bolsa:v1" {
		t.Fatalf("v2 retirada revoco implicitamente v1: decision=%+v err=%v", decision, err)
	}

	instantanea, err := almacen.ObtenerInstantaneaAutorizacion(context.Background(), principalAutorizacionPrueba, perfilBolsaAutorizacionPrueba)
	if err != nil {
		t.Fatalf("obtener instantanea v1: %v", err)
	}
	control := instantanea.ControlVigenciaVersionRol
	control.Revision++
	control.Estado = domain.EstadoControlVigenciaVersionRolRetirada
	control.ActualizadoPor = "responsable-seguridad"
	control.ActualizadoEn = ahora
	control.ActoRef = "acto:retirada-rol:v1"
	control.MotivoCodigo = "incidente_seguridad"
	if err := almacen.SembrarControlVigenciaVersionRol(control); err != nil {
		t.Fatalf("retirar v1 expresamente: %v", err)
	}
	decision, err = autorizador.Exigir(context.Background(), solicitudAutorizacionPrueba())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Concedida || decision.Codigo != "rol_retirado" {
		t.Fatalf("la retirada expresa de v1 no denego: decision=%+v err=%v", decision, err)
	}
}

type registroAutorizacionConBarrera struct {
	destino   ports.RegistroDecisionesAutorizacion
	iniciada  chan struct{}
	continuar chan struct{}
	unaVez    sync.Once
}

func (r *registroAutorizacionConBarrera) RegistrarDecisionSiInstantaneaVigente(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
) error {
	r.unaVez.Do(func() { close(r.iniciada) })
	select {
	case <-r.continuar:
	case <-ctx.Done():
		return ctx.Err()
	}
	return r.destino.RegistrarDecisionSiInstantaneaVigente(ctx, decision)
}

func (r *registroAutorizacionConBarrera) RegistrarDenegacionAutorizacion(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
) error {
	destino, ok := r.destino.(ports.RegistroDenegacionesAutorizacion)
	if !ok {
		return ports.ErrRegistroDenegacionNoDisponible
	}
	return destino.RegistrarDenegacionAutorizacion(ctx, decision)
}

type registroDecisionesConFallo struct {
	err error
}

func (r registroDecisionesConFallo) RegistrarDecisionSiInstantaneaVigente(context.Context, domain.DecisionAutorizacion) error {
	return r.err
}

func (r registroDecisionesConFallo) RegistrarDenegacionAutorizacion(context.Context, domain.DecisionAutorizacion) error {
	return r.err
}

func autorizadorBasePrueba(t *testing.T, ahora time.Time) (*vecapp.ServicioAutorizacion, *AlmacenAutorizacionMemoria) {
	t.Helper()
	almacen := NuevoAlmacenAutorizacionMemoria()
	rol := versionRolAutorizacionPrueba("tecnico_bolsa", "bolsa.expediente.leer", domain.AuthAssuranceSubstantial, []string{"gestion_bolsa"})
	sembrarVersionAutorizacion(t, almacen, rol)
	sembrarAsignacionAutorizacion(t, almacen, asignacionAutorizacionPrueba(perfilBolsaAutorizacionPrueba, "asig-bolsa", rol.Referencia(), "seleccion"))
	return nuevoAutorizadorPrueba(t, almacen, almacen, ahora), almacen
}

func nuevoAutorizadorPrueba(t *testing.T, fuente ports.FuenteAutorizacion, registro ports.RegistroDecisionesAutorizacion, ahora time.Time) *vecapp.ServicioAutorizacion {
	t.Helper()
	autorizador, err := vecapp.NuevoServicioAutorizacion(
		fuente,
		registro,
		registro.(ports.RegistroDenegacionesAutorizacion),
		relojAutorizacionPrueba{ahora: ahora},
		seguridad.GeneradorReferenciasCriptograficas{},
		vecapp.ConfiguracionServicioAutorizacion{
			VigenciaDecision: 90 * time.Second,
		})
	if err != nil {
		t.Fatalf("crear autorizador: %v", err)
	}
	return autorizador
}

type relojAutorizacionPrueba struct{ ahora time.Time }

func (r relojAutorizacionPrueba) Ahora() time.Time { return r.ahora }

type generadorReferenciaAutorizacionPrueba string

func (g generadorReferenciaAutorizacionPrueba) NuevaReferenciaDecisionAutorizacion() (string, error) {
	return string(g), nil
}

func sembrarVersionAutorizacion(t *testing.T, almacen *AlmacenAutorizacionMemoria, version domain.VersionRol) {
	t.Helper()
	if err := almacen.SembrarVersionRol(version); err != nil {
		t.Fatalf("sembrar version de rol: %v", err)
	}
}

func sembrarAsignacionAutorizacion(t *testing.T, almacen *AlmacenAutorizacionMemoria, asignacion domain.AsignacionPerfil) {
	t.Helper()
	if err := almacen.SembrarAsignacionPerfil(asignacion); err != nil {
		t.Fatalf("sembrar asignacion: %v", err)
	}
}

func sembrarPoliticaAutorizacion(t *testing.T, almacen *AlmacenAutorizacionMemoria, politica domain.PoliticaRestrictiva) {
	t.Helper()
	if err := almacen.SembrarPoliticaRestrictiva(politica); err != nil {
		t.Fatalf("sembrar politica: %v", err)
	}
}

func versionRolAutorizacionPrueba(id, accion string, garantia domain.AuthAssurance, finalidades []string) domain.VersionRol {
	ahora := instanteAutorizacionPrueba()
	return domain.VersionRol{
		RolID:   id,
		Version: 1,
		Nombre:  id,
		Estado:  domain.EstadoVersionRolPublicada,
		Concesiones: []domain.ConcesionRol{{
			Accion:           accion,
			ModuloID:         "bolsa",
			TipoRecurso:      "expediente",
			Finalidades:      append([]string(nil), finalidades...),
			GarantiaMinima:   garantia,
			CamposPermitidos: []string{"estado", "nombre"},
			Obligaciones:     []string{"trazar_acceso"},
		}},
		PublicadaPor: "responsable-seguridad",
		PublicadaEn:  ahora.Add(-24 * time.Hour),
	}
}

func asignacionAutorizacionPrueba(perfilRef, asignacionID, versionRolRef, unidad string) domain.AsignacionPerfil {
	ahora := instanteAutorizacionPrueba()
	return domain.AsignacionPerfil{
		AsignacionID:    asignacionID,
		Version:         1,
		PerfilActivoRef: perfilRef,
		PrincipalID:     principalAutorizacionPrueba,
		VersionRolRef:   versionRolRef,
		Estado:          domain.EstadoAsignacionPerfilActiva,
		Ambitos:         []domain.AmbitoPerfil{{Clave: "unidad", Valores: []string{unidad}}},
		VigenteDesde:    ahora.Add(-time.Hour),
		VigenteHasta:    ahora.Add(time.Hour),
		EmitidaPor:      "administrador-identidades",
		EmitidaEn:       ahora.Add(-2 * time.Hour),
	}
}

func politicaAutorizacionPrueba(id string, efecto domain.EfectoPoliticaRestrictiva) domain.PoliticaRestrictiva {
	ahora := instanteAutorizacionPrueba()
	return domain.PoliticaRestrictiva{
		PoliticaID:   id,
		Version:      1,
		Nombre:       id,
		Estado:       domain.EstadoPoliticaRestrictivaPublicada,
		Efecto:       efecto,
		Acciones:     []string{"bolsa.expediente.leer"},
		Modulos:      []string{"bolsa"},
		TiposRecurso: []string{"expediente"},
		VigenteDesde: ahora.Add(-time.Hour),
		VigenteHasta: ahora.Add(time.Hour),
		PublicadaPor: "responsable-seguridad",
		PublicadaEn:  ahora.Add(-24 * time.Hour),
	}
}

type revalidadorAutenticacionActorMemoriaPrueba struct {
	solicitud     domain.SolicitudRevalidacionAutenticacionActorV1
	autenticacion domain.AutenticacionRevalidadaV1
}

func (r revalidadorAutenticacionActorMemoriaPrueba) RevalidarAutenticacionActorV1(
	ctx context.Context,
	solicitud domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	if ctx == nil {
		return domain.AutenticacionRevalidadaV1{}, domain.ErrAutenticacionRevalidadaInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.AutenticacionRevalidadaV1{}, err
	}
	if solicitud != r.solicitud {
		return domain.AutenticacionRevalidadaV1{}, domain.ErrAutenticacionRevalidadaInvalida
	}
	return r.autenticacion, nil
}

func solicitudAutorizacionPrueba() domain.SolicitudAutorizacion {
	return solicitudAutorizacionPerfilPrueba(perfilBolsaAutorizacionPrueba, domain.AuthAssuranceHigh)
}

func solicitudAutorizacionPerfilPrueba(
	perfilActivoRef string,
	garantia domain.AuthAssurance,
) domain.SolicitudAutorizacion {
	ahora := instanteAutorizacionPrueba()
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_cuenta_autorizacion_prueba_0001",
		Metodo:    domain.AuthMethodCertificate,
		Garantia:  garantia,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef:      "vca_contexto_autorizacion_prueba_0001",
		VinculoVersion:  1,
		CuentaRef:       cuenta.CuentaRef,
		PersonaRef:      principalAutorizacionPrueba,
		PersonaVersion:  1,
		PerfilActivoRef: perfilActivoRef,
		PerfilVersion:   1,
		Estado:          domain.EstadoVinculoContextoActorActivo,
		VigenteDesde:    ahora.Add(-24 * time.Hour),
		VigenteHasta:    ahora.Add(24 * time.Hour),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, ahora.Add(-time.Minute))
	if err != nil {
		panic("fixture de actor de autorizacion invalido: " + err.Error())
	}
	solicitudRevalidacion := domain.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: "aut_autenticacion_autorizacion_prueba_0001",
		SesionRef:        "ses_sesion_autorizacion_prueba_0001",
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:             solicitudRevalidacion.AutenticacionRef,
		AutenticacionHuellaSHA256:    strings.Repeat("a", 64),
		AsercionRef:                  "ase_asercion_autorizacion_prueba_0001",
		SesionRef:                    solicitudRevalidacion.SesionRef,
		ControlSesionRef:             "cse_control_sesion_autorizacion_prueba_0001",
		ControlSesionRevision:        1,
		ControlSesionHuellaSHA256:    strings.Repeat("b", 64),
		CuentaRef:                    cuenta.CuentaRef,
		CuentaOrdinariaRef:           cuenta.CuentaRef,
		Superficie:                   domain.SuperficieAutenticacionExternaPersonalV1,
		MetodoObservado:              cuenta.Metodo,
		GarantiaObservada:            cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_politica_garantia_autorizacion_0001",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("c", 64),
		AutenticacionVerificadaEn:    ahora.Add(-2 * time.Hour),
		SesionEmitidaEn:              ahora.Add(-90 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-2 * time.Minute),
		SesionValidaHasta:            ahora.Add(time.Hour),
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV1(
		context.Background(),
		revalidadorAutenticacionActorMemoriaPrueba{
			solicitud: solicitudRevalidacion, autenticacion: autenticacion,
		},
		solicitudRevalidacion,
		actor,
		ahora,
	)
	if err != nil {
		panic("fixture de vinculo de autorizacion invalido: " + err.Error())
	}
	return domain.SolicitudAutorizacion{
		Principal:                 actor.Principal,
		PerfilActivoRef:           perfilActivoRef,
		ContextoActor:             actor,
		VinculoAutenticacionActor: vinculo,
		Accion:                    "bolsa.expediente.leer",
		Recurso: domain.RecursoAutorizable{
			Referencia: "expediente:bolsa:1",
			ModuloID:   "bolsa",
			Tipo:       "expediente",
			Ambitos:    map[string]string{"unidad": "seleccion"},
			Atributos:  map[string]string{},
		},
		Finalidad:      "gestion_bolsa",
		CorrelacionRef: "corr-autorizacion-1",
		Motivo:         "consultar expediente para tramitar la convocatoria",
	}
}

func instanteAutorizacionPrueba() time.Time {
	return time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
}
