package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	claveAmbitoRegistroPrueba   = "vec.contratacion-temporal.ambito-idempotencia/v1"
	clavePeticionRegistroPrueba = "vec.contratacion-temporal.huella-peticion/v1"
)

type resolutorContextoDoble struct {
	contexto ports.ContextoAutorizacionAltaV3
	err      error
	llamadas int
	margen   time.Duration
}

func (d *resolutorContextoDoble) ResolverContextoAutorizacionAltaV3(
	ctx context.Context,
	_ ports.SolicitudResolverContextoAutorizacionAltaV3,
) (ports.ContextoAutorizacionAltaV3, error) {
	d.llamadas++
	if limite, existe := ctx.Deadline(); existe {
		d.margen = time.Until(limite)
	}
	return d.contexto, d.err
}

type resolutorFlujoDoble struct {
	configuracion ports.ConfiguracionAltaFlujo
	err           error
}

func (d *resolutorFlujoDoble) ResolverFlujoAlta(
	context.Context,
	ports.SolicitudResolverFlujo,
) (ports.ConfiguracionAltaFlujo, error) {
	return d.configuracion, d.err
}

type derivadorHuellaDoble struct {
	coleccion ports.ColeccionSellosHMAC
	err       error
	antes     func(*ports.MaterialHuellaAlta)
}

func (d *derivadorHuellaDoble) DerivarHuellaAlta(
	_ context.Context,
	material ports.MaterialHuellaAlta,
) (ports.ColeccionSellosHMAC, error) {
	if d.antes != nil {
		d.antes(&material)
	}
	return d.coleccion, d.err
}

type selladorAmbitoDoble struct {
	coleccion ports.ColeccionSellosHMAC
	err       error
}

func (d *selladorAmbitoDoble) SellarAmbitoIdempotencia(
	context.Context,
	ports.SolicitudSellarAmbitoIdempotencia,
) (ports.ColeccionSellosHMAC, error) {
	return d.coleccion, d.err
}

type resolutorMotivoDoble struct {
	motivo  dominiovec.ReferenciaEntradaCatalogo
	err     error
	despues func()
}

func (d *resolutorMotivoDoble) ResolverMotivoAutorizacionAltaV3(
	context.Context,
	ports.SolicitudResolverMotivoAutorizacionAltaV3,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	if d.despues != nil {
		d.despues()
	}
	return d.motivo, d.err
}

type generadorReferenciasDoble struct {
	correlacion string
	err         error
	despues     func()
	llamadas    int
}

func (d *generadorReferenciasDoble) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	d.llamadas++
	if d.despues != nil {
		d.despues()
	}
	return d.correlacion, d.err
}

func (d *generadorReferenciasDoble) NuevaClaveMotivoAutorizacionV2(
	context.Context,
) (string, error) {
	return "", errors.New("no se usa para el alta")
}

type preparadorAltaDoble struct {
	preparacion ports.PreparacionAlta
	err         error
	llamadas    int
	antes       func()
}

func (d *preparadorAltaDoble) PrepararAlta(
	context.Context,
	ports.SolicitudPrepararAlta,
) (ports.PreparacionAlta, error) {
	d.llamadas++
	if d.antes != nil {
		d.antes()
	}
	return d.preparacion, d.err
}

type relojMutable struct {
	mu       sync.Mutex
	instante time.Time
	llamadas int
	despues  func(int)
}

func (r *relojMutable) Ahora() time.Time {
	r.mu.Lock()
	r.llamadas++
	instante, llamadas, despues := r.instante, r.llamadas, r.despues
	r.mu.Unlock()
	if despues != nil {
		despues(llamadas)
	}
	return instante
}

func (r *relojMutable) fijar(instante time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instante = instante
}

type transaccionAltaDoble struct {
	recibo   ports.ReciboAlta
	err      error
	llamadas int
	orden    ports.OrdenConfirmarAlta
	despues  func()
}

