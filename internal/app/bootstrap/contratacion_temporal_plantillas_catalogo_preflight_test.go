package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
)

func TestPreflightPlantillasDocumentalCT137FallaCerrado(t *testing.T) {
	for _, tc := range []struct {
		nombre string
		fila   filaPreflightPlantillasCT
	}{
		{nombre: "fachada o ACL ausente", fila: filaPreflightPlantillasCT{valida: false}},
		{nombre: "error SQL", fila: filaPreflightPlantillasCT{err: errors.New("42501")}},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			consulta := &consultaPreflightPlantillasCT{fila: tc.fila}
			if err := comprobarPreflightCatalogoPlantillasCT(t.Context(), consulta); !errors.Is(err, plantillasapp.ErrNoDisponible) {
				t.Fatalf("preflight documental sin postimagen = %v", err)
			}
			if len(consulta.args) != 3 || consulta.args[1] != true || len(consulta.funciones) != 3 ||
				!strings.Contains(consulta.funciones[2], "_documental_ambitos_v1") {
				t.Fatalf("contrato de fachada CT137 no exigido: %v", consulta.args)
			}
		})
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := comprobarPreflightCatalogoPlantillasCT(ctx, &consultaPreflightPlantillasCT{}); !errors.Is(err, plantillasapp.ErrNoDisponible) {
		t.Fatalf("contexto cancelado admitido: %v", err)
	}
}

func TestPreflightPlantillasDocumentalCT137ContratoACL(t *testing.T) {
	consulta := &consultaPreflightPlantillasCT{fila: filaPreflightPlantillasCT{valida: true}}
	if err := comprobarPreflightCatalogoPlantillasCT(t.Context(), consulta); err != nil {
		t.Fatal(err)
	}
	for _, fragmento := range []string{
		"registrar_y_consumir_plantillas_doc_ct_ambitos_v3_atestada",
		"registrar_y_consumir_plantillas_doc_ct_org_v3_atestada",
		"obtener_catalogo_plantillas_publicado_documental_v1",
		"NOT pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')",
		"NOT pg_catalog.has_function_privilege(g.oid,p.oid,'EXECUTE')",
		"a.grantee NOT IN (p.proowner,'vec_contratacion_temporal_propietario'::pg_catalog.regrole)",
		"AND ($2::boolean IS NOT TRUE OR (",
	} {
		if !strings.Contains(consulta.sql, fragmento) {
			t.Fatalf("preflight documental omite %q", fragmento)
		}
	}
	if strings.Contains(consulta.sql, "pg_get_functiondef") ||
		strings.Contains(consulta.sql, "to_regprocedure('vec_autorizacion_atestada_v3.") {
		t.Fatal("preflight depende del cuerpo SQL o resuelve AD3 sin USAGE")
	}
}

