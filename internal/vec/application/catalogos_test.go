package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

var instanteCatalogosPrueba = time.Date(2026, time.July, 14, 14, 0, 0, 0, time.UTC)

type relojCatalogosFijo struct{ ahora time.Time }

func (r relojCatalogosFijo) Ahora() time.Time { return r.ahora }

type autorizadorCatalogosPrueba struct {
	ahora     time.Time
	siguiente int
}

func (a *autorizadorCatalogosPrueba) Exigir(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacion,
) (domain.DecisionAutorizacion, error) {
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	if err := solicitud.ValidarVinculoAutenticacionActor(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if solicitud.Principal.ID == personaAutorizacionPrueba("sin-autorizacion") {
		return domain.DecisionAutorizacion{}, domain.ErrAutorizacionDenegada
	}
	huellaContexto, err := solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	a.siguiente++
	return domain.DecisionAutorizacion{
		DecisionRef: fmt.Sprintf("decision:catalogos:%03d", a.siguiente), Concedida: true, Codigo: "concedida",
		PrincipalID: solicitud.Principal.ID, PerfilActivoRef: solicitud.PerfilActivoRef,
		Accion: solicitud.Accion, RecursoRef: solicitud.Recurso.Referencia,
		ModuloID: solicitud.Recurso.ModuloID, TipoRecurso: solicitud.Recurso.Tipo,
		ContextoRecursoHuellaSHA256: huellaContexto, Finalidad: solicitud.Finalidad,
		CorrelacionRef: solicitud.CorrelacionRef, VinculoAutenticacionActor: solicitud.VinculoAutenticacionActor,
		AsignacionRef: "asignacion:catalogos:v1", AsignacionHuellaSHA256: strings.Repeat("1", 64),
		VersionRolRef: "rol:catalogos:v1", VersionRolHuellaSHA256: strings.Repeat("2", 64),
		ControlVigenciaVersionRolRef: "rol:catalogos:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("3", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{}, GarantiaMinima: domain.AuthAssuranceHigh,
		EmitidaEn: a.ahora, ValidaHasta: a.ahora.Add(5 * time.Minute),
	}, nil
}

func nuevoServicioCatalogosPrueba(t *testing.T) (*ServicioCatalogos, *memory.Store) {
	t.Helper()
	store := memory.NewStore()
	servicio, err := NuevoServicioCatalogos(
		store,
		store,
		&autorizadorCatalogosPrueba{ahora: instanteCatalogosPrueba},
		relojCatalogosFijo{ahora: instanteCatalogosPrueba},
	)
	if err != nil {
		t.Fatalf("NuevoServicioCatalogos() error = %v", err)
	}
	return servicio, store
}

func credencialesCatalogosPrueba(t *testing.T, semilla string) CredencialesGobiernoCatalogo {
	return credencialesCatalogosPruebaEn(t, semilla, instanteCatalogosPrueba)
}

func credencialesCatalogosPruebaEn(t *testing.T, semilla string, instante time.Time) CredencialesGobiernoCatalogo {
	t.Helper()
	actor, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		instante,
		personaAutorizacionPrueba(semilla),
		perfilAutorizacionPrueba(semilla),
		domain.AuthMethodCertificate,
		domain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatalf("crear credenciales de catalogo: %v", err)
	}
	return CredencialesGobiernoCatalogo{ContextoActor: actor, VinculoAutenticacionActor: vinculo}
}

func entradasCatalogoPrueba(fecha time.Time) []domain.EntradaCatalogoConfigurable {
	return []domain.EntradaCatalogoConfigurable{
		{Clave: "disponible", Etiqueta: "Disponible", Orden: 10, VigenteDesde: fecha, Atributos: map[string]string{"publicable": "si"}},
		{Clave: "trabajando", Etiqueta: "Trabajando", Orden: 20, VigenteDesde: fecha, Atributos: map[string]string{"publicable": "si"}},
	}
}

