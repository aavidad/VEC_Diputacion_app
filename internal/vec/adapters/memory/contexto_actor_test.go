package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestAlmacenContextoActorDevuelveTodasLasCoincidenciasExactas(t *testing.T) {
	instante := instanteAlmacenContextoActorPrueba()
	solicitud := solicitudAlmacenContextoActorPrueba()
	primera := instantaneaAlmacenContextoActorPrueba(instante, solicitud, "a")
	segunda := instantaneaAlmacenContextoActorPrueba(instante, solicitud, "b")
	// Incluso dos vinculos de la misma cuenta, perfil y persona deben llegar a la
	// aplicacion para que esta rechace la ambiguedad; el adaptador no elige uno.
	segunda.PersonaRef = primera.PersonaRef
	ajena := instantaneaAlmacenContextoActorPrueba(instante, solicitud, "x")
	ajena.CuentaRef = referenciaAlmacenContextoActorPrueba("cta_", "x")
	otroPerfil := instantaneaAlmacenContextoActorPrueba(instante, solicitud, "y")
	otroPerfil.PerfilActivoRef = referenciaAlmacenContextoActorPrueba("prf_", "y")
	almacen, err := NuevoAlmacenContextoActor(primera, segunda, ajena, otroPerfil)
	if err != nil {
		t.Fatalf("crear almacen: %v", err)
	}

	resultados, err := almacen.BuscarInstantaneasContextoActor(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("buscar: %v", err)
	}
	if len(resultados) != 2 {
		t.Fatalf("el adaptador selecciono o mezclo perfiles: %#v", resultados)
	}
	if resultados[0].CuentaRef != solicitud.Cuenta.CuentaRef ||
		resultados[1].PerfilActivoRef != solicitud.PerfilActivoRef {
		t.Fatalf("resultado no exacto: %#v", resultados)
	}
}

func TestAlmacenContextoActorMantieneCopiasDefensivas(t *testing.T) {
	instante := instanteAlmacenContextoActorPrueba()
	solicitud := solicitudAlmacenContextoActorPrueba()
	entrada := instantaneaAlmacenContextoActorPrueba(instante, solicitud, "a")
	referenciaEsperada := entrada.Vinculos[0].Referencia
	almacen, err := NuevoAlmacenContextoActor(entrada)
	if err != nil {
		t.Fatalf("crear almacen: %v", err)
	}

	entrada.Vinculos[0].Referencia = referenciaAlmacenContextoActorPrueba("can_", "z")
	primera, err := almacen.BuscarInstantaneasContextoActor(context.Background(), solicitud)
	if err != nil || len(primera) != 1 {
		t.Fatalf("primera lectura: %#v, %v", primera, err)
	}
	if primera[0].Vinculos[0].Referencia != referenciaEsperada {
		t.Fatal("el constructor conservo la memoria de entrada")
	}
	primera[0].Vinculos[0].Referencia = referenciaAlmacenContextoActorPrueba("can_", "x")
	segunda, err := almacen.BuscarInstantaneasContextoActor(context.Background(), solicitud)
	if err != nil || segunda[0].Vinculos[0].Referencia != referenciaEsperada {
		t.Fatalf("la salida altero el almacen: %#v, %v", segunda, err)
	}
}

func TestAlmacenContextoActorRechazaInstantaneaInvalida(t *testing.T) {
	instante := instanteAlmacenContextoActorPrueba()
	solicitud := solicitudAlmacenContextoActorPrueba()
	instantanea := instantaneaAlmacenContextoActorPrueba(instante, solicitud, "a")
	instantanea.PersonaRef = "referencia-persona-sintetica-no-opaca"
	if almacen, err := NuevoAlmacenContextoActor(instantanea); almacen != nil ||
		!errors.Is(err, domain.ErrInstantaneaContextoActorInvalida) {
		t.Fatalf("instantanea no opaca admitida: almacen=%#v error=%v", almacen, err)
	}
}

