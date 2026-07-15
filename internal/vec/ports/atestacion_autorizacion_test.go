package ports

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestSolicitudFirmaAtestacionEsInmutableYQuedaLigada(t *testing.T) {
	cabecera := cabeceraFirmaAtestacionPrueba()
	decision := decisionFirmaAtestacionPrueba(t, "decision:firma:1")
	solicitud, err := NuevaSolicitudFirmaAtestacionAutorizacionV1(cabecera, decision)
	if err != nil {
		t.Fatalf("crear solicitud: %v", err)
	}
	mensaje, err := solicitud.Mensaje()
	if err != nil {
		t.Fatalf("obtener mensaje: %v", err)
	}
	original := append([]byte(nil), mensaje...)
	mensaje[0] ^= 0xff
	otro, err := solicitud.Mensaje()
	if err != nil || !bytes.Equal(otro, original) {
		t.Fatalf("el llamador modifico el mensaje interno: err=%v", err)
	}
	huella, err := solicitud.HuellaMensajeSHA256()
	if err != nil || len(huella) != 64 {
		t.Fatalf("huella invalida: %q, %v", huella, err)
	}
	referencia, err := solicitud.ReferenciaDecision()
	if err != nil || referencia != decision.DecisionRef {
		t.Fatalf("referencia no ligada: %q, %v", referencia, err)
	}
	recuperada, err := solicitud.Cabecera()
	if err != nil || recuperada != cabecera {
		t.Fatalf("cabecera no ligada: %+v, %v", recuperada, err)
	}
}

func TestResultadoYAtestacionNoPermitenSustituirSolicitudOCabecera(t *testing.T) {
	instante := time.Date(2026, 7, 15, 12, 0, 0, 123_000, time.UTC)
	primera, err := NuevaSolicitudFirmaAtestacionAutorizacionV1(
		cabeceraFirmaAtestacionPrueba(), decisionFirmaAtestacionPrueba(t, "decision:firma:1"),
	)
	if err != nil {
		t.Fatalf("primera solicitud: %v", err)
	}
	segunda, err := NuevaSolicitudFirmaAtestacionAutorizacionV1(
		cabeceraFirmaAtestacionPrueba(), decisionFirmaAtestacionPrueba(t, "decision:firma:2"),
	)
	if err != nil {
		t.Fatalf("segunda solicitud: %v", err)
	}
	firma := []byte{1, 2, 3, 4}
	resultado, err := NuevoResultadoFirmaAtestacionAutorizacionV1(
		primera, firma, "evidencia:firma:1", instante,
	)
	if err != nil {
		t.Fatalf("crear resultado: %v", err)
	}
	firma[0] = 9
	recuperada, err := resultado.Firma()
	if err != nil || recuperada[0] != 1 {
		t.Fatalf("la firma compartio memoria de entrada: %v, %v", recuperada, err)
	}
	recuperada[0] = 8
	otra, _ := resultado.Firma()
	if otra[0] != 1 {
		t.Fatal("la firma compartio memoria de salida")
	}
	if resultado.ValidarPara(segunda) == nil {
		t.Fatal("un resultado de otra decision fue aceptado")
	}
	atestacion, err := NuevaAtestacionAutorizacionV1(primera, resultado)
	if err != nil || atestacion.ValidarPara(primera) != nil {
		t.Fatalf("atestacion valida: %+v, %v", atestacion, err)
	}
	atestacion.cabecera.Audiencia = "vec/otra"
	if atestacion.ValidarPara(primera) == nil {
		t.Fatal("una audiencia sustituida fue aceptada")
	}
}

