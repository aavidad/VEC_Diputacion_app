package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"
)

type fuenteInscripcionPrueba struct {
	Referencia    string                          `json:"referencia"`
	Version       int                             `json:"version"`
	HuellaSHA256  string                          `json:"huella_sha256"`
	EntradasCanon string                          `json:"entradas_canon"`
	Entradas      []EntradaAccionAdministracionV1 `json:"entradas"`
}

func leerFuenteInscripcionPrueba(t *testing.T, partes ...string) fuenteInscripcionPrueba {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(append([]string{"..", "..", "..", "deploy", "principal"}, partes...)...))
	if err != nil {
		t.Fatal(err)
	}
	var f fuenteInscripcionPrueba
	if err := json.Unmarshal(b, &f); err != nil || len(f.Entradas) != 8 {
		t.Fatalf("fuente de inscripción ilegible: %v", err)
	}
	huella := sha256.Sum256([]byte(f.EntradasCanon))
	if hex.EncodeToString(huella[:]) != f.HuellaSHA256 {
		t.Fatal("canon de la fuente divergente")
	}
	for _, e := range f.Entradas {
		if e.FuenteHuellaSHA256 != f.HuellaSHA256 || e.FuenteRef != f.Referencia || e.FuenteVersion != f.Version || e.Validar() != nil {
			t.Fatalf("entrada ajena a su fuente: %s", e.Referencia)
		}
	}
	return f
}

// F1-101: la v2 cambia solo los campos de la lista y la ficha de
// convocatorias; las otras seis concesiones son las de la v1.
func TestFuenteInscripcionV2AmpliaSoloConvocatoriasYConservaLoDemas(t *testing.T) {
	v1 := leerFuenteInscripcionPrueba(t, "inscripcion_recorrido_20261013", "inscripcion_bolsa_actos_v1.json")
	v2 := leerFuenteInscripcionPrueba(t, "fuentes", "inscripcion_bolsa_actos_v2.json")
	if v2.Referencia != v1.Referencia || v2.Version != 2 {
		t.Fatal("la v2 debe ser la misma fuente con versión 2")
	}
	for j, e := range v2.Entradas {
		anterior := v1.Entradas[j]
		if e.Referencia != anterior.Referencia || e.Version != 2 || e.ClaseControl != anterior.ClaseControl ||
			!slices.Equal(e.DimensionesAmbito, anterior.DimensionesAmbito) {
			t.Fatalf("entrada %d divergente", j)
		}
		if e.VigenteHasta.Sub(e.VigenteDesde) < 60*24*time.Hour {
			t.Fatalf("vigencia de desarrollo menor de 60 días: %s", e.Referencia)
		}
		esperada := anterior.Concesion
		accion := esperada.Accion
		if accion == accionesInscripcionBolsaExterna[0] || accion == accionesInscripcionBolsaExterna[1] {
			esperada.CamposPermitidos = camposInscripcionBolsaExternaV2(accion)
			if slices.Equal(esperada.CamposPermitidos, anterior.Concesion.CamposPermitidos) {
				t.Fatalf("la v2 no amplía %s", accion)
			}
		}
		if !reflect.DeepEqual(e.Concesion, esperada) {
			t.Fatalf("concesión divergente: %s", accion)
		}
		if slices.Contains(accionesInscripcionBolsaExterna[:], accion) && !concesionDeltaBolsaExterna(e.Concesion, accion) {
			t.Fatalf("el dominio rechaza la concesión v2 de %s", accion)
		}
	}
}

