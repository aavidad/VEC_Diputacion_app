package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	claveAmbitoRegistroPrueba   = "vec.contratacion-temporal.ambito-idempotencia/v1"
	clavePeticionRegistroPrueba = "vec.contratacion-temporal.huella-peticion/v1"
)

type resolutorIdentidadDoble struct {
	resolver func(context.Context, ports.SolicitudResolverIdentidad) (ports.IdentidadOperacion, error)
}

func (d *resolutorIdentidadDoble) ResolverIdentidadOperacion(
	ctx context.Context,
	solicitud ports.SolicitudResolverIdentidad,
) (ports.IdentidadOperacion, error) {
	return d.resolver(ctx, solicitud)
}

type resolutorFlujoDoble struct {
	resolver func(context.Context, ports.SolicitudResolverFlujo) (ports.ConfiguracionAltaFlujo, error)
}

func (d *resolutorFlujoDoble) ResolverFlujoAlta(
	ctx context.Context,
	solicitud ports.SolicitudResolverFlujo,
) (ports.ConfiguracionAltaFlujo, error) {
	return d.resolver(ctx, solicitud)
}

type derivadorHuellaDoble struct {
	derivar func(context.Context, ports.MaterialHuellaAlta) (string, error)
}

func (d *derivadorHuellaDoble) DerivarHuellaAlta(
	ctx context.Context,
	material ports.MaterialHuellaAlta,
) (string, error) {
	return d.derivar(ctx, material)
}

type preparadorAltaDoble struct {
	preparar func(context.Context, ports.SolicitudPrepararAlta) (ports.PreparacionAlta, error)
}

func (d *preparadorAltaDoble) PrepararAlta(
	ctx context.Context,
	solicitud ports.SolicitudPrepararAlta,
) (ports.PreparacionAlta, error) {
	return d.preparar(ctx, solicitud)
}

type autorizadorAltaDoble struct {
	autorizar func(context.Context, ports.SolicitudAutorizarAlta) (ports.AutorizacionEfecto, error)
}

func (d *autorizadorAltaDoble) AutorizarAltaExpediente(
	ctx context.Context,
	solicitud ports.SolicitudAutorizarAlta,
) (ports.AutorizacionEfecto, error) {
	return d.autorizar(ctx, solicitud)
}

type relojFijo struct {
	instante time.Time
}

func (r *relojFijo) Ahora() time.Time {
	return r.instante
}

type transaccionAltaDoble struct {
	confirmar func(context.Context, ports.OrdenConfirmarAlta) (ports.ReciboAlta, error)
}

func (d *transaccionAltaDoble) ConfirmarAlta(
	ctx context.Context,
	orden ports.OrdenConfirmarAlta,
) (ports.ReciboAlta, error) {
	return d.confirmar(ctx, orden)
}

type escenarioRegistro struct {
	instante      time.Time
	solicitud     SolicitudRegistrarExpediente
	identidad     ports.IdentidadOperacion
	configuracion ports.ConfiguracionAltaFlujo
	preparacion   ports.PreparacionAlta
	autorizacion  ports.AutorizacionEfecto
	recibo        ports.ReciboAlta
}

