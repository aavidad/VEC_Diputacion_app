package mibolsa

import (
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestHistorialContratoPrevistoNoSeConfundeConHechoFuturo(t *testing.T) {
	consulta := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	inicioPrevisto := consulta.Add(48 * time.Hour)
	p := puertosbolsa.PaginaHistorialMiBolsa{
		ConsultadaEn: consulta, Pagina: 1, Tamano: 20,
		Items: []puertosbolsa.HechoHistorialMiBolsa{{
			Clase: "contrato_bolsa", Bolsa: "bolsa:auxiliar", Categoria: "Auxiliar",
			OcurridoEn: consulta.Add(-time.Hour), Tipo: "incorporacion",
			Inicio: &inicioPrevisto, Procedencia: "evento_ct_recibido",
		}},
	}
	if !historialValido(p, consulta, 1) {
		t.Fatal("un inicio previsto futuro no invalida un evento CT recibido anteriormente")
	}
	p.Items[0].OcurridoEn = inicioPrevisto
	if historialValido(p, consulta, 1) {
		t.Fatal("un evento ocurrido después de la consulta no es histórico")
	}
}

func TestHistorialAvisoPreparadoConservaResultadoPendiente(t *testing.T) {
	consulta := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	pagina := puertosbolsa.PaginaHistorialMiBolsa{ConsultadaEn: consulta, Pagina: 1, Tamano: 20, Items: []puertosbolsa.HechoHistorialMiBolsa{{Clase: "llamamiento", Bolsa: "bolsa:sintetica", Categoria: "Auxiliar", OcurridoEn: consulta.Add(-time.Hour), Canal: "correo", Resultado: "aviso_pendiente"}}}
	if !historialValido(pagina, consulta, 1) {
		t.Fatal("la cola durable puede consultarse sin afirmar envío")
	}
	pagina.Items[0].Resultado = "smtp_supuesto"
	if historialValido(pagina, consulta, 1) {
		t.Fatal("no se admiten resultados de envío inventados")
	}
}
