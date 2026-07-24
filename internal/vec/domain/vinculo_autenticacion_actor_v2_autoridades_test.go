package domain

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCrearVinculoAutenticacionActorV2FallaCerradoEnAutoridadesYMezclas(t *testing.T) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	resultado := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	solicitudAuth := solicitudRevalidacionVinculoPrueba(autenticacion)
	solicitudActor := solicitudContextoVinculoV2Prueba(resultado)
	errFuente := errors.New("fuente no disponible")
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	casos := []struct {
		nombre      string
		ctx         context.Context
		revalidador *revalidadorAutenticacionV2Prueba
		resolutor   *resolutorContextoRegistradoV2Prueba
		reloj       *relojVinculoV2Prueba
		solicitudA  SolicitudRevalidacionAutenticacionActorV1
		solicitudC  SolicitudContextoActor
	}{
		{"contexto nil", nil, &revalidadorAutenticacionV2Prueba{resultado: autenticacion}, &resolutorContextoRegistradoV2Prueba{resultado: resultado}, &relojVinculoV2Prueba{ahora}, solicitudAuth, solicitudActor},
		{"cancelado", cancelado, &revalidadorAutenticacionV2Prueba{resultado: autenticacion}, &resolutorContextoRegistradoV2Prueba{resultado: resultado}, &relojVinculoV2Prueba{ahora}, solicitudAuth, solicitudActor},
		{"auth falla", context.Background(), &revalidadorAutenticacionV2Prueba{err: errFuente}, &resolutorContextoRegistradoV2Prueba{resultado: resultado}, &relojVinculoV2Prueba{ahora}, solicitudAuth, solicitudActor},
		{"contexto falla", context.Background(), &revalidadorAutenticacionV2Prueba{resultado: autenticacion}, &resolutorContextoRegistradoV2Prueba{err: errFuente}, &relojVinculoV2Prueba{ahora}, solicitudAuth, solicitudActor},
		{"solicitud auth", context.Background(), &revalidadorAutenticacionV2Prueba{resultado: autenticacion}, &resolutorContextoRegistradoV2Prueba{resultado: resultado}, &relojVinculoV2Prueba{ahora}, SolicitudRevalidacionAutenticacionActorV1{}, solicitudActor},
		{"solicitud contexto", context.Background(), &revalidadorAutenticacionV2Prueba{resultado: autenticacion}, &resolutorContextoRegistradoV2Prueba{resultado: resultado}, &relojVinculoV2Prueba{ahora}, solicitudAuth, SolicitudContextoActor{}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			vinculo, err := CrearVinculoAutenticacionActorV2(
				caso.ctx, caso.revalidador, caso.solicitudA, caso.resolutor, caso.solicitudC, caso.reloj,
			)
			if err == nil || vinculo.Validar() == nil {
				t.Fatalf("caso inseguro aceptado: %#v, %v", vinculo, err)
			}
			if caso.ctx == nil || caso.ctx.Err() != nil || caso.solicitudA.Validar() != nil || caso.solicitudC.Validar() != nil {
				if caso.revalidador.invocaciones != 0 || caso.resolutor.invocaciones != 0 {
					t.Fatalf("autoridad consultada con precondicion invalida: auth=%d contexto=%d",
						caso.revalidador.invocaciones, caso.resolutor.invocaciones)
				}
			}
		})
	}

	for _, mutar := range []func(*AutenticacionRevalidadaV1){
		func(a *AutenticacionRevalidadaV1) { a.CuentaRef = "cta_otra234567890abcdefghijkl" },
		func(a *AutenticacionRevalidadaV1) { a.MetodoObservado = AuthMethodSSO },
		func(a *AutenticacionRevalidadaV1) { a.GarantiaObservada = AuthAssuranceLow },
		func(a *AutenticacionRevalidadaV1) { a.SesionRef = "ses_otra234567890abcdefghijkl" },
	} {
		adulterada := autenticacion
		mutar(&adulterada)
		vinculo, err := CrearVinculoAutenticacionActorV2(
			context.Background(), &revalidadorAutenticacionV2Prueba{resultado: adulterada},
			solicitudAuth, &resolutorContextoRegistradoV2Prueba{resultado: resultado}, solicitudActor,
			&relojVinculoV2Prueba{ahora},
		)
		if err == nil || vinculo.Validar() == nil {
			t.Fatalf("mezcla de autoridad aceptada: %+v", adulterada)
		}
	}
}

