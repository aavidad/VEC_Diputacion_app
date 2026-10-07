package domain

import "testing"

func TestPoliticaNumeroExpedienteConfigurada(t *testing.T) {
	p := PoliticaNumeroExpediente{Referencia: "catalogo:ct:moad", Version: 1, Patron: `^[0-9]{4}/[1-9][0-9]{0,9}$`, Ejemplo: "2026/5487"}
	if p.ValidarNumero("2026/5487") != nil {
		t.Fatal("número MOAD válido rechazado")
	}
	for _, n := range []string{"", "2026/CT-5487", "2026/05487", " 2026/5487", "2026/5487\n"} {
		if p.ValidarNumero(n) == nil {
			t.Fatalf("número inválido aceptado %q", n)
		}
	}
	p.Version = 2
	p.Patron = `^[0-9]{4}/EXP-[1-9][0-9]{0,9}$`
	p.Ejemplo = "2026/EXP-5487"
	if p.ValidarNumero("2026/EXP-5487") != nil || p.ValidarNumero("2026/5487") == nil {
		t.Fatal("se ignoró la publicación configurada")
	}
	for _, patron := range []string{"[", "[0-9]+", "^.*$"} {
		p.Patron = patron
		p.Ejemplo = "sin-numero"
		if p.Validar() == nil {
			t.Fatalf("publicación inválida admitida %q", patron)
		}
	}
}
