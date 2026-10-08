package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// autoridadProvisionBolsaPrueba añade a la autoridad de prueba la lectura de
// la asignación publicada y registra la preimagen de cada publicación.
type autoridadProvisionBolsaPrueba struct {
	autoridadInicialBorradorBolsaPrueba
	leida      instantaneaPublicadaDesarrollo
	encontrada bool
	preimagen  int
}

func (a *autoridadProvisionBolsaPrueba) leerAsignacionPublicada(context.Context, string) (instantaneaPublicadaDesarrollo, bool, error) {
	return a.leida, a.encontrada, nil
}

func (a *autoridadProvisionBolsaPrueba) publicarInstantaneaDesdePreimagen(
	_ context.Context, instantanea, preimagen dominiovec.InstantaneaAutorizacion,
) error {
	a.publicadas++
	a.preimagen = preimagen.VersionRol.Version
	a.publicada = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(instantanea)
	return nil
}

func politicaProvisionBolsaPrueba(t *testing.T, versionPublicada int) (*politicaBorradorLlamamientoBolsaDesarrollo, *autoridadProvisionBolsaPrueba, dominiovec.DatosVinculoAutenticacionActorV2) {
	t.Helper()
	directorio, soporteCT, principal, ahora := fixtureSoporteSesionBorradorBolsa(t)
	escribirManifiestoIdentidadBorradorBolsa(t, directorio, principal, ahora, nil)
	soporte, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, soporteCT, ahora)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadProvisionBolsaPrueba{}
	if versionPublicada > 0 {
		publicada, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
			datos.PrincipalID, datos.PerfilActivoRef, soporte.unidadRef, soporte.ambitoRef, relojContratacionTemporalDesarrollo{}.Ahora(), versionPublicada)
		if err != nil {
			t.Fatal(err)
		}
		autoridad.leida, autoridad.encontrada = instantaneaPublicadaDesarrollo{instantanea: publicada}, true
	}
	politica, err := nuevaPoliticaBorradorLlamamientoBolsaDesarrollo(soporte, autoridad, &registroBorradorBolsaPrueba{}, relojContratacionTemporalDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	return politica, autoridad, datos
}

func aprobarProvisionBolsaPrueba(t *testing.T, politica *politicaBorradorLlamamientoBolsaDesarrollo, autoridad *autoridadProvisionBolsaPrueba,
	datos dominiovec.DatosVinculoAutenticacionActorV2, objetivo int,
) {
	t.Helper()
	preimagen, huella, err := huellasProvisionRRHHBolsa(autoridad.leida.instantanea, datos, politica.soporte, politica.reloj.Ahora(), objetivo)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("VEC_BOLSA_DOCUMENTAL_PROVISION_APROBACION_REF", "aprobacion:prueba-b78")
	t.Setenv("VEC_BOLSA_DOCUMENTAL_PROVISION_PREIMAGEN_SHA256", preimagen)
	t.Setenv("VEC_BOLSA_DOCUMENTAL_PROVISION_OBJETIVO_SHA256", huella)
}

func concedeConsultaDatosContacto(i dominiovec.InstantaneaAutorizacion) bool {
	for _, c := range i.VersionRol.Concesiones {
		if c.Accion == puertosbolsa.AccionConsultarDatosContactoParticipacion {
			return c.ModuloID == puertosbolsa.ModuloSituacionParticipacion && c.TipoRecurso == puertosbolsa.TipoRecursoSituacionParticipacion &&
				len(c.Finalidades) == 1 && c.Finalidades[0] == puertosbolsa.FinalidadConsultarDatosContactoParticipacion &&
				len(c.CamposPermitidos) == 0 && len(c.Obligaciones) == 0
		}
	}
	return false
}

