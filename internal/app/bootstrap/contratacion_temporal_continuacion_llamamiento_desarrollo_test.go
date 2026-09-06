package bootstrap

import (
	"context"
	"crypto/x509"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Dobles exclusivos de unidad: comprueban composición y material nominal,
// no acreditan firmas V3, consumo SQL ni durabilidad PostgreSQL.
type autorizacionContinuacionPrueba struct {
	t *testing.T
	puente *puenteBolsaLlamamientoDesarrollo
	etapas []string
	contextos []context.Context
	huellas []string
}

func (a *autorizacionContinuacionPrueba) AutorizarOperacion(ctx context.Context, accion string, recurso dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	etapa, motivo, audiencia := "", motivoContinuacionDesarrollo(false), postgresct.AudienciaRegistroComunicacionLlamamiento
	switch accion {
	case postgresct.AccionContinuacionLlamamiento:
		m, _ := ctx.Value(claveMaterialContinuacionDesarrollo{}).(ports.MaterialContinuacionLlamamiento)
		etapa = "ct:" + m.Etapa
	case postgresct.AccionConsultaJustificanteRespuestaRecibida:
		etapa, motivo = "justificante", motivoConsultaJustificanteRespuestaDesarrollo()
	case puertosbolsa.AccionAbrirSiguienteLlamamientoDesarrollo:
		etapa, motivo, audiencia = "bolsa", motivoContinuacionDesarrollo(true), puertosbolsa.AudienciaIntegracionLlamamientoDesarrollo
	default:
		a.t.Fatal("acción ajena a la continuación", accion)
	}
	d := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: accion, Finalidad: "gestionar_contratacion_temporal", ReferenciaMotivo: motivo, Recurso: recurso}
	if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaContinuacionLlamamiento, d) {
		a.t.Fatal("material no ligado a su etapa", etapa)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil { a.t.Fatal(err) }
	a.etapas, a.contextos, a.huellas = append(a.etapas, etapa), append(a.contextos, ctx), append(a.huellas, huella)
	h, ahora := strings.Repeat("a", 64), a.puente.reloj.Ahora()
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:continuacion:"+strconv.Itoa(len(a.etapas)), h, h, "contexto:unidad", h,
		accion, recurso.Referencia, huella, audiencia, ahora, ahora.Add(5*time.Second))
	if err != nil { a.t.Fatal(err) }
	spki, err := x509.MarshalPKIXPublicKey(a.puente.privadaFuente.Public())
	if err != nil { a.t.Fatal(err) }
	return puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen,
		[]byte("{}"), []byte("{}"), []byte("{}"), 1, 1, []byte("unidad"), []byte("unidad"), []byte("unidad"), spki)
}

type registroContinuacionCoordinadorPrueba struct {
	t *testing.T
	proveedor *proveedorContinuacionLlamamientoDesarrollo
	antecedente ports.AntecedenteContinuacionLlamamiento
	bolsa *repositorioAceptacionPuentePrueba
	fallarConfirmacion bool
	recibidos []ports.ReciboBolsaContinuacion
	confirmado ports.ResultadoContinuacionLlamamiento
}

func (r *registroContinuacionCoordinadorPrueba) LeerAntecedente(ctx context.Context, s ports.SolicitudContinuarLlamamiento) (ports.AntecedenteContinuacionLlamamiento, error) {
	_, err := r.proveedor.AutorizarContinuacionLlamamiento(ctx, ports.MaterialContinuacionLlamamiento{Etapa: "consulta", Solicitud: s})
	if err != nil { return ports.AntecedenteContinuacionLlamamiento{}, err }
	return r.antecedente, nil
}

