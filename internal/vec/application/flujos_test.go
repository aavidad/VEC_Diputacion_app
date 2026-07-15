package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const finalidadEjecucionFlujoPrueba = "tramitar_solicitud_bolsa"

type autorizadorFlujosPrueba struct {
	mu        sync.Mutex
	ahora     time.Time
	siguiente int
	acciones  []string
}

func (a *autorizadorFlujosPrueba) Exigir(
	_ context.Context,
	solicitud domain.SolicitudAutorizacion,
) (domain.DecisionAutorizacion, error) {
	if err := solicitud.Validar(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if solicitud.Principal.ID == personaAutorizacionPrueba("sin-autorizacion") {
		return domain.DecisionAutorizacion{}, domain.ErrAutorizacionDenegada
	}
	a.mu.Lock()
	a.siguiente++
	secuencia := a.siguiente
	a.acciones = append(a.acciones, solicitud.Accion)
	a.mu.Unlock()
	return completarDecisionAutorizacionPrueba(solicitud, domain.DecisionAutorizacion{
		DecisionRef:            fmt.Sprintf("decision-flujo-interna-%03d", secuencia),
		Concedida:              true,
		Codigo:                 "concedida",
		PrincipalID:            solicitud.Principal.ID,
		PerfilActivoRef:        solicitud.PerfilActivoRef,
		Accion:                 solicitud.Accion,
		RecursoRef:             solicitud.Recurso.Referencia,
		Finalidad:              solicitud.Finalidad,
		CorrelacionRef:         solicitud.CorrelacionRef,
		AsignacionRef:          "asignacion:flujos:v1",
		AsignacionHuellaSHA256: strings.Repeat("a", 64),
		VersionRolRef:          "rol:flujos:v1",
		VersionRolHuellaSHA256: strings.Repeat("b", 64),
		GarantiaMinima:         domain.AuthAssuranceLow,
		EmitidaEn:              a.ahora,
		ValidaHasta:            a.ahora.Add(time.Minute),
	}), nil
}

func (a *autorizadorFlujosPrueba) Acciones() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.acciones...)
}

type evaluadorReglasFlujoPrueba struct {
	mu        sync.Mutex
	ahora     time.Time
	siguiente int
	concedida bool
	alterar   func(*domain.DecisionReglaFlujo)
	ultimaRef string
	llegadas  chan<- struct{}
	continuar <-chan struct{}
}

func (e *evaluadorReglasFlujoPrueba) EvaluarReglaFlujo(
	_ context.Context,
	solicitud ports.SolicitudEvaluarReglaFlujo,
) (domain.DecisionReglaFlujo, error) {
	if e.llegadas != nil {
		e.llegadas <- struct{}{}
		<-e.continuar
	}
	huella, err := solicitud.Definicion.HuellaContenidoSHA256()
	if err != nil {
		return domain.DecisionReglaFlujo{}, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.siguiente++
	codigo := "regla_satisfecha"
	if !e.concedida {
		codigo = "regla_no_satisfecha"
	}
	decision := domain.DecisionReglaFlujo{
		DecisionRef:                     fmt.Sprintf("decision-regla-flujo-%03d", e.siguiente),
		Concedida:                       e.concedida,
		Codigo:                          codigo,
		DefinicionRef:                   solicitud.Definicion.Referencia(),
		DefinicionContenidoHuellaSHA256: huella,
		InstanciaRef:                    solicitud.Instancia.ID,
		InstanciaRevision:               solicitud.Instancia.Revision,
		EstadoOrigen:                    solicitud.Instancia.EstadoActual,
		TransicionClave:                 solicitud.Transicion.Clave,
		ReglaRef:                        solicitud.Transicion.ReglaRef,
		ActorID:                         solicitud.ActorID,
		Finalidad:                       solicitud.Finalidad,
		CorrelacionRef:                  solicitud.CorrelacionRef,
		EntradaHuellaHMAC:               "hmac-sha256:regla:" + strings.Repeat("c", 64),
		ResultadoHuellaSHA256:           strings.Repeat("d", 64),
		EvaluadaEn:                      e.ahora,
		ValidaHasta:                     e.ahora.Add(10 * time.Minute),
	}
	if e.alterar != nil {
		e.alterar(&decision)
	}
	e.ultimaRef = decision.DecisionRef
	return decision, nil
}

func (e *evaluadorReglasFlujoPrueba) Configurar(concedida bool, alterar func(*domain.DecisionReglaFlujo)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.concedida = concedida
	e.alterar = alterar
}

func (e *evaluadorReglasFlujoPrueba) UltimaReferencia() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.ultimaRef
}