func (d *transaccionAltaDoble) ConfirmarAlta(
	_ context.Context,
	orden ports.OrdenConfirmarAlta,
) (ports.ReciboAlta, error) {
	d.llamadas++
	d.orden = orden
	if d.despues != nil {
		d.despues()
	}
	return d.recibo, d.err
}

type registroConcesionV3Doble struct{ registradaEn time.Time }

func (d registroConcesionV3Doble) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	context.Context,
	puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	return d.registradaEn, nil
}

type autorizadorV3Doble struct {
	t                 *testing.T
	instante          time.Time
	motivo            dominiovec.ReferenciaEntradaCatalogo
	err               error
	llamadas          int
	solicitud         dominiovec.SolicitudAutorizacionLigadaV3
	antes             func()
	confirmacionAjena bool
	decisionDenegada  bool
	transformar       func(
		dominiovec.SolicitudAutorizacionLigadaV3,
		dominiovec.ResultadoContextoActorRegistradoV2,
	) (
		dominiovec.SolicitudAutorizacionLigadaV3,
		dominiovec.ResultadoContextoActorRegistradoV2,
		dominiovec.ReferenciaEntradaCatalogo,
	)
}

func (d *autorizadorV3Doble) ExigirSolicitudLigadaV3(
	_ context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	error,
) {
	d.llamadas++
	d.solicitud = solicitud
	if d.antes != nil {
		d.antes()
	}
	if d.err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			d.err
	}
	if d.transformar != nil {
		solicitud, resultado, d.motivo = d.transformar(solicitud, resultado)
	}
	decision, confirmacion, err := concesionAutorizacionV3Prueba(
		d.t,
		solicitud,
		resultado,
		d.motivo,
		d.instante,
		"dec_0123456789abcdef0123456789abcdef",
		!d.decisionDenegada,
	)
	if d.decisionDenegada {
		return decision,
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			err
	}
	if !d.confirmacionAjena {
		return decision, confirmacion, err
	}
	_, ajena, errAjeno := concesionAutorizacionV3Prueba(
		d.t,
		solicitud,
		resultado,
		d.motivo,
		d.instante,
		"dec_fedcba9876543210fedcba9876543210",
		true,
	)
	return decision, ajena, errors.Join(err, errAjeno)
}

type escenarioRegistro struct {
	instante      time.Time
	solicitud     SolicitudRegistrarExpediente
	contexto      ports.ContextoAutorizacionAltaV3
	configuracion ports.ConfiguracionAltaFlujo
	preparacion   ports.PreparacionAlta
	motivo        dominiovec.ReferenciaEntradaCatalogo
	recibo        ports.ReciboAlta
}

