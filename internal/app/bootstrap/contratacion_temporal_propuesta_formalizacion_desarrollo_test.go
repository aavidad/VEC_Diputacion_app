package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Dobles de composición, no acreditación criptográfica ni persistencia real.
type registroPropuestaCoordinadorPrueba struct {
	t                                  *testing.T
	antecedente                        ports.AntecedentePropuestaFormalizacion
	proveedor                          *proveedorPropuestaFormalizacionDesarrollo
	consultas, confirmaciones          int
	huellaConsulta, huellaConfirmacion string
	err                                error
	resultado                          ports.ResultadoPropuestaFormalizacion
}

func (r *registroPropuestaCoordinadorPrueba) validar(ctx context.Context, m ports.MaterialPropuestaFormalizacion) string {
	r.t.Helper()
	recurso, err := postgresct.RecursoPropuestaFormalizacion(m)
	if err != nil {
		r.t.Fatal(err)
	}
	d := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: postgresct.AccionPropuestaFormalizacion,
		Finalidad: "gestionar_contratacion_temporal", ReferenciaMotivo: motivoPropuestaFormalizacionDesarrollo(), Recurso: recurso}
	if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaPropuestaFormalizacion, d) {
		r.t.Fatal("material de propuesta no ligado", m.Etapa)
	}
	d.ReferenciaMotivo = motivoContinuacionDesarrollo(false)
	if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaPropuestaFormalizacion, d) {
		r.t.Fatal("permiso de continuación aceptado para propuesta")
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		r.t.Fatal(err)
	}
	return h
}

func (r *registroPropuestaCoordinadorPrueba) LeerAntecedente(ctx context.Context, s ports.SolicitudPropuestaFormalizacion) (ports.AntecedentePropuestaFormalizacion, error) {
	r.consultas++
	r.huellaConsulta = r.validar(ctx, ports.MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s})
	return r.antecedente, nil
}

func (r *registroPropuestaCoordinadorPrueba) ConfirmarPropuesta(ctx context.Context, s ports.SolicitudPropuestaFormalizacion) (ports.ResultadoPropuestaFormalizacion, error) {
	m, err := r.proveedor.PrepararPropuestaFormalizacion(ctx, s)
	if err != nil {
		return ports.ResultadoPropuestaFormalizacion{}, err
	}
	r.confirmaciones++
	r.huellaConfirmacion = r.validar(ctx, m)
	if r.err != nil {
		return ports.ResultadoPropuestaFormalizacion{}, r.err
	}
	if r.resultado.PropuestaRef != "" {
		replay := r.resultado.Clonar()
		replay.Estado = ports.ResultadoPropuestaFormalizacionReplay
		return replay, nil
	}
	r.resultado = ports.ResultadoPropuestaFormalizacion{Solicitud: s.Clonar(), PropuestaRef: "propuesta:nombramiento-prueba",
		ReciboLocalRef: "recibo:propuesta-prueba", AuditoriaRef: "auditoria:propuesta-prueba", VersionResultante: 7,
		ConfirmadaEn: m.AceptacionBolsa.ResueltaEn.Add(time.Second), Estado: ports.ResultadoPropuestaFormalizacionConfirmado}
	return r.resultado.Clonar(), nil
}

type escenarioPropuestaPrueba struct {
	ctx       context.Context
	ejecutor  *ejecutorPropuestaFormalizacionDesarrollo
	solicitud ports.SolicitudPropuestaFormalizacion
	registro  *registroPropuestaCoordinadorPrueba
	lector    *lectorExpedienteConsultaJustificantePrueba
	bolsa     *repositorioAceptacionPuentePrueba
	terminal  string
}

func nuevaPropuestaCoordinadorPrueba(t *testing.T) escenarioPropuestaPrueba {
	t.Helper()
	_, _, l := escenarioRevisionManualPrueba(t)
	p, ctx, _, seleccion, _, bolsa, _ := escenarioAceptacionPuentePrueba(t)
	if seleccion.OperacionRef != l.justificante.Seleccion.OperacionRef {
		t.Fatal("antecedentes distintos")
	}
	bolsa.reloj.instante = bolsa.reloj.instante.Add(time.Second)
	b, err := p.AceptarRespuestaRRHH(ctx, l.solicitud, l.justificante.Seleccion, puertosbolsa.ResolucionLlamamientoDesarrollo{
		AperturaOperacionRef: seleccion.OperacionRef, JustificanteRef: l.solicitud.PruebaRespuestaRef,
		EvaluacionPlazoRef: l.local.EvaluacionPlazoRef, PoliticaRef: l.local.Politica.Referencia,
		PoliticaVersion: l.local.Politica.Version, PoliticaSHA256: l.local.Politica.HuellaSHA256, VersionEsperada: 1})
	if err != nil {
		t.Fatal(err)
	}
	pub, err := cargarPublicacionesPropuestaDesarrollo()
	if err != nil {
		t.Fatal(err)
	}
	s := ports.SolicitudPropuestaFormalizacion{ClaveIdempotencia: "33333333-3333-4333-8333-333333333333",
		OrganizacionRef: l.solicitud.OrganizacionRef, ExpedienteRef: l.solicitud.ExpedienteRef, LlamamientoRef: l.solicitud.LlamamientoRef,
		ResolucionLlamamientoAceptadaRef: l.local.ResolucionRef, ReciboResolucionAceptadaRef: l.local.ReciboLocalRef,
		VersionEsperada: 6, TipoFormalizacion: pub["tipo_formalizacion"], Plantilla: pub["plantilla"],
		PoliticaFirma: pub["politica_firma"], PlanFirma: pub["plan_firma"]}
	a := ports.AntecedentePropuestaFormalizacion{Resolucion: l.local, Justificante: l.justificante, SeleccionClave: "11111111-1111-4111-8111-111111111111"}
	if a.ValidarPara(s) != nil {
		t.Fatal("antecedente de propuesta inválido")
	}
	c, _ := p.alta.soporte.capacidadValida(ctx)
	c.ruta = httpinterno.RutaPropuestaFormalizacion
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
	registro := &registroPropuestaCoordinadorPrueba{t: t, antecedente: a,
		proveedor: &proveedorPropuestaFormalizacionDesarrollo{soporte: p.alta.soporte, reloj: p.reloj, publicaciones: pub}}
	servicio, err := application.NuevoServicioPropuestaFormalizacion(registro)
	if err != nil {
		t.Fatal(err)
	}
	lector := &lectorExpedienteConsultaJustificantePrueba{expediente: expedientePuenteBolsaPrueba(t)}
	e := &ejecutorPropuestaFormalizacionDesarrollo{soporte: p.alta.soporte, lector: lector, registro: registro,
		bolsa: bolsa, publicaciones: pub, servicio: servicio}
	return escenarioPropuestaPrueba{ctx: ctx, ejecutor: e, solicitud: s, registro: registro, lector: lector, bolsa: bolsa, terminal: b.Registro.OperacionRef}
}

