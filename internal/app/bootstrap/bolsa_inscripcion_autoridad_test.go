package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type decisorLecturaInscripcionPrueba struct {
	decision DecisionLecturaActualInscripcionBolsa
	err      error
	llamadas int
}

func (d *decisorLecturaInscripcionPrueba) DecidirLecturaActual(_ context.Context, _ contextoSeguridadComunDesarrollo, _ AcreditacionSesionInscripcionBolsa, _, _ string, _ inscripcion.Filtro) (DecisionLecturaActualInscripcionBolsa, error) {
	d.llamadas++
	return d.decision, d.err
}

func TestAutoridadInscripcionDeniegaSinFuenteActual(t *testing.T) {
	if _, err := NuevaAutoridadInscripcionBolsa(ConfiguracionAutoridadInscripcionBolsa{}); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("sin fuente común: %v", err)
	}
	a := &autoridadNominalInscripcionBolsa{}
	if _, err := a.CapturarLectura(context.Background(), contextoSeguridadComunDesarrollo{}, AcreditacionSesionInscripcionBolsa{}, inscripcion.AccionListarPropias, "inscripciones_propias_x", inscripcion.Filtro{Limite: 20}); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("lectura sin autoridad: %v", err)
	}
	if _, err := a.AutorizarEscritura(context.Background(), contextoSeguridadComunDesarrollo{}, AcreditacionSesionInscripcionBolsa{}, inscripcion.AccionPresentar, "solicitud_inscripcion_x", []byte("{}")); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("escritura sin V3: %v", err)
	}
}

func TestAutoridadInscripcionLecturaLigaDecisionYSesion(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	s, _ := contextoInscripcionCanalPrueba(t, ahora, true, false)
	acreditacion := acreditacionSesionInscripcionPrueba(t, s, "externa_personal", ahora)
	v, err := s.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	filtro := inscripcion.Filtro{Limite: 20}
	recurso, err := inscripcion.RecursoLectura(inscripcion.AccionListarPropias, s.Resultado.Contexto.PersonaRef, "es", filtro, "")
	if err != nil {
		t.Fatal(err)
	}
	d := DecisionLecturaActualInscripcionBolsa{
		Concedida: true, PersonaRef: s.Resultado.Contexto.PersonaRef, PerfilRef: v.PerfilActivoRef,
		CuentaRef: v.CuentaRef, SesionRef: v.SesionRef, AutenticacionRef: v.AutenticacionRef,
		CertificadoHuellaSHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		Canal:                   "externa_personal", Accion: inscripcion.AccionListarPropias, RecursoRef: recurso,
		Finalidad: "consulta_inscripcion_propia", CorrelacionRef: "correlacion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RevisionPermisos: 1, HuellaInstantaneaSHA256: "0000000000000001000000000000000000000000000000000000000000000000",
		Campos: []string{"solicitud_ref"}, Filtro: filtro, EmitidaEn: ahora, ValidaHasta: ahora.Add(time.Second),
	}
	decisor := &decisorLecturaInscripcionPrueba{decision: d}
	a := &autoridadNominalInscripcionBolsa{c: ConfiguracionAutoridadInscripcionBolsa{Lectura: decisor, Reloj: relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora}, Lecturas: map[ClaveOperacionInscripcionBolsa]DescriptorLecturaInscripcionBolsa{
		{inscripcion.AccionListarPropias, "externa_personal"}: {Accion: inscripcion.AccionListarPropias, Finalidad: d.Finalidad, Campos: []string{"solicitud_ref"}},
	}}}
	captura, err := a.CapturarLectura(context.Background(), s, acreditacion, d.Accion, recurso, filtro)
	if err != nil || captura.RecursoRef != recurso || captura.RevisionPermisos != 1 ||
		captura.HuellaInstantaneaSHA256 != d.HuellaInstantaneaSHA256 || captura.CorrelacionRef != d.CorrelacionRef {
		t.Fatalf("captura exacta: %+v, %v", captura, err)
	}
	casos := []struct {
		nombre  string
		cambiar func(*DecisionLecturaActualInscripcionBolsa)
	}{
		{"permiso ausente", func(x *DecisionLecturaActualInscripcionBolsa) { x.Concedida = false }},
		{"perfil cambiado", func(x *DecisionLecturaActualInscripcionBolsa) { x.PerfilRef = "perfil_ajeno" }},
		{"certificado invalido", func(x *DecisionLecturaActualInscripcionBolsa) {
			x.CertificadoHuellaSHA256 = "huella_no_canonica"
		}},
		{"recurso ajeno", func(x *DecisionLecturaActualInscripcionBolsa) { x.RecursoRef = "inscripciones_propias_otras" }},
		{"filtro cambiado", func(x *DecisionLecturaActualInscripcionBolsa) { x.Filtro.Limite = 10 }},
		{"campos adicionales", func(x *DecisionLecturaActualInscripcionBolsa) { x.Campos = []string{"solicitud_ref", "dni"} }},
		{"huella ajena", func(x *DecisionLecturaActualInscripcionBolsa) {
			x.HuellaInstantaneaSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}},
		{"sesion vencida", func(x *DecisionLecturaActualInscripcionBolsa) { x.ValidaHasta = ahora }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutada := d
			caso.cambiar(&mutada)
			decisor.decision = mutada
			if _, err := a.CapturarLectura(context.Background(), s, acreditacion, d.Accion, recurso, filtro); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
				t.Fatalf("decisión alterada aceptada: %v", err)
			}
		})
	}
	decisor.err = errors.New("revocada")
	if _, err := a.CapturarLectura(context.Background(), s, acreditacion, d.Accion, recurso, filtro); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("sesión revocada aceptada: %v", err)
	}
	if decisor.llamadas != len(casos)+2 {
		t.Fatalf("decisiones consultadas: %d", decisor.llamadas)
	}
	decisor.err = nil
	decisor.decision = d
	acreditacionAjena := acreditacion
	acreditacionAjena.CertificadoHuellaSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := a.CapturarLectura(context.Background(), s, acreditacionAjena, d.Accion, recurso, filtro); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("certificado ajeno aceptado: %v", err)
	}
	if _, err := a.CapturarLectura(context.Background(), s, acreditacion, inscripcion.AccionListarRRHH, recurso, filtro); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("RRHH exterior: %v", err)
	}
	if v.Superficie != vecdomain.SuperficieAutenticacionExternaPersonalV1 {
		t.Fatal("fixture sin canal externo")
	}
}

