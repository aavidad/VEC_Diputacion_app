package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type selladorAmbitoAsignacionDoble struct {
	coleccion ports.ColeccionSellosHMAC
	llamadas  int
}

func (d *selladorAmbitoAsignacionDoble) SellarAmbitoAsignacion(
	_ context.Context,
	_ ports.SolicitudSellarAmbitoIdempotencia,
) (ports.ColeccionSellosHMAC, error) {
	d.llamadas++
	return d.coleccion, nil
}

type derivadorHuellaAsignacionDoble struct {
	coleccion ports.ColeccionSellosHMAC
	llamadas  int
}

func (d *derivadorHuellaAsignacionDoble) DerivarHuellaAsignacion(
	_ context.Context,
	material ports.MaterialHuellaAsignacion,
) (ports.ColeccionSellosHMAC, error) {
	d.llamadas++
	if material.Validar() != nil {
		return ports.ColeccionSellosHMAC{},
			ports.ErrPreparacionAsignacionInvalida
	}
	return d.coleccion, nil
}

type preparadorAsignacionDoble struct {
	preparacion ports.PreparacionAsignacion
	llamadas    int
}

func (d *preparadorAsignacionDoble) PrepararAsignacion(
	_ context.Context,
	_ ports.SolicitudPrepararAsignacion,
) (ports.PreparacionAsignacion, error) {
	d.llamadas++
	return d.preparacion, nil
}

type resolutorDestinoAsignacionDoble struct {
	destino   ports.DestinoAsignacionResuelto
	adulterar bool
	llamadas  int
}

func (d *resolutorDestinoAsignacionDoble) ResolverDestinoAsignacion(
	_ context.Context,
	solicitud ports.SolicitudResolverDestinoAsignacion,
) (ports.DestinoAsignacionResuelto, error) {
	d.llamadas++
	destino := d.destino
	destino.OrganizacionRef = solicitud.OrganizacionRef
	destino.ExpedienteRef = solicitud.ExpedienteRef
	destino.VersionExpediente = solicitud.VersionExpediente
	destino.ActorRef = solicitud.ActorRef
	destino.UnidadRef = solicitud.UnidadRef
	destino.ResponsableRef = solicitud.ResponsableRef
	destino.EvaluadoEn = solicitud.Instante
	destino.ValidoHasta = solicitud.Instante.Add(5 * time.Minute)
	if d.adulterar {
		destino.ResponsableRef = "persona:ajena:0123456789abcdef"
	}
	return destino, nil
}

type resolutorPoliticaAsignacionDoble struct {
	motivo   dominiovec.ReferenciaEntradaCatalogo
	llamadas int
}

func (d *resolutorPoliticaAsignacionDoble) ResolverPoliticaAsignacion(
	_ context.Context,
	solicitud ports.SolicitudResolverPoliticaAsignacion,
) (ports.PoliticaAsignacion, error) {
	d.llamadas++
	return ports.PoliticaAsignacion{
		Operacion:                    solicitud.Operacion,
		OrganizacionRef:              solicitud.OrganizacionRef,
		ExpedienteRef:                solicitud.ExpedienteRef,
		VersionExpediente:            solicitud.VersionExpediente,
		ActorRef:                     solicitud.ActorRef,
		PerfilRef:                    solicitud.PerfilRef,
		DestinoEvidenciaRef:          solicitud.Destino.EvidenciaRef,
		DestinoEvidenciaHuellaSHA256: solicitud.Destino.EvidenciaHuellaSHA256,
		DefinicionRef:                "politica:asignacion-sintetica-001",
		DefinicionVersion:            2,
		DefinicionHuellaSHA256:       strings.Repeat("c", 64),
		Accion: domain.ClaveCatalogo(
			ports.AccionRegistrarAsignacion,
		),
		Finalidad:          "gestionar_contratacion_temporal",
		UnidadEjecutoraRef: "unidad:rrhh-sintetica-001",
		MotivoAutorizacion: d.motivo,
		EvaluadaEn:         solicitud.Instante,
		ValidaHasta:        solicitud.Instante.Add(5 * time.Minute),
	}, nil
}

type transaccionAsignacionDoble struct {
	recibo   ports.ReciboAsignacion
	llamadas int
}

func (d *transaccionAsignacionDoble) ConfirmarAsignacion(
	_ context.Context,
	orden ports.OrdenConfirmarAsignacion,
) (ports.ReciboAsignacion, error) {
	d.llamadas++
	if err := orden.ValidarDentroDeTransaccion(d.recibo.ConfirmadaEn); err != nil {
		return ports.ReciboAsignacion{}, err
	}
	return d.recibo, nil
}

type escenarioAsignacion struct {
	instante    time.Time
	solicitud   SolicitudAsignarUnidad
	contexto    ports.ContextoAutorizacionAltaV3
	preparacion ports.PreparacionAsignacion
	destino     ports.DestinoAsignacionResuelto
	motivo      dominiovec.ReferenciaEntradaCatalogo
	recibo      ports.ReciboAsignacion
}

