package domain

import (
	"reflect"
	"testing"
)

func revisionPrueba(t *testing.T, extra ...DecisionAdmision) (ListaAdmisionProvisional, ListaAdmisionProvisional) {
	t.Helper()
	decisiones := []DecisionAdmision{
		{Antecedente: antecedentePrueba("p:a"), Decision: DecisionAdmitida},
		{Antecedente: antecedentePrueba("p:b"), Decision: DecisionExcluida, Motivos: []string{"tasa_no_justificada"}},
	}
	anterior, err := ComponerListaProvisional("lista:1", 1, basesPrueba, catalogoPrueba(), decisiones)
	if err != nil {
		t.Fatal(err)
	}
	nueva, err := ComponerListaProvisional("lista:1", 2, basesPrueba, catalogoPrueba(), append(decisiones, extra...))
	if err != nil {
		t.Fatal(err)
	}
	return anterior, nueva
}

func TestRevisionConservaLaAnteriorYDevuelveLasIncorporadas(t *testing.T) {
	anterior, nueva := revisionPrueba(t, DecisionAdmision{Antecedente: antecedentePrueba("p:d"), Decision: DecisionAdmitida},
		DecisionAdmision{Antecedente: antecedentePrueba("p:c"), Decision: DecisionExcluida, Motivos: []string{"solicitud_fuera_de_plazo"}})
	incorporadas, err := ComprobarRevisionProvisional(anterior, nueva)
	if err != nil || !reflect.DeepEqual(incorporadas, []string{"p:c", "p:d"}) {
		t.Fatalf("%v %v", incorporadas, err)
	}
}

func TestRevisionAceptaMotivosReordenados(t *testing.T) {
	catalogo := catalogoPrueba()
	dos := []string{"tasa_no_justificada", "solicitud_fuera_de_plazo"}
	anterior, err := ComponerListaProvisional("lista:1", 1, basesPrueba, catalogo, []DecisionAdmision{{Antecedente: antecedentePrueba("p:b"), Decision: DecisionExcluida, Motivos: dos}})
	if err != nil {
		t.Fatal(err)
	}
	nueva, err := ComponerListaProvisional("lista:1", 2, basesPrueba, catalogo, []DecisionAdmision{
		{Antecedente: antecedentePrueba("p:b"), Decision: DecisionExcluida, Motivos: []string{dos[1], dos[0]}},
		{Antecedente: antecedentePrueba("p:c"), Decision: DecisionAdmitida}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ComprobarRevisionProvisional(anterior, nueva); err != nil {
		t.Fatal(err)
	}
}

func TestRevisionRechazaCambiosSobreLaAnterior(t *testing.T) {
	extra := DecisionAdmision{Antecedente: antecedentePrueba("p:c"), Decision: DecisionAdmitida}
	cambios := map[string]func(a, n *ListaAdmisionProvisional){
		"sin_incorporadas": func(a, n *ListaAdmisionProvisional) { *n = *a; n.Revision = 2 },
		"salta_revision":   func(a, n *ListaAdmisionProvisional) { n.Revision = 3 },
		"otra_lista":       func(a, n *ListaAdmisionProvisional) { n.ListaRef = "lista:2" },
		"otras_bases":      func(a, n *ListaAdmisionProvisional) { n.Bases.Version = "2" },
		"otro_catalogo":    func(a, n *ListaAdmisionProvisional) { n.Catalogo.Version = "2" },
		"otro_plazo":       func(a, n *ListaAdmisionProvisional) { n.PlazoSubsanacion.Cantidad = 5 },
		"motivo_cambiado": func(a, n *ListaAdmisionProvisional) {
			n.Excluidas[0].Motivos = []MotivoAplicado{{"solicitud_fuera_de_plazo", false}}
			n.Excluidas[0].Subsanable = false
			n.Resumen.ExcluidasSubsanables = 0
		},
		"admitida_pasa_excluir": func(a, n *ListaAdmisionProvisional) {
			n.Excluidas = append(n.Excluidas, SolicitudExcluida{n.Admitidas[0].Antecedente, []MotivoAplicado{{"tasa_no_justificada", true}}, true})
			n.Admitidas = n.Admitidas[1:]
			n.Resumen = ResumenListaAdmision{Solicitudes: 3, Admitidas: len(n.Admitidas), Excluidas: len(n.Excluidas), ExcluidasSubsanables: 2}
		},
		"falta_una_anterior": func(a, n *ListaAdmisionProvisional) {
			a.Admitidas = append(a.Admitidas, SolicitudAdmitida{antecedentePrueba("p:z")})
			a.Resumen.Admitidas++
			a.Resumen.Solicitudes++
		},
		"anterior_aprobada":    func(a, n *ListaAdmisionProvisional) { a.Aprobada = true },
		"anterior_incoherente": func(a, n *ListaAdmisionProvisional) { a.Resumen.Solicitudes = 9 },
		"huella_s4_distinta":   func(a, n *ListaAdmisionProvisional) { a.Admitidas[0].Antecedente.Revision = 2 },
	}
	for nombre, cambiar := range cambios {
		anterior, nueva := revisionPrueba(t, extra)
		cambiar(&anterior, &nueva)
		if _, err := ComprobarRevisionProvisional(anterior, nueva); err != ErrRevisionLista {
			t.Errorf("%s: %v", nombre, err)
		}
	}
}
