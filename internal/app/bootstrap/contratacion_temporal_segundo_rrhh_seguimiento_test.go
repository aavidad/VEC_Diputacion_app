package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func TestSeleccionSoporteSeguimientoRRHHUsaHojaVerificadaYSujeto(t *testing.T) {
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	original := identidadRRHHOriginalPrueba()
	segundo := original
	segundo.huella = sha256.Sum256([]byte("hoja-segunda"))
	segundo.principal = clonarPrincipalDesarrollo(original.principal)
	segundo.principal.ID = "desarrollo:segunda-rrhh"
	segundo.principal.Attributes["certificate_sha256"] = hex.EncodeToString(segundo.huella[:])
	base := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: original.principal.ID, certificadoSHA256: original.principal.Attributes["certificate_sha256"]}
	adicional := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: segundo.principal.ID, certificadoSHA256: segundo.principal.Attributes["certificate_sha256"]}
	selector := seleccionSoporteSeguimientoRRHH{original: base, adicional: adicional}
	capacidad := func(i identidadCertificadoDesarrollo, ruta string) context.Context {
		return context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{},
			capacidadConsultaContratacionTemporalDesarrollo{sello: sello, ruta: ruta, metodo: http.MethodPost, principal: i.principal})
	}
	for _, caso := range []struct {
		nombre   string
		ctx      context.Context
		esperado *soporteAltaContratacionTemporalDesarrollo
	}{
		{"original", capacidad(original, httpinterno.RutaNoIncorporaciones), base},
		{"adicional", capacidad(segundo, httpinterno.RutaNoIncorporaciones), adicional},
		{"adicional en cese", capacidad(segundo, httpinterno.RutaCesesNombramiento), nil},
		{"adicional en GINPIX", capacidad(segundo, httpinterno.RutaConfirmacionesGINPIX), nil},
		{"sin capacidad", context.Background(), nil},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			got, ok := selector.elegir(caso.ctx)
			if got != caso.esperado || ok != (caso.esperado != nil) {
				t.Fatalf("soporte=%p válido=%v", got, ok)
			}
		})
	}
	autoridadOriginal := &autoridadSeguimientoCeseDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: base}}
	autoridadAdicional := &autoridadSeguimientoCeseDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: adicional}}
	autoridadOriginal.adicional = autoridadAdicional
	if autoridadOriginal.paraPeticion(capacidad(segundo, httpinterno.RutaNoIncorporaciones)) != autoridadAdicional {
		t.Fatal("el consumidor c22 no selecciona el soporte de la segunda hoja")
	}
	if autoridadOriginal.paraPeticion(capacidad(segundo, httpinterno.RutaCesesNombramiento)) != autoridadOriginal {
		t.Fatal("la segunda hoja selecciona el soporte en una ruta distinta de c22")
	}
	falsa := segundo
	falsa.principal = clonarPrincipalDesarrollo(segundo.principal)
	falsa.principal.ID = original.principal.ID
	if _, ok := selector.elegir(capacidad(falsa, httpinterno.RutaNoIncorporaciones)); ok {
		t.Fatal("sujeto original con hoja segunda aceptado")
	}
	falsa = segundo
	falsa.principal = clonarPrincipalDesarrollo(segundo.principal)
	falsa.principal.Attributes["certificate_sha256"] = original.principal.Attributes["certificate_sha256"]
	if _, ok := selector.elegir(capacidad(falsa, httpinterno.RutaNoIncorporaciones)); ok {
		t.Fatal("hoja original con sujeto segundo aceptada")
	}
}
