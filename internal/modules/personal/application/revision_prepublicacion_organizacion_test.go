package application

import (
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

func TestCompletitudPreparacionNoConfundeDeclaracionConPublicacion(t *testing.T) {
	p := paqueteRevisionPrueba()
	_, informe, comprobacion := ComprobarCompletitudPreparacionOrganizacion(p)
	if comprobacion.Completa || len(comprobacion.Faltantes) != 8 ||
		comprobacion.Faltantes[0] != (FaltanteCompletitudOrganizacion{"documento_ref", "presente", "ausente"}) ||
		comprobacion.Faltantes[7] != (FaltanteCompletitudOrganizacion{"decision_sin_decision", "vinculada", "sin_decision"}) {
		t.Fatalf("faltantes: %+v", comprobacion)
	}
	p.Manifiesto.DocumentoRef, p.Manifiesto.CustodiaRef = "documento:sintetico", "custodia:sintetica"
	p.Manifiesto.DiccionarioRef, p.Manifiesto.ActoRef = "diccionario:sintetico", "acto:sintetico"
	p.Manifiesto.AprobadaEn, p.Manifiesto.PublicadaEn, p.Manifiesto.EfectosDesde = "2026-01-01", "2026-01-02", "2026-01-03"
	p.Decisiones = []domain.DecisionConciliacionOrganizacion{{FilaFuenteRef: "fila:1", Clase: "unidad", Resultado: "vinculada", DestinoRef: "unidad:sintetica", Motivo: "Correspondencia declarada", EvidenciaRef: "evidencia:sintetica"}}
	_, informe, comprobacion = ComprobarCompletitudPreparacionOrganizacion(p)
	if !comprobacion.Completa || len(comprobacion.Faltantes) != 0 || informe.Estado != "preparacion_no_autoritativa" || len(informe.PendientesPublicacion) < 5 {
		t.Fatalf("el preparado no se debe elevar a publicación: %+v %+v", comprobacion, informe)
	}
	p.Decisiones[0].Resultado, p.Decisiones[0].DestinoRef = "pendiente", ""
	_, informe, comprobacion = ComprobarCompletitudPreparacionOrganizacion(p)
	if !reflect.DeepEqual(comprobacion.Faltantes, []FaltanteCompletitudOrganizacion{{"decision_pendiente", "vinculada", "pendiente"}}) {
		t.Fatalf("decisión pendiente: %+v", comprobacion)
	}
	p.Decisiones[0].Resultado, p.Decisiones[0].DestinoRef = "vinculada", "unidad:sintetica"
	p.Decisiones = append(p.Decisiones, domain.DecisionConciliacionOrganizacion{
		FilaFuenteRef: "fila:1", Clase: "clasificacion", Resultado: "pendiente",
		Motivo: "Clasificación sin revisar", EvidenciaRef: "evidencia:sintetica",
	})
	_, informe, comprobacion = ComprobarCompletitudPreparacionOrganizacion(p)
	if !informe.Valido || comprobacion.Completa || !reflect.DeepEqual(comprobacion.Faltantes, []FaltanteCompletitudOrganizacion{{"decision_pendiente", "vinculada", "pendiente"}}) {
		t.Fatalf("clasificación adicional pendiente: %+v", comprobacion)
	}
}