type verificadorAprobacionesFlujoPrueba struct {
	mu      sync.Mutex
	ahora   time.Time
	alterar func(*domain.EvidenciaAprobacionFlujo)
}

func (v *verificadorAprobacionesFlujoPrueba) VerificarAprobacionFlujo(
	_ context.Context,
	solicitud ports.SolicitudVerificarAprobacionFlujo,
) (domain.EvidenciaAprobacionFlujo, error) {
	huella, err := solicitud.Definicion.HuellaContenidoSHA256()
	if err != nil {
		return domain.EvidenciaAprobacionFlujo{}, err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	evidencia := domain.EvidenciaAprobacionFlujo{
		AprobacionRef:                   solicitud.ReferenciaAprobacion,
		AprobadorID:                     "responsable-aprobador-independiente",
		PerfilAprobadorRef:              "perfil:responsable:aprobador",
		Garantia:                        domain.AuthAssuranceHigh,
		SolicitanteID:                   solicitud.SolicitanteID,
		DefinicionRef:                   solicitud.Definicion.Referencia(),
		DefinicionContenidoHuellaSHA256: huella,
		InstanciaRef:                    solicitud.Instancia.ID,
		InstanciaRevision:               solicitud.Instancia.Revision,
		EstadoOrigen:                    solicitud.Instancia.EstadoActual,
		TransicionClave:                 solicitud.Transicion.Clave,
		DecisionReglaRef:                solicitud.DecisionRegla.DecisionRef,
		Motivo:                          "Revision independiente favorable",
		EvidenciaHuellaSHA256:           strings.Repeat("e", 64),
		AprobadaEn:                      v.ahora,
		ValidaHasta:                     v.ahora.Add(10 * time.Minute),
	}
	if v.alterar != nil {
		v.alterar(&evidencia)
	}
	return evidencia, nil
}

func (v *verificadorAprobacionesFlujoPrueba) Configurar(alterar func(*domain.EvidenciaAprobacionFlujo)) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.alterar = alterar
}

type entornoFlujosPrueba struct {
	ahora         time.Time
	store         *memory.Store
	autorizador   *autorizadorFlujosPrueba
	evaluador     *evaluadorReglasFlujoPrueba
	aprobaciones  *verificadorAprobacionesFlujoPrueba
	gobierno      *ServicioGobiernoFlujos
	ejecucion     *ServicioEjecucionFlujos
	definicion    domain.DefinicionFlujo
	configuracion domain.ConfiguracionBorradorFlujo
}

