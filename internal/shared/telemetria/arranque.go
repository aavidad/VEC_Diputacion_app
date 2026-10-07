package telemetria

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// EventoArranque describe un hito técnico del proceso. Preparada significa
// que la composición terminó; no afirma que la escucha ya esté disponible.
type EventoArranque struct {
	Servicio   string
	Superficie string
	Entorno    string
	Fase       string // configuracion, composicion o escucha
	Resultado  string // preparada o fallida
	Causa      string // clase cerrada; nunca el texto del error
	Duracion   time.Duration
}

// RegistrarArranque escribe un único hito JSON en el destino técnico. Solo
// acepta valores de cardinalidad cerrada para impedir que lleguen al registro
// rutas, DSN o mensajes de PostgreSQL por una llamada futura.
func RegistrarArranque(destino io.Writer, e EventoArranque) {
	if destino == nil {
		return
	}
	servicio := valorCerrado(e.Servicio, "vec-server", "vec-admin")
	superficie := valorCerrado(e.Superficie, "interno", "externo", "integrada", "administracion")
	entorno := Entorno(e.Entorno, "")
	fase := valorCerrado(e.Fase, "configuracion", "composicion", "escucha")
	resultado := valorCerrado(e.Resultado, "preparada", "fallida")
	if resultado != "preparada" {
		resultado = "fallida"
	}
	duracion := e.Duracion
	if duracion < 0 {
		duracion = 0
	}
	atributos := []slog.Attr{
		slog.String("service.name", servicio),
		slog.String("service.version", Version()),
		slog.String("deployment.environment.name", entorno),
		slog.String("vec.superficie", superficie),
		slog.String("vec.arranque.fase", fase),
		slog.String("vec.arranque.resultado", resultado),
		slog.Float64("vec.arranque.duracion", segundos(duracion)),
	}
	nivel := slog.LevelInfo
	if resultado == "fallida" {
		nivel = slog.LevelError
		atributos = append(atributos, slog.String("error.type", causaArranque(e.Causa)))
	}
	slog.New(slog.NewJSONHandler(destino, nil)).LogAttrs(context.Background(), nivel, "vec.process.startup", atributos...)
}

// ClaseErrorArranque reduce un error interno a una clase sin su mensaje.
// Comparte las clases cerradas del trazador PostgreSQL.
func ClaseErrorArranque(err error) string {
	if err == nil {
		return "otro"
	}
	clase := claseError(err)
	if clase != "otro" {
		return clase
	}
	var clasificado interface{ ClaseErrorArranque() string }
	if errors.As(err, &clasificado) {
		return causaArranque(clasificado.ClaseErrorArranque())
	}
	return "otro"
}

// MensajeErrorArranque conserva las causas técnicas que se pueden construir
// sin datos de la petición. Un Error() libre no tiene procedencia fiable: puede
// contener nombres, rutas o secretos, y solo se muestra si coincide con un
// diagnóstico interno conocido. Los errores de sistema se descomponen sin
// imprimir los campos que contienen datos variables.
func MensajeErrorArranque(err error) string {
	return limitarMensajeArranque(mensajeErrorArranque(err, 0))
}

const maximoMensajeArranque = 240