func TestPropuestaFormalizacionDesarrolloRecuperaConVersionAvanzada(t *testing.T) {
	f := nuevaPropuestaCoordinadorPrueba(t)
	guardados := f.bolsa.guardados
	primero, err := f.ejecutor.PrepararYConfirmar(f.ctx, f.solicitud)
	if err != nil || primero.ValidarPara(f.solicitud) != nil || primero.EsReplayConfirmado() {
		t.Fatal("confirmación", err)
	}
	f.lector.expediente.VersionActual = 7
	replay, err := f.ejecutor.PrepararYConfirmar(f.ctx, f.solicitud)
	if err != nil || !replay.EsReplayConfirmado() || replay.ReciboLocalRef != primero.ReciboLocalRef ||
		replay.PropuestaRef != primero.PropuestaRef || !replay.ConfirmadaEn.Equal(primero.ConfirmadaEn) {
		t.Fatal("replay v7", err)
	}
	if f.registro.consultas != 2 || f.registro.confirmaciones != 2 || f.registro.huellaConsulta == f.registro.huellaConfirmacion ||
		f.bolsa.guardados != guardados {
		t.Fatal("permisos/etapas o aceptación repetida")
	}
}

func TestPropuestaFormalizacionDesarrolloNoGuardaSinAceptacionBolsa(t *testing.T) {
	f := nuevaPropuestaCoordinadorPrueba(t)
	delete(f.bolsa.filas, f.terminal)
	r, err := f.ejecutor.PrepararYConfirmar(f.ctx, f.solicitud)
	if !errors.Is(err, application.ErrResolucionFormalizacionNoAceptada) || !r.EsCero() || f.registro.confirmaciones != 0 {
		t.Fatal("propuesta sin terminal aceptado", err)
	}
}

func TestPropuestaFormalizacionDesarrolloNoConfundePermisosNiPublicaciones(t *testing.T) {
	for _, caso := range []string{"sin_identidad", "otra_ruta", "otra_organizacion", "hash_plantilla", "version_resolucion", "recibo_ajeno"} {
		t.Run(caso, func(t *testing.T) {
			f := nuevaPropuestaCoordinadorPrueba(t)
			s, ctx := f.solicitud.Clonar(), f.ctx
			switch caso {
			case "sin_identidad":
				ctx = context.Background()
			case "otra_ruta":
				c, _ := f.ejecutor.soporte.capacidadValida(ctx)
				c.ruta = httpinterno.RutaResolucionComunicacionLlamamiento
				ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
			case "otra_organizacion":
				s.OrganizacionRef = "organizacion:ajena"
			case "hash_plantilla":
				s.Plantilla.HuellaSHA256 = strings.Repeat("a", 64)
			case "version_resolucion":
				s.VersionEsperada = 3
			case "recibo_ajeno":
				s.ReciboResolucionAceptadaRef = "recibo:ajeno"
			}
			r, err := f.ejecutor.PrepararYConfirmar(ctx, s)
			if err == nil || !r.EsCero() || f.registro.confirmaciones != 0 {
				t.Fatal("antecedente ajeno admitido", err)
			}
			if caso != "recibo_ajeno" && f.lector.llamadas != 0 {
				t.Fatal("leyó sin identidad/publicaciones propias")
			}
		})
	}
}

func TestPropuestaFormalizacionDesarrolloFalloCommitNoDevuelveRecibo(t *testing.T) {
	f := nuevaPropuestaCoordinadorPrueba(t)
	f.registro.err = errors.New("commit incierto")
	r, err := f.ejecutor.PrepararYConfirmar(f.ctx, f.solicitud)
	if err == nil || !r.EsCero() || f.registro.confirmaciones != 1 {
		t.Fatal("éxito falso o reintento automático", err)
	}
}

func TestPropuestaFormalizacionDesarrolloModosAutoridadExclusivos(t *testing.T) {
	a := &autorizadorLlamamientoDesarrollo{propuestaFormalizacion: true}
	if !a.modoResolucionOContinuacionValido(httpinterno.RutaPropuestaFormalizacion) ||
		a.modoResolucionOContinuacionValido(httpinterno.RutaResolucionComunicacionLlamamiento) ||
		a.modoResolucionOContinuacionValido(httpinterno.RutaContinuacionLlamamiento) {
		t.Fatal("permiso confundido")
	}
	a.aceptacionBolsa = true
	if a.modoResolucionOContinuacionValido(httpinterno.RutaPropuestaFormalizacion) {
		t.Fatal("dos permisos mezclados")
	}
}