func nuevoEntornoFlujosPrueba(t *testing.T) *entornoFlujosPrueba {
	t.Helper()
	ahora := time.Date(2026, time.July, 14, 18, 0, 0, 0, time.UTC)
	reloj := relojDocumentalFijo{ahora: ahora}
	store := memory.NewStore()
	autorizador := &autorizadorFlujosPrueba{ahora: ahora}
	servicioCatalogos, err := NuevoServicioCatalogos(store, store, autorizador, reloj)
	if err != nil {
		t.Fatalf("NuevoServicioCatalogos() error = %v", err)
	}
	claves := []string{"borrador", "presentada", "en_revision", "admitida", "excluida"}
	entradas := make([]domain.EntradaCatalogoConfigurable, 0, len(claves))
	for indice, clave := range claves {
		entradas = append(entradas, domain.EntradaCatalogoConfigurable{
			Clave: clave, Etiqueta: "Estado " + clave, Orden: (indice + 1) * 10, VigenteDesde: ahora.Add(-time.Hour),
		})
	}
	borradorCatalogo, err := servicioCatalogos.CrearBorrador(context.Background(), OrdenCrearBorradorCatalogo{
		Credenciales:   credencialesCatalogosPruebaEn(t, "tecnico-catalogos-flujo", ahora),
		Finalidad:      "gobierno_configuracion_bolsa",
		ID:             "bolsa.estados_solicitud",
		Version:        1,
		ModuloID:       "bolsa",
		Nombre:         "Estados de solicitud",
		Descripcion:    "Estados administrables del procedimiento.",
		FuenteRef:      "bases:bolsa:2026",
		Entradas:       entradas,
		Motivo:         "Crear catalogo para el flujo de solicitud",
		CorrelacionRef: "corr-catalogo-flujo-crear",
	})
	if err != nil {
		t.Fatalf("crear catalogo de flujo: %v", err)
	}
	catalogo, err := servicioCatalogos.Publicar(context.Background(), OrdenPublicarCatalogo{
		Credenciales:   credencialesCatalogosPruebaEn(t, "responsable-catalogos-flujo", ahora),
		Finalidad:      "gobierno_configuracion_bolsa",
		ID:             borradorCatalogo.ID,
		Version:        1,
		AprobacionRef:  "aprobacion-catalogo-flujo-1",
		Motivo:         "Aprobar catalogo para la convocatoria",
		CorrelacionRef: "corr-catalogo-flujo-publicar",
	})
	if err != nil {
		t.Fatalf("publicar catalogo de flujo: %v", err)
	}
	huellaCatalogo, err := catalogo.HuellaContenidoSHA256()
	if err != nil {
		t.Fatalf("huella de catalogo: %v", err)
	}
	estado := func(clave string, orden int, terminal bool) domain.EstadoFlujoConfigurable {
		return domain.EstadoFlujoConfigurable{
			Clave: clave,
			Catalogo: domain.ReferenciaEntradaCatalogo{
				CatalogoID:           catalogo.ID,
				CatalogoVersion:      catalogo.Version,
				CatalogoHuellaSHA256: huellaCatalogo,
				EntradaClave:         clave,
			},
			Orden: orden, Terminal: terminal,
		}
	}
	configuracion := domain.ConfiguracionBorradorFlujo{
		Nombre:                          "Tramitacion de solicitud de bolsa",
		Descripcion:                     "Flujo administrable del procedimiento selectivo.",
		FuenteRef:                       "bases:bolsa:2026",
		EstadoInicial:                   "borrador",
		AccionInicio:                    "bolsa.solicitud.crear",
		GarantiaInicio:                  domain.AuthAssuranceSubstantial,
		PermiteFinalizacionTrasRetirada: true,
		Estados: []domain.EstadoFlujoConfigurable{
			estado("borrador", 10, false), estado("presentada", 20, false), estado("en_revision", 30, false),
			estado("admitida", 40, true), estado("excluida", 50, true),
		},
		Transiciones: []domain.TransicionFlujoConfigurable{
			{
				Clave: "presentar", Desde: []string{"borrador"}, Hacia: "presentada",
				Accion: "bolsa.solicitud.presentar", ReglaRef: "regla:bolsa:presentar:v1",
				Prioridad: 10, GarantiaMinima: domain.AuthAssuranceSubstantial,
			},
			{
				Clave: "iniciar_revision", Desde: []string{"presentada"}, Hacia: "en_revision",
				Accion: "bolsa.solicitud.revisar", ReglaRef: "regla:bolsa:revisar:v1",
				Prioridad: 20, GarantiaMinima: domain.AuthAssuranceHigh,
			},
			{
				Clave: "admitir", Desde: []string{"en_revision"}, Hacia: "admitida",
				Accion: "bolsa.solicitud.admitir", ReglaRef: "regla:bolsa:admitir:v1",
				Prioridad: 30, GarantiaMinima: domain.AuthAssuranceHigh, RequiereAprobacion: true,
			},
			{
				Clave: "excluir", Desde: []string{"en_revision"}, Hacia: "excluida",
				Accion: "bolsa.solicitud.excluir", ReglaRef: "regla:bolsa:excluir:v1",
				Prioridad: 40, GarantiaMinima: domain.AuthAssuranceHigh, RequiereAprobacion: true,
			},
		},
	}
	gobierno, err := NuevoServicioGobiernoFlujos(store, store, store, autorizador, reloj)
	if err != nil {
		t.Fatalf("NuevoServicioGobiernoFlujos() error = %v", err)
	}
	borrador, err := gobierno.CrearBorrador(context.Background(), OrdenCrearBorradorFlujo{
		Principal:      principalFlujosPrueba("tecnico-flujos-aplicacion"),
		PerfilActivo:   perfilAutorizacionPrueba("flujos:tecnico"),
		Finalidad:      "gobierno_flujos_bolsa",
		ID:             "bolsa.solicitud",
		Version:        1,
		ModuloID:       "bolsa",
		TipoEntidad:    "solicitud",
		Configuracion:  configuracion,
		Motivo:         "Crear flujo inicial de la convocatoria",
		CorrelacionRef: "corr-flujo-crear-1",
	})
	if err != nil {
		t.Fatalf("crear flujo: %v", err)
	}
	publicada, err := gobierno.Publicar(context.Background(), OrdenPublicarFlujo{
		Principal:      principalFlujosPrueba("responsable-flujos-aplicacion"),
		PerfilActivo:   perfilAutorizacionPrueba("flujos:responsable"),
		Finalidad:      "gobierno_flujos_bolsa",
		ID:             borrador.ID,
		Version:        borrador.Version,
		AprobacionRef:  "aprobacion-flujo-aplicacion-1",
		Motivo:         "Aprobar flujo para la convocatoria",
		CorrelacionRef: "corr-flujo-publicar-1",
	})
	if err != nil {
		t.Fatalf("publicar flujo: %v", err)
	}
	evaluador := &evaluadorReglasFlujoPrueba{ahora: ahora, concedida: true}
	aprobaciones := &verificadorAprobacionesFlujoPrueba{ahora: ahora}
	ejecucion, err := NuevoServicioEjecucionFlujos(
		store, store, store, evaluador, store, aprobaciones, autorizador, store, reloj,
	)
	if err != nil {
		t.Fatalf("NuevoServicioEjecucionFlujos() error = %v", err)
	}
	return &entornoFlujosPrueba{
		ahora: ahora, store: store, autorizador: autorizador, evaluador: evaluador,
		aprobaciones: aprobaciones, gobierno: gobierno, ejecucion: ejecucion,
		definicion: publicada, configuracion: configuracion,
	}
}

