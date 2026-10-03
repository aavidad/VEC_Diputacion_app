package auditoria

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func filtroNominalPrueba() FiltroNominal {
	return FiltroNominal{ActorRef: "principal:sintetico:1", Desde: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC),
		Hasta: time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC), Limite: 20, FinalidadRef: "auditoria_rrhh",
		MotivoRef: "motivos_autorizacion_auditoria:1:motivo_a36f10964f684ec2bea55091667db47a"}
}

func TestNominalFiltroLigaTodosLosSelectores(t *testing.T) {
	f := filtroNominalPrueba()
	h, err := HuellaFiltroNominal(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutar := range []func(*FiltroNominal){
		func(f *FiltroNominal) { f.ActorRef = "principal:sintetico:2" },
		func(f *FiltroNominal) { f.RecursoRef = "recurso:sintetico:1" },
		func(f *FiltroNominal) { f.Accion = "personal.consultar" },
		func(f *FiltroNominal) { f.Desde = f.Desde.Add(time.Minute) },
		func(f *FiltroNominal) { f.Hasta = f.Hasta.Add(time.Minute) },
		func(f *FiltroNominal) { f.Limite++ },
		func(f *FiltroNominal) { f.AntesSecuencia = 25 },
		func(f *FiltroNominal) { f.FinalidadRef = "revision_sintetica" },
		func(f *FiltroNominal) { f.MotivoRef = "catalogo:1:otro" },
	} {
		c := f
		mutar(&c)
		otra, err := HuellaFiltroNominal(c)
		if err != nil || otra == h {
			t.Fatalf("selector not bound: %+v err=%v", c, err)
		}
	}
	f.ActorRef = ""
	f.Accion = "personal.consultar"
	if f.Validar() != nil {
		t.Fatal("exact action anchor rejected")
	}
	f.Accion = ""
	if f.Validar() == nil {
		t.Fatal("unanchored query accepted")
	}
	f.ActorRef = "*"
	if f.Validar() == nil {
		t.Fatal("wildcard query accepted")
	}
	f = filtroNominalPrueba()
	f.Hasta = f.Desde.Add(MaximoIntervalo + time.Microsecond)
	if f.Validar() == nil {
		t.Fatal("unbounded time range accepted")
	}
}

func TestNominalCursorRechazaAlteracionYTraslado(t *testing.T) {
	s := &ServicioNominal{claveCursor: [32]byte{1}}
	f := filtroNominalPrueba()
	c, err := s.codificarCursorNominal(f, 40)
	if err != nil {
		t.Fatal(err)
	}
	if antes, err := s.decodificarCursorNominal(c, f); err != nil || antes != 40 {
		t.Fatalf("cursor=%d err=%v", antes, err)
	}
	f.ActorRef = "principal:sintetico:2"
	if _, err := s.decodificarCursorNominal(c, f); err == nil {
		t.Fatal("cursor transplanted to another filter")
	}
	if _, err := s.decodificarCursorNominal(c[:len(c)-2]+"ZZ", filtroNominalPrueba()); err == nil {
		t.Fatal("altered cursor accepted")
	}
}

func TestNominalHTTPParserNoAceptaIdentidadNiClavesAmbiguas(t *testing.T) {
	base := `{"actor_ref":"principal:sintetico:1","desde":"2026-10-03T00:00:00Z","hasta":"2026-10-04T00:00:00Z","limite":20,"finalidad_ref":"auditoria_rrhh","motivo_ref":"catalogo:1:motivo"}`
	for _, cuerpo := range []string{
		strings.Replace(base, `"actor_ref":`, `"perfil_activo_ref":"perfil:admin","actor_ref":`, 1),
		strings.Replace(base, `"actor_ref":`, `"ACTOR_REF":`, 1),
		strings.Replace(base, `"limite":20`, `"limite":20,"limite":1`, 1),
		strings.Replace(base, `"limite":20`, `"limite":null`, 1),
		base + `{}`, strings.Repeat(" ", maximoCuerpoConsulta) + base,
	} {
		r := httptest.NewRequest("POST", RutaConsultaNominal, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		if _, err := decodificarCuerpoNominal(httptest.NewRecorder(), r); err == nil {
			t.Fatal("untrusted HTTP shape accepted")
		}
	}
	r := httptest.NewRequest("POST", RutaConsultaNominal, strings.NewReader(base))
	r.Header.Set("Content-Type", "application/json")
	if _, err := decodificarCuerpoNominal(httptest.NewRecorder(), r); err != nil || r.Body != http.NoBody || r.ContentLength != 0 {
		t.Fatal("valid body not consumed before identity resolution")
	}
}

type fuenteNominalPrueba struct{ llamadas int }

func (f *fuenteNominalPrueba) ConsultarAuditoriaNominal(context.Context, ConsultaNominalAutorizada) (PaginaFuenteNominal, error) {
	f.llamadas++
	return PaginaFuenteNominal{}, ErrNoDisponible
}

func TestNominalIdentidadCaducadaYEmisorDenegadoNuncaLeenFuente(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	p := peticionAuditoriaIdentidadPrueba(t, ahora)
	f := FiltroNominal{ActorRef: "principal:sintetico:1", Desde: p.Filtro.Desde, Hasta: p.Filtro.Hasta,
		Limite: p.Filtro.Limite, FinalidadRef: p.Filtro.FinalidadRef, MotivoRef: p.Filtro.MotivoRef}
	emisor := &emisorAuditoriaIdentidadPrueba{}
	fuente := &fuenteNominalPrueba{}
	s, err := NuevoServicioNominal(emisor, fuente)
	if err != nil {
		t.Fatal(err)
	}
	s.ahora = func() time.Time { return ahora }
	if _, err = s.Consultar(t.Context(), PeticionNominal{Filtro: f, Contexto: p.Contexto}); !errors.Is(err, ErrDenegada) || fuente.llamadas != 0 || emisor.llamadas != 1 {
		t.Fatalf("read without V3: err=%v source=%d issuer=%d", err, fuente.llamadas, emisor.llamadas)
	}
	s.ahora = func() time.Time { return ahora.Add(2 * time.Hour) }
	if _, err = s.Consultar(t.Context(), PeticionNominal{Filtro: f, Contexto: p.Contexto}); !errors.Is(err, ErrDenegada) || fuente.llamadas != 0 || emisor.llamadas != 1 {
		t.Fatalf("expired identity reached issuer: err=%v", err)
	}
}

func TestNominalProyeccionNoConvierteConsumoEnExitoDeNegocio(t *testing.T) {
	f := filtroNominalPrueba()
	r := RegistroNominal{AuditoriaRef: "aud_v3_sintetica", Secuencia: 5, ActorRef: f.ActorRef, PerfilActivoRef: "perfil:sintetico:1",
		AsignacionRef: "asignacion:sintetica:1", VersionRolRef: "rol:sintetico:1", ModuloID: "personal", Accion: "personal.consultar",
		RecursoRef: "recurso:sintetico:1", FinalidadRef: "gestion_personal", CorrelacionRef: "correlacion_sintetica",
		Canal: "interna_corporativa", RegistradaEn: f.Desde.Add(time.Hour), TipoRegistro: TipoRegistroConsumoConfirmado}
	if !registroNominalValido(r, f) {
		t.Fatal("consumption projection rejected")
	}
	r.TipoRegistro = "permitido"
	if registroNominalValido(r, f) {
		t.Fatal("business permission substituted for consumption")
	}
	r.TipoRegistro = TipoRegistroConsumoConfirmado
	r.ActorRef = "principal:otro"
	if registroNominalValido(r, f) {
		t.Fatal("foreign actor returned")
	}
}
