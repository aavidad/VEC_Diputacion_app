package application_test

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

type conectorEjecucionDocumentalAtestadaV4Prueba struct {
	resultado ports.ResultadoConectorEjecucionDocumentalAtestadaV4
	err       error
	llamadas  int
}

func (c *conectorEjecucionDocumentalAtestadaV4Prueba) EjecutarDocumentalAtestadoV4(
	context.Context,
	ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	domain.CabeceraAtestacionAutorizacionV1,
	ports.SobreCriptograficoDocumentalCrudoV4,
) (ports.ResultadoConectorEjecucionDocumentalAtestadaV4, error) {
	c.llamadas++
	return c.resultado, c.err
}

func TestEjecutorDocumentalAtestadoV4DelegaEnPuertoNeutralYRedactaFallos(t *testing.T) {
	instante := time.Date(2026, time.July, 15, 18, 30, 0, 123456000, time.UTC)
	resultado, err := ports.NuevoResultadoConectorEjecucionDocumentalAtestadaV4(
		"efecto:documental:v4:prueba", "pendiente_generacion",
		"auditoria:documental:v4:"+strings.Repeat("a", 64),
		"evento:documental:v4:"+strings.Repeat("b", 64), instante,
	)
	if err != nil {
		t.Fatalf("preparar resultado: %v", err)
	}
	conector := &conectorEjecucionDocumentalAtestadaV4Prueba{resultado: resultado}
	ejecutor, err := application.NuevoEjecutorDocumentalAtestadoV4(conector)
	if err != nil {
		t.Fatalf("construir caso de uso: %v", err)
	}
	vinculo, cabecera, sobre := entradasValidasEjecucionDocumentalAtestadaV4Prueba(t)
	confirmacion, err := ejecutor.Ejecutar(
		context.Background(), vinculo, cabecera, sobre,
	)
	if err != nil || conector.llamadas != 1 || confirmacion.OrdenRef != "efecto:documental:v4:prueba" ||
		confirmacion.Estado != "pendiente_generacion" || !confirmacion.RegistradaEn.Equal(instante) {
		t.Fatalf("delegacion neutral incorrecta: resultado=%+v err=%v llamadas=%d",
			confirmacion, err, conector.llamadas)
	}

	conector.err = errors.New("dsn=postgresql://usuario:secreto@servidor/base")
	if _, err = ejecutor.Ejecutar(
		context.Background(), vinculo, cabecera, sobre,
	); !errors.Is(err, application.ErrEjecucionDocumentalAtestadaV4NoDisponible) ||
		strings.Contains(err.Error(), "secreto") || strings.Contains(err.Error(), "postgresql") {
		t.Fatalf("el caso de uso filtro el error del conector: %v", err)
	}
}

func TestEjecutorDocumentalAtestadoV4RechazaEntradasInvalidasAntesDelConector(t *testing.T) {
	resultado, err := ports.NuevoResultadoConectorEjecucionDocumentalAtestadaV4(
		"efecto:documental:v4:prueba", "pendiente_generacion",
		"auditoria:documental:v4:"+strings.Repeat("a", 64),
		"evento:documental:v4:"+strings.Repeat("b", 64),
		time.Date(2026, time.July, 15, 18, 30, 0, 123456000, time.UTC),
	)
	if err != nil {
		t.Fatalf("preparar resultado: %v", err)
	}
	conector := &conectorEjecucionDocumentalAtestadaV4Prueba{resultado: resultado}
	ejecutor, err := application.NuevoEjecutorDocumentalAtestadoV4(conector)
	if err != nil {
		t.Fatalf("construir caso de uso: %v", err)
	}
	if _, err = ejecutor.Ejecutar(
		context.Background(), ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4{},
		domain.CabeceraAtestacionAutorizacionV1{},
		ports.SobreCriptograficoDocumentalCrudoV4{},
	); !errors.Is(err, application.ErrEjecucionDocumentalAtestadaV4NoDisponible) ||
		conector.llamadas != 0 {
		t.Fatalf("entradas cero alcanzaron el conector: err=%v llamadas=%d", err, conector.llamadas)
	}
}

