package domain

import (
	"strings"
	"testing"
)

func catalogoPrueba() CatalogoAdmision {
	return CatalogoAdmision{Referencia: "catalogo:admision", Version: "1", PaqueteEjemplo: true, DudaRef: "dudas-139",
		PlazoSubsanacion: PlazoSubsanacion{Unidad: "dias_habiles", Cantidad: 10},
		Motivos:          []MotivoExclusion{{"tasa_no_justificada", true}, {"solicitud_fuera_de_plazo", false}}}
}

func antecedentePrueba(ref string) AntecedenteAdmision {
	return AntecedenteAdmision{EsquemaMaterial: EsquemaMaterialAdmisionLocal, PreparacionRef: ref, Revision: 1, HuellaMaterialSHA256: strings.Repeat("c", 64)}
}

var basesPrueba = BasesAdmision{Referencia: "bases:1", Version: "1", HuellaSHA256: strings.Repeat("a", 64)}

func TestListaProvisionalRepartePorDecisionYMotivos(t *testing.T) {
	decisiones := []DecisionAdmision{
		{Antecedente: antecedentePrueba("preparacion:c"), Decision: DecisionExcluida, Motivos: []string{"tasa_no_justificada", "solicitud_fuera_de_plazo"}},
		{Antecedente: antecedentePrueba("preparacion:a"), Decision: DecisionAdmitida},
		{Antecedente: antecedentePrueba("preparacion:b"), Decision: DecisionExcluida, Motivos: []string{"tasa_no_justificada"}},
	}
	l, err := ComponerListaProvisional("lista:1", 1, basesPrueba, catalogoPrueba(), decisiones)
	if err != nil {
		t.Fatal(err)
	}
	if l.Aprobada || l.Publicada || l.Persistida || l.Estado != "borrador_pendiente_aprobacion" || l.Vencimiento != "pendiente_publicacion" ||
		l.Resumen != (ResumenListaAdmision{Solicitudes: 3, Admitidas: 1, Excluidas: 2, ExcluidasSubsanables: 1}) || len(l.Pendientes) != 5 {
		t.Fatalf("frontera incorrecta: %+v", l)
	}
	if l.Excluidas[0].Antecedente.PreparacionRef != "preparacion:b" || !l.Excluidas[0].Subsanable || l.Excluidas[1].Subsanable ||
		l.Excluidas[1].Motivos[1] != (MotivoAplicado{"solicitud_fuera_de_plazo", false}) {
		t.Fatalf("exclusiones incorrectas: %+v", l.Excluidas)
	}
}

func TestListaProvisionalRechazaDecisionesIncoherentes(t *testing.T) {
	casos := map[string][]DecisionAdmision{
		"excluida_sin_motivo":  {{Antecedente: antecedentePrueba("p:1"), Decision: DecisionExcluida}},
		"admitida_con_motivo":  {{Antecedente: antecedentePrueba("p:1"), Decision: DecisionAdmitida, Motivos: []string{"tasa_no_justificada"}}},
		"motivo_desconocido":   {{Antecedente: antecedentePrueba("p:1"), Decision: DecisionExcluida, Motivos: []string{"otro"}}},
		"motivo_repetido":      {{Antecedente: antecedentePrueba("p:1"), Decision: DecisionExcluida, Motivos: []string{"tasa_no_justificada", "tasa_no_justificada"}}},
		"decision_desconocida": {{Antecedente: antecedentePrueba("p:1"), Decision: "pendiente"}},
		"solicitud_repetida":   {{Antecedente: antecedentePrueba("p:1"), Decision: DecisionAdmitida}, {Antecedente: antecedentePrueba("p:1"), Decision: DecisionAdmitida}},
		"vacia":                {},
	}
	for nombre, d := range casos {
		if _, err := ComponerListaProvisional("lista:1", 1, basesPrueba, catalogoPrueba(), d); err != ErrListaAdmision {
			t.Errorf("%s: %v", nombre, err)
		}
	}
}

func TestCatalogoAdmisionValidaPlazoYMotivos(t *testing.T) {
	if catalogoPrueba().Validar() != nil {
		t.Fatal("catálogo válido rechazado")
	}
	cambios := []func(*CatalogoAdmision){
		func(c *CatalogoAdmision) { c.PlazoSubsanacion.Unidad = "horas" },
		func(c *CatalogoAdmision) { c.PlazoSubsanacion.Cantidad = 0 },
		func(c *CatalogoAdmision) { c.PlazoSubsanacion.Cantidad = 251 },
		func(c *CatalogoAdmision) { c.Motivos = nil },
		func(c *CatalogoAdmision) { c.Motivos = append(c.Motivos, c.Motivos[0]) },
		func(c *CatalogoAdmision) { c.DudaRef = "" },
		func(c *CatalogoAdmision) { c.Version = "" },
	}
	for i, cambiar := range cambios {
		c := catalogoPrueba()
		cambiar(&c)
		if c.Validar() == nil {
			t.Errorf("caso %d aceptado", i)
		}
	}
}
