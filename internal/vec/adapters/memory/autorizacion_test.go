package memory

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

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

type generadorCorrelacionAutorizacionMemoriaPrueba string

func (g generadorCorrelacionAutorizacionMemoriaPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return string(g), nil
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
			Correlacion: func() domain.ReferenciaCorrelacionAutorizacionV2 {
				referencia, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(
					context.Background(),
					generadorCorrelacionAutorizacionMemoriaPrueba(referenciaCorrelacionAutorizacionV2Prueba),
				)
				if err != nil {
					t.Fatalf("generar correlacion nominal: %v", err)
				}
				return referencia
			}(),
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
	if almacen.referenciasMotivoConcesionesV2[decisionV2.DecisionRef] != referenciaMotivo {
		t.Fatal("el registro V2 no conservo la referencia exacta de motivo")
	}
	if _, existe := almacen.referenciasMotivoDenegacionesV2[decisionV2.DecisionRef]; existe {
		t.Fatal("la preimagen de una concesion V2 aparecio en el almacen de denegaciones")
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

func TestRegistroDenegacionV1ConservaTrazaTrasRevocacionPosterior(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	autorizadorOrigen, origen := autorizadorBasePrueba(t, ahora)
	politica := politicaAutorizacionPrueba("denegacion_probatoria_v1", domain.EfectoPoliticaDenegar)
	sembrarPoliticaAutorizacion(t, origen, politica)
	decision, err := autorizadorOrigen.Exigir(context.Background(), solicitudAutorizacionPrueba())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Concedida ||
		decision.Codigo != "denegada_por_politica" {
		t.Fatalf("no se obtuvo la denegacion V1 de prueba: decision=%+v err=%v", decision, err)
	}

	_, destino := autorizadorBasePrueba(t, ahora)
	revocada := asignacionAutorizacionPrueba(
		perfilBolsaAutorizacionPrueba,
		"asig-bolsa",
		versionRolAutorizacionPrueba(
			"tecnico_bolsa",
			"bolsa.expediente.leer",
			domain.AuthAssuranceSubstantial,
			[]string{"gestion_bolsa"},
		).Referencia(),
		"seleccion",
	)
	revocada.Version = 2
	revocada.Estado = domain.EstadoAsignacionPerfilRevocada
	revocada.RevocadaPor = "responsable-seguridad"
	revocada.RevocadaEn = ahora.Add(-time.Minute)
	revocada.RevocacionRef = "acto:revocacion:posterior:denegacion:v1"
	sembrarAsignacionAutorizacion(t, destino, revocada)

	if err := destino.RegistrarDenegacionAutorizacion(context.Background(), decision); err != nil {
		t.Fatalf("la revocacion posterior borro la traza negativa V1: %v", err)
	}
	referenciaPolitica := decision.PoliticasEvaluadasRefs[0]
	huellaPolitica := decision.PoliticasEvaluadasHuellasSHA256[referenciaPolitica]
	decision.PoliticasEvaluadasRefs[0] = "politica:mutada:v1"
	decision.PoliticasEvaluadasHuellasSHA256[referenciaPolitica] = strings.Repeat("9", 64)
	registrada, err := destino.ObtenerDenegacion(context.Background(), decision.DecisionRef)
	if err != nil || registrada.PoliticasEvaluadasRefs[0] != referenciaPolitica ||
		registrada.PoliticasEvaluadasHuellasSHA256[referenciaPolitica] != huellaPolitica {
		t.Fatalf("la denegacion V1 no quedo copiada defensivamente: decision=%+v err=%v", registrada, err)
	}

	malformada := registrada
	malformada.DecisionRef = "decision:denegacion:v1:malformada"
	malformada.AsignacionHuellaSHA256 = "huella-no-canonica"
	if err := destino.RegistrarDenegacionAutorizacion(context.Background(), malformada); !errors.Is(err, domain.ErrDecisionAutorizacionInvalida) {
		t.Fatalf("el registro negativo V1 acepto evidencia malformada: %v", err)
	}

	// La unicidad es fisica y cruzada: una referencia ya materializada como
	// capacidad no puede reaparecer en el almacen probatorio.
	autorizadorConcesion, destinoCruce := autorizadorBasePrueba(t, ahora)
	concesion, err := autorizadorConcesion.Exigir(context.Background(), solicitudAutorizacionPrueba())
	if err != nil {
		t.Fatalf("crear concesion para probar el cruce fisico: %v", err)
	}
	cruzada := registrada
	cruzada.DecisionRef = concesion.DecisionRef
	if err := destinoCruce.RegistrarDenegacionAutorizacion(context.Background(), cruzada); !errors.Is(err, ports.ErrVersionAutorizacionYaExiste) {
		t.Fatalf("una referencia concedida reaparecio como denegacion: %v", err)
	}
}

func TestRegistroDenegacionV2ConservaTrazaTrasCambioDeCatalogo(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	_, origen := autorizadorBasePrueba(t, ahora)
	referenciaMotivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("f", 64),
		EntradaClave:         claveMotivoAutorizacionV2Prueba,
	}
	servicioV2, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV2(
		origen,
		origen,
		origen,
		validadorMotivoAutorizacionMemoriaPrueba{referencia: referenciaMotivo},
		relojAutorizacionPrueba{ahora: ahora},
		generadorReferenciaAutorizacionPrueba("decision:denegacion:probatoria:v2"),
		vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatalf("crear servicio V2: %v", err)
	}
	solicitudBase := solicitudAutorizacionPrueba()
	solicitudBase.Recurso.Ambitos["unidad"] = "nominas"
	solicitudV2 := nuevaSolicitudAutorizacionLigadaV2MemoriaPrueba(t, solicitudBase, referenciaMotivo)
	decision, err := servicioV2.ExigirSolicitudLigadaV2(context.Background(), solicitudV2)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Concedida ||
		decision.Codigo != "ambito_no_autorizado" {
		t.Fatalf("no se obtuvo la denegacion V2 de prueba: decision=%+v err=%v", decision, err)
	}
	orden, err := ports.NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(decision, referenciaMotivo)
	if err != nil {
		t.Fatalf("crear orden V2: %v", err)
	}

	_, destino := autorizadorBasePrueba(t, ahora)
	sembrarPoliticaAutorizacion(t, destino, politicaAutorizacionPrueba(
		"publicada_despues_de_evaluar_v2",
		domain.EfectoPoliticaRestringir,
	))
	if err := destino.RegistrarDenegacionAutorizacionSolicitudLigadaV2(context.Background(), orden); err != nil {
		t.Fatalf("el cambio posterior de catalogo borro la traza negativa V2: %v", err)
	}
	registrada, err := destino.ObtenerDenegacion(context.Background(), decision.DecisionRef)
	if err != nil || !registrada.TieneSolicitudLigadaV2() ||
		destino.referenciasMotivoDenegacionesV2[decision.DecisionRef] != referenciaMotivo {
		t.Fatalf("la denegacion V2 no conservo decision y preimagen: decision=%+v err=%v", registrada, err)
	}
	if _, existe := destino.referenciasMotivoConcesionesV2[decision.DecisionRef]; existe {
		t.Fatal("la preimagen de una denegacion V2 aparecio en el almacen de concesiones")
	}
	if err := destino.RegistrarDenegacionAutorizacion(context.Background(), decision); !errors.Is(err, domain.ErrDecisionAutorizacionInvalida) {
		t.Fatalf("el registro V1 acepto una denegacion V2: %v", err)
	}
	referenciaNoLigada := referenciaMotivo
	referenciaNoLigada.EntradaClave = "motivo_44444444444444444444444444444444"
	if _, err := ports.NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(decision, referenciaNoLigada); !errors.Is(err, ports.ErrOrdenRegistroAutorizacionSolicitudLigadaV2Invalida) {
		t.Fatalf("se acepto una preimagen de motivo no ligada: %v", err)
	}

	// Varias escrituras de la misma evidencia compiten bajo un unico cerrojo:
	// exactamente una gana y el resto observa el duplicado.
	_, destinoConcurrente := autorizadorBasePrueba(t, ahora)
	const total = 16
	errores := make(chan error, total)
	var grupo sync.WaitGroup
	for indice := 0; indice < total; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			errores <- destinoConcurrente.RegistrarDenegacionAutorizacionSolicitudLigadaV2(
				context.Background(),
				orden,
			)
		}()
	}
	grupo.Wait()
	close(errores)
	correctas, duplicadas := 0, 0
	for err := range errores {
		switch {
		case err == nil:
			correctas++
		case errors.Is(err, ports.ErrVersionAutorizacionYaExiste):
			duplicadas++
		default:
			t.Fatalf("error concurrente inesperado: %v", err)
		}
	}
	if correctas != 1 || duplicadas != total-1 {
		t.Fatalf("registro concurrente V2: correctas=%d duplicadas=%d", correctas, duplicadas)
	}
}

