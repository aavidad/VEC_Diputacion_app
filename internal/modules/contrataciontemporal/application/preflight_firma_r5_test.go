package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type disponibilidadPreflightPrueba struct {
	err     error
	alterar func(*ports.PreparacionViaFirmaR5)
	vistas  []ports.SolicitudDisponibilidadFirmaR5
}

func (f *disponibilidadPreflightPrueba) PrepararViasFirmaR5(_ context.Context, q ports.SolicitudDisponibilidadFirmaR5) ([]ports.PreparacionViaFirmaR5, error) {
	f.vistas = append(f.vistas, q)
	if f.err != nil {
		return nil, f.err
	}
	d := ports.PreparacionViaFirmaR5{Solicitud: q, Via: ports.ViaFirmaCertificadoVEC,
		Original:    ports.ComprobanteComponenteFirmaR5{Referencia: q.Preflight.OriginalRef, HuellaSHA256: q.OriginalHuella},
		PreparadaEn: time.Now().UTC().Truncate(time.Microsecond).Add(-time.Second), ValidaHasta: time.Now().UTC().Truncate(time.Microsecond).Add(time.Minute)}
	if f.alterar != nil {
		f.alterar(&d)
	}
	return []ports.PreparacionViaFirmaR5{d}, nil
}

func solicitudPreflightPrueba() ports.SolicitudPreflightFirmaR5 {
	return ports.SolicitudPreflightFirmaR5{Canal: ports.SolicitudConsultaCircuitoRRHH{
		AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa", SesionRef: "ses_bbbbbbbbbbbbbbbbbbbbbbbb",
		PerfilRef: "prf_cccccccccccccccccccccccc", OrganizacionRef: "organizacion:ct:001",
		ExpedienteRef: "expediente:ct:001", VersionObservada: 7}, Documento: "informe_definitivo", OriginalRef: "original:ct:001", OriginalVersion: 1}
}

func servicioPreflightPrueba(d ports.PreparadorViasFirmaR5) (*ServicioPreflightFirmaR5, *autorizadorConsultaFirmasR5Prueba, *registroExternoPrueba) {
	registro := &registroExternoPrueba{historiaHuella: strings.Repeat("d", 64)}
	autorizador := &autorizadorConsultaFirmasR5Prueba{}
	base := &ServicioFirmaDocumento{circuito: circuitoFirmaPrueba{}, original: &originalExternoPrueba{contenido: []byte("original sintetico")}}
	return &ServicioPreflightFirmaR5{firmas: &ServicioFirmaExterna{base: base, consulta: autorizador, registro: registro}, disponibilidad: d}, autorizador, registro
}

// Estas pruebas empiezan después de la resolución central del actor. No
// acreditan un adaptador de disponibilidad real ni el montaje de HTTP.
func TestPreflightFirmaR5NilMantieneViasCerradasConLecturaNominal(t *testing.T) {
	s, a, registro := servicioPreflightPrueba(nil)
	q := solicitudPreflightPrueba()
	r, err := s.consultarNominal(context.Background(), q, "per_actual_sintetico_001", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if r.PasoPendiente != 1 || r.ViasDisponibles == nil || len(r.ViasDisponibles) != 0 || len(registro.registrado) != 0 || len(a.vistas) != 1 {
		t.Fatalf("preflight no cerrado: %+v", r)
	}
	m := a.vistas[0]
	if m.FirmantePrincipalCandidatoRef != "per_actual_sintetico_001" || m.VersionExpediente != 7 || m.Documento != q.Documento || m.ClaveIdempotencia == "" {
		t.Fatal("lectura no ligada al actor y material exactos")
	}
}

func TestPreflightFirmaR5PreparacionExigeLigadurasExactas(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*ports.PreparacionViaFirmaR5)
	}{
		{"actor", func(d *ports.PreparacionViaFirmaR5) { d.Solicitud.ActorRef = "per_ajeno_sintetico_002" }},
		{"perfil", func(d *ports.PreparacionViaFirmaR5) { d.Solicitud.PerfilRef = "perfil:ajeno" }},
		{"documento", func(d *ports.PreparacionViaFirmaR5) { d.Solicitud.Preflight.Documento = "resolucion" }},
		{"original", func(d *ports.PreparacionViaFirmaR5) { d.Solicitud.Preflight.OriginalVersion = 2 }},
		{"paso", func(d *ports.PreparacionViaFirmaR5) { d.Solicitud.PasoOrden = 2 }},
		{"catalogo", func(d *ports.PreparacionViaFirmaR5) { d.Solicitud.CatalogoHuella = strings.Repeat("f", 64) }},
		{"original_componente", func(d *ports.PreparacionViaFirmaR5) { d.Original.HuellaSHA256 = strings.Repeat("f", 64) }},
		{"caducada", func(d *ports.PreparacionViaFirmaR5) {
			d.ValidaHasta = time.Now().UTC().Truncate(time.Microsecond).Add(-time.Second)
		}},
		{"futura", func(d *ports.PreparacionViaFirmaR5) {
			d.PreparadaEn = time.Now().UTC().Truncate(time.Microsecond).Add(time.Minute)
		}},
		{"via_desconocida", func(d *ports.PreparacionViaFirmaR5) { d.Via = "otra" }},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			d := &disponibilidadPreflightPrueba{alterar: tc.alterar}
			s, _, _ := servicioPreflightPrueba(d)
			r, err := s.consultarNominal(context.Background(), solicitudPreflightPrueba(), "per_actual_sintetico_001", strings.Repeat("a", 64))
			if !errors.Is(err, ports.ErrPreflightFirmaR5NoConfiable) || len(r.ViasDisponibles) != 0 {
				t.Fatalf("evidencia aceptada: %+v %v", r, err)
			}
		})
	}
	d := &disponibilidadPreflightPrueba{}
	s, _, registro := servicioPreflightPrueba(d)
	q := solicitudPreflightPrueba()
	r, err := s.consultarNominal(context.Background(), q, "per_actual_sintetico_001", strings.Repeat("a", 64))
	if err != nil || !reflect.DeepEqual(r.ViasDisponibles, []string{ports.ViaFirmaCertificadoVEC}) || len(registro.registrado) != 0 {
		t.Fatalf("preflight comprobado: %+v %v", r, err)
	}
}

