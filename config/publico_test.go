package config

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/shared/limiteshttp"
)

func TestNormalizePublicTransportAcotaTiempoEscritura(t *testing.T) {
	for _, prueba := range []struct {
		nombre   string
		entrada  time.Duration
		esperada time.Duration
	}{
		{nombre: "ausente", entrada: 0, esperada: limiteshttp.DuracionMaximaPeticionPublica},
		{nombre: "excesivo", entrada: 60 * time.Second, esperada: limiteshttp.DuracionMaximaPeticionPublica},
		{nombre: "menor", entrada: 4 * time.Second, esperada: limiteshttp.DuracionMaximaPeticionPublica},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			obtenida := (Config{WriteTimeout: prueba.entrada}).NormalizePublicTransport().WriteTimeout
			if obtenida != prueba.esperada {
				t.Fatalf("WriteTimeout = %s; esperado %s", obtenida, prueba.esperada)
			}
		})
	}
}
