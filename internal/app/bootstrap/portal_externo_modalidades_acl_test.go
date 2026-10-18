package bootstrap

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestACLPortalExternoAdmiteSoloB138NominalOpcional(t *testing.T) {
	base := funcionesMiBolsaPortalExternoConPreferencias(true)
	modalidades := append(slices.Clone(base), "guardar_preferencias_modalidades_v1")
	casos := []struct {
		nombre    string
		funciones []string
		permitido bool
		espera    bool
	}{
		{"B138 ausente", base, true, true},
		{"B138 presente y exacta", modalidades, true, true},
		{"B138 presente sin firma o ACL", modalidades, false, false},
		{"funcion ajena", append(slices.Clone(modalidades), "guardar_ajeno_v1"), true, false},
		{"sobrecarga B138", append(slices.Clone(modalidades), "guardar_preferencias_modalidades_v1"), true, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			q := &consultadorACLMiBolsaInscripcionPrueba{
				funciones: caso.funciones, permitido: caso.permitido,
				funcionesB96: 1, b96Exacta: true,
				funcionesB104: 1, b104Exacta: true,
				funcionesLecturaB104: 1, lecturaB104Exacta: true,
				funcionesIntentoB104: 1, intentoB104Exacta: true,
			}
			if got := comprobarACLMiBolsaPortalExterno(context.Background(), q, true); got != caso.espera {
				t.Fatalf("preflight=%v, esperado=%v", got, caso.espera)
			}
			for _, guarda := range []string{
				"b138 AS", "guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint",
				"p.prokind='f'", "p.provolatile='v'", "p.prosecdef",
				"p.proowner='vec_bolsa_llamamientos_propietario'::regrole",
				"statement_timeout=15s", "pg_catalog.aclexplode", "a.grantee IN",
				"preferencia_modalidades_version", "t.relrowsecurity AND t.relforcerowsecurity",
				"count(*)=8 AND count(DISTINCT a.privilege_type)=8",
				"a.grantee=t.relowner AND a.grantor=t.relowner AND NOT a.is_grantable",
				"'SELECT','INSERT','UPDATE','DELETE','TRUNCATE','REFERENCES','TRIGGER','MAINTAIN'",
				"coalesce(t.relacl,pg_catalog.acldefault('r',t.relowner))",
				"huella_solicitud_modalidades_v1", "huella_contexto_modalidades_v1",
				"NOT EXISTS(SELECT 1 FROM funciones f WHERE f.proname='guardar_preferencias_modalidades_v1')",
			} {
				if !strings.Contains(q.consulta, guarda) {
					t.Fatalf("falta guarda de B138: %s", guarda)
				}
			}
			if strings.Contains(q.consulta, "t.relacl IS NULL") {
				t.Fatal("la guarda confundió ACL semántica con NULL")
			}
		})
	}
}
