package bootstrap

import (
	"context"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestArranqueAdoptaRolS6PublicadoSinVolverAPublicar(t *testing.T) {
	politica, autoridad, datos := politicaProvisionBolsaPrueba(t, 0)
	actual, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, politica.soporte.unidadRef,
		politica.soporte.ambitoRef, politica.reloj.Ahora(), 5)
	if err != nil {
		t.Fatal(err)
	}
	actual.VersionRol.Version = 17
	actual.VersionRol.PublicadaPor = "seguridad:acto-gobernado"
	actual.AsignacionPerfil.VersionRolRef = actual.VersionRol.Referencia()
	actual.AsignacionPerfil.EmitidaPor = "identidad:acto-gobernado"
	actual.ControlVigenciaVersionRol.VersionRolRef = actual.VersionRol.Referencia()
	actual.ControlVigenciaVersionRol.ActualizadoPor = "seguridad:acto-gobernado"
	actual.VersionRol.Concesiones = append(actual.VersionRol.Concesiones,
		dominiovec.ConcesionRol{Accion: puertosbolsa.AccionConsultarTransicionesSituacion, ModuloID: "bolsa",
			TipoRecurso:      puertosbolsa.TipoRecursoGobiernoTransicionesSituacion,
			Finalidades:      []string{puertosbolsa.FinalidadConsultaTransicionesSituacion},
			CamposPermitidos: []string{puertosbolsa.CampoConsultaTransicionesSituacion},
			GarantiaMinima:   dominiovec.AuthAssuranceHigh},
		dominiovec.ConcesionRol{Accion: puertosbolsa.AccionPublicarTransicionesSituacion, ModuloID: "bolsa",
			TipoRecurso:    puertosbolsa.TipoRecursoGobiernoTransicionesSituacion,
			Finalidades:    []string{puertosbolsa.FinalidadGobiernoTransicionesSituacion},
			GarantiaMinima: dominiovec.AuthAssuranceHigh})
	if err := actual.Validar(); err != nil {
		t.Fatal(err)
	}
	autoridad.leida = instantaneaPublicadaDesarrollo{instantanea: actual}
	autoridad.encontrada = true
	if err := politica.PublicarInicial(context.Background()); err != nil || autoridad.publicadas != 0 ||
		politica.instantanea.VersionRol.Version != 17 {
		t.Fatalf("el arranque no adoptó el rol publicado: err=%v escrituras=%d versión=%d",
			err, autoridad.publicadas, politica.instantanea.VersionRol.Version)
	}
	ajena := actual.VersionRol.Concesiones[0]
	ajena.Accion = "bolsa.otra_accion.consultar"
	actual.VersionRol.Concesiones = append(actual.VersionRol.Concesiones, ajena)
	if instantaneaBolsaS6Compatible(actual, datos, politica.soporte, politica.reloj.Ahora(), 5) {
		t.Fatal("el arranque adoptó una concesión adicional a las acciones de situación")
	}
}
