package memory

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

const (
	personaCatalogoMemoriaUno = "per_0123456789abcdefghijkl"
	personaCatalogoMemoriaDos = "per_abcdefghijkl0123456789"
	perfilCatalogoMemoria     = "prf_0123456789abcdefghijkl"
)

func catalogoMemoriaPrueba() domain.CatalogoConfigurable {
	fecha := time.Date(2026, time.July, 14, 21, 0, 0, 0, time.UTC)
	return domain.CatalogoConfigurable{
		ID:             "bolsa.tipos_merito",
		Version:        1,
		Revision:       1,
		ModuloID:       "bolsa",
		Nombre:         "Tipos de merito",
		FuenteRef:      "bases-convocatoria:2026-1",
		MotivoCreacion: "Configurar los tipos previstos en las bases",
		Entradas: []domain.EntradaCatalogoConfigurable{
			{Clave: "experiencia", Etiqueta: "Experiencia", Orden: 10, VigenteDesde: fecha},
		},
		Estado:    domain.EstadoCatalogoBorrador,
		CreadoPor: personaCatalogoMemoriaUno,
		CreadoEn:  fecha,
	}
}

func evidenciaCatalogoMemoriaPrueba(
	t *testing.T,
	catalogo domain.CatalogoConfigurable,
	accion, antes string,
) (domain.AuditEntry, domain.Event, ports.EvidenciaUsoDecisionAutorizacion) {
	t.Helper()
	huella, _ := catalogo.HuellaSHA256()
	actor, fecha, regla, motivo := catalogo.CreadoPor, catalogo.CreadoEn, catalogo.FuenteRef, catalogo.MotivoCreacion
	if accion == domain.AccionCatalogoBorradorActualizado {
		actor, fecha, motivo = catalogo.UltimaModificacionPor, catalogo.UltimaModificacionEn, catalogo.MotivoModificacion
	}
	traza := domain.AuditEntry{
		ActorID:          actor,
		ActorProfile:     perfilCatalogoMemoria,
		AuthMethod:       domain.AuthMethodCertificate,
		AuthAssurance:    domain.AuthAssuranceHigh,
		AuthorizationRef: fmt.Sprintf("decision:catalogo:%s:%d:%s", accion, catalogo.Revision, actor),
		Purpose:          "gobierno_configuracion",
		Action:           accion,
		ModuleID:         catalogo.ModuloID,
		SubjectRef:       catalogo.Referencia(),
		ObjectVersion:    catalogo.Version,
		RuleRef:          regla,
		Reason:           motivo,
		Result:           "correcto",
		BeforeHash:       antes,
		AfterHash:        huella,
		CorrelationRef:   "corr-catalogo-memoria",
		OccurredAt:       fecha,
		Metadata:         map[string]string{"revision": strconv.Itoa(catalogo.Revision)},
	}
	evento := domain.Event{
		Type:       accion,
		ModuleID:   catalogo.ModuloID,
		SubjectRef: catalogo.Referencia(),
		ActorID:    actor,
		OccurredAt: fecha,
		Payload: map[string]string{
			"catalogo_id":       catalogo.ID,
			"catalogo_version":  strconv.Itoa(catalogo.Version),
			"catalogo_revision": strconv.Itoa(catalogo.Revision),
			"estado":            string(catalogo.Estado),
			"huella_sha256":     huella,
		},
	}
	accionAutorizacion, valida := accionAutorizacionCatalogo(accion)
	if !valida {
		t.Fatalf("accion sin autorizacion: %s", accion)
	}
	recurso := domain.RecursoAutorizable{
		Referencia: catalogo.Referencia(), ModuloID: catalogo.ModuloID, Tipo: "catalogo_configurable",
		Atributos: map[string]string{"estado": string(catalogo.Estado), "revision": strconv.Itoa(catalogo.Revision)},
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	_, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		fecha, actor, perfilCatalogoMemoria, domain.AuthMethodCertificate, domain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogoPoliticas, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: traza.AuthorizationRef, Concedida: true, Codigo: "concedida",
		PrincipalID: actor, PerfilActivoRef: perfilCatalogoMemoria, Accion: accionAutorizacion,
		RecursoRef: recurso.Referencia, ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo,
		ContextoRecursoHuellaSHA256: huellaContexto, Finalidad: traza.Purpose, CorrelacionRef: traza.CorrelationRef,
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             "asignacion:catalogo:v1", AsignacionHuellaSHA256: strings.Repeat("1", 64),
		VersionRolRef: "rol:catalogo:v1", VersionRolHuellaSHA256: strings.Repeat("2", 64),
		ControlVigenciaVersionRolRef: "rol:catalogo:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("3", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogoPoliticas,
		PoliticasEvaluadasHuellasSHA256: map[string]string{}, GarantiaMinima: domain.AuthAssuranceHigh,
		EmitidaEn: fecha, ValidaHasta: fecha.Add(5 * time.Minute),
	}
	evidencia, err := ports.NuevaEvidenciaUsoDecisionAutorizacion(decision, fecha)
	if err != nil {
		t.Fatalf("crear evidencia de autorizacion: %v", err)
	}
	return traza, evento, evidencia
}

func TestRepositorioCatalogosRechazaEvidenciaFalsaSinPersistir(t *testing.T) {
	store := NewStore()
	catalogo := catalogoMemoriaPrueba()
	traza, evento, evidencia := evidenciaCatalogoMemoriaPrueba(t, catalogo, domain.AccionCatalogoBorradorCreado, "")
	evento.ActorID = "actor-distinto"
	if err := store.ConfirmarAltaBorradorCatalogo(context.Background(), catalogo, traza, evento, evidencia); !errors.Is(err, domain.ErrCatalogoConfigurableInvalido) {
		t.Fatalf("evidencia falsa: error = %v", err)
	}
	if _, err := store.ObtenerCatalogo(context.Background(), catalogo.ID, catalogo.Version); !errors.Is(err, ports.ErrCatalogoNoEncontrado) {
		t.Fatalf("la evidencia falsa persistio el catalogo: %v", err)
	}
}

