package interna

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autoridadesCronosPrueba struct{}

func (autoridadesCronosPrueba) AhoraUTC() time.Time { return time.Now().UTC() }
func (autoridadesCronosPrueba) ResolverContextoActorMarcajePropio(*http.Request) (vecdomain.ContextoActor, error) {
	return vecdomain.ContextoActor{}, errors.New("sin capsula")
}
func (autoridadesCronosPrueba) AcreditarCanalMarcajePropio(*http.Request) (domain.AcreditacionCanalMarcaje, error) {
	return domain.AcreditacionCanalMarcaje{}, errors.New("sin canal")
}
func (autoridadesCronosPrueba) ProveerMaterialMarcajePropio(context.Context, domain.MaterialAutorizacionMarcajePropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin material")
}
func TestCronosComposicionExigeAuditoriaDeResultados(t *testing.T) {
	a := autoridadesCronosPrueba{}
	if h, err := NuevoManejadorMarcajePropio(&pgxpool.Pool{}, a, a, a, nil); h != nil || !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("composicion permite perder fallo postPDP")
	}
}
