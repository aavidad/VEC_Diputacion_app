package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestCierreAdministrativoRechazaPrincipalV3CruzadoAntesDelCallback(t *testing.T) {
	probarPreparacionCierreAdministrativoRechazada(t, func(
		p *ports.PreparacionTransaccionCierreAdministrativo,
	) {
		reconstruirAutorizacionCierreAdministrativoPrueba(t, p, func(
			d *dominiovec.DatosSolicitudAutorizacionLigadaV3,
		) {
			d.Recurso.Atributos["principal_v3_ref"] = "per_abcdefghijkl0123456789"
		})
	})
}

func TestCierreAdministrativoRechazaActorSeguimientoCruzadoAntesDelCallback(t *testing.T) {
	probarPreparacionCierreAdministrativoRechazada(t, func(
		p *ports.PreparacionTransaccionCierreAdministrativo,
	) {
		p.ActorRef = referenciaCierreAdministrativoPrueba("actor_ajeno_01")
	})
}

func TestCierreAdministrativoRechazaActorDeReciboCruzado(t *testing.T) {
	definicion, seguimiento := escenarioSeguimientoVigente(t)
	solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
	transaccion := &transaccionCierreAdministrativoPrueba{
		preparacion: preparacionCierreAdministrativoPrueba(
			t, solicitudTransaccionalCierrePrueba(solicitud), definicion,
			seguimiento, instanteCierreAdministrativoPrueba,
		),
		actorResultadoRef: referenciaCierreAdministrativoPrueba("actor_recibo_ajeno_01"),
	}
	resultado, err := nuevoServicioCierreAdministrativoPrueba(t, transaccion).
		Cerrar(context.Background(), solicitud)
	if !errors.Is(err, ErrResultadoCierreAdministrativoInvalido) ||
		resultado.VersionSeguimiento() != 0 || resultado.ReciboRef() != "" ||
		resultado.EsReplayConfirmado() {
		t.Fatalf("se publicó un recibo con actor ajeno: resultado=%#v error=%v", resultado, err)
	}
}

func TestCierreAdministrativoRechazaPerfilV3CruzadoOAusente(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		perfil string
	}{
		{"cruzado", "prf_abcdefghijkl0123456789"},
		{"ausente", ""},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			probarPreparacionCierreAdministrativoRechazada(t, func(
				p *ports.PreparacionTransaccionCierreAdministrativo,
			) {
				p.PerfilRef = caso.perfil
			})
		})
	}
}

func TestCierreAdministrativoRechazaAccionV3Cruzada(t *testing.T) {
	probarPreparacionCierreAdministrativoRechazada(t, func(
		p *ports.PreparacionTransaccionCierreAdministrativo,
	) {
		reconstruirAutorizacionCierreAdministrativoPrueba(t, p, func(
			d *dominiovec.DatosSolicitudAutorizacionLigadaV3,
		) {
			d.Accion = "contratacion_temporal.seguimiento.consultar"
		})
	})
}

func TestCierreAdministrativoRechazaFinalidadV3Cruzada(t *testing.T) {
	probarPreparacionCierreAdministrativoRechazada(t, func(
		p *ports.PreparacionTransaccionCierreAdministrativo,
	) {
		reconstruirAutorizacionCierreAdministrativoPrueba(t, p, func(
			d *dominiovec.DatosSolicitudAutorizacionLigadaV3,
		) {
			d.Finalidad = "consultar_seguimiento_contratacion_temporal"
		})
	})
}

func TestCierreAdministrativoRechazaMotivoV3Ausente(t *testing.T) {
	probarPreparacionCierreAdministrativoRechazada(t, func(
		p *ports.PreparacionTransaccionCierreAdministrativo,
	) {
		p.MotivoAutorizacionV3 = dominiovec.ReferenciaEntradaCatalogo{}
	})
}

func TestCierreAdministrativoRechazaCorrelacionV3CruzadaAntesDelCallback(t *testing.T) {
	probarPreparacionCierreAdministrativoRechazada(t, func(
		p *ports.PreparacionTransaccionCierreAdministrativo,
	) {
		reconstruirAutorizacionCierreAdministrativoPrueba(t, p, func(
			d *dominiovec.DatosSolicitudAutorizacionLigadaV3,
		) {
			correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
				context.Background(),
				generadorCorrelacionCierreAdministrativoPrueba{
					valor: "correlacion_0123456789abcdef0123456789abcdef",
				},
			)
			if err != nil {
				t.Fatalf("generar correlación V3 cruzada: %v", err)
			}
			d.Correlacion = correlacion
		})
	})
}

func TestCierreAdministrativoRechazaCorrelacionSeguimientoCruzadaAntesDelCallback(t *testing.T) {
	probarPreparacionCierreAdministrativoRechazada(t, func(
		p *ports.PreparacionTransaccionCierreAdministrativo,
	) {
		p.CorrelacionRef = referenciaCierreAdministrativoPrueba(
			"correlacion_ajena_01",
		)
	})
}

