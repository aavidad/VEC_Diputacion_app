package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestProveedorCatalogoPlantillasCTExigeRecursosExactos(t *testing.T) {
	huella := strings.Repeat("a", 64)
	base := vecdomain.RecursoAutorizable{
		Referencia: plantillasapp.CatalogoID, ModuloID: plantillasapp.ModuloID,
		Tipo: tipoCatalogoPlantillasCT, Ambitos: map[string]string{},
		Atributos: map[string]string{"material_sha256": huella},
	}
	if !recursoCatalogoPlantillasCTValido("contratacion_temporal.plantillas_documentos.consultar", base, false) {
		t.Fatal("lectura CT131 legítima rechazada")
	}
	for nombre, mutar := range map[string]func(*vecdomain.RecursoAutorizable){
		"ambito_ajeno": func(r *vecdomain.RecursoAutorizable) {
			r.Ambitos = map[string]string{"expediente_ref": "expediente:ct:ajeno"}
		},
		"atributo_extra":  func(r *vecdomain.RecursoAutorizable) { r.Atributos["operacion"] = "editar" },
		"huella_invalida": func(r *vecdomain.RecursoAutorizable) { r.Atributos["material_sha256"] = strings.Repeat("A", 64) },
		"tipo_ajeno":      func(r *vecdomain.RecursoAutorizable) { r.Tipo = "expediente" },
	} {
		t.Run(nombre, func(t *testing.T) {
			r := base
			r.Ambitos = map[string]string{}
			r.Atributos = map[string]string{"material_sha256": huella}
			mutar(&r)
			if recursoCatalogoPlantillasCTValido("contratacion_temporal.plantillas_documentos.consultar", r, false) {
				t.Fatal("recurso ampliado aceptado")
			}
		})
	}
	indicador := base
	indicador.Atributos = map[string]string{"operacion": "editar", "estado": "publicado", "version": "1", "revision": "0"}
	if !recursoCatalogoPlantillasCTValido("contratacion_temporal.plantillas_documentos.editar", indicador, true) ||
		recursoCatalogoPlantillasCTValido("contratacion_temporal.plantillas_documentos.editar", indicador, false) {
		t.Fatal("indicador confundido con decisión de efecto")
	}
	if recursoCatalogoPlantillasCTValido("contratacion_temporal.plantillas_documentos.publicar", indicador, true) {
		t.Fatal("indicador de edición usado para publicación")
	}
	indicador.Atributos["version"] = "01"
	if recursoCatalogoPlantillasCTValido("contratacion_temporal.plantillas_documentos.editar", indicador, true) {
		t.Fatal("versión no canónica admitida")
	}
	documental := vecdomain.RecursoAutorizable{Referencia: "expediente:ct:123", ModuloID: plantillasapp.ModuloID,
		Tipo: tipoDocumentalPlantillasCT, Ambitos: map[string]string{}, Atributos: map[string]string{"material_sha256": huella}}
	if !recursoDocumentalPlantillasCTValido("contratacion_temporal.plantillas_documentos.documental_descargar", documental) {
		t.Fatal("lectura documental CT133 legítima rechazada")
	}
	documental.Ambitos["organizacion_ref"] = "organizacion:ajena"
	if recursoDocumentalPlantillasCTValido("contratacion_temporal.plantillas_documentos.documental_descargar", documental) {
		t.Fatal("ámbito extra admitido")
	}
}

func TestProveedorCatalogoPlantillasCTDeniegaSinSesionNiMotivos(t *testing.T) {
	p := &proveedorCatalogoPlantillasCT{}
	if _, err := p.ResolverContextoActor(context.Background()); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatalf("actor sin fuente = %v", err)
	}
	if _, err := nuevoProveedorCatalogoPlantillasCT(nil, nil, nil, nil,
		vecdomain.ReferenciaEntradaCatalogo{}, vecdomain.ReferenciaEntradaCatalogo{}, relojContratacionTemporalDesarrollo{}); !errors.Is(err, plantillasapp.ErrNoDisponible) {
		t.Fatalf("constructor incompleto = %v", err)
	}
}

type filaPreflightPlantillasCT struct {
	valida bool
	err    error
}

func (f filaPreflightPlantillasCT) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destinos[0].(*bool)) = f.valida
	return nil
}

type consultaPreflightPlantillasCT struct {
	sql  string
	fila filaPreflightPlantillasCT
}

func (c *consultaPreflightPlantillasCT) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	c.sql = sql
	return c.fila
}

func TestProveedorCatalogoPlantillasCTPreflightCompruebaLOGINYFunciones(t *testing.T) {
	c := &consultaPreflightPlantillasCT{fila: filaPreflightPlantillasCT{valida: true}}
	if err := comprobarPreflightCatalogoPlantillasCT(context.Background(), c); err != nil {
		t.Fatalf("preflight válido = %v", err)
	}
	for _, fragmento := range []string{
		"current_user = session_user", "rolcanlogin", "rolbypassrls", "pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')", "to_regprocedure",
		"operar_catalogo_plantillas_v1", "obtener_catalogo_plantillas_publicado_documental_v1",
		"comprobar_catalogo_plantillas_base_v1", "NOT pg_catalog.has_function_privilege",
		"provisionar_catalogo_plantillas_base_v1",
	} {
		if !strings.Contains(c.sql, fragmento) {
			t.Fatalf("preflight omite %s", fragmento)
		}
	}
	c.fila.valida = false
	if err := comprobarPreflightCatalogoPlantillasCT(context.Background(), c); !errors.Is(err, plantillasapp.ErrNoDisponible) {
		t.Fatalf("preflight denegado = %v", err)
	}
}
