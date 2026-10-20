package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type repositorioPublicacionOficialPrueba struct {
	version                    ports.VersionPublicacionOficial
	errConsulta                error
	consultas, normalizaciones int
	declaraciones              int
	intencion                  ports.IntencionPublicacionOficial
}

func (r *repositorioPublicacionOficialPrueba) ConsultarVersiones(_ context.Context, ref string, limite int) (ports.PaginaVersionesPublicacionOficial, error) {
	r.consultas++
	if r.errConsulta != nil {
		return ports.PaginaVersionesPublicacionOficial{}, r.errConsulta
	}
	if ref != r.version.ConvocatoriaRef || limite != 1 {
		return ports.PaginaVersionesPublicacionOficial{}, ports.ErrPublicacionOficialNoVigente
	}
	return ports.PaginaVersionesPublicacionOficial{Items: []ports.VersionPublicacionOficial{r.version}, Completa: true}, nil
}

func (r *repositorioPublicacionOficialPrueba) NormalizarDeclaracion(_ context.Context, d ports.DeclaracionPublicacionOficial) (string, string, error) {
	r.normalizaciones++
	b, _ := json.Marshal(d)
	suma := sha256.Sum256(b)
	return string(b), hex.EncodeToString(suma[:]), nil
}

func (r *repositorioPublicacionOficialPrueba) Declarar(_ context.Context, i ports.IntencionPublicacionOficial) (ports.ReciboPublicacionOficial, error) {
	r.declaraciones++
	r.intencion = i
	return ports.ReciboPublicacionOficial{PublicacionRef: "publicacion_bases:" + strings.Repeat("b", 64)}, nil
}

func declaracionPublicacionPrueba() ports.DeclaracionPublicacionOficial {
	boletin := int64(201)
	return ports.DeclaracionPublicacionOficial{AnuncioRef: "ENSAYO-BOP-2026-000201", BoletinNumero: &boletin,
		ClaveIdempotencia: "f195-prueba-0001", ConvocatoriaRef: "cv1_" + strings.Repeat("a", 64) + "_v1",
		FechaPublicacion: "2026-10-15", HuellaVersionSHA256: strings.Repeat("c", 64), Medio: "bop_granada"}
}

func solicitudPublicacionPrueba(t *testing.T) ports.SolicitudDeclaracionPublicacionOficial {
	t.Helper()
	base := solicitudSituacionPrueba(t, time.Date(2026, 9, 30, 11, 0, 0, 0, time.UTC))
	return ports.SolicitudDeclaracionPublicacionOficial{Vinculo: base.Vinculo, ResultadoContexto: base.ResultadoContexto,
		Correlacion: base.Correlacion, MotivoAutorizacion: base.MotivoAutorizacion, Declaracion: declaracionPublicacionPrueba()}
}

func versionPublicacionPrueba() ports.VersionPublicacionOficial {
	d := declaracionPublicacionPrueba()
	return ports.VersionPublicacionOficial{ConvocatoriaRef: d.ConvocatoriaRef, Secuencia: 1,
		HuellaVersionSHA256: d.HuellaVersionSHA256, UnidadRef: "unidad:rrhh", AmbitoRef: "ambito:rrhh"}
}

