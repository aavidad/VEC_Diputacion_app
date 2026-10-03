package application_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	adaptador "vec-diputacion-granada/internal/modules/seleccion/adapters/bolsa"
	seleccion "vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

func TestSolicitudBasesConservaPreimagenYReferenciasPropuestas(t *testing.T) {
	ref := bolsa.ReferenciaConfiguracionConvocatoria{ID: "baremo:propuesto", Version: 7, HuellaContenidoSHA256: strings.Repeat("a", 64)}
	m := ports.MaterialBasesPropuesto{Alcance: "preparacion_sintetica", IdentidadMaterial: "material:local", VersionMaterial: 99,
		Contenido: contenidoPreparacionValidoPrueba(), Referencias: ports.ReferenciasPreparacionBases{ReglasBaremacion: ref}}
	esperada := prep.Esperada{PreparacionRef: "preparacion:anterior", Revision: 3, HuellaMaterialSHA256: strings.Repeat("b", 64)}
	s, err := seleccion.PrepararSolicitudGuardadoBases(context.Background(), m, esperada, "operacion:original", adaptador.CanonizadorBases{})
	if err != nil || s.Esperada != esperada || s.ClaveOperacion != "operacion:original" {
		t.Fatalf("preimagen alterada: %+v / %v", s, err)
	}
	refs := map[string]bolsa.ReferenciaConfiguracionConvocatoria{}
	for _, r := range s.Material.Referencias {
		refs[r.Campo] = r.Referencia
	}
	if len(refs) != 10 || refs["reglas_baremacion"] != ref || refs["plaza"] != (bolsa.ReferenciaConfiguracionConvocatoria{}) {
		t.Fatalf("referencias sustituidas: %+v", refs)
	}
	pendientes, err := s.Material.Pendientes()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pendientes {
		if p.Campo == "reglas_baremacion" && p.Codigo != "referencia_no_verificada" {
			t.Fatalf("baremo promovido: %+v", p)
		}
	}
	s.Material.Contenido.Categorias[0] = "modificada"
	if m.Contenido.Categorias[0] == "modificada" {
		t.Fatal("material compartido")
	}
	// Una revisión local distinta conserva la misma intención durable. Detectar
	// una preimagen obsoleta corresponde al CAS del servidor, no a esta traducción.
	m.VersionMaterial = 1
	repetida, err := seleccion.PrepararSolicitudGuardadoBases(context.Background(), m, esperada, s.ClaveOperacion, adaptador.CanonizadorBases{})
	if err != nil || repetida.Esperada != esperada || repetida.Material.Referencias[0] != s.Material.Referencias[0] {
		t.Fatal("revision local usada como version durable", err)
	}
}

func TestSolicitudBasesRechazaPreimagenIncompletaSinEmitirMaterial(t *testing.T) {
	m := ports.MaterialBasesPropuesto{Alcance: "preparacion_sintetica", IdentidadMaterial: "material:local", VersionMaterial: 1}
	for _, e := range []prep.Esperada{
		{}, {PreparacionRef: "prep:1", Revision: 1},
		{PreparacionRef: "prep:1", Revision: 0, HuellaMaterialSHA256: strings.Repeat("a", 64)},
		{PreparacionRef: "prep:1", Revision: -1},
		{PreparacionRef: "prep:1", Revision: 1_000_000, HuellaMaterialSHA256: strings.Repeat("a", 64)},
	} {
		s, err := seleccion.PrepararSolicitudGuardadoBases(context.Background(), m, e, "clave:1", adaptador.CanonizadorBases{})
		if !errors.Is(err, ports.ErrMaterialBasesInvalido) || !reflect.DeepEqual(s, seleccion.SolicitudGuardadoBases{}) {
			t.Fatalf("salida parcial: %+v / %v", s, err)
		}
	}
	for _, clave := range []string{"", "con espacios", strings.Repeat("a", 129)} {
		s, err := seleccion.PrepararSolicitudGuardadoBases(context.Background(), m, prep.Esperada{PreparacionRef: "prep:1"}, clave, adaptador.CanonizadorBases{})
		if err == nil || !reflect.DeepEqual(s, seleccion.SolicitudGuardadoBases{}) {
			t.Fatalf("clave invalida emite material: %+v / %v", s, err)
		}
	}
}

func TestSolicitudBasesCanceladaOMaterialNoAlmacenableNoEmitePeticion(t *testing.T) {
	m := ports.MaterialBasesPropuesto{Alcance: "preparacion_sintetica", IdentidadMaterial: "material:local", VersionMaterial: 1}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	s, err := seleccion.PrepararSolicitudGuardadoBases(ctx, m, prep.Esperada{PreparacionRef: "prep:1"}, "clave:1", adaptador.CanonizadorBases{})
	if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(s, seleccion.SolicitudGuardadoBases{}) {
		t.Fatal("cancelacion emite material", err)
	}
	m.Contenido.Titulo = strings.Repeat("a", 12001)
	s, err = seleccion.PrepararSolicitudGuardadoBases(context.Background(), m, prep.Esperada{PreparacionRef: "prep:1"}, "clave:1", adaptador.CanonizadorBases{})
	if !errors.Is(err, ports.ErrMaterialBasesInvalido) || !reflect.DeepEqual(s, seleccion.SolicitudGuardadoBases{}) {
		t.Fatal("material no almacenable emitido", err)
	}
}
