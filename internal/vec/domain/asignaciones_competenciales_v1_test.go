package domain_test

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func asignacionCompetencialPrueba(t *testing.T) (domain.SolicitudAsignacionCompetencialV1, domain.EvidenciaAsignacionCompetencialV1, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	s := domain.SolicitudAsignacionCompetencialV1{
		Actor: resultado.Contexto, ResultadoContexto: resultado, Vinculo: vinculo,
		PersonaRef: "per_0123456789abcdefghijkm", PerfilFirmanteRef: "perfil:secretaria:v1",
		CertificadoHuellaSHA256: strings.Repeat("1", 64),
		Cargo:                   domain.ReferenciaCargoCompetencialV1{Referencia: "cargo:secretaria", Version: 2, HuellaSHA256: strings.Repeat("2", 64)},
		Recurso: domain.RecursoAutorizable{Referencia: "documento:resolucion", ModuloID: "contratacion_temporal", Tipo: "documento",
			Ambitos: map[string]string{"organizacion": "organizacion:diputacion", "unidad": "unidad:secretaria"}},
		AccionLectura: "administracion.asignaciones_competenciales.consultar", FinalidadLectura: "comprobar_competencia",
		AccionCompetencial: "contratacion_temporal.documento.firmar", FinalidadCompetencial: "formalizar",
		CorrelacionRef: "correlacion_" + strings.Repeat("a", 32),
		Motivo: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("3", 64), EntradaClave: "motivo_" + strings.Repeat("b", 32)},
	}
	rol := domain.VersionRol{RolID: "firmante_secretaria", Version: 1, Nombre: "Secretaria", Estado: domain.EstadoVersionRolPublicada,
		PublicadaPor: "acto:publicacion:rol", PublicadaEn: ahora.Add(-time.Hour),
		Concesiones: []domain.ConcesionRol{{Accion: s.AccionCompetencial, ModuloID: s.Recurso.ModuloID, TipoRecurso: s.Recurso.Tipo,
			Finalidades: []string{s.FinalidadCompetencial}, GarantiaMinima: domain.AuthAssuranceHigh}}}
	e := domain.EvidenciaAsignacionCompetencialV1{
		PersonaRef: s.PersonaRef, PerfilFirmanteRef: s.PerfilFirmanteRef, PerfilActivoFirmanteRef: "prf_0123456789abcdefghijkm",
		CertificadoHuellaSHA256: s.CertificadoHuellaSHA256, UnidadRef: "unidad:secretaria", Cargo: s.Cargo,
		CargoVigenteDesde: ahora.Add(-time.Hour), CargoVigenteHasta: ahora.Add(time.Hour),
		EnlaceOcupante: domain.EnlaceOcupanteCompetencialV1{Referencia: "ocupacion:secretaria", Version: 4,
			HuellaSHA256: strings.Repeat("4", 64), PersonaRef: s.PersonaRef, CargoRef: s.Cargo.Referencia,
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)},
		AccionCompetencial: s.AccionCompetencial, FinalidadCompetencial: s.FinalidadCompetencial,
		RecursoRef: s.Recurso.Referencia, ModuloID: s.Recurso.ModuloID, TipoRecurso: s.Recurso.Tipo,
		Asignacion: domain.AsignacionPerfil{AsignacionID: "firmante-secretaria", Version: 3,
			PerfilActivoRef: "prf_0123456789abcdefghijkm", PrincipalID: s.PersonaRef, VersionRolRef: rol.Referencia(),
			Estado: domain.EstadoAsignacionPerfilActiva,
			Ambitos: []domain.AmbitoPerfil{{Clave: "organizacion", Valores: []string{"organizacion:diputacion"}},
				{Clave: "unidad", Valores: []string{"unidad:secretaria"}}},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "acto:asignacion", EmitidaEn: ahora.Add(-time.Hour)},
		VersionRol: rol,
		ControlVigencia: domain.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 2,
			Estado: domain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "acto:control", ActualizadoEn: ahora.Add(-time.Hour)},
		ActoCompetenciaRef: "acto:competencia", ComprobadaEn: ahora,
	}
	e.RecursoHuellaSHA256, err = s.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	e.AsignacionHuellaSHA256, err = e.Asignacion.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	e.VersionRolHuellaSHA256, err = e.VersionRol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	e.ControlVigenciaHuellaSHA256, err = e.ControlVigencia.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return s, e, ahora
}