func TestPublicacionOficialDeclaraConDecisionLigadaAVersionYAmbito(t *testing.T) {
	q := solicitudPublicacionPrueba(t)
	repo := &repositorioPublicacionOficialPrueba{version: versionPublicacionPrueba()}
	a := &autorizadorComprobacionPrueba{t: t, audiencia: ports.AudienciaDeclararPublicacionOficialBases}
	s, err := NuevoServicioPublicacionOficialBases(a, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Declarar(context.Background(), q); err != nil || repo.declaraciones != 1 {
		t.Fatalf("error=%v declaraciones=%d", err, repo.declaraciones)
	}
	datos, _ := a.ultima.Datos()
	i := repo.intencion
	if datos.Accion != ports.AccionDeclararPublicacionOficialBases || datos.Finalidad != ports.FinalidadDeclararPublicacionOficialBases ||
		datos.Recurso.Tipo != ports.TipoRecursoVersionConvocatoria || datos.Recurso.Referencia != q.Declaracion.ConvocatoriaRef ||
		datos.Recurso.Ambitos["unidad_ref"] != "unidad:rrhh" || datos.Recurso.Ambitos["ambito_ref"] != "ambito:rrhh" ||
		datos.Recurso.Atributos["material_sha256"] != i.HuellaSHA256 {
		t.Fatalf("autorización desacoplada del recurso: %+v", datos.Recurso)
	}
	if !reflect.DeepEqual(i.Solicitud.Declaracion, q.Declaracion) ||
		i.Material.ResumenCapacidad().AudienciaConsumo() != ports.AudienciaDeclararPublicacionOficialBases {
		t.Fatal("intención o material desacoplados")
	}
}

func TestPublicacionOficialCierraSinEscribir(t *testing.T) {
	for _, caso := range []struct {
		nombre    string
		alterar   func(*repositorioPublicacionOficialPrueba, *autorizadorComprobacionPrueba)
		esperado  error
		autorizar bool
	}{
		{"pdp_deniega", func(_ *repositorioPublicacionOficialPrueba, a *autorizadorComprobacionPrueba) {
			a.err = vp.ErrDenegacionExplicitaAutorizacionLigadaV3
		}, vd.ErrAutorizacionDenegada, true},
		{"sin_material_del_nucleo", func(_ *repositorioPublicacionOficialPrueba, a *autorizadorComprobacionPrueba) {
			a.err = errors.New("clave de capacidad no publicada")
		}, ports.ErrPublicacionOficialNoDisponible, true},
		{"audiencia_ajena", func(_ *repositorioPublicacionOficialPrueba, a *autorizadorComprobacionPrueba) {
			a.audiencia = ports.AudienciaComprobacionOfrecimiento
		}, ports.ErrPublicacionOficialNoDisponible, true},
		{"version_cambiada", func(r *repositorioPublicacionOficialPrueba, _ *autorizadorComprobacionPrueba) {
			r.version.HuellaVersionSHA256 = strings.Repeat("d", 64)
		}, ports.ErrPublicacionOficialNoVigente, false},
		{"sin_ambito", func(r *repositorioPublicacionOficialPrueba, _ *autorizadorComprobacionPrueba) {
			r.version.AmbitoRef = ""
		}, vd.ErrAutorizacionDenegada, false},
		{"version_no_vigente", func(r *repositorioPublicacionOficialPrueba, _ *autorizadorComprobacionPrueba) {
			r.errConsulta = ports.ErrPublicacionOficialNoVigente
		}, ports.ErrPublicacionOficialNoVigente, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := &repositorioPublicacionOficialPrueba{version: versionPublicacionPrueba()}
			a := &autorizadorComprobacionPrueba{t: t, audiencia: ports.AudienciaDeclararPublicacionOficialBases}
			caso.alterar(repo, a)
			s, _ := NuevoServicioPublicacionOficialBases(a, repo)
			_, err := s.Declarar(context.Background(), solicitudPublicacionPrueba(t))
			if !errors.Is(err, caso.esperado) || repo.declaraciones != 0 {
				t.Fatalf("error=%v declaraciones=%d", err, repo.declaraciones)
			}
			if !caso.autorizar && a.ultima != (vd.SolicitudAutorizacionLigadaV3{}) {
				t.Fatal("se pidió una decisión para una versión que no se puede declarar")
			}
		})
	}
}

func TestPublicacionOficialValidaLaDeclaracion(t *testing.T) {
	texto := func(s string) *string { return &s }
	numero := func(n int64) *int64 { return &n }
	if err := ValidarDeclaracionPublicacionOficial(declaracionPublicacionPrueba()); err != nil {
		t.Fatal(err)
	}
	tablon := declaracionPublicacionPrueba()
	tablon.Medio, tablon.BoletinNumero, tablon.Direccion = "tablon_edictos_electronico", nil, texto("https://sede.ejemplo.es/tablon/1")
	if err := ValidarDeclaracionPublicacionOficial(tablon); err != nil {
		t.Fatal(err)
	}
	for nombre, alterar := range map[string]func(*ports.DeclaracionPublicacionOficial){
		"referencia":       func(d *ports.DeclaracionPublicacionOficial) { d.ConvocatoriaRef = "convocatoria:1" },
		"huella":           func(d *ports.DeclaracionPublicacionOficial) { d.HuellaVersionSHA256 = "x" },
		"medio":            func(d *ports.DeclaracionPublicacionOficial) { d.Medio = "periodico" },
		"boletin_ausente":  func(d *ports.DeclaracionPublicacionOficial) { d.BoletinNumero = nil },
		"boletin_tablon":   func(d *ports.DeclaracionPublicacionOficial) { d.Medio = "tablon_edictos_electronico" },
		"boletin_cero":     func(d *ports.DeclaracionPublicacionOficial) { d.BoletinNumero = numero(0) },
		"boletin_grande":   func(d *ports.DeclaracionPublicacionOficial) { d.BoletinNumero = numero(100000) },
		"anuncio":          func(d *ports.DeclaracionPublicacionOficial) { d.AnuncioRef = "<script>" },
		"fecha":            func(d *ports.DeclaracionPublicacionOficial) { d.FechaPublicacion = "2026-02-30" },
		"fecha_antigua":    func(d *ports.DeclaracionPublicacionOficial) { d.FechaPublicacion = "1999-12-31" },
		"direccion_http":   func(d *ports.DeclaracionPublicacionOficial) { d.Direccion = texto("http://sede.ejemplo.es/x") },
		"direccion_script": func(d *ports.DeclaracionPublicacionOficial) { d.Direccion = texto("javascript:alert(1)") },
		"sustituye":        func(d *ports.DeclaracionPublicacionOficial) { d.SustituyeRef = texto("publicacion:1") },
		"clave":            func(d *ports.DeclaracionPublicacionOficial) { d.ClaveIdempotencia = "corta" },
	} {
		d := declaracionPublicacionPrueba()
		alterar(&d)
		if !errors.Is(ValidarDeclaracionPublicacionOficial(d), ports.ErrPublicacionOficialInvalida) {
			t.Fatalf("%s admitido", nombre)
		}
	}
}