func TestCierreAdministrativoRechazaGarantiaV3Insuficiente(t *testing.T) {
	probarPreparacionCierreAdministrativoRechazada(t, func(
		p *ports.PreparacionTransaccionCierreAdministrativo,
	) {
		p.ContextoAutorizacionV3 = contextoCierreAdministrativoConGarantiaPrueba(
			t,
			p.ContextoAutorizacionV3,
			dominiovec.AuthAssuranceSubstantial,
		)
		reconstruirAutorizacionCierreAdministrativoPrueba(t, p, nil)
	})
}

func TestCierreAdministrativoRechazaVigenciaV3Agotada(t *testing.T) {
	probarPreparacionCierreAdministrativoRechazada(t, func(
		p *ports.PreparacionTransaccionCierreAdministrativo,
	) {
		p.RegistradaEn = instanteCierreAdministrativoPrueba.Add(2 * time.Hour)
		p.EfectivoEn = p.RegistradaEn
	})
}

func TestCierreAdministrativoRechazaConfirmacionDurableCruzadaOAusente(t *testing.T) {
	for _, caso := range []struct {
		nombre    string
		modificar func(*testing.T, *ports.PreparacionTransaccionCierreAdministrativo)
	}{
		{
			nombre: "cruzada",
			modificar: func(t *testing.T, p *ports.PreparacionTransaccionCierreAdministrativo) {
				otra := *p
				reconstruirAutorizacionCierreAdministrativoPrueba(t, &otra, func(
					d *dominiovec.DatosSolicitudAutorizacionLigadaV3,
				) {
					d.Accion = "contratacion_temporal.seguimiento.consultar"
				})
				p.ConfirmacionAutorizacionV3 = otra.ConfirmacionAutorizacionV3
			},
		},
		{
			nombre: "ausente",
			modificar: func(_ *testing.T, p *ports.PreparacionTransaccionCierreAdministrativo) {
				p.ConfirmacionAutorizacionV3 = ports.PreparacionTransaccionCierreAdministrativo{}.
					ConfirmacionAutorizacionV3
			},
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			probarPreparacionCierreAdministrativoRechazada(t, func(
				p *ports.PreparacionTransaccionCierreAdministrativo,
			) {
				caso.modificar(t, p)
			})
		})
	}
}

func TestCierreAdministrativoCancelaTrasCalcularAntesDeConfirmar(t *testing.T) {
	definicion, seguimiento := escenarioSeguimientoVigente(t)
	solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
	ctx, cancelar := context.WithCancel(context.Background())
	transaccion := &transaccionCierreAdministrativoPrueba{
		preparacion: preparacionCierreAdministrativoPrueba(
			t, solicitudTransaccionalCierrePrueba(solicitud), definicion,
			seguimiento, instanteCierreAdministrativoPrueba,
		),
		despuesDeAplicar: cancelar,
	}
	servicio := nuevoServicioCierreAdministrativoPrueba(t, transaccion)
	resultado, err := servicio.Cerrar(ctx, solicitud)
	if !errors.Is(err, context.Canceled) || transaccion.confirmada ||
		transaccion.aplicaciones != 1 ||
		transaccion.siguiente.Version() != seguimiento.Version()+1 ||
		resultado.VersionSeguimiento() != 0 || resultado.ReciboRef() != "" ||
		resultado.EsReplayConfirmado() {
		t.Fatalf("cancelación tras calcular produjo falso éxito: resultado=%#v error=%v", resultado, err)
	}

	transaccion.despuesDeAplicar = nil
	resultado, err = servicio.Cerrar(context.Background(), solicitud)
	if err != nil || resultado.EsReplayConfirmado() || !transaccion.confirmada ||
		transaccion.aplicaciones != 2 {
		t.Fatalf("la cancelación se convirtió en replay confirmado: resultado=%#v error=%v", resultado, err)
	}
}

func probarPreparacionCierreAdministrativoRechazada(
	t *testing.T,
	modificar func(*ports.PreparacionTransaccionCierreAdministrativo),
) {
	t.Helper()
	definicion, seguimiento := escenarioSeguimientoVigente(t)
	solicitud := solicitudCerrarAdministrativamentePrueba(seguimiento.Version())
	preparacion := preparacionCierreAdministrativoPrueba(
		t, solicitudTransaccionalCierrePrueba(solicitud), definicion,
		seguimiento, instanteCierreAdministrativoPrueba,
	)
	modificar(&preparacion)
	transaccion := &transaccionCierreAdministrativoPrueba{preparacion: preparacion}
	resultado, err := nuevoServicioCierreAdministrativoPrueba(t, transaccion).
		Cerrar(context.Background(), solicitud)
	if !errors.Is(err, ErrCierreAdministrativoNoPermitido) ||
		transaccion.aplicaciones != 0 || transaccion.confirmada ||
		resultado.VersionSeguimiento() != 0 ||
		resultado.ReciboRef() != "" || resultado.EsReplayConfirmado() {
		t.Fatalf("autoridad cruzada o ausente llegó al callback: aplicaciones=%d resultado=%#v error=%v",
			transaccion.aplicaciones, resultado, err)
	}
}