func nuevoEscenarioRegistro(t *testing.T) escenarioRegistro {
	t.Helper()
	instante := time.Date(2026, time.July, 23, 9, 15, 0, 0, time.UTC)
	identidad, err := ports.NuevaIdentidadOperacion(ports.DatosIdentidadOperacion{
		ActorRef:            "actor:tecnica-rrhh-001",
		CuentaRef:           "cuenta:corporativa-001",
		PerfilRef:           "perfil:tecnica-rrhh",
		ContextoRegistroRef: "contexto:registro-001",
		Superficie:          ports.SuperficieGestionInterna,
		Garantia:            ports.GarantiaAlta,
		ResueltaEn:          instante.Add(-time.Minute),
		ValidaHasta:         instante.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	autorizacion, err := ports.NuevaAutorizacionEfecto(ports.DatosAutorizacionEfecto{
		DecisionRef:  "decision:crear-expediente-001",
		HuellaSHA256: strings.Repeat("c", 64),
		Accion:       ports.AccionCrearSolicitud,
		RecursoRef:   "expediente:ct-2026-0001",
		ActorRef:     "actor:tecnica-rrhh-001",
		PerfilRef:    "perfil:tecnica-rrhh",
		EmitidaEn:    instante,
		ValidaHasta:  instante.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	return escenarioRegistro{
		instante: instante,
		solicitud: SolicitudRegistrarExpediente{
			SesionRef:             "sesion:corporativa-001",
			PerfilRef:             "perfil:tecnica-rrhh",
			CorrelacionRef:        "correlacion:alta-001",
			MotivoAutorizacionRef: "motivo:necesidad-temporal",
			OrganizacionRef:       "organizacion:diputacion-granada",
			ClaveIdempotencia:     "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
			Solicitud: domain.SolicitudCentro{
				CentroRef:     "centro:residencia-rodriguez-penalva",
				ContactoRef:   "persona:responsable-centro-001",
				CategoriaRef:  "categoria:auxiliar-enfermeria",
				GrupoSubgrupo: "C2",
				MotivoClave:   "sustitucion.incapacidad_temporal",
				Detalle:       "Sustitución temporal necesaria para mantener la atención asistencial.",
				Periodo: domain.PeriodoPrevisto{
					Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
					Fin:    time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
				},
				DocumentosAdjuntos: []string{"documento:informe-necesidad-001"},
			},
		},
		identidad: identidad,
		configuracion: ports.ConfiguracionAltaFlujo{
			Flujo: domain.ReferenciaFlujo{
				DefinicionRef: "flujo:contratacion-temporal-general",
				Version:       1,
				HuellaSHA256:  strings.Repeat("a", 64),
			},
			FaseInicial:      "recepcion_solicitud",
			UnidadInicialRef: "unidad:recursos-humanos",
			AccionInicial:    "solicitud.registrada",
		},
		preparacion: ports.PreparacionAlta{
			ReservaRef: "reserva:alta-001",
			Referencias: ports.ReferenciasAlta{
				ExpedienteRef: "expediente:ct-2026-0001",
				NumeroVisible: "2026/CT-0001",
				ReciboRef:     "recibo:alta-001",
			},
			AmbitoIdempotenciaHMAC: selloHMACRegistroPrueba(claveAmbitoRegistroPrueba, "d"),
			HuellaPeticionHMAC:     selloHMACRegistroPrueba(clavePeticionRegistroPrueba, "b"),
			OrganizacionRef:        "organizacion:diputacion-granada",
			ActorRef:               "actor:tecnica-rrhh-001",
			PerfilRef:              "perfil:tecnica-rrhh",
			Estado:                 ports.PreparacionReservada,
		},
		autorizacion: autorizacion,
		recibo: ports.ReciboAlta{
			ExpedienteRef: "expediente:ct-2026-0001",
			NumeroVisible: "2026/CT-0001",
			Version:       1,
			ReciboRef:     "recibo:alta-001",
			AuditoriaRef:  "auditoria:alta-001",
			EventoRef:     "evento:expediente-creado-001",
			ConfirmadaEn:  instante,
		},
	}
}

func construirServicioRegistro(
	t *testing.T,
	escenario escenarioRegistro,
	modificar func(
		*resolutorIdentidadDoble,
		*resolutorFlujoDoble,
		*derivadorHuellaDoble,
		*preparadorAltaDoble,
		*autorizadorAltaDoble,
		*transaccionAltaDoble,
	),
) *ServicioRegistroSolicitud {
	t.Helper()
	identidades := &resolutorIdentidadDoble{
		resolver: func(
			_ context.Context,
			_ ports.SolicitudResolverIdentidad,
		) (ports.IdentidadOperacion, error) {
			return escenario.identidad, nil
		},
	}
	flujos := &resolutorFlujoDoble{
		resolver: func(
			_ context.Context,
			_ ports.SolicitudResolverFlujo,
		) (ports.ConfiguracionAltaFlujo, error) {
			return escenario.configuracion, nil
		},
	}
	huellas := &derivadorHuellaDoble{
		derivar: func(
			_ context.Context,
			_ ports.MaterialHuellaAlta,
		) (string, error) {
			return selloHMACRegistroPrueba(clavePeticionRegistroPrueba, "b"), nil
		},
	}
	preparaciones := &preparadorAltaDoble{
		preparar: func(
			_ context.Context,
			_ ports.SolicitudPrepararAlta,
		) (ports.PreparacionAlta, error) {
			return escenario.preparacion, nil
		},
	}
	autorizador := &autorizadorAltaDoble{
		autorizar: func(
			_ context.Context,
			_ ports.SolicitudAutorizarAlta,
		) (ports.AutorizacionEfecto, error) {
			return escenario.autorizacion, nil
		},
	}
	transaccion := &transaccionAltaDoble{
		confirmar: func(
			_ context.Context,
			orden ports.OrdenConfirmarAlta,
		) (ports.ReciboAlta, error) {
			if _, err := orden.Datos(); err != nil {
				t.Fatalf("orden no válida: %v", err)
			}
			return escenario.recibo, nil
		},
	}
	if modificar != nil {
		modificar(identidades, flujos, huellas, preparaciones, autorizador, transaccion)
	}
	servicio, err := NuevoServicioRegistroSolicitud(
		identidades, flujos, huellas, preparaciones, autorizador,
		&relojFijo{instante: escenario.instante}, transaccion,
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}

func selloHMACRegistroPrueba(dominio, caracter string) string {
	return "hmac-sha256:" + dominio + ":" + strings.Repeat(caracter, 64)
}

func TestRegistroSolicitudConfirmaExpedienteAutorizado(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio := construirServicioRegistro(t, escenario, nil)

	recibo, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if recibo != escenario.recibo {
		t.Fatalf("recibo distinto: %#v", recibo)
	}
}

func TestRegistroSolicitudReintentoConfirmadoDevuelveMismoRecibo(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	confirmado := escenario.recibo
	escenario.preparacion.Estado = ports.PreparacionConfirmada
	escenario.preparacion.ReciboConfirmado = &confirmado
	llamadasAutorizacion, llamadasTransaccion := 0, 0
	servicio := construirServicioRegistro(t, escenario, func(
		_ *resolutorIdentidadDoble,
		_ *resolutorFlujoDoble,
		_ *derivadorHuellaDoble,
		_ *preparadorAltaDoble,
		autorizador *autorizadorAltaDoble,
		transaccion *transaccionAltaDoble,
	) {
		autorizador.autorizar = func(
			context.Context,
			ports.SolicitudAutorizarAlta,
		) (ports.AutorizacionEfecto, error) {
			llamadasAutorizacion++
			return ports.AutorizacionEfecto{}, errors.New("no debe autorizar")
		}
		transaccion.confirmar = func(
			context.Context,
			ports.OrdenConfirmarAlta,
		) (ports.ReciboAlta, error) {
			llamadasTransaccion++
			return ports.ReciboAlta{}, errors.New("no debe confirmar")
		}
	})

	recibo, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if recibo != escenario.recibo || llamadasAutorizacion != 0 || llamadasTransaccion != 0 {
		t.Fatalf(
			"reintento no estable: recibo=%#v autorizaciones=%d transacciones=%d",
			recibo,
			llamadasAutorizacion,
			llamadasTransaccion,
		)
	}
}

func TestRegistroSolicitudDeniegaAntesDePersistir(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	llamadasTransaccion := 0
	servicio := construirServicioRegistro(t, escenario, func(
		_ *resolutorIdentidadDoble,
		_ *resolutorFlujoDoble,
		_ *derivadorHuellaDoble,
		_ *preparadorAltaDoble,
		autorizador *autorizadorAltaDoble,
		transaccion *transaccionAltaDoble,
	) {
		autorizador.autorizar = func(
			context.Context,
			ports.SolicitudAutorizarAlta,
		) (ports.AutorizacionEfecto, error) {
			return ports.AutorizacionEfecto{}, errors.New("PDP deniega")
		}
		transaccion.confirmar = func(
			context.Context,
			ports.OrdenConfirmarAlta,
		) (ports.ReciboAlta, error) {
			llamadasTransaccion++
			return ports.ReciboAlta{}, nil
		}
	})

	_, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ports.ErrAutorizacionDenegada) {
		t.Fatalf("error inesperado: %v", err)
	}
	if llamadasTransaccion != 0 {
		t.Fatal("se intentó persistir una operación denegada")
	}
}

func TestRegistroSolicitudRechazaReciboNoConfiable(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio := construirServicioRegistro(t, escenario, func(
		_ *resolutorIdentidadDoble,
		_ *resolutorFlujoDoble,
		_ *derivadorHuellaDoble,
		_ *preparadorAltaDoble,
		_ *autorizadorAltaDoble,
		transaccion *transaccionAltaDoble,
	) {
		transaccion.confirmar = func(
			context.Context,
			ports.OrdenConfirmarAlta,
		) (ports.ReciboAlta, error) {
			adulterado := escenario.recibo
			adulterado.ExpedienteRef = "expediente:otro-0001"
			return adulterado, nil
		}
	})

	_, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ErrResultadoRegistroNoConfiable) {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestRegistroSolicitudAislaMutacionDelDerivador(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio := construirServicioRegistro(t, escenario, func(
		_ *resolutorIdentidadDoble,
		_ *resolutorFlujoDoble,
		huellas *derivadorHuellaDoble,
		_ *preparadorAltaDoble,
		_ *autorizadorAltaDoble,
		_ *transaccionAltaDoble,
	) {
		huellas.derivar = func(
			_ context.Context,
			material ports.MaterialHuellaAlta,
		) (string, error) {
			material.Solicitud.DocumentosAdjuntos[0] = "documento:adulterado-001"
			return selloHMACRegistroPrueba(clavePeticionRegistroPrueba, "b"), nil
		}
	})

	if _, err := servicio.Registrar(context.Background(), escenario.solicitud); err != nil {
		t.Fatal(err)
	}
	if escenario.solicitud.Solicitud.DocumentosAdjuntos[0] !=
		"documento:informe-necesidad-001" {
		t.Fatal("el puerto pudo mutar la solicitud aportada")
	}
}

func TestConstructorRegistroRechazaInterfazConPunteroNulo(t *testing.T) {
	var identidades *resolutorIdentidadDoble
	escenario := nuevoEscenarioRegistro(t)
	_, err := NuevoServicioRegistroSolicitud(
		identidades,
		&resolutorFlujoDoble{},
		&derivadorHuellaDoble{},
		&preparadorAltaDoble{},
		&autorizadorAltaDoble{},
		&relojFijo{instante: escenario.instante},
		&transaccionAltaDoble{},
	)
	if !errors.Is(err, ErrServicioRegistroInvalido) {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestRegistroSolicitudRespetaContextoCancelado(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio := construirServicioRegistro(t, escenario, nil)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()

	_, err := servicio.Registrar(ctx, escenario.solicitud)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestRegistroSolicitudNoConvierteCommitConfirmadoEnCancelacionTardia(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	ctx, cancelar := context.WithCancel(context.Background())
	servicio := construirServicioRegistro(t, escenario, func(
		_ *resolutorIdentidadDoble,
		_ *resolutorFlujoDoble,
		_ *derivadorHuellaDoble,
		_ *preparadorAltaDoble,
		_ *autorizadorAltaDoble,
		transaccion *transaccionAltaDoble,
	) {
		transaccion.confirmar = func(
			context.Context,
			ports.OrdenConfirmarAlta,
		) (ports.ReciboAlta, error) {
			cancelar()
			return escenario.recibo, nil
		}
	})

	recibo, err := servicio.Registrar(ctx, escenario.solicitud)
	if err != nil {
		t.Fatalf("éxito durable convertido en error: %v", err)
	}
	if recibo != escenario.recibo {
		t.Fatalf("recibo distinto: %#v", recibo)
	}
}
