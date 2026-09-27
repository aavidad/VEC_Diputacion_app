package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type lectorPoliticaCesePrueba struct{ llamadas int }

func (l *lectorPoliticaCesePrueba) ConsultarPoliticaCese(context.Context, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (domain.PoliticaCese, error) {
	l.llamadas++
	return domain.PoliticaCese{}, nil
}

type autorizadorPoliticaCesePrueba struct{ llamadas int }

func (a *autorizadorPoliticaCesePrueba) EmitirMaterialAutorizacionAtestadaV3(context.Context, vd.SolicitudAutorizacionLigadaV3, vd.ResultadoContextoActorRegistradoV2) (vd.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	return vd.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errors.New("sin concesión")
}

func TestConsultaPoliticaCeseNoLlegaASQLSinIdentidadV3(t *testing.T) {
	l, a := &lectorPoliticaCesePrueba{}, &autorizadorPoliticaCesePrueba{}
	s, err := NuevoServicioConsultaPoliticaCese(l, a, func() time.Time { return time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Consultar(context.Background(), ports.OrdenConsultaPoliticaCese{})
	if !errors.Is(err, vd.ErrAutorizacionDenegada) || l.llamadas != 0 || a.llamadas != 0 {
		t.Fatalf("orden no vinculada llegó al lector: err=%v lector=%d autorizador=%d", err, l.llamadas, a.llamadas)
	}
}
