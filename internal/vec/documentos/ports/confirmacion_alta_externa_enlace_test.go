package ports

import (
	"encoding/json"
	"strings"
	"testing"
)

func solicitudEnlacePrueba() SolicitudConfirmacionAltaExternaEnlace {
	ref := func(digito string) string { return "ref:" + strings.Repeat(digito, 64) }
	return SolicitudConfirmacionAltaExternaEnlace{
		DocumentoID: ref("1"), ExpedienteRef: ref("2"), TipoRef: ref("3"),
		Version: 1, ContenidoSHA256: strings.Repeat("a", 64), CustodioID: "cronos.justificantes",
		CustodiaRef: "cronos:justificante:001", ClaveAlta: ref("4"),
		ActorAltaRef: "per_" + strings.Repeat("A", 22), SolicitudRef: "permiso:cronos:solicitud:ensayo001",
		MaterialEnlaceSHA256: strings.Repeat("b", 64),
	}
}

func TestConfirmacionAltaExternaLigaMaterialYCambiaConDestino(t *testing.T) {
	s := solicitudEnlacePrueba()
	p, err := s.Preimagen()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(p, &m); err != nil || len(m) != 13 ||
		m["accion"] != AccionConfirmarAltaExternaEnlace || m["modulo_origen"] != "cronos" ||
		m["actor_ref"] != s.ActorAltaRef || m["clave_alta"] != s.ClaveAlta {
		t.Fatalf("preimagen de enlace incompleta: %v %v", m, err)
	}
	recurso, err := s.RecursoV3()
	if err != nil || recurso.Validar() != nil || recurso.Tipo != "alta_externa_enlace" {
		t.Fatalf("recurso documental inválido: %v", err)
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || h != HuellaEfectoV3(p) {
		t.Fatalf("huella V3 distinta de la preimagen: %s %v", h, err)
	}
	s.MaterialEnlaceSHA256 = strings.Repeat("c", 64)
	q, err := s.Preimagen()
	if err != nil || HuellaEfectoV3(q) == h {
		t.Fatal("otro material Cronos conserva la concesión")
	}
	s.SolicitudRef = "otra-solicitud"
	if _, err := s.Preimagen(); err == nil {
		t.Fatal("solicitud no opaca aceptada")
	}
}
