package competenciacentral

import (
	"context"
	"errors"
	"testing"
	"time"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type revalidadorCentralPrueba struct {
	llamadas  int
	err       error
	evidencia vecdomain.EvidenciaAsignacionCompetencialV1
}

func (r *revalidadorCentralPrueba) RevalidarAsignacionCompetencialV1(_ context.Context, _ vecdomain.SolicitudAsignacionCompetencialV1, e vecdomain.EvidenciaAsignacionCompetencialV1) error {
	r.llamadas++
	r.evidencia = e
	return r.err
}

type revalidadorBindingPrueba struct {
	llamadas int
	err      error
	vinculo  VinculoCertificadoFirmante
}

func (r *revalidadorBindingPrueba) RevalidarVinculoCertificadoFirmante(_ context.Context, v VinculoCertificadoFirmante) error {
	r.llamadas++
	r.vinculo = v
	return r.err
}

func TestRevalidadorMantieneFuentesCompletasYNoAceptaProyeccion(t *testing.T) {
	f, i, _, _, q := datosPrueba(t)
	a, err := f.AcreditarCompetenciaCentral(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	central := &revalidadorCentralPrueba{}
	binding := &revalidadorBindingPrueba{}
	r, err := NuevoRevalidador(central, binding, f.reloj)
	if err != nil {
		t.Fatal(err)
	}
	// La misma acreditación se revalida de nuevo en replay con fuentes actuales.
	for j := 0; j < 2; j++ {
		if err = r.RevalidarCompetenciaCentral(context.Background(), a); err != nil {
			t.Fatal(err)
		}
	}
	if central.llamadas != 2 || binding.llamadas != 2 || binding.vinculo != i.vinculo || central.evidencia.Cargo.Version != 2 || central.evidencia.EnlaceOcupante.Version != 4 {
		t.Fatal("perdió fuentes centrales")
	}
	if err = r.RevalidarCompetenciaCentral(context.Background(), Acreditacion{proyeccion: a.Proyeccion()}); err != ctports.ErrCompetenciaFirmanteNoAcreditada {
		t.Fatal("proyección aceptada sin fuentes")
	}
}

func TestRevalidadorRevocacionYCaducidadImpidenEfecto(t *testing.T) {
	f, _, _, _, q := datosPrueba(t)
	a, err := f.AcreditarCompetenciaCentral(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	central := &revalidadorCentralPrueba{}
	binding := &revalidadorBindingPrueba{err: ctports.ErrCompetenciaFirmanteNoAcreditada}
	r, _ := NuevoRevalidador(central, binding, f.reloj)
	if err = r.RevalidarCompetenciaCentral(context.Background(), a); err != ctports.ErrCompetenciaFirmanteNoAcreditada || central.llamadas != 0 {
		t.Fatal("revocación aceptada")
	}
	binding.err = nil
	central.err = errors.New("valor privado de la fuente")
	if err = r.RevalidarCompetenciaCentral(context.Background(), a); err != ctports.ErrCompetenciaFirmanteNoDisponible {
		t.Fatal("error privado expuesto")
	}
	central.err = nil
	r.reloj = &relojPrueba{a.evidencia.CargoVigenteHasta}
	if err = r.RevalidarCompetenciaCentral(context.Background(), a); err != ctports.ErrCompetenciaFirmanteNoAcreditada {
		t.Fatal("caducidad aceptada")
	}
	r.reloj = &relojPrueba{a.evidencia.ComprobadaEn.Add(time.Microsecond)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = r.RevalidarCompetenciaCentral(ctx, a); err != context.Canceled {
		t.Fatal("cancelación ignorada")
	}
}