func (r *registroContinuacionCoordinadorPrueba) Confirmar(ctx context.Context, s ports.SolicitudContinuarLlamamiento, b ports.ReciboBolsaContinuacion) (ports.ResultadoContinuacionLlamamiento, error) {
	_, err := r.proveedor.AutorizarContinuacionLlamamiento(ctx, ports.MaterialContinuacionLlamamiento{Etapa: "confirmacion", Solicitud: s, ReciboBolsa: &b})
	if err != nil { return ports.ResultadoContinuacionLlamamiento{}, err }
	fila, existe := r.bolsa.filas[b.OperacionRef]
	canon, err := fila.Registro.Canonico()
	if !existe || err != nil || fila.ReciboRef != b.ReciboRef || fila.ConfirmadaEn != b.ConfirmadaEn ||
		fila.Registro.Propuesta.OrdenSeleccionado != 3 || b.RegistroSHA256 != huellaPuenteLlamamientoDesarrollo(canon) {
		r.t.Fatal("CT confirmó sin recibir el efecto y recibo del servicio Bolsa")
	}
	r.recibidos = append(r.recibidos, b)
	if r.fallarConfirmacion { return ports.ResultadoContinuacionLlamamiento{}, ports.ErrOperacionContinuacionNoDisponible }
	if r.confirmado.ReciboRef != "" {
		if r.confirmado.Solicitud != s || r.confirmado.ReciboBolsa != b { r.t.Fatal("replay cambió material") }
		copia := r.confirmado
		copia.Estado = "replay_confirmado"
		return copia, nil
	}
	r.confirmado = ports.ResultadoContinuacionLlamamiento{Solicitud: s,
		LlamamientoAnteriorRef: r.antecedente.Resolucion.Solicitud.LlamamientoRef, ReciboBolsa: b,
		ReciboRef: "recibo:continuacion-prueba", AuditoriaRef: "auditoria:continuacion-prueba",
		ConfirmadaEn: b.ConfirmadaEn.Add(time.Microsecond), Estado: "confirmado"}
	return r.confirmado, nil
}

type lectorJustificanteContinuacionPrueba struct {
	proveedor *proveedorConsultaJustificanteRespuestaDesarrollo
	justificante ports.JustificanteRespuestaRecibida
}

func (l *lectorJustificanteContinuacionPrueba) ConsultarJustificanteRespuestaRecibida(ctx context.Context, s ports.SolicitudResolverLlamamiento) (ports.JustificanteRespuestaRecibida, error) {
	if _, err := l.proveedor.AutorizarConsultaJustificanteRespuestaRecibida(ctx, s); err != nil { return ports.JustificanteRespuestaRecibida{}, err }
	return l.justificante, nil
}

type escenarioContinuacionCoordinador struct {
	ctx context.Context
	ejecutor *ejecutorComunicacionLlamamientoDesarrollo
	solicitud ports.SolicitudContinuarLlamamiento
	registro *registroContinuacionCoordinadorPrueba
	lector *lectorExpedienteConsultaJustificantePrueba
	autoridad *autorizacionContinuacionPrueba
}

