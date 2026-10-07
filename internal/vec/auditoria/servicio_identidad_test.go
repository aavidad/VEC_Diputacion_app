package auditoria

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type emisorAuditoriaIdentidadPrueba struct {
	llamadas int
	fallo    error
}

func (e *emisorAuditoriaIdentidadPrueba) EmitirMaterialAutorizacionAtestadaV3(
	context.Context, domain.SolicitudAutorizacionLigadaV3, domain.ResultadoContextoActorRegistradoV2,
) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, ports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	if e.fallo != nil {
		return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, e.fallo
	}
	return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ErrDenegada
}

type fuenteAuditoriaIdentidadPrueba struct{}

func (fuenteAuditoriaIdentidadPrueba) ConsultarAuditoria(context.Context, ConsultaAutorizada) (PaginaFuente, error) {
	return PaginaFuente{}, errors.New("la fuente no debe leerse sin V3")
}

func peticionAuditoriaIdentidadPrueba(t *testing.T, ahora time.Time) Peticion {
	t.Helper()
	identidad := identidadVigenteAuditoriaHTTPPrueba(t, ahora)
	motivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion_auditoria", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("a", 64),
		EntradaClave:         "motivo_a36f10964f684ec2bea55091667db47a",
	}
	return Peticion{
		Filtro: Filtro{
			Fuente: "ct", ExpedienteRef: "expediente:ct:prueba",
			Desde: ahora.Add(-time.Hour), Hasta: ahora.Add(time.Hour), Limite: 5,
			FinalidadRef: "auditoria_rrhh", MotivoRef: motivo.Referencia(),
		},
		Contexto: ContextoConsulta{
			Vinculo: identidad.Vinculo, Resultado: identidad.Resultado,
			Motivo: motivo, Correlacion: identidad.Correlacion,
		},
	}
}

func TestConsultaAuditoriaLlegaAlEmisorConRelojRealSubmicrosegundo(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	emisor := &emisorAuditoriaIdentidadPrueba{}
	s, err := NuevoServicio(emisor, fuenteAuditoriaIdentidadPrueba{}, fuenteAuditoriaIdentidadPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	s.ahora = func() time.Time { return time.Now().UTC().Truncate(time.Microsecond).Add(125 * time.Nanosecond) }
	_, err = s.Consultar(t.Context(), peticionAuditoriaIdentidadPrueba(t, ahora))
	if !errors.Is(err, ErrDenegada) || emisor.llamadas != 1 {
		t.Fatalf("la identidad vigente no alcanzo V3: llamadas=%d err=%v", emisor.llamadas, err)
	}
}

func TestConsultaAuditoriaCaducadaNoLlegaAlEmisor(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	emisor := &emisorAuditoriaIdentidadPrueba{}
	s, err := NuevoServicio(emisor, fuenteAuditoriaIdentidadPrueba{}, fuenteAuditoriaIdentidadPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	s.ahora = func() time.Time { return time.Now().Add(2 * time.Hour) }
	_, err = s.Consultar(t.Context(), peticionAuditoriaIdentidadPrueba(t, ahora))
	if !errors.Is(err, ErrDenegada) || emisor.llamadas != 0 {
		t.Fatalf("la identidad caducada alcanzo V3: llamadas=%d err=%v", emisor.llamadas, err)
	}
}
