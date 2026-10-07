package telemetria

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"time"
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
	return claseError(err)
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
