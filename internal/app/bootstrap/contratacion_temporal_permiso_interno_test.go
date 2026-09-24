package bootstrap

import (
	"reflect"
	"strings"
	"testing"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestPermisoInternoCTSoloConsultaTemporalEnPerfilPropio(t *testing.T) {
	s := SolicitudPermisoInternoCT{
		CuentaRef: "cta_0123456789abcdefghijkl", PrincipalRef: "per_0123456789abcdefghijkl",
		PerfilRef: "prf_0123456789abcdefghijkl", OrganizacionRef: "org_0123456789abcdefghijkl",
		PoliticaRef: "pga_0123456789abcdefghijkl", PoliticaHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	hasta := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	i, err := plantillaPermisoInternoCT(s, ahora, hasta)
	if err != nil || i.Validar() != nil {
		t.Fatalf("plantilla CT inválida: %v", err)
	}
	if len(i.VersionRol.Concesiones) != 1 || i.VersionRol.Concesiones[0].Accion != ct.AccionConsultarDetalleRRHH ||
		i.VersionRol.Concesiones[0].GarantiaMinima != core.AuthAssuranceSubstantial ||
		i.VersionRol.Concesiones[0].ModuloID != ct.ModuloContratacion ||
		i.VersionRol.Concesiones[0].TipoRecurso != ct.TipoRecursoExpediente ||
		len(i.VersionRol.Concesiones[0].Obligaciones) != 0 ||
		i.AsignacionPerfil.PerfilActivoRef != s.PerfilRef || !i.AsignacionPerfil.VigenteHasta.Equal(hasta) ||
		len(i.AsignacionPerfil.Ambitos) != 3 {
		t.Fatal("se amplió el permiso interno")
	}
	permitidos := map[string]bool{}
	for _, campo := range i.VersionRol.Concesiones[0].CamposPermitidos {
		permitidos[campo] = true
	}
	proyeccion := reflect.TypeOf(ct.VistaSeguimientoIncorporacionV2{})
	if len(permitidos) != proyeccion.NumField() {
		t.Fatal("la lista V3 difiere de la proyección pública")
	}
	for indice := 0; indice < proyeccion.NumField(); indice++ {
		campo := strings.Split(proyeccion.Field(indice).Tag.Get("json"), ",")[0]
		if campo == "" || !permitidos[campo] {
			t.Fatalf("campo HTTP de seguimiento fuera de V3: %s", campo)
		}
	}
	if !perfilInternoPublicadoExacto(s, i.AsignacionPerfil, i.VersionRol, i.ControlVigenciaVersionRol, hasta, ahora) {
		t.Fatal("replay exacto rechazado")
	}
	rol := i.VersionRol
	rol.Concesiones = append([]core.ConcesionRol(nil), rol.Concesiones...)
	rol.Concesiones[0].Accion = ct.AccionConfirmarIncorporacion
	if perfilInternoPublicadoExacto(s, i.AsignacionPerfil, rol, i.ControlVigenciaVersionRol, hasta, ahora) {
		t.Fatal("replay con operación de efecto aceptado")
	}
	otroPerfil := s
	otroPerfil.PerfilRef = "prf_abcdefghijkl0123456789"
	if perfilInternoPublicadoExacto(otroPerfil, i.AsignacionPerfil, i.VersionRol, i.ControlVigenciaVersionRol, hasta, ahora) {
		t.Fatal("asignación de otro perfil aceptada")
	}
	controlRetirado := i.ControlVigenciaVersionRol
	controlRetirado.Estado = core.EstadoControlVigenciaVersionRolRetirada
	controlRetirado.ActoRef = "acto:retirada:prueba"
	controlRetirado.MotivoCodigo = "retirada_temporal"
	if perfilInternoPublicadoExacto(s, i.AsignacionPerfil, i.VersionRol, controlRetirado, hasta, ahora) {
		t.Fatal("rol retirado aceptado como replay")
	}
	if _, err := plantillaPermisoInternoCT(s, ahora, retiradaMaximaPermisoInternoCT.Add(time.Microsecond)); err == nil {
		t.Fatal("se aceptó vigencia posterior al límite")
	}
	if _, err := plantillaPermisoInternoCT(s, hasta, hasta); err == nil {
		t.Fatal("se aceptó vigencia vacía")
	}
}