func TestAsignacionCompetencialV1SeparaConsultanteFirmanteYPerfiles(t *testing.T) {
	s, e, ahora := asignacionCompetencialPrueba(t)
	if s.Actor.PersonaRef == e.PersonaRef || s.Actor.PerfilActivoRef == e.PerfilActivoFirmanteRef ||
		e.PerfilFirmanteRef == e.PerfilActivoFirmanteRef {
		t.Fatal("la prueba debe distinguir las identidades y los perfiles")
	}
	if err := e.ValidarParaEn(s, ahora); err != nil {
		t.Fatalf("asignacion coherente rechazada: %v", err)
	}
	e.Delegacion = &domain.DelegacionCompetencialV1{Referencia: "delegacion:secretaria", Version: 2,
		HuellaSHA256: strings.Repeat("5", 64), DelegantePersonaRef: s.Actor.PersonaRef, DelegadoPersonaRef: s.PersonaRef,
		CargoRef: s.Cargo.Referencia, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	e.EnlaceOcupante.PersonaRef = e.Delegacion.DelegantePersonaRef
	if err := e.ValidarParaEn(s, ahora); err != nil {
		t.Fatal(err)
	}
	e.Delegacion.VigenteHasta = ahora
	if e.ValidarParaEn(s, ahora) == nil {
		t.Fatal("delegacion vencida aceptada")
	}
}

func TestAsignacionCompetencialV1RechazaCrucesYVigencias(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*domain.SolicitudAsignacionCompetencialV1, *domain.EvidenciaAsignacionCompetencialV1, time.Time)
	}{
		{"persona de otra asignacion", func(s *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			e.Asignacion.PrincipalID = s.Actor.PersonaRef
		}},
		{"perfil activo cruzado", func(s *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			e.Asignacion.PerfilActivoRef = s.Actor.PerfilActivoRef
		}},
		{"perfil circuito confundido", func(_ *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			e.PerfilFirmanteRef = e.PerfilActivoFirmanteRef
		}},
		{"certificado distinto", func(_ *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			e.CertificadoHuellaSHA256 = strings.Repeat("6", 64)
		}},
		{"rol cruzado", func(_ *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			e.Asignacion.VersionRolRef = "rol:otro:v1"
		}},
		{"control cruzado", func(_ *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			e.ControlVigencia.VersionRolRef = "rol:otro:v1"
		}},
		{"control retirado", func(_ *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, ahora time.Time) {
			e.ControlVigencia.Estado = domain.EstadoControlVigenciaVersionRolRetirada
			e.ControlVigencia.ActoRef = "acto:retirada"
			e.ControlVigencia.MotivoCodigo = "retirada"
			e.ControlVigencia.ActualizadoEn = ahora
		}},
		{"huella asignacion alterada", func(_ *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			e.AsignacionHuellaSHA256 = strings.Repeat("7", 64)
		}},
		{"huella rol alterada", func(_ *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			e.VersionRolHuellaSHA256 = strings.Repeat("7", 64)
		}},
		{"huella control alterada", func(_ *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			e.ControlVigenciaHuellaSHA256 = strings.Repeat("7", 64)
		}},
		{"ocupante ajeno", func(s *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			e.EnlaceOcupante.PersonaRef = s.Actor.PersonaRef
		}},
		{"cargo cruzado", func(_ *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			e.Cargo.Version++
		}},
		{"cese ocupante", func(_ *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, ahora time.Time) {
			e.EnlaceOcupante.VigenteHasta = ahora
		}},
		{"caducidad asignacion", func(_ *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, ahora time.Time) {
			e.Asignacion.VigenteHasta = ahora
		}},
		{"ambito ajeno", func(s *domain.SolicitudAsignacionCompetencialV1, _ *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			s.Recurso.Ambitos["unidad"] = "unidad:ajena"
		}},
		{"recurso cruzado con iguales ambitos", func(s *domain.SolicitudAsignacionCompetencialV1, _ *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			s.Recurso.Referencia = "documento:otro"
		}},
		{"evidencia futura", func(_ *domain.SolicitudAsignacionCompetencialV1, e *domain.EvidenciaAsignacionCompetencialV1, ahora time.Time) {
			e.ComprobadaEn = ahora.Add(time.Second)
		}},
		{"actor con otra evidencia", func(s *domain.SolicitudAsignacionCompetencialV1, _ *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			s.Actor.PerfilActivoRef = "prf_0123456789abcdefghijkn"
		}},
		{"sesion ausente", func(s *domain.SolicitudAsignacionCompetencialV1, _ *domain.EvidenciaAsignacionCompetencialV1, _ time.Time) {
			s.Vinculo = domain.VinculoAutenticacionActorV2{}
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			s, e, ahora := asignacionCompetencialPrueba(t)
			caso.cambiar(&s, &e, ahora)
			if !errors.Is(e.ValidarParaEn(s, ahora), domain.ErrEvidenciaAsignacionCompetencialV1Invalida) {
				t.Fatal("evidencia incoherente aceptada")
			}
		})
	}
}

