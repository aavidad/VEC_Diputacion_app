package composicion

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	personalhttp "vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type actorOriginalFichaHTTPPrueba struct{}

func (actorOriginalFichaHTTPPrueba) ResolverActorFichaPropia(ctx context.Context) (core.ContextoActor, error) {
	id, err := IdentidadOriginalFichaPropia(ctx)
	if err != nil {
		return core.ContextoActor{}, err
	}
	return id.Resultado.Contexto.Clonar()
}

type consultaFichaHTTPComunPrueba struct{ llamadas int }

func (c *consultaFichaHTTPComunPrueba) Consultar(context.Context, domain.SolicitudFichaPropia) (ports.ResultadoFichaPropia, error) {
	c.llamadas++
	return ports.ResultadoFichaPropia{}, domain.ErrFichaPropiaNoDisponible
}

func TestFichaPropiaHTTPCanceladaTrasCapturaRegistraIntentoComun(t *testing.T) {
	instante := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	id := identidadIntentoFichaPropiaVigenciaPrueba(t, instante, "")
	ctx, resolver := contextoFichaCapturadaPrueba(t, id)
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	corr, err := correlacion.ValorCanonico()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(ctx)
	cancelar()
	destino := &destinoIntentosFichaPrueba{}
	registro, err := NuevoRegistroIntentosFichaPropia(destino, configuracionIntentosFichaPrueba())
	if err != nil {
		t.Fatal(err)
	}
	consulta := &consultaFichaHTTPComunPrueba{}
	handler, err := personalhttp.NuevoManejadorFichaPropia(actorOriginalFichaHTTPPrueba{}, consulta, registro, func() time.Time { return instante }, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, personalhttp.RutaFichaPropia, nil).WithContext(ctx))
	if w.Code != http.StatusServiceUnavailable || w.Body.String() != `{"error":"no_disponible"}` || consulta.llamadas != 0 || len(destino.ordenes) != 1 || resolver.llamadas != 1 {
		t.Fatalf("estado=%d consultas=%d intentos=%d resoluciones=%d", w.Code, consulta.llamadas, len(destino.ordenes), resolver.llamadas)
	}
	orden, err := destino.ordenes[0].Datos()
	if err != nil || orden.Datos.Resultado != core.ResultadoIntentoAuditoriaError || orden.Datos.CorrelacionRef != corr || orden.ResultadoContexto.Contexto.PerfilActivoRef != id.Resultado.Contexto.PerfilActivoRef || orden.ResultadoContexto.HuellaSHA256 != id.Resultado.HuellaSHA256 {
		t.Fatal("el intento común perdió la identidad o correlación originales", err)
	}
}
