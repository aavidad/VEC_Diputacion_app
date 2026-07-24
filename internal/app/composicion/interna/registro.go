package interna

import (
	"io"
	"sync"
	"time"
)

const (
	mensajeEventoHTTPSaneado   = "servidor interno: evento HTTP/TLS rechazado\n"
	intervaloEventoHTTPSaneado = 30 * time.Second
)

// escritorEventosHTTPSaneados descarta el texto de net/http, que puede incluir
// direcciones y errores TLS, y conserva una evidencia fija con limite temporal.
type escritorEventosHTTPSaneados struct {
	mu      sync.Mutex
	destino io.Writer
	ahora   func() time.Time
	ultimo  time.Time
}

func nuevoEscritorEventosHTTPSaneados(destino io.Writer) *escritorEventosHTTPSaneados {
	return &escritorEventosHTTPSaneados{destino: destino, ahora: time.Now}
}

func (e *escritorEventosHTTPSaneados) Write(entrada []byte) (int, error) {
	longitud := len(entrada)
	if e == nil || e.destino == nil || e.ahora == nil {
		return longitud, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	ahora := e.ahora()
	if !e.ultimo.IsZero() && ahora.Sub(e.ultimo) < intervaloEventoHTTPSaneado {
		return longitud, nil
	}
	e.ultimo = ahora
	_, _ = io.WriteString(e.destino, mensajeEventoHTTPSaneado)
	return longitud, nil
}
