package domain

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

type resolutorContextoActorGobernadoV1Prueba struct {
	resultado     ResultadoContextoActorRegistradoV2
	err           error
	invocaciones  int
	autenticacion AutenticacionRevalidadaV1
	despues       func()
}

func (r *resolutorContextoActorGobernadoV1Prueba) ResolverContextoActorGobernadoV1(
	_ context.Context,
	autenticacion AutenticacionRevalidadaV1,
) (ResultadoContextoActorRegistradoV2, error) {
	r.invocaciones++
	r.autenticacion = autenticacion
	if r.despues != nil {
		r.despues()
	}
	return r.resultado, r.err
}

func TestCrearVinculoAutenticacionActorGobernadoV1ResuelveSoloDesdeAutenticacion(t *testing.T) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	resultadoFuente := resultadoContextoActorRegistradoV2ConContenedoresVaciosPrueba(t, ahora)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	revalidador := &revalidadorAutenticacionV2Prueba{resultado: autenticacion}
	resolutor := &resolutorContextoActorGobernadoV1Prueba{resultado: resultadoFuente}

	vinculo, resultado, err := CrearVinculoAutenticacionActorGobernadoV1(
		context.Background(), revalidador, solicitudRevalidacionVinculoPrueba(autenticacion),
		resolutor, &relojVinculoV2Prueba{ahora: ahora},
	)
	if err != nil || vinculo.ValidarPara(resultado) != nil || !vinculo.VigenteEn(ahora, resultado) {
		t.Fatalf("fábrica gobernada rechazó el par válido: %v", err)
	}
	if revalidador.invocaciones != 1 || resolutor.invocaciones != 1 {
		t.Fatalf("autoridades no invocadas exactamente una vez: auth=%d contexto=%d", revalidador.invocaciones, resolutor.invocaciones)
	}
	if resolutor.autenticacion != autenticacion {
		t.Fatal("el resolutor no recibió la autenticación durable revalidada exacta")
	}
	datos, err := vinculo.Datos()
	if err != nil || datos.Superficie != autenticacion.Superficie ||
		datos.CuentaRef != autenticacion.CuentaRef ||
		datos.MetodoObservado != autenticacion.MetodoObservado ||
		datos.GarantiaObservada != autenticacion.GarantiaObservada {
		t.Fatalf("vínculo no comprometió autenticación gobernada: %+v, %v", datos, err)
	}

	canon := append([]byte(nil), resultado.RepresentacionCanonica...)
	resultadoFuente.RepresentacionCanonica[0] ^= 1
	if !bytes.Equal(resultado.RepresentacionCanonica, canon) || resultado.Validar() != nil {
		t.Fatal("el resultado devuelto comparte memoria mutable con el resolutor")
	}
}

func TestCrearVinculoAutenticacionActorGobernadoV1FallaCerradoEnCrucesYVigencias(t *testing.T) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	baseResultado := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	baseAutenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	for _, caso := range []struct {
		nombre          string
		mutarAuth       func(*AutenticacionRevalidadaV1)
		mutarResultado  func(*ResultadoContextoActorRegistradoV2)
		reloj           time.Time
		esperaResolutor int
	}{
		{"sesión", func(a *AutenticacionRevalidadaV1) { a.SesionRef = "ses_otra234567890abcdefghijkl" }, nil, ahora, 0},
		{"superficie", func(a *AutenticacionRevalidadaV1) { a.Superficie = "anonima" }, nil, ahora, 0},
		{"cuenta", func(a *AutenticacionRevalidadaV1) {
			a.CuentaRef = "cta_otra234567890abcdefghijkl"
			a.CuentaOrdinariaRef = a.CuentaRef
		}, nil, ahora, 1},
		{"método", func(a *AutenticacionRevalidadaV1) { a.MetodoObservado = AuthMethodSSO }, nil, ahora, 1},
		{"garantía", func(a *AutenticacionRevalidadaV1) { a.GarantiaObservada = AuthAssuranceLow }, nil, ahora, 1},
		{"revocación", nil, func(r *ResultadoContextoActorRegistradoV2) {
			r.Contexto.Instantanea.Estado = EstadoVinculoContextoActorRevocado
		}, ahora, 1},
		{"caducidad de contexto", nil, func(r *ResultadoContextoActorRegistradoV2) { r.Contexto.Instantanea.VigenteHasta = ahora }, ahora, 1},
		{"caducidad de sesión", nil, nil, baseAutenticacion.SesionValidaHasta, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			autenticacion := baseAutenticacion
			resultado := baseResultado
			if caso.mutarAuth != nil {
				caso.mutarAuth(&autenticacion)
			}
			if caso.mutarResultado != nil {
				caso.mutarResultado(&resultado)
			}
			revalidador := &revalidadorAutenticacionV2Prueba{resultado: autenticacion}
			resolutor := &resolutorContextoActorGobernadoV1Prueba{resultado: resultado}
			vinculo, recibido, err := CrearVinculoAutenticacionActorGobernadoV1(
				context.Background(), revalidador, solicitudRevalidacionVinculoPrueba(baseAutenticacion),
				resolutor, &relojVinculoV2Prueba{ahora: caso.reloj},
			)
			if !errors.Is(err, ErrVinculoAutenticacionActorV2Invalido) || vinculo.Validar() == nil || recibido.Validar() == nil {
				t.Fatalf("caso %s aceptado: vínculo=%#v resultado=%#v err=%v", caso.nombre, vinculo, recibido, err)
			}
			if revalidador.invocaciones != 1 || resolutor.invocaciones != caso.esperaResolutor {
				t.Fatalf("llamadas inseguras: auth=%d contexto=%d", revalidador.invocaciones, resolutor.invocaciones)
			}
		})
	}
}

func TestCrearVinculoAutenticacionActorGobernadoV1RecompruebaCancelacion(t *testing.T) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	resultado := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	for _, trasAutenticacion := range []bool{true, false} {
		ctx, cancelar := context.WithCancel(context.Background())
		revalidador := &revalidadorAutenticacionV2Prueba{resultado: autenticacion}
		resolutor := &resolutorContextoActorGobernadoV1Prueba{resultado: resultado}
		if trasAutenticacion {
			revalidador.despues = cancelar
		} else {
			resolutor.despues = cancelar
		}
		vinculo, recibido, err := CrearVinculoAutenticacionActorGobernadoV1(
			ctx, revalidador, solicitudRevalidacionVinculoPrueba(autenticacion), resolutor,
			&relojVinculoV2Prueba{ahora: ahora},
		)
		if !errors.Is(err, context.Canceled) || vinculo.Validar() == nil || recibido.Validar() == nil {
			t.Fatalf("cancelación aceptada: vínculo=%#v resultado=%#v err=%v", vinculo, recibido, err)
		}
		esperadasResolutor := 1
		if trasAutenticacion {
			esperadasResolutor = 0
		}
		if revalidador.invocaciones != 1 || resolutor.invocaciones != esperadasResolutor {
			t.Fatalf("cancelación consultó autoridades indebidas: auth=%d contexto=%d", revalidador.invocaciones, resolutor.invocaciones)
		}
	}
}