func principalFlujosPrueba(id string) domain.Principal {
	return domain.Principal{
		ID: personaAutorizacionPrueba(id), Roles: []string{"declarado_no_autoritativo"}, Permissions: []string{"vec.flujos.permiso_declarado_no_autoritativo"},
		AuthMethod: domain.AuthMethodCertificate, AuthAssurance: domain.AuthAssuranceHigh,
	}
}

func ordenInicioFlujoPrueba(entorno *entornoFlujosPrueba, actorID, entidadRef string) OrdenIniciarInstanciaFlujo {
	return OrdenIniciarInstanciaFlujo{
		Principal:      principalFlujosPrueba(actorID),
		PerfilActivo:   perfilAutorizacionPrueba("candidato:1"),
		Finalidad:      finalidadEjecucionFlujoPrueba,
		DefinicionID:   entorno.definicion.ID,
		Version:        entorno.definicion.Version,
		EntidadRef:     entidadRef,
		Motivo:         "Crear solicitud propia",
		CorrelacionRef: "corr-iniciar-" + entidadRef,
	}
}

func TestServicioGobiernoFlujosPublicaConCatalogosExactosYTrazabilidad(t *testing.T) {
	entorno := nuevoEntornoFlujosPrueba(t)
	guardada, err := entorno.store.ObtenerDefinicionFlujo(context.Background(), entorno.definicion.ID, 1)
	if err != nil || guardada.Estado != domain.EstadoDefinicionFlujoPublicada {
		t.Fatalf("definicion guardada = %+v, %v", guardada, err)
	}
	auditoria, err := entorno.store.ListAudit(context.Background(), entorno.definicion.Referencia())
	if err != nil || len(auditoria) != 2 {
		t.Fatalf("auditoria de gobierno = %+v, %v", auditoria, err)
	}
	if auditoria[0].AfterHash != auditoria[1].BeforeHash ||
		auditoria[0].Metadata["huella_contenido_sha256"] != auditoria[1].Metadata["huella_contenido_sha256"] {
		t.Fatalf("cadena de gobierno incoherente: %+v", auditoria)
	}

	configuracionInvalida := entorno.configuracion
	configuracionInvalida.Estados = append([]domain.EstadoFlujoConfigurable(nil), entorno.configuracion.Estados...)
	configuracionInvalida.Estados[0].Catalogo.CatalogoHuellaSHA256 = strings.Repeat("f", 64)
	borrador, err := entorno.gobierno.CrearBorrador(context.Background(), OrdenCrearBorradorFlujo{
		Principal:      principalFlujosPrueba("tecnico-flujo-invalido"),
		PerfilActivo:   perfilAutorizacionPrueba("flujos:tecnico"),
		Finalidad:      "gobierno_flujos_bolsa",
		ID:             "bolsa.solicitud_invalida",
		Version:        1,
		ModuloID:       "bolsa",
		TipoEntidad:    "solicitud",
		Configuracion:  configuracionInvalida,
		Motivo:         "Probar referencia exacta de catalogo",
		CorrelacionRef: "corr-flujo-invalido-crear",
	})
	if err != nil {
		t.Fatalf("crear borrador con referencia pendiente de validar: %v", err)
	}
	_, err = entorno.gobierno.Publicar(context.Background(), OrdenPublicarFlujo{
		Principal:      principalFlujosPrueba("responsable-flujo-invalido"),
		PerfilActivo:   perfilAutorizacionPrueba("flujos:responsable"),
		Finalidad:      "gobierno_flujos_bolsa",
		ID:             borrador.ID,
		Version:        1,
		AprobacionRef:  "aprobacion-flujo-invalido",
		Motivo:         "Intentar publicar referencia alterada",
		CorrelacionRef: "corr-flujo-invalido-publicar",
	})
	if !errors.Is(err, ErrReferenciaCatalogoFlujoInvalida) {
		t.Fatalf("referencia de catalogo alterada: error = %v", err)
	}
	guardada, err = entorno.store.ObtenerDefinicionFlujo(context.Background(), borrador.ID, 1)
	if err != nil || guardada.Estado != domain.EstadoDefinicionFlujoBorrador {
		t.Fatalf("el fallo altero el borrador: %+v, %v", guardada, err)
	}
}