func TestPreflightFirmaR5DenegacionNoInvocaDisponibilidad(t *testing.T) {
	d := &disponibilidadPreflightPrueba{}
	s, a, _ := servicioPreflightPrueba(d)
	a.err = ports.ErrFirmaDocumentoDenegada
	r, err := s.consultarNominal(context.Background(), solicitudPreflightPrueba(), "per_actual_sintetico_001", strings.Repeat("a", 64))
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(d.vistas) != 0 || len(r.ViasDisponibles) != 0 {
		t.Fatalf("denegacion: %+v %v", r, err)
	}
}

func TestPreflightFirmaR5ReleePasoPendienteExacto(t *testing.T) {
	s, a, registro := servicioPreflightPrueba(nil)
	registro.firmas = []ports.FirmaRegistrada{{FirmaRef: "firma:ct:001", ReciboRef: "recibo:ct:001", Documento: "informe_definitivo",
		Secuencia: 1, PasoOrden: 1, Resultado: domain.ResultadoFirmaFirmado, CatalogoHuella: strings.Repeat("c", 64),
		OriginalHuella: strings.Repeat("b", 64), FirmadoHuella: strings.Repeat("f", 64), ExpedienteVersion: 6}}
	r, err := s.consultarNominal(context.Background(), solicitudPreflightPrueba(), "per_actual_sintetico_001", strings.Repeat("a", 64))
	if err != nil || r.PasoPendiente != 2 || len(a.vistas) != 2 || a.vistas[1].PasoOrden != 2 {
		t.Fatalf("paso no autorizado: %+v %v", r, err)
	}
}

func TestPreflightFirmaR5DisponibilidadCaidaNoSeConfundeConExito(t *testing.T) {
	d := &disponibilidadPreflightPrueba{err: errors.New("detalle privado")}
	s, _, _ := servicioPreflightPrueba(d)
	r, err := s.consultarNominal(context.Background(), solicitudPreflightPrueba(), "per_actual_sintetico_001", strings.Repeat("a", 64))
	if !errors.Is(err, ports.ErrPreflightFirmaR5NoDisponible) || len(r.ViasDisponibles) != 0 {
		t.Fatalf("caida: %+v %v", r, err)
	}
}

type contextoPreflightPrueba struct{ llamadas int }

func (r *contextoPreflightPrueba) ResolverContextoAutorizacionAltaV3(context.Context, ports.SolicitudResolverContextoAutorizacionAltaV3) (ports.ContextoAutorizacionAltaV3, error) {
	r.llamadas++
	return ports.ContextoAutorizacionAltaV3{}, nil
}