func ordenCrearCatalogoPrueba(t *testing.T) OrdenCrearBorradorCatalogo {
	t.Helper()
	fecha := time.Date(2026, time.July, 14, 13, 0, 0, 0, time.UTC)
	return OrdenCrearBorradorCatalogo{
		Credenciales:   credencialesCatalogosPrueba(t, "tecnico-configuracion-1"),
		Finalidad:      "gobierno_configuracion_bolsa",
		ID:             "bolsa.estados_participacion",
		Version:        1,
		ModuloID:       "bolsa",
		Nombre:         "Estados de participacion en bolsa",
		Descripcion:    "Catalogo administrable por Seleccion Externa.",
		FuenteRef:      "reglamento-bolsas:2026",
		Entradas:       entradasCatalogoPrueba(fecha),
		Motivo:         "Configuracion inicial solicitada por Recursos Humanos",
		CorrelacionRef: "corr-catalogo-alta-1",
	}
}

func TestServicioCatalogosGobiernaCicloCompletoConTrazabilidad(t *testing.T) {
	servicio, store := nuevoServicioCatalogosPrueba(t)
	borrador, err := servicio.CrearBorrador(context.Background(), ordenCrearCatalogoPrueba(t))
	if err != nil {
		t.Fatalf("CrearBorrador() error = %v", err)
	}
	if borrador.Revision != 1 || borrador.Estado != domain.EstadoCatalogoBorrador {
		t.Fatalf("borrador inesperado: %+v", borrador)
	}

	entradas := append([]domain.EntradaCatalogoConfigurable(nil), borrador.Entradas...)
	entradas = append(entradas, domain.EntradaCatalogoConfigurable{
		Clave:        "disponible_desde_fecha",
		Etiqueta:     "Disponible desde una fecha",
		Orden:        30,
		VigenteDesde: borrador.CreadoEn,
	})
	actualizado, err := servicio.ActualizarBorrador(context.Background(), OrdenActualizarBorradorCatalogo{
		Credenciales:     credencialesCatalogosPrueba(t, "tecnico-configuracion-2"),
		Finalidad:        "gobierno_configuracion_bolsa",
		ID:               borrador.ID,
		Version:          borrador.Version,
		RevisionEsperada: borrador.Revision,
		Nombre:           borrador.Nombre,
		Descripcion:      "Catalogo revisado con vigencias temporales.",
		FuenteRef:        "reglamento-bolsas:2026",
		Entradas:         entradas,
		Motivo:           "Incorporar indisponibilidad programada",
		CorrelacionRef:   "corr-catalogo-actualizar-1",
	})
	if err != nil {
		t.Fatalf("ActualizarBorrador() error = %v", err)
	}
	if actualizado.Revision != 2 || len(actualizado.Entradas) != 3 {
		t.Fatalf("actualizacion inesperada: %+v", actualizado)
	}

	publicado, err := servicio.Publicar(context.Background(), OrdenPublicarCatalogo{
		Credenciales:   credencialesCatalogosPrueba(t, "responsable-configuracion-3"),
		Finalidad:      "gobierno_configuracion_bolsa",
		ID:             actualizado.ID,
		Version:        actualizado.Version,
		AprobacionRef:  "aprobacion-catalogo-1",
		Motivo:         "Catalogo revisado y aprobado por responsable competente",
		CorrelacionRef: "corr-catalogo-publicar-1",
	})
	if err != nil {
		t.Fatalf("Publicar() error = %v", err)
	}
	if _, err := publicado.ObtenerEntradaVigente("disponible_desde_fecha", publicado.PublicadoEn); err != nil {
		t.Fatalf("la opcion nueva no quedo disponible: %v", err)
	}

	retirado, err := servicio.Retirar(context.Background(), OrdenRetirarCatalogo{
		Credenciales:   credencialesCatalogosPrueba(t, "responsable-configuracion-4"),
		Finalidad:      "gobierno_configuracion_bolsa",
		ID:             publicado.ID,
		Version:        publicado.Version,
		AprobacionRef:  "retirada-catalogo-1",
		Motivo:         "Sustituido por una version normativa posterior",
		CorrelacionRef: "corr-catalogo-retirar-1",
	})
	if err != nil || retirado.Estado != domain.EstadoCatalogoRetirado {
		t.Fatalf("Retirar() = %+v, %v", retirado, err)
	}

	auditoria, err := store.ListAudit(context.Background(), borrador.Referencia())
	if err != nil || len(auditoria) != 4 {
		t.Fatalf("auditoria = %+v, %v", auditoria, err)
	}
	for indice := range auditoria {
		if auditoria[indice].AuthorizationRef == "" || auditoria[indice].Signature == "" {
			t.Fatalf("evidencia incompleta en posicion %d: %+v", indice, auditoria[indice])
		}
		if indice > 0 && auditoria[indice].BeforeHash != auditoria[indice-1].AfterHash {
			t.Fatalf("historia de huellas rota en posicion %d: %+v", indice, auditoria)
		}
	}
	eventos, err := store.ListEvents(context.Background(), []string{
		domain.AccionCatalogoBorradorCreado,
		domain.AccionCatalogoBorradorActualizado,
		domain.AccionCatalogoPublicado,
		domain.AccionCatalogoRetirado,
	})
	if err != nil || len(eventos) != 4 {
		t.Fatalf("eventos = %+v, %v", eventos, err)
	}
	for indice, evento := range eventos {
		if evento.Payload["auditoria_ref"] != auditoria[indice].ID {
			t.Fatalf("evento %d no enlazado: %+v", indice, evento)
		}
	}
}

