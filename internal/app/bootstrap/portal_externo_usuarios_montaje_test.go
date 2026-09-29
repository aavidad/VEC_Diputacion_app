package bootstrap

import (
	"testing"
	"vec-diputacion-granada/config"
)

func TestFronteraUsuariosExternaNoAdmiteRolesInternos(t *testing.T) {
	frontera := fronteraPreferenciasUsuariosPortalExterno(topologiaPostgreSQLPreferenciasUsuarios{})
	for _, rol := range []string{"vec_contexto_actor_v1_runtime", "vec_autorizacion_fuente", "vec_autorizacion_registro", "vec_autorizacion_motivos_evaluador", "vec_contexto_actor_v1_candidato_externo", ""} {
		if pool, _, err := frontera.abrirPool(t.Context(), "postgres://localhost/vec", rol); err == nil || pool != nil {
			t.Fatalf("se admitió el rol de otra autoridad %q", rol)
		}
	}
}

func TestFronteraUsuariosSinDependenciasDeniegaAntesDeLeerMaterial(t *testing.T) {
	if _, err := nuevaRutaUsuariosPreferenciasConFrontera(config.Config{}, nil, nil, nil, topologiaPostgreSQLPreferenciasUsuarios{}, "", "", nil, nil, nil, nil, fronteraPreferenciasUsuarios{}); err == nil {
		t.Fatal("montaje incompleto aceptado")
	}
}
