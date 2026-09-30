package application

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

type registroInboxPrueba struct {
	mu                sync.Mutex
	evento            ports.EventoAvisoExterno
	recibo            ports.ReciboAvisoExterno
	reservado         bool
	estado            string
	sinDestino        bool
	confirmacionFalla bool
	token             string
}

func (r *registroInboxPrueba) AceptarAvisoExterno(_ context.Context, b []byte) (ports.ReciboAvisoExterno, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, err := canonico.LeerAvisoExterno(b)
	if err != nil {
		return ports.ReciboAvisoExterno{}, err
	}
	h, _ := canonico.HuellaAvisoExterno(e)
	if r.recibo.ReciboRef != "" {
		if r.recibo.Huella != h {
			return ports.ReciboAvisoExterno{}, ports.ErrAvisoExternoConflicto
		}
		out := r.recibo
		out.Replay = true
		return out, nil
	}
	r.evento = e
	r.recibo = ports.ReciboAvisoExterno{ReciboRef: "aviso_recibo:" + strings.Repeat("a", 32), Huella: h, AceptadoEn: "2026-09-30T12:00:00.000000Z"}
	return r.recibo, nil
}
func (r *registroInboxPrueba) ReservarAvisoExterno(_ context.Context, ref string) (ports.ReservaAvisoExterno, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ref != r.recibo.ReciboRef {
		return ports.ReservaAvisoExterno{}, ports.ErrAvisoExternoNoDisponible
	}
	if r.reservado {
		return ports.ReservaAvisoExterno{Recibo: r.recibo, Replay: true, Estado: r.estado}, nil
	}
	r.reservado = true
	r.estado = "reservado"
	r.token = "reserva:" + strings.Repeat("b", 32)
	out := ports.ReservaAvisoExterno{Recibo: r.recibo, Evento: r.evento, ReservaRef: r.token, Estado: r.estado}
	if !r.sinDestino {
		out.PersonaRef = "per_" + strings.Repeat("c", 24)
		out.Sobre = ports.SobreDireccionCorreo{CorreoRef: "correo:" + strings.Repeat("d", 32), Version: 1}
	}
	return out, nil
}
func (r *registroInboxPrueba) ConfirmarAvisoExterno(_ context.Context, ref, token, estado string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.confirmacionFalla {
		return ports.ErrAvisoExternoNoDisponible
	}
	if ref != r.recibo.ReciboRef || token != r.token {
		return ports.ErrAvisoExternoConflicto
	}
	r.estado = estado
	return nil
}

type protectorInboxPrueba struct{ falla bool }

func (*protectorInboxPrueba) CifrarDireccionCorreo(context.Context, string, string, uint64, []byte) (ports.SobreDireccionCorreo, error) {
	return ports.SobreDireccionCorreo{}, errors.New("no usado")
}
func (p *protectorInboxPrueba) ConDireccionCorreoDescifrada(_ context.Context, _ string, _ ports.SobreDireccionCorreo, usar func([]byte) error) error {
	if p.falla {
		return errors.New("fallo sobre")
	}
	return usar([]byte("sintetico@example.test"))
}

type transportadorInboxPrueba struct {
	mu        sync.Mutex
	n         int
	resultado ports.ResultadoTransporteAvisoExterno
}

func (t *transportadorInboxPrueba) EnviarAvisoExterno(_ context.Context, m ports.MensajeAvisoExterno) ports.ResultadoTransporteAvisoExterno {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.n++
	if m.Destino != "sintetico@example.test" {
		return ports.AvisoExternoNoAceptado
	}
	return t.resultado
}

type catalogoInboxPrueba bool

