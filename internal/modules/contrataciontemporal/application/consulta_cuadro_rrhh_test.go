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

type autoridadConsultaRRHHPrueba struct {
	contexto ports.ContextoConsultaRRHH
	err      error
	cancelar context.CancelFunc
	llamadas int
}

func (a *autoridadConsultaRRHHPrueba) ResolverContextoConsultaRRHH(
	context.Context,
) (ports.ContextoConsultaRRHH, error) {
	a.llamadas++
	if a.cancelar != nil {
		a.cancelar()
	}
	return a.contexto, a.err
}

type autorizadorConsultaRRHHPrueba struct {
	capacidadCuadro  ports.CapacidadConsultaRRHH
	capacidadDetalle ports.CapacidadConsultaRRHH
	errCuadro        error
	errDetalle       error
	llamadasCuadro   int
	llamadasDetalle  int
}

func (a *autorizadorConsultaRRHHPrueba) AutorizarCuadroRRHH(
	context.Context,
	ports.ContextoConsultaRRHH,
	ports.SolicitudCuadroRRHH,
	time.Time,
) (ports.CapacidadConsultaRRHH, error) {
	a.llamadasCuadro++
	return a.capacidadCuadro, a.errCuadro
}

func (a *autorizadorConsultaRRHHPrueba) AutorizarDetalleRRHH(
	context.Context,
	ports.ContextoConsultaRRHH,
	ports.SolicitudDetalleRRHH,
	time.Time,
) (ports.CapacidadConsultaRRHH, error) {
	a.llamadasDetalle++
	return a.capacidadDetalle, a.errDetalle
}

type sesionConsultaRRHHPrueba struct {
	pagina          ports.PaginaCuadroRRHH
	detalle         ports.DetalleExpedienteRRHH
	errCuadro       error
	errDetalle      error
	cancelarCuadro  context.CancelFunc
	cancelarDetalle context.CancelFunc
	llamadasCuadro  int
	llamadasDetalle int
}

func (s *sesionConsultaRRHHPrueba) ConsultarCuadroYRegistrar(
	context.Context,
	ports.OrdenConsultaCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	s.llamadasCuadro++
	if s.cancelarCuadro != nil {
		s.cancelarCuadro()
	}
	return s.pagina, s.errCuadro
}

func (s *sesionConsultaRRHHPrueba) ConsultarDetalleYRegistrar(
	context.Context,
	ports.OrdenConsultaDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	s.llamadasDetalle++
	if s.cancelarDetalle != nil {
		s.cancelarDetalle()
	}
	return s.detalle, s.errDetalle
}

type relojConsultaRRHHPrueba struct{ instante time.Time }

func (r *relojConsultaRRHHPrueba) Ahora() time.Time { return r.instante }

func TestConsultaCuadroRRHHCierraLecturaRegistrada(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	servicio, err := NuevoServicioConsultaCuadroRRHH(
		entorno.autoridad, entorno.autorizador, entorno.sesion, entorno.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	obtenida, err := servicio.Consultar(context.Background(), entorno.cuadro)
	if err != nil {
		t.Fatalf("consultar: %v", err)
	}
	if entorno.autoridad.llamadas != 1 ||
		entorno.autorizador.llamadasCuadro != 1 ||
		entorno.sesion.llamadasCuadro != 1 ||
		len(obtenida.Expedientes) != 1 ||
		obtenida.Lectura.TotalPublicado() != 1 {
		t.Fatalf("recorrido incompleto: %#v", obtenida)
	}
	obtenida.Expedientes[0].NumeroVisible = "2026/ALTERADO"
	if entorno.sesion.pagina.Expedientes[0].NumeroVisible == "2026/ALTERADO" {
		t.Fatal("el resultado comparte la colección del adaptador")
	}
}

func TestConsultaCuadroRRHHNoDistingueDenegadoAusenteYAjeno(t *testing.T) {
	t.Parallel()
	for _, caso := range []struct {
		nombre        string
		falloAutoriza error
		falloSesion   error
	}{
		{"denegado", ports.ErrConsultaRRHHNoObservable, nil},
		{"ausente", nil, ports.ErrConsultaRRHHNoObservable},
		{"ajeno", nil, ports.ErrConsultaRRHHNoObservable},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			entorno.autorizador.errCuadro = caso.falloAutoriza
			entorno.sesion.errCuadro = caso.falloSesion
			servicio, err := NuevoServicioConsultaCuadroRRHH(
				entorno.autoridad, entorno.autorizador,
				entorno.sesion, entorno.reloj,
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = servicio.Consultar(context.Background(), entorno.cuadro)
			if !errors.Is(err, ErrConsultaRRHHNoObservable) {
				t.Fatalf("oráculo diferente: %v", err)
			}
		})
	}
}