func reconstruirAutorizacionCierreAdministrativoPrueba(
	t *testing.T,
	preparacion *ports.PreparacionTransaccionCierreAdministrativo,
	modificar func(*dominiovec.DatosSolicitudAutorizacionLigadaV3),
) {
	t.Helper()
	datos, err := preparacion.SolicitudAutorizacionV3.Datos()
	if err != nil {
		t.Fatalf("extraer solicitud V3 para regresión: %v", err)
	}
	datos.VinculoAutenticacionActor = preparacion.ContextoAutorizacionV3.Vinculo
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatalf("extraer vínculo V3 para regresión: %v", err)
	}
	correlacion, err := datos.Correlacion.ValorCanonico()
	if err != nil {
		t.Fatalf("extraer correlación V3 para regresión: %v", err)
	}
	datos.Recurso.Atributos["principal_v3_ref"] = vinculo.PrincipalID
	datos.Recurso.Atributos["actor_seguimiento_ref"] = preparacion.ActorRef
	datos.Recurso.Atributos["correlacion_v3_ref"] = correlacion
	datos.Recurso.Atributos["correlacion_seguimiento_ref"] = preparacion.CorrelacionRef
	if modificar != nil {
		modificar(&datos)
	}
	solicitudV3, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		t.Fatalf("reconstruir solicitud V3 para regresión: %v", err)
	}
	decision, confirmacion, err := concesionAutorizacionV3Prueba(
		t, solicitudV3, preparacion.ContextoAutorizacionV3.Resultado,
		datos.ReferenciaMotivo, instanteCierreAdministrativoPrueba,
		"decision:cierre-administrativo:regresion", true,
	)
	if err != nil {
		t.Fatalf("reconstruir concesión V3 para regresión: %v", err)
	}
	preparacion.SolicitudAutorizacionV3 = solicitudV3
	preparacion.DecisionAutorizacionV3 = decision
	preparacion.ConfirmacionAutorizacionV3 = confirmacion
	preparacion.MotivoAutorizacionV3 = datos.ReferenciaMotivo
	preparacion.PerfilRef = vinculo.PerfilActivoRef
}

func contextoCierreAdministrativoConGarantiaPrueba(
	t *testing.T,
	base ports.ContextoAutorizacionAltaV3,
	garantia dominiovec.AuthAssurance,
) ports.ContextoAutorizacionAltaV3 {
	t.Helper()
	datosVinculo, err := base.Vinculo.Datos()
	if err != nil {
		t.Fatalf("extraer vínculo base para garantía: %v", err)
	}
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: base.Resultado.Contexto.Instantanea.CuentaRef,
		Metodo:    base.Resultado.Contexto.Principal.AuthMethod,
		Garantia:  garantia,
	}
	actor, err := dominiovec.NuevoContextoActor(
		cuenta,
		base.Resultado.Contexto.Instantanea,
		base.Resultado.Contexto.ResueltoEn,
	)
	if err != nil {
		t.Fatalf("crear contexto con garantía insuficiente: %v", err)
	}
	resultado := base.Resultado
	resultado.Contexto = actor
	resultado.RepresentacionCanonica, err = actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatalf("canonizar contexto con garantía insuficiente: %v", err)
	}
	resultado.HuellaSHA256, err = actor.HuellaSHA256VinculadaV2()
	if err != nil || resultado.Validar() != nil {
		t.Fatalf("validar contexto con garantía insuficiente: %v", err)
	}
	autenticacion := datosVinculo.Autenticacion()
	autenticacion.GarantiaObservada = garantia
	vinculo, resultadoVinculado, err := dominiovec.CrearVinculoAutenticacionActorV2ConResultado(
		context.Background(),
		revalidadorVinculoPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		resolutorResultadoVinculoPrueba{resultado: resultado},
		dominiovec.SolicitudContextoActor{
			Cuenta:          cuenta,
			PerfilActivoRef: actor.PerfilActivoRef,
		},
		relojVinculoPrueba{instante: instanteCierreAdministrativoPrueba},
	)
	if err != nil {
		t.Fatalf("ligar contexto con garantía insuficiente: %v", err)
	}
	return ports.ContextoAutorizacionAltaV3{
		Vinculo: vinculo, Resultado: resultadoVinculado,
	}
}
