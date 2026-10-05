package domain

import (
	"strings"
	"testing"
)

func provisionalPrueba(t *testing.T) (ListaAdmisionProvisional, AntecedenteLista) {
	t.Helper()
	catalogo := catalogoPrueba()
	catalogo.Motivos = append(catalogo.Motivos, MotivoExclusion{"documento_identidad_no_aportado", true})
	l, err := ComponerListaProvisional("lista:provisional", 1, basesPrueba, catalogo, []DecisionAdmision{
		{Antecedente: antecedentePrueba("p:admitida"), Decision: DecisionAdmitida},
		{Antecedente: antecedentePrueba("p:dos_motivos"), Decision: DecisionExcluida, Motivos: []string{"tasa_no_justificada", "documento_identidad_no_aportado"}},
		{Antecedente: antecedentePrueba("p:fuera_plazo"), Decision: DecisionExcluida, Motivos: []string{"solicitud_fuera_de_plazo"}},
		{Antecedente: antecedentePrueba("p:sin_escrito"), Decision: DecisionExcluida, Motivos: []string{"tasa_no_justificada"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return l, AntecedenteLista{EsquemaAntecedenteLista, l.ListaRef, l.Revision, strings.Repeat("d", 64)}
}

func resolucionesPrueba() []ResolucionSubsanacion {
	return []ResolucionSubsanacion{
		{Antecedente: antecedentePrueba("p:dos_motivos"), Resultado: ResolucionDesestimada, Via: ViaSubsanacion, MotivosPersistentes: []string{"documento_identidad_no_aportado"}},
		{Antecedente: antecedentePrueba("p:fuera_plazo"), Resultado: ResolucionEstimada, Via: ViaReclamacion},
		{Antecedente: antecedentePrueba("p:sin_escrito"), Resultado: ResolucionNoPresentada},
	}
}

func TestDefinitivaAplicaCadaResolucionSobreLaProvisional(t *testing.T) {
	p, a := provisionalPrueba(t)
	d, err := ComponerListaDefinitiva("lista:definitiva", 1, p, a, resolucionesPrueba())
	if err != nil {
		t.Fatal(err)
	}
	if d.Aprobada || d.Publicada || d.Persistida || d.Estado != "borrador_pendiente_aprobacion" || d.Provisional != a ||
		d.Resumen != (ResumenListaDefinitiva{Solicitudes: 4, Admitidas: 2, AdmitidasTrasEscrito: 1, Excluidas: 2, ExcluidasSinEscrito: 1}) {
		t.Fatalf("frontera incorrecta: %+v", d)
	}
	if d.Admitidas[0].Antecedente.PreparacionRef != "p:admitida" || d.Admitidas[0].Origen != OrigenProvisional ||
		d.Admitidas[1].Antecedente.PreparacionRef != "p:fuera_plazo" || d.Admitidas[1].Origen != ViaReclamacion {
		t.Fatalf("admitidas: %+v", d.Admitidas)
	}
	if len(d.Excluidas[0].Motivos) != 1 || d.Excluidas[0].Motivos[0].Codigo != "documento_identidad_no_aportado" ||
		d.Excluidas[1].Resolucion != ResolucionNoPresentada || d.Excluidas[1].Motivos[0].Codigo != "tasa_no_justificada" {
		t.Fatalf("excluidas: %+v", d.Excluidas)
	}
	d.Excluidas[1].Motivos[0].Codigo = "otro"
	if p.Excluidas[2].Motivos[0].Codigo != "tasa_no_justificada" {
		t.Fatal("la definitiva comparte memoria con la provisional")
	}
}

func TestDefinitivaRechazaResolucionesIncoherentes(t *testing.T) {
	cambios := map[string]func([]ResolucionSubsanacion) []ResolucionSubsanacion{
		"falta_una": func(r []ResolucionSubsanacion) []ResolucionSubsanacion { return r[:2] },
		"repetida":  func(r []ResolucionSubsanacion) []ResolucionSubsanacion { r[2] = r[0]; return r },
		"admitida_en_provisional": func(r []ResolucionSubsanacion) []ResolucionSubsanacion {
			r[2].Antecedente = antecedentePrueba("p:admitida")
			return r
		},
		"huella_distinta":        func(r []ResolucionSubsanacion) []ResolucionSubsanacion { r[0].Antecedente.Revision = 2; return r },
		"subsanar_no_subsanable": func(r []ResolucionSubsanacion) []ResolucionSubsanacion { r[1].Via = ViaSubsanacion; return r },
		"estimar_sin_via":        func(r []ResolucionSubsanacion) []ResolucionSubsanacion { r[1].Via = ""; return r },
		"estimar_con_motivos": func(r []ResolucionSubsanacion) []ResolucionSubsanacion {
			r[1].MotivosPersistentes = []string{"solicitud_fuera_de_plazo"}
			return r
		},
		"desestimar_sin_motivos": func(r []ResolucionSubsanacion) []ResolucionSubsanacion { r[0].MotivosPersistentes = nil; return r },
		"motivo_nuevo": func(r []ResolucionSubsanacion) []ResolucionSubsanacion {
			r[0].MotivosPersistentes = []string{"solicitud_fuera_de_plazo"}
			return r
		},
		"motivo_repetido": func(r []ResolucionSubsanacion) []ResolucionSubsanacion {
			r[0].MotivosPersistentes = []string{"documento_identidad_no_aportado", "documento_identidad_no_aportado"}
			return r
		},
		"no_presentada_con_via":      func(r []ResolucionSubsanacion) []ResolucionSubsanacion { r[2].Via = ViaSubsanacion; return r },
		"resultado_desconocido":      func(r []ResolucionSubsanacion) []ResolucionSubsanacion { r[2].Resultado = "pendiente"; return r },
		"desestimar_via_desconocida": func(r []ResolucionSubsanacion) []ResolucionSubsanacion { r[0].Via = "recurso"; return r },
	}
	for nombre, cambiar := range cambios {
		p, a := provisionalPrueba(t)
		if _, err := ComponerListaDefinitiva("lista:definitiva", 1, p, a, cambiar(resolucionesPrueba())); err != ErrListaDefinitiva {
			t.Errorf("%s: %v", nombre, err)
		}
	}
	p, a := provisionalPrueba(t)
	otra := a
	otra.Revision = 2
	if _, err := ComponerListaDefinitiva("lista:definitiva", 1, p, otra, resolucionesPrueba()); err != ErrListaDefinitiva {
		t.Error("antecedente de otra revisión aceptado")
	}
	if _, err := ComponerListaDefinitiva(p.ListaRef, 1, p, a, resolucionesPrueba()); err != ErrListaDefinitiva {
		t.Error("la definitiva reutiliza la referencia de la provisional")
	}
}
