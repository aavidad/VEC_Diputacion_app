package separacionportales

import "testing"

func TestMaterialAuditoriaIntentosExterna(t *testing.T) {
	material := materialSintetico(t, PortalExterno)
	escribir(t, material, "auditoria-intentos-externa.json", `{"dsn_file":"externo/auditoria-intentos.dsn"}`)
	escribir(t, material, "externo/auditoria-intentos.dsn", "conexion sintetica")
	if err := ComprobarMaterial(PortalExterno, material); err != nil {
		t.Fatalf("el material nominal de auditoria externa fue rechazado: %v", err)
	}

	for _, relativa := range []string{
		"auditoria-intentos.json",
		"auditoria-intentos-externa.json.bak",
		"otra-auditoria.json",
		"auditoria-intentos.dsn",
		"otra-conexion.dsn",
		"ca/ca.key",
		"externo/cliente.key",
	} {
		t.Run(relativa, func(t *testing.T) {
			material := materialSintetico(t, PortalExterno)
			escribir(t, material, relativa, "contenido sintetico")
			if got := motivo(t, ComprobarMaterial(PortalExterno, material)); got != relativa {
				t.Fatalf("se esperaba rechazar %s, llego %s", relativa, got)
			}
		})
	}

	for _, relativa := range []string{
		"auditoria-intentos-externa.json",
		"externo/auditoria-intentos.dsn",
	} {
		t.Run("interno/"+relativa, func(t *testing.T) {
			material := materialSintetico(t, PortalInterno)
			escribir(t, material, relativa, "contenido sintetico")
			if got := motivo(t, ComprobarMaterial(PortalInterno, material)); got != relativa {
				t.Fatalf("el portal interno acepto %s: %s", relativa, got)
			}
		})
	}
}
