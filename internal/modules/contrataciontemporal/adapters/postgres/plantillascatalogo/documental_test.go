package plantillascatalogo

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type resolverDocumentalNoInvocable struct{ llamadas int }

func (r *resolverDocumentalNoInvocable) ResolverContextoActor(context.Context) (vecdomain.ContextoActor, error) {
	r.llamadas++
	return vecdomain.ContextoActor{}, errors.New("no se debe resolver identidad")
}

type autorizadorDocumentalNoInvocable struct{ llamadas int }

func (a *autorizadorDocumentalNoInvocable) AutorizarCatalogoDocumental(context.Context, vecdomain.ContextoActor, string, vecdomain.RecursoAutorizable) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("no se debe autorizar")
}

func TestProveedorDocumentalRechazaOrganizacionAjenaAntesDeIdentidad(t *testing.T) {
	resolver := &resolverDocumentalNoInvocable{}
	autorizador := &autorizadorDocumentalNoInvocable{}
	p := &ProveedorDocumental{pool: &pgxpool.Pool{}, resolver: resolver, proveedor: autorizador, organizacionRef: organizacionCatalogoCT}
	s := app.SolicitudDocumental{
		Operacion: "listar", OrganizacionRef: "organizacion:ajena:001", ExpedienteRef: "expediente:ct:0001",
		VersionObservada: 7, ConsultaHuellaSHA256: strings.Repeat("a", 64),
	}
	if _, _, err := p.ObtenerPlantillasDocumento(context.Background(), s, time.Now().UTC()); !errors.Is(err, app.ErrNoDisponible) || resolver.llamadas != 0 || autorizador.llamadas != 0 {
		t.Fatalf("organización ajena pasó frontera: %v", err)
	}
	s.OrganizacionRef = ""
	if _, _, err := p.ObtenerPlantillasDocumento(context.Background(), s, time.Now().UTC()); !errors.Is(err, app.ErrNoDisponible) || resolver.llamadas != 0 || autorizador.llamadas != 0 {
		t.Fatalf("organización ausente pasó frontera: %v", err)
	}
}