func TestSolicitudYResultadoFirmaAtestacionRechazanManipulaciones(t *testing.T) {
	solicitud, err := NuevaSolicitudFirmaAtestacionAutorizacionV1(
		cabeceraFirmaAtestacionPrueba(), decisionFirmaAtestacionPrueba(t, "decision:firma:1"),
	)
	if err != nil {
		t.Fatalf("crear solicitud: %v", err)
	}
	mutaciones := []struct {
		nombre string
		mutar  func(*SolicitudFirmaAtestacionAutorizacionV1)
	}{
		{"cabecera", func(s *SolicitudFirmaAtestacionAutorizacionV1) { s.cabecera.ClaveID = "clave:otra" }},
		{"mensaje", func(s *SolicitudFirmaAtestacionAutorizacionV1) { s.mensaje[0] ^= 1 }},
		{"longitud", func(s *SolicitudFirmaAtestacionAutorizacionV1) { s.mensaje[len(s.mensaje)-1] ^= 1 }},
		{"huella", func(s *SolicitudFirmaAtestacionAutorizacionV1) { s.huellaMensaje = strings.Repeat("0", 64) }},
		{"referencia", func(s *SolicitudFirmaAtestacionAutorizacionV1) { s.referenciaDecision = "" }},
	}
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			candidata := solicitud
			candidata.mensaje = append([]byte(nil), solicitud.mensaje...)
			caso.mutar(&candidata)
			if candidata.Validar() == nil {
				t.Fatal("solicitud manipulada aceptada")
			}
		})
	}

	if _, err := NuevoResultadoFirmaAtestacionAutorizacionV1(
		solicitud, nil, "evidencia:firma:1", time.Now().UTC().Truncate(time.Microsecond),
	); !errors.Is(err, ErrResultadoFirmaAtestacionInvalido) {
		t.Fatalf("firma vacia aceptada: %v", err)
	}
	if _, err := NuevoResultadoFirmaAtestacionAutorizacionV1(
		solicitud, []byte{1}, "evidencia:firma:1",
		time.Now().UTC().Truncate(time.Microsecond).Add(time.Nanosecond),
	); !errors.Is(err, ErrResultadoFirmaAtestacionInvalido) {
		t.Fatalf("instante no canonico aceptado: %v", err)
	}
	if _, err := NuevoResultadoFirmaAtestacionAutorizacionV1(
		solicitud, []byte{1}, "evidencia:firma:1", time.Date(10_000, 1, 1, 0, 0, 0, 0, time.UTC),
	); !errors.Is(err, ErrResultadoFirmaAtestacionInvalido) {
		t.Fatalf("instante fuera del intervalo interoperable aceptado: %v", err)
	}
}

func TestFirmanteAtestacionesAutorizacionV1ConservaContrato(t *testing.T) {
	var _ FirmanteAtestacionesAutorizacionV1 = firmanteAtestacionContratoPrueba{}
}

type firmanteAtestacionContratoPrueba struct{}

func (firmanteAtestacionContratoPrueba) FirmarAtestacionAutorizacionV1(
	context.Context,
	SolicitudFirmaAtestacionAutorizacionV1,
) (ResultadoFirmaAtestacionAutorizacionV1, error) {
	return ResultadoFirmaAtestacionAutorizacionV1{}, nil
}

func cabeceraFirmaAtestacionPrueba() domain.CabeceraAtestacionAutorizacionV1 {
	return domain.CabeceraAtestacionAutorizacionV1{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV1,
		Suite:          "VEC-AD-PRUEBA-1",
		ClaveID:        "clave:prueba:2026-01",
		Audiencia:      "vec/pruebas/autorizacion",
	}
}

func decisionFirmaAtestacionPrueba(t *testing.T, referencia string) domain.DecisionAutorizacion {
	t.Helper()
	evaluadas := []string{"politica:ambito:v1", "politica:horario:v1"}
	huellas := map[string]string{evaluadas[0]: strings.Repeat("a", 64), evaluadas[1]: strings.Repeat("b", 64)}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(evaluadas, huellas)
	if err != nil {
		t.Fatalf("huella de catalogo: %v", err)
	}
	emitida := time.Date(2026, 7, 15, 11, 0, 0, 0, time.UTC)
	decision := domain.DecisionAutorizacion{
		DecisionRef: referencia, Concedida: true, Codigo: "concedida",
		PrincipalID: personaVinculoPuertoPrueba, PerfilActivoRef: perfilVinculoPuertoPrueba, Accion: "bolsa.merito.revisar",
		RecursoRef: "merito:1", ModuloID: "bolsa", TipoRecurso: "merito",
		ContextoRecursoHuellaSHA256: strings.Repeat("c", 64), Finalidad: "gestion_bolsa",
		CorrelacionRef:            "correlacion:1",
		VinculoAutenticacionActor: vinculoAutenticacionActorPuertoPrueba(t, emitida),
		AsignacionRef:             "asignacion:1",
		AsignacionHuellaSHA256:    strings.Repeat("d", 64), VersionRolRef: "rol:1:v1",
		VersionRolHuellaSHA256: strings.Repeat("e", 64), ControlVigenciaVersionRolRef: "rol:1:v1",
		ControlVigenciaVersionRolRevision: 1, ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("f", 64),
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasRefs: evaluadas, PoliticasEvaluadasHuellasSHA256: huellas,
		PoliticasRefs: []string{evaluadas[0]}, PoliticasHuellasSHA256: map[string]string{evaluadas[0]: huellas[evaluadas[0]]},
		GarantiaMinima: domain.AuthAssuranceHigh, CamposPermitidos: []string{"estado"},
		Obligaciones: []string{"registrar_acceso"}, EmitidaEn: emitida, ValidaHasta: emitida.Add(time.Minute),
	}
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision de firma invalida: %v", err)
	}
	return decision
}
