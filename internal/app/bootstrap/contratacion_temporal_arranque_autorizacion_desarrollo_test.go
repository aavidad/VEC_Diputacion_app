package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type fuenteArranqueAutorizacionCTPrueba struct {
	instantanea dominiovec.InstantaneaAutorizacion
	err         error
	lecturas    int
}

func (f *fuenteArranqueAutorizacionCTPrueba) ObtenerInstantaneaAutorizacion(
	_ context.Context, _, _ string,
) (dominiovec.InstantaneaAutorizacion, error) {
	f.lecturas++
	return f.instantanea, f.err
}

type publicadorArranqueAutorizacionCTPrueba struct {
	publicaciones int
	err           error
}

type fuenteDosPerfilesArranqueCTPrueba struct {
	porPerfil map[string]dominiovec.InstantaneaAutorizacion
	errores   map[string]error
	lecturas  []string
}

func (f *fuenteDosPerfilesArranqueCTPrueba) ObtenerInstantaneaAutorizacion(
	_ context.Context, principal, perfil string,
) (dominiovec.InstantaneaAutorizacion, error) {
	f.lecturas = append(f.lecturas, perfil)
	if err := f.errores[perfil]; err != nil {
		return dominiovec.InstantaneaAutorizacion{}, err
	}
	i, ok := f.porPerfil[perfil]
	if !ok || i.AsignacionPerfil.PrincipalID != principal {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrAsignacionPerfilNoEncontrada
	}
	return i, nil
}

func (p *publicadorArranqueAutorizacionCTPrueba) PublicarInicial(
	_ context.Context, i dominiovec.InstantaneaAutorizacion,
) (dominiovec.InstantaneaAutorizacion, error) {
	p.publicaciones++
	return i, p.err
}

func TestArranqueAutorizacionCTConsumePublicadaSinRevivirla(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	semilla := soporte.instantanea
	ahora := semilla.AsignacionPerfil.VigenteDesde.Add(time.Minute)
	casos := []struct {
		nombre  string
		mutar   func(*dominiovec.InstantaneaAutorizacion)
		permite bool
	}{
		{"vigente_anterior", func(i *dominiovec.InstantaneaAutorizacion) {
			i.AsignacionPerfil.Version = 2
		}, true},
		{"revocada", func(i *dominiovec.InstantaneaAutorizacion) {
			i.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
			i.AsignacionPerfil.Version = 2
			i.AsignacionPerfil.RevocadaPor = "autoridad:prueba"
			i.AsignacionPerfil.RevocadaEn = ahora
			i.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
		}, false},
		{"control_retirado", func(i *dominiovec.InstantaneaAutorizacion) {
			i.ControlVigenciaVersionRol.Estado = dominiovec.EstadoControlVigenciaVersionRolRetirada
			i.ControlVigenciaVersionRol.ActoRef = "acto:prueba"
			i.ControlVigenciaVersionRol.MotivoCodigo = "motivo:prueba"
		}, false},
		{"ambito_restringido", func(i *dominiovec.InstantaneaAutorizacion) {
			i.AsignacionPerfil.Ambitos[0].Valores = []string{"organizacion:otra"}
		}, false},
		{"identidad_incompatible", func(i *dominiovec.InstantaneaAutorizacion) {
			i.AsignacionPerfil.AsignacionID = "asignacion:otra"
		}, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			actual := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
			caso.mutar(&actual)
			fuente := &fuenteArranqueAutorizacionCTPrueba{instantanea: actual}
			publicador := &publicadorArranqueAutorizacionCTPrueba{}
			for reinicio := 0; reinicio < 2; reinicio++ {
				obtenida, err := asegurarAutorizacionInicialContratacionTemporalDesarrollo(
					context.Background(), fuente, publicador, semilla, ahora)
				if caso.permite && (err != nil || !reflect.DeepEqual(obtenida, actual)) {
					t.Fatalf("no consumio publicada en reinicio %d: %v", reinicio, err)
				}
				if !caso.permite && !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
					t.Fatalf("acepto autoridad no vigente en reinicio %d: %v", reinicio, err)
				}
			}
			if publicador.publicaciones != 0 || fuente.lecturas != 2 {
				t.Fatalf("publicaciones=%d lecturas=%d", publicador.publicaciones, fuente.lecturas)
			}
		})
	}
}

func TestArranqueAutorizacionCTSoloInicialAnteAusenciaInequivoca(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	semilla := soporte.instantanea
	ahora := semilla.AsignacionPerfil.VigenteDesde.Add(time.Minute)
	for _, caso := range []struct {
		nombre    string
		errFuente error
		errCAS    error
		publica   bool
		acierta   bool
	}{
		{"ausente", puertosvec.ErrAsignacionPerfilNoEncontrada, nil, true, true},
		{"fuente_caida", puertosvec.ErrFuenteAutorizacionNoDisponible, nil, false, false},
		{"lectura_ambigua", errors.New("lectura ambigua"), nil, false, false},
		{"carrera_con_revocacion", puertosvec.ErrAsignacionPerfilNoEncontrada,
			errors.New("CAS soloInicial rechazado"), true, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente := &fuenteArranqueAutorizacionCTPrueba{err: caso.errFuente}
			publicador := &publicadorArranqueAutorizacionCTPrueba{err: caso.errCAS}
			_, err := asegurarAutorizacionInicialContratacionTemporalDesarrollo(
				context.Background(), fuente, publicador, semilla, ahora)
			if (err == nil) != caso.acierta || (publicador.publicaciones == 1) != caso.publica {
				t.Fatalf("resultado=%v publicaciones=%d", err, publicador.publicaciones)
			}
		})
	}
}