func nuevoEscenarioRegistro(t *testing.T) escenarioRegistro {
	t.Helper()
	instante := time.Date(2026, time.July, 23, 9, 15, 0, 0, time.UTC)
	contexto := contextoAutorizacionAltaV3Prueba(t, instante)
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return escenarioRegistro{
		instante: instante,
		solicitud: SolicitudRegistrarExpediente{
			AutenticacionRef:  vinculo.AutenticacionRef,
			SesionRef:         vinculo.SesionRef,
			PerfilRef:         vinculo.PerfilActivoRef,
			OrganizacionRef:   "organizacion:diputacion-granada",
			ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
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
		contexto: contexto,
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
			ActorRef:               vinculo.PrincipalID,
			PerfilRef:              vinculo.PerfilActivoRef,
			Estado:                 ports.PreparacionReservada,
		},
		motivo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           "motivos_autorizacion",
			CatalogoVersion:      2,
			CatalogoHuellaSHA256: strings.Repeat("d", 64),
			EntradaClave:         "motivo_11111111111111111111111111111111",
		},
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

type dependenciasRegistro struct {
	contextos     *resolutorContextoDoble
	flujos        *resolutorFlujoDoble
	huellas       *derivadorHuellaDoble
	ambitos       *selladorAmbitoDoble
	motivos       *resolutorMotivoDoble
	correlaciones *generadorReferenciasDoble
	preparaciones *preparadorAltaDoble
	autorizador   *autorizadorV3Doble
	reloj         *relojMutable
	transaccion   *transaccionAltaDoble
}

func construirServicioRegistro(
	t *testing.T,
	escenario escenarioRegistro,
) (*ServicioRegistroSolicitud, *dependenciasRegistro) {
	t.Helper()
	ambitosHMAC, err := ports.NuevaColeccionSellosHMAC(
		selloHMACRegistroPrueba(
			"vec.contratacion-temporal.ambito-idempotencia/v2",
			"c",
		),
		[]string{escenario.preparacion.AmbitoIdempotenciaHMAC},
	)
	if err != nil {
		t.Fatal(err)
	}
	huellasHMAC, err := ports.NuevaColeccionSellosHMAC(
		selloHMACRegistroPrueba(
			"vec.contratacion-temporal.huella-peticion/v2",
			"c",
		),
		[]string{escenario.preparacion.HuellaPeticionHMAC},
	)
	if err != nil {
		t.Fatal(err)
	}
	d := &dependenciasRegistro{
		contextos: &resolutorContextoDoble{contexto: escenario.contexto},
		flujos:    &resolutorFlujoDoble{configuracion: escenario.configuracion},
		huellas:   &derivadorHuellaDoble{coleccion: huellasHMAC},
		ambitos:   &selladorAmbitoDoble{coleccion: ambitosHMAC},
		motivos:   &resolutorMotivoDoble{motivo: escenario.motivo},
		correlaciones: &generadorReferenciasDoble{
			correlacion: "correlacion_11111111111111111111111111111111",
		},
		preparaciones: &preparadorAltaDoble{preparacion: escenario.preparacion},
		autorizador: &autorizadorV3Doble{
			t: t, instante: escenario.instante, motivo: escenario.motivo,
		},
		reloj:       &relojMutable{instante: escenario.instante},
		transaccion: &transaccionAltaDoble{recibo: escenario.recibo},
	}
	servicio, err := NuevoServicioRegistroSolicitud(
		d.contextos,
		d.flujos,
		d.huellas,
		d.ambitos,
		d.motivos,
		d.correlaciones,
		d.preparaciones,
		d.autorizador,
		d.reloj,
		d.transaccion,
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio, d
}

func TestRegistroSolicitudAutorizaV3AntesDeReservarYConfirma(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	secuencia := make([]string, 0, 2)
	d.autorizador.antes = func() { secuencia = append(secuencia, "autorizar") }
	d.preparaciones.antes = func() { secuencia = append(secuencia, "preparar") }

	recibo, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatalf("registrar: %v", err)
	}
	if recibo != escenario.recibo || d.transaccion.llamadas != 1 ||
		len(secuencia) != 2 || secuencia[0] != "autorizar" || secuencia[1] != "preparar" {
		t.Fatalf("secuencia o recibo incorrectos: %#v, %v", recibo, secuencia)
	}
	datosSolicitud, err := d.autorizador.solicitud.Datos()
	datosAmbitos, errAmbitos := d.ambitos.coleccion.Datos()
	datosHuellas, errHuellas := d.huellas.coleccion.Datos()
	if err != nil || errAmbitos != nil || errHuellas != nil ||
		datosSolicitud.Recurso.Referencia !=
			datosAmbitos.Activo.Valor ||
		datosSolicitud.Recurso.Atributos[ports.AtributoHuellaPeticionHMACActiva] !=
			datosHuellas.Activo.Valor ||
		datosSolicitud.Accion != ports.AccionCrearSolicitud ||
		datosSolicitud.Finalidad != ports.FinalidadCrearSolicitud ||
		datosSolicitud.Recurso.ModuloID != ports.ModuloContratacion ||
		datosSolicitud.Recurso.Tipo != ports.TipoRecursoExpediente {
		t.Fatalf("solicitud V3 no ligada al ámbito: %#v, %v", datosSolicitud, err)
	}
	datosOrden, err := d.transaccion.orden.Datos()
	vinculo, errVinculo := datosSolicitud.VinculoAutenticacionActor.Datos()
	_, errSolicitudV3 := datosOrden.SolicitudAutorizacionV3.Datos()
	if err != nil || errVinculo != nil || errSolicitudV3 != nil ||
		datosOrden.CorrelacionV3Ref != d.correlaciones.correlacion ||
		datosOrden.Expediente.Actuaciones[0].ActorRef != vinculo.PrincipalID ||
		datosOrden.DecisionAutorizacionV3.ValidarPara(datosOrden.SolicitudAutorizacionV3) != nil ||
		datosOrden.ConfirmacionRegistroV3.Validar() != nil {
		t.Fatalf("orden sin capacidades V3: %v", err)
	}
	expedienteCruzado := datosOrden.Expediente
	expedienteCruzado.Actuaciones[0].ActorRef = "per_2222222222222222222222"
	if _, err := ports.NuevaOrdenConfirmarAlta(ports.DatosOrdenConfirmarAlta{
		Expediente:              expedienteCruzado,
		SolicitudAutorizacionV3: datosOrden.SolicitudAutorizacionV3,
		DecisionAutorizacionV3:  datosOrden.DecisionAutorizacionV3,
		ConfirmacionRegistroV3:  datosOrden.ConfirmacionRegistroV3,
		AmbitosIdempotenciaHMAC: datosOrden.AmbitosIdempotenciaHMAC,
		HuellasPeticionHMAC:     datosOrden.HuellasPeticionHMAC,
		Preparacion:             datosOrden.Preparacion,
	}); !errors.Is(err, ports.ErrOrdenAltaInvalida) {
		t.Fatalf("actuación atribuida a otro actor aceptada: %v", err)
	}
}

func TestContratosNoAceptanCorrelacionFuncionalAportada(t *testing.T) {
	for _, tipo := range []reflect.Type{
		reflect.TypeOf(SolicitudRegistrarExpediente{}),
		reflect.TypeOf(ports.DatosOrdenConfirmarAlta{}),
	} {
		for _, nombre := range []string{"Correlacion", "CorrelacionRef", "CorrelacionV3Ref"} {
			if _, existe := tipo.FieldByName(nombre); existe {
				t.Fatalf("%s acepta correlación aportada mediante %s", tipo.Name(), nombre)
			}
		}
	}
	if _, existe := reflect.TypeOf(ports.EvidenciaOrdenConfirmarAlta{}).
		FieldByName("CorrelacionV3Ref"); !existe {
		t.Fatal("la evidencia de persistencia no propagó la correlación V3")
	}
}

func TestIntegracionV3PermiteCanonMinimizadoPeroNoReconstruyeCapacidades(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	if _, err := servicio.Registrar(context.Background(), escenario.solicitud); err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(escenario.contexto.Vinculo)
	if err != nil {
		t.Fatalf("canon JSON deliberado del vínculo rechazado: %v", err)
	}
	for _, prohibido := range [][]byte{
		[]byte("dni"), []byte("nombre"), []byte("email"), []byte("roles"),
		[]byte("permissions"), []byte("attributes"), []byte("operacion_ref"),
	} {
		if bytes.Contains(bytes.ToLower(contenido), prohibido) {
			t.Fatalf("canon del vínculo contiene campo no minimizado: %s", prohibido)
		}
	}
	var reconstruido dominiovec.VinculoAutenticacionActorV2
	if err := json.Unmarshal(contenido, &reconstruido); !errors.Is(err, dominiovec.ErrReconstruccionVinculoAutenticacionActorProhibida) ||
		reconstruido.Validar() == nil {
		t.Fatalf("JSON reconstruyó una capacidad: %v", err)
	}
	if _, err := escenario.contexto.Vinculo.MarshalText(); !errors.Is(
		err,
		dominiovec.ErrSerializacionAlternativaVinculoAutenticacionActorV2Prohibida,
	) {
		t.Fatalf("serialización alternativa del vínculo aceptada: %v", err)
	}
	datosOrden, err := d.transaccion.orden.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(datosOrden.SolicitudAutorizacionV3); !errors.Is(err, dominiovec.ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
		t.Fatalf("solicitud nominal serializable: %v", err)
	}
	if _, err := json.Marshal(datosOrden.DecisionAutorizacionV3); !errors.Is(err, dominiovec.ErrSerializacionDecisionAutorizacionLigadaV3Prohibida) {
		t.Fatalf("decisión nominal serializable: %v", err)
	}
	if _, err := json.Marshal(datosOrden.ConfirmacionRegistroV3); !errors.Is(err, puertosvec.ErrSerializacionRegistroAutorizacionLigadaV3Prohibida) {
		t.Fatalf("confirmación nominal serializable: %v", err)
	}
}

func TestRegistroSolicitudAislaMutacionesDePuertosYOrden(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	d.huellas.antes = func(material *ports.MaterialHuellaAlta) {
		material.Solicitud.DocumentosAdjuntos[0] = "documento:mutado"
	}
	if _, err := servicio.Registrar(context.Background(), escenario.solicitud); err != nil {
		t.Fatal(err)
	}
	if escenario.solicitud.Solicitud.DocumentosAdjuntos[0] !=
		"documento:informe-necesidad-001" {
		t.Fatal("un puerto mutó la solicitud original")
	}
	primera, err := d.transaccion.orden.Datos()
	if err != nil {
		t.Fatal(err)
	}
	primera.Expediente.Solicitud.DocumentosAdjuntos[0] = "documento:orden-mutada"
	datosPrimera, err := primera.HuellasPeticionHMAC.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosPrimera.Retenidos[0] = ports.SelloGeneracionalHMAC{}
	segunda, err := d.transaccion.orden.Datos()
	datosSegunda, errHuellas := segunda.HuellasPeticionHMAC.Datos()
	if err != nil || segunda.Expediente.Solicitud.DocumentosAdjuntos[0] !=
		"documento:informe-necesidad-001" ||
		errHuellas != nil || datosSegunda.Retenidos[0].Generacion != 1 {
		t.Fatalf("la orden no hizo copia defensiva: %v", err)
	}
}

func TestRegistroSolicitudReplayConfirmadoRevalidaPDP(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	recibo := escenario.recibo
	escenario.preparacion.Estado = ports.PreparacionConfirmada
	escenario.preparacion.ReciboConfirmado = &recibo
	servicio, d := construirServicioRegistro(t, escenario)

	devuelto, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if err != nil || devuelto != recibo || d.autorizador.llamadas != 1 ||
		d.preparaciones.llamadas != 1 || d.transaccion.llamadas != 0 {
		t.Fatalf("replay no revalidado: recibo=%#v err=%v auth=%d prep=%d tx=%d",
			devuelto, err, d.autorizador.llamadas, d.preparaciones.llamadas, d.transaccion.llamadas)
	}
}

func TestRegistroSolicitudDenegadaNoReservaNada(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	recibo := escenario.recibo
	escenario.preparacion.Estado = ports.PreparacionConfirmada
	escenario.preparacion.ReciboConfirmado = &recibo
	servicio, d := construirServicioRegistro(t, escenario)
	d.autorizador.err = errors.New("PDP deniega")

	_, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ports.ErrAutorizacionDenegada) ||
		d.autorizador.llamadas != 1 || d.preparaciones.llamadas != 0 ||
		d.transaccion.llamadas != 0 {
		t.Fatalf("denegación produjo efecto: err=%v auth=%d prep=%d tx=%d",
			err, d.autorizador.llamadas, d.preparaciones.llamadas, d.transaccion.llamadas)
	}
}

