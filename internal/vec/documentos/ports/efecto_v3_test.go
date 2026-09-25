package ports

import (
	"strings"
	"testing"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestHuellaEfectoV3CoincideConRecursoYVectorSQL(t *testing.T) {
	// Mismo vector que pruebas_sql/frontera_000004.sql y huella_efecto_v1.
	const vector = "fe8d170279095db5ae762a288e07bec21479a73c0050ce00929e893698a65f6a"
	if got := HuellaEfectoV3([]byte("abc")); got != vector {
		t.Fatalf("huella de efecto %s, esperado %s", got, vector)
	}
	for _, accion := range []string{AccionAlta, AccionListar, AccionDescargar, AccionPrepararNotificacion, AccionRegistrarExterno} {
		recurso, err := RecursoV3(accion, "ref:"+strings.Repeat("7", 64), []byte("abc"))
		if err != nil {
			t.Fatalf("%s: %v", accion, err)
		}
		huella, err := recurso.HuellaContextoAutorizacionSHA256()
		if err != nil || huella != vector || recurso.ModuloID != "documentos" {
			t.Fatalf("%s: huella de recurso %s %v", accion, huella, err)
		}
	}
	if _, err := RecursoV3("otra.accion", "ref:"+strings.Repeat("7", 64), []byte("abc")); err == nil {
		t.Fatal("acción desconocida aceptada")
	}
	if _, err := RecursoV3(AccionListar, "no/opaca", []byte("abc")); err == nil {
		t.Fatal("referencia no opaca aceptada")
	}
	if HuellaEfectoV3(nil) != "" || HuellaEfectoV3(make([]byte, 16385)) != "" {
		t.Fatal("preimagen no admisible con huella")
	}
}

// Vectores fijos compartidos con pruebas_sql/frontera_000004.sql, que los
// recalcula con vec_documentos.huella_efecto_v1: una preimagen de lista real y
// otra con texto no ASCII (UTF-8 de varios bytes).
func TestHuellaEfectoV3VectoresCompartidosConSQL(t *testing.T) {
	casos := []struct{ preimagen, huella string }{
		{`{"accion":"documentos.expediente.listar","expediente_ref":"exp:00000000-0000-4000-8000-000000000001","cursor":"","limite":1}`,
			"f0702b7b829d9741431f97288b3f2a78c8a896720af841f6b1e3aaeb33a8eb60"},
		{`{"motivo":"año"}`, "228f2c7ed9ba24be6394310f4cf55b8bbaa4389d2f7b8bcd8fa1a94e7742e9c9"},
	}
	for _, c := range casos {
		if got := HuellaEfectoV3([]byte(c.preimagen)); got != c.huella {
			t.Fatalf("huella de %s: %s, esperado %s", c.preimagen, got, c.huella)
		}
	}
}

func asignacionDocumentosPrueba(ambitos ...vecdomain.AmbitoPerfil) vecdomain.AsignacionPerfil {
	emitida := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	return vecdomain.AsignacionPerfil{AsignacionID: "asig-documentos", Version: 1, PerfilActivoRef: "perfil:rrhh",
		PrincipalID: "per:ensayo", VersionRolRef: "rol:documentos:v1", Estado: vecdomain.EstadoAsignacionPerfilActiva,
		Ambitos: ambitos, VigenteDesde: emitida, VigenteHasta: emitida.Add(24 * time.Hour), EmitidaPor: "per:gobierno", EmitidaEn: emitida}
}

// Una asignación AD3 exige al menos un ámbito y sólo cubre recursos con las
// mismas claves: el recurso documental lleva el de organización para que una
// asignación de esa organización conceda la lista y el registro externo, y
// ninguna otra lo haga.
func TestRecursoV3CubiertoSoloPorAsignacionDeLaOrganizacion(t *testing.T) {
	organizacion := vecdomain.AmbitoPerfil{Clave: "organizacion_ref", Valores: []string{OrganizacionRefV3}}
	concede := asignacionDocumentosPrueba(organizacion)
	if concede.Validar() != nil {
		t.Fatal("asignación de ensayo inválida")
	}
	negativas := map[string]vecdomain.AsignacionPerfil{
		"otra organización": asignacionDocumentosPrueba(vecdomain.AmbitoPerfil{Clave: "organizacion_ref", Valores: []string{"organizacion:otra"}}),
		"otra clave":        asignacionDocumentosPrueba(vecdomain.AmbitoPerfil{Clave: "expediente_ref", Valores: []string{OrganizacionRefV3}}),
		"ámbito de más": asignacionDocumentosPrueba(organizacion,
			vecdomain.AmbitoPerfil{Clave: "unidad_ref", Valores: []string{"unidad:rrhh"}}),
	}
	for _, accion := range []string{AccionListar, AccionRegistrarExterno, AccionAlta, AccionDescargar, AccionPrepararNotificacion} {
		recurso, err := RecursoV3(accion, "exp:00000000-0000-4000-8000-000000000001", []byte("abc"))
		if err != nil {
			t.Fatalf("%s: %v", accion, err)
		}
		if !concede.Cubre(recurso) {
			t.Fatalf("%s: la asignación de la organización no cubre el recurso", accion)
		}
		for nombre, a := range negativas {
			if a.Validar() != nil {
				t.Fatalf("%s: asignación negativa %q inválida", accion, nombre)
			}
			if a.Cubre(recurso) {
				t.Fatalf("%s: la asignación con %s cubre el recurso", accion, nombre)
			}
		}
	}
}