func TestAutoridadInscripcionRechazaMaterialEscrituraAjenoAntesDeV3(t *testing.T) {
	p := inscripcion.Presentacion{ConvocatoriaRef: "cv1_prueba_v1", CategoriaRef: "categoria_prueba", CatalogoVersion: 1, ClaveIdempotencia: "clave_inscripcion_prueba_01"}
	material, huella, err := inscripcion.MaterialPresentacion(p)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := inscripcion.ReferenciaSolicitud("persona_prueba", p)
	if err != nil {
		t.Fatal(err)
	}
	atributos, ok := atributosMaterialEscrituraInscripcion(inscripcion.AccionPresentar, "persona_prueba", ref, material)
	if !ok || atributos["material_sha256"] != huella || atributos["convocatoria_ref"] != p.ConvocatoriaRef ||
		atributos["categoria_ref"] != p.CategoriaRef || atributos["catalogo_version"] != "1" {
		t.Fatal("canon válido rechazado")
	}
	canon, err := recursoCanonicoDesdeMaterialInscripcion(inscripcion.AccionPresentar, material, huella,
		map[string]string{"empleado_ref": "emp_persona_prueba"})
	var v struct {
		Ambitos   map[string]string `json:"ambitos"`
		Atributos map[string]string `json:"atributos"`
	}
	if err != nil || json.Unmarshal(canon, &v) != nil || len(v.Ambitos) != 1 ||
		v.Ambitos["empleado_ref"] != "emp_persona_prueba" || len(v.Atributos) != 4 ||
		v.Atributos["convocatoria_ref"] != p.ConvocatoriaRef || v.Atributos["material_sha256"] != huella {
		t.Fatalf("recurso nominal no canónico: %+v %v", v, err)
	}
	for nombre, caso := range map[string]struct {
		persona, ref string
		material     []byte
	}{
		"persona ajena":    {"persona_ajena", ref, material},
		"referencia ajena": {"persona_prueba", "solicitud_inscripcion_otra", material},
		"material ajeno":   {"persona_prueba", ref, []byte(`{"esquema":"vec.bolsa.inscripcion.presentar.v1","convocatoria_ref":"cv1_otro_v1"}`)},
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, ok := atributosMaterialEscrituraInscripcion(inscripcion.AccionPresentar, caso.persona, caso.ref, caso.material); ok {
				t.Fatal("material ajeno aceptado")
			}
		})
	}
}