func TestRolBolsaConcedeConsultaDatosContactoSoloEnTercerGrupo(t *testing.T) {
	politica, _, datos := politicaProvisionBolsaPrueba(t, 0)
	for version := 5; version <= 16; version++ {
		i, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
			datos.PrincipalID, datos.PerfilActivoRef, politica.soporte.unidadRef, politica.soporte.ambitoRef, politica.reloj.Ahora(), version)
		if err != nil {
			t.Fatal(err)
		}
		if concedeConsultaDatosContacto(i) != (version >= 13) {
			t.Fatalf("v%d: concesión de consulta de datos de contacto=%t", version, concedeConsultaDatosContacto(i))
		}
	}
	// Las ramas previas conservan su numeración.
	for _, v := range []int{6, 8, 10, 12, 14, 16} {
		if !versionRolBolsaConPoliticaOfertas(v) || versionRolBolsaConPoliticaOfertas(v-1) {
			t.Fatalf("política de ofertas mal ubicada en v%d", v)
		}
	}
	for _, v := range []int{7, 8, 11, 12, 15, 16} {
		if !versionRolBolsaConReincorporacion(v) {
			t.Fatalf("reincorporación ausente en v%d", v)
		}
	}
	for _, v := range []int{5, 6, 9, 10, 13, 14, 17} {
		if versionRolBolsaConReincorporacion(v) {
			t.Fatalf("reincorporación indebida en v%d", v)
		}
	}
}

func TestProvisionBolsaAmpliaDocumentalAConsultaDatosContacto(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	// Documental publicada sin aprobación nueva: se conserva tal cual.
	politica, autoridad, _ := politicaProvisionBolsaPrueba(t, 9)
	if err := politica.PublicarInicial(context.Background()); err != nil || autoridad.publicadas != 0 || politica.instantanea.VersionRol.Version != 9 {
		t.Fatalf("documental sin aprobación: err=%v publicadas=%d versión=%d", err, autoridad.publicadas, politica.instantanea.VersionRol.Version)
	}
	// El motivo es válido aunque falte la concesión: deniega el PDP, no el motivo.
	if err := politica.ValidarReferenciaMotivoAutorizacionV2(context.Background(), motivoConsultarDatosContactoParticipacionBolsaDesarrollo(), ahora); err != nil || concedeConsultaDatosContacto(politica.instantanea) {
		t.Fatalf("v9: motivo=%v concesión=%t", err, concedeConsultaDatosContacto(politica.instantanea))
	}
	// Con aprobación exacta: v9 → v13 por CAS sobre la publicada.
	politica, autoridad, datos := politicaProvisionBolsaPrueba(t, 9)
	aprobarProvisionBolsaPrueba(t, politica, autoridad, datos, 13)
	if err := politica.PublicarInicial(context.Background()); err != nil || autoridad.publicadas != 1 || autoridad.preimagen != 9 ||
		politica.instantanea.VersionRol.Version != 13 || !concedeConsultaDatosContacto(politica.instantanea) {
		t.Fatalf("v9→v13: err=%v publicadas=%d preimagen=%d versión=%d", err, autoridad.publicadas, autoridad.preimagen, politica.instantanea.VersionRol.Version)
	}
	if politica.instantanea.AsignacionPerfil.Version != autoridad.leida.instantanea.AsignacionPerfil.Version+1 {
		t.Fatal("la provisión no avanzó la versión de la asignación")
	}
	if err := politica.ValidarReferenciaMotivoAutorizacionV2(context.Background(), motivoConsultarDatosContactoParticipacionBolsaDesarrollo(), ahora); err != nil {
		t.Fatalf("v13 rechazó el motivo de consulta: %v", err)
	}
	// Base con política de ofertas y aprobación: v6 → v14 directamente.
	politica, autoridad, datos = politicaProvisionBolsaPrueba(t, 6)
	aprobarProvisionBolsaPrueba(t, politica, autoridad, datos, 14)
	if err := politica.PublicarInicialConPoliticaOfertas(context.Background()); err != nil || autoridad.preimagen != 6 || politica.instantanea.VersionRol.Version != 14 {
		t.Fatalf("v6→v14: err=%v preimagen=%d versión=%d", err, autoridad.preimagen, politica.instantanea.VersionRol.Version)
	}
}

