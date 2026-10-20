package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestIntencionTransicionesLigaCuerpoYPerfilComoSQL(t *testing.T) {
	actor := "per_" + strings.Repeat("a", 24)
	perfil := "prf_rrhh_sintetico_1234567890123456"
	sha := strings.Repeat("b", 64)
	e := bolsahttp.EntradaGobiernoTransicionesSituacion{VersionEsperada: 2,
		Transiciones: []string{"disponible>excluido", "renuncia>excluido"},
		Motivo:       "ajuste aprobado", ClaveIdempotencia: "clave-12345678"}
	recurso := dominiovec.RecursoAutorizable{Referencia: "bolsa:politica:transiciones:global", ModuloID: "bolsa",
		Tipo: "politica_transiciones_global", Ambitos: map[string]string{"ambito_ref": "ambito:bolsa", "unidad_ref": "unidad:rrhh"},
		Atributos: atributosIntencionTransicionesGobierno(e, actor, perfil, 3, sha)}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	// Misma preimagen ASCII que reconstruye B122 antes de consumir el efecto.
	preimagen := fmt.Sprintf(`{"ambitos":{"ambito_ref":"%s","unidad_ref":"%s"},"atributos":{`+
		`"actor_ref":"%s","catalogo_sha256":"%s","catalogo_version":"%s",`+
		`"clave_sha256":"%s","motivo_sha256":"%s","perfil_activo_ref":"%s",`+
		`"transiciones_sha256":"%s","version_esperada":"%s"}}`,
		"ambito:bolsa", "unidad:rrhh", actor, sha, "3", huellaTransicionesGobierno(e.ClaveIdempotencia),
		huellaTransicionesGobierno(e.Motivo), perfil, huellaTransicionesGobierno(strings.Join(e.Transiciones, ",")), "2")
	suma := sha256.Sum256([]byte(preimagen))
	if huella != hex.EncodeToString(suma[:]) {
		t.Fatalf("Go y B122 no ligan la misma intención: %s", huella)
	}
}
