package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	vec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type motivoRPTFixtureNoInvocable struct{ llamadas int }

func (m *motivoRPTFixtureNoInvocable) ValidarReferenciaMotivoAutorizacionV2(context.Context, vec.ReferenciaEntradaCatalogo, time.Time) error {
	m.llamadas++
	return vec.ErrAutorizacionDenegada
}

func TestRPTUsosFixtureConstructorSinREADYNoConsultaMotivoNiConecta(t *testing.T) {
	_, cfg, _ := admisionRPTUsosFixturePrueba()
	dsn := func(usuario string) string {
		return "postgres://" + usuario + "@127.0.0.1:55531/fixture_clon?sslmode=disable"
	}
	var err error
	cfg.ContratacionTemporalPostgreSQL, err = config.NuevaConfiguracionPostgreSQLContratacionTemporal(
		dsn("fixture_ejecutor"), dsn("fixture_gobierno"), dsn("fixture_registro"), dsn("fixture_confirmador"), dsn("fixture_lector"))
	if err != nil {
		t.Fatal(err)
	}
	c := ConfiguracionRPTUsosFixtureV3{Activar: true, Host: "127.0.0.1", Puerto: 55531, Base: "fixture_clon",
		ClonHuellaSHA256: strings.Repeat("a", 64), ReadyClonHuellaSHA256: strings.Repeat("a", 64),
		AprobacionRef: "aprobacion:rpt:fixture", PreimagenAsignacionSHA256: strings.Repeat("b", 64),
		Descriptor: ports.DescriptorCatalogoRPT{CatalogoID: "fixture_rpt", ModuloID: "personal"},
		Motivo:     vec.ReferenciaEntradaCatalogo{CatalogoID: "fixture_motivos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("c", 64), EntradaClave: "motivo_00000000000000000000000000000001"}}
	motivo := &motivoRPTFixtureNoInvocable{}
	f, err := NuevoRPTUsosFixtureV3(t.Context(), cfg, c, motivo)
	if f != nil || !errors.Is(err, ErrSeguridadComunDesarrolloDenegada) || motivo.llamadas != 0 {
		t.Fatalf("constructor sin READY: fixture=%v, error=%v, consultas=%d", f, err, motivo.llamadas)
	}
}

func TestRPTUsosFixtureDeniegaDependenciasNulas(t *testing.T) {
	var f *RPTUsosFixtureV3
	if _, _, err := f.Emitir(context.Background(), ports.PreparacionAutorizacionUsoCategoriaRPT{}); !errors.Is(err, ErrSeguridadComunDesarrolloDenegada) {
		t.Fatal(err)
	}
	if f.Preparador() != nil || f.Gestor() != nil {
		t.Fatal("fixture nulo entregó puertos")
	}
	f.Cerrar()
	var motivos *motivoRPTFixtureNoInvocable
	if fixture, err := NuevoRPTUsosFixtureV3(context.Background(), config.Config{}, ConfiguracionRPTUsosFixtureV3{}, motivos); fixture != nil || !errors.Is(err, ErrSeguridadComunDesarrolloDenegada) {
		t.Fatal(err)
	}
	if _, err := PlanificarPerfilRPTUsosFixtureV3(config.Config{}, ports.DescriptorCatalogoRPT{}); !errors.Is(err, ErrSeguridadComunDesarrolloDenegada) {
		t.Fatal(err)
	}
}

func TestRPTUsosFixturePerfilNoConcedeLecturaNiOtroConsumidor(t *testing.T) {
	d := ports.DescriptorCatalogoRPT{CatalogoID: "fixture_rpt", ModuloID: "personal"}
	plantilla, err := plantillaRPTUsosFixture("cta_fixture", "prf_fixture", d, time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	r := vec.RecursoAutorizable{Referencia: "fixture:rpt:uso:001", ModuloID: d.ModuloID, Tipo: "uso_categoria",
		Ambitos: map[string]string{"catalogo_id": d.CatalogoID, "modulo_id": d.ModuloID, "consumidor": "contratacion_temporal"}}
	if !plantilla.AsignacionPerfil.Cubre(r) {
		t.Fatal("perfil no cubre su recurso nominal")
	}
	r.Ambitos["consumidor"] = "bolsa"
	if plantilla.AsignacionPerfil.Cubre(r) {
		t.Fatal("perfil cruza a otro consumidor")
	}
	for _, concesion := range plantilla.VersionRol.Concesiones {
		if concesion.Accion == "vec.catalogos.categorias.consultar_uso" || concesion.Accion == "vec.catalogos.categorias.listar_habilitadas" || concesion.Accion == "vec.catalogos.categorias.consultar_historica" {
			t.Fatal("el perfil añadió lectura")
		}
	}
}

func TestRPTUsosFixturePreimagenNuncaRehabilitaRevocacionNiActoAjeno(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC)
	plantilla, err := plantillaRPTUsosFixture("cta_fixture", "prf_fixture", ports.DescriptorCatalogoRPT{CatalogoID: "fixture_rpt", ModuloID: "personal"}, ahora)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := plantilla.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if err := admitirPreimagenRPTUsosFixture(instantaneaPublicadaDesarrollo{}, false, plantilla, huella, ahora); err != nil {
		t.Fatal(err)
	}
	if err := admitirPreimagenRPTUsosFixture(instantaneaPublicadaDesarrollo{}, false, plantilla, strings.Repeat("a", 64), ahora); !errors.Is(err, ErrSeguridadComunDesarrolloDenegada) {
		t.Fatal(err)
	}
	publicada := instantaneaPublicadaDesarrollo{instantanea: plantilla, actoAsignacion: actoAsignacionRPTUsosFixture, actualizadaPor: plantilla.AsignacionPerfil.EmitidaPor, actoControl: actoControlRPTUsosFixture}
	if err := admitirPreimagenRPTUsosFixture(publicada, true, plantilla, huella, ahora); err != nil {
		t.Fatal(err)
	}
	publicada.actoAsignacion = "acto:otro:circuito"
	if err := admitirPreimagenRPTUsosFixture(publicada, true, plantilla, huella, ahora); !errors.Is(err, ErrSeguridadComunDesarrolloDenegada) {
		t.Fatal(err)
	}
	publicada.actoAsignacion = actoAsignacionRPTUsosFixture
	publicada.instantanea.AsignacionPerfil.Estado = vec.EstadoAsignacionPerfilRevocada
	publicada.instantanea.AsignacionPerfil.RevocadaPor = "autoridad:revocacion"
	publicada.instantanea.AsignacionPerfil.RevocadaEn = ahora
	publicada.instantanea.AsignacionPerfil.RevocacionRef = "revocacion:rpt:fixture"
	huellaRevocada, err := publicada.instantanea.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if err := admitirPreimagenRPTUsosFixture(publicada, true, plantilla, huellaRevocada, ahora); !errors.Is(err, ErrSeguridadComunDesarrolloDenegada) {
		t.Fatal(err)
	}
}
