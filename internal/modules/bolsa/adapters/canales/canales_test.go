package canales

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type proveedorSMSPrueba struct {
	err      error
	enviados []string
}

func (p *proveedorSMSPrueba) EnviarSMS(_ context.Context, movil, texto string) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	p.enviados = append(p.enviados, movil+"|"+texto)
	return "sms-1", nil
}

type movilesPrueba map[string]string

func (m movilesPrueba) MovilParticipacion(_ context.Context, p string) (string, error) {
	return m[p], nil
}

type telegramPrueba struct{ enviados int }

func (t *telegramPrueba) EnviarMensaje(context.Context, string, string) (string, error) {
	t.enviados++
	return "tg-1", nil
}

type vinculosPrueba map[string]bool

func (v vinculosPrueba) ChatVinculado(_ context.Context, p string) (string, bool, error) {
	return "chat-" + p, v[p], nil
}

func mensajePrueba(texto string) puertosbolsa.MensajeAvisoLlamamiento {
	return puertosbolsa.MensajeAvisoLlamamiento{BolsaRef: "bolsa:1", LlamamientoRef: "llamamiento:1", ParticipacionRef: "participacion:1", Texto: texto, EnlacePortal: "https://vec.example/portal", Instante: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)}
}

func TestSMSEnviaAvisoMinimoYNuncaDaPorEnviadoUnFallo(t *testing.T) {
	ctx := context.Background()
	if _, err := NuevoSMS(nil, movilesPrueba{}, 160); err == nil {
		t.Fatal("SMS sin proveedor real compuesto")
	}
	proveedor := &proveedorSMSPrueba{}
	sms, err := NuevoSMS(proveedor, movilesPrueba{"participacion:1": "+34600000000"}, 160)
	if err != nil || !sms.Disponible(ctx) || sms.Canal() != dominiobolsa.CanalAvisoSMS {
		t.Fatalf("SMS: %v", err)
	}
	r, err := sms.EnviarAviso(ctx, mensajePrueba("Tiene un llamamiento de Bolsa. Consúltelo en el portal:"))
	if err != nil || r.Resultado != dominiobolsa.ResultadoContactoEnviado || r.ReferenciaProveedor != "sms-1" || len(proveedor.enviados) != 1 {
		t.Fatalf("envío: %+v %v", r, err)
	}
	for _, texto := range []string{"", " con espacios ", "Escriba a rrhh@example.org", "Su DNI 12345678Z", strings.Repeat("a", 200)} {
		if _, err := sms.EnviarAviso(ctx, mensajePrueba(texto)); !errors.Is(err, puertosbolsa.ErrMensajeAvisoInvalido) {
			t.Errorf("texto %q admitido", texto)
		}
	}
	sinMovil := mensajePrueba("Aviso")
	sinMovil.ParticipacionRef = "participacion:2"
	if r, err = sms.EnviarAviso(ctx, sinMovil); err != nil || r.Resultado != dominiobolsa.ResultadoContactoNoEnviado {
		t.Fatalf("sin móvil: %+v %v", r, err)
	}
	proveedor.err = errors.New("proveedor caído")
	if r, err = sms.EnviarAviso(ctx, mensajePrueba("Aviso")); err != nil || r.Resultado != dominiobolsa.ResultadoContactoNoEnviado {
		t.Fatalf("caída como envío: %+v %v", r, err)
	}
}

func TestTelegramSoloConAltaVoluntariaVigente(t *testing.T) {
	ctx := context.Background()
	if _, err := NuevoTelegram(&telegramPrueba{}, nil, 1000); err == nil {
		t.Fatal("Telegram sin vinculaciones compuesto")
	}
	bot := &telegramPrueba{}
	tg, err := NuevoTelegram(bot, vinculosPrueba{"participacion:1": true}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if r, err := tg.EnviarAviso(ctx, mensajePrueba("Aviso")); err != nil || r.Resultado != dominiobolsa.ResultadoContactoEnviado || bot.enviados != 1 {
		t.Fatalf("con alta: %+v %v", r, err)
	}
	sinAlta := mensajePrueba("Aviso")
	sinAlta.ParticipacionRef = "participacion:2"
	if r, err := tg.EnviarAviso(ctx, sinAlta); err != nil || r.Resultado != dominiobolsa.ResultadoContactoNoEnviado || bot.enviados != 1 {
		t.Fatalf("sin alta: %+v %v", r, err)
	}
}

func TestTelefonoRecuerdaLaInstalacionYNoSeEnciendeSinElla(t *testing.T) {
	ctx := context.Background()
	llamadas, instalado := 0, false
	tel, err := NuevoTelefono(func(context.Context) (bool, error) { llamadas++; return instalado, nil })
	if err != nil || tel.Disponible(ctx) {
		t.Fatalf("sin B87 disponible: %v", err)
	}
	instalado = true
	if !tel.Disponible(ctx) || !tel.Disponible(ctx) || llamadas != 2 {
		t.Fatalf("llamadas=%d", llamadas)
	}
	if NuevoCorreo(false).Disponible(ctx) || !NuevoCorreo(true).Disponible(ctx) {
		t.Fatal("correo")
	}
}
