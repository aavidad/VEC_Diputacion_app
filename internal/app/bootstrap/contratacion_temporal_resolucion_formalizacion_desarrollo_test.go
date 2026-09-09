package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pdfvec "vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func solicitudResolucionPrueba() ports.SolicitudResolucionFormalizacion {
	return ports.SolicitudResolucionFormalizacion{ExpedienteRef: "expediente:prueba", VersionEsperada: 7, PropuestaRef: "propuesta:prueba",
		ClaveIdempotencia: "31111111-1111-4111-8111-111111111111", NumeroResolucion: "EJ-2026/1", FechaResolucion: "2026-09-06",
		Motivo: "Revisión manual del ejercicio.", ConfirmaRevisionPropuesta: true, ConfirmaEjercicioManual: true}
}
func reciboResolucionPrueba(s ports.SolicitudResolucionFormalizacion) ports.ResultadoResolucionFormalizacion {
	return ports.ResultadoResolucionFormalizacion{Solicitud: s, Estado: "registrada", ResolucionRef: "resolucion:prueba",
		DocumentoRef: ports.ReferenciaDocumentoResolucion(s.PropuestaRef), DocumentoSHA256: strings.Repeat("a", 64), DocumentoVersion: 7,
		ActuacionRef: "resolucion:prueba", AuditoriaRef: "auditoria:prueba", OutboxRef: "evento:prueba", ReciboRef: "recibo:prueba",
		VersionResultante: 8, RegistradaEn: time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)}
}

func detalleResolucionDesarrolloPrueba() ports.DetalleExpedienteRRHH {
	inicio := time.Date(2026, 9, 6, 1, 0, 0, 0, time.UTC)
	periodo := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	d := ports.DetalleExpedienteRRHH{
		Resumen: ports.ResumenExpedienteRRHH{
			ExpedienteRef: "expediente:ct:sintetico:informe", OrganizacionRef: "organizacion:sintetica:001",
			NumeroVisible: "2026/CT-0001", Version: 7, FlujoRef: "flujo:ct:sintetico", FlujoVersion: 1,
			FlujoHuella: strings.Repeat("a", 64), FaseClave: "nombramiento", EstadoClave: domain.EstadoEnCurso,
			CentroRef: "centro:sintetico:001", CategoriaRef: "categoria:sintetica:c2", ModalidadClave: "sustitucion",
			UnidadRef: "unidad:sintetica:rrhh", CreadoEn: inicio, ActualizadoEn: inicio.Add(6 * time.Minute),
		},
		Solicitud:  ports.SolicitudOperativaRRHH{GrupoSubgrupo: "C2", MotivoClave: "sustitucion", PeriodoInicio: periodo, PeriodoFin: periodo.AddDate(0, 3, 0)},
		Analisis:   &ports.AnalisisOperativoRRHH{ModalidadClave: "sustitucion", CategoriaRef: "categoria:sintetica:c2", CausaClave: "necesidad_temporal", PeriodoInicio: periodo, PeriodoFin: periodo.AddDate(0, 3, 0), PorcentajeJornada: 10000, ResultadoRC: domain.RCValidada},
		Cobertura:  &ports.CoberturaOperativaRRHH{ViaClave: "bolsa_vigente", DecisionGobernada: true},
		Asignacion: &ports.AsignacionOperativaRRHH{UnidadRef: "unidad:sintetica:rrhh", AsignadaEn: inicio},
	}
	for i := uint64(1); i <= 7; i++ {
		h := ports.HitoExpedienteRRHH{Secuencia: i, VersionExpediente: i, AccionClave: "actuacion_sintetica", RealizadaEn: inicio.Add(time.Duration(i-1) * time.Minute), FaseOrigen: "fiscalizacion", FaseDestino: "fiscalizacion", EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso}
		if i == 1 {
			h.FaseOrigen = ""
			h.EstadoOrigen = domain.EstadoPendiente
		}
		if i == 7 {
			h.FaseDestino = "nombramiento"
			h.AccionClave = "registrar_propuesta_formalizacion"
		}
		d.Hitos = append(d.Hitos, h)
	}
	return d
}

type detalleResolucionPrueba struct {
	t        *testing.T
	detalle  ports.DetalleExpedienteRRHH
	llamadas int
	err      error
}

func (d *detalleResolucionPrueba) Consultar(ctx context.Context, s ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	d.llamadas++
	c, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !ok || c.ruta != httpinterno.RutaConsultaDetalleRRHH || c.consultaRRHH == nil || s.VersionObservada() != 7 {
		d.t.Fatal("consulta sin autoridad separada v7")
	}
	return d.detalle, d.err
}

