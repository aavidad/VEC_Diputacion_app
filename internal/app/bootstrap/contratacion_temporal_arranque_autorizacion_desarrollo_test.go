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
