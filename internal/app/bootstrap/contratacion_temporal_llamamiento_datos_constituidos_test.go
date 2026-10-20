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

func TestResolverPuestoRPTConstituidoExigePaginacionCompletaEIdentificadorPublicado(t *testing.T) {
	// El formato de la importación de Personal conserva todos estos dígitos.
	const puesto = "puesto:10000000-0000-f000-4000-000000000001"
	const unidadGestora = "unidad:desarrollo:rrhh"
	unidadPuesto, ok := unidadPuestoPublicadoCT(personaldomain.PuestoRPTPublico{CentroCodigo: "520"})
	if !ok || unidadPuesto != "unidad:centro:520" || unidadPuesto == unidadGestora {
		t.Fatalf("unidad del puesto=%q, unidad gestora=%q", unidadPuesto, unidadGestora)
	}
	instante := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	consultas := 0
	cliente := consultaPuestoM10Prueba(func(_ context.Context, s personaldomain.SelectorOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
		consultas++
		if s.UnidadClave != unidadPuesto && s.UnidadClave != unidadGestora {
			t.Fatalf("selector de unidad inesperado: %q", s.UnidadClave)
		}
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
	obtenido, err := resolver(t.Context(), "520-001", unidadPuesto)
	if err != nil || obtenido != puesto || consultas != 2 {
		t.Fatalf("puesto=%q consultas=%d error=%v", obtenido, consultas, err)
	}
	if otro, err := resolver(t.Context(), "520-001", unidadGestora); err == nil || otro != "" {
		t.Fatal("la unidad gestora no puede resolver el puesto de otro centro")
	}
}

func TestResolverPuestoRPTConstituidoDeniegaPuestoAjenoYAmbiguo(t *testing.T) {
	const puesto = "puesto:10000000-0000-4000-8000-000000000001"
	const otroPuesto = "puesto:10000000-0000-4000-8000-000000000002"
	instante := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	for _, caso := range []struct {
		nombre string
		filas  []personaldomain.PuestoIndividualOrganizacionHistorica
	}{
		{"ajeno", []personaldomain.PuestoIndividualOrganizacionHistorica{{
			Traza: personaldomain.TrazaOrganizacionHistorica{ID: puesto}, CodigoFuente: "520-001/01",
			UnidadID: "unidad:centro:999", EstadoEstructural: "vigente",
		}}},
		{"identificador_sin_formato", []personaldomain.PuestoIndividualOrganizacionHistorica{{
			Traza: personaldomain.TrazaOrganizacionHistorica{ID: "puesto:inventado"}, CodigoFuente: "520-001/01",
			UnidadID: "unidad:centro:520", EstadoEstructural: "vigente",
		}}},
		{"identificador_otra_clase", []personaldomain.PuestoIndividualOrganizacionHistorica{{
			Traza: personaldomain.TrazaOrganizacionHistorica{ID: "plaza:10000000-0000-f000-4000-000000000001"}, CodigoFuente: "520-001/01",
			UnidadID: "unidad:centro:520", EstadoEstructural: "vigente",
		}}},
		{"ambiguo", []personaldomain.PuestoIndividualOrganizacionHistorica{
			{Traza: personaldomain.TrazaOrganizacionHistorica{ID: puesto}, CodigoFuente: "520-001/01", UnidadID: "unidad:centro:520", EstadoEstructural: "vigente"},
			{Traza: personaldomain.TrazaOrganizacionHistorica{ID: otroPuesto}, CodigoFuente: "520-001/02", UnidadID: "unidad:centro:520", EstadoEstructural: "vigente"},
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			cliente := consultaPuestoM10Prueba(func(_ context.Context, s personaldomain.SelectorOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
				return personalports.ResultadoConsultaOrganizacionHistorica{
					Pagina:    personalports.PaginaOrganizacionHistorica{Selector: s, VersionRPTRef: "rpt:publicada", PuestosIndividuales: caso.filas},
					Evidencia: personalports.EvidenciaConsultaOrganizacionHistorica{AuditoriaRef: "auditoria:m10:real"},
				}, nil
			})
			resolver := nuevoResolutorPuestoRPTConstituidoCT(cliente, "organizacion:desarrollo:dipgra", func() time.Time { return instante })
			if obtenido, err := resolver(t.Context(), "520-001", "unidad:centro:520"); err == nil || obtenido != "" {
				t.Fatalf("puesto %s no rechazado: %q, %v", caso.nombre, obtenido, err)
			}
		})
	}
	for _, codigo := range []string{"", "520:ajeno", "520/otro", "520 espacio"} {
		if unidad, ok := unidadPuestoPublicadoCT(personaldomain.PuestoRPTPublico{CentroCodigo: codigo}); ok || unidad != "" {
			t.Fatalf("centro sin enlace fiable %q produjo unidad %q", codigo, unidad)
		}
	}
}
