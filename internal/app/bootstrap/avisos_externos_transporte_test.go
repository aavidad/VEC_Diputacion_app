package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	smtpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/smtp"
	usuariosapp "vec-diputacion-granada/internal/modules/usuarios/application"
	usuarioscanonico "vec-diputacion-granada/internal/modules/usuarios/canonico"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
)

func TestTextosCorreoAvisosExternosUsanCatalogoComunAnidado(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "web", "static", "textos", idioma, "avisos-externos.json"))
		if err != nil {
			t.Fatal(err)
		}
		asunto, cuerpo, err := leerTextosCorreoAvisosExternos(raw)
		if err != nil || strings.TrimSpace(asunto) == "" || !strings.Contains(cuerpo, "{portal}") || strings.ContainsAny(asunto, "\r\n") {
			t.Fatalf("catálogo de aviso inválido en %s: %v", idioma, err)
		}
	}
	for _, raw := range [][]byte{
		[]byte(`{"usuarios.avisos_externos.asunto":"A","usuarios.avisos_externos.cuerpo":"{portal}"}`),
		[]byte(`{"usuarios":{"avisos_externos":"A"}}`),
		[]byte(`[]`),
	} {
		if _, _, err := leerTextosCorreoAvisosExternos(raw); err == nil {
			t.Fatal("catálogo ajeno admitido")
		}
	}
}

func TestTransporteAvisosExternosConservaEstadoSMTP(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		smtp    smtpct.Estado
		neutral usuariosports.ResultadoTransporteAvisoExterno
	}{
		{"aceptado", smtpct.AceptadoPorRelay, usuariosports.AvisoExternoAceptadoPorRelay},
		{"rechazo_transitorio", smtpct.NoAceptadoTransitorio, usuariosports.AvisoExternoNoAceptado},
		{"rechazo_permanente", smtpct.NoAceptadoPermanente, usuariosports.AvisoExternoNoAceptado},
		{"indeterminado", smtpct.Indeterminado, usuariosports.AvisoExternoIndeterminado},
		{"cero", smtpct.Estado(0), usuariosports.AvisoExternoIndeterminado},
		{"desconocido", smtpct.Estado(99), usuariosports.AvisoExternoIndeterminado},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			smtp := &enviadorCorreoLlamamientoDesarrolloPrueba{resultado: smtpct.Resultado{Estado: caso.smtp}}
			transporte := &transporteAvisosExternos{smtp: smtp, asunto: "asunto", cuerpo: "cuerpo", dominio: "example.test", ahora: func() time.Time { return time.Unix(0, 0) }}
			mensaje := usuariosports.MensajeAvisoExterno{EnvioRef: "aviso_recibo:" + strings.Repeat("a", 32), Destino: "sintetico@example.test"}
			if resultado := transporte.EnviarAvisoExterno(context.Background(), mensaje); resultado != caso.neutral || len(smtp.mensajes) != 1 {
				t.Fatal("perdió estado SMTP", resultado, len(smtp.mensajes))
			}
			if smtp.mensajes[0].MessageID != "<aviso_recibo-"+strings.Repeat("a", 32)+"@example.test>" {
				t.Fatal("sustituyó identificador de despacho")
			}
		})
	}
}

func TestTransporteAvisosExternosAusenteNoAfirmaRechazo(t *testing.T) {
	var transporte *transporteAvisosExternos
	if resultado := transporte.EnviarAvisoExterno(context.Background(), usuariosports.MensajeAvisoExterno{}); resultado != usuariosports.AvisoExternoIndeterminado {
		t.Fatal("transporte ausente afirmó un resultado", resultado)
	}
}

// El doble conserva el inbox entre instancias del servicio; no sustituye la
// prueba PostgreSQL de reserva/historia/auditoría.
type registroAvisosTransportePrueba struct {
	evento usuariosports.EventoAvisoExterno
	recibo usuariosports.ReciboAvisoExterno
	estado string
}