type registroResolucionDesarrolloPrueba struct {
	m        ports.MaterialResolucionFormalizacion
	recibo   ports.ResultadoResolucionFormalizacion
	efectos  int
	llamadas int
}

func (r *registroResolucionDesarrolloPrueba) RegistrarResolucionFormalizacion(ctx context.Context, s ports.SolicitudResolucionFormalizacion) (ports.ResultadoResolucionFormalizacion, error) {
	r.llamadas++
	m, ok := ctx.Value(claveMaterialResolucionFormalizacionDesarrollo{}).(ports.MaterialResolucionFormalizacion)
	if !ok || m.Validar() != nil || m.Solicitud != s {
		return ports.ResultadoResolucionFormalizacion{}, ports.ErrResolucionFormalizacionDenegada
	}
	if r.efectos > 0 {
		if m != r.m {
			return ports.ResultadoResolucionFormalizacion{}, ports.ErrClaveResolucionFormalizacionUsada
		}
		out := r.recibo
		out.Estado = "replay_registrada"
		return out, nil
	}
	r.efectos++
	r.m = m
	r.recibo = reciboResolucionPrueba(s)
	r.recibo.DocumentoSHA256 = m.DocumentoSHA256
	return r.recibo, nil
}

type preparacionResolucionBootstrapPrueba struct {
	t        *testing.T
	p        ports.PreparacionResolucionFormalizacion
	llamadas int
	err      error
}

func (p *preparacionResolucionBootstrapPrueba) ConsultarPreparacionResolucionFormalizacion(ctx context.Context, ref string) (ports.PreparacionResolucionFormalizacion, error) {
	p.llamadas++
	c, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !ok || c.ruta != httpinterno.RutaConsultaDetalleRRHH || c.consultaRRHH == nil ||
		ctx.Value(claveMaterialResolucionFormalizacionDesarrollo{}) != nil {
		p.t.Fatal("sin separación lectura/efecto")
	}
	return p.p, p.err
}
func TestResolucionFormalizacionGETBootstrapNominalSinEfecto(t *testing.T) {
	for _, caso := range []string{"v7", "v8", "sin_lector", "sin_identidad", "otra_ruta", "denegado", "dependencia"} {
		t.Run(caso, func(t *testing.T) {
			p, ctx, _, _, _, _, _ := escenarioAceptacionPuentePrueba(t)
			c, _ := p.alta.soporte.capacidadValida(ctx)
			c.ruta = httpinterno.RutaResolucionFormalizacion
			ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
			s := solicitudResolucionPrueba()
			prep := ports.PreparacionResolucionFormalizacion{ExpedienteRef: s.ExpedienteRef, PropuestaRef: s.PropuestaRef, VersionEsperada: 7, VersionActual: 7}
			if caso == "v8" {
				r := reciboResolucionPrueba(s)
				prep.VersionActual = 8
				prep.Recibo = &r
			}
			lector := &preparacionResolucionBootstrapPrueba{t: t, p: prep}
			efecto := &registroResolucionDesarrolloPrueba{}
			e := &ejecutorResolucionFormalizacionDesarrollo{soporte: p.alta.soporte, preparacion: lector, servicio: efecto, reloj: p.reloj}
			switch caso {
			case "sin_lector":
				e.preparacion = nil
			case "sin_identidad":
				ctx = context.Background()
			case "otra_ruta":
				c.ruta = httpinterno.RutaConsultaDetalleRRHH
				ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
			case "denegado":
				lector.err = ports.ErrAutorizacionDenegada
			case "dependencia":
				lector.err = ports.ErrConsultaRRHHNoDisponible
			}
			r, err := e.ConsultarPreparacionResolucionFormalizacion(ctx, s.ExpedienteRef)
			if efecto.llamadas != 0 {
				t.Fatal("GET invoca efecto")
			}
			if caso == "v7" || caso == "v8" {
				if err != nil || r.ValidarPara(s.ExpedienteRef) != nil || lector.llamadas != 1 {
					t.Fatal(r, err)
				}
			} else if err == nil || r.ExpedienteRef != "" {
				t.Fatal("fallo abierto")
			}
		})
	}
	if _, err := nuevasDependenciasResolucionFormalizacionDesarrollo(nil, nil, nil, nil, nil); err == nil {
		t.Fatal("composición vacía")
	}
}

