package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Mismos bytes y huellas que pruebas_sql/imagen_vector_material.sql: Go y SQL
// deben calcular igual la huella de petición y la de contexto V3.
const (
	materialImagenVector = `{"superficie":"externa_personal","persona_ref":"per_rrrrrrrrrrrrrrrrrrrrrrrr","perfil_ref":"prf_pppppppppppppppppppppppp","accion":"vec.imagen.actualizar","finalidad_ref":"finalidad:usuarios:imagen-propia:v1","catalogo_version_ref":"usuarios-imagen-v1","version_esperada":3,"clave_operacion":"operacion-1234567890","huella_peticion":"5326d57059087f951ba0cf93e12118736b3d45ba1fb85a4f866fb5a04040d035","eleccion":{"modo":"foto","paleta":"verde","icono":""},"foto_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	contextoImagenVector = "318c9ea3115b62fd039316e8fc8215090e08ffc199034b7c8f672aa5c36cdae7"
)

func materialImagenPrueba() ports.MaterialImagen {
	e := domain.EleccionImagen{Modo: domain.ModoImagenFoto, Paleta: "verde"}
	foto := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	return ports.MaterialImagen{Superficie: vecdomain.SuperficieAutenticacionExternaPersonalV1, PersonaRef: "per_rrrrrrrrrrrrrrrrrrrrrrrr",
		PerfilRef: "prf_pppppppppppppppppppppppp", Accion: ports.AccionActualizarImagen, FinalidadRef: ports.FinalidadImagenPropia,
		CatalogoVersionRef: "usuarios-imagen-v1", VersionEsperada: 3, ClaveOperacion: "operacion-1234567890",
		HuellaPeticion: HuellaPeticionImagen("per_rrrrrrrrrrrrrrrrrrrrrrrr", 3, "usuarios-imagen-v1", e, foto), Eleccion: e, FotoSHA256: foto}
}

func TestVectorMaterialImagenCoincideConSQL(t *testing.T) {
	b, err := SerializarMaterialImagen(materialImagenPrueba())
	if err != nil || string(b) != materialImagenVector {
		t.Fatalf("bytes del material distintos del vector SQL: %v\n%s", err, b)
	}
	r, err := RecursoImagen(materialImagenPrueba())
	if err != nil {
		t.Fatal(err)
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil || h != contextoImagenVector || r.Tipo != ports.TipoRecursoImagen || r.Ambitos["persona_ref"] != "per_rrrrrrrrrrrrrrrrrrrrrrrr" {
		t.Fatalf("recurso V3 distinto del vector SQL: %v %s", err, h)
	}
	s := sha256.Sum256(b)
	if r.Atributos["material_sha256"] != hex.EncodeToString(s[:]) {
		t.Fatal("el recurso no liga la huella de los bytes del material")
	}
}

func TestMaterialImagenRechazaIncoherencias(t *testing.T) {
	casos := map[string]func(*ports.MaterialImagen){
		"huella ajena": func(m *ports.MaterialImagen) {
			m.HuellaPeticion = HuellaPeticionImagen(m.PersonaRef, 4, m.CatalogoVersionRef, m.Eleccion, m.FotoSHA256)
		},
		"foto fuera de modo": func(m *ports.MaterialImagen) {
			m.Eleccion = domain.EleccionImagen{Modo: domain.ModoImagenIniciales, Paleta: "verde"}
		},
		"consulta con clave":     func(m *ports.MaterialImagen) { m.Accion = ports.AccionConsultarImagen },
		"superficie inexistente": func(m *ports.MaterialImagen) { m.Superficie = "otra" },
		"finalidad ajena":        func(m *ports.MaterialImagen) { m.FinalidadRef = ports.FinalidadCorreosPropios },
		"foto no hexadecimal":    func(m *ports.MaterialImagen) { m.FotoSHA256 = "ZZ" },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			m := materialImagenPrueba()
			mutar(&m)
			if _, err := SerializarMaterialImagen(m); err == nil {
				t.Fatal("material incoherente aceptado")
			}
		})
	}
	consulta := ports.MaterialImagen{Superficie: vecdomain.SuperficieAutenticacionInternaCorporativaV1, PersonaRef: "per_rrrrrrrrrrrrrrrrrrrrrrrr",
		PerfilRef: "prf_pppppppppppppppppppppppp", Accion: ports.AccionConsultarImagen, FinalidadRef: ports.FinalidadImagenPropia, CatalogoVersionRef: "usuarios-imagen-v1"}
	if b, err := SerializarMaterialImagen(consulta); err != nil || string(b) == "" {
		t.Fatalf("consulta válida rechazada: %v", err)
	}
	if a, err := AudienciaImagen(ports.AccionConsultarImagen, vecdomain.SuperficieAutenticacionExternaPersonalV1); err != nil || a != ports.AudienciaConsultarImagenExterna {
		t.Fatal("audiencia de consulta externa incorrecta")
	}
}
