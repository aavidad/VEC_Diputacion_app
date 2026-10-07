package application_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	adaptador "vec-diputacion-granada/internal/modules/seleccion/adapters/bolsa"
	seleccion "vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

type evaluadorPreparacionPrueba struct {
	adaptador.CanonizadorBases
	llamadas int
	evaluar  func(context.Context, ports.MaterialBasesPropuesto) (ports.EvaluacionMaterialBases, error)
}

func (e *evaluadorPreparacionPrueba) EvaluarMaterialBases(ctx context.Context, m ports.MaterialBasesPropuesto) (ports.EvaluacionMaterialBases, error) {
	e.llamadas++
	return e.evaluar(ctx, m)
}

func TestPreparacionBasesNoEmiteMaterialTrasCancelacionOFalloEvaluador(t *testing.T) {
	material := ports.MaterialBasesPropuesto{Alcance: "preparacion_sintetica", IdentidadMaterial: "material:sintetico", VersionMaterial: 1}
	ctx, cancelar := context.WithCancel(context.Background())
	e := &evaluadorPreparacionPrueba{evaluar: func(context.Context, ports.MaterialBasesPropuesto) (ports.EvaluacionMaterialBases, error) {
		cancelar()
		return ports.EvaluacionMaterialBases{Pendientes: []ports.PendientePreparacionBases{{Campo: "titulo", Codigo: "material_ausente"}}}, nil
	}}
	r, err := seleccion.PrepararMaterialBases(ctx, material, e)
	if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(r, ports.PreparacionBases{}) || e.llamadas != 1 {
		t.Fatalf("salida tras cancelar=%+v / %v", r, err)
	}
	if _, err := seleccion.PrepararMaterialBases(ctx, material, e); !errors.Is(err, context.Canceled) || e.llamadas != 1 {
		t.Fatal("evaluacion invocada con contexto cancelado")
	}
	for _, fallo := range []error{ports.ErrMaterialBasesInvalido, errors.New("dependencia_prueba_caida")} {
		e.evaluar = func(context.Context, ports.MaterialBasesPropuesto) (ports.EvaluacionMaterialBases, error) {
			return ports.EvaluacionMaterialBases{}, fallo
		}
		r, err = seleccion.PrepararMaterialBases(context.Background(), material, e)
		esperado := ports.ErrPreparadorBasesNoDisponible
		if errors.Is(fallo, ports.ErrMaterialBasesInvalido) {
			esperado = ports.ErrMaterialBasesInvalido
		}
		if !errors.Is(err, esperado) || !reflect.DeepEqual(r, ports.PreparacionBases{}) {
			t.Fatalf("fallo cambia salida/contrato: %+v / %v", r, err)
		}
	}
}

func TestPreparacionBasesComparteEvaluacionVaciaParcialYValida(t *testing.T) {
	canonizador := adaptador.CanonizadorBases{}

	for _, caso := range []struct {
		nombre    string
		contenido bolsa.ContenidoPublicableConvocatoria
		cantidad  int
	}{
		{"vacio", bolsa.ContenidoPublicableConvocatoria{}, 23},
		{"parcial", bolsa.ContenidoPublicableConvocatoria{Titulo: "Propuesta sintética"}, 22},
		{"estructuralmente_valido", contenidoPreparacionValidoPrueba(), 14},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			material := ports.MaterialBasesPropuesto{Alcance: "preparacion_sintetica", IdentidadMaterial: "material:sintetico", VersionMaterial: 1, Contenido: caso.contenido}
			cli, err := seleccion.PrepararMaterialBases(context.Background(), material, canonizador)
			if err != nil {
				t.Fatal(err)
			}
			bolsaEvaluacion, err := prep.EvaluarMaterialBases(prep.Material{Contenido: caso.contenido})
			if err != nil {
				t.Fatal(err)
			}
			pendientes := make([]ports.PendientePreparacionBases, len(bolsaEvaluacion.Pendientes))
			for i, p := range bolsaEvaluacion.Pendientes {
				pendientes[i] = ports.PendientePreparacionBases{Campo: p.Campo, Codigo: p.Codigo}
			}
			if len(pendientes) != caso.cantidad || !reflect.DeepEqual(cli.Pendientes, pendientes) {
				t.Fatalf("politica divergente: CLI=%+v Bolsa=%+v", cli.Pendientes, pendientes)
			}
			if !reflect.DeepEqual(cli.ContenidoCanonicoBolsa, bolsaEvaluacion.ContenidoCanonico) {
				t.Fatal("contenido canonico divergente")
			}
			if cli.Estado != "pendiente" {
				t.Fatal("material promovido")
			}
			if (bolsa.ConfiguracionFijadaConvocatoria{}).ValidarPara(caso.contenido) == nil {
				t.Fatal("firma/custodia formal rebajadas")
			}
		})
	}
}