func TestRegistroConcesionV2MantieneCASTrasCambioDeCatalogo(t *testing.T) {
	ahora := instanteAutorizacionPrueba()
	_, origen := autorizadorBasePrueba(t, ahora)
	referenciaMotivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("f", 64),
		EntradaClave:         claveMotivoAutorizacionV2Prueba,
	}
	servicioV2, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV2(
		origen,
		origen,
		origen,
		validadorMotivoAutorizacionMemoriaPrueba{referencia: referenciaMotivo},
		relojAutorizacionPrueba{ahora: ahora},
		generadorReferenciaAutorizacionPrueba("decision:concesion:cas:v2"),
		vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatalf("crear servicio V2: %v", err)
	}
	solicitudV2 := nuevaSolicitudAutorizacionLigadaV2MemoriaPrueba(
		t,
		solicitudAutorizacionPrueba(),
		referenciaMotivo,
	)
	decision, err := servicioV2.ExigirSolicitudLigadaV2(context.Background(), solicitudV2)
	if err != nil || !decision.Concedida {
		t.Fatalf("no se obtuvo la concesion V2 de prueba: decision=%+v err=%v", decision, err)
	}
	orden, err := ports.NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(decision, referenciaMotivo)
	if err != nil {
		t.Fatalf("crear orden V2: %v", err)
	}

	_, destino := autorizadorBasePrueba(t, ahora)
	sembrarPoliticaAutorizacion(t, destino, politicaAutorizacionPrueba(
		"publicada_antes_del_registro_concesion_v2",
		domain.EfectoPoliticaRestringir,
	))
	err = destino.RegistrarDecisionSolicitudLigadaV2SiInstantaneaVigente(
		context.Background(),
		orden,
	)
	if !errors.Is(err, ports.ErrInstantaneaAutorizacionObsoleta) {
		t.Fatalf("la concesion V2 omitio el CAS del catalogo actual: %v", err)
	}
	if _, err := destino.ObtenerDecision(context.Background(), decision.DecisionRef); !errors.Is(err, ports.ErrDecisionAutorizacionNoEncontrada) {
		t.Fatalf("la concesion V2 obsoleta llego al almacen ejecutable: %v", err)
	}
	if _, existe := destino.referenciasMotivoConcesionesV2[decision.DecisionRef]; existe {
		t.Fatal("la concesion V2 obsoleta dejo una preimagen huerfana")
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
