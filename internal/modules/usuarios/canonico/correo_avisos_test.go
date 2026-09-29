package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

func solicitudAvisosPrueba() ports.SolicitudCorreoAvisos {
	return ports.SolicitudCorreoAvisos{
		BolsaRef: "bolsa:encargado:2026-09-17", UnidadRef: "unidad:rrhh/bolsa", AmbitoRef: "ambito:diputacion",
		LlamamientoRef: "llamamiento:" + strings.Repeat("a1", 32), CandidatoRef: "can_" + strings.Repeat("Z", 30),
	}
}

// La huella que firma la V3 debe coincidir con la que PostgreSQL recalcula
// (Usuarios 000008) concatenando el canon a mano.
func TestHuellaRecursoAvisosCoincideConLaDeSQL(t *testing.T) {
	m, err := MaterialCorreoAvisos(solicitudAvisosPrueba())
	if err != nil {
		t.Fatal(err)
	}
	recurso, material, err := RecursoCorreoAvisos(m)
	if err != nil {
		t.Fatal(err)
	}
	huellaGo, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	texto := func(v string) string { b, _ := json.Marshal(v); return string(b) }
	hm := sha256.Sum256(material)
	canonSQL := `{"ambitos":{"ambito_ref":` + texto(m.AmbitoRef) + `,"unidad_ref":` + texto(m.UnidadRef) + `},"atributos":{"material_sha256":"` + hex.EncodeToString(hm[:]) + `"}}`
	hs := sha256.Sum256([]byte(canonSQL))
	if huellaGo != hex.EncodeToString(hs[:]) {
		t.Fatalf("huella Go %s distinta de la SQL %s", huellaGo, hex.EncodeToString(hs[:]))
	}
	if recurso.Referencia != m.BolsaRef || recurso.ModuloID != "bolsa" || recurso.Tipo != "bolsa_constituida" {
		t.Fatalf("recurso: %+v", recurso)
	}
	var claves map[string]any
	if json.Unmarshal(material, &claves) != nil || len(claves) != 8 || claves["superficie"] != "interna_corporativa" || claves["finalidad_ref"] != "gestion_llamamientos_bolsa" {
		t.Fatalf("material: %s", material)
	}
}

func TestMaterialAvisosRechazaReferenciasAmbiguas(t *testing.T) {
	cambios := []func(*ports.SolicitudCorreoAvisos){
		func(s *ports.SolicitudCorreoAvisos) { s.CandidatoRef = "per_" + strings.Repeat("Z", 30) },
		func(s *ports.SolicitudCorreoAvisos) { s.CandidatoRef = "" },
		func(s *ports.SolicitudCorreoAvisos) { s.LlamamientoRef = "llamamiento:xyz" },
		func(s *ports.SolicitudCorreoAvisos) { s.BolsaRef = "bolsa<1>" },
		func(s *ports.SolicitudCorreoAvisos) { s.UnidadRef = "a&b" },
		func(s *ports.SolicitudCorreoAvisos) { s.AmbitoRef = "con\"comillas" },
		func(s *ports.SolicitudCorreoAvisos) { s.AmbitoRef = "linea\nnueva" },
		func(s *ports.SolicitudCorreoAvisos) { s.UnidadRef = "sep " },
		func(s *ports.SolicitudCorreoAvisos) { s.BolsaRef = " bolsa" },
		func(s *ports.SolicitudCorreoAvisos) { s.BolsaRef = strings.Repeat("b", 513) },
		func(s *ports.SolicitudCorreoAvisos) { s.UnidadRef = "" },
	}
	for i, cambiar := range cambios {
		s := solicitudAvisosPrueba()
		cambiar(&s)
		if _, err := MaterialCorreoAvisos(s); err == nil {
			t.Fatalf("caso %d admitido", i)
		}
	}
	m, _ := MaterialCorreoAvisos(solicitudAvisosPrueba())
	m.Superficie = "externa_personal"
	if _, err := SerializarMaterialCorreoAvisos(m); err == nil {
		t.Fatal("la lectura sólo existe en la superficie interna")
	}
}

func TestCorreoRefValida(t *testing.T) {
	if !CorreoRefValida("correo:0123456789abcdef0123456789abcdef") || CorreoRefValida("correo:0123") || CorreoRefValida("persona@ejemplo.es") {
		t.Fatal("forma de referencia de correo")
	}
}