func TestPreflightFirmaR5ContextoVacioYCancelacionDenieganAntesDeLeer(t *testing.T) {
	s, a, _ := servicioPreflightPrueba(nil)
	contextos := &contextoPreflightPrueba{}
	s.contextos = contextos
	r, err := s.Consultar(context.Background(), solicitudPreflightPrueba())
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(a.vistas) != 0 || len(r.ViasDisponibles) != 0 {
		t.Fatalf("contexto vacío aceptado: %+v %v", r, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.Consultar(ctx, solicitudPreflightPrueba())
	if !errors.Is(err, context.Canceled) || contextos.llamadas != 1 || len(a.vistas) != 0 {
		t.Fatal("cancelación ejecutó lectura")
	}
	var nulo *contextoPreflightPrueba
	if _, err := NuevoServicioPreflightFirmaR5(s.firmas, nulo, nil); !errors.Is(err, ports.ErrPreflightFirmaR5NoDisponible) {
		t.Fatal("resolutor nil aceptado")
	}
}

func TestPreflightFirmaR5CambioDeCabezaDuranteDisponibilidadDescartaVias(t *testing.T) {
	d := &disponibilidadPreflightPrueba{}
	s, _, registro := servicioPreflightPrueba(d)
	d.alterar = func(_ *ports.PreparacionViaFirmaR5) { registro.historiaRevision++ }
	r, err := s.consultarNominal(context.Background(), solicitudPreflightPrueba(), "per_actual_sintetico_001", strings.Repeat("a", 64))
	if !errors.Is(err, ports.ErrPreflightFirmaR5NoDisponible) || len(r.ViasDisponibles) != 0 {
		t.Fatalf("cabeza sustituida aceptada: %+v %v", r, err)
	}
}

type registroPreflightCoincidenciaPrueba struct{ *registroExternoPrueba }

func (r registroPreflightCoincidenciaPrueba) ConsultarFirmasAutorizadas(ctx context.Context, m ports.MaterialConsultaFirmasR5, c ports.CapacidadConsultaFirmasR5) (ports.LecturaFirmasR5, error) {
	lectura, err := r.registroExternoPrueba.ConsultarFirmasAutorizadas(ctx, m, c)
	lectura.CoincideFirmanteEnOtroPaso = true
	lectura.HistoriaSeparacionAcreditada = true
	return lectura, err
}

func TestPreflightFirmaR5NoConfundeRegistradorExternoConFirmante(t *testing.T) {
	for _, via := range []string{ports.ViaFirmaCertificadoVEC, ports.ViaFirmaExternaPortafirmas} {
		d := &disponibilidadPreflightPrueba{alterar: func(p *ports.PreparacionViaFirmaR5) { p.Via = via }}
		s, _, registro := servicioPreflightPrueba(d)
		s.firmas.registro = registroPreflightCoincidenciaPrueba{registro}
		r, err := s.consultarNominal(context.Background(), solicitudPreflightPrueba(), "per_actual_sintetico_001", strings.Repeat("a", 64))
		if err != nil {
			t.Fatal(err)
		}
		esperado := 0
		if via == ports.ViaFirmaExternaPortafirmas {
			esperado = 1
		}
		if len(r.ViasDisponibles) != esperado {
			t.Fatalf("identidades mezcladas para %s: %+v", via, r)
		}
	}
}

func TestPreflightFirmaR5RechazaCapacidadDeOtroContextoAntesDeConsumir(t *testing.T) {
	casos := []struct {
		nombre, ref, huella string
		deniega             bool
	}{
		{"actual", "contexto-consulta-firmas-r5-001", strings.Repeat("a", 64), false},
		{"otro_vinculo", "contexto:otro:001", strings.Repeat("a", 64), true},
		{"otra_revision", "contexto-consulta-firmas-r5-001", strings.Repeat("b", 64), true},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			s, a, registro := servicioPreflightPrueba(nil)
			s.firmas.consulta = consultaPreflightR5ContextoActual{origen: a, contextoRef: tc.ref, contextoHuella: tc.huella}
			_, err := s.consultarNominal(context.Background(), solicitudPreflightPrueba(), "per_actual_sintetico_001", strings.Repeat("a", 64))
			if tc.deniega {
				if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || len(registro.consultas) != 0 {
					t.Fatalf("capacidad ajena consumida: %v", err)
				}
			} else if err != nil || len(registro.consultas) != 1 {
				t.Fatalf("contexto exacto rechazado: %v", err)
			}
		})
	}
}

// Detiene la consulta en el autorizador para comprobar el material previo a
// canonización, capacidad y acceso SQL; no reescribe el material en el emisor.
type capturaAmbitosPreflightV2 struct {
	material ports.MaterialConsultaFirmasR5V2
}

func (a *capturaAmbitosPreflightV2) AutorizarConsultaFirmasR5V2(_ context.Context, m ports.MaterialConsultaFirmasR5V2) (ports.CapacidadConsultaFirmasR5V2, error) {
	a.material = m
	return ports.CapacidadConsultaFirmasR5V2{}, ports.ErrFirmaDocumentoDenegada
}
func TestPreflightV2UsaViaYUnidadNominalAntesDeAutorizar(t *testing.T) {
	s, _, _ := servicioPreflightPrueba(nil)
	a := &capturaAmbitosPreflightV2{}
	s.firmas.multiple = &dependenciasFirmaMultipleR5{consulta: a}
	s.viaConsulta, s.unidadConsulta = ports.ViaFirmaCertificadoVEC, "unidad:rrhh"
	q := solicitudPreflightPrueba()
	_, _, _, err := s.leerHistoriaPreflightFirmaR5(context.Background(), q.Canal.OrganizacionRef,
		q.Canal.ExpedienteRef, q.Documento, q.Canal.VersionObservada, "per_actual_sintetico_001", "clave-firma-preflight-001", 1)
	if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || a.material.Via != ports.ViaFirmaCertificadoVEC ||
		a.material.UnidadRef != "unidad:rrhh" || a.material.OrganizacionRef != q.Canal.OrganizacionRef {
		t.Fatalf("material nominal de consulta perdido: %+v %v", a.material, err)
	}
}