func TestServicioEjecucionFlujosRecorreInicioYTransicionConAccionesDinamicas(t *testing.T) {
	entorno := nuevoEntornoFlujosPrueba(t)
	instancia, err := entorno.ejecucion.IniciarInstancia(
		context.Background(), ordenInicioFlujoPrueba(entorno, "persona-candidata-1", "solicitud-1001"),
	)
	if err != nil {
		t.Fatalf("IniciarInstancia() error = %v", err)
	}
	actualizada, err := entorno.ejecucion.AplicarTransicion(context.Background(), OrdenAplicarTransicionFlujo{
		Principal:       principalFlujosPrueba("persona-candidata-1"),
		PerfilActivo:    perfilAutorizacionPrueba("candidato:1"),
		Finalidad:       finalidadEjecucionFlujoPrueba,
		InstanciaID:     instancia.ID,
		TransicionClave: "presentar",
		Motivo:          "Presentar solicitud completa",
		CorrelacionRef:  "corr-presentar-1001",
	})
	if err != nil || actualizada.EstadoActual != "presentada" || actualizada.Revision != 2 {
		t.Fatalf("AplicarTransicion() = %+v, %v", actualizada, err)
	}
	decision, err := entorno.store.ObtenerDecisionReglaFlujo(context.Background(), entorno.evaluador.UltimaReferencia())
	if err != nil || !decision.Concedida || decision.InstanciaRevision != 1 || decision.EstadoOrigen != "borrador" {
		t.Fatalf("decision registrada = %+v, %v", decision, err)
	}
	auditoria, err := entorno.store.ListAudit(context.Background(), instancia.ID)
	if err != nil || len(auditoria) != 3 {
		t.Fatalf("auditoria de instancia = %+v, %v", auditoria, err)
	}
	if auditoria[0].Action != domain.AccionInstanciaFlujoIniciada ||
		auditoria[1].Action != domain.AccionDecisionReglaFlujoRegistrada ||
		auditoria[2].Action != domain.AccionInstanciaFlujoTransicionada ||
		auditoria[2].BeforeHash != auditoria[0].AfterHash {
		t.Fatalf("secuencia de evidencia inesperada: %+v", auditoria)
	}
	acciones := entorno.autorizador.Acciones()
	if !contieneTexto(acciones, "bolsa.solicitud.crear") || !contieneTexto(acciones, "bolsa.solicitud.presentar") {
		t.Fatalf("no se usaron las acciones configuradas: %v", acciones)
	}
}