func TestAutoridadInscripcionAmbitoEmpleadoDelContextoYExternoSinCandidato(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	s, empleado := contextoInscripcionCanalPrueba(t, ahora, false, true)
	acreditacion := acreditacionSesionInscripcionPrueba(t, s, "interna_corporativa", ahora)
	a := &autoridadNominalInscripcionBolsa{}
	ambitos, err := a.ambitosEscrituraInscripcion(context.Background(), s, acreditacion, inscripcion.AccionPresentar, "solicitud_inscripcion_prueba", ahora)
	if err != nil || len(ambitos) != 1 || ambitos["empleado_ref"] != empleado {
		t.Fatalf("ámbito empleado: %+v %v", ambitos, err)
	}
	sExterna, _ := contextoInscripcionCanalPrueba(t, ahora, true, false)
	acreditacionExterna := acreditacionSesionInscripcionPrueba(t, sExterna, "externa_personal", ahora)
	if _, err := a.ambitosEscrituraInscripcion(context.Background(), sExterna, acreditacionExterna, inscripcion.AccionPresentar, "solicitud_inscripcion_prueba", ahora); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("candidato no acreditado obtuvo ámbito: %v", err)
	}
}

func TestAutoridadInscripcionAtributosDecisionEIncorporacionExactos(t *testing.T) {
	ref := "solicitud_inscripcion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	clave := "clave_inscripcion_prueba_01"
	decision := inscripcion.Decision{SolicitudRef: ref, Tipo: "admitir", VersionEsperada: 1, ClaveIdempotencia: clave}
	material, huella, err := inscripcion.MaterialDecision(decision)
	if err != nil {
		t.Fatal(err)
	}
	attrs, ok := atributosMaterialEscrituraInscripcion(inscripcion.AccionDecidir, "persona_prueba", ref, material)
	if !ok || len(attrs) != 4 || attrs["material_sha256"] != huella || attrs["decision"] != "admitir" || attrs["version_esperada"] != "1" {
		t.Fatalf("atributos decidir: %+v", attrs)
	}
	decision.Tipo, decision.MotivoCodigo = "rechazar", "motivo_prueba"
	material, _, err = inscripcion.MaterialDecision(decision)
	if err != nil {
		t.Fatal(err)
	}
	attrs, ok = atributosMaterialEscrituraInscripcion(inscripcion.AccionDecidir, "persona_prueba", ref, material)
	if !ok || len(attrs) != 5 || attrs["motivo_codigo"] != "motivo_prueba" {
		t.Fatalf("motivo perdido: %+v", attrs)
	}
	incorporacion := inscripcion.Incorporacion{SolicitudRef: ref, EvidenciaRef: "evidencia_prueba", VersionEsperada: 2, ClaveIdempotencia: clave}
	material, _, err = inscripcion.MaterialIncorporacion(incorporacion)
	if err != nil {
		t.Fatal(err)
	}
	attrs, ok = atributosMaterialEscrituraInscripcion(inscripcion.AccionIncorporar, "persona_prueba", ref, material)
	if !ok || len(attrs) != 4 || attrs["evidencia_ref"] != incorporacion.EvidenciaRef || attrs["version_esperada"] != "2" {
		t.Fatalf("atributos incorporar: %+v", attrs)
	}
}

func TestAutoridadInscripcionDieciseisDescriptoresPorCanal(t *testing.T) {
	claves := append(clavesLecturaInscripcionBolsa(), clavesEscrituraInscripcionBolsa()...)
	if len(claves) != 16 {
		t.Fatalf("claves = %d", len(claves))
	}
	vistas := make(map[ClaveOperacionInscripcionBolsa]struct{}, len(claves))
	for _, clave := range claves {
		if _, duplicada := vistas[clave]; duplicada {
			t.Fatalf("clave duplicada: %+v", clave)
		}
		vistas[clave] = struct{}{}
		tipo, finalidad, ok := tipoFinalidadInscripcionBolsa(clave)
		if !ok || tipo == "" || finalidad == "" {
			t.Fatalf("descriptor ausente: %+v", clave)
		}
		if clave.Canal == "interna_corporativa" && !accionRRHHInscripcion(clave.Accion) && !strings.HasSuffix(tipo, "_empleado") {
			t.Fatalf("empleado recibió tipo externo: %+v %s", clave, tipo)
		}
		if clave.Canal == "externa_personal" && strings.HasSuffix(tipo, "_empleado") {
			t.Fatalf("externo recibió tipo empleado: %+v %s", clave, tipo)
		}
	}
}