func TestEjecutorDocumentalAtestadoV4RechazaContextoNuloOCancelado(t *testing.T) {
	resultado, err := ports.NuevoResultadoConectorEjecucionDocumentalAtestadaV4(
		"efecto:documental:v4:prueba", "pendiente_generacion",
		"auditoria:documental:v4:"+strings.Repeat("a", 64),
		"evento:documental:v4:"+strings.Repeat("b", 64),
		time.Date(2026, time.July, 15, 18, 30, 0, 123456000, time.UTC),
	)
	if err != nil {
		t.Fatalf("preparar resultado: %v", err)
	}
	vinculo, cabecera, sobre := entradasValidasEjecucionDocumentalAtestadaV4Prueba(t)
	for _, caso := range []struct {
		nombre string
		ctx    context.Context
	}{
		{nombre: "nulo", ctx: nil},
		{nombre: "cancelado", ctx: contextoCanceladoEjecucionDocumentalV4Prueba()},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			conector := &conectorEjecucionDocumentalAtestadaV4Prueba{resultado: resultado}
			ejecutor, err := application.NuevoEjecutorDocumentalAtestadoV4(conector)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = ejecutor.Ejecutar(caso.ctx, vinculo, cabecera, sobre); !errors.Is(
				err, application.ErrEjecucionDocumentalAtestadaV4NoDisponible,
			) || conector.llamadas != 0 {
				t.Fatalf("contexto %s alcanzo el conector: err=%v llamadas=%d", caso.nombre, err, conector.llamadas)
			}
		})
	}
}

func contextoCanceladoEjecucionDocumentalV4Prueba() context.Context {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	return ctx
}

func TestEjecutorDocumentalAtestadoV4RechazaPuertoNulo(t *testing.T) {
	var conector *conectorEjecucionDocumentalAtestadaV4Prueba
	if _, err := application.NuevoEjecutorDocumentalAtestadoV4(conector); !errors.Is(
		err, application.ErrEjecucionDocumentalAtestadaV4NoDisponible,
	) {
		t.Fatalf("puerto nulo aceptado: %v", err)
	}
	if _, err := application.NuevoEjecutorDocumentalAtestadoV4(nil); !errors.Is(
		err, application.ErrEjecucionDocumentalAtestadaV4NoDisponible,
	) {
		t.Fatalf("nil aceptado: %v", err)
	}
}

