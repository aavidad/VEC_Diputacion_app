package application

import (
	"context"
	"errors"
	"strings"
	"sync"
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

type sesionConsultaRRHHPrueba struct {
	t               *testing.T
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
	_ context.Context,
	orden ports.OrdenConsultaCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	s.llamadasCuadro++
	if s.cancelarCuadro != nil {
		s.cancelarCuadro()
	}
	if s.errCuadro != nil {
		return ports.PaginaCuadroRRHH{}, s.errCuadro
	}
	pagina := clonarPaginaCuadroRRHH(s.pagina)
	pagina.Lectura = reciboConsultaRRHHPrueba(
		s.t, orden.Contexto(), orden.Capacidad(), orden.Instante(),
		"", 0, uint16(len(pagina.Expedientes)),
	)
	return pagina, nil
}

func (s *sesionConsultaRRHHPrueba) ConsultarDetalleYRegistrar(
	_ context.Context,
	orden ports.OrdenConsultaDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	s.llamadasDetalle++
	if s.cancelarDetalle != nil {
		s.cancelarDetalle()
	}
	if s.errDetalle != nil {
		return ports.DetalleExpedienteRRHH{}, s.errDetalle
	}
	detalle := s.detalle.Clonar()
	detalle.Lectura = reciboConsultaRRHHPrueba(
		s.t, orden.Contexto(), orden.Capacidad(), orden.Instante(),
		orden.Solicitud().ExpedienteRef(), detalle.Resumen.Version, 1,
	)
	return detalle, nil
}

type relojConsultaRRHHPrueba struct {
	mu        sync.Mutex
	instante  time.Time
	instantes []time.Time
	llamadas  int
	despuesDe func(int)
}

func (r *relojConsultaRRHHPrueba) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadas++
	instante := r.instante
	if len(r.instantes) > 0 {
		instante = r.instantes[0]
		if len(r.instantes) > 1 {
			r.instantes = r.instantes[1:]
		}
	}
	if r.despuesDe != nil {
		r.despuesDe(r.llamadas)
	}
	return instante
}

func TestConsultaCuadroRRHHCierraLecturaRegistrada(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	servicio, err := NuevoServicioConsultaCuadroRRHH(
		entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	obtenida, err := servicio.Consultar(context.Background(), entorno.cuadro)
	if err != nil {
		t.Fatalf("consultar: %v", err)
	}
	if entorno.autoridad.llamadas != 1 ||
		entorno.emision.motivos.llamadasCuadro != 1 ||
		entorno.emision.motivos.llamadasDetalle != 0 ||
		entorno.emision.correlaciones.llamadas != 1 ||
		entorno.emision.reloj.llamadas != 2 ||
		entorno.emision.cuadro.llamadas != 1 ||
		entorno.emision.detalle.llamadas != 0 ||
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

func TestConsultaCuadroRRHHNoDistingueAusenteYAjeno(t *testing.T) {
	t.Parallel()
	for _, nombre := range []string{"ausente", "ajeno"} {
		nombre := nombre
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			entorno.sesion.errCuadro = ports.ErrConsultaRRHHNoObservable
			servicio, err := NuevoServicioConsultaCuadroRRHH(
				entorno.autoridad, entorno.emisor,
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
		entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj,
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
		entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj,
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
		entorno.emision.cuadro.llamadas != 0 ||
		entorno.sesion.llamadasCuadro != 0 {
		t.Fatal("una consulta ya cancelada invocó puertos")
	}
}

func TestConstructoresConsultaRRHHRechazanTypedNil(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	var autoridad *autoridadConsultaRRHHPrueba
	if _, err := NuevoServicioConsultaCuadroRRHH(
		autoridad, entorno.emisor, entorno.sesion, entorno.reloj,
	); !errors.Is(err, ErrServicioConsultaRRHHInvalido) {
		t.Fatalf("typed nil aceptado en cuadro: %v", err)
	}
	if _, err := NuevoServicioConsultaDetalleRRHH(
		autoridad, entorno.emisor, entorno.sesion, entorno.reloj,
	); !errors.Is(err, ErrServicioConsultaRRHHInvalido) {
		t.Fatalf("typed nil aceptado en detalle: %v", err)
	}
	var emisor *ports.EmisorMaterialConsultaRRHH
	if _, err := NuevoServicioConsultaCuadroRRHH(
		entorno.autoridad, emisor, entorno.sesion, entorno.reloj,
	); !errors.Is(err, ErrServicioConsultaRRHHInvalido) {
		t.Fatalf("emisor typed nil aceptado en cuadro: %v", err)
	}
	if _, err := NuevoServicioConsultaDetalleRRHH(
		entorno.autoridad, emisor, entorno.sesion, entorno.reloj,
	); !errors.Is(err, ErrServicioConsultaRRHHInvalido) {
		t.Fatalf("emisor typed nil aceptado en detalle: %v", err)
	}
}

func TestConsultasRRHHRechazanMaterialDeOtroContexto(t *testing.T) {
	t.Parallel()
	for _, operacion := range []string{"cuadro", "detalle"} {
		operacion := operacion
		t.Run(operacion, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			ajeno := contextoAutorizacionAltaV3PruebaConMarcas(
				t, entorno.ahora, "b", "b",
			).Resultado
			var err error
			switch operacion {
			case "cuadro":
				entorno.emision.cuadro.resultadoAjeno = &ajeno
				servicio, fallo := NuevoServicioConsultaCuadroRRHH(
					entorno.autoridad, entorno.emisor,
					entorno.sesion, entorno.reloj,
				)
				if fallo != nil {
					t.Fatal(fallo)
				}
				_, err = servicio.Consultar(context.Background(), entorno.cuadro)
			case "detalle":
				entorno.emision.detalle.resultadoAjeno = &ajeno
				servicio, fallo := NuevoServicioConsultaDetalleRRHH(
					entorno.autoridad, entorno.emisor,
					entorno.sesion, entorno.reloj,
				)
				if fallo != nil {
					t.Fatal(fallo)
				}
				_, err = servicio.Consultar(context.Background(), entorno.detalle)
			}
			if !errors.Is(err, ErrConsultaRRHHNoDisponible) ||
				entorno.sesion.llamadasCuadro != 0 ||
				entorno.sesion.llamadasDetalle != 0 ||
				entorno.reloj.llamadas != 0 {
				t.Fatalf(
					"material cruzado aceptado: error=%v cuadro=%d "+
						"detalle=%d reloj=%d",
					err, entorno.sesion.llamadasCuadro,
					entorno.sesion.llamadasDetalle, entorno.reloj.llamadas,
				)
			}
		})
	}
}

type entornoConsultaRRHH struct {
	ahora      time.Time
	contexto   ports.ContextoConsultaRRHH
	cuadro     ports.SolicitudCuadroRRHH
	detalle    ports.SolicitudDetalleRRHH
	autoridad  *autoridadConsultaRRHHPrueba
	emisor     *ports.EmisorMaterialConsultaRRHH
	emision    *entornoEmisorConsultaRRHHPrueba
	sesion     *sesionConsultaRRHHPrueba
	reloj      *relojConsultaRRHHPrueba
	expediente domain.Expediente
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
		t, contexto, cuadro, ahora,
	)
	capacidadDetalle := capacidadConsultaDetalleRRHHV3Prueba(
		t, contexto, detalle, ahora,
	)
	expediente := expedienteConsultaRRHHPrueba(t, ahora)
	emision := nuevoEmisorConsultaRRHHV3Prueba(t, ahora)
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
		emisor:    emision.emisor, emision: emision,
		sesion: &sesionConsultaRRHHPrueba{
			t: t,
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
