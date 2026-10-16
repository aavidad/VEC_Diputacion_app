package bootstrap

import (
	"errors"
	"testing"

	firmaemisorv2 "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/firmaemisorv2"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/reglas"
)

// La composición de rutas debe devolver su fallo original aunque falte el
// cierre. Si lo hay, el error y el cierre general no duplican la limpieza.
func TestRutasFirmaVecFalloConservaErrorYCierraUnaVez(t *testing.T) {
	firma := &firmaDocumentoCTDesarrollo{
		servicio: &ctapp.ServicioFirmaDocumento{}, circuito: &reglas.Resolutor{},
		verificadorR5: dependenciasR5PresentesPrueba{},
	}
	original := &montajeOriginalFirmaR5Desarrollo{servicioOriginal: &vecapp.ServicioOriginalFirmableCT{}}
	for _, caso := range []struct {
		nombre       string
		conCierre    bool
		cierresFinal int
	}{
		{"sin_cierre", false, 0},
		{"con_cierre", true, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			m := &montajeFirmaVecR5CT{alta: &dependenciasAltaContratacionTemporalDesarrollo{}, recuperacion: &firmaemisorv2.Emisor{}}
			cierres := 0
			if caso.conCierre {
				m.cerrarAuditoria = func() { cierres++ }
			}
			rutas, err := m.rutas(firma, original)
			if rutas != nil || !errors.Is(err, ports.ErrRegistroFirmaDocumentoNoDisponible) {
				t.Fatalf("fallo de registro sustituido por el cierre: rutas=%v error=%v", rutas, err)
			}
			// El cierre de composición también se ejecuta tras el error de rutas.
			m.cerrarAuditoriaSiExiste()
			if cierres != caso.cierresFinal {
				t.Fatalf("limpieza repetida u omitida: %d", cierres)
			}
		})
	}
}