func TestRegistroSolicitudRechazaContextoV3QueNoCorrespondeALaPeticion(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	escenario.solicitud.PerfilRef = "prf_xxxxxxxxxxxxxxxxxxxxxx"

	_, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ports.ErrAutorizacionDenegada) ||
		d.contextos.llamadas != 1 || d.autorizador.llamadas != 0 ||
		d.preparaciones.llamadas != 0 {
		t.Fatalf("contexto cruzado alcanzó PDP o persistencia: %v", err)
	}
}

func TestRegistroSolicitudExpiradaMientrasPreparaNoConfirmaNiDevuelveReplay(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	d.preparaciones.antes = func() {
		d.reloj.fijar(escenario.instante.Add(2 * time.Minute))
	}

	_, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ports.ErrAutorizacionDenegada) ||
		d.preparaciones.llamadas != 1 || d.transaccion.llamadas != 0 {
		t.Fatalf("concesión expirada llegó al efecto: err=%v prep=%d tx=%d",
			err, d.preparaciones.llamadas, d.transaccion.llamadas)
	}

	recibo := escenario.recibo
	escenario.preparacion.Estado = ports.PreparacionConfirmada
	escenario.preparacion.ReciboConfirmado = &recibo
	servicio, d = construirServicioRegistro(t, escenario)
	d.preparaciones.antes = func() {
		d.reloj.fijar(escenario.instante.Add(2 * time.Minute))
	}
	if _, err = servicio.Registrar(context.Background(), escenario.solicitud); !errors.Is(err, ports.ErrAutorizacionDenegada) || d.transaccion.llamadas != 0 {
		t.Fatalf("replay expirado eludió PDP: %v", err)
	}
}

