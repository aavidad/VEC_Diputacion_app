package application

import (
	"slices"
	"testing"

	"vec-diputacion-granada/internal/modules/carrera/domain"
)

func TestPreparacionPoliticaNoPrestaReferenciaAprobacionNiCambiaOtrasVias(t *testing.T) {
	e, _ := escenarioAntecedentes()
	e.Casos = append(e.Casos, domain.Caso{Referencia: "otro-caso", Via: "progresion", PersonaNombre: "Otra persona sintética", Politica: domain.Politica{Referencia: "politica-laboral"}})
	c := domain.CatalogoPoliticaGrado{Alcance: domain.AlcanceSintetico, Politica: domain.Politica{Referencia: "catalogo-sintetico", Version: "v1", AprobacionReferencia: "acto-declarado"}}
	out, err := (Servicio{}).PrepararConPoliticaGradoSintetica(e, c)
	if err != nil {
		t.Fatal(err)
	}
	p := out.Preparacion.Casos[0]
	if p.EstadoGlobal != "pendiente" || p.Antecedentes.Politica.AprobacionReferencia != "" || out.Politica.Datos.Politica.AprobacionReferencia != "acto-declarado" || !slices.Contains(p.Pendientes, "carrera.pendiente.politica") || out.Preparacion.Casos[1].Antecedentes.Politica.Referencia != "politica-laboral" || e.Casos[0].Politica.Referencia != "" {
		t.Fatal("activa aprobación, cambia otra vía o modifica la entrada")
	}
}