func TestVersionarRolBolsaExternaInscripcionDesdeFuenteV2(t *testing.T) {
	v2 := leerFuenteInscripcionPrueba(t, "fuentes", "inscripcion_bolsa_actos_v2.json")
	c, base, i, _ := escenarioVersionarRolBolsaExterna(t)
	c.Entradas = v2.Entradas
	c.VigenteDesde, c.VigenteHasta = v2.Entradas[0].VigenteDesde, v2.Entradas[0].VigenteHasta
	i.Selecciones = nil
	for _, e := range v2.Entradas {
		if slices.Contains(accionesInscripcionBolsaExterna[:], e.Concesion.Accion) {
			i.Selecciones = append(i.Selecciones, SeleccionAccionAdministracionV1{EntradaRef: e.Referencia, EntradaVersion: e.Version})
		}
	}
	refrescarCatalogoExternoPorReferencia(t, &c, &i)
	ahora := c.VigenteDesde.Add(24 * time.Hour)
	plan, err := PrepararPlanVersionarRolBolsaExterna(c, base, i, ahora)
	if err != nil {
		t.Fatalf("plan de inscripción v2 rechazado: %v", err)
	}
	nueva := VersionRol{RolID: base.Rol.RolID, Version: plan.DefinicionNueva.Version, Concesiones: plan.DefinicionNueva.Concesiones}
	if estadoRolBolsaExterna(nueva) != "i" || len(nueva.Concesiones) != 12 {
		t.Fatalf("estado tras la v2: %q", estadoRolBolsaExterna(nueva))
	}
	if !slices.Equal(nueva.Concesiones[7].CamposPermitidos, camposInscripcionBolsaExternaV2(accionesInscripcionBolsaExterna[0])) ||
		!slices.Contains(nueva.Concesiones[8].CamposPermitidos, "modalidad_inscripcion") {
		t.Fatal("la versión nueva no lleva los campos v2")
	}

	// La lista de la v2 con la ficha de la v1 no forma un rol válido ni un plan.
	v1 := leerFuenteInscripcionPrueba(t, "inscripcion_recorrido_20261013", "inscripcion_bolsa_actos_v1.json")
	mezcla := plan
	mezcla.DefinicionNueva.Concesiones = slices.Clone(plan.DefinicionNueva.Concesiones)
	mezcla.DefinicionNueva.Concesiones[8] = v1.Entradas[1].Concesion
	if !concesionDeltaBolsaExterna(mezcla.DefinicionNueva.Concesiones[8], accionesInscripcionBolsaExterna[1]) {
		t.Fatal("la ficha de la v1 debe ser válida por sí sola")
	}
	if estadoRolBolsaExterna(VersionRol{RolID: base.Rol.RolID, Version: 2, Concesiones: mezcla.DefinicionNueva.Concesiones}) != "" {
		t.Fatal("rol con proyecciones v1 y v2 mezcladas aceptado")
	}
	if _, err := mezcla.HuellaSHA256(); err == nil {
		t.Fatal("plan con proyecciones v1 y v2 mezcladas aceptado")
	}

	// Una concesión con solo parte de las hojas nuevas no es ninguna de las dos proyecciones.
	parcial := v2.Entradas[0].Concesion
	parcial.CamposPermitidos = append(camposInscripcionBolsaExterna(parcial.Accion), "convocatorias[].modalidad_inscripcion")
	slices.Sort(parcial.CamposPermitidos)
	if concesionDeltaBolsaExterna(parcial, parcial.Accion) {
		t.Fatal("proyección parcial aceptada")
	}
	// Las hojas nuevas no se admiten en las solicitudes propias.
	propias := v2.Entradas[2].Concesion
	propias.CamposPermitidos = append(slices.Clone(propias.CamposPermitidos), "solicitudes[].modalidad_inscripcion")
	slices.Sort(propias.CamposPermitidos)
	if concesionDeltaBolsaExterna(propias, propias.Accion) {
		t.Fatal("hoja nueva aceptada fuera de convocatorias")
	}
}

func refrescarCatalogoExternoPorReferencia(t *testing.T, c *CatalogoAccionesAdministracionV1, i *IntencionVersionarRolBolsaExterna) {
	t.Helper()
	for j := range i.Selecciones {
		k := slices.IndexFunc(c.Entradas, func(e EntradaAccionAdministracionV1) bool { return e.Referencia == i.Selecciones[j].EntradaRef })
		if k < 0 {
			t.Fatalf("selección sin entrada: %s", i.Selecciones[j].EntradaRef)
		}
		h, err := c.Entradas[k].HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		i.Selecciones[j].EntradaHuellaSHA256 = h
	}
	h, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	i.CatalogoHuellaSHA256 = h
}