func TestServicioEjecucionFlujosRegistraDenegacionSinCambiarEstado(t *testing.T) {
	entorno := nuevoEntornoFlujosPrueba(t)
	instancia, err := entorno.ejecucion.IniciarInstancia(
		context.Background(), ordenInicioFlujoPrueba(entorno, "persona-candidata-2", "solicitud-1002"),
	)
	if err != nil {
		t.Fatalf("IniciarInstancia() error = %v", err)
	}
	entorno.evaluador.Configurar(false, nil)
	_, err = entorno.ejecucion.AplicarTransicion(context.Background(), OrdenAplicarTransicionFlujo{
		Principal:       principalFlujosPrueba("persona-candidata-2"),
		PerfilActivo:    perfilAutorizacionPrueba("candidato:2"),
		Finalidad:       finalidadEjecucionFlujoPrueba,
		InstanciaID:     instancia.ID,
		TransicionClave: "presentar",
		Motivo:          "Intentar presentar sin cumplir requisitos",
		CorrelacionRef:  "corr-presentar-denegada-1002",
	})
	if !errors.Is(err, domain.ErrReglaFlujoDenegada) {
		t.Fatalf("regla denegada: error = %v", err)
	}
	guardada, err := entorno.store.ObtenerInstanciaFlujo(context.Background(), instancia.ID)
	if err != nil || guardada.EstadoActual != "borrador" || guardada.Revision != 1 {
		t.Fatalf("la denegacion cambio estado: %+v, %v", guardada, err)
	}
	decision, err := entorno.store.ObtenerDecisionReglaFlujo(context.Background(), entorno.evaluador.UltimaReferencia())
	if err != nil || decision.Concedida {
		t.Fatalf("denegacion no registrada: %+v, %v", decision, err)
	}
	auditoria, _ := entorno.store.ListAudit(context.Background(), instancia.ID)
	if len(auditoria) != 2 || auditoria[1].Result != "denegada" {
		t.Fatalf("traza de denegacion = %+v", auditoria)
	}
}

func TestServicioEjecucionFlujosRechazaDecisionForjadaYPermisosDeclarados(t *testing.T) {
	entorno := nuevoEntornoFlujosPrueba(t)
	ordenSinAutorizacion := ordenInicioFlujoPrueba(entorno, "sin-autorizacion", "solicitud-1003")
	ordenSinAutorizacion.Principal.Roles = []string{"administrador_declarado_no_autoritativo"}
	ordenSinAutorizacion.Principal.Permissions = []string{"vec.administracion.declarada", "bolsa.solicitud.crear"}
	if _, err := entorno.ejecucion.IniciarInstancia(context.Background(), ordenSinAutorizacion); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("permisos declarados concedieron acceso: error = %v", err)
	}

	instancia, err := entorno.ejecucion.IniciarInstancia(
		context.Background(), ordenInicioFlujoPrueba(entorno, "persona-candidata-3", "solicitud-1004"),
	)
	if err != nil {
		t.Fatalf("IniciarInstancia() error = %v", err)
	}
	entorno.evaluador.Configurar(true, func(decision *domain.DecisionReglaFlujo) {
		decision.ActorID = "persona-distinta"
	})
	_, err = entorno.ejecucion.AplicarTransicion(context.Background(), OrdenAplicarTransicionFlujo{
		Principal:       principalFlujosPrueba("persona-candidata-3"),
		PerfilActivo:    perfilAutorizacionPrueba("candidato:3"),
		Finalidad:       finalidadEjecucionFlujoPrueba,
		InstanciaID:     instancia.ID,
		TransicionClave: "presentar",
		Motivo:          "Presentar con decision manipulada",
		CorrelacionRef:  "corr-decision-forjada-1004",
	})
	if !errors.Is(err, domain.ErrDecisionReglaInvalida) {
		t.Fatalf("decision forjada: error = %v", err)
	}
	if _, err := entorno.store.ObtenerDecisionReglaFlujo(context.Background(), entorno.evaluador.UltimaReferencia()); !errors.Is(err, ports.ErrDecisionReglaFlujoNoEncontrada) {
		t.Fatalf("la decision forjada fue registrada: %v", err)
	}
}

