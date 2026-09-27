package domain

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestRegistrarReincorporacionTitularTrasCeseConservaHistoria(t *testing.T) {
	e, original := expedienteNombramientoPrueba(t)
	fecha := time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC)
	ref := "documento:ct:justificante:1"
	sha := "3259ad24878afcc3e4cf6ad860377c7d84f6b2b5152dc61146686a8e69ee895b"
	inc := instantePrueba(e)
	cese, err := e.RegistrarCese(e.Version, DatosCese{CausaClave: "fin_sustitucion", FechaEfecto: fecha,
		JustificanteTipo: "comunicacion_reincorporacion", JustificanteRef: ref, JustificanteSHA256: sha},
		DatosActuacion{AccionClave: AccionCesarNombramiento, ActorRef: "per_actor", UnidadRef: e.Asignacion.UnidadRef,
			ReciboRef: "recibo:cese:1", RealizadaEn: inc, FaseDestino: FaseNombramiento, EstadoDestino: EstadoEnCurso, DocumentosRef: []string{ref}})
	if err != nil {
		t.Fatal(err)
	}
	datos := DatosReincorporacionTitular{RelacionRef: "relacion:personal:1", FechaEfectiva: fecha, DocumentoRef: ref, DocumentoSHA256: sha}
	act := DatosActuacion{AccionClave: AccionRegistrarReincorporacionTitular, ActorRef: "per_actor", UnidadRef: cese.Asignacion.UnidadRef,
		ReciboRef: "recibo:reincorporacion:1", RealizadaEn: inc.Add(time.Minute), FaseDestino: FaseNombramiento,
		EstadoDestino: EstadoEnCurso, DocumentosRef: []string{ref}}
	if _, err = e.RegistrarReincorporacionTitular(e.Version, datos, act); !errors.Is(err, ErrReincorporacionTitularInvalida) {
		t.Fatalf("sin cese: %v", err)
	}
	siguiente, err := cese.RegistrarReincorporacionTitular(cese.Version, datos, act)
	if err != nil {
		t.Fatal(err)
	}
	if siguiente.Version != cese.Version+1 || siguiente.FaseActual != FaseNombramiento || siguiente.EstadoActual != EstadoEnCurso ||
		!reflect.DeepEqual(siguiente.Actuaciones[:len(cese.Actuaciones)], cese.Actuaciones) ||
		!reflect.DeepEqual(mapaJSON(t, e), original) {
		t.Fatal("historia previa o estado alterado")
	}
	if _, err = siguiente.RegistrarReincorporacionTitular(siguiente.Version, datos, act); !errors.Is(err, ErrReincorporacionTitularInvalida) {
		t.Fatalf("duplicado: %v", err)
	}
	if _, err = cese.RegistrarReincorporacionTitular(cese.Version-1, datos, act); !errors.Is(err, ErrVersionEnConflicto) {
		t.Fatalf("version: %v", err)
	}
	malo := act
	malo.DocumentosRef = []string{"documento:otro"}
	if _, err = cese.RegistrarReincorporacionTitular(cese.Version, datos, malo); !errors.Is(err, ErrReincorporacionTitularInvalida) {
		t.Fatalf("documento: %v", err)
	}
}