func TestAsignacionCompetencialV1NoSeSerializaNiExponePorLog(t *testing.T) {
	s, e, _ := asignacionCompetencialPrueba(t)
	for _, valor := range []any{s, e} {
		if _, err := json.Marshal(valor); !errors.Is(err, domain.ErrSerializacionAsignacionCompetencialV1Prohibida) {
			t.Fatalf("serializacion aceptada: %v", err)
		}
		if _, err := xml.Marshal(valor); !errors.Is(err, domain.ErrSerializacionAsignacionCompetencialV1Prohibida) {
			t.Fatalf("serializacion XML aceptada: %v", err)
		}
		if texto := fmt.Sprintf("%+v %#v", valor, valor); strings.Contains(texto, s.PersonaRef) || strings.Contains(texto, s.CertificadoHuellaSHA256) {
			t.Fatal("log expone evidencia")
		}
	}
	var reconstruida domain.EvidenciaAsignacionCompetencialV1
	if err := json.Unmarshal([]byte(`{}`), &reconstruida); !errors.Is(err, domain.ErrSerializacionAsignacionCompetencialV1Prohibida) {
		t.Fatal("reconstruccion aceptada")
	}
}

func TestAsignacionCompetencialV1RechazaRolSinCompetenciaAunqueHuellaCoincida(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*domain.VersionRol)
	}{
		{"accion", func(r *domain.VersionRol) { r.Concesiones[0].Accion = "documento.consultar" }},
		{"modulo", func(r *domain.VersionRol) { r.Concesiones[0].ModuloID = "personal" }},
		{"tipo", func(r *domain.VersionRol) { r.Concesiones[0].TipoRecurso = "expediente" }},
		{"finalidad", func(r *domain.VersionRol) { r.Concesiones[0].Finalidades = []string{"consultar"} }},
		{"sin unir concesiones", func(r *domain.VersionRol) {
			otra := r.Concesiones[0]
			r.Concesiones[0].ModuloID = "personal"
			otra.Accion = "documento.consultar"
			r.Concesiones = append(r.Concesiones, otra)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			s, e, ahora := asignacionCompetencialPrueba(t)
			caso.cambiar(&e.VersionRol)
			var err error
			e.VersionRolHuellaSHA256, err = e.VersionRol.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			if !errors.Is(e.ValidarParaEn(s, ahora), domain.ErrEvidenciaAsignacionCompetencialV1Invalida) {
				t.Fatal("rol coherente sin la competencia declarada aceptado")
			}
		})
	}
}