func TestRegistroSolicitudRechazaNulosYReferenciasFabricables(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)

	invalida := escenario.solicitud
	invalida.AutenticacionRef = "aut_corta"
	if _, err := servicio.Registrar(context.Background(), invalida); !errors.Is(err, ports.ErrAutorizacionDenegada) ||
		d.contextos.llamadas != 0 || d.preparaciones.llamadas != 0 {
		t.Fatalf("referencia de autenticación fabricable aceptada: %v", err)
	}
	if _, err := NuevoServicioRegistroSolicitud(
		nil, d.flujos, d.huellas, d.ambitos, d.motivos, d.correlaciones,
		d.preparaciones, d.autorizador, d.reloj, d.transaccion,
	); !errors.Is(err, ErrServicioRegistroInvalido) {
		t.Fatalf("dependencia nula aceptada: %v", err)
	}
	var nuloTipado *resolutorContextoDoble
	if _, err := NuevoServicioRegistroSolicitud(
		nuloTipado, d.flujos, d.huellas, d.ambitos, d.motivos, d.correlaciones,
		d.preparaciones, d.autorizador, d.reloj, d.transaccion,
	); !errors.Is(err, ErrServicioRegistroInvalido) {
		t.Fatalf("dependencia nula tipada aceptada: %v", err)
	}
}

func TestRegistroSolicitudRechazaAmbitoDistintoDelAutorizado(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	d.preparaciones.preparacion.AmbitoIdempotenciaHMAC =
		selloHMACRegistroPrueba(claveAmbitoRegistroPrueba, "e")

	_, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ports.ErrPreparacionAltaInvalida) || d.transaccion.llamadas != 0 {
		t.Fatalf("ámbito preparado distinto alcanzó el efecto: %v", err)
	}
}

func TestRegistroSolicitudRechazaConfirmacionV3CruzadaAntesDePreparar(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	servicio, d := construirServicioRegistro(t, escenario)
	d.autorizador.confirmacionAjena = true

	_, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if !errors.Is(err, ports.ErrAutorizacionDenegada) ||
		d.preparaciones.llamadas != 0 || d.transaccion.llamadas != 0 {
		t.Fatalf("confirmación V3 cruzada produjo reserva o efecto: %v", err)
	}
}
