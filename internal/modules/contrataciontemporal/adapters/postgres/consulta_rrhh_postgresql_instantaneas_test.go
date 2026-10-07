package postgres

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/diagnostico"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

func fixtureInstantaneaCuadroRRHH(t *testing.T, fase string) (salidaCuadroConsultaRRHH, []ports.ResumenExpedienteRRHH, []time.Time) {
	t.Helper()
	publicado := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	faseDesde := time.Date(2026, 9, 29, 9, 0, 0, 0, time.UTC)
	capturada := faseDesde.Add(time.Second)
	borrador := dominiovec.CatalogoConfigurable{
		ID: reglas.CatalogoContratacionTemporal, Version: 1, Revision: 1,
		ModuloID: reglas.ModuloContratacionTemporal, Nombre: "Reglas CT", FuenteRef: reglas.MarcaPaqueteEjemplo,
		MotivoCreacion: "Prueba sintética.",
		Entradas: []dominiovec.EntradaCatalogoConfigurable{{
			Clave: reglas.CTPlazoFiscalizacion, Etiqueta: "Plazo", VigenteDesde: publicado,
			Atributos: map[string]string{
				"origen": "ejemplo", "norma": "Norma sintética", "duda": "Regla de prueba",
				"unidad": "dias_habiles", "cantidad": "10", "inicio": "entrada_fase",
				"computo": "administrativo", "fases": "fiscalizacion",
			},
		}},
		Estado: dominiovec.EstadoCatalogoBorrador, CreadoPor: "prueba", CreadoEn: publicado,
	}
	catalogo, err := borrador.Publicar("revisor", "sin_aprobacion", "Prueba sintética.", publicado)
	if err != nil {
		t.Fatal(err)
	}
	base, huellaBase, err := reglas.CanonicoCatalogoBaseReglas(catalogo)
	if err != nil {
		t.Fatal(err)
	}
	ajustes := map[string]map[string]string{}
	canonicoAjustes, err := reglas.CanonicoAjustes(ajustes)
	if err != nil {
		t.Fatal(err)
	}
	huellaAjustes, err := reglas.HuellaAjustes(ajustes)
	if err != nil {
		t.Fatal(err)
	}
	idAjustes := reglas.CatalogoAjustesDe(catalogo.ID)
	serializar := func(v any) []byte {
		t.Helper()
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	fila := map[string]any{
		"estado": "capturada", "expediente_ref": "expediente:ct:prueba", "version_expediente": 7,
		"fase": fase, "fase_desde": faseDesde,
		"catalogo_base_id": catalogo.ID, "base_version": catalogo.Version,
		"base_huella_sha256": huellaBase, "catalogo_ajustes_id": idAjustes,
		"ajustes_encontrados": false, "ajustes_version": 0,
		"ajustes_huella_sha256": huellaAjustes, "ajustes_vigente_desde": nil,
		"capturada_en": capturada,
	}
	salida := salidaCuadroConsultaRRHH{
		cierre:            salidaCierreConsultaRRHH{generadaEn: capturada.Add(time.Minute)},
		instantaneasRegla: serializar([]any{fila}),
		basesRegla: serializar([]any{map[string]any{
			"catalogo_base_id": catalogo.ID, "base_version": catalogo.Version,
			"base_huella_sha256": huellaBase, "base_canonico": string(base),
		}}),
		ajustesRegla: serializar([]any{map[string]any{
			"catalogo_ajustes_id": idAjustes, "ajustes_version": 0,
			"ajustes_huella_sha256": huellaAjustes, "ajustes_canonico": string(canonicoAjustes),
		}}),
	}
	return salida, []ports.ResumenExpedienteRRHH{{ExpedienteRef: "expediente:ct:prueba", Version: 7, FaseClave: ctdomain.ClaveFase(fase)}}, []time.Time{faseDesde}
}

func TestSalidaCuadroRRHHRehidrataContextoCapturadoYAdmiteFaseSinRegla(t *testing.T) {
	t.Parallel()
	for _, fase := range []string{"fiscalizacion", "analisis"} {
		salida, resumenes, fasesDesde := fixtureInstantaneaCuadroRRHH(t, fase)
		instantaneas, err := salida.instantaneasAlineadas(resumenes, fasesDesde)
		if err != nil || len(instantaneas) != 1 || instantaneas[0] == nil ||
			instantaneas[0].Fase != fase || instantaneas[0].AjustesEncontrados {
			t.Fatalf("instantánea de %s: %#v, %v", fase, instantaneas, err)
		}
	}
}

func TestSalidaCuadroRRHHGrupoConservaSuContextoCapturado(t *testing.T) {
	t.Parallel()
	salida, _, fechas := fixtureInstantaneaCuadroRRHH(t, "fiscalizacion")
	var grupos []map[string]any
	if err := json.Unmarshal(salida.instantaneasRegla, &grupos); err != nil {
		t.Fatal(err)
	}
	delete(grupos[0], "expediente_ref")
	delete(grupos[0], "version_expediente")
	var err error
	salida.plazoContextos, err = json.Marshal(grupos)
	if err != nil {
		t.Fatal(err)
	}
	salida.plazoBases, salida.plazoAjustes = salida.basesRegla, salida.ajustesRegla
	salida.plazoFases, salida.plazoDesde = []string{"fiscalizacion"}, fechas
	contextos, err := salida.instantaneasGrupos()
	if err != nil || len(contextos) != 1 || contextos[0] == nil ||
		contextos[0].CatalogoBaseVersion != 1 || !contextos[0].FaseDesde.Equal(fechas[0]) {
		t.Fatalf("grupo sin regla capturada: %+v, %v", contextos, err)
	}
}

func TestSalidaCuadroRRHHRechazaContextosNoLigados(t *testing.T) {
	t.Parallel()
	original, resumenes, fasesDesde := fixtureInstantaneaCuadroRRHH(t, "fiscalizacion")
	probar := func(nombre string, alterar func(*salidaCuadroConsultaRRHH)) {
		t.Run(nombre, func(t *testing.T) {
			salida := original
			alterar(&salida)
			if _, err := salida.instantaneasAlineadas(resumenes, fasesDesde); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("contexto no confiable aceptado: %v", err)
			}
		})
	}
	probar("base ausente", func(s *salidaCuadroConsultaRRHH) { s.basesRegla = []byte("[]") })
	probar("ajustes ausentes", func(s *salidaCuadroConsultaRRHH) { s.ajustesRegla = []byte("[]") })
	probar("base duplicada", func(s *salidaCuadroConsultaRRHH) {
		var bases []json.RawMessage
		if err := json.Unmarshal(s.basesRegla, &bases); err != nil {
			t.Fatal(err)
		}
		s.basesRegla, _ = json.Marshal([]json.RawMessage{bases[0], bases[0]})
	})
	probar("huella base divergente", func(s *salidaCuadroConsultaRRHH) {
		var bases []map[string]any
		if err := json.Unmarshal(s.basesRegla, &bases); err != nil {
			t.Fatal(err)
		}
		bases[0]["base_huella_sha256"] = "0000000000000000000000000000000000000000000000000000000000000000"
		s.basesRegla, _ = json.Marshal(bases)
	})
	probar("fase distinta", func(s *salidaCuadroConsultaRRHH) {
		s.instantaneasRegla = []byte(`[{"estado":"legado_sin_instantanea","expediente_ref":"expediente:ct:prueba","version_expediente":7,"fase":"otra","fase_desde":"2026-09-29T09:00:00Z"}]`)
		s.basesRegla, s.ajustesRegla = []byte("[]"), []byte("[]")
	})
	probar("expediente distinto", func(s *salidaCuadroConsultaRRHH) {
		s.instantaneasRegla = []byte(`[{"estado":"legado_sin_instantanea","expediente_ref":"expediente:ct:otro","version_expediente":7,"fase":"fiscalizacion","fase_desde":"2026-09-29T09:00:00Z"}]`)
		s.basesRegla, s.ajustesRegla = []byte("[]"), []byte("[]")
	})
	probar("version distinta", func(s *salidaCuadroConsultaRRHH) {
		s.instantaneasRegla = []byte(`[{"estado":"legado_sin_instantanea","expediente_ref":"expediente:ct:prueba","version_expediente":8,"fase":"fiscalizacion","fase_desde":"2026-09-29T09:00:00Z"}]`)
		s.basesRegla, s.ajustesRegla = []byte("[]"), []byte("[]")
	})
	probar("carga excesiva", func(s *salidaCuadroConsultaRRHH) {
		s.basesRegla = make([]byte, maximoBytesContextoPlazosRRHH+1)
	})
}