func TestServicioEjecucionFlujosVerificaAprobacionExacta(t *testing.T) {
	entorno := nuevoEntornoFlujosPrueba(t)
	instancia, err := entorno.ejecucion.IniciarInstancia(
		context.Background(), ordenInicioFlujoPrueba(entorno, "persona-candidata-4", "solicitud-1005"),
	)
	if err != nil {
		t.Fatalf("IniciarInstancia() error = %v", err)
	}
	aplicar := func(actorID, perfil, transicion, correlacion string) domain.InstanciaFlujo {
		t.Helper()
		resultado, err := entorno.ejecucion.AplicarTransicion(context.Background(), OrdenAplicarTransicionFlujo{
			Principal:       principalFlujosPrueba(actorID),
			PerfilActivo:    perfil,
			Finalidad:       finalidadEjecucionFlujoPrueba,
			InstanciaID:     instancia.ID,
			TransicionClave: transicion,
			Motivo:          "Ejecutar " + transicion,
			CorrelacionRef:  correlacion,
		})
		if err != nil {
			t.Fatalf("%s: %v", transicion, err)
		}
		instancia = resultado
		return resultado
	}
	aplicar("persona-candidata-4", perfilAutorizacionPrueba("candidato:4"), "presentar", "corr-presentar-1005")
	aplicar("tecnico-rrhh-4", perfilAutorizacionPrueba("rrhh:tecnico"), "iniciar_revision", "corr-revisar-1005")

	ordenAdmitir := OrdenAplicarTransicionFlujo{
		Principal:       principalFlujosPrueba("responsable-rrhh-4"),
		PerfilActivo:    perfilAutorizacionPrueba("rrhh:responsable"),
		Finalidad:       finalidadEjecucionFlujoPrueba,
		InstanciaID:     instancia.ID,
		TransicionClave: "admitir",
		Motivo:          "Admitir solicitud revisada",
		CorrelacionRef:  "corr-admitir-1005-sin-aprobacion",
	}
	if _, err := entorno.ejecucion.AplicarTransicion(context.Background(), ordenAdmitir); !errors.Is(err, domain.ErrAprobacionFlujoRequerida) {
		t.Fatalf("admision sin aprobacion: error = %v", err)
	}

	entorno.aprobaciones.Configurar(func(evidencia *domain.EvidenciaAprobacionFlujo) {
		evidencia.InstanciaRevision++
	})
	ordenAdmitir.AprobacionRef = "aprobacion-admitir-1005-invalida"
	ordenAdmitir.CorrelacionRef = "corr-admitir-1005-invalida"
	if _, err := entorno.ejecucion.AplicarTransicion(context.Background(), ordenAdmitir); !errors.Is(err, ports.ErrAprobacionFlujoNoVerificada) {
		t.Fatalf("aprobacion de otra revision: error = %v", err)
	}

	entorno.aprobaciones.Configurar(nil)
	ordenAdmitir.AprobacionRef = "aprobacion-admitir-1005"
	ordenAdmitir.CorrelacionRef = "corr-admitir-1005"
	admitida, err := entorno.ejecucion.AplicarTransicion(context.Background(), ordenAdmitir)
	if err != nil || admitida.EstadoActual != "admitida" || admitida.UltimaAprobacionRef != ordenAdmitir.AprobacionRef {
		t.Fatalf("admision = %+v, %v", admitida, err)
	}
	auditoria, _ := entorno.store.ListAudit(context.Background(), instancia.ID)
	ultima := auditoria[len(auditoria)-1]
	if ultima.Action != domain.AccionInstanciaFlujoTransicionada || ultima.Metadata["aprobacion_huella_sha256"] == "" {
		t.Fatalf("aprobacion no fijada en la evidencia: %+v", ultima)
	}
}