func TestResolucionFormalizacionDesarrolloFuenteRealYReplaySinEstadoWeb(t *testing.T) {
	p, ctx, _, _, _, _, _ := escenarioAceptacionPuentePrueba(t)
	c, _ := p.alta.soporte.capacidadValida(ctx)
	c.ruta = httpinterno.RutaResolucionFormalizacion
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
	d := detalleResolucionDesarrolloPrueba()
	d.Resumen.OrganizacionRef = organizacionAltaContratacionTemporalDesarrollo
	s := solicitudResolucionPrueba()
	s.ExpedienteRef = d.Resumen.ExpedienteRef
	consulta := &detalleResolucionPrueba{t: t, detalle: d}
	registro := &registroResolucionDesarrolloPrueba{}
	pdf := informejuridico.RenderizadorBorradorDesarrollo{PDF: pdfvec.Renderizador{}}
	e := &ejecutorResolucionFormalizacionDesarrollo{soporte: p.alta.soporte, detalle: consulta, renderizador: pdf, servicio: registro, reloj: p.reloj}
	primero, err := e.RegistrarResolucionFormalizacion(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := pdf.RenderizarBorrador(ctx, ports.BorradorResolucion, d)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(contenido)
	if primero.DocumentoSHA256 != hex.EncodeToString(h[:]) || registro.m.PropuestaConfirmadaEn != d.Hitos[6].RealizadaEn {
		t.Fatal("PDF no ligado a original")
	}
	// Nuevo coordinador, sin estado de formulario; doble durable compartido.
	nuevo := *e
	segundo, err := nuevo.RegistrarResolucionFormalizacion(ctx, s)
	if err != nil || segundo.Estado != "replay_registrada" || segundo.ReciboRef != primero.ReciboRef ||
		!segundo.RegistradaEn.Equal(primero.RegistradaEn) || registro.efectos != 1 || consulta.llamadas != 2 {
		t.Fatal(segundo, err)
	}
	s.Motivo = "Otro material de ejercicio"
	if _, err = nuevo.RegistrarResolucionFormalizacion(ctx, s); !errors.Is(err, ports.ErrClaveResolucionFormalizacionUsada) || registro.efectos != 1 {
		t.Fatal("conflicto sin efectos", err)
	}
	consulta.err = errors.New("fuente no disponible")
	if _, err = nuevo.RegistrarResolucionFormalizacion(ctx, s); !errors.Is(err, ports.ErrResolucionFormalizacionNoDisponible) || registro.llamadas != 3 {
		t.Fatal("fuente fallida no bloqueó", err)
	}
}
func TestResolucionFormalizacionDesarrolloAutoridadLigadaYModos(t *testing.T) {
	s := solicitudResolucionPrueba()
	m := ports.MaterialResolucionFormalizacion{Solicitud: s, OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		DocumentoRef: ports.ReferenciaDocumentoResolucion(s.PropuestaRef), DocumentoVersion: 7, DocumentoSHA256: strings.Repeat("a", 64),
		PropuestaConfirmadaEn: time.Date(2026, 9, 6, 1, 0, 0, 0, time.UTC)}
	ctx := context.WithValue(context.Background(), claveMaterialResolucionFormalizacionDesarrollo{}, m)
	r, _ := postgresct.RecursoResolucionFormalizacion(m)
	d := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: postgresct.AccionResolucionFormalizacion,
		Finalidad: "gestionar_contratacion_temporal", ReferenciaMotivo: motivoResolucionFormalizacionDesarrollo(), Recurso: r}
	if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaResolucionFormalizacion, d) {
		t.Fatal("material válido denegado")
	}
	for _, cambio := range []string{"documento", "motivo", "accion", "organizacion"} {
		otro := d
		otro.Recurso = r
		otro.Recurso.Ambitos = map[string]string{"organizacion_ref": m.OrganizacionRef}
		otro.Recurso.Atributos = map[string]string{"material_sha256": r.Atributos["material_sha256"]}
		switch cambio {
		case "documento":
			otro.Recurso.Atributos["material_sha256"] = strings.Repeat("f", 64)
		case "motivo":
			otro.ReferenciaMotivo = motivoPropuestaFormalizacionDesarrollo()
		case "accion":
			otro.Accion = postgresct.AccionPropuestaFormalizacion
		case "organizacion":
			otro.Recurso.Ambitos["organizacion_ref"] = "organizacion:otra"
		}
		if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaResolucionFormalizacion, otro) {
			t.Fatal(cambio)
		}
	}
	a := &autorizadorLlamamientoDesarrollo{resolucionFormalizacion: true}
	if !a.modoResolucionOContinuacionValido(httpinterno.RutaResolucionFormalizacion) || a.modoResolucionOContinuacionValido(httpinterno.RutaPropuestaFormalizacion) || a.modoResolucionOContinuacionValido(httpinterno.RutaResolucionComunicacionLlamamiento) {
		t.Fatal("modos cruzados")
	}
	a.propuestaFormalizacion = true
	if a.modoResolucionOContinuacionValido(httpinterno.RutaResolucionFormalizacion) {
		t.Fatal("dos modos")
	}
}
