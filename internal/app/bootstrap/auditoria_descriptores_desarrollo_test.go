package bootstrap

import (
	"net/http"
	"testing"

	"vec-diputacion-granada/internal/vec/auditoria"
)

func TestDescriptoresAuditoriaRRHHSeparanOpcionesYFuenteNominal(t *testing.T) {
	d, err := descriptoresFronterasAuditoriaRRHHDesarrollo("prf_ct_consultor", "prf_bolsa_consultor")
	if err != nil {
		t.Fatal(err)
	}
	c, err := nuevoCatalogoFronterasComunDesarrollo(d)
	if err != nil {
		t.Fatal(err)
	}
	get, okGet := c.resolver(http.MethodGet, auditoria.RutaOpciones)
	post, okPost := c.resolver(http.MethodPost, auditoria.RutaConsulta)
	if !okGet || !okPost || get.ClaveCapacidad == post.ClaveCapacidad ||
		!get.admitePerfil("prf_ct_consultor") || get.admitePerfil("prf_bolsa_consultor") ||
		!post.admitePerfil("prf_ct_consultor") || !post.admitePerfil("prf_bolsa_consultor") {
		t.Fatalf("fronteras de auditoría mezcladas: GET=%+v POST=%+v", get, post)
	}
	if _, err := descriptoresFronterasAuditoriaRRHHDesarrollo("prf_igual", "prf_igual"); err == nil {
		t.Fatal("un solo perfil admitido para dos fuentes")
	}
}
