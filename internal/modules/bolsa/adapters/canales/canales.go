// Package canales reúne los adaptadores de los canales de aviso de un
// llamamiento de Bolsa. Correo y teléfono están compuestos en la aplicación;
// SMS y Telegram quedan preparados (contrato y pruebas) y solo se componen
// cuando exista su proveedor corporativo real y el visto bueno del DPD.
package canales

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Correo es el canal que ya avisa al emitir. Está disponible cuando la
// emisión por correo está compuesta.
type Correo struct{ compuesto bool }

func NuevoCorreo(emisionCompuesta bool) *Correo { return &Correo{compuesto: emisionCompuesta} }

func (c *Correo) Canal() string                   { return dominiobolsa.CanalAvisoCorreo }
func (c *Correo) Disponible(context.Context) bool { return c != nil && c.compuesto }

// Telefono es el seguimiento manual: una persona de RRHH llama y anota el
// resultado con la hora del servidor. Está disponible cuando el registro de
// llamadas (Bolsa 000087) está instalado y el ejecutor puede usarlo. Una
// instalación no se deshace: el primer «sí» se recuerda; un fallo de lectura
// apaga el canal sin error para el resto de la pantalla.
type Telefono struct {
	comprobar func(context.Context) (bool, error)
	instalado atomic.Bool
}

func NuevoTelefono(comprobar func(context.Context) (bool, error)) (*Telefono, error) {
	if comprobar == nil {
		return nil, puertosbolsa.ErrCanalAvisoNoDisponible
	}
	return &Telefono{comprobar: comprobar}, nil
}

func (t *Telefono) Canal() string { return dominiobolsa.CanalAvisoTelefono }

func (t *Telefono) Disponible(ctx context.Context) bool {
	if t == nil || ctx == nil {
		return false
	}
	if t.instalado.Load() {
		return true
	}
	ok, err := t.comprobar(ctx)
	if err != nil {
		// Se apaga el canal en esta lectura y queda constancia de la causa.
		slog.Warn("canal teléfono del llamamiento no comprobado", "causa", err)
		return false
	}
	if !ok {
		return false
	}
	t.instalado.Store(true)
	return true
}

var (
	patronMovil = regexp.MustCompile(`^\+?[0-9]{9,15}$`)
	// Un texto de aviso no lleva correos ni números largos (teléfonos o
	// documentos): solo el aviso del catálogo y el enlace al portal.
	patronDatoEnTexto = regexp.MustCompile(`@|[0-9]{7,}`)
)

// validarMensaje aplica el contrato común de los mensajes cortos.
func validarMensaje(m puertosbolsa.MensajeAvisoLlamamiento, limite int) error {
	texto := strings.TrimSpace(m.Texto)
	if m.BolsaRef == "" || m.LlamamientoRef == "" || m.ParticipacionRef == "" || m.Instante.IsZero() ||
		texto == "" || texto != m.Texto || !strings.HasPrefix(m.EnlacePortal, "https://") || strings.ContainsAny(m.EnlacePortal, " \t\r\n") ||
		patronDatoEnTexto.MatchString(m.Texto) || utf8.RuneCountInString(m.Texto)+1+utf8.RuneCountInString(m.EnlacePortal) > limite {
		return puertosbolsa.ErrMensajeAvisoInvalido
	}
	return nil
}

func noEnviado() puertosbolsa.ResultadoAvisoCanal {
	return puertosbolsa.ResultadoAvisoCanal{Resultado: dominiobolsa.ResultadoContactoNoEnviado}
}

// SMS envía un aviso corto al móvil de la persona por el proveedor
// corporativo. Sin móvil o con el proveedor caído el aviso queda «no
// enviado»: la indisponibilidad nunca se registra como envío.
type SMS struct {
	proveedor puertosbolsa.ProveedorSMS
	moviles   puertosbolsa.FuenteMovilParticipacion
	limite    int
}

func NuevoSMS(proveedor puertosbolsa.ProveedorSMS, moviles puertosbolsa.FuenteMovilParticipacion, limiteCaracteres int) (*SMS, error) {
	if proveedor == nil || moviles == nil || limiteCaracteres < 1 || limiteCaracteres > 4096 {
		return nil, puertosbolsa.ErrCanalAvisoNoDisponible
	}
	return &SMS{proveedor: proveedor, moviles: moviles, limite: limiteCaracteres}, nil
}

