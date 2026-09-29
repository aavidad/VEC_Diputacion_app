package bootstrap

import (
	"testing"

	"vec-diputacion-granada/config"
	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestPortalExternoAdicionalesExigeMaterialCompletoDeSuSuperficie(t *testing.T) {
	correos := map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo{}
	for _, audiencia := range audienciasCorreosUsuariosDesarrollo()[len(accionesCorreosUsuarios):] {
		correos[audiencia] = &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	}
	materialCorreo, ok := materialesCorreosPortalExterno(correos)
	if !ok || materialCorreo.avisos != nil || materialCorreo.lote[0] != nil || materialCorreo.lote[5] != nil || materialCorreo.lote[6] == nil || materialCorreo.lote[11] == nil {
		t.Fatal("material de correos internos incluido o material externo perdido")
	}
	delete(correos, audienciasCorreosUsuariosDesarrollo()[6])
	if _, ok := materialesCorreosPortalExterno(correos); ok {
		t.Fatal("audiencia externa de correos ausente aceptada")
	}
	correos[audienciasCorreosUsuariosDesarrollo()[0]] = &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	if _, ok := materialesCorreosPortalExterno(correos); ok {
		t.Fatal("audiencia interna usada como sustituto")
	}
	imagen := map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo{}
	for _, audiencia := range audienciasImagenUsuariosDesarrollo()[len(accionesImagenUsuarios):] {
		imagen[audiencia] = &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	}
	materialImagen, ok := materialesImagenPortalExterno(imagen)
	if !ok || materialImagen[0] != nil || materialImagen[1] != nil || materialImagen[2] == nil || materialImagen[3] == nil {
		t.Fatal("material de imagen internos incluido o material externo perdido")
	}
	imagen[audienciasImagenUsuariosDesarrollo()[0]] = &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	if _, ok := materialesImagenPortalExterno(imagen); ok {
		t.Fatal("audiencia interna de imagen aceptada")
	}
}

func TestPortalExternoCorreosNoArrancaSinFuentePropia(t *testing.T) {
	t.Setenv(envUsuariosCorreosDesarrollo, "true")
	t.Setenv(envUsuariosImagenDesarrollo, "")
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
	if _, _, err := nuevasDependenciasAdicionalesPortalExterno(t.Context(), cfg, nil); err == nil {
		t.Fatal("correos activados sin fuente de claves propia")
	}
}

func TestPortalExternoAutoridadNoAdmiteRutasInternasNiAusentes(t *testing.T) {
	a := autoridadExactasPortalExterno{preferencias: &autoridadPreferenciasUsuariosDesarrollo{superficie: core.SuperficieAutenticacionExternaPersonalV1}}
	for _, ruta := range []string{
		usuarioshttp.RutaMisPreferencias, usuarioshttp.RutaMisCorreos, usuarioshttp.RutaMiImagen,
		usuarioshttp.RutaMisCorreosAreaPersonal, usuarioshttp.RutaMiImagenAreaPersonal,
	} {
		if err := a.AutorizarRutaExacta(t.Context(), ruta); err == nil {
			t.Fatalf("ruta %s aceptada sin autoridad de petición", ruta)
		}
	}
}
