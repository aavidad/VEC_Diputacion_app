package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestLecturaExclusionesExigeConcesionYAmbitoExactos(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	sesion, fuente := escenarioLecturaRespuestasLlamamiento(t, ahora)
	ref := "llamamiento:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	capturar := func(unidad, ambito string) error {
		_, err := CapturarLecturaExclusionesOfrecimientoRRHH(context.Background(), fuente,
			sesion.Vinculo, sesion.Resultado, unidad, ambito, "bolsa:prueba", ref)
		return err
	}
	if err := capturar("unidad:prueba", "ambito:prueba"); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("la concesión de respuestas permitió motivos", err)
	}
	fuente.i.VersionRol.Concesiones = append(fuente.i.VersionRol.Concesiones, vecdomain.ConcesionRol{
		Accion: accionLecturaExclusionesOfrecimientoRRHH, ModuloID: "bolsa", TipoRecurso: "llamamiento_bolsa",
		Finalidades:      []string{finalidadLecturaExclusionesOfrecimientoRRHH},
		CamposPermitidos: slices.Clone(camposLecturaExclusionesOfrecimientoRRHH),
		GarantiaMinima:   vecdomain.AuthAssuranceHigh, Obligaciones: []string{"auditar"},
	})
	if err := capturar("unidad:ajena", "ambito:prueba"); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("la unidad ajena permitió motivos", err)
	}
	if err := capturar("unidad:prueba", "ambito:ajeno"); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("el ámbito ajeno permitió motivos", err)
	}
	captura, err := CapturarLecturaExclusionesOfrecimientoRRHH(context.Background(), fuente,
		sesion.Vinculo, sesion.Resultado, "unidad:prueba", "ambito:prueba", "bolsa:prueba", ref)
	if err != nil {
		t.Fatal(err)
	}
	var orden map[string]any
	if json.Unmarshal(captura.Orden, &orden) != nil || orden["accion"] != accionLecturaExclusionesOfrecimientoRRHH ||
		orden["motivo_ref"] != "lectura_exclusiones_llamamiento_correcta" || orden["recurso_ref"] != ref {
		t.Fatalf("captura no ligó acción, motivo y llamamiento: %+v", orden)
	}
	i := len(fuente.i.VersionRol.Concesiones) - 1
	fuente.i.VersionRol.Concesiones[i].CamposPermitidos = append(slices.Clone(camposLecturaExclusionesOfrecimientoRRHH), "dato_ajeno")
	if err := capturar("unidad:prueba", "ambito:prueba"); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("concesión ampliada permitió motivos", err)
	}
}

type lectorExclusionesPrueba struct {
	pagina puertosbolsa.PaginaExclusionesOfrecimientoRRHH
	err    error
	desde  int64
}

func (l *lectorExclusionesPrueba) ConsultarExclusionesOfrecimientoRRHH(_ context.Context, _, _ string, desde int64) (puertosbolsa.PaginaExclusionesOfrecimientoRRHH, error) {
	l.desde = desde
	return l.pagina, l.err
}

func TestSeguimientoExclusionesConservaPermisoYFalloAislados(t *testing.T) {
	q, ok := consultaCandidatos("llamamiento=llamamiento:prueba_123&exclusiones_desde=7")
	if !ok || !q.exclusionesSolicitadas || q.exclusionesDesde != 7 {
		t.Fatalf("consulta de exclusiones inválida: %+v, %v", q, ok)
	}
	if _, ok := consultaCandidatos("exclusiones_desde=0"); ok {
		t.Fatal("aceptó cursor sin llamamiento")
	}
	if _, ok := consultaCandidatos("llamamiento=llamamiento:prueba_123&exclusiones_desde=-1"); ok {
		t.Fatal("aceptó cursor negativo")
	}

	lector := &lectorExclusionesPrueba{err: vecdomain.ErrAutorizacionDenegada}
	h := &bolsasRRHHDesarrollo{exclusionesOrigen: lector}
	denegada := h.consultarExclusionesOrigen(context.Background(), "bolsa:prueba", q.llamamiento, q.exclusionesDesde)
	if denegada["estado"] != "sin_permiso" || len(denegada) != 1 || lector.desde != 7 {
		t.Fatalf("denegación expuso contenido o perdió cursor: %+v", denegada)
	}
	lector.err = errors.New("lectura fallida")
	fallida := h.consultarExclusionesOrigen(context.Background(), "bolsa:prueba", q.llamamiento, 0)
	if fallida["estado"] != "no_disponible" || fallida["error_estado"] != "lectura_no_disponible" || len(fallida) != 2 {
		t.Fatalf("lectura fallida no quedó aislada: %+v", fallida)
	}
	lector.err = nil
	lector.pagina = puertosbolsa.PaginaExclusionesOfrecimientoRRHH{
		BolsaRef: "bolsa:prueba", LlamamientoRef: q.llamamiento,
		Exclusiones: []puertosbolsa.ExclusionOfrecimientoRRHH{}, TotalExclusiones: 0,
		AuditoriaRef: "auditoria:prueba", CorrelacionRef: "correlacion:prueba", ConsultadaEn: time.Now().UTC(),
	}
	disponible := h.consultarExclusionesOrigen(context.Background(), "bolsa:prueba", q.llamamiento, 0)
	if disponible["estado"] != "disponible" || disponible["pagina"] == nil {
		t.Fatalf("no entregó página auditada: %+v", disponible)
	}
}
