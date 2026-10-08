package bootstrap

import (
	"context"
	"errors"
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

func (d *decisorLecturaInscripcionPrueba) DecidirLecturaActual(_ context.Context, _ contextoSeguridadComunDesarrollo, _, _ string, _ inscripcion.Filtro) (DecisionLecturaActualInscripcionBolsa, error) {
	d.llamadas++
	return d.decision, d.err
}

func TestAutoridadInscripcionDeniegaSinFuenteActual(t *testing.T) {
	if _, err := NuevaAutoridadInscripcionBolsa(ConfiguracionAutoridadInscripcionBolsa{}); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatalf("sin fuente común: %v", err)
	}
	a := &autoridadNominalInscripcionBolsa{}
	if _, err := a.CapturarLectura(context.Background(), contextoSeguridadComunDesarrollo{}, inscripcion.AccionListarPropias, "inscripciones_propias_x", inscripcion.Filtro{Limite: 20}); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("lectura sin autoridad: %v", err)
	}
	if _, err := a.AutorizarEscritura(context.Background(), contextoSeguridadComunDesarrollo{}, inscripcion.AccionPresentar, "solicitud_inscripcion_x", []byte("{}"), []byte("{}")); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("escritura sin V3: %v", err)
	}
}

func TestAutoridadInscripcionLecturaLigaDecisionYSesion(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	s, _ := contextoInscripcionCanalPrueba(t, ahora, true, false)
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
		RevisionPermisos: 1, Campos: []string{"solicitud_ref"}, Filtro: filtro, EmitidaEn: ahora, ValidaHasta: ahora.Add(time.Second),
	}
	decisor := &decisorLecturaInscripcionPrueba{decision: d}
	a := &autoridadNominalInscripcionBolsa{c: ConfiguracionAutoridadInscripcionBolsa{Lectura: decisor, Reloj: relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora}, Lecturas: map[string]DescriptorLecturaInscripcionBolsa{
		inscripcion.AccionListarPropias: {Accion: inscripcion.AccionListarPropias, Finalidad: d.Finalidad, Campos: []string{"solicitud_ref"}},
	}}}
	captura, err := a.CapturarLectura(context.Background(), s, d.Accion, recurso, filtro)
	if err != nil || captura.RecursoRef != recurso || captura.RevisionPermisos != 1 || captura.CorrelacionRef != d.CorrelacionRef {
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
		{"sesion vencida", func(x *DecisionLecturaActualInscripcionBolsa) { x.ValidaHasta = ahora }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutada := d
			caso.cambiar(&mutada)
			decisor.decision = mutada
			if _, err := a.CapturarLectura(context.Background(), s, d.Accion, recurso, filtro); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
				t.Fatalf("decisión alterada aceptada: %v", err)
			}
		})
	}
	decisor.err = errors.New("revocada")
	if _, err := a.CapturarLectura(context.Background(), s, d.Accion, recurso, filtro); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
		t.Fatalf("sesión revocada aceptada: %v", err)
	}
	if decisor.llamadas != len(casos)+2 {
		t.Fatalf("decisiones consultadas: %d", decisor.llamadas)
	}
	decisor.err = nil
	decisor.decision = d
	if _, err := a.CapturarLectura(context.Background(), s, inscripcion.AccionListarRRHH, recurso, filtro); !errors.Is(err, inscripcion.ErrAccesoDenegado) {
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
	rc, err := inscripcion.RecursoPresentacion(p, huella)
	if err != nil {
		t.Fatal(err)
	}
	if !materialEscrituraInscripcionExacto(inscripcion.AccionPresentar, "persona_prueba", ref, material, rc) {
		t.Fatal("canon válido rechazado")
	}
	for nombre, caso := range map[string]struct {
		persona, ref      string
		material, recurso []byte
	}{
		"persona ajena":    {"persona_ajena", ref, material, rc},
		"referencia ajena": {"persona_prueba", "solicitud_inscripcion_otra", material, rc},
		"recurso ajeno":    {"persona_prueba", ref, material, []byte(`{"ambitos":{},"atributos":{}}`)},
		"material ajeno":   {"persona_prueba", ref, []byte(`{"esquema":"vec.bolsa.inscripcion.presentar.v1","convocatoria_ref":"cv1_otro_v1"}`), rc},
	} {
		t.Run(nombre, func(t *testing.T) {
			if materialEscrituraInscripcionExacto(inscripcion.AccionPresentar, caso.persona, caso.ref, caso.material, caso.recurso) {
				t.Fatal("material ajeno aceptado")
			}
		})
	}
}