func TestRepositorioCatalogosSoloAceptaUnaActualizacionConcurrentePorRevision(t *testing.T) {
	store := NewStore()
	catalogo := catalogoMemoriaPrueba()
	trazaAlta, eventoAlta, evidenciaAlta := evidenciaCatalogoMemoriaPrueba(t, catalogo, domain.AccionCatalogoBorradorCreado, "")
	if err := store.ConfirmarAltaBorradorCatalogo(context.Background(), catalogo, trazaAlta, eventoAlta, evidenciaAlta); err != nil {
		t.Fatalf("alta: %v", err)
	}
	huella, _ := catalogo.HuellaSHA256()
	primera, err := catalogo.ActualizarBorrador(1, personaCatalogoMemoriaUno, catalogo.Nombre,
		"Primera propuesta", catalogo.FuenteRef, "Primera edicion concurrente", catalogo.Entradas, catalogo.CreadoEn.Add(time.Minute))
	if err != nil {
		t.Fatalf("primera propuesta: %v", err)
	}
	segunda, err := catalogo.ActualizarBorrador(1, personaCatalogoMemoriaDos, catalogo.Nombre,
		"Segunda propuesta", catalogo.FuenteRef, "Segunda edicion concurrente", catalogo.Entradas, catalogo.CreadoEn.Add(time.Minute))
	if err != nil {
		t.Fatalf("segunda propuesta: %v", err)
	}

	propuestas := []domain.CatalogoConfigurable{primera, segunda}
	errores := make(chan error, len(propuestas))
	var grupo sync.WaitGroup
	for _, propuesta := range propuestas {
		propuesta := propuesta
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			traza, evento, evidencia := evidenciaCatalogoMemoriaPrueba(t, propuesta, domain.AccionCatalogoBorradorActualizado, huella)
			errores <- store.ConfirmarActualizacionBorradorCatalogo(context.Background(), huella, propuesta, traza, evento, evidencia)
		}()
	}
	grupo.Wait()
	close(errores)
	correctas, conflictos := 0, 0
	for err := range errores {
		switch {
		case err == nil:
			correctas++
		case errors.Is(err, ports.ErrRevisionCatalogoEnConflicto):
			conflictos++
		default:
			t.Fatalf("error inesperado: %v", err)
		}
	}
	if correctas != 1 || conflictos != 1 {
		t.Fatalf("correctas=%d conflictos=%d", correctas, conflictos)
	}
	guardado, err := store.ObtenerCatalogo(context.Background(), catalogo.ID, 1)
	if err != nil || guardado.Revision != 2 {
		t.Fatalf("catalogo final = %+v, %v", guardado, err)
	}
	auditoria, _ := store.ListAudit(context.Background(), catalogo.Referencia())
	if len(auditoria) != 2 {
		t.Fatalf("la concurrencia duplico la auditoria: %+v", auditoria)
	}
}

func TestRepositorioCatalogosConsumeDecisionConElEfectoExactoEIdempotente(t *testing.T) {
	store := NewStore()
	catalogo := catalogoMemoriaPrueba()
	traza, evento, evidencia := evidenciaCatalogoMemoriaPrueba(t, catalogo, domain.AccionCatalogoBorradorCreado, "")
	if err := store.ConfirmarAltaBorradorCatalogo(context.Background(), catalogo, traza, evento, evidencia); err != nil {
		t.Fatalf("confirmar efecto inicial: %v", err)
	}
	if err := store.ConfirmarAltaBorradorCatalogo(context.Background(), catalogo, traza, evento, evidencia); err != nil {
		t.Fatalf("reintento exacto no fue idempotente: %v", err)
	}
	auditoria, err := store.ListAudit(context.Background(), catalogo.Referencia())
	if err != nil || len(auditoria) != 1 {
		t.Fatalf("el reintento duplico evidencia: %#v, %v", auditoria, err)
	}

	otro := catalogoMemoriaPrueba()
	otro.ID = "bolsa.tipos_merito_alternativo"
	trazaOtra, eventoOtro, _ := evidenciaCatalogoMemoriaPrueba(t, otro, domain.AccionCatalogoBorradorCreado, "")
	trazaOtra.AuthorizationRef = traza.AuthorizationRef
	if err := store.ConfirmarAltaBorradorCatalogo(
		context.Background(), otro, trazaOtra, eventoOtro, evidencia,
	); !errors.Is(err, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("una decision se reutilizo para otro efecto: %v", err)
	}
	if _, err := store.ObtenerCatalogo(context.Background(), otro.ID, otro.Version); !errors.Is(err, ports.ErrCatalogoNoEncontrado) {
		t.Fatalf("la reutilizacion dejo un efecto parcial: %v", err)
	}

	tercero := catalogoMemoriaPrueba()
	tercero.ID = "bolsa.tipos_merito_sin_evidencia"
	trazaTercero, eventoTercero, _ := evidenciaCatalogoMemoriaPrueba(t, tercero, domain.AccionCatalogoBorradorCreado, "")
	if err := store.ConfirmarAltaBorradorCatalogo(
		context.Background(), tercero, trazaTercero, eventoTercero, ports.EvidenciaUsoDecisionAutorizacion{},
	); !errors.Is(err, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("un valor cero autorizo el efecto: %v", err)
	}
}