func (s *SMS) Canal() string                   { return dominiobolsa.CanalAvisoSMS }
func (s *SMS) Disponible(context.Context) bool { return s != nil && s.proveedor != nil }

func (s *SMS) EnviarAviso(ctx context.Context, m puertosbolsa.MensajeAvisoLlamamiento) (puertosbolsa.ResultadoAvisoCanal, error) {
	if s == nil || ctx == nil {
		return puertosbolsa.ResultadoAvisoCanal{}, puertosbolsa.ErrCanalAvisoNoDisponible
	}
	if err := validarMensaje(m, s.limite); err != nil {
		return puertosbolsa.ResultadoAvisoCanal{}, err
	}
	movil, err := s.moviles.MovilParticipacion(ctx, m.ParticipacionRef)
	if err != nil || !patronMovil.MatchString(movil) {
		return noEnviado(), nil
	}
	referencia, err := s.proveedor.EnviarSMS(ctx, movil, m.Texto+" "+m.EnlacePortal)
	if err != nil || strings.TrimSpace(referencia) == "" {
		return noEnviado(), nil
	}
	return puertosbolsa.ResultadoAvisoCanal{Resultado: dominiobolsa.ResultadoContactoEnviado, ReferenciaProveedor: referencia}, nil
}

// Telegram avisa solo a quien se dio de alta voluntariamente desde su área
// personal (vinculación con código de un solo uso, revocable). La vinculación,
// el consentimiento y el bot son piezas pendientes (duda al DPD): hasta
// entonces este adaptador no se compone.
type Telegram struct {
	proveedor     puertosbolsa.ProveedorTelegram
	vinculaciones puertosbolsa.VinculacionesTelegram
	limite        int
}

func NuevoTelegram(proveedor puertosbolsa.ProveedorTelegram, vinculaciones puertosbolsa.VinculacionesTelegram, limiteCaracteres int) (*Telegram, error) {
	if proveedor == nil || vinculaciones == nil || limiteCaracteres < 1 || limiteCaracteres > 4096 {
		return nil, puertosbolsa.ErrCanalAvisoNoDisponible
	}
	return &Telegram{proveedor: proveedor, vinculaciones: vinculaciones, limite: limiteCaracteres}, nil
}

func (t *Telegram) Canal() string                   { return dominiobolsa.CanalAvisoTelegram }
func (t *Telegram) Disponible(context.Context) bool { return t != nil && t.proveedor != nil }

func (t *Telegram) EnviarAviso(ctx context.Context, m puertosbolsa.MensajeAvisoLlamamiento) (puertosbolsa.ResultadoAvisoCanal, error) {
	if t == nil || ctx == nil {
		return puertosbolsa.ResultadoAvisoCanal{}, puertosbolsa.ErrCanalAvisoNoDisponible
	}
	if err := validarMensaje(m, t.limite); err != nil {
		return puertosbolsa.ResultadoAvisoCanal{}, err
	}
	chat, vigente, err := t.vinculaciones.ChatVinculado(ctx, m.ParticipacionRef)
	if err != nil || !vigente || strings.TrimSpace(chat) == "" {
		return noEnviado(), nil
	}
	referencia, err := t.proveedor.EnviarMensaje(ctx, chat, m.Texto+"\n"+m.EnlacePortal)
	if err != nil || strings.TrimSpace(referencia) == "" {
		return noEnviado(), nil
	}
	return puertosbolsa.ResultadoAvisoCanal{Resultado: dominiobolsa.ResultadoContactoEnviado, ReferenciaProveedor: referencia}, nil
}

var (
	_ puertosbolsa.CanalAvisoLlamamiento    = (*Correo)(nil)
	_ puertosbolsa.CanalAvisoLlamamiento    = (*Telefono)(nil)
	_ puertosbolsa.EnviadorAvisoLlamamiento = (*SMS)(nil)
	_ puertosbolsa.EnviadorAvisoLlamamiento = (*Telegram)(nil)
)