func TestPreparacionBasesEvaluaReferenciaExactaSinAprobarla(t *testing.T) {
	canonizador := adaptador.CanonizadorBases{}
	for _, ref := range []bolsa.ReferenciaConfiguracionConvocatoria{
		{ID: "baremo:sintetico", Version: 1, HuellaContenidoSHA256: strings.Repeat("a", 64)},
		{ID: "baremo:sintetico", Version: 1},
		{ID: "baremo:sintetico", Version: -1, HuellaContenidoSHA256: strings.Repeat("a", 64)},
	} {
		material := ports.MaterialBasesPropuesto{Alcance: "preparacion_sintetica", IdentidadMaterial: "material:sintetico", VersionMaterial: 1, Referencias: ports.ReferenciasPreparacionBases{ReglasBaremacion: ref}}
		cli, err := seleccion.PrepararMaterialBases(context.Background(), material, canonizador)
		if err != nil {
			t.Fatal(err)
		}
		comun, err := prep.EvaluarMaterialBases(prep.Material{Referencias: []prep.ReferenciaPropuesta{{Campo: "reglas_baremacion", Referencia: ref}}})
		if err != nil {
			t.Fatal(err)
		}
		for i, p := range comun.Pendientes {
			if cli.Pendientes[i].Campo != p.Campo || cli.Pendientes[i].Codigo != p.Codigo {
				t.Fatalf("referencia divergente: %+v / %+v", cli.Pendientes, comun.Pendientes)
			}
		}
		almacenable := prep.Material{Referencias: []prep.ReferenciaPropuesta{{Campo: "reglas_baremacion", Referencia: ref}}}
		pendientes, err := almacenable.Pendientes()
		if err != nil || !reflect.DeepEqual(pendientes, comun.Pendientes) {
			t.Fatalf("material durable evalua distinto: %+v / %v", pendientes, err)
		}
	}
}

func contenidoPreparacionValidoPrueba() bolsa.ContenidoPublicableConvocatoria {
	instante := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	return bolsa.ContenidoPublicableConvocatoria{
		IdentificadorPublico: "auxiliar-2026", Tipo: "oposicion", Titulo: "Propuesta sintética", Descripcion: "Material sintético para preparar bases.", Resumen: "Preparación de bases sintéticas.",
		CatalogoCategorias: bolsa.ReferenciaCatalogoCategorias{CatalogoID: "categorias-profesionales", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64)},
		Categorias:         []string{"auxiliar_administrativo"},
		Plazos:             []bolsa.PlazoConvocatoria{{Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripción", Descripcion: "Plazo de inscripción sintético.", AbreEn: instante, CierraEn: instante.Add(time.Hour)}},
		Documentos:         []bolsa.DocumentoPublicableConvocatoria{{Referencia: "documento:bases", Tipo: "bases", Orden: 1, Titulo: "Bases sintéticas", Descripcion: "Propuesta de bases sintéticas.", Formato: "pdf", URL: "/bolsa/documentos/bases.pdf"}},
	}
}