func (r *registroAvisosTransportePrueba) AceptarAvisoExterno(_ context.Context, material []byte) (usuariosports.ReciboAvisoExterno, error) {
	if r.recibo.ReciboRef != "" {
		recibo := r.recibo
		recibo.Replay = true
		return recibo, nil
	}
	if err := json.Unmarshal(material, &r.evento); err != nil {
		return usuariosports.ReciboAvisoExterno{}, err
	}
	h, err := usuarioscanonico.HuellaAvisoExterno(r.evento)
	if err != nil {
		return usuariosports.ReciboAvisoExterno{}, err
	}
	r.recibo = usuariosports.ReciboAvisoExterno{ReciboRef: "aviso_recibo:" + strings.Repeat("a", 32), Huella: h, AceptadoEn: "2026-09-30T12:00:00.123456Z"}
	return r.recibo, nil
}
func (r *registroAvisosTransportePrueba) ReservarAvisoExterno(_ context.Context, ref string) (usuariosports.ReservaAvisoExterno, error) {
	if ref != r.recibo.ReciboRef {
		return usuariosports.ReservaAvisoExterno{}, usuariosports.ErrAvisoExternoConflicto
	}
	if r.estado != "" {
		return usuariosports.ReservaAvisoExterno{Recibo: r.recibo, Estado: r.estado, Replay: true}, nil
	}
	r.estado = "reservado"
	return usuariosports.ReservaAvisoExterno{Recibo: r.recibo, Evento: r.evento, ReservaRef: "reserva:" + strings.Repeat("b", 32), Estado: r.estado, PersonaRef: "per_" + strings.Repeat("c", 24), Sobre: usuariosports.SobreDireccionCorreo{CorreoRef: "correo:" + strings.Repeat("d", 32)}}, nil
}
func (r *registroAvisosTransportePrueba) ConfirmarAvisoExterno(_ context.Context, ref, token, estado string) error {
	if ref != r.recibo.ReciboRef || token != "reserva:"+strings.Repeat("b", 32) {
		return usuariosports.ErrAvisoExternoConflicto
	}
	r.estado = estado
	return nil
}

type dependenciasAvisosTransportePrueba struct{}

func (dependenciasAvisosTransportePrueba) CifrarDireccionCorreo(context.Context, string, string, uint64, []byte) (usuariosports.SobreDireccionCorreo, error) {
	return usuariosports.SobreDireccionCorreo{}, usuariosports.ErrAvisoExternoNoDisponible
}
func (dependenciasAvisosTransportePrueba) ConDireccionCorreoDescifrada(_ context.Context, _ string, _ usuariosports.SobreDireccionCorreo, usar func([]byte) error) error {
	return usar([]byte("sintetico@example.test"))
}
func (dependenciasAvisosTransportePrueba) AdmitePlantillaAvisoExterno(context.Context, string, string, string) bool {
	return true
}

