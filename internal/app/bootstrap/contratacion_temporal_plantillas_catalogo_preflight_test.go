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
			if len(consulta.args) != 2 || consulta.args[1] != true || len(consulta.funciones) != 3 ||
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
