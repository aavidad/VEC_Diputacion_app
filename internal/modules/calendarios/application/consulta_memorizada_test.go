package application

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/calendarios/domain"
	"vec-diputacion-granada/internal/modules/calendarios/ports"
)

// historiaContada cuenta las lecturas del repositorio: en PostgreSQL cada una
// es una consulta (una transacción de solo lectura).
type historiaContada struct {
	*historiaEnMemoria
	lecturas atomic.Int64
}

func (h *historiaContada) VersionesVigentes(ctx context.Context, c ports.ConsultaVersiones) ([]domain.VersionConDias, error) {
	h.lecturas.Add(1)
	return h.historiaEnMemoria.VersionesVigentes(ctx, c)
}

func servicioContado(t *testing.T) (*Servicio, *historiaContada) {
	t.Helper()
	h := &historiaContada{historiaEnMemoria: historia(t)}
	s, err := NuevoServicio(h, relojFijo(ahora))
	if err != nil {
		t.Fatal(err)
	}
	return s, h
}

// Cincuenta plazos de un cuadro (un grupo por fecha de entrada) leen el
// calendario de 2026 una sola vez y dan los mismos vencimientos que antes,
// cuando se leía el calendario en cada plazo.
func TestParaConsultaLeeCadaAnioUnaVezConLosMismosPlazos(t *testing.T) {
	const grupos = 50
	original, antes := servicioContado(t)
	porConsulta, despues := servicioContado(t)
	consulta := porConsulta.ParaConsulta()
	inicio := time.Date(2026, 1, 12, 9, 0, 0, 0, time.UTC)
	for i := range grupos {
		solicitud := ports.SolicitudCalculoPlazo{
			NotificadoEn: inicio.AddDate(0, 0, 5*i), Unidad: domain.UnidadDiasHabiles,
			Cantidad: 10, MunicipioSede: "municipio:ine:18087",
		}
		esperado, errEsperado := original.CalcularPlazo(context.Background(), solicitud)
		obtenido, errObtenido := consulta.CalcularPlazo(context.Background(), solicitud)
		if errEsperado != nil || !reflect.DeepEqual(esperado, obtenido) || errObtenido != nil {
			t.Fatalf("grupo %d distinto:\n%+v %v\n%+v %v", i, esperado, errEsperado, obtenido, errObtenido)
		}
	}
	if antes.lecturas.Load() != 2*grupos {
		t.Fatalf("antes se esperaban %d lecturas y hubo %d", 2*grupos, antes.lecturas.Load())
	}
	// Una lectura de los calendarios locales y otra de los oficiales.
	if despues.lecturas.Load() != 2 {
		t.Fatalf("la consulta leyó el calendario %d veces", despues.lecturas.Load())
	}
}

// Un fallo no se memoriza, la memoria no pasa a otra consulta y el contexto
// cancelado se respeta.
func TestParaConsultaNoMemorizaFallosNiSeComparte(t *testing.T) {
	s, h := servicioContado(t)
	solicitud := ports.SolicitudCalculoPlazo{
		Inicio: f(t, "2026-03-02"), Unidad: domain.UnidadDiasHabiles, Cantidad: 2, MunicipioSede: "municipio:ine:18087",
	}
	consulta := s.ParaConsulta()
	h.fallo = errors.New("caida")
	if _, err := consulta.CalcularPlazo(context.Background(), solicitud); err == nil {
		t.Fatal("un fallo del repositorio dio un plazo")
	}
	h.fallo = nil
	if _, err := consulta.CalcularPlazo(context.Background(), solicitud); err != nil {
		t.Fatalf("tras el fallo no se volvió a leer: %v", err)
	}
	if _, err := s.ParaConsulta().CalcularPlazo(context.Background(), solicitud); err != nil {
		t.Fatal(err)
	}
	if h.lecturas.Load() != 5 {
		t.Fatalf("lecturas: %d (fallo, dos de la primera consulta y dos de otra consulta)", h.lecturas.Load())
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := s.ParaConsulta().CalcularPlazo(cancelado, solicitud); !errors.Is(err, context.Canceled) {
		t.Fatalf("contexto cancelado: %v", err)
	}
	var nulo *Servicio
	if consulta := nulo.ParaConsulta(); consulta != ports.ConsultaCalendarios(nulo) {
		t.Fatal("un servicio nulo debe seguir siendo nulo")
	}
}