func TestServicioAsignacionConfirmaVerticalSegura(t *testing.T) {
	escenario := nuevoEscenarioAsignacion(t)
	servicio, dependencias := construirServicioAsignacion(t, escenario)

	recibo, err := servicio.Asignar(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatalf(
			"asignar: %v (destinos=%d politicas=%d autorizador=%d transaccion=%d)",
			err,
			dependencias.destinos.llamadas,
			dependencias.politicas.llamadas,
			dependencias.autorizador.llamadas,
			dependencias.transaccion.llamadas,
		)
	}
	if recibo != dependencias.transaccion.recibo ||
		dependencias.destinos.llamadas != 1 ||
		dependencias.politicas.llamadas != 1 ||
		dependencias.transaccion.llamadas != 1 {
		t.Fatalf("resultado o efectos inesperados: %#v", recibo)
	}
}

func TestServicioAsignacionRechazaDestinoNoLigadoAntesDelPDP(t *testing.T) {
	escenario := nuevoEscenarioAsignacion(t)
	servicio, dependencias := construirServicioAsignacion(t, escenario)
	dependencias.destinos.adulterar = true

	_, err := servicio.Asignar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ErrAsignacionDenegada) ||
		dependencias.politicas.llamadas != 0 ||
		dependencias.transaccion.llamadas != 0 {
		t.Fatalf("destino adulterado no se cerró: %v", err)
	}
}

func TestServicioAsignacionDevuelveReplaySinDuplicarEfectos(t *testing.T) {
	escenario := nuevoEscenarioAsignacion(t)
	servicio, dependencias := construirServicioAsignacion(t, escenario)
	confirmado := dependencias.preparaciones.preparacion
	reciboConfirmado := dependencias.transaccion.recibo
	confirmado.Estado = ports.PreparacionAsignacionConfirmada
	confirmado.ReciboConfirmado = &reciboConfirmado
	dependencias.preparaciones.preparacion = confirmado

	recibo, err := servicio.Asignar(context.Background(), escenario.solicitud)
	if err != nil || recibo != dependencias.transaccion.recibo ||
		dependencias.destinos.llamadas != 0 ||
		dependencias.transaccion.llamadas != 0 {
		t.Fatalf("replay no fue exacto: %#v, %v", recibo, err)
	}
}

type dependenciasAsignacion struct {
	destinos      *resolutorDestinoAsignacionDoble
	politicas     *resolutorPoliticaAsignacionDoble
	preparaciones *preparadorAsignacionDoble
	autorizador   *autorizadorV3Doble
	transaccion   *transaccionAsignacionDoble
}

