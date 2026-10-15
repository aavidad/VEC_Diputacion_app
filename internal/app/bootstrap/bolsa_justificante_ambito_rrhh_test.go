package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

func TestJustificanteRRHHCapturaSoloLaRelacionYElAmbitoAutorizados(t *testing.T) {
	sesion, fuente := escenarioLecturaRespuestasLlamamiento(t, time.Now().UTC().Truncate(time.Microsecond))
	c := capturadorAmbitoJustificanteBolsaRRHH{fuente: fuente}
	q := bolsaports.SolicitudConsultaJustificanteRenunciaRRHH{ActorRegistradoV2: sesion.Resultado,
		VinculoV2: sesion.Vinculo, BolsaRef: "bolsa:prueba", ParticipacionRef: "participacion:prueba"}
	a := bolsaports.AmbitoJustificanteRenunciaRRHH{BolsaRef: q.BolsaRef, ParticipacionRef: q.ParticipacionRef,
		LlamamientoRef: "llamamiento:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		UnidadRef:      "unidad:prueba", AmbitoRef: "ambito:prueba"}
	if captura, err := c.CapturarAmbitoJustificanteRenunciaRRHH(t.Context(), q, a); err != nil ||
		captura.BolsaRef != q.BolsaRef || captura.LlamamientoRef != a.LlamamientoRef || len(captura.Orden) == 0 {
		t.Fatalf("la sesión autorizada perdió su captura: %v", err)
	}
	for _, cambiar := range []func(*bolsaports.AmbitoJustificanteRenunciaRRHH){
		func(a *bolsaports.AmbitoJustificanteRenunciaRRHH) { a.UnidadRef = "unidad:ajena" },
		func(a *bolsaports.AmbitoJustificanteRenunciaRRHH) { a.AmbitoRef = "ambito:ajeno" },
		func(a *bolsaports.AmbitoJustificanteRenunciaRRHH) { a.BolsaRef = "bolsa:ajena" },
		func(a *bolsaports.AmbitoJustificanteRenunciaRRHH) { a.ParticipacionRef = "participacion:ajena" },
	} {
		ajeno := a
		cambiar(&ajeno)
		if _, err := c.CapturarAmbitoJustificanteRenunciaRRHH(context.Background(), q, ajeno); !errors.Is(err, vec.ErrAutorizacionDenegada) {
			t.Fatal("se preparó una lectura fuera del ámbito o de la relación solicitada")
		}
	}
}
