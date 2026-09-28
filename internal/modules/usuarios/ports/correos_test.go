package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestAudienciasCorreosCerradasPorSuperficie(t *testing.T) {
	acciones := []string{AccionConsultarCorreos, AccionAnadirCorreo, AccionReenviarCorreo, AccionVerificarCorreo, AccionActivarCorreo, AccionRetirarCorreo}
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
	if _, err := AudienciaCorreos(AccionConsultarCorreos, vecdomain.SuperficieAutenticacionAdministracionPrivilegiadaV1); !errors.Is(err, ErrCorreosProhibido) {
		t.Fatalf("superficie privilegiada aceptada: %v", err)
	}
	if _, err := AudienciaCorreos("vec.correos.otro", vecdomain.SuperficieAutenticacionInternaCorporativaV1); !errors.Is(err, ErrCorreosProhibido) {
		t.Fatalf("acción no declarada aceptada: %v", err)
	}
}

func TestSerializarMaterialCorreosVectorCanonico(t *testing.T) {
	m := MaterialCorreos{Superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1, PersonaRef: "per_prueba", PerfilRef: "prf_prueba", Accion: AccionAnadirCorreo, FinalidadRef: FinalidadCorreosPropios, ClaveOperacion: "operacion-1234567890", HuellasPeticion: HuellasSemanticasCorreo{Activa: HuellaSemanticaCorreo{ClaveRef: "h3", Valor: strings.Repeat("c", 64)}, Retenidas: []HuellaSemanticaCorreo{{ClaveRef: "h2", Valor: strings.Repeat("b", 64)}, {ClaveRef: "h1", Valor: strings.Repeat("a", 64)}}}}
	b, err := SerializarMaterialCorreos(m)
	if err != nil {
		t.Fatal(err)
	}
	porMarshal, err := json.Marshal(m)
	if err != nil || string(porMarshal) != string(b) {
		t.Fatal("json.Marshal omitió serializador canónico")
	}
	esperado := `{"superficie":"interna_corporativa","persona_ref":"per_prueba","perfil_ref":"prf_prueba","accion":"vec.correos.anadir","finalidad_ref":"finalidad:usuarios:correos-propios:v1","version_esperada":0,"clave_operacion":"operacion-1234567890","huellas_peticion":{"activa":{"clave_ref":"h3","valor":"` + strings.Repeat("c", 64) + `"},"retenidas":[{"clave_ref":"h1","valor":"` + strings.Repeat("a", 64) + `"},{"clave_ref":"h2","valor":"` + strings.Repeat("b", 64) + `"}]},"correo_ref":"","sustituto_ref":""}`
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
	if recurso.Referencia != m.PersonaRef || recurso.ModuloID != "usuarios" || recurso.Tipo != TipoRecursoCorreos || len(recurso.Ambitos) != 1 || recurso.Ambitos["persona_ref"] != m.PersonaRef || len(recurso.Atributos) != 1 || recurso.Atributos["material_sha256"] != hex.EncodeToString(h[:]) {
		t.Fatalf("recurso V3 no canónico: %+v", recurso)
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || huellaContexto != "31e961606d676ba64bb577e4666777ad78c0a3100710693edd219388a8c954a8" || huellaContexto == hex.EncodeToString(h[:]) {
		t.Fatalf("huella V3 confundida con SHA material: %s, %v", huellaContexto, err)
	}
	t.Logf("vector V3 contexto SHA256=%s", huellaContexto)
	m.PersonaRef, m.PerfilRef = "per_prueba", "prf_prueba"

	m.Accion = AccionConsultarCorreos
	m.ClaveOperacion = ""
	m.HuellasPeticion = HuellasSemanticasCorreo{}
	b, err = SerializarMaterialCorreos(m)
	if err != nil {
		t.Fatal(err)
	}
	esperado = `{"superficie":"interna_corporativa","persona_ref":"per_prueba","perfil_ref":"prf_prueba","accion":"vec.correos.consultar","finalidad_ref":"finalidad:usuarios:correos-propios:v1","version_esperada":0,"clave_operacion":"","huellas_peticion":{},"correo_ref":"","sustituto_ref":""}`
	if string(b) != esperado {
		t.Fatalf("lectura no usa objeto vacío: %s", b)
	}
}