func TestAvisoExternoSMTPIndeterminadoReplayConservaReciboSinEnviar(t *testing.T) {
	smtp := &enviadorCorreoLlamamientoDesarrolloPrueba{resultado: smtpct.Resultado{Estado: smtpct.Indeterminado}}
	transporte := &transporteAvisosExternos{smtp: smtp, asunto: "asunto", cuerpo: "cuerpo", dominio: "example.test", ahora: func() time.Time { return time.Unix(0, 0) }}
	registro := &registroAvisosTransportePrueba{}
	nuevo := func() *usuariosapp.ServicioAvisosExternos {
		t.Helper()
		s, err := usuariosapp.NuevoServicioAvisosExternos(registro, dependenciasAvisosTransportePrueba{}, transporte, dependenciasAvisosTransportePrueba{}, "productor:bolsa")
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	s := nuevo()
	ctx := context.Background()
	evento := usuariosports.EventoAvisoExterno{EventoRef: "evento:1", ProductorRef: "productor:bolsa", TipoVersionado: usuariosports.TipoAvisoLlamamientoExternoV1, OcurridoEn: "2026-09-30T12:13:14.123456Z", CorrelacionRef: "corr:1", DestinatarioExternoRef: "can_" + strings.Repeat("a", 24), ComunicacionRef: "llamamiento:" + strings.Repeat("b", 64), PlantillaRef: "plantilla:aviso", PlantillaVersion: "1"}
	recibo, err := s.Aceptar(ctx, evento)
	if err != nil {
		t.Fatal(err)
	}
	primero, err := s.Despachar(ctx, recibo.ReciboRef)
	if err != nil || primero.Estado != "reservado_incierto" || primero.Replay || primero.ReciboRef != recibo.ReciboRef || len(smtp.mensajes) != 1 {
		t.Fatal(primero, err, len(smtp.mensajes))
	}
	s = nuevo()
	ack, err := s.Aceptar(ctx, evento)
	if err != nil || !ack.Replay || ack.ReciboRef != recibo.ReciboRef || ack.Huella != recibo.Huella || ack.AceptadoEn != recibo.AceptadoEn {
		t.Fatal(ack, err)
	}
	replay, err := s.Despachar(ctx, ack.ReciboRef)
	if err != nil || !replay.Replay || replay.Estado != "reservado_incierto" || replay.ReciboRef != primero.ReciboRef || registro.estado != "reservado_incierto" || len(smtp.mensajes) != 1 {
		t.Fatal("replay repitió SMTP o sustituyó resultado", replay, err, len(smtp.mensajes))
	}
}

type repositorioAvisosOrdenPrueba struct {
	evento     bolsaports.AvisoExternoPendiente
	falloACK   bool
	ack        bool
	resultados []string
}

func (r *repositorioAvisosOrdenPrueba) Extraer(context.Context, int) ([]bolsaports.AvisoExternoPendiente, error) {
	return []bolsaports.AvisoExternoPendiente{r.evento}, nil
}
func (r *repositorioAvisosOrdenPrueba) ConfirmarAceptacion(context.Context, string, string, string, string) error {
	if r.falloACK {
		return errAvisosExternos
	}
	r.ack = true
	return nil
}
func (r *repositorioAvisosOrdenPrueba) RegistrarResultadoDespacho(_ context.Context, _, _, _, _, estado string) error {
	if !r.ack {
		return errAvisosExternos
	}
	r.resultados = append(r.resultados, estado)
	return nil
}

func TestAvisoExternoAcusePrecedeReservaYResultado(t *testing.T) {
	smtp := &enviadorCorreoLlamamientoDesarrolloPrueba{resultado: smtpct.Resultado{Estado: smtpct.AceptadoPorRelay}}
	transporte := &transporteAvisosExternos{smtp: smtp, asunto: "asunto", cuerpo: "cuerpo", dominio: "example.test", ahora: func() time.Time { return time.Unix(0, 0) }}
	registro := &registroAvisosTransportePrueba{}
	servicio, err := usuariosapp.NuevoServicioAvisosExternos(registro, dependenciasAvisosTransportePrueba{}, transporte, dependenciasAvisosTransportePrueba{}, "productor:bolsa")
	if err != nil {
		t.Fatal(err)
	}
	e := usuariosports.EventoAvisoExterno{EventoRef: "evento:1", ProductorRef: "productor:bolsa", TipoVersionado: usuariosports.TipoAvisoLlamamientoExternoV1, OcurridoEn: "2026-09-30T12:13:14.123456Z", CorrelacionRef: "corr:1", DestinatarioExternoRef: "can_" + strings.Repeat("a", 24), ComunicacionRef: "llamamiento:" + strings.Repeat("b", 64), PlantillaRef: "plantilla:aviso", PlantillaVersion: "1"}
	h, err := usuarioscanonico.HuellaAvisoExterno(e)
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioAvisosOrdenPrueba{evento: bolsaports.AvisoExternoPendiente{Evento: bolsaports.EventoAvisoExterno{EventoRef: e.EventoRef, ProductorRef: e.ProductorRef, TipoVersionado: e.TipoVersionado, OcurridoEn: e.OcurridoEn, CorrelacionRef: e.CorrelacionRef, DestinatarioExternoRef: e.DestinatarioExternoRef, ComunicacionRef: e.ComunicacionRef, PlantillaRef: e.PlantillaRef, PlantillaVersion: e.PlantillaVersion}, Huella: h}, falloACK: true}
	cfg := configuracionAvisosExternos{Lote: 1}
	if err := procesarLoteAvisosExternos(context.Background(), repo, servicio, cfg); err == nil || len(smtp.mensajes) != 0 || len(repo.resultados) != 0 || registro.estado != "" {
		t.Fatal("ACK fallido permitió reserva o SMTP", err, len(smtp.mensajes), repo.resultados, registro.estado)
	}
	repo.falloACK = false
	if err := procesarLoteAvisosExternos(context.Background(), repo, servicio, cfg); err != nil || !repo.ack || len(smtp.mensajes) != 1 || len(repo.resultados) != 1 || repo.resultados[0] != "aceptado" {
		t.Fatal("recuperación tras ACK", err, repo.ack, len(smtp.mensajes), repo.resultados)
	}
}
