package bootstrap

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

func TestTokenFinalizacionCTConservaVersionEnRotacionYLigaOrigen(t *testing.T) {
	antes, err := nuevoDerivadorTokenFinalizacionCTDesarrollo(nuevoDerivadorIdempotenciaPrueba(t, 3, 2))
	if err != nil {
		t.Fatal(err)
	}
	despues, err := nuevoDerivadorTokenFinalizacionCTDesarrollo(nuevoDerivadorIdempotenciaPrueba(t, 2, 1))
	if err != nil {
		t.Fatal(err)
	}
	datos := bolsaports.DatosDerivacionTokenFinalizacionCT{
		BolsaRef: "bolsa:constituida", ClaveB7: "clave:ct:uno", ActorRef: "per_actor_001",
		HuellaOrigenSHA256: strings.Repeat("a", 64),
	}
	primero, version, err := antes.DerivarTokenFinalizacionCT(t.Context(), datos)
	if err != nil || len(primero) != 32 || version != "bolsa.finalizacion.v1:g3" {
		t.Fatalf("primera version: %q, %v", version, err)
	}
	segundo, versionNueva, err := despues.DerivarTokenFinalizacionCT(t.Context(), datos)
	if err != nil || len(segundo) != 32 || versionNueva != "bolsa.finalizacion.v1:g2" || bytes.Equal(primero, segundo) {
		t.Fatalf("rotacion no cambio la version activa: %q, %v", versionNueva, err)
	}
	datos.VersionClave = versionNueva
	recuperado, versionHistorica, err := antes.DerivarTokenFinalizacionCT(t.Context(), datos)
	if err != nil || versionHistorica != versionNueva || !bytes.Equal(recuperado, segundo) {
		t.Fatalf("replay no recupero g2: %q, %v", versionHistorica, err)
	}
	datos.HuellaOrigenSHA256 = strings.Repeat("b", 64)
	otro, _, err := despues.DerivarTokenFinalizacionCT(t.Context(), datos)
	if err != nil || bytes.Equal(otro, segundo) {
		t.Fatal("el token no quedo ligado al origen")
	}
	datos.VersionClave = "bolsa.finalizacion.v1:g999"
	if token, _, err := despues.DerivarTokenFinalizacionCT(context.Background(), datos); err == nil || len(token) != 0 {
		t.Fatal("una version no disponible debe denegar")
	}
}

type consultaPuestoM10Prueba func(context.Context, personaldomain.SelectorOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error)

func (f consultaPuestoM10Prueba) Consultar(ctx context.Context, s personaldomain.SelectorOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
	return f(ctx, s)
}

func TestResolverPuestoRPTConstituidoExigePaginacionCompletaYUnicoUUID(t *testing.T) {
	const puesto = "puesto:10000000-0000-4000-8000-000000000001"
	instante := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	consultas := 0
	cliente := consultaPuestoM10Prueba(func(_ context.Context, s personaldomain.SelectorOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
		consultas++
		p := personalports.PaginaOrganizacionHistorica{Selector: s, VersionRPTRef: "rpt:publicada"}
		if s.Cursor == "" {
			p.CursorSiguiente = "p_siguiente"
		} else {
			p.PuestosIndividuales = []personaldomain.PuestoIndividualOrganizacionHistorica{{
				Traza:        personaldomain.TrazaOrganizacionHistorica{ID: puesto},
				CodigoFuente: "520-001/01", UnidadID: "unidad:centro:520", EstadoEstructural: "vigente",
			}}
		}
		return personalports.ResultadoConsultaOrganizacionHistorica{
			Pagina: p, Evidencia: personalports.EvidenciaConsultaOrganizacionHistorica{AuditoriaRef: "auditoria:m10:real"},
		}, nil
	})
	resolver := nuevoResolutorPuestoRPTConstituidoCT(cliente, "organizacion:desarrollo:dipgra", func() time.Time { return instante })
	obtenido, err := resolver(t.Context(), "520-001", "unidad:centro:520")
	if err != nil || obtenido != puesto || consultas != 2 {
		t.Fatalf("puesto=%q consultas=%d error=%v", obtenido, consultas, err)
	}
	if otro, err := resolver(t.Context(), "520-001", "unidad:centro:999"); err == nil || otro != "" {
		t.Fatal("una unidad diferente no puede resolver el puesto")
	}
}