func TestArranqueCTTresPerfilesNoReviveRevocadosNiFuerzaLegado(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	legado := soporte.instantanea
	alta, err := nuevaInstantaneaAutorizacionAltaFijaContratacionTemporalDesarrollo(
		legado.AsignacionPerfil.PrincipalID,
		"prf_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		legado.AsignacionPerfil.VigenteDesde,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	cobertura, err := nuevaInstantaneaAutorizacionCoberturaContratacionTemporalDesarrollo(
		legado.AsignacionPerfil.PrincipalID,
		"prf_cccccccccccccccccccccccccccccccc",
		legado.AsignacionPerfil.VigenteDesde,
	)
	if err != nil || legado.AsignacionPerfil.PerfilActivoRef == alta.AsignacionPerfil.PerfilActivoRef ||
		alta.AsignacionPerfil.PerfilActivoRef == cobertura.AsignacionPerfil.PerfilActivoRef {
		t.Fatalf("perfiles de ensayo no separados: %v", err)
	}
	ahora := legado.AsignacionPerfil.VigenteDesde.Add(time.Minute)
	perfilLegado := legado.AsignacionPerfil.PerfilActivoRef
	perfilAlta, perfilCobertura := alta.AsignacionPerfil.PerfilActivoRef, cobertura.AsignacionPerfil.PerfilActivoRef
	casos := []struct {
		nombre           string
		mutar            func(*fuenteDosPerfilesArranqueCTPrueba)
		acierta          bool
		legadoDisponible bool
	}{
		{"tres_vigentes", nil, true, true},
		{"cobertura_ausente", func(f *fuenteDosPerfilesArranqueCTPrueba) { delete(f.porPerfil, perfilCobertura) }, false, false},
		{"cobertura_restringida", func(f *fuenteDosPerfilesArranqueCTPrueba) {
			i := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(f.porPerfil[perfilCobertura])
			i.AsignacionPerfil.Ambitos[0].Valores = []string{"organizacion:restringida"}
			f.porPerfil[perfilCobertura] = i
		}, false, false},
		{"cobertura_revocada", func(f *fuenteDosPerfilesArranqueCTPrueba) {
			i := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(f.porPerfil[perfilCobertura])
			i.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
			i.AsignacionPerfil.RevocadaPor = "autoridad:prueba"
			i.AsignacionPerfil.RevocadaEn = ahora
			i.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
			f.porPerfil[perfilCobertura] = i
		}, false, false},
		{"alta_revocada", func(f *fuenteDosPerfilesArranqueCTPrueba) {
			i := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(f.porPerfil[perfilAlta])
			i.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
			i.AsignacionPerfil.RevocadaPor = "autoridad:prueba"
			i.AsignacionPerfil.RevocadaEn = ahora
			i.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
			f.porPerfil[perfilAlta] = i
		}, false, false},
		{"fuente_ilegible", func(f *fuenteDosPerfilesArranqueCTPrueba) {
			f.errores[perfilCobertura] = puertosvec.ErrFuenteAutorizacionNoDisponible
		}, false, false},
		{"alta_ausente", func(f *fuenteDosPerfilesArranqueCTPrueba) { delete(f.porPerfil, perfilAlta) }, false, false},
		{"legado_ausente", func(f *fuenteDosPerfilesArranqueCTPrueba) { delete(f.porPerfil, perfilLegado) }, true, false},
		{"legado_revocado", func(f *fuenteDosPerfilesArranqueCTPrueba) {
			i := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(f.porPerfil[perfilLegado])
			i.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
			i.AsignacionPerfil.RevocadaPor = "autoridad:prueba"
			i.AsignacionPerfil.RevocadaEn = ahora
			i.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
			f.porPerfil[perfilLegado] = i
		}, true, false},
		{"legado_ilegible", func(f *fuenteDosPerfilesArranqueCTPrueba) {
			f.errores[perfilLegado] = puertosvec.ErrFuenteAutorizacionNoDisponible
		}, true, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente := &fuenteDosPerfilesArranqueCTPrueba{
				porPerfil: map[string]dominiovec.InstantaneaAutorizacion{
					perfilLegado: legado, perfilAlta: alta, perfilCobertura: cobertura,
				},
				errores: make(map[string]error),
			}
			if caso.mutar != nil {
				caso.mutar(fuente)
			}
			obtenidoLegado, obtenidaAlta, obtenidaCobertura, disponible, err :=
				consumirAutoridadTresPerfilesContratacionTemporalDesarrollo(
					context.Background(), fuente, legado, alta, cobertura, ahora)
			if (err == nil) != caso.acierta || disponible != caso.legadoDisponible {
				t.Fatalf("resultado=%v legado=%t", err, disponible)
			}
			if caso.acierta && (obtenidaAlta.Validar() != nil || obtenidaCobertura.Validar() != nil ||
				obtenidaAlta.AsignacionPerfil.PerfilActivoRef != perfilAlta ||
				obtenidaCobertura.AsignacionPerfil.PerfilActivoRef != perfilCobertura) {
				t.Fatal("se perdieron los dos perfiles fijos publicados")
			}
			if caso.acierta && (obtenidoLegado.Validar() == nil) != disponible {
				t.Fatal("legado no cerró según su lectura")
			}
			if len(fuente.lecturas) == 0 || fuente.lecturas[0] != perfilAlta {
				t.Fatalf("perfiles fijos no se comprobaron antes de legado: %v", fuente.lecturas)
			}
		})
	}
}