func mensajeErrorArranque(err error, profundidad int) string {
	if err == nil {
		return "causa no disponible"
	}
	if profundidad >= 8 {
		return "cadena de errores demasiado profunda"
	}
	switch e := err.(type) {
	case *pgconn.PgError:
		if e == nil {
			return "PostgreSQL bd_codigo_no_disponible"
		}
		codigo := causaArranque("bd_" + e.Code)
		if codigo == "otro" {
			codigo = "bd_codigo_no_disponible"
		}
		mensaje := "PostgreSQL " + codigo
		if e.Routine != "" {
			mensaje += " rutina=presente"
		}
		if e.ConstraintName != "" {
			mensaje += " restriccion=presente"
		}
		return mensaje
	case *os.PathError:
		if e == nil {
			return "archivo: causa no disponible"
		}
		return "archivo " + operacionArranque(e.Op) + ": " + mensajeErrorArranque(e.Err, profundidad+1)
	case *url.Error:
		if e == nil {
			return "URL: causa no disponible"
		}
		return "URL " + operacionArranque(e.Op) + ": " + mensajeErrorArranque(e.Err, profundidad+1)
	case *net.OpError:
		if e == nil {
			return "red: causa no disponible"
		}
		return "red " + operacionArranque(e.Op) + ": " + mensajeErrorArranque(e.Err, profundidad+1)
	case *net.DNSError:
		if e == nil {
			return "DNS: causa no disponible"
		}
		if e.IsTimeout {
			return "DNS: plazo vencido"
		}
		if e.IsNotFound {
			return "DNS: nombre no encontrado"
		}
		return "DNS: consulta fallida"
	case *os.SyscallError:
		if e == nil {
			return "sistema: causa no disponible"
		}
		return "sistema " + operacionArranque(e.Syscall) + ": " + mensajeErrorArranque(e.Err, profundidad+1)
	case syscall.Errno:
		return fmt.Sprintf("sistema errno=%d (%s)", e, e.Error())
	case *time.ParseError:
		return "formato de fecha no valido"
	}
	if errors.Is(err, context.Canceled) {
		return "operacion cancelada"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "plazo vencido"
	}
	if permitido := mensajeInternoArranque(err.Error()); permitido != "" {
		return permitido
	}
	if varios, ok := err.(interface{ Unwrap() []error }); ok {
		partes := make([]string, 0, 3)
		for _, causa := range varios.Unwrap() {
			if len(partes) == 3 {
				break
			}
			partes = append(partes, mensajeErrorArranque(causa, profundidad+1))
		}
		if len(partes) > 0 {
			return strings.Join(partes, "; ")
		}
	}
	if causa := errors.Unwrap(err); causa != nil {
		mensaje := mensajeErrorArranque(causa, profundidad+1)
		if prefijo := prefijoArranqueConocido(err.Error(), causa.Error()); prefijo != "" {
			return prefijo + " " + mensaje
		}
		return mensaje
	}
	return "error interno no catalogado tipo=" + tipoErrorArranque(err)
}

func prefijoArranqueConocido(texto, causa string) string {
	if !strings.HasSuffix(texto, causa) {
		return ""
	}
	prefijo := strings.TrimSpace(strings.TrimSuffix(texto, causa))
	switch prefijo {
	case "bootstrap:", "bootstrap server:", "composicion:", "serve:",
		"vec-admin:", "auditoria-intentos:", "auditoria_intentos:":
		return prefijo
	}
	return ""
}

func mensajeInternoArranque(valor string) string {
	switch valor {
	case "bootstrap: material criptografico de desarrollo invalido",
		"bootstrap: fuente configurada no disponible",
		"bootstrap: material criptografico de desarrollo invalido: ruta enlazada o no canonica",
		"bootstrap: material criptografico de desarrollo invalido: el directorio pertenece al repositorio",
		"bootstrap: material criptografico de desarrollo invalido: TLS no corresponde al material del perfil",
		"auditoria.intentos.configuracion_no_disponible",
		"administracion: configuracion no valida":
		return valor
	}
	return ""
}

func tipoErrorArranque(err error) string {
	valor := reflect.TypeOf(err).String()
	if len(valor) == 0 || len(valor) > 80 {
		return "desconocido"
	}
	for _, c := range valor {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			c == '_' || c == '*' || c == '.' || c == '/' || c == '[' || c == ']' || c == '-' {
			continue
		}
		return "desconocido"
	}
	return valor
}

func operacionArranque(valor string) string {
	switch strings.ToLower(valor) {
	case "open", "read", "write", "stat", "lstat", "remove", "dial", "listen", "accept", "get", "post", "put", "head", "connect", "bind", "send", "recv":
		return strings.ToLower(valor)
	}
	return "operacion"
}

func limitarMensajeArranque(valor string) string {
	if len(valor) > maximoMensajeArranque {
		valor = valor[:maximoMensajeArranque]
	}
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, valor)
}

func causaArranque(causa string) string {
	switch causa {
	case "configuracion", "cancelada", "plazo_vencido", "red", "otro":
		return causa
	}
	if strings.HasPrefix(causa, "bd_") && len(causa) == 8 && strings.Trim(causa[3:], "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ") == "" {
		return causa
	}
	return "otro"
}

func valorCerrado(valor string, admitidos ...string) string {
	for _, admitido := range admitidos {
		if valor == admitido {
			return valor
		}
	}
	return "desconocido"
}
