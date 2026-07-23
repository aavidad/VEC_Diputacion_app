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
}

func (d *resolutorContextoDoble) ResolverContextoAutorizacionAltaV3(
	_ context.Context,
	_ ports.SolicitudResolverContextoAutorizacionAltaV3,
) (ports.ContextoAutorizacionAltaV3, error) {
	d.llamadas++
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
	huella string
	err    error
}

func (d *derivadorHuellaDoble) DerivarHuellaAlta(
	context.Context,
	ports.MaterialHuellaAlta,
) (string, error) {
	return d.huella, d.err
}

type selladorAmbitoDoble struct {
	ambito string
	err    error
}

func (d *selladorAmbitoDoble) SellarAmbitoIdempotencia(
	context.Context,
	ports.SolicitudSellarAmbitoIdempotencia,
) (string, error) {
	return d.ambito, d.err
}

type resolutorMotivoDoble struct {
	motivo dominiovec.ReferenciaEntradaCatalogo
	err    error
}

func (d *resolutorMotivoDoble) ResolverMotivoAutorizacionAltaV3(
	context.Context,
	ports.SolicitudResolverMotivoAutorizacionAltaV3,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	return d.motivo, d.err
}

type generadorReferenciasDoble struct {
	correlacion string
	err         error
}

func (d *generadorReferenciasDoble) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
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
}

func (r *relojMutable) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.instante
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
}

