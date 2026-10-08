package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMaterialLectorServiciosCertificadosCanonicoYRecurso(t *testing.T) {
	empleado := "emp_" + strings.Repeat("b", 24)
	actor := actorFichaPropiaPrueba(t, vinculoEmpleadoPrueba("pep_", empleado))
	m, err := NuevoMaterialLectorServiciosCertificados(SolicitudLectorServiciosCertificados{Actor: actor, EmpleadoRef: empleado, OrganismoRef: "dipgra", Corte: corteFichaPropiaPrueba()})
	if err != nil {
		t.Fatal(err)
	}
	z := strings.Repeat("a", 24)
	// Personal36 reconstruye exactamente estos bytes; cualquier cambio de orden
	// o de forma rompe la concesión.
	esperado := `{"esquema":"vec.personal.servicios-certificados.consulta.v1","operacion":"servicios_certificados_propios","empleado_ref":"` + empleado +
		`","organismo_ref":"dipgra","vigente_en":"2026-09-25","conocido_en":"2026-09-25T09:59:59.123456Z","actor_ref":"per_` + z +
		`","contexto_actor_ref":"vca_` + z + `","contexto_version":3,"cuenta_ref":"cta_` + z + `","cuenta_version":2,"perfil_ref":"prf_` + z +
		`","perfil_version":5,"persona_ref":"per_` + z + `","persona_version":4}`
	if string(m.Canonico()) != esperado {
		t.Fatalf("material no canónico:\n%s\n%s", m.Canonico(), esperado)
	}
	suma := sha256.Sum256([]byte(esperado))
	contexto := `{"ambitos":{"empleado_ref":"` + empleado + `","organismo_ref":"dipgra"},"atributos":{"conocido_en":"2026-09-25T09:59:59.123456Z","material_sha256":"` +
		hex.EncodeToString(suma[:]) + `","operacion":"servicios_certificados_propios","vigente_en":"2026-09-25"}}`
	sumaContexto := sha256.Sum256([]byte(contexto))
	h, err := m.HuellaSHA256()
	r := m.Recurso()
	if err != nil || h != hex.EncodeToString(sumaContexto[:]) || r.Referencia != empleado || r.Tipo != TipoRecursoLectorServiciosCertificados || r.ModuloID != "personal" {
		t.Fatalf("recurso divergente: %s %+v %v", h, r, err)
	}
	r.Ambitos["empleado_ref"] = "otro"
	if m.Recurso().Ambitos["empleado_ref"] != empleado {
		t.Fatal("el recurso comparte sus mapas")
	}
}

func TestMaterialLectorServiciosCertificadosSoloEmpleadoPropio(t *testing.T) {
	propio := "emp_" + strings.Repeat("b", 24)
	for _, caso := range []struct {
		nombre   string
		actorEmp []string
		prefijo  string
		pedido   string
		err      error
	}{
		{"ajeno", []string{propio}, "pep_", "emp_" + strings.Repeat("c", 24), ErrLectorServiciosCertificadosDenegado},
		{"sin_empleado", nil, "pep_", propio, ErrLectorServiciosCertificadosDenegado},
		{"heredado", []string{propio}, "vin_", propio, ErrLectorServiciosCertificadosDenegado},
		{"mal_formado", []string{propio}, "pep_", "empleado", ErrLectorServiciosCertificadosInvalido},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			actor := actorFichaPropiaPrueba(t)
			if len(caso.actorEmp) > 0 {
				actor = actorFichaPropiaPrueba(t, vinculoEmpleadoPrueba(caso.prefijo, caso.actorEmp[0]))
			}
			_, err := NuevoMaterialLectorServiciosCertificados(SolicitudLectorServiciosCertificados{Actor: actor, EmpleadoRef: caso.pedido, OrganismoRef: "dipgra", Corte: corteFichaPropiaPrueba()})
			if !errors.Is(err, caso.err) {
				t.Fatalf("esperado %v, obtenido %v", caso.err, err)
			}
		})
	}
}

func TestServicioLeidoCertificadosForma(t *testing.T) {
	corte := CorteEmpleadoB2{VigenteEn: "2026-09-25", ConocidoEn: time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)}
	base := ServicioLeidoCertificados{ServicioRef: "srv_" + strings.Repeat("s", 24), RelacionRef: "rel_" + strings.Repeat("r", 24), Estado: "comprobado", ClaseRef: "interinidad",
		Version: 2, DiasReconocidos: 0, ClaseVersion: 0, Desde: "2025-01-01", Hasta: "2027-01-01", ActoRef: "acto:a1", FuenteRef: "fuente:b2", FuenteVersion: "1"}
	if ValidarServicioLeidoCertificados(base, corte) != nil {
		t.Fatal("servicio en curso al corte rechazado")
	}
	for nombre, mutar := range map[string]func(*ServicioLeidoCertificados){
		"estado":       func(s *ServicioLeidoCertificados) { s.Estado = "acreditado" },
		"periodo":      func(s *ServicioLeidoCertificados) { s.Hasta = s.Desde },
		"tras_corte":   func(s *ServicioLeidoCertificados) { s.Desde = "2026-09-26" },
		"dias":         func(s *ServicioLeidoCertificados) { s.DiasReconocidos = -1 },
		"fuente":       func(s *ServicioLeidoCertificados) { s.FuenteVersion = "01" },
		"version":      func(s *ServicioLeidoCertificados) { s.Version = 0 },
		"servicio_ref": func(s *ServicioLeidoCertificados) { s.ServicioRef = "srv_corto" },
	} {
		s := base
		mutar(&s)
		if !errors.Is(ValidarServicioLeidoCertificados(s, corte), ErrLectorServiciosCertificadosInvalido) {
			t.Fatalf("%s admitido", nombre)
		}
	}
}
