package memory

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
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
		CreadoPor: "tecnico-configuracion-1",
		CreadoEn:  fecha,
	}
}

func evidenciaCatalogoMemoriaPrueba(catalogo domain.CatalogoConfigurable, accion, antes string) (domain.AuditEntry, domain.Event) {
	huella, _ := catalogo.HuellaSHA256()
	actor, fecha, regla, motivo := catalogo.CreadoPor, catalogo.CreadoEn, catalogo.FuenteRef, catalogo.MotivoCreacion
	if accion == domain.AccionCatalogoBorradorActualizado {
		actor, fecha, motivo = catalogo.UltimaModificacionPor, catalogo.UltimaModificacionEn, catalogo.MotivoModificacion
	}
	traza := domain.AuditEntry{
		ActorID:          actor,
		ActorProfile:     "perfil-configuracion",
		AuthMethod:       domain.AuthMethodCertificate,
		AuthAssurance:    domain.AuthAssuranceHigh,
		AuthorizationRef: "decision-interna",
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
	return traza, evento
}

func TestRepositorioCatalogosRechazaEvidenciaFalsaSinPersistir(t *testing.T) {
	store := NewStore()
	catalogo := catalogoMemoriaPrueba()
	traza, evento := evidenciaCatalogoMemoriaPrueba(catalogo, domain.AccionCatalogoBorradorCreado, "")
	evento.ActorID = "actor-distinto"
	if err := store.ConfirmarAltaBorradorCatalogo(context.Background(), catalogo, traza, evento); !errors.Is(err, domain.ErrCatalogoConfigurableInvalido) {
		t.Fatalf("evidencia falsa: error = %v", err)
	}
	if _, err := store.ObtenerCatalogo(context.Background(), catalogo.ID, catalogo.Version); !errors.Is(err, ports.ErrCatalogoNoEncontrado) {
		t.Fatalf("la evidencia falsa persistio el catalogo: %v", err)
	}
}

func TestRepositorioCatalogosSoloAceptaUnaActualizacionConcurrentePorRevision(t *testing.T) {
	store := NewStore()
	catalogo := catalogoMemoriaPrueba()
	trazaAlta, eventoAlta := evidenciaCatalogoMemoriaPrueba(catalogo, domain.AccionCatalogoBorradorCreado, "")
	if err := store.ConfirmarAltaBorradorCatalogo(context.Background(), catalogo, trazaAlta, eventoAlta); err != nil {
		t.Fatalf("alta: %v", err)
	}
	huella, _ := catalogo.HuellaSHA256()
	primera, err := catalogo.ActualizarBorrador(1, "tecnico-configuracion-2", catalogo.Nombre,
		"Primera propuesta", catalogo.FuenteRef, "Primera edicion concurrente", catalogo.Entradas, catalogo.CreadoEn.Add(time.Minute))
	if err != nil {
		t.Fatalf("primera propuesta: %v", err)
	}
	segunda, err := catalogo.ActualizarBorrador(1, "tecnico-configuracion-3", catalogo.Nombre,
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
			traza, evento := evidenciaCatalogoMemoriaPrueba(propuesta, domain.AccionCatalogoBorradorActualizado, huella)
			errores <- store.ConfirmarActualizacionBorradorCatalogo(context.Background(), huella, propuesta, traza, evento)
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
