package internactproveedores

import (
	"context"
	"testing"
)

func TestOrganizacionHistoricaConstructorEmisorCompartidoConservaValidacion(t *testing.T) {
	p, f, _, e, m := escenarioOHPrueba(t)
	combinado, err := ConstruirOrganizacionHistoricaConEmisor(context.Background(), m, f, p.emisor, e.reloj)
	if err != nil || combinado.firmante != nil {
		t.Fatal("el emisor compartido exigió otro firmante", err)
	}
	material := consultaOHPrueba(t, f, f.organismo, f.unidad)
	if _, err := combinado.AutorizarConsultaOrganizacionHistorica(context.Background(), material); err != nil {
		t.Fatal("la decisión nominal no pasó por el emisor común", err)
	}
	f.unidad = "unidad:otra"
	if _, err := combinado.AutorizarConsultaOrganizacionHistorica(context.Background(), material); err == nil {
		t.Fatal("el ámbito alterado se admitió")
	}
}