func (d *transaccionAltaDoble) ConfirmarAlta(
	_ context.Context,
	orden ports.OrdenConfirmarAlta,
) (ports.ReciboAlta, error) {
	d.llamadas++
	d.orden = orden
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
	decision, confirmacion, err := concesionAutorizacionV3Prueba(
		d.t,
		solicitud,
		resultado,
		d.motivo,
		d.instante,
		"dec_0123456789abcdef0123456789abcdef",
	)
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
			CorrelacionRef:    "correlacion:operacion-alta-001",
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
	d := &dependenciasRegistro{
		contextos: &resolutorContextoDoble{contexto: escenario.contexto},
		flujos:    &resolutorFlujoDoble{configuracion: escenario.configuracion},
		huellas: &derivadorHuellaDoble{
			huella: escenario.preparacion.HuellaPeticionHMAC,
		},
		ambitos: &selladorAmbitoDoble{
			ambito: escenario.preparacion.AmbitoIdempotenciaHMAC,
		},
		motivos: &resolutorMotivoDoble{motivo: escenario.motivo},
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
	if err != nil ||
		datosSolicitud.Recurso.Referencia != escenario.preparacion.AmbitoIdempotenciaHMAC ||
		datosSolicitud.Accion != ports.AccionCrearSolicitud ||
		datosSolicitud.Finalidad != ports.FinalidadCrearSolicitud ||
		datosSolicitud.Recurso.ModuloID != ports.ModuloContratacion ||
		datosSolicitud.Recurso.Tipo != ports.TipoRecursoExpediente {
		t.Fatalf("solicitud V3 no ligada al ámbito: %#v, %v", datosSolicitud, err)
	}
	datosOrden, err := d.transaccion.orden.Datos()
	_, errSolicitudV3 := datosOrden.SolicitudAutorizacionV3.Datos()
	if err != nil || errSolicitudV3 != nil ||
		datosOrden.DecisionAutorizacionV3.ValidarPara(datosOrden.SolicitudAutorizacionV3) != nil ||
		datosOrden.ConfirmacionRegistroV3.Validar() != nil {
		t.Fatalf("orden sin capacidades V3: %v", err)
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

func selloHMACRegistroPrueba(referencia, digito string) string {
	return "hmac-sha256:" + referencia + ":" + strings.Repeat(digito, 64)
}

type revalidadorVinculoPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (d revalidadorVinculoPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return d.resultado, nil
}

type resolutorResultadoVinculoPrueba struct {
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (d resolutorResultadoVinculoPrueba) ResolverContextoActorRegistradoV2(
	context.Context,
	dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return d.resultado, nil
}

type relojVinculoPrueba struct{ instante time.Time }

func (d relojVinculoPrueba) Ahora() time.Time { return d.instante }

func contextoAutorizacionAltaV3Prueba(
	t *testing.T,
	ahora time.Time,
) ports.ContextoAutorizacionAltaV3 {
	t.Helper()
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    dominiovec.AuthMethodCertificate,
		Garantia:  dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef:      "vca_0123456789abcdefghijkl",
		VinculoVersion:  3,
		CuentaRef:       cuenta.CuentaRef,
		CuentaVersion:   4,
		PersonaRef:      "per_0123456789abcdefghijkl",
		PersonaVersion:  2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl",
		PerfilVersion:   5,
		Estado:          dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde:    ahora.Add(-time.Hour),
		VigenteHasta:    ahora.Add(time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := dominiovec.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:          "prc_0123456789abcdefghijkl",
		ProcedenciaVersion:      1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad:    dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := dominiovec.ManifiestoProcedenciaContextoActorV1{
		Esquema:           dominiovec.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: dominiovec.ProcedenciaCuentaContextoActorV1{
			CuentaRef: instantanea.CuentaRef,
			Version:   instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: dominiovec.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantanea.PersonaRef,
			Version:    instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: dominiovec.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantanea.PerfilActivoRef,
			Version:   instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: dominiovec.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantanea.VinculoRef,
			Version:    instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []dominiovec.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	manifiestoHuella, err := dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		manifiestoCanon,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef:               "rca_0123456789abcdefghijklmn",
		Contexto:                          actor,
		RepresentacionCanonica:            canon,
		HuellaSHA256:                      huella,
		ManifiestoProcedenciaCanonico:     manifiestoCanon,
		ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva: dominiovec.
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo: actor.ResueltoEn,
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef:             "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256:    strings.Repeat("1", 64),
		AsercionRef:                  "ase_0123456789abcdefghijkl",
		SesionRef:                    "ses_0123456789abcdefghijkl",
		ControlSesionRef:             "cse_0123456789abcdefghijkl",
		ControlSesionRevision:        2,
		ControlSesionHuellaSHA256:    strings.Repeat("2", 64),
		CuentaRef:                    cuenta.CuentaRef,
		CuentaOrdinariaRef:           cuenta.CuentaRef,
		Superficie:                   dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:              cuenta.Metodo,
		GarantiaObservada:            cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute),
		SesionValidaHasta:            ahora.Add(20 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
	}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(
		context.Background(),
		revalidadorVinculoPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		resolutorResultadoVinculoPrueba{resultado: resultado},
		dominiovec.SolicitudContextoActor{
			Cuenta:          cuenta,
			PerfilActivoRef: instantanea.PerfilActivoRef,
		},
		relojVinculoPrueba{instante: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}
}

func concesionAutorizacionV3Prueba(
	t *testing.T,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	ahora time.Time,
	referenciaDecision string,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	error,
) {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ambitos := make([]dominiovec.AmbitoPerfil, 0, len(datos.Recurso.Ambitos))
	for clave, valor := range datos.Recurso.Ambitos {
		ambitos = append(ambitos, dominiovec.AmbitoPerfil{Clave: clave, Valores: []string{valor}})
	}
	version := dominiovec.VersionRol{
		RolID:   "tecnico_rrhh",
		Version: 1,
		Nombre:  "Técnico de RRHH",
		Estado:  dominiovec.EstadoVersionRolPublicada,
		Concesiones: []dominiovec.ConcesionRol{{
			Accion: datos.Accion, ModuloID: datos.Recurso.ModuloID,
			TipoRecurso:    datos.Recurso.Tipo,
			Finalidades:    []string{datos.Finalidad},
			GarantiaMinima: dominiovec.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable-seguridad",
		PublicadaEn:  ahora.Add(-24 * time.Hour),
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: dominiovec.AsignacionPerfil{
			AsignacionID:    "asig-contratacion-temporal",
			Version:         1,
			PerfilActivoRef: vinculo.PerfilActivoRef,
			PrincipalID:     vinculo.PrincipalID,
			VersionRolRef:   version.Referencia(),
			Estado:          dominiovec.EstadoAsignacionPerfilActiva,
			Ambitos:         ambitos,
			VigenteDesde:    ahora.Add(-time.Hour),
			VigenteHasta:    ahora.Add(time.Hour),
			EmitidaPor:      "administrador-identidades",
			EmitidaEn:       ahora.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef:  version.Referencia(),
			Revision:       1,
			Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor,
			ActualizadoEn:  version.PublicadaEn,
		},
		RevisionCatalogoPoliticas:     1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := dominiovec.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud,
		instantanea,
		referenciaDecision,
		ahora,
		ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := dominiovec.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		solicitud,
		decision,
		motivo,
		resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := puertosvec.
		RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
			context.Background(),
			registroConcesionV3Doble{registradaEn: ahora},
			orden,
		)
	if err != nil {
		t.Fatal(err)
	}
	return decision, confirmacion, nil
}
