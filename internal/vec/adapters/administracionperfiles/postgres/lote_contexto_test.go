package postgres

import (
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

// Vector Python SHA256/JSON con orden explícito: AUT44 reconstruye
// exactamente estos bytes en recurso_lote_admin_v1.
func TestLoteHuellaContextoV3IndependienteDeHuellaSolicitud(t *testing.T) {
	const solicitudSHA = "2309a242dda19bcb4396565a8aae684f35bce0b10abca88dd45ef560f853e492"
	const contextoSHA = "2fb8a0cc32cce33eeaff3153b8b7be63285f9b0116ea1fff52cf08479fe4cf41"
	r := domain.RecursoAutorizable{Referencia: "per_aaaaaaaaaaaaaaaaaaaaaa", ModuloID: "administracion", Tipo: "persona",
		Ambitos:   map[string]string{"organizacion_ref": "org_prueba", "unidad_ref": "unidad:prueba"},
		Atributos: map[string]string{"solicitud_sha256": solicitudSHA}}
	h, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil || h != contextoSHA || h == solicitudSHA {
		t.Fatal("capacidad liga huella equivocada")
	}
}
