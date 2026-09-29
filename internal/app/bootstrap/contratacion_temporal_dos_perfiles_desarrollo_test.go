package bootstrap

import (
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func solicitudResolverContextoAltaV3Prueba(v dominiovec.DatosVinculoAutenticacionActorV2) ports.SolicitudResolverContextoAutorizacionAltaV3 {
	return ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: v.AutenticacionRef,
		SesionRef:        v.SesionRef,
		PerfilRef:        v.PerfilActivoRef,
	}
}

func TestDosPerfilesCTConservanCuentaPersonaYProcedencia(t *testing.T) {
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	alta, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	cobertura, err := nuevoContextoCoberturaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	vAlta, err := alta.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	vCobertura, err := cobertura.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	a, c := alta.Resultado.Contexto, cobertura.Resultado.Contexto
	condiciones := map[string]bool{
		"alta no valida":            alta.ValidarPara(solicitudResolverContextoAltaV3Prueba(vAlta), ahora) != nil,
		"cobertura no valida":       cobertura.ValidarPara(solicitudResolverContextoAltaV3Prueba(vCobertura), ahora) != nil,
		"principal distinto":        vAlta.PrincipalID != vCobertura.PrincipalID,
		"principal actor distinto":  a.Principal.ID != c.Principal.ID,
		"perfil compartido":         vAlta.PerfilActivoRef == vCobertura.PerfilActivoRef,
		"cuenta distinta":           a.Instantanea.CuentaRef != c.Instantanea.CuentaRef,
		"persona distinta":          a.PersonaRef != c.PersonaRef,
		"vinculo compartido":        a.Instantanea.VinculoRef == c.Instantanea.VinculoRef,
		"registro compartido":       alta.Resultado.RegistroContextoRef == cobertura.Resultado.RegistroContextoRef,
		"autenticacion compartida":  vAlta.AutenticacionRef == vCobertura.AutenticacionRef,
		"sesion compartida":         vAlta.SesionRef == vCobertura.SesionRef,
		"control sesion compartido": vAlta.ControlSesionRef == vCobertura.ControlSesionRef,
	}
	for nombre, incumplida := range condiciones {
		if incumplida {
			t.Error(nombre)
		}
	}
	mAlta, err := dominiovec.RehidratarManifiestoProcedenciaContextoActorV1(alta.Resultado.ManifiestoProcedenciaCanonico)
	if err != nil {
		t.Fatal(err)
	}
	mCobertura, err := dominiovec.RehidratarManifiestoProcedenciaContextoActorV1(cobertura.Resultado.ManifiestoProcedenciaCanonico)
	if err != nil {
		t.Fatal(err)
	}
	if mAlta.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1 != mCobertura.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1 ||
		mAlta.Persona.AcreditacionProcedenciaComponenteContextoActorV1 != mCobertura.Persona.AcreditacionProcedenciaComponenteContextoActorV1 ||
		mAlta.Perfil.PerfilRef == mCobertura.Perfil.PerfilRef ||
		mAlta.Contexto.VinculoRef == mCobertura.Contexto.VinculoRef {
		t.Fatal("procedencia compartida y objetos propios no conservados")
	}
}

func TestDosPerfilesCTConservanConcesionesYAmbitosSeparados(t *testing.T) {
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	contextoAlta, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	contextoCobertura, err := nuevoContextoCoberturaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	vAlta, _ := contextoAlta.Vinculo.Datos()
	vCobertura, _ := contextoCobertura.Vinculo.Datos()
	alta, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		vAlta.PrincipalID, vAlta.PerfilActivoRef, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	cobertura, err := nuevaInstantaneaAutorizacionCoberturaContratacionTemporalDesarrollo(
		vCobertura.PrincipalID, vCobertura.PerfilActivoRef, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	if alta.Validar() != nil || cobertura.Validar() != nil ||
		alta.AsignacionPerfil.PrincipalID != cobertura.AsignacionPerfil.PrincipalID ||
		alta.AsignacionPerfil.PerfilActivoRef == cobertura.AsignacionPerfil.PerfilActivoRef ||
		alta.AsignacionPerfil.AsignacionID == cobertura.AsignacionPerfil.AsignacionID ||
		alta.VersionRol.RolID != "tecnico_rrhh_desarrollo" ||
		cobertura.VersionRol.RolID != "tecnico_rrhh_cobertura_desarrollo" ||
		len(alta.AsignacionPerfil.Ambitos) != 3 ||
		len(cobertura.AsignacionPerfil.Ambitos) != 2 ||
		alta.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" ||
		alta.AsignacionPerfil.Ambitos[1].Clave != "centro_ref" ||
		alta.AsignacionPerfil.Ambitos[2].Clave != "categoria_ref" ||
		cobertura.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" ||
		cobertura.AsignacionPerfil.Ambitos[1].Clave != "unidad_ejecutora_ref" {
		t.Fatal("concesiones o ámbitos mezclados")
	}
}

func TestContextoCoberturaCTNoAceptaOtroPrincipalOCertificado(t *testing.T) {
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	contexto, err := nuevoContextoCoberturaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	otro := principal
	otro.Attributes = make(map[string]string, len(principal.Attributes))
	for clave, valor := range principal.Attributes {
		otro.Attributes[clave] = valor
	}
	otro.Attributes["certificate_sha256"] = strings.Repeat("f", 64)
	contextoOtro, err := nuevoContextoCoberturaContratacionTemporalDesarrollo(otro, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if contexto.Resultado.Contexto.Instantanea.CuentaRef == contextoOtro.Resultado.Contexto.Instantanea.CuentaRef ||
		contexto.Resultado.Contexto.PersonaRef == contextoOtro.Resultado.Contexto.PersonaRef ||
		contexto.Resultado.Contexto.PerfilActivoRef == contextoOtro.Resultado.Contexto.PerfilActivoRef {
		t.Fatal("cambio de certificado reutilizó identidad sintética")
	}
	otro = principal
	otro.Roles = []string{"intervencion"}
	if _, err := nuevoContextoCoberturaContratacionTemporalDesarrollo(otro, ahora); err == nil {
		t.Fatal("otro rol obtuvo contexto RRHH de cobertura")
	}
}