// La lista cerrada de fachadas V3 es la misma para el montaje documental y el
// de gobierno, se pasa por copia y solo contiene las seis de categorías RPT.
func TestPreflightPlantillasCTListaCerradaFuncionesV3(t *testing.T) {
	esperadas := map[string]bool{}
	for _, nombre := range []string{"listar_categorias_habilitadas", "leer_publicacion_categoria", "consultar_uso_categoria",
		"reservar_uso_categoria", "confirmar_uso_categoria", "cancelar_uso_categoria"} {
		esperadas[nombre+"_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"] = true
	}
	if len(funcionesV3PermitidasLoginCT) != len(esperadas) {
		t.Fatalf("lista V3 ampliada o recortada: %v", funcionesV3PermitidasLoginCT)
	}
	for _, firma := range funcionesV3PermitidasLoginCT {
		if !esperadas[firma] || strings.Contains(firma, ".") || strings.Contains(firma, " ") ||
			strings.Contains(firma, "plantillas") || strings.Contains(firma, "registrar_y_consumir") {
			t.Fatalf("firma V3 fuera de la lista cerrada: %q", firma)
		}
	}
	for _, tc := range []struct {
		nombre     string
		documental []bool
	}{
		{nombre: "documental", documental: nil},
		{nombre: "gobierno", documental: []bool{false}},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			consulta := &consultaPreflightPlantillasCT{fila: filaPreflightPlantillasCT{valida: true}}
			if err := comprobarPreflightCatalogoPlantillasCT(t.Context(), consulta, tc.documental...); err != nil {
				t.Fatal(err)
			}
			permitidas, ok := consulta.args[2].([]string)
			if !ok || strings.Join(permitidas, "|") != strings.Join(funcionesV3PermitidasLoginCT, "|") {
				t.Fatalf("lista V3 distinta de la única fuente: %v", consulta.args[2])
			}
			permitidas[0] = "manipulada()"
			if funcionesV3PermitidasLoginCT[0] == "manipulada()" {
				t.Fatal("la lista V3 se comparte sin copia defensiva")
			}
			// Cada caso es una guarda ensayada en PostgreSQL 18.4 real (PR #956):
			// si el fragmento falta, el caso correspondiente dejaría de fallar.
			for _, guarda := range []struct{ caso, fragmento string }{
				{"función V3 ejecutable fuera de la lista (nueva, otra o a PUBLIC)",
					"WHERE n.nspname='vec_autorizacion_atestada_v3'\n        AND (pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE WITH GRANT OPTION')\n          OR (pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')"},
				{"opción de concesión sobre cualquier función V3, también las seis",
					"has_function_privilege(session_user,p.oid,'EXECUTE WITH GRANT OPTION')"},
				{"PROCEDURE con la firma de una permitida", "WHERE p.prokind='f'"},
				{"sobrecarga o mismo nombre con otros tipos", "AND p.proname=pg_catalog.split_part(l.firma,'(',1)"},
				{"tipos fijados en pg_catalog sin search_path", "pg_catalog.to_regtype('pg_catalog.'||t.tipo)::pg_catalog.oid"},
				{"firma comparada como oidvector", "AND p.proargtypes=(\n                  SELECT pg_catalog.array_to_string(x.tipos,' ')::pg_catalog.oidvector"},
				{"tipo de la lista no resuelto", "WHERE pg_catalog.array_position(x.tipos,NULL) IS NULL"},
				{"privilegio por defecto al LOGIN, al grupo CT o a PUBLIC", "AND (a.grantee=0 OR pg_catalog.pg_has_role(session_user,a.grantee,'USAGE'))"},
				{"privilegio por defecto en el esquema V3 o global de quien crea en él",
					"WHERE (d.defaclnamespace=n.oid\n          OR (d.defaclnamespace=0 AND pg_catalog.has_schema_privilege(d.defaclrole,n.oid,'CREATE')))"},
				{"propietario V3 sin fila global de funciones (EXECUTE a PUBLIC de fábrica)",
					"WHERE d.defaclrole=n.nspowner AND d.defaclnamespace=0 AND d.defaclobjtype='f')"},
				{"CREATE en el esquema V3", "to_regnamespace('vec_autorizacion_atestada_v3'),'CREATE'"},
				{"privilegio en una secuencia V3", "pg_catalog.has_sequence_privilege(session_user,c.oid,'USAGE,SELECT,UPDATE')"},
				{"privilegio en una tabla o vista V3", "WHEN c.relkind IN ('r','p','v','m','f') THEN"},
				{"consumidor V3 de catálogo de plantillas", "registrar_y_consumir_catalogo_plantillas_ct_org_v3_atestada"},
				{"consumidor V3 documental de plantillas", "registrar_y_consumir_plantillas_doc_ct_ambitos_v3_atestada"},
			} {
				if !strings.Contains(consulta.sql, guarda.fragmento) {
					t.Fatalf("preflight no cubre %q: falta %q", guarda.caso, guarda.fragmento)
				}
			}
			for _, prohibido := range []string{
				"to_regnamespace('vec_autorizacion_atestada_v3'),'USAGE'", // el USAGE legítimo de AD3-117/177
				"oidvectortypes", // firma como texto, dependiente del search_path
			} {
				if strings.Contains(consulta.sql, prohibido) {
					t.Fatalf("preflight conserva %q", prohibido)
				}
			}
		})
	}
}
