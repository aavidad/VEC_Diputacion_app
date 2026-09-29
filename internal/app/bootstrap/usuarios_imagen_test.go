package bootstrap

import (
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestDescriptoresImagenUnicosYEnElGobierno(t *testing.T) {
	d := descriptoresMaterialImagenUsuariosDesarrollo()
	if len(d) != 4 {
		t.Fatalf("descriptores: %d", len(d))
	}
	todos := append(append(descriptoresMaterialPreferenciasUsuariosDesarrollo(), descriptoresMaterialCorreosUsuariosDesarrollo()...), d...)
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(todos); err != nil {
		t.Fatal("descriptores de imagen repetidos con preferencias o correos")
	}
	for i, a := range accionesImagenUsuarios {
		if !strings.Contains(d[i].Audiencia, "."+a.segmento+".interna_corporativa.") || !strings.Contains(d[i+2].Audiencia, "."+a.segmento+".externa_personal.") {
			t.Fatalf("orden de audiencias roto en %d", i)
		}
	}
	gobierno := map[string]bool{}
	for _, a := range audienciasConsumoGobiernoCTDesarrollo() {
		gobierno[a] = true
	}
	for _, a := range audienciasImagenUsuariosDesarrollo() {
		if !gobierno[a] {
			t.Fatalf("audiencia %s fuera del gobierno", a)
		}
	}
	dep := &dependenciasImagenUsuariosDesarrollo{}
	for i := range dep.materiales {
		dep.materiales[i] = &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	}
	interna, ok := dep.materialesSuperficie(core.SuperficieAutenticacionInternaCorporativaV1)
	externa, ok2 := dep.materialesSuperficie(core.SuperficieAutenticacionExternaPersonalV1)
	if !ok || !ok2 || interna[0] != dep.materiales[0] || externa[0] != dep.materiales[2] || externa[1] != dep.materiales[3] {
		t.Fatal("materiales por superficie mal repartidos")
	}
	if _, ok := dep.materialesSuperficie(core.SuperficieAutenticacionAdministracionPrivilegiadaV1); ok {
		t.Fatal("superficie privilegiada admitida")
	}
	if rutaImagenSuperficie(core.SuperficieAutenticacionInternaCorporativaV1) != usuarioshttp.RutaMiImagen ||
		rutaImagenSuperficie(core.SuperficieAutenticacionExternaPersonalV1) != usuarioshttp.RutaMiImagenAreaPersonal ||
		rutaImagenSuperficie(core.SuperficieAutenticacionAdministracionPrivilegiadaV1) != "" {
		t.Fatal("rutas de imagen por superficie incorrectas")
	}
}

func TestImagenSinActivarNoCompone(t *testing.T) {
	t.Setenv(envUsuariosImagenDesarrollo, "")
	d, err := nuevasDependenciasImagenUsuariosDesarrollo(config.Config{}, proveedoresMaterialImagenUsuarios{})
	if d != nil || err != nil {
		t.Fatal("«Mi imagen» compuesta sin pedirla")
	}
	activa, descriptores, err := seleccionImagenUsuariosDesarrollo(config.Config{}, true)
	if activa || descriptores != nil || err != nil {
		t.Fatal("audiencias de imagen publicadas sin pedirlas")
	}
	t.Setenv(envUsuariosImagenDesarrollo, "si")
	if _, err := nuevasDependenciasImagenUsuariosDesarrollo(config.Config{}, proveedoresMaterialImagenUsuarios{}); err == nil {
		t.Fatal("valor ilegible aceptado")
	}
}

func TestAutoridadImagenSinProveedorNoResuelveOrden(t *testing.T) {
	var a *autoridadPreferenciasUsuariosDesarrollo
	if _, err := a.ResolverOrdenImagen(t.Context()); err == nil {
		t.Fatal("autoridad nula resolvió orden")
	}
	if _, err := (&autoridadPreferenciasUsuariosDesarrollo{}).ResolverOrdenImagen(t.Context()); err == nil {
		t.Fatal("autoridad sin proveedor resolvió orden")
	}
}
