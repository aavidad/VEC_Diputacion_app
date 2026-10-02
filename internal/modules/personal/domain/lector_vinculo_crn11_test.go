package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

func materialVinculoCRN11Prueba(t *testing.T) MaterialVinculoPropioCRN11 {
	t.Helper()
	empleado := "emp_" + strings.Repeat("b", 24)
	m, err := NuevoMaterialVinculoPropioCRN11(SolicitudVinculoPropioCRN11{Actor: actorFichaPropiaPrueba(t, vinculoEmpleadoPrueba("pep_", empleado)), EmpleadoRef: empleado})
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func TestMaterialVinculoCRN11CanonicoMinimoSinCorte(t *testing.T) {
	m := materialVinculoCRN11Prueba(t)
	z := strings.Repeat("a", 24)
	esperado := `{"esquema":"vec.personal.vinculo-propio-crn11.consulta.v1","empleado_ref":"` + m.EmpleadoRef() + `","actor_ref":"per_` + z + `","contexto_actor_ref":"vca_` + z + `","contexto_version":3,"cuenta_ref":"cta_` + z + `","cuenta_version":2,"perfil_ref":"prf_` + z + `","perfil_version":5,"persona_ref":"per_` + z + `","persona_version":4,"vinculo_ref":"pep_` + strings.Repeat("e", 24) + `","vinculo_version":1}`
	if string(m.Canonico()) != esperado {
		t.Fatalf("canon divergente: %s", m.Canonico())
	}
	suma := sha256.Sum256(m.Canonico())
	r := m.Recurso()
	if r.ModuloID != "personal" || r.Tipo != TipoRecursoVinculoPropioCRN11 || len(r.Ambitos) != 1 || len(r.Atributos) != 2 || r.Atributos["material_sha256"] != hex.EncodeToString(suma[:]) {
		t.Fatal("recurso divergente")
	}
	contexto := `{"ambitos":{"empleado_ref":"` + m.EmpleadoRef() + `"},"atributos":{"material_sha256":"` + hex.EncodeToString(suma[:]) + `","operacion":"vinculo_propio_historico_crn11"}}`
	h := sha256.Sum256([]byte(contexto))
	huella, err := m.HuellaSHA256()
	if err != nil || huella != hex.EncodeToString(h[:]) {
		t.Fatal("huella divergente del contrato SQL", err)
	}
	for _, p := range []string{"fecha", "corte", "fuente_ref", "vigente_en", "conocido_en"} {
		if bytes.Contains(m.Canonico(), []byte(p)) {
			t.Fatal("material incorpora historia sin fuente", p)
		}
	}
	b := m.Canonico()
	b[0] = 'x'
	r.Ambitos["empleado_ref"] = "ajeno"
	actor := m.Actor()
	actor.Instantanea.Vinculos[0].Referencia = "ajeno"
	if string(m.Canonico()) != esperado || m.Recurso().Ambitos["empleado_ref"] != m.EmpleadoRef() || m.Actor().Instantanea.Vinculos[0].Referencia != m.EmpleadoRef() {
		t.Fatal("material comparte estado mutable")
	}
}
func TestMaterialVinculoCRN11RechazaAjenoAmbiguoOLegado(t *testing.T) {
	emp := "emp_" + strings.Repeat("b", 24)
	otro := "emp_" + strings.Repeat("c", 24)
	segundo := vinculoEmpleadoPrueba("pep_", otro)
	segundo.VinculoRef = "pep_" + strings.Repeat("f", 24)
	actorAmbiguo := actorFichaPropiaPrueba(t, vinculoEmpleadoPrueba("pep_", emp))
	actorAmbiguo.Instantanea.Vinculos = append(actorAmbiguo.Instantanea.Vinculos, segundo)
	if _, err := NuevoMaterialVinculoPropioCRN11(SolicitudVinculoPropioCRN11{Actor: actorAmbiguo, EmpleadoRef: emp}); !errors.Is(err, ErrVinculoCRN11Invalido) {
		t.Fatal("contexto ambiguo admitido", err)
	}
	for nombre, vinculos := range map[string][]core.VinculoReferenciaContextoActor{
		"sin_empleado": nil, "legado": {vinculoEmpleadoPrueba("vin_", emp)}, "ajeno": {vinculoEmpleadoPrueba("pep_", otro)},
	} {
		t.Run(nombre, func(t *testing.T) {
			_, err := NuevoMaterialVinculoPropioCRN11(SolicitudVinculoPropioCRN11{Actor: actorFichaPropiaPrueba(t, vinculos...), EmpleadoRef: emp})
			if !errors.Is(err, ErrVinculoCRN11Denegado) {
				t.Fatal("titular no acreditado admitido", err)
			}
		})
	}
}
func TestVinculoCRN11ExigeParejaVersionYFuenteHistoricas(t *testing.T) {
	m := materialVinculoCRN11Prueba(t)
	base := VinculoHistoricoCRN11{PersonaRef: m.Actor().PersonaRef, EmpleadoRef: m.EmpleadoRef(), VinculoRef: m.VinculoRef(), FuenteRef: "prc_" + strings.Repeat("f", 24), Version: m.VinculoVersion()}
	if base.ValidarPara(m) != nil {
		t.Fatal("vinculo historico minimo rechazado")
	}
	for nombre, mutar := range map[string]func(*VinculoHistoricoCRN11){"persona": func(v *VinculoHistoricoCRN11) { v.PersonaRef = "per_" + strings.Repeat("x", 24) }, "empleado": func(v *VinculoHistoricoCRN11) { v.EmpleadoRef = "emp_" + strings.Repeat("x", 24) }, "vinculo": func(v *VinculoHistoricoCRN11) { v.VinculoRef = "pep_" + strings.Repeat("x", 24) }, "version": func(v *VinculoHistoricoCRN11) { v.Version++ }, "fuente": func(v *VinculoHistoricoCRN11) { v.FuenteRef = "" }} {
		t.Run(nombre, func(t *testing.T) {
			v := base
			mutar(&v)
			if v.ValidarPara(m) == nil {
				t.Fatal("historia cruzada admitida")
			}
		})
	}
	b, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]any
	if json.Unmarshal(b, &campos) != nil || len(campos) != 5 {
		t.Fatal("resultado excede minimo")
	}
	if !m.VigenteParaLectura(m.Actor().ResueltoEn) || m.VigenteParaLectura(m.Actor().ResueltoEn.Add(time.Hour)) || m.VigenteParaLectura(time.Time{}) {
		t.Fatal("vigencia de contexto incorrecta")
	}
}