func escenarioContinuacionCoordinadorPrueba(t *testing.T) escenarioContinuacionCoordinador {
	t.Helper()
	_, _, l := escenarioRevisionManualPrueba(t)
	p, ctx, _, seleccion, _, repo, permisoRenuncia := escenarioAceptacionPuentePrueba(t)
	if seleccion.OperacionRef != l.justificante.Seleccion.OperacionRef || seleccion.Propuesta != l.justificante.Seleccion.Propuesta { t.Fatal("fixtures de apertura distintos") }
	l.solicitud.Respuesta = ports.RespuestaLlamamientoRenunciada
	l.justificante.Respuesta.Solicitud.Respuesta = ports.RespuestaLlamamientoRenunciada
	l.local.Solicitud = l.solicitud
	l.local.IntencionSiguiente = ports.IntencionOutboxSiguienteCandidato{
		Solicitud: l.solicitud, ResolucionRef: l.local.ResolucionRef, LlamamientoRef: l.solicitud.LlamamientoRef,
		ClaveIdempotencia: l.solicitud.ClaveIdempotencia, VersionEsperada: 2, VersionResultante: 3,
		IntencionRef: "intencion:f4bd0049-8b96-410b-8144-4384bdb47ed0", ComandoOpacoRef: "comando:renuncia-prueba",
		Estado: ports.OutboxSiguienteCandidatoPendiente, ActualizadaEn: l.local.ResueltaEn}
	if l.justificante.ValidarPara(l.solicitud) != nil { t.Fatal("justificante de renuncia inválido") }
	// La renuncia pasa por el puente, dominio y servicio existentes. Su instante
	// Bolsa es posterior al de CT; no se impone igualdad entre ambos relojes.
	p.autorizadorRenuncia = permisoRenuncia
	repo.reloj.instante = repo.reloj.instante.Add(time.Second)
	_, err := p.RenunciarRespuestaRRHH(ctx, l.solicitud, l.justificante.Seleccion, puertosbolsa.ResolucionLlamamientoDesarrollo{
		AperturaOperacionRef: seleccion.OperacionRef, JustificanteRef: l.solicitud.PruebaRespuestaRef,
		EvaluacionPlazoRef: l.local.EvaluacionPlazoRef, PoliticaRef: l.local.Politica.Referencia,
		PoliticaVersion: l.local.Politica.Version, PoliticaSHA256: l.local.Politica.HuellaSHA256, VersionEsperada: 1})
	if err != nil { t.Fatal("renuncia previa", err) }
	s := ports.SolicitudContinuarLlamamiento{ClaveIdempotencia: "33333333-3333-4333-8333-333333333333",
		OrganizacionRef: l.solicitud.OrganizacionRef, ExpedienteRef: l.solicitud.ExpedienteRef,
		ResolucionRef: l.local.ResolucionRef, IntencionRef: l.local.IntencionSiguiente.IntencionRef}
	a := ports.AntecedenteContinuacionLlamamiento{Resolucion: l.local, ComandoSiguienteRef: l.local.IntencionSiguiente.ComandoOpacoRef,
		ComandoSiguiente: ports.ComandoSiguienteLlamamiento{Esquema: "vec.contratacion-temporal.siguiente-candidato.intencion.v1",
			ComandoRef: l.local.IntencionSiguiente.ComandoOpacoRef, IntencionRef: s.IntencionRef,
			OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, LlamamientoRef: l.solicitud.LlamamientoRef,
			JustificanteRef: l.solicitud.PruebaRespuestaRef, SeleccionClave: "11111111-1111-4111-8111-111111111111"}}
	if a.ValidarPara(s) != nil { t.Fatal("antecedente de continuación inválido") }
	capacidad, _ := p.alta.soporte.capacidadValida(ctx)
	capacidad.ruta = httpinterno.RutaContinuacionLlamamiento
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	autoridad := &autorizacionContinuacionPrueba{t: t, puente: p}
	p.autorizadorSiguiente = autoridad
	registro := &registroContinuacionCoordinadorPrueba{t: t, antecedente: a, bolsa: repo,
		proveedor: &proveedorContinuacionLlamamientoDesarrollo{soporte: p.alta.soporte, autorizador: autoridad, reloj: p.reloj}}
	lector := &lectorExpedienteConsultaJustificantePrueba{expediente: expedientePuenteBolsaPrueba(t)}
	j := &lectorJustificanteContinuacionPrueba{justificante: l.justificante,
		proveedor: &proveedorConsultaJustificanteRespuestaDesarrollo{soporte: p.alta.soporte, autorizador: autoridad, reloj: p.reloj}}
	e := &ejecutorComunicacionLlamamientoDesarrollo{soporte: p.alta.soporte, lector: lector, lectorJustificante: j, continuaciones: registro, continuador: p}
	return escenarioContinuacionCoordinador{ctx: ctx, ejecutor: e, solicitud: s, registro: registro, lector: lector, autoridad: autoridad}
}

func TestContinuacionLlamamientoDesarrolloSinIdentidadNoLee(t *testing.T) {
	f := escenarioContinuacionCoordinadorPrueba(t)
	for _, ctx := range []context.Context{context.Background(), contextoConOtraRutaContinuacionPrueba(f, httpinterno.RutaResolucionComunicacionLlamamiento)} {
		r, err := f.ejecutor.Continuar(ctx, f.solicitud)
		if !errors.Is(err, ports.ErrOperacionContinuacionDenegada) || r != (ports.ResultadoContinuacionLlamamiento{}) || f.lector.llamadas != 0 || len(f.autoridad.etapas) != 0 {
			t.Fatal("continuación leyó sin identidad o con la ruta de resolución", err)
		}
	}
}