func TestApplicationNoImportaInfraestructuraEjecucionDocumentalV4(t *testing.T) {
	_, fichero, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no se pudo localizar application")
	}
	raiz := filepath.Dir(fichero)
	err := filepath.WalkDir(raiz, func(ruta string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.IsDir() || !strings.HasSuffix(entrada.Name(), ".go") ||
			strings.HasSuffix(entrada.Name(), "_test.go") {
			return nil
		}
		archivo, err := parser.ParseFile(token.NewFileSet(), ruta, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, importacion := range archivo.Imports {
			destino := strings.Trim(importacion.Path.Value, "\"")
			if strings.Contains(destino, "pgx") || strings.Contains(
				strings.ToLower(destino), "postgres",
			) || strings.Contains(destino, "/internal/vec/adapters/") {
				t.Errorf("application importa infraestructura en %s: %s", ruta, destino)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("auditar imports de application: %v", err)
	}

	// Evita que una futura fachada recupere parametros de infraestructura aunque
	// el import se oculte tras un alias o tipo local.
	conjunto := token.NewFileSet()
	archivo, err := parser.ParseFile(
		conjunto, filepath.Join(raiz, "ejecucion_documental_atestada_v4.go"), nil, 0,
	)
	if err != nil {
		t.Fatalf("analizar fachada V4: %v", err)
	}
	ast.Inspect(archivo, func(nodo ast.Node) bool {
		identificador, ok := nodo.(*ast.Ident)
		if ok && (strings.Contains(strings.ToLower(identificador.Name), "postgres") ||
			strings.Contains(strings.ToLower(identificador.Name), "pgx") ||
			strings.Contains(strings.ToLower(identificador.Name), "pool")) {
			t.Errorf("fachada V4 contiene infraestructura: %s", identificador.Name)
		}
		return true
	})
}

func entradasValidasEjecucionDocumentalAtestadaV4Prueba(
	t *testing.T,
) (
	ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	domain.CabeceraAtestacionAutorizacionV1,
	ports.SobreCriptograficoDocumentalCrudoV4,
) {
	t.Helper()
	instante := time.Date(2026, time.July, 15, 18, 0, 0, 123456000, time.UTC)
	_, vinculoAutenticacion, err := pruebasvec.NuevoContextoYVinculo(
		instante,
		"per_0123456789abcdefghijkl",
		"prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate,
		domain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatalf("crear vinculo de autenticacion: %v", err)
	}
	datosVinculo, err := vinculoAutenticacion.Datos()
	if err != nil {
		t.Fatalf("leer vinculo de autenticacion: %v", err)
	}
	recurso := domain.RecursoAutorizable{
		Referencia: "recurso:documental:v4:prueba",
		ModuloID:   "bolsa",
		Tipo:       "documento_bolsa",
		Ambitos:    map[string]string{"organizacion": "diputacion_granada"},
		Atributos: map[string]string{
			ports.AtributoAutorizacionDocumentalEfectoRef:        "efecto:documental:v4:prueba",
			ports.AtributoAutorizacionDocumentalHuellaPlanSHA256: strings.Repeat("c", 64),
		},
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("calcular huella de recurso: %v", err)
	}
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatalf("calcular huella de catalogo: %v", err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision:documental:v4:prueba", Concedida: true, Codigo: "concedida",
		PrincipalID: datosVinculo.PrincipalID, PerfilActivoRef: datosVinculo.PerfilActivoRef,
		Accion: ports.AccionEjecutarPlanDocumentalV4, RecursoRef: recurso.Referencia,
		ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo,
		ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad:                   "tramitar_bolsa", CorrelacionRef: "correlacion:documental:v4:prueba",
		VinculoAutenticacionActor: vinculoAutenticacion,
		AsignacionRef:             "asignacion:documental:v4:prueba", AsignacionHuellaSHA256: strings.Repeat("d", 64),
		VersionRolRef: "rol:documental:v4:prueba", VersionRolHuellaSHA256: strings.Repeat("e", 64),
		ControlVigenciaVersionRolRef:          "rol:documental:v4:prueba",
		ControlVigenciaVersionRolRevision:     1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("f", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasRefs: []string{}, PoliticasEvaluadasHuellasSHA256: map[string]string{},
		PoliticasRefs: []string{}, PoliticasHuellasSHA256: map[string]string{},
		GarantiaMinima:   domain.AuthAssuranceHigh,
		CamposPermitidos: []string{"documento.generado"}, Obligaciones: []string{},
		EmitidaEn: instante, ValidaHasta: instante.Add(2 * time.Minute),
	}
	if err = decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("crear decision de prueba: %v", err)
	}
	evidencia, err := ports.NuevaEvidenciaUsoDecisionAutorizacion(
		decision, instante.Add(time.Microsecond),
	)
	if err != nil {
		t.Fatalf("crear evidencia de prueba: %v", err)
	}
	expectativa := ports.ExpectativaAutorizacionEjecucionDocumentalV4{
		DecisionEsperada: decision,
		PrincipalID:      decision.PrincipalID, PerfilActivoRef: decision.PerfilActivoRef,
		AutenticacionRef: datosVinculo.AutenticacionRef, SesionRef: datosVinculo.SesionRef,
		ControlSesionRef:          datosVinculo.ControlSesionRef,
		ControlSesionRevision:     datosVinculo.ControlSesionRevision,
		ControlSesionHuellaSHA256: datosVinculo.ControlSesionHuellaSHA256,
		ContextoActorRef:          datosVinculo.ContextoActorRef,
		ContextoActorVersion:      datosVinculo.ContextoActorVersion,
		ContextoActorHuellaSHA256: datosVinculo.ContextoActorHuellaSHA256,
		Recurso:                   recurso, Finalidad: decision.Finalidad,
		CorrelacionRef:            decision.CorrelacionRef,
		EfectoRef:                 recurso.Atributos[ports.AtributoAutorizacionDocumentalEfectoRef],
		HuellaPlanSHA256:          recurso.Atributos[ports.AtributoAutorizacionDocumentalHuellaPlanSHA256],
		CamposPermitidosEsperados: []string{"documento.generado"},
		ObligacionesEsperadas:     []string{}, CumplimientosObligacionesPorRef: map[string]string{},
	}
	vinculo, err := ports.NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
		evidencia, expectativa, instante.Add(2*time.Microsecond),
	)
	if err != nil {
		t.Fatalf("crear solicitud vinculada: %v", err)
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV1{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV1,
		Suite:          "cose-sign1-eddsa-v1", ClaveID: "clave:pdp:prueba-v4",
		Audiencia: "diputacion-granada:prueba-v4",
	}
	sobre, err := ports.NuevoSobreCriptograficoDocumentalCrudoV4(
		[]byte("sobre-cose-prueba-v4"),
	)
	if err != nil {
		t.Fatalf("crear sobre de prueba: %v", err)
	}
	return vinculo, cabecera, sobre
}
