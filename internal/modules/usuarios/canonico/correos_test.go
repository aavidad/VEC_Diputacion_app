package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestAudienciasCorreosCerradasPorSuperficie(t *testing.T) {
	acciones := []string{ports.AccionConsultarCorreos, ports.AccionAnadirCorreo, ports.AccionReenviarCorreo, ports.AccionVerificarCorreo, ports.AccionActivarCorreo, ports.AccionRetirarCorreo}
	vistas := map[string]bool{}
	for _, superficie := range []vecdomain.SuperficieAutenticacionActorV1{vecdomain.SuperficieAutenticacionInternaCorporativaV1, vecdomain.SuperficieAutenticacionExternaPersonalV1} {
		for _, accion := range acciones {
			audiencia, err := AudienciaCorreos(accion, superficie)
			if err != nil || audiencia == "" || vistas[audiencia] {
				t.Fatalf("audiencia ausente/duplicada %s %s: %v", accion, superficie, err)
			}
			vistas[audiencia] = true
		}
	}
	if len(vistas) != 12 {
		t.Fatal("faltan audiencias nominales")
	}
	if _, err := AudienciaCorreos(ports.AccionConsultarCorreos, vecdomain.SuperficieAutenticacionAdministracionPrivilegiadaV1); !errors.Is(err, ports.ErrCorreosProhibido) {
		t.Fatalf("superficie privilegiada aceptada: %v", err)
	}
	if _, err := AudienciaCorreos("vec.correos.otro", vecdomain.SuperficieAutenticacionInternaCorporativaV1); !errors.Is(err, ports.ErrCorreosProhibido) {
		t.Fatalf("acción no declarada aceptada: %v", err)
	}
}

func TestSerializarMaterialCorreosVectorCanonico(t *testing.T) {
	m := ports.MaterialCorreos{Superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1, PersonaRef: "per_prueba", PerfilRef: "prf_prueba", Accion: ports.AccionAnadirCorreo, FinalidadRef: ports.FinalidadCorreosPropios, ClaveOperacion: "operacion-1234567890", HuellasPeticion: ports.HuellasSemanticasCorreo{Activa: ports.HuellaSemanticaCorreo{ClaveRef: "h3", Valor: strings.Repeat("c", 64)}, Retenidas: []ports.HuellaSemanticaCorreo{{ClaveRef: "h2", Valor: strings.Repeat("b", 64)}, {ClaveRef: "h1", Valor: strings.Repeat("a", 64)}}}}
	b, err := SerializarMaterialCorreos(m)
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"superficie":"interna_corporativa","persona_ref":"per_prueba","perfil_ref":"prf_prueba","accion":"vec.correos.anadir","finalidad_ref":"finalidad:usuarios:correos-propios:v1","version_esperada":0,"clave_operacion":"operacion-1234567890","huellas_peticion":{"activa":{"clave_ref":"h3","valor":"` + strings.Repeat("c", 64) + `"},"retenidas":[{"clave_ref":"h1","valor":"` + strings.Repeat("a", 64) + `"},{"clave_ref":"h2","valor":"` + strings.Repeat("b", 64) + `"}]},"correo_ref":""}`
	if string(b) != esperado {
		t.Fatalf("bytes distintos:\n%s\n%s", b, esperado)
	}
	if m.HuellasPeticion.Retenidas[0].ClaveRef != "h2" {
		t.Fatal("serializador mutó el material")
	}
	h := sha256.Sum256(b)
	t.Logf("vector SQL bytes=%s", b)
	t.Logf("vector SQL SHA256=%s", hex.EncodeToString(h[:]))
	m.PersonaRef = "per_" + strings.Repeat("r", 24)
	m.PerfilRef = "prf_" + strings.Repeat("p", 24)
	b, err = SerializarMaterialCorreos(m)
	if err != nil {
		t.Fatal(err)
	}
	esperadoIdentidad := strings.ReplaceAll(strings.ReplaceAll(esperado, "per_prueba", m.PersonaRef), "prf_prueba", m.PerfilRef)
	if string(b) != esperadoIdentidad {
		t.Fatalf("bytes con identidad válida distintos: %s", b)
	}
	h = sha256.Sum256(b)
	t.Logf("vector SQL identidad válida bytes=%s", b)
	t.Logf("vector SQL identidad válida SHA256=%s", hex.EncodeToString(h[:]))
	recurso, err := RecursoCorreos(m)
	if err != nil {
		t.Fatal(err)
	}
	if recurso.Referencia != m.PersonaRef || recurso.ModuloID != "usuarios" || recurso.Tipo != ports.TipoRecursoCorreos || len(recurso.Ambitos) != 1 || recurso.Ambitos["persona_ref"] != m.PersonaRef || len(recurso.Atributos) != 1 || recurso.Atributos["material_sha256"] != hex.EncodeToString(h[:]) {
		t.Fatalf("recurso V3 no canónico: %+v", recurso)
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || huellaContexto != "2a74f978a1b52f3985c3c53fda94b0f5cb052628fa6f93ceec4254ed61a529b5" || huellaContexto == hex.EncodeToString(h[:]) {
		t.Fatalf("huella V3 confundida con SHA material: %s, %v", huellaContexto, err)
	}
	t.Logf("vector V3 contexto SHA256=%s", huellaContexto)
	m.PersonaRef, m.PerfilRef = "per_prueba", "prf_prueba"

	m.Accion = ports.AccionConsultarCorreos
	m.ClaveOperacion = ""
	m.HuellasPeticion = ports.HuellasSemanticasCorreo{}
	b, err = SerializarMaterialCorreos(m)
	if err != nil {
		t.Fatal(err)
	}
	esperado = `{"superficie":"interna_corporativa","persona_ref":"per_prueba","perfil_ref":"prf_prueba","accion":"vec.correos.consultar","finalidad_ref":"finalidad:usuarios:correos-propios:v1","version_esperada":0,"clave_operacion":"","huellas_peticion":{},"correo_ref":""}`
	if string(b) != esperado {
		t.Fatalf("lectura no usa objeto vacío: %s", b)
	}
}