func contextoConOtraRutaContinuacionPrueba(f escenarioContinuacionCoordinador, ruta string) context.Context {
	c, _ := f.ejecutor.soporte.capacidadValida(f.ctx)
	c.ruta = ruta
	return context.WithValue(f.ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
}

func TestContinuacionLlamamientoDesarrolloCorteCTYReplayAutorizado(t *testing.T) {
	f := escenarioContinuacionCoordinadorPrueba(t)
	f.registro.fallarConfirmacion = true
	var primero ports.ResultadoContinuacionLlamamiento
	for intento := 0; intento < 3; intento++ {
		r, err := f.ejecutor.Continuar(f.ctx, f.solicitud)
		if intento == 0 {
			if !errors.Is(err, ports.ErrOperacionContinuacionNoDisponible) || r != (ports.ResultadoContinuacionLlamamiento{}) || f.registro.confirmado.ReciboRef != "" { t.Fatal("éxito parcial tras fallo CT", err) }
		} else {
			if err != nil || r.ValidarPara(f.solicitud) != nil || r.Estado != []string{"", "confirmado", "replay_confirmado"}[intento] { t.Fatal("recuperación falló", err) }
			if intento == 1 { primero = r } else if r.ReciboRef != primero.ReciboRef || r.ConfirmadaEn != primero.ConfirmadaEn || r.ReciboBolsa != primero.ReciboBolsa { t.Fatal("replay perdió recibo o fecha") }
		}
		if len(f.registro.bolsa.filas) != 4 || len(f.registro.recibidos) != intento+1 { t.Fatal("efectos omitidos o duplicados") }
		if f.registro.recibidos[intento] != f.registro.recibidos[0] { t.Fatal("reintento cambió operación o recibo Bolsa") }
		f.registro.fallarConfirmacion = false
		f.registro.bolsa.reloj.instante = f.registro.bolsa.reloj.instante.Add(time.Minute)
	}
	secuencia := []string{"ct:consulta", "justificante", "bolsa", "ct:confirmacion"}
	if len(f.autoridad.etapas) != 12 { t.Fatal("falta autorización fresca", f.autoridad.etapas) }
	for i := 0; i < 3; i++ {
		if !reflect.DeepEqual(f.autoridad.etapas[4*i:4*i+4], secuencia) { t.Fatal("permisos omitidos o desordenados", f.autoridad.etapas) }
		if f.autoridad.huellas[4*i] == f.autoridad.huellas[4*i+3] { t.Fatal("lectura y confirmación comparten material") }
	}
	if f.lector.llamadas != 3 || f.registro.antecedente.Resolucion.IntencionSiguiente.Estado != ports.OutboxSiguienteCandidatoPendiente { t.Fatal("se omitió relectura o se alteró la renuncia original") }
	// El permiso de consulta no puede reutilizarse para confirmar otro material.
	consultaCtx := f.autoridad.contextos[0]
	m := ports.MaterialContinuacionLlamamiento{Etapa: "confirmacion", Solicitud: f.solicitud, ReciboBolsa: &primero.ReciboBolsa}
	if _, err := f.registro.proveedor.AutorizarContinuacionLlamamiento(consultaCtx, m); !errors.Is(err, ports.ErrOperacionContinuacionDenegada) || len(f.autoridad.etapas) != 12 { t.Fatal("contexto de consulta autorizó confirmación") }
	for _, a := range []*autorizadorLlamamientoDesarrollo{{continuacionCT: true, siguienteBolsa: true}, {renunciaBolsa: true}, {comunicacion: true}} {
		if a.modoResolucionOContinuacionValido(httpinterno.RutaContinuacionLlamamiento) { t.Fatal("modos de autorización confundidos") }
	}
}