func TestServicioEjecucionFlujosSoloConfirmaUnaTransicionConcurrente(t *testing.T) {
	entorno := nuevoEntornoFlujosPrueba(t)
	instancia, err := entorno.ejecucion.IniciarInstancia(
		context.Background(), ordenInicioFlujoPrueba(entorno, "persona-candidata-5", "solicitud-1006"),
	)
	if err != nil {
		t.Fatalf("IniciarInstancia() error = %v", err)
	}
	llegadas := make(chan struct{}, 2)
	continuar := make(chan struct{})
	entorno.evaluador.llegadas = llegadas
	entorno.evaluador.continuar = continuar

	errores := make(chan error, 2)
	var grupo sync.WaitGroup
	for indice := 0; indice < 2; indice++ {
		indice := indice
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			_, err := entorno.ejecucion.AplicarTransicion(context.Background(), OrdenAplicarTransicionFlujo{
				Principal:       principalFlujosPrueba("persona-candidata-5"),
				PerfilActivo:    perfilAutorizacionPrueba("candidato:5"),
				Finalidad:       finalidadEjecucionFlujoPrueba,
				InstanciaID:     instancia.ID,
				TransicionClave: "presentar",
				Motivo:          "Presentacion concurrente",
				CorrelacionRef:  fmt.Sprintf("corr-presentar-concurrente-%d", indice),
			})
			errores <- err
		}()
	}
	<-llegadas
	<-llegadas
	close(continuar)
	grupo.Wait()
	close(errores)
	correctas, conflictos := 0, 0
	for err := range errores {
		switch {
		case err == nil:
			correctas++
		case errors.Is(err, ports.ErrRevisionInstanciaFlujoConflicto):
			conflictos++
		default:
			t.Fatalf("error concurrente inesperado: %v", err)
		}
	}
	if correctas != 1 || conflictos != 1 {
		t.Fatalf("correctas=%d conflictos=%d", correctas, conflictos)
	}
	guardada, err := entorno.store.ObtenerInstanciaFlujo(context.Background(), instancia.ID)
	if err != nil || guardada.Revision != 2 || guardada.EstadoActual != "presentada" {
		t.Fatalf("instancia final = %+v, %v", guardada, err)
	}
	auditoria, _ := entorno.store.ListAudit(context.Background(), instancia.ID)
	transiciones := 0
	for _, entrada := range auditoria {
		if entrada.Action == domain.AccionInstanciaFlujoTransicionada {
			transiciones++
		}
	}
	if transiciones != 1 {
		t.Fatalf("transiciones auditadas=%d, auditoria=%+v", transiciones, auditoria)
	}
}

func TestRepositorioFlujosRechazaEvidenciaFalsaSinPersistir(t *testing.T) {
	entorno := nuevoEntornoFlujosPrueba(t)
	nueva, err := entorno.definicion.NuevaVersion(
		2,
		"tecnico-flujos-version-2",
		"bases:bolsa:2027",
		"Preparar version siguiente",
		entorno.ahora,
	)
	if err != nil {
		t.Fatalf("NuevaVersion() error = %v", err)
	}
	huella, err := nueva.HuellaSHA256()
	if err != nil {
		t.Fatalf("huella de version 2: %v", err)
	}
	traza, evento := evidenciaGobiernoFlujo(
		nueva,
		principalFlujosPrueba(nueva.CreadaPor),
		"perfil:flujos:tecnico",
		"decision-autorizacion-version-2",
		"gobierno_flujos_bolsa",
		domain.AccionDefinicionFlujoBorradorCreada,
		"",
		huella,
		"corr-version-2-falsa",
	)
	evento.ActorID = "actor-falsificado"
	if err := entorno.store.ConfirmarAltaBorradorFlujo(context.Background(), nueva, traza, evento); !errors.Is(err, domain.ErrDefinicionFlujoInvalida) {
		t.Fatalf("evidencia falsa: error = %v", err)
	}
	if _, err := entorno.store.ObtenerDefinicionFlujo(context.Background(), nueva.ID, 2); !errors.Is(err, ports.ErrDefinicionFlujoNoEncontrada) {
		t.Fatalf("la evidencia falsa persistio la version: %v", err)
	}
}

func contieneTexto(valores []string, buscado string) bool {
	for _, valor := range valores {
		if valor == buscado {
			return true
		}
	}
	return false
}
