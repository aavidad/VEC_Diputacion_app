package application

import (
	"context"
	"errors"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type contextoContactoPrueba struct{ err error }

func (c contextoContactoPrueba) ResolverContextoSituacionParticipacion(context.Context, dominiovec.ContextoActor, string, string) (puertosbolsa.ContextoSituacionParticipacionResuelto, error) {
	return c.resolver()
}
func (c contextoContactoPrueba) ResolverContextoContactosBolsa(context.Context, dominiovec.ContextoActor, string) (puertosbolsa.ContextoSituacionParticipacionResuelto, error) {
	return c.resolver()
}
func (c contextoContactoPrueba) resolver() (puertosbolsa.ContextoSituacionParticipacionResuelto, error) {
	if c.err != nil {
		return puertosbolsa.ContextoSituacionParticipacionResuelto{}, c.err
	}
	return puertosbolsa.ContextoSituacionParticipacionResuelto{UnidadRef: "unidad:rrhh", AmbitoRef: "ambito:bolsa"}, nil
}

type repositorioContactoPrueba struct{ lecturasParticipacion, lecturasBolsa int }

func (*repositorioContactoPrueba) ParticipacionPerteneceABolsa(context.Context, string, string) (bool, error) {
	return true, nil
}
func (*repositorioContactoPrueba) RegistrarContacto(context.Context, puertosbolsa.ComandoRegistrarContactoParticipacion) (puertosbolsa.RegistroContactoParticipacion, error) {
	return puertosbolsa.RegistroContactoParticipacion{}, nil
}
func (r *repositorioContactoPrueba) ListarContactosParticipacion(context.Context, puertosbolsa.ConsultaContactosParticipacion) (puertosbolsa.PaginaContactosParticipacion, error) {
	r.lecturasParticipacion++
	return puertosbolsa.PaginaContactosParticipacion{}, nil
}
func (r *repositorioContactoPrueba) ListarContactosBolsa(context.Context, puertosbolsa.ConsultaContactosBolsa) (puertosbolsa.PaginaContactosParticipacion, error) {
	r.lecturasBolsa++
	return puertosbolsa.PaginaContactosParticipacion{}, nil
}

func resultadoContextoContactoPrueba(t *testing.T) (dominiovec.ResultadoContextoActorRegistradoV2, dominiovec.VinculoAutenticacionActorV2) {
	t.Helper()
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC), "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	return resultado, vinculo
}

func TestServicioContactoParticipacionExigeDependencias(t *testing.T) {
	if _, err := NuevoServicioContactoParticipacion(nil, nil, nil); err == nil {
		t.Fatal("servicio incompleto admitido")
	}
}

func TestServicioContactoDeniegaLecturasFueraDeAmbitoAntesDelRepositorio(t *testing.T) {
	repo := &repositorioContactoPrueba{}
	servicio, err := NuevoServicioContactoParticipacion(contextoContactoPrueba{err: dominiovec.ErrAutorizacionDenegada}, &autorizadorBorradorPrueba{}, repo)
	if err != nil {
		t.Fatal(err)
	}
	resultado, vinculo := resultadoContextoContactoPrueba(t)
	_, errParticipacion := servicio.ListarContactosParticipacion(context.Background(), puertosbolsa.ConsultaContactosParticipacion{Vinculo: vinculo, ResultadoContexto: resultado, BolsaRef: "bolsa:ajena", ParticipacionRef: "participacion:01", Limite: 20})
	_, errBolsa := servicio.ListarContactosBolsa(context.Background(), puertosbolsa.ConsultaContactosBolsa{Vinculo: vinculo, ResultadoContexto: resultado, BolsaRef: "bolsa:ajena", Limite: 20})
	if !errors.Is(errParticipacion, dominiovec.ErrAutorizacionDenegada) || !errors.Is(errBolsa, dominiovec.ErrAutorizacionDenegada) || repo.lecturasParticipacion != 0 || repo.lecturasBolsa != 0 {
		t.Fatalf("participacion=%v bolsa=%v lecturas=%+v", errParticipacion, errBolsa, repo)
	}
}

func TestServicioContactoLeeSoloTrasResolverElAmbito(t *testing.T) {
	repo := &repositorioContactoPrueba{}
	servicio, err := NuevoServicioContactoParticipacion(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)}, repo)
	if err != nil {
		t.Fatal(err)
	}
	resultado, vinculo := resultadoContextoContactoPrueba(t)
	if _, err = servicio.ListarContactosParticipacion(context.Background(), puertosbolsa.ConsultaContactosParticipacion{Vinculo: vinculo, ResultadoContexto: resultado, BolsaRef: "bolsa:permitida", ParticipacionRef: "participacion:01", Limite: 20, Correlacion: correlacionBorradorPrueba(t), MotivoAutorizacion: motivoBorradorPrueba()}); err != nil || repo.lecturasParticipacion != 1 {
		t.Fatalf("err=%v lecturas=%d", err, repo.lecturasParticipacion)
	}
}
