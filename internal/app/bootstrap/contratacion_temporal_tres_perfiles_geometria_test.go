package bootstrap

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type publicadorAltaFijaProhibidoPrueba struct{ llamadas int }

func (p *publicadorAltaFijaProhibidoPrueba) PrepararInstantanea(
	context.Context, dominiovec.InstantaneaAutorizacion,
) (dominiovec.InstantaneaAutorizacion, error) {
	p.llamadas++
	return dominiovec.InstantaneaAutorizacion{}, errAltaContratacionTemporalDesarrolloNoDisponible
}

func (p *publicadorAltaFijaProhibidoPrueba) PublicarInstantanea(
	context.Context, dominiovec.InstantaneaAutorizacion,
) error {
	p.llamadas++
	return errAltaContratacionTemporalDesarrolloNoDisponible
}

func TestTresPerfilesCTCompartenPersonaSinCompartirSesion(t *testing.T) {
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	constructores := []struct {
		nombre string
		nuevo  func(dominiovec.Principal, time.Time) (ports.ContextoAutorizacionAltaV3, error)
	}{
		{"legado", nuevoContextoAltaContratacionTemporalDesarrollo},
		{"alta directa", nuevoContextoAltaFijoContratacionTemporalDesarrollo},
		{"cobertura", nuevoContextoCoberturaContratacionTemporalDesarrollo},
	}
	contextos := make([]ports.ContextoAutorizacionAltaV3, len(constructores))
	vinculos := make([]dominiovec.DatosVinculoAutenticacionActorV2, len(constructores))
	for i, caso := range constructores {
		var err error
		contextos[i], err = caso.nuevo(principal, ahora)
		if err != nil {
			t.Fatalf("%s: %v", caso.nombre, err)
		}
		vinculos[i], err = contextos[i].Vinculo.Datos()
		if err != nil || contextos[i].ValidarPara(solicitudResolverContextoAltaV3Prueba(vinculos[i]), ahora) != nil {
			t.Fatalf("%s: contexto o vínculo inválido: %v", caso.nombre, err)
		}
	}
	for i := range contextos {
		for j := i + 1; j < len(contextos); j++ {
			a, b := contextos[i].Resultado.Contexto, contextos[j].Resultado.Contexto
			if vinculos[i].PrincipalID != vinculos[j].PrincipalID ||
				a.Instantanea.CuentaRef != b.Instantanea.CuentaRef || a.PersonaRef != b.PersonaRef ||
				vinculos[i].PerfilActivoRef == vinculos[j].PerfilActivoRef ||
				a.Instantanea.VinculoRef == b.Instantanea.VinculoRef ||
				vinculos[i].SesionRef == vinculos[j].SesionRef ||
				contextos[i].Resultado.RegistroContextoRef == contextos[j].Resultado.RegistroContextoRef {
				t.Fatalf("%s y %s mezclan identidad o perfil", constructores[i].nombre, constructores[j].nombre)
			}
		}
	}
	manifiestoLegado, err := dominiovec.RehidratarManifiestoProcedenciaContextoActorV1(contextos[0].Resultado.ManifiestoProcedenciaCanonico)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(contextos); i++ {
		manifiesto, err := dominiovec.RehidratarManifiestoProcedenciaContextoActorV1(contextos[i].Resultado.ManifiestoProcedenciaCanonico)
		if err != nil || manifiesto.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1 != manifiestoLegado.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1 ||
			manifiesto.Persona.AcreditacionProcedenciaComponenteContextoActorV1 != manifiestoLegado.Persona.AcreditacionProcedenciaComponenteContextoActorV1 {
			t.Fatalf("%s pierde procedencia de cuenta o persona: %v", constructores[i].nombre, err)
		}
	}
}

func TestAltaDirectaCTUsaInstantaneaFijaSinPreparacionPorPeticion(t *testing.T) {
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	legado, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	fijo, err := nuevoContextoAltaFijoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	vLegado, _ := legado.Vinculo.Datos()
	vFijo, _ := fijo.Vinculo.Datos()
	iLegado, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(vLegado.PrincipalID, vLegado.PerfilActivoRef, ahora)
	if err != nil {
		t.Fatal(err)
	}
	iFija, err := nuevaInstantaneaAutorizacionAltaFijaContratacionTemporalDesarrollo(vFijo.PrincipalID, vFijo.PerfilActivoRef, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if iLegado.VersionRol.RolID == iFija.VersionRol.RolID ||
		iLegado.AsignacionPerfil.AsignacionID == iFija.AsignacionPerfil.AsignacionID ||
		len(iFija.AsignacionPerfil.Ambitos) != 3 {
		t.Fatal("alta fija comparte rol/asignación o pierde ámbito")
	}
	publicador := &publicadorAltaFijaProhibidoPrueba{}
	soporte := &soporteAltaContratacionTemporalDesarrollo{instantanea: iLegado, instantaneaAltaFija: iFija, autoridadAsignaciones: publicador}
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		Accion: ports.AccionCrearSolicitud,
		Recurso: dominiovec.RecursoAutorizable{
			ModuloID: ports.ModuloContratacion,
			Tipo:     ports.TipoRecursoExpediente,
			Ambitos: map[string]string{
				"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
				"centro_ref":       centroAltaContratacionTemporalDesarrollo,
				"categoria_ref":    categoriaAltaContratacionTemporalDesarrollo,
			},
		},
		Finalidad: ports.FinalidadCrearSolicitud,
	}
	ctx := context.WithValue(context.Background(), claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	actual, valida := soporte.instantaneaParaContexto(ctx, httpinterno.RutaAltaSolicitudes)
	if !valida || actual.AsignacionPerfil.PerfilActivoRef != vFijo.PerfilActivoRef || publicador.llamadas != 0 {
		t.Fatal("alta directa no selecciona perfil fijo provisionado")
	}
	soporte.instantaneaAltaFija.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
	if _, valida := soporte.instantaneaParaContexto(ctx, httpinterno.RutaAltaSolicitudes); valida || publicador.llamadas != 0 {
		t.Fatal("alta directa reactivada con perfil legado tras revocación fija")
	}
}
