package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func materialLectorRPTPrueba(t *testing.T) MaterialLectorRelacionRPT {
	t.Helper()
	actor := actorFichaPropiaPrueba(t, vinculoEmpleadoPrueba("pep_", "emp_"+strings.Repeat("b", 24)))
	m, err := NuevoMaterialLectorRelacionRPT(SolicitudLectorRelacionRPT{Actor: actor, EmpleadoRef: "emp_" + strings.Repeat("c", 24), RelacionRef: "rel_" + strings.Repeat("d", 24), OrganismoRef: "organismo:dipgra", VersionEsperada: 2, Corte: corteFichaPropiaPrueba()})
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func TestLectorRPTCanon17YObjetivoAjeno(t *testing.T) {
	m := materialLectorRPTPrueba(t)
	z := strings.Repeat("a", 24)
	s := m.Solicitud()
	esperado := `{"esquema":"vec.personal.relacion-rpt.consulta.v1","operacion":"relacion_para_rpt","empleado_ref":"` + s.EmpleadoRef + `","relacion_ref":"` + s.RelacionRef + `","organismo_ref":"organismo:dipgra","version_esperada":2,"vigente_en":"2026-09-25","conocido_en":"2026-09-25T09:59:59.123456Z","actor_ref":"per_` + z + `","contexto_actor_ref":"vca_` + z + `","contexto_version":3,"cuenta_ref":"cta_` + z + `","cuenta_version":2,"perfil_ref":"prf_` + z + `","perfil_version":5,"persona_ref":"per_` + z + `","persona_version":4}`
	if string(m.Canonico()) != esperado {
		t.Fatalf("canon divergente: %s", m.Canonico())
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(m.Canonico(), &campos) != nil || len(campos) != 17 {
		t.Fatal("canon no tiene17campos")
	}
	if s.Actor.Instantanea.Vinculos[0].Referencia == s.EmpleadoRef {
		t.Fatal("fixture no acredita actor diferente objetivo")
	}
	r := m.Recurso()
	hash := sha256.Sum256(m.Canonico())
	contexto := `{"ambitos":{"empleado_ref":"` + s.EmpleadoRef + `","organismo_ref":"organismo:dipgra","relacion_ref":"` + s.RelacionRef + `"},"atributos":{"conocido_en":"2026-09-25T09:59:59.123456Z","material_sha256":"` + hex.EncodeToString(hash[:]) + `","operacion":"relacion_para_rpt","version_esperada":"2","vigente_en":"2026-09-25"}}`
	h := sha256.Sum256([]byte(contexto))
	got, err := m.HuellaSHA256()
	if err != nil || got != hex.EncodeToString(h[:]) || r.Referencia != s.RelacionRef || r.Tipo != TipoRecursoLectorRelacionRPT || len(r.Ambitos) != 3 || len(r.Atributos) != 5 {
		t.Fatal("recurso divergente de SQL", err)
	}
	b := m.Canonico()
	b[0] = 'x'
	r.Ambitos["relacion_ref"] = "ajena"
	s.Actor.Instantanea.Vinculos[0].Referencia = "ajena"
	if string(m.Canonico()) != esperado || m.Recurso().Ambitos["relacion_ref"] != m.Solicitud().RelacionRef || m.Actor().Instantanea.Vinculos[0].Referencia == "ajena" {
		t.Fatal("material no clona")
	}
}
func TestLectorRPTActorActualIndependienteCorteHistorico(t *testing.T) {
	m := materialLectorRPTPrueba(t)
	s := m.Solicitud()
	s.Corte.ConocidoEn = s.Corte.ConocidoEn.Add(-30 * 24 * time.Hour)
	s.Corte.VigenteEn = "2020-01-01"
	antiguo, err := NuevoMaterialLectorRelacionRPT(s)
	if err != nil {
		t.Fatal(err)
	}
	if !antiguo.ActorVigenteEn(s.Actor.ResueltoEn) || antiguo.ActorVigenteEn(s.Actor.ResueltoEn.Add(time.Hour)) || antiguo.ActorVigenteEn(s.Corte.ConocidoEn) {
		t.Fatal("corte histórico revive actor")
	}
}
func TestLectorRPTInvalidaSeleccionSinReinterpretar(t *testing.T) {
	base := materialLectorRPTPrueba(t).Solicitud()
	for nombre, cambio := range map[string]func(*SolicitudLectorRelacionRPT){"empleado": func(s *SolicitudLectorRelacionRPT) { s.EmpleadoRef = "otro" }, "relacion": func(s *SolicitudLectorRelacionRPT) { s.RelacionRef = "rel:*" }, "organismo": func(s *SolicitudLectorRelacionRPT) { s.OrganismoRef = "*" }, "version": func(s *SolicitudLectorRelacionRPT) { s.VersionEsperada = 0 }, "corte": func(s *SolicitudLectorRelacionRPT) { s.Corte.VigenteEn = "ayer" }, "actor": func(s *SolicitudLectorRelacionRPT) { s.Actor.Principal.ID = "per_" + strings.Repeat("x", 24) }} {
		t.Run(nombre, func(t *testing.T) {
			s := base
			cambio(&s)
			_, err := NuevoMaterialLectorRelacionRPT(s)
			if !errors.Is(err, ErrLectorRelacionRPTInvalido) {
				t.Fatal("selector invalido admitido", err)
			}
		})
	}
}
func TestLectorRPTConservaEstadoYPeriodoNoInfiereEficacia(t *testing.T) {
	for _, estado := range []string{"vigente", "suspendida", "finalizada"} {
		if err := ValidarEstadoPeriodoLectorRelacionRPT(estado, "2020-01-01", ""); err != nil {
			t.Fatal("estado real abierto rechazado", err)
		}
	}
	if ValidarEstadoPeriodoLectorRelacionRPT("vigente", "2020-01-01", "2019-01-01") == nil || ValidarProcedenciaLectorRelacionRPT("acto:prueba", "fuente:prueba", "01") == nil {
		t.Fatal("fuente reinterpretada")
	}
	if ValidarProcedenciaLectorRelacionRPT("acto:prueba", "fuente:prueba", "2") != nil {
		t.Fatal("fuente valida rechazada")
	}
	if bytes.Contains(materialLectorRPTPrueba(t).Canonico(), []byte("fuente_ref")) {
		t.Fatal("copió fuente del actor")
	}
}