func TestServicioCatalogosCreaNuevaVersionSinAlterarLaPublicada(t *testing.T) {
	servicio, store := nuevoServicioCatalogosPrueba(t)
	borrador, err := servicio.CrearBorrador(context.Background(), ordenCrearCatalogoPrueba(t))
	if err != nil {
		t.Fatalf("crear v1: %v", err)
	}
	publicado, err := servicio.Publicar(context.Background(), OrdenPublicarCatalogo{
		Credenciales:   credencialesCatalogosPrueba(t, "responsable-configuracion-2"),
		Finalidad:      "gobierno_configuracion_bolsa",
		ID:             borrador.ID,
		Version:        1,
		AprobacionRef:  "aprobacion-v1",
		Motivo:         "Publicacion de version inicial",
		CorrelacionRef: "corr-publicar-v1",
	})
	if err != nil {
		t.Fatalf("publicar v1: %v", err)
	}
	ordenV2 := ordenCrearCatalogoPrueba(t)
	ordenV2.Credenciales = credencialesCatalogosPrueba(t, "tecnico-configuracion-3")
	ordenV2.Version = 2
	ordenV2.FuenteRef = "resolucion-bolsas:2026-99"
	ordenV2.Motivo = "Incorporar nueva opcion aprobada"
	ordenV2.CorrelacionRef = "corr-crear-v2"
	ordenV2.Entradas = append(ordenV2.Entradas, domain.EntradaCatalogoConfigurable{
		Clave:        "cosa_cuatro",
		Etiqueta:     "Cosa cuatro",
		Orden:        40,
		VigenteDesde: publicado.PublicadoEn,
	})
	v2, err := servicio.CrearBorrador(context.Background(), ordenV2)
	if err != nil {
		t.Fatalf("crear v2: %v", err)
	}
	v1Guardada, err := store.ObtenerCatalogo(context.Background(), publicado.ID, 1)
	if err != nil || len(v1Guardada.Entradas) != 2 || len(v2.Entradas) != 3 || v2.VersionAnteriorRef != publicado.Referencia() {
		t.Fatalf("versiones incorrectas: v1=%+v v2=%+v error=%v", v1Guardada, v2, err)
	}
}