func TestAlmacenContextoActorNoInfierePerfilEnConsultaDirecta(t *testing.T) {
	instante := instanteAlmacenContextoActorPrueba()
	solicitud := solicitudAlmacenContextoActorPrueba()
	almacen, err := NuevoAlmacenContextoActor(instantaneaAlmacenContextoActorPrueba(instante, solicitud, "a"))
	if err != nil {
		t.Fatalf("crear almacen: %v", err)
	}
	for _, perfil := range []string{"", solicitud.PerfilActivoRef + "*", " " + solicitud.PerfilActivoRef} {
		invalida := solicitud
		invalida.PerfilActivoRef = perfil
		if resultados, err := almacen.BuscarInstantaneasContextoActor(context.Background(), invalida); len(resultados) != 0 || !errors.Is(err, ports.ErrFuenteContextoActorNoDisponible) {
			t.Fatalf("perfil no canonico consultado: %#v, %v", resultados, err)
		}
	}
	var nulo *AlmacenContextoActor
	if _, err := nulo.BuscarInstantaneasContextoActor(context.Background(), solicitud); !errors.Is(err, ports.ErrFuenteContextoActorNoDisponible) {
		t.Fatalf("receptor nulo no fallo cerrado: %v", err)
	}
}

func TestAlmacenContextoActorEsSeguroEnLecturasConcurrentes(t *testing.T) {
	instante := instanteAlmacenContextoActorPrueba()
	solicitud := solicitudAlmacenContextoActorPrueba()
	almacen, err := NuevoAlmacenContextoActor(instantaneaAlmacenContextoActorPrueba(instante, solicitud, "a"))
	if err != nil {
		t.Fatalf("crear almacen: %v", err)
	}

	const lectores = 64
	errores := make(chan error, lectores)
	var grupo sync.WaitGroup
	for indice := 0; indice < lectores; indice++ {
		grupo.Add(1)
		go func(indice int) {
			defer grupo.Done()
			for repeticion := 0; repeticion < 50; repeticion++ {
				resultados, err := almacen.BuscarInstantaneasContextoActor(context.Background(), solicitud)
				if err != nil || len(resultados) != 1 || resultados[0].Validar() != nil {
					errores <- fmt.Errorf("lector %d repeticion %d: resultados=%d error=%v", indice, repeticion, len(resultados), err)
					return
				}
				resultados[0].Vinculos[0].Referencia = referenciaAlmacenContextoActorPrueba("can_", "z")
			}
		}(indice)
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Error(err)
	}
	finales, err := almacen.BuscarInstantaneasContextoActor(context.Background(), solicitud)
	if err != nil || len(finales) != 1 || finales[0].Validar() != nil {
		t.Fatalf("almacen alterado tras concurrencia: %#v, %v", finales, err)
	}
}

func solicitudAlmacenContextoActorPrueba() domain.SolicitudContextoActor {
	return domain.SolicitudContextoActor{
		Cuenta: domain.CuentaAutenticadaContextoActor{
			CuentaRef: referenciaAlmacenContextoActorPrueba("cta_", "a"),
			Metodo:    domain.AuthMethodCertificate,
			Garantia:  domain.AuthAssuranceHigh,
		},
		PerfilActivoRef: referenciaAlmacenContextoActorPrueba("prf_", "p"),
	}
}

func instantaneaAlmacenContextoActorPrueba(
	instante time.Time,
	solicitud domain.SolicitudContextoActor,
	caracter string,
) domain.InstantaneaContextoActor {
	return domain.InstantaneaContextoActor{
		VinculoRef:      referenciaAlmacenContextoActorPrueba("vca_", caracter),
		VinculoVersion:  1,
		CuentaRef:       solicitud.Cuenta.CuentaRef,
		PersonaRef:      referenciaAlmacenContextoActorPrueba("per_", caracter),
		PersonaVersion:  1,
		PerfilActivoRef: solicitud.PerfilActivoRef,
		PerfilVersion:   1,
		Estado:          domain.EstadoVinculoContextoActorActivo,
		VigenteDesde:    instante.Add(-time.Hour),
		VigenteHasta:    instante.Add(time.Hour),
		Vinculos: []domain.VinculoReferenciaContextoActor{
			{
				VinculoRef:   referenciaAlmacenContextoActorPrueba("vin_", caracter),
				Version:      1,
				Tipo:         domain.TipoReferenciaContextoActorCandidato,
				Referencia:   referenciaAlmacenContextoActorPrueba("can_", "c"),
				Estado:       domain.EstadoVinculoContextoActorActivo,
				VigenteDesde: instante.Add(-time.Hour),
				VigenteHasta: instante.Add(time.Hour),
			},
		},
	}
}

func instanteAlmacenContextoActorPrueba() time.Time {
	return time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
}

func referenciaAlmacenContextoActorPrueba(prefijo, caracter string) string {
	return prefijo + strings.Repeat(caracter, 22)
}