func (c catalogoInboxPrueba) AdmitePlantillaAvisoExterno(context.Context, string, string, string) bool {
	return bool(c)
}
func eventoInboxPrueba() ports.EventoAvisoExterno {
	return ports.EventoAvisoExterno{EventoRef: "evento:1", ProductorRef: "productor:bolsa", TipoVersionado: ports.TipoAvisoLlamamientoExternoV1, OcurridoEn: "2026-09-30T12:13:14.123456Z", CorrelacionRef: "corr:1", DestinatarioExternoRef: "can_" + strings.Repeat("a", 24), ComunicacionRef: "llamamiento:" + strings.Repeat("b", 64), PlantillaRef: "plantilla:aviso", PlantillaVersion: "1"}
}
func servicioInboxPrueba(t *testing.T, r *registroInboxPrueba, p *protectorInboxPrueba, tr *transportadorInboxPrueba) *ServicioAvisosExternos {
	t.Helper()
	s, err := NuevoServicioAvisosExternos(r, p, tr, catalogoInboxPrueba(true), "productor:bolsa")
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestInboxAceptacionNoEnviaYMismoEventoConservaRecibo(t *testing.T) {
	r := &registroInboxPrueba{}
	tr := &transportadorInboxPrueba{resultado: ports.AvisoExternoAceptadoPorRelay}
	s := servicioInboxPrueba(t, r, &protectorInboxPrueba{}, tr)
	ctx := context.Background()
	e := eventoInboxPrueba()
	primero, err := s.Aceptar(ctx, e)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.Aceptar(ctx, e)
	if err != nil || !replay.Replay || replay.ReciboRef != primero.ReciboRef || replay.Huella != primero.Huella || replay.AceptadoEn != primero.AceptadoEn || tr.n != 0 {
		t.Fatal("ACK no conserva el inbox", replay, err)
	}
	e.PlantillaVersion = "2"
	if _, err := s.Aceptar(ctx, e); !errors.Is(err, ports.ErrAvisoExternoConflicto) {
		t.Fatal("evento alterado", err)
	}
}
func TestInboxConcurrenteDespachaUnaVez(t *testing.T) {
	r := &registroInboxPrueba{}
	tr := &transportadorInboxPrueba{resultado: ports.AvisoExternoAceptadoPorRelay}
	s := servicioInboxPrueba(t, r, &protectorInboxPrueba{}, tr)
	ctx := context.Background()
	rec, err := s.Aceptar(ctx, eventoInboxPrueba())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.Despachar(ctx, rec.ReciboRef); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if tr.n != 1 || r.estado != "aceptado" {
		t.Fatal("duplicó SMTP", tr.n, r.estado)
	}
	s = servicioInboxPrueba(t, r, &protectorInboxPrueba{}, tr)
	out, err := s.Despachar(ctx, rec.ReciboRef)
	if err != nil || !out.Replay || tr.n != 1 {
		t.Fatal("reinicio duplica", out, err)
	}
}
func TestInboxFalloTrasSMTPNoReenvia(t *testing.T) {
	r := &registroInboxPrueba{confirmacionFalla: true}
	tr := &transportadorInboxPrueba{resultado: ports.AvisoExternoAceptadoPorRelay}
	s := servicioInboxPrueba(t, r, &protectorInboxPrueba{}, tr)
	ctx := context.Background()
	rec, _ := s.Aceptar(ctx, eventoInboxPrueba())
	if _, err := s.Despachar(ctx, rec.ReciboRef); err == nil {
		t.Fatal("ocultó fallo confirmación")
	}
	r.confirmacionFalla = false
	out, err := s.Despachar(ctx, rec.ReciboRef)
	if err != nil || !out.Replay || out.Estado != "reservado" || tr.n != 1 {
		t.Fatal("reintentó envío incierto", out, err, tr.n)
	}
}
func TestInboxSinCorreoOFalloSobreNoEnvia(t *testing.T) {
	for _, caso := range []struct {
		sinDestino, falla bool
		estado            string
	}{{true, false, "sin_destino"}, {false, true, "no_aceptado"}} {
		r := &registroInboxPrueba{sinDestino: caso.sinDestino}
		tr := &transportadorInboxPrueba{resultado: ports.AvisoExternoAceptadoPorRelay}
		s := servicioInboxPrueba(t, r, &protectorInboxPrueba{falla: caso.falla}, tr)
		ctx := context.Background()
		rec, _ := s.Aceptar(ctx, eventoInboxPrueba())
		out, err := s.Despachar(ctx, rec.ReciboRef)
		if tr.n != 0 || out.Estado != caso.estado || (caso.falla && err == nil) {
			t.Fatal(out, err, tr.n)
		}
	}
}
func TestInboxRechazaProductorYPlantillaAntesPersistir(t *testing.T) {
	r := &registroInboxPrueba{}
	tr := &transportadorInboxPrueba{}
	s := servicioInboxPrueba(t, r, &protectorInboxPrueba{}, tr)
	e := eventoInboxPrueba()
	e.ProductorRef = "productor:ajeno"
	if _, err := s.Aceptar(context.Background(), e); !errors.Is(err, ports.ErrAvisoExternoInvalido) {
		t.Fatal(err)
	}
	s.catalogo = catalogoInboxPrueba(false)
	if _, err := s.Aceptar(context.Background(), eventoInboxPrueba()); !errors.Is(err, ports.ErrAvisoExternoInvalido) || r.recibo.ReciboRef != "" {
		t.Fatal("aceptó plantilla ajena", err)
	}
	var typedNil *registroInboxPrueba
	if _, err := NuevoServicioAvisosExternos(typedNil, &protectorInboxPrueba{}, tr, catalogoInboxPrueba(true), "productor:bolsa"); err == nil {
		t.Fatal("dependencia nil admitida")
	}
}

func TestInboxResultadoTransporteConservadoSinReenvio(t *testing.T) {
	for _, caso := range []struct {
		nombre     string
		transporte ports.ResultadoTransporteAvisoExterno
		estado     string
	}{
		{"aceptado", ports.AvisoExternoAceptadoPorRelay, "aceptado"},
		{"no_aceptado", ports.AvisoExternoNoAceptado, "no_aceptado"},
		{"indeterminado_cero", ports.AvisoExternoIndeterminado, "reservado_incierto"},
		{"desconocido", ports.ResultadoTransporteAvisoExterno(99), "reservado_incierto"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			r := &registroInboxPrueba{}
			tr := &transportadorInboxPrueba{resultado: caso.transporte}
			s := servicioInboxPrueba(t, r, &protectorInboxPrueba{}, tr)
			ctx := context.Background()
			evento := eventoInboxPrueba()
			recibo, err := s.Aceptar(ctx, evento)
			if err != nil {
				t.Fatal(err)
			}
			primero, err := s.Despachar(ctx, recibo.ReciboRef)
			if err != nil || primero.Estado != caso.estado || primero.Replay || primero.ReciboRef != recibo.ReciboRef || tr.n != 1 {
				t.Fatal("resultado de transporte perdido", primero, err, tr.n)
			}
			token := r.token
			// Otra instancia recupera el mismo registro tras una interrupción.
			s = servicioInboxPrueba(t, r, &protectorInboxPrueba{}, tr)
			ack, err := s.Aceptar(ctx, evento)
			if err != nil || !ack.Replay || ack.ReciboRef != recibo.ReciboRef || ack.Huella != recibo.Huella || ack.AceptadoEn != recibo.AceptadoEn {
				t.Fatal("recibo original sustituido", ack, err)
			}
			replay, err := s.Despachar(ctx, ack.ReciboRef)
			if err != nil || !replay.Replay || replay.ReciboRef != primero.ReciboRef || replay.Estado != caso.estado || r.estado != caso.estado || r.token != token || !r.reservado || tr.n != 1 {
				t.Fatal("replay cambió resultado o repitió SMTP", replay, err, tr.n)
			}
		})
	}
}
