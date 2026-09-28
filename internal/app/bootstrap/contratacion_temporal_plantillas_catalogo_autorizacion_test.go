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
		Tipo: tipoCatalogoPlantillasCT, Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo},
		Atributos: map[string]string{"material_sha256": huella},
	}
	if !recursoCatalogoPlantillasCTValido("contratacion_temporal.plantillas_documentos.consultar", base, false) {
		t.Fatal("lectura CT131 legítima rechazada")
	}
	for nombre, mutar := range map[string]func(*vecdomain.RecursoAutorizable){
		"ambito_ajeno": func(r *vecdomain.RecursoAutorizable) {
			r.Ambitos = map[string]string{"organizacion_ref": "organizacion:ajena"}
		},
		"atributo_extra":  func(r *vecdomain.RecursoAutorizable) { r.Atributos["operacion"] = "editar" },
		"huella_invalida": func(r *vecdomain.RecursoAutorizable) { r.Atributos["material_sha256"] = strings.Repeat("A", 64) },
		"tipo_ajeno":      func(r *vecdomain.RecursoAutorizable) { r.Tipo = "expediente" },
	} {
		t.Run(nombre, func(t *testing.T) {
			r := base
			r.Ambitos = map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo}
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
}

func TestProveedorCatalogoPlantillasCTDeniegaSinSesionNiMotivos(t *testing.T) {
	p := &proveedorCatalogoPlantillasCT{}
	if _, err := p.ResolverContextoActor(context.Background()); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatalf("actor sin fuente = %v", err)
	}
	if _, err := nuevoProveedorCatalogoPlantillasCT(nil, nil, nil,
		vecdomain.ReferenciaEntradaCatalogo{}, relojContratacionTemporalDesarrollo{}); !errors.Is(err, plantillasapp.ErrNoDisponible) {
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
	sql       string
	funciones []string
	args      []any
	fila      filaPreflightPlantillasCT
}

func (c *consultaPreflightPlantillasCT) QueryRow(_ context.Context, sql string, argumentos ...any) pgx.Row {
	c.sql = sql
	c.args = append([]any(nil), argumentos...)
	if len(argumentos) == 1 {
		c.funciones, _ = argumentos[0].([]string)
	}
	return c.fila
}

func TestProveedorCatalogoPlantillasCTPreflightCompruebaLOGINYFunciones(t *testing.T) {
	c := &consultaPreflightPlantillasCT{fila: filaPreflightPlantillasCT{valida: true}}
	if err := comprobarPreflightCatalogoPlantillasCT(context.Background(), c); err != nil {
		t.Fatalf("preflight válido = %v", err)
	}
	for _, fragmento := range []string{
		"session_user=current_user", "rolcanlogin", "rolbypassrls", "pg_auth_members", "inherit_option", "NOT m.set_option", "to_regprocedure",
		"has_schema_privilege(session_user,'vec_contratacion_temporal','USAGE')", "has_schema_privilege(g.oid,'vec_contratacion_temporal','USAGE')",
		"has_table_privilege", "has_any_column_privilege", "vec_bolsa_llamamientos", "vec_autorizacion_atestada_v3",
		"consultar_auditoria_ct_atestada_v1", "registrar_auditoria_frontera_ruta_exacta_v1", "registrar_auditoria_frontera_auditoria_v1",
		"registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada", "catalogo_plantillas_historia_v1", "organizacion_ref", "pg_get_functiondef",
		"has_schema_privilege('vec_contratacion_temporal_propietario','vec_autorizacion_atestada_v3','USAGE')",
		"has_function_privilege('vec_contratacion_temporal_propietario'",
		"NOT pg_catalog.has_function_privilege",
		"provisionar_catalogo_plantillas_base_v1",
	} {
		if !strings.Contains(c.sql, fragmento) {
			t.Fatalf("preflight omite %s", fragmento)
		}
	}
	if len(c.funciones) != 3 || !strings.Contains(strings.Join(c.funciones, " "), "obtener_catalogo_plantillas_publicado_documental_v1") {
		t.Fatalf("preflight CT131/133 no exige ambas fachadas: %v", c.funciones)
	}
	if err := comprobarPreflightCatalogoPlantillasCT(context.Background(), c, false); err != nil || len(c.funciones) != 2 ||
		strings.Contains(strings.Join(c.funciones, " "), "obtener_catalogo_plantillas_publicado_documental_v1") {
		t.Fatalf("preflight CT131 independiente de CT133: %v %v", err, c.funciones)
	}
	catalogo, err := CargarCatalogoPlantillasCT("../../../" + rutaPlantillasCTEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	if err := comprobarPreimagenCatalogoPlantillasCT(context.Background(), c, catalogo); err != nil ||
		!strings.Contains(c.sql, "comprobar_catalogo_plantillas_base_v1") || len(c.args) != 3 ||
		c.args[1] != int64(catalogo.Version) || c.args[2] != catalogo.FuenteRef {
		t.Fatalf("preimagen no cotejada con contenido canónico: %v %v", err, c.args)
	}
	c.fila.valida = false
	if err := comprobarPreimagenCatalogoPlantillasCT(context.Background(), c, catalogo); !errors.Is(err, plantillasapp.ErrNoDisponible) {
		t.Fatalf("preimagen ausente admitida: %v", err)
	}
	if err := comprobarPreflightCatalogoPlantillasCT(context.Background(), c); !errors.Is(err, plantillasapp.ErrNoDisponible) {
		t.Fatalf("preflight denegado = %v", err)
	}
}