func TestServicioCatalogosDeniegaActorSinConcesionCruceDeVinculoYRevisionObsoleta(t *testing.T) {
	servicio, store := nuevoServicioCatalogosPrueba(t)
	orden := ordenCrearCatalogoPrueba(t)
	orden.Credenciales = credencialesCatalogosPrueba(t, "sin-autorizacion")
	if _, err := servicio.CrearBorrador(context.Background(), orden); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("permisos declarados: error = %v", err)
	}
	if _, err := store.ObtenerCatalogo(context.Background(), orden.ID, 1); !errors.Is(err, ports.ErrCatalogoNoEncontrado) {
		t.Fatalf("una denegacion persistio el catalogo: %v", err)
	}
	orden.Credenciales.VinculoAutenticacionActor = credencialesCatalogosPrueba(t, "actor-ajeno").VinculoAutenticacionActor
	if _, err := servicio.CrearBorrador(context.Background(), orden); !errors.Is(err, domain.ErrVinculoAutenticacionActorInvalido) {
		t.Fatalf("un vinculo de otro actor alcanzo el PDP: %v", err)
	}

	borrador, err := servicio.CrearBorrador(context.Background(), ordenCrearCatalogoPrueba(t))
	if err != nil {
		t.Fatalf("crear borrador: %v", err)
	}
	_, err = servicio.ActualizarBorrador(context.Background(), OrdenActualizarBorradorCatalogo{
		Credenciales:     credencialesCatalogosPrueba(t, "tecnico-configuracion-2"),
		Finalidad:        "gobierno_configuracion_bolsa",
		ID:               borrador.ID,
		Version:          borrador.Version,
		RevisionEsperada: 99,
		Nombre:           borrador.Nombre,
		Descripcion:      borrador.Descripcion,
		FuenteRef:        borrador.FuenteRef,
		Entradas:         borrador.Entradas,
		Motivo:           "Intento con revision obsoleta",
		CorrelacionRef:   "corr-revision-obsoleta",
	})
	if !errors.Is(err, domain.ErrTransicionCatalogoInvalida) {
		t.Fatalf("revision obsoleta: error = %v", err)
	}
}

func TestOrdenesCatalogoNoSeSerializanComoDTO(t *testing.T) {
	ordenes := []any{
		ordenCrearCatalogoPrueba(t),
		OrdenActualizarBorradorCatalogo{},
		OrdenPublicarCatalogo{},
		OrdenRetirarCatalogo{},
	}
	for _, orden := range ordenes {
		if _, err := json.Marshal(orden); !errors.Is(err, ErrSerializacionOrdenCatalogo) {
			t.Fatalf("se serializo una capacidad interna %T: %v", orden, err)
		}
	}
	var destino OrdenCrearBorradorCatalogo
	if err := json.Unmarshal([]byte(`{"id":"inyectado"}`), &destino); !errors.Is(err, ErrSerializacionOrdenCatalogo) {
		t.Fatalf("se reconstruyo una orden desde JSON: %v", err)
	}
}

func TestServicioCatalogosFallaCerradoSinDependenciasOConContextoCancelado(t *testing.T) {
	var storeNulo *memory.Store
	if _, err := NuevoServicioCatalogos(
		storeNulo,
		storeNulo,
		&autorizadorCatalogosPrueba{ahora: instanteCatalogosPrueba},
		relojCatalogosFijo{ahora: instanteCatalogosPrueba},
	); !errors.Is(err, ErrDependenciaCatalogosRequerida) {
		t.Fatalf("dependencia tipada nula aceptada: %v", err)
	}
	var servicioNulo *ServicioCatalogos
	if _, err := servicioNulo.CrearBorrador(context.Background(), ordenCrearCatalogoPrueba(t)); !errors.Is(err, ErrDependenciaCatalogosRequerida) {
		t.Fatalf("servicio nulo aceptado: %v", err)
	}
	servicio, store := nuevoServicioCatalogosPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	orden := ordenCrearCatalogoPrueba(t)
	if _, err := servicio.CrearBorrador(ctx, orden); !errors.Is(err, context.Canceled) {
		t.Fatalf("contexto cancelado no se propago: %v", err)
	}
	if _, err := store.ObtenerCatalogo(context.Background(), orden.ID, orden.Version); !errors.Is(err, ports.ErrCatalogoNoEncontrado) {
		t.Fatalf("el contexto cancelado dejo un efecto: %v", err)
	}
}