func TestCrearVinculoAutenticacionActorV2RechazaNulosTipados(t *testing.T) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	resultado := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	var revalidador *revalidadorAutenticacionV2Prueba
	var resolutor *resolutorContextoRegistradoV2Prueba
	var reloj *relojVinculoV2Prueba
	casos := []struct {
		revalidador RevalidadorAutenticacionActorV1
		resolutor   ResolutorContextoActorRegistradoV2
		reloj       RelojVinculoAutenticacionActorV2
	}{
		{revalidador, &resolutorContextoRegistradoV2Prueba{resultado: resultado}, &relojVinculoV2Prueba{ahora}},
		{&revalidadorAutenticacionV2Prueba{resultado: autenticacion}, resolutor, &relojVinculoV2Prueba{ahora}},
		{&revalidadorAutenticacionV2Prueba{resultado: autenticacion}, &resolutorContextoRegistradoV2Prueba{resultado: resultado}, reloj},
	}
	for i, caso := range casos {
		vinculo, err := CrearVinculoAutenticacionActorV2(
			context.Background(), caso.revalidador, solicitudRevalidacionVinculoPrueba(autenticacion),
			caso.resolutor, solicitudContextoVinculoV2Prueba(resultado), caso.reloj,
		)
		if !errors.Is(err, ErrVinculoAutenticacionActorV2Invalido) || vinculo.Validar() == nil {
			t.Fatalf("nulo tipado %d aceptado: %v", i, err)
		}
	}
}

func TestCrearVinculoAutenticacionActorV2RechazaResultadoAjenoALaSolicitud(t *testing.T) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	resultado := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	base := solicitudContextoVinculoV2Prueba(resultado)
	casos := []struct {
		nombre string
		mutar  func(*SolicitudContextoActor)
	}{
		{"cuenta", func(s *SolicitudContextoActor) {
			s.Cuenta.CuentaRef = "cta_otra234567890abcdefghijkl"
		}},
		{"perfil", func(s *SolicitudContextoActor) {
			s.PerfilActivoRef = "prf_otra234567890abcdefghijkl"
		}},
		{"metodo", func(s *SolicitudContextoActor) {
			s.Cuenta.Metodo = AuthMethodSSO
		}},
		{"garantia", func(s *SolicitudContextoActor) {
			s.Cuenta.Garantia = AuthAssuranceLow
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := base
			caso.mutar(&solicitud)
			if solicitud.Validar() != nil {
				t.Fatalf("fixture de solicitud ajena invalida: %#v", solicitud)
			}
			resolutor := &resolutorContextoRegistradoV2Prueba{resultado: resultado}
			vinculo, err := CrearVinculoAutenticacionActorV2(
				context.Background(), &revalidadorAutenticacionV2Prueba{resultado: autenticacion},
				solicitudRevalidacionVinculoPrueba(autenticacion), resolutor, solicitud,
				&relojVinculoV2Prueba{ahora},
			)
			if !errors.Is(err, ErrVinculoAutenticacionActorV2Invalido) || vinculo.Validar() == nil {
				t.Fatalf("resolutor que ignoro %s aceptado: vinculo=%#v error=%v", caso.nombre, vinculo, err)
			}
			if resolutor.invocaciones != 1 {
				t.Fatalf("la autoridad no fue consultada exactamente una vez: %d", resolutor.invocaciones)
			}
		})
	}
}

func TestCrearVinculoAutenticacionActorV2RecompruebaCancelacion(t *testing.T) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	resultado := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	for nombre, cancelarEnAuth := range map[string]bool{"tras autenticacion": true, "tras contexto": false} {
		t.Run(nombre, func(t *testing.T) {
			ctx, cancelar := context.WithCancel(context.Background())
			revalidador := &revalidadorAutenticacionV2Prueba{resultado: autenticacion}
			resolutor := &resolutorContextoRegistradoV2Prueba{resultado: resultado}
			if cancelarEnAuth {
				revalidador.despues = cancelar
			} else {
				resolutor.despues = cancelar
			}
			vinculo, err := CrearVinculoAutenticacionActorV2(
				ctx, revalidador, solicitudRevalidacionVinculoPrueba(autenticacion), resolutor,
				solicitudContextoVinculoV2Prueba(resultado), &relojVinculoV2Prueba{ahora},
			)
			if !errors.Is(err, context.Canceled) || vinculo.Validar() == nil {
				t.Fatalf("cancelacion no cerro fabrica: %v", err)
			}
		})
	}
}

