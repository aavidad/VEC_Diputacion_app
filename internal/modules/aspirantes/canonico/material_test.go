package canonico

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/aspirantes/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func materialBase(accion string) ports.MaterialFicha {
	return ports.MaterialFicha{
		Superficie: vecdomain.SuperficieAutenticacionExternaPersonalV1, PersonaRef: "per_" + strings.Repeat("l", 24), PerfilRef: "prf_" + strings.Repeat("p", 24),
		Accion: accion, FinalidadRef: ports.FinalidadFicha,
		IndiceDocumento: ports.IndiceDocumento{ClaveRef: "clave:indice:v1", Valor: strings.Repeat("ab", 32)},
	}
}

// El vector fija los bytes exactos que PostgreSQL recibe como p_material.
func TestVectorMaterialConsultar(t *testing.T) {
	b, err := SerializarMaterial(materialBase(ports.AccionConsultar))
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"superficie":"externa_personal","persona_ref":"per_llllllllllllllllllllllll","perfil_ref":"prf_pppppppppppppppppppppppp","accion":"vec.aspirantes.ficha.consultar","finalidad_ref":"finalidad:aspirantes:ficha-propia:v1","version_esperada":0,"clave_operacion":"","huellas_peticion":{},"indice_documento":{"clave_ref":"clave:indice:v1","valor":"` + strings.Repeat("ab", 32) + `"}}`
	if string(b) != esperado {
		t.Fatalf("vector\n%s\n%s", b, esperado)
	}
	r, err := Recurso(materialBase(ports.AccionConsultar))
	if err != nil || r.Referencia != "per_"+strings.Repeat("l", 24) || r.ModuloID != "aspirantes" || r.Tipo != ports.TipoRecursoFicha || len(r.Atributos["material_sha256"]) != 64 {
		t.Fatalf("recurso %+v %v", r, err)
	}
}

func TestMaterialReglasPorAccion(t *testing.T) {
	huellas := ports.HuellasSemanticas{Activa: ports.HuellaSemantica{ClaveRef: "clave:huella:v2", Valor: strings.Repeat("0", 64)},
		Retenidas: []ports.HuellaSemantica{{ClaveRef: "clave:huella:v1", Valor: strings.Repeat("1", 64)}}}
	alta := materialBase(ports.AccionAlta)
	alta.ClaveOperacion, alta.HuellasPeticion = "alta-0000000000001", huellas
	b, err := SerializarMaterial(alta)
	if err != nil || !strings.Contains(string(b), `"retenidas":[{"clave_ref":"clave:huella:v1"`) {
		t.Fatalf("alta %s %v", b, err)
	}
	malos := []func(*ports.MaterialFicha){
		func(m *ports.MaterialFicha) { m.VersionEsperada = 1 },
		func(m *ports.MaterialFicha) { m.Superficie = vecdomain.SuperficieAutenticacionInternaCorporativaV1 },
		func(m *ports.MaterialFicha) { m.FinalidadRef = "otra" },
		func(m *ports.MaterialFicha) { m.IndiceDocumento.Valor = "12345678Z" },
		func(m *ports.MaterialFicha) { m.ClaveOperacion = "" },
		func(m *ports.MaterialFicha) {
			m.HuellasPeticion.Retenidas = append(m.HuellasPeticion.Retenidas, m.HuellasPeticion.Activa)
		},
		func(m *ports.MaterialFicha) { m.Accion = "vec.correos.consultar" },
	}
	for i, mal := range malos {
		m := alta
		m.HuellasPeticion.Retenidas = append([]ports.HuellaSemantica{}, huellas.Retenidas...)
		mal(&m)
		if _, err := SerializarMaterial(m); err == nil {
			t.Fatalf("caso %d aceptado", i)
		}
	}
	rect := alta
	rect.Accion = ports.AccionRectificar
	if _, err := SerializarMaterial(rect); err == nil {
		t.Fatal("rectificar exige versión esperada")
	}
	rect.VersionEsperada = 3
	if _, err := SerializarMaterial(rect); err != nil {
		t.Fatal(err)
	}
	consulta := materialBase(ports.AccionConsultar)
	consulta.ClaveOperacion = "no-debe-llevar-clave"
	if _, err := SerializarMaterial(consulta); err == nil {
		t.Fatal("consultar sin clave")
	}
}