func construirServicioAsignacion(
	t *testing.T,
	escenario escenarioAsignacion,
) (*ServicioAsignacion, *dependenciasAsignacion) {
	t.Helper()
	ambitos := coleccionAsignacionPrueba(
		t,
		ports.DominioAmbitoIdempotenciaAsignacion+"/v2",
		"a",
	)
	huellas := coleccionAsignacionPrueba(
		t,
		ports.DominioHuellaPeticionAsignacion+"/v2",
		"b",
	)
	escenario.preparacion.AmbitoIdempotenciaHMAC =
		selloHMACRegistroPrueba(
			ports.DominioAmbitoIdempotenciaAsignacion+"/v2",
			"a",
		)
	escenario.preparacion.HuellaPeticionHMAC =
		selloHMACRegistroPrueba(
			ports.DominioHuellaPeticionAsignacion+"/v2",
			"b",
		)
	escenario.recibo.AmbitoIdempotenciaHMAC =
		escenario.preparacion.AmbitoIdempotenciaHMAC
	escenario.recibo.HuellaPeticionHMAC =
		escenario.preparacion.HuellaPeticionHMAC
	autorizador := &autorizadorV3Doble{
		t: t, instante: escenario.instante, motivo: escenario.motivo,
	}
	d := &dependenciasAsignacion{
		destinos: &resolutorDestinoAsignacionDoble{
			destino: escenario.destino,
		},
		politicas: &resolutorPoliticaAsignacionDoble{
			motivo: escenario.motivo,
		},
		preparaciones: &preparadorAsignacionDoble{
			preparacion: escenario.preparacion,
		},
		autorizador: autorizador,
		transaccion: &transaccionAsignacionDoble{
			recibo: escenario.recibo,
		},
	}
	servicio, err := NuevoServicioAsignacion(
		&resolutorContextoDoble{contexto: escenario.contexto},
		&selladorAmbitoAsignacionDoble{coleccion: ambitos},
		&derivadorHuellaAsignacionDoble{coleccion: huellas},
		d.preparaciones,
		d.destinos,
		d.politicas,
		&generadorReferenciasDoble{
			correlacion: "correlacion_88888888888888888888888888888888",
		},
		autorizador,
		&relojMutable{instante: escenario.instante},
		d.transaccion,
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio, d
}

func nuevoEscenarioAsignacion(t *testing.T) escenarioAsignacion {
	t.Helper()
	base := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRectificarAnalisis,
		"-asignacion-sintetica",
	)
	expediente := base.expediente
	instanteCobertura := base.instante.Add(-10 * time.Minute)
	conCobertura, err := expediente.RegistrarViaCobertura(
		expediente.Version,
		domain.DecisionViaCobertura{
			ViaClave:         "via.bolsa",
			ProcedimientoRef: "procedimiento:sintetico-001",
			BolsaRef:         "bolsa:sintetica-001",
			Comprobaciones: []domain.ComprobacionCobertura{{
				Clave:      "bolsa.disponible",
				Resultado:  domain.ComprobacionAfirmativa,
				FuenteRef:  "fuente:bolsa-sintetica-001",
				ReciboRef:  "recibo:bolsa-sintetica-001",
				EvaluadaEn: instanteCobertura,
			}},
			Motivacion: "Cobertura sintética válida para la prueba.",
		},
		domain.DatosActuacion{
			AccionClave:   "cobertura.registrada",
			ActorRef:      "persona:analista-sintetica-001",
			UnidadRef:     "unidad:rrhh-sintetica-001",
			ReciboRef:     "recibo:cobertura-sintetica-001",
			RealizadaEn:   instanteCobertura,
			FaseDestino:   expediente.FaseActual,
			EstadoDestino: expediente.EstadoActual,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := base.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_asignacion",
		CatalogoVersion:      2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64),
		EntradaClave:         "motivo_88888888888888888888888888888888",
	}
	referencias := ports.ReferenciasEfectoAsignacion{
		ReservaRef:      "reserva:asignacion-sintetica-001",
		ReciboRef:       "recibo:asignacion-sintetica-001",
		NotificacionRef: "notificacion:asignacion-sintetica-001",
		BandejaRef:      "bandeja:asignacion-sintetica-001",
		AuditoriaRef:    "auditoria:asignacion-sintetica-001",
		EventoRef:       "evento:asignacion-sintetica-001",
	}
	solicitud := SolicitudAsignarUnidad{
		AutenticacionRef:  vinculo.AutenticacionRef,
		SesionRef:         vinculo.SesionRef,
		PerfilRef:         vinculo.PerfilActivoRef,
		OrganizacionRef:   conCobertura.OrganizacionRef,
		ExpedienteRef:     conCobertura.Referencia,
		VersionEsperada:   conCobertura.Version,
		ClaveIdempotencia: "99999999-aaaa-4bbb-8ccc-dddddddddddd",
		UnidadRef:         "unidad:seleccion-sintetica-001",
		ResponsableRef:    "persona:responsable-sintetica-001",
	}
	return escenarioAsignacion{
		instante:  base.instante,
		solicitud: solicitud,
		contexto:  base.contexto,
		preparacion: ports.PreparacionAsignacion{
			Expediente:      conCobertura,
			Referencias:     referencias,
			Operacion:       ports.OperacionRegistrarAsignacion,
			OrganizacionRef: conCobertura.OrganizacionRef,
			ActorRef:        vinculo.PrincipalID,
			PerfilRef:       vinculo.PerfilActivoRef,
			UnidadRef:       solicitud.UnidadRef,
			ResponsableRef:  solicitud.ResponsableRef,
			Estado:          ports.PreparacionAsignacionReservada,
		},
		destino: ports.DestinoAsignacionResuelto{
			DefinicionRef:          "catalogo:organizacion-sintetica-001",
			DefinicionVersion:      4,
			DefinicionHuellaSHA256: strings.Repeat("e", 64),
			EvidenciaRef:           "evidencia:destino-sintetico-001",
			EvidenciaHuellaSHA256:  strings.Repeat("f", 64),
		},
		motivo: motivo,
		recibo: ports.ReciboAsignacion{
			Operacion:              ports.OperacionRegistrarAsignacion,
			OrganizacionRef:        conCobertura.OrganizacionRef,
			ExpedienteRef:          conCobertura.Referencia,
			VersionAnterior:        conCobertura.Version,
			VersionResultante:      conCobertura.Version + 1,
			UnidadRef:              solicitud.UnidadRef,
			ResponsableRef:         solicitud.ResponsableRef,
			ReciboRef:              referencias.ReciboRef,
			NotificacionRef:        referencias.NotificacionRef,
			BandejaRef:             referencias.BandejaRef,
			AuditoriaRef:           referencias.AuditoriaRef,
			EventoRef:              referencias.EventoRef,
			ConcesionV3DecisionRef: "dec_0123456789abcdef0123456789abcdef",
			ConfirmadaEn:           base.instante,
		},
	}
}

func coleccionAsignacionPrueba(
	t *testing.T,
	dominio string,
	caracter string,
) ports.ColeccionSellosHMAC {
	t.Helper()
	coleccion, err := ports.NuevaColeccionSellosHMAC(
		selloHMACRegistroPrueba(dominio, caracter),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return coleccion
}