func TestVinculoAutenticacionActorV2AplicaVentanasHalfOpen(t *testing.T) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	vinculo, resultado, autenticacion := vinculoAutenticacionActorV2Prueba(t, ahora)
	if !vinculo.VigenteEn(ahora, resultado) ||
		vinculo.VigenteEn(autenticacion.SesionValidaHasta, resultado) ||
		vinculo.VigenteEn(resultado.Contexto.Instantanea.VigenteHasta, resultado) ||
		vinculo.VigenteEn(ahora.Add(time.Nanosecond), resultado) {
		t.Fatal("ventanas o precision no son half-open canonicas")
	}

	for _, instante := range []time.Time{
		resultado.ResueltoEnAutoritativo.Add(-time.Microsecond),
		autenticacion.SesionValidaHasta,
		resultado.Contexto.Instantanea.VigenteHasta,
	} {
		v, err := CrearVinculoAutenticacionActorV2(
			context.Background(), &revalidadorAutenticacionV2Prueba{resultado: autenticacion},
			solicitudRevalidacionVinculoPrueba(autenticacion),
			&resolutorContextoRegistradoV2Prueba{resultado: resultado}, solicitudContextoVinculoV2Prueba(resultado),
			&relojVinculoV2Prueba{instante},
		)
		if err == nil || v.Validar() == nil {
			t.Fatalf("instante de borde aceptado: %s", instante)
		}
	}
}

func vinculoAutenticacionActorV2Prueba(
	t *testing.T,
	ahora time.Time,
) (VinculoAutenticacionActorV2, ResultadoContextoActorRegistradoV2, AutenticacionRevalidadaV1) {
	t.Helper()
	resultado := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	vinculo, err := CrearVinculoAutenticacionActorV2(
		context.Background(), &revalidadorAutenticacionV2Prueba{resultado: autenticacion},
		solicitudRevalidacionVinculoPrueba(autenticacion),
		&resolutorContextoRegistradoV2Prueba{resultado: resultado}, solicitudContextoVinculoV2Prueba(resultado),
		&relojVinculoV2Prueba{ahora},
	)
	if err != nil {
		t.Fatalf("crear vinculo V2: %v", err)
	}
	return vinculo, resultado, autenticacion
}

func resultadoContextoActorRegistradoV2Prueba(t *testing.T, ahora time.Time) ResultadoContextoActorRegistradoV2 {
	t.Helper()
	actor := contextoActorVinculoPrueba(t, ahora)
	actor.Instantanea.CuentaVersion = 7
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:          "prc_0123456789abcdefghijkl",
		ProcedenciaVersion:      1,
		ProcedenciaHuellaSHA256: strings.Repeat("a", 64),
		ProcedenciaAutoridad:    AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := ManifiestoProcedenciaContextoActorV1{
		Esquema:           EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: ProcedenciaCuentaContextoActorV1{CuentaRef: actor.Instantanea.CuentaRef,
			Version: actor.Instantanea.CuentaVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Persona: ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef,
			Version: actor.Instantanea.PersonaVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Perfil: ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef,
			Version: actor.Instantanea.PerfilVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Contexto: ProcedenciaVinculoContextoActorV1{VinculoRef: actor.Instantanea.VinculoRef,
			Version: actor.Instantanea.VinculoVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Vinculos: []ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	bytesManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := HuellaSHA256ManifiestoProcedenciaContextoActorV1(bytesManifiesto)
	if err != nil {
		t.Fatal(err)
	}
	return ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn", Contexto: actor,
		RepresentacionCanonica: representacion, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico:     bytesManifiesto,
		ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva:                 AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
}

func solicitudContextoVinculoV2Prueba(r ResultadoContextoActorRegistradoV2) SolicitudContextoActor {
	return SolicitudContextoActor{Cuenta: CuentaAutenticadaContextoActor{
		CuentaRef: r.Contexto.Instantanea.CuentaRef, Metodo: r.Contexto.Principal.AuthMethod,
		Garantia: r.Contexto.Principal.AuthAssurance,
	}, PerfilActivoRef: r.Contexto.PerfilActivoRef}
}

func instanteVinculoAutenticacionActorV2Prueba() time.Time {
	return time.Date(2026, 7, 15, 10, 0, 0, 123_456_000, time.UTC)
}