func TestProvisionBolsaConservaCompletaYRechazaHuellaORamaAjena(t *testing.T) {
	politica, autoridad, _ := politicaProvisionBolsaPrueba(t, 13)
	if err := politica.PublicarInicial(context.Background()); err != nil || autoridad.publicadas != 0 || politica.instantanea.VersionRol.Version != 13 {
		t.Fatalf("v13 publicada no se conservó: err=%v versión=%d", err, politica.instantanea.VersionRol.Version)
	}
	// Huella de objetivo distinta: PARO sin publicar.
	politica, autoridad, datos := politicaProvisionBolsaPrueba(t, 9)
	aprobarProvisionBolsaPrueba(t, politica, autoridad, datos, 9)
	if err := politica.PublicarInicial(context.Background()); err == nil || autoridad.publicadas != 0 || politica.publicada {
		t.Fatalf("aprobación de otra huella aceptada: err=%v publicadas=%d", err, autoridad.publicadas)
	}
	// Aprobación de la provisión documental anterior aún en el entorno: con
	// v9 publicada no coincide su preimagen; se ignora y se conserva v9.
	politica, autoridad, _ = politicaProvisionBolsaPrueba(t, 9)
	t.Setenv("VEC_BOLSA_DOCUMENTAL_PROVISION_APROBACION_REF", "aprobacion:prueba-b77-anterior")
	t.Setenv("VEC_BOLSA_DOCUMENTAL_PROVISION_PREIMAGEN_SHA256", strings.Repeat("a", 64))
	t.Setenv("VEC_BOLSA_DOCUMENTAL_PROVISION_OBJETIVO_SHA256", strings.Repeat("b", 64))
	if err := politica.PublicarInicial(context.Background()); err != nil || autoridad.publicadas != 0 || politica.instantanea.VersionRol.Version != 9 {
		t.Fatalf("aprobación anterior impidió el arranque: err=%v publicadas=%d versión=%d", err, autoridad.publicadas, politica.instantanea.VersionRol.Version)
	}
	// Una versión de otra rama (reincorporación) no se toma por la propia.
	politica, autoridad, _ = politicaProvisionBolsaPrueba(t, 15)
	if err := politica.PublicarInicial(context.Background()); err == nil || autoridad.publicadas != 0 || politica.publicada {
		t.Fatalf("rama ajena aceptada: err=%v", err)
	}
}

func TestCargaConvocaNoSeProvisionaEnArranqueSinPlantillaGobernada(t *testing.T) {
	concede := func(i dominiovec.InstantaneaAutorizacion) bool {
		for _, c := range i.VersionRol.Concesiones {
			if c.Accion == puertosbolsa.AccionConfirmarCargaConvoca {
				return true
			}
		}
		return false
	}
	politicaBase, _, datosBase := politicaProvisionBolsaPrueba(t, 0)
	for v := 1; v <= 16; v++ {
		base, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(datosBase.PrincipalID, datosBase.PerfilActivoRef,
			politicaBase.soporte.unidadRef, politicaBase.soporte.ambitoRef, politicaBase.reloj.Ahora(), v)
		if err != nil || concede(base) {
			t.Fatalf("v%d concedió B1 sin plantilla gobernada: %v", v, err)
		}
	}
	for _, v := range []int{17, 21, 29, 32} {
		if _, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(datosBase.PrincipalID, datosBase.PerfilActivoRef,
			politicaBase.soporte.unidadRef, politicaBase.soporte.ambitoRef, politicaBase.reloj.Ahora(), v); err == nil {
			t.Fatalf("v%d fue admitida sin plantilla gobernada", v)
		}
	}
	politica, autoridad, _ := politicaProvisionBolsaPrueba(t, 13)
	if err := politica.PublicarInicial(context.Background()); err != nil || autoridad.publicadas != 0 || concede(politica.instantanea) || politica.permiteCargaConvoca() {
		t.Fatalf("v13 adquirió B1: err=%v publicaciones=%d", err, autoridad.publicadas)
	}
	politica, autoridad, _ = politicaProvisionBolsaPrueba(t, 13)
	t.Setenv(envCargaConvocaAprobacion, "aprobacion:prueba-b1")
	t.Setenv(envCargaConvocaPreimagen, strings.Repeat("a", 64))
	t.Setenv(envCargaConvocaObjetivo, strings.Repeat("b", 64))
	if err := politica.PublicarInicial(context.Background()); err == nil || autoridad.publicadas != 0 || politica.publicada {
		t.Fatalf("arranque aceptó ajustes B1 sin plantilla: err=%v publicaciones=%d", err, autoridad.publicadas)
	}
}
