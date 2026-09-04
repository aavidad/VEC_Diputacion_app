package application

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestSolicitudRegistrarResultadoFiscalizacionValidaResultadosCerrados(t *testing.T) {
	base := SolicitudRegistrarResultadoFiscalizacion{
		AutenticacionRef:  "aut_0123456789abcdefghijkl",
		SesionRef:         "ses_0123456789abcdefghijkl",
		PerfilRef:         "prf_0123456789abcdefghijkl",
		OrganizacionRef:   "organizacion:diputacion-granada",
		ExpedienteRef:     "expediente:temporal:sintetico:01",
		VersionEsperada:   5,
		ClaveIdempotencia: "123e4567-e89b-42d3-a456-426614174000",
	}
	casos := []struct {
		resultado     domain.ResultadoFiscalizacion
		observaciones string
		valida        bool
	}{
		{domain.FiscalizacionFavorable, "", true},
		{domain.FiscalizacionFavorableConObservaciones, "Condición sintética.", true},
		{domain.FiscalizacionDesfavorable, "Reparo sintético.", true},
		{domain.FiscalizacionFavorable, "residuo", false},
		{domain.FiscalizacionDesfavorable, "", false},
		{domain.ResultadoFiscalizacion("otro"), "", false},
	}
	for _, caso := range casos {
		solicitud := base
		solicitud.Resultado = caso.resultado
		solicitud.Observaciones = caso.observaciones
		err := solicitud.Validar()
		if (err == nil) != caso.valida ||
			(err != nil && !errors.Is(err, ErrSolicitudFiscalizacionInvalida)) {
			t.Fatalf("resultado %q, observaciones %q: %v", caso.resultado, caso.observaciones, err)
		}
	}
}

func TestNuevoServicioFiscalizacionesRechazaDependenciasNulas(t *testing.T) {
	servicio, err := NuevoServicioFiscalizaciones(
		nil, nil, nil, nil, nil, nil, nil, nil,
	)
	if servicio != nil || !errors.Is(err, ErrServicioFiscalizacionesInvalido) {
		t.Fatalf("constructor con dependencias nulas: servicio=%#v err=%v", servicio, err)
	}
}