func TestSalidaCuadroRRHHLegadoNoRecibeCatalogoActual(t *testing.T) {
	t.Parallel()
	salida, resumenes, fasesDesde := fixtureInstantaneaCuadroRRHH(t, "fiscalizacion")
	salida.instantaneasRegla = []byte(`[{"estado":"legado_sin_instantanea","expediente_ref":"expediente:ct:prueba","version_expediente":7,"fase":"fiscalizacion","fase_desde":"2026-09-29T09:00:00Z"}]`)
	salida.basesRegla, salida.ajustesRegla = []byte("[]"), []byte("[]")
	instantaneas, err := salida.instantaneasAlineadas(resumenes, fasesDesde)
	if err != nil || len(instantaneas) != 1 || instantaneas[0] != nil {
		t.Fatalf("legado recibió instantánea: %#v, %v", instantaneas, err)
	}
}

func TestSalidaCuadroRRHHPropagaFalloNominalSinExponerContenido(t *testing.T) {
	t.Parallel()
	salida, resumenes, fasesDesde := fixtureInstantaneaCuadroRRHH(t, "fiscalizacion")
	var bases []map[string]any
	if err := json.Unmarshal(salida.basesRegla, &bases); err != nil {
		t.Fatal(err)
	}
	bases[0]["base_canonico"] = `{persona_confidencial`
	var err error
	salida.basesRegla, err = json.Marshal(bases)
	if err != nil {
		t.Fatal(err)
	}
	_, err = salida.instantaneasAlineadas(resumenes, fasesDesde)
	var fallo *diagnostico.FalloConsultaRRHH
	if !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) ||
		!errors.As(err, &fallo) || fallo.Etapa != diagnostico.EtapaResultadoSQL ||
		fallo.Causa == nil || strings.Contains(err.Error(), "persona_confidencial") {
		t.Fatalf("fallo no nominal o filtrado: %v", err)
	}
}
