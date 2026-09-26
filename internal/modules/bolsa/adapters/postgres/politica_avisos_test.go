package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestPoliticaAvisosPostgreSQLFallaCerradoAntesDeLaBase(t *testing.T) {
	if _, err := NuevoRepositorioPoliticaAvisosPostgreSQL(nil); !errors.Is(err, puertosbolsa.ErrPoliticaAvisosNoDisponible) {
		t.Fatalf("sin pool: %v", err)
	}
	var r *RepositorioPoliticaAvisosPostgreSQL
	valida := puertosbolsa.PublicacionPoliticaAvisos{CatalogoRef: "vec.bolsa.reglas:1:avisos", CatalogoSHA256: strings.Repeat("a", 64)}
	if _, err := r.PublicarPoliticaAvisos(context.Background(), valida); !errors.Is(err, puertosbolsa.ErrPoliticaAvisosNoDisponible) {
		t.Fatalf("repositorio nulo: %v", err)
	}
	r = &RepositorioPoliticaAvisosPostgreSQL{}
	invalida := valida
	invalida.Politica.EncadenamientoUmbralMeses = 18
	if _, err := r.PublicarPoliticaAvisos(context.Background(), invalida); !errors.Is(err, puertosbolsa.ErrPoliticaAvisosNoDisponible) {
		t.Fatalf("política inválida: %v", err)
	}
	if _, err := r.MarcasParticipaciones(context.Background(), "bolsa:1", time.Time{}); !errors.Is(err, puertosbolsa.ErrPoliticaAvisosNoDisponible) {
		t.Fatalf("sin corte: %v", err)
	}
}

func TestMarcaLeidaSoloAdmiteValoresConocidos(t *testing.T) {
	buena := dominiobolsa.MarcasParticipacion{ParticipacionRef: "p", PrestaServicios: "aviso", EnRevision: "solicitud_pendiente"}
	if !marcaLeidaValida(buena, 10) {
		t.Fatal("marca válida rechazada")
	}
	for _, mala := range []dominiobolsa.MarcasParticipacion{
		{ParticipacionRef: "p", PrestaServicios: "bloquear"},
		{ParticipacionRef: "p", EnRevision: "baja_propuesta"},
		{PrestaServicios: "aviso"},
	} {
		if marcaLeidaValida(mala, 0) {
			t.Fatalf("marca inválida admitida: %+v", mala)
		}
	}
	if marcaLeidaValida(buena, -1) {
		t.Fatal("días negativos admitidos")
	}
}