func TestConsultaCuadroRRHHCancelacionPrevalece(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	ctx, cancelar := context.WithCancel(context.Background())
	entorno.sesion.cancelarCuadro = cancelar
	servicio, err := NuevoServicioConsultaCuadroRRHH(
		entorno.autoridad, entorno.autorizador, entorno.sesion, entorno.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Consultar(ctx, entorno.cuadro)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación no prevalece: %v", err)
	}
}

func TestConsultaCuadroRRHHCanceladaNoInvocaPuertos(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	servicio, err := NuevoServicioConsultaCuadroRRHH(
		entorno.autoridad, entorno.autorizador, entorno.sesion, entorno.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = servicio.Consultar(
		ctx, entorno.cuadro,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación no prevalece: %v", err)
	}
	if entorno.autoridad.llamadas != 0 ||
		entorno.autorizador.llamadasCuadro != 0 ||
		entorno.sesion.llamadasCuadro != 0 {
		t.Fatal("una consulta ya cancelada invocó puertos")
	}
}

func TestConstructoresConsultaRRHHRechazanTypedNil(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	var autoridad *autoridadConsultaRRHHPrueba
	if _, err := NuevoServicioConsultaCuadroRRHH(
		autoridad, entorno.autorizador, entorno.sesion, entorno.reloj,
	); !errors.Is(err, ErrServicioConsultaRRHHInvalido) {
		t.Fatalf("typed nil aceptado en cuadro: %v", err)
	}
	if _, err := NuevoServicioConsultaDetalleRRHH(
		autoridad, entorno.autorizador, entorno.sesion, entorno.reloj,
	); !errors.Is(err, ErrServicioConsultaRRHHInvalido) {
		t.Fatalf("typed nil aceptado en detalle: %v", err)
	}
}

type entornoConsultaRRHH struct {
	ahora       time.Time
	contexto    ports.ContextoConsultaRRHH
	cuadro      ports.SolicitudCuadroRRHH
	detalle     ports.SolicitudDetalleRRHH
	autoridad   *autoridadConsultaRRHHPrueba
	autorizador *autorizadorConsultaRRHHPrueba
	sesion      *sesionConsultaRRHHPrueba
	reloj       *relojConsultaRRHHPrueba
	expediente  domain.Expediente
}

func nuevoEntornoConsultaRRHH(t *testing.T) *entornoConsultaRRHH {
	t.Helper()
	ahora := time.Date(2026, 7, 26, 9, 0, 0, 0, time.UTC)
	contexto := contextoConsultaRRHHV3Prueba(t, ahora)
	cuadro, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 50, "")
	if err != nil {
		t.Fatal(err)
	}
	detalle, err := ports.NuevaSolicitudDetalleRRHH(
		"expediente:rrhh:001", 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidadCuadro := capacidadConsultaCuadroRRHHV3Prueba(
		t,
		contexto,
		cuadro,
		ahora,
		ports.AmbitoOrganizacionRRHH,
		contexto.OrganizacionRef(),
	)
	capacidadDetalle := capacidadConsultaDetalleRRHHV3Prueba(
		t,
		contexto,
		detalle,
		ahora,
		ports.AmbitoOrganizacionRRHH,
		contexto.OrganizacionRef(),
	)
	expediente := expedienteConsultaRRHHPrueba(t, ahora)
	resumen := ports.ResumenExpedienteRRHH{
		ExpedienteRef:   expediente.Referencia,
		OrganizacionRef: expediente.OrganizacionRef,
		NumeroVisible:   expediente.NumeroVisible, Version: expediente.Version,
		FlujoRef:     expediente.Flujo.DefinicionRef,
		FlujoVersion: expediente.Flujo.Version,
		FlujoHuella:  expediente.Flujo.HuellaSHA256,
		FaseClave:    expediente.FaseActual, EstadoClave: expediente.EstadoActual,
		CentroRef:    expediente.Solicitud.CentroRef,
		CategoriaRef: expediente.Solicitud.CategoriaRef,
		CreadoEn:     expediente.CreadoEn, ActualizadoEn: expediente.ActualizadoEn,
	}
	reciboCuadro := reciboConsultaRRHHPrueba(
		t, contexto, capacidadCuadro, ahora, "", 0, 1,
	)
	reciboDetalle := reciboConsultaRRHHPrueba(
		t, contexto, capacidadDetalle, ahora,
		expediente.Referencia, expediente.Version, 1,
	)
	proyeccionDetalle, err := ports.NuevoDetalleExpedienteRRHH(
		expediente, reciboDetalle,
	)
	if err != nil {
		t.Fatal(err)
	}
	return &entornoConsultaRRHH{
		ahora: ahora, contexto: contexto, cuadro: cuadro, detalle: detalle,
		autoridad: &autoridadConsultaRRHHPrueba{contexto: contexto},
		autorizador: &autorizadorConsultaRRHHPrueba{
			capacidadCuadro: capacidadCuadro, capacidadDetalle: capacidadDetalle,
		},
		sesion: &sesionConsultaRRHHPrueba{
			pagina: ports.PaginaCuadroRRHH{
				GeneradaEn: ahora, Expedientes: []ports.ResumenExpedienteRRHH{resumen},
				Lectura: reciboCuadro,
			},
			detalle: proyeccionDetalle,
		},
		reloj:      &relojConsultaRRHHPrueba{instante: ahora},
		expediente: expediente,
	}
}

func reciboConsultaRRHHPrueba(
	t *testing.T,
	contexto ports.ContextoConsultaRRHH,
	capacidad ports.CapacidadConsultaRRHH,
	ahora time.Time,
	expedienteRef string,
	version uint64,
	total uint16,
) ports.ReciboLecturaRRHH {
	t.Helper()
	recibo, err := ports.NuevoReciboLecturaRRHH(
		"lectura:rrhh:001", "auditoria:rrhh:001",
		contexto, capacidad, expedienteRef, version, total, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	return recibo
}

func expedienteConsultaRRHHPrueba(
	t *testing.T,
	ahora time.Time,
) domain.Expediente {
	t.Helper()
	inicio := ahora.AddDate(0, 1, 0)
	inicio = time.Date(
		inicio.Year(), inicio.Month(), inicio.Day(),
		0, 0, 0, 0, time.UTC,
	)
	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia:      "expediente:rrhh:001",
		OrganizacionRef: "organizacion:diputacion-granada",
		NumeroVisible:   "2026/CT-001",
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:rrhh:001", Version: 1,
			HuellaSHA256: strings.Repeat("a", 64),
		},
		FaseInicial: "solicitud",
		Solicitud: domain.SolicitudCentro{
			CentroRef: "centro:rrhh:001", ContactoRef: "contacto:rrhh:001",
			CategoriaRef: "categoria:rrhh:001", GrupoSubgrupo: "C2",
			MotivoClave: "sustitucion", Detalle: "Necesidad temporal.",
			Periodo: domain.PeriodoPrevisto{
				Inicio: inicio, Fin: inicio.AddDate(0, 1, 0),
			},
			RC:                 domain.DeclaracionRC{Existe: false},
			DocumentosAdjuntos: []string{},
		},
		Actuacion: domain.DatosActuacion{
			AccionClave: "solicitud.registrada", ActorRef: "actor:rrhh:001",
			UnidadRef: "unidad:rrhh:001", ReciboRef: "recibo:rrhh:001",
			RealizadaEn: ahora, FaseDestino: "solicitud",
			EstadoDestino: domain.EstadoEnCurso,
		},
	})
	if err != nil {
		t.Fatalf("expediente: %v", err)
	}
	return expediente
}
