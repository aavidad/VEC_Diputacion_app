package interna

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestErrorLogHTTPEmiteSoloEvidenciaFijaConRateLimit(t *testing.T) {
	const marcador = "MARCADOR_CRUDO_10.7.15.40:8443_handshake"
	var salida bytes.Buffer
	instante := time.Unix(100, 0)
	escritor := nuevoEscritorEventosHTTPSaneados(&salida)
	escritor.ahora = func() time.Time { return instante }
	registro := log.New(escritor, "", 0)
	registro.Printf("error desde %s: %s", marcador, errors.New(marcador))
	registro.Printf("repetido %s", marcador)
	if texto := salida.String(); texto != mensajeEventoHTTPSaneado || strings.Contains(texto, marcador) {
		t.Fatalf("evidencia inicial = %q", texto)
	}
	instante = instante.Add(intervaloEventoHTTPSaneado)
	registro.Printf("nuevo intervalo %s", marcador)
	if texto := salida.String(); texto != mensajeEventoHTTPSaneado+mensajeEventoHTTPSaneado || strings.Contains(texto, marcador) {
		t.Fatalf("evidencia tras intervalo = %q", texto)
	}
}

func TestErrorLogHTTPSaneadoEsSeguroEnConcurrencia(t *testing.T) {
	var salida bytes.Buffer
	escritor := nuevoEscritorEventosHTTPSaneados(&salida)
	escritor.ahora = func() time.Time { return time.Unix(100, 0) }
	var grupo sync.WaitGroup
	for range 32 {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			_, _ = escritor.Write([]byte("MARCADOR_CRUDO"))
		}()
	}
	grupo.Wait()
	if salida.String() != mensajeEventoHTTPSaneado {
		t.Fatalf("evidencia concurrente = %q", salida.String())
	}
}
