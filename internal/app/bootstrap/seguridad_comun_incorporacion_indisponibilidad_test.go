package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type sesionFalloIncorporacionPrueba struct{ err error }

func (s sesionFalloIncorporacionPrueba) ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error) {
	return contextoSeguridadComunDesarrollo{}, s.err
}

func TestSeguridadComunIncorporacionConservaCaidaYDenegacion(t *testing.T) {
	ruta := httpinterno.RutaIncorporacionEjercicioV2
	d := fronteraComunPrueba("ct-incorporacion", http.MethodGet, ruta, false)
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{d})
	if err != nil {
		t.Fatal(err)
	}
	frontera := fronteraSeguridadComunDesarrollo{metodo: http.MethodGet, ruta: ruta, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: catalogo, descriptor: d}
	ctx := context.WithValue(context.Background(), claveFronteraSeguridadComunDesarrollo{}, frontera)
	for _, caso := range []struct {
		nombre string
		err    error
		espera error
	}{
		{"dependencia", ct.ErrConsultaRRHHNoDisponible, ct.ErrConsultaRRHHNoDisponible},
		{"dependencia_unida_denegacion", errors.Join(ErrSeguridadComunDesarrolloDenegada, ct.ErrConsultaRRHHNoDisponible), ct.ErrConsultaRRHHNoDisponible},
		{"revocacion", ErrSeguridadComunDesarrolloDenegada, ErrSeguridadComunDesarrolloDenegada},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			s, err := nuevaSeguridadComunDesarrollo(sesionFalloIncorporacionPrueba{caso.err}, relojContratacionTemporalDesarrollo{})
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.ResolverContexto(ctx)
			if !errors.Is(err, caso.espera) || caso.espera == ct.ErrConsultaRRHHNoDisponible && errors.Is(err, ErrSeguridadComunDesarrolloDenegada) {
				t.Fatalf("categoría de sesión incorrecta: %v", err)
			}
		})
	}
	if !rutaSesionConIndisponibilidadCTDesarrollo(ruta) || rutaSesionConIndisponibilidadCTDesarrollo(httpinterno.RutaConsultaSeguimientoV2) {
		t.Fatal("la excepción de indisponibilidad excede la ruta exacta comprobada")
	}
}
