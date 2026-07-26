package ports_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestComposicionVisualRRHHValidaContratoGobernadoMinimizado(t *testing.T) {
	t.Parallel()
	entorno := entornoComposicionVisualPrueba(t)
	if err := entorno.composicion.ValidarPara(entorno.orden); err != nil {
		t.Fatalf("composición válida: %v", err)
	}
	contenido, err := json.Marshal(entorno.composicion)
	if err != nil {
		t.Fatalf("serializar proyección pública: %v", err)
	}
	texto := string(contenido)
	for _, prohibido := range []string{
		"lectura:visual:001", "auditoria:visual:001",
		"decision:visual:001", "actor:rrhh:001", "sesion:rrhh:001",
		"perfil:rrhh:001", "organizacion:diputacion-granada",
		"responsable", "dni", "roles",
	} {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("la proyección filtra %q: %s", prohibido, texto)
		}
	}
	for nombre, material := range map[string]any{
		"solicitud": entorno.solicitud,
		"capacidad": entorno.capacidad,
		"orden":     entorno.orden,
		"recibo":    entorno.composicion.Lectura,
	} {
		if serializado, err := json.Marshal(material); !errors.Is(
			err, ports.ErrMaterialComposicionVisualRRHHSensible,
		) || serializado != nil {
			t.Fatalf("%s serializable: %q, %v", nombre, serializado, err)
		}
		if representacion := fmt.Sprintf("%v %#v", material, material); strings.Contains(representacion, "decision:visual:001") {
			t.Fatalf("%s filtra autoridad: %s", nombre, representacion)
		}
	}
	atestacion, err := ports.NuevaSolicitudAtestacionPublicacionesVisualesRRHH(
		entorno.orden, entorno.composicion,
	)
	if err != nil {
		t.Fatalf("solicitud de atestación: %v", err)
	}
	if serializado, err := json.Marshal(atestacion); !errors.Is(
		err, ports.ErrMaterialComposicionVisualRRHHSensible,
	) || serializado != nil {
		t.Fatalf("atestación serializable: %q, %v", serializado, err)
	}
	publicaciones := atestacion.Publicaciones()
	publicaciones[0] = ports.PublicacionAtestableVisualRRHH{}
	if atestacion.Publicaciones()[0].Referencia() !=
		entorno.composicion.Flujo.Referencia {
		t.Fatal("la solicitud de atestación comparte publicaciones mutables")
	}
}

func TestCapacidadComposicionVisualQuedaLigadaAVersionAmbitoYVigencia(
	t *testing.T,
) {
	t.Parallel()
	entorno := entornoComposicionVisualPrueba(t)
	otraVersion, err := ports.NuevaSolicitudComposicionVisualRRHH(
		entorno.solicitud.FlujoRef(), entorno.solicitud.FlujoVersion()+1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ports.NuevaOrdenConsultaComposicionVisualRRHH(
		entorno.contexto, entorno.capacidad, entorno.vocabulario,
		otraVersion, entorno.ahora,
	); !errors.Is(err, ports.ErrOrdenComposicionVisualRRHHInvalida) {
		t.Fatalf("capacidad aceptada para otra versión: %v", err)
	}
	otroContexto := contextoPuertosRRHHConMarcas(
		t, entorno.ahora, "a", "b", entorno.contexto.OrganizacionRef(),
	)
	if _, err = ports.NuevaOrdenConsultaComposicionVisualRRHH(
		otroContexto, entorno.capacidad, entorno.vocabulario,
		entorno.solicitud, entorno.ahora,
	); !errors.Is(err, ports.ErrOrdenComposicionVisualRRHHInvalida) {
		t.Fatalf("capacidad aceptada para otro perfil: %v", err)
	}
	otraOrganizacion := contextoPuertosRRHHConMarcas(
		t, entorno.ahora, "a", "a", "organizacion:otra",
	)
	if _, err = ports.NuevaOrdenConsultaComposicionVisualRRHH(
		otraOrganizacion, entorno.capacidad, entorno.vocabulario,
		entorno.solicitud, entorno.ahora,
	); !errors.Is(err, ports.ErrOrdenComposicionVisualRRHHInvalida) {
		t.Fatalf("capacidad aceptada para otra organización: %v", err)
	}
	if _, err = ports.NuevoReciboComposicionVisualRRHH(
		"lectura:visual:002", "auditoria:visual:002", entorno.orden,
		strings.Repeat("a", 64), entorno.capacidad.ValidaHasta(),
	); !errors.Is(err, ports.ErrResultadoComposicionVisualRRHHNoConfiable) {
		t.Fatalf("capacidad aceptada al caducar: %v", err)
	}
}

func TestComposicionVisualRechazaAdulteracionObsolescenciaYDuplicados(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre string
		mutar  func(*ports.ComposicionVisualRRHH)
	}{
		{"flujo_obsoleto", func(c *ports.ComposicionVisualRRHH) {
			c.Flujo.VigenteHasta = c.GeneradaEn
		}},
		{"i18n_invalida", func(c *ports.ComposicionVisualRRHH) {
			c.Flujo.Tareas[0].ClaveI18n = "Crear solicitud"
		}},
		{"fase_duplicada", func(c *ports.ComposicionVisualRRHH) {
			c.Flujo.Fases = append(c.Flujo.Fases, c.Flujo.Fases[0])
		}},
		{"panel_duplicado", func(c *ports.ComposicionVisualRRHH) {
			c.Flujo.Paneles = append(c.Flujo.Paneles, c.Flujo.Paneles[0])
		}},
		{"orden_panel_duplicado", func(c *ports.ComposicionVisualRRHH) {
			panel := c.Flujo.Paneles[0]
			panel.Referencia = "panel:solicitud:duplicado"
			c.Flujo.Paneles = append(c.Flujo.Paneles, panel)
			c.Flujo.Tareas[0].Paneles = append(
				c.Flujo.Tareas[0].Paneles, panel.Referencia,
			)
			c.Flujo.Huella, _ =
				ports.CalcularHuellaDefinicionFlujoVisualRRHH(c.Flujo)
		}},
		{"opcion_duplicada", func(c *ports.ComposicionVisualRRHH) {
			c.Catalogos[0].Opciones = append(
				c.Catalogos[0].Opciones, c.Catalogos[0].Opciones[0],
			)
		}},
		{"catalogo_no_publicado", func(c *ports.ComposicionVisualRRHH) {
			c.Flujo.Paneles[0].Campos[0].CatalogoVersion++
		}},
		{"seleccion_sin_catalogo", func(c *ports.ComposicionVisualRRHH) {
			c.Flujo.Paneles[0].Campos[0].CatalogoRef = ""
			c.Flujo.Paneles[0].Campos[0].CatalogoVersion = 0
		}},
		{"operacion_no_soportada", func(c *ports.ComposicionVisualRRHH) {
			c.Capacidades[0].OperacionClave = "operacion.no_publicada"
		}},
		{"capacidad_distinta", func(c *ports.ComposicionVisualRRHH) {
			c.Capacidades[0].CapacidadClave =
				"contratacion_temporal.expediente.cerrar"
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entorno := entornoComposicionVisualPrueba(t)
			copia, err := entorno.composicion.Clonar()
			if err != nil {
				t.Fatal(err)
			}
			caso.mutar(&copia)
			if err := copia.ValidarPara(entorno.orden); !errors.Is(
				err, ports.ErrResultadoComposicionVisualRRHHNoConfiable,
			) {
				t.Fatalf("adulteración aceptada: %v", err)
			}
		})
	}
}

func TestComposicionVisualLimitaReferenciasPanelAntesDeReservar(t *testing.T) {
	entorno := entornoComposicionVisualPrueba(t)
	porTarea := entorno.composicion
	porTarea.Flujo.Tareas[0].Paneles = make([]string, 1_000_000)
	if asignaciones := testing.AllocsPerRun(5, func() {
		_, _ = porTarea.Clonar()
	}); asignaciones != 0 {
		t.Fatalf("se reservó memoria al rechazar paneles enormes: %.0f", asignaciones)
	}
	if _, err := porTarea.Clonar(); !errors.Is(
		err, ports.ErrResultadoComposicionVisualRRHHNoConfiable,
	) {
		t.Fatalf("lista enorme por tarea aceptada: %v", err)
	}

	total := entorno.composicion
	const tareas = ports.MaximoReferenciasPanelesVisualRRHH/
		ports.MaximoPanelesPorTareaVisualRRHH + 1
	total.Flujo.Tareas = make([]ports.TareaVisualRRHH, tareas)
	for indice := range total.Flujo.Tareas {
		total.Flujo.Tareas[indice] = entorno.composicion.Flujo.Tareas[0]
		total.Flujo.Tareas[indice].Paneles = make(
			[]string, ports.MaximoPanelesPorTareaVisualRRHH,
		)
	}
	if _, err := total.Clonar(); !errors.Is(
		err, ports.ErrResultadoComposicionVisualRRHHNoConfiable,
	) {
		t.Fatalf("total excesivo de referencias aceptado: %v", err)
	}
}

func TestComposicionVisualNormalizaColeccionesJSONVacias(t *testing.T) {
	t.Parallel()
	entorno := entornoComposicionVisualPrueba(t)
	copia := entorno.composicion
	copia.Flujo.Tareas[0].Paneles = nil
	copia.Flujo.Tareas[0].Operaciones = nil
	copia.Flujo.Paneles[0].Campos = nil
	copia.Catalogos[0].Opciones = nil
	copia.Capacidades = nil
	contenido, err := json.Marshal(copia)
	if err != nil {
		t.Fatal(err)
	}
	for _, esperado := range []string{
		`"paneles":[]`, `"operaciones":[]`, `"campos":[]`,
		`"opciones":[]`, `"capacidades_visuales":[]`,
	} {
		if !strings.Contains(string(contenido), esperado) {
			t.Fatalf("colección no normalizada como []: %s", contenido)
		}
	}
	copia.Catalogos = nil
	contenido, err = json.Marshal(copia)
	if err != nil || !strings.Contains(string(contenido), `"catalogos":[]`) {
		t.Fatalf("catálogos no normalizados como []: %s, %v", contenido, err)
	}
}

func TestComposicionVisualRechazaAdulteracionValidaPorHuella(t *testing.T) {
	t.Parallel()
	entorno := entornoComposicionVisualPrueba(t)
	copia, err := entorno.composicion.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	copia.Catalogos[0].Opciones[0].ClaveI18n =
		"ui.contratacion_temporal.motivo.interinidad"
	if err := copia.ValidarPara(entorno.orden); !errors.Is(
		err, ports.ErrResultadoComposicionVisualRRHHNoConfiable,
	) {
		t.Fatalf("adulteración estructuralmente válida aceptada: %v", err)
	}
}

func TestComposicionVisualLimitaAntesDeClonarYHaceCopiaDefensiva(t *testing.T) {
	t.Parallel()
	entorno := entornoComposicionVisualPrueba(t)
	copia, err := entorno.composicion.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	copia.Flujo.Fases[0].ClaveI18n = "ui.alterada"
	copia.Flujo.Tareas[0].Paneles[0] = "panel:alterado:001"
	copia.Flujo.Paneles[0].Campos[0].Clave = "alterada"
	copia.Catalogos[0].Opciones[0].Clave = "otra"
	copia.Capacidades[0].OperacionClave = "operacion.alterada"
	if entorno.composicion.Flujo.Fases[0].ClaveI18n == "ui.alterada" ||
		entorno.composicion.Flujo.Tareas[0].Paneles[0] == "panel:alterado:001" ||
		entorno.composicion.Flujo.Paneles[0].Campos[0].Clave == "alterada" ||
		entorno.composicion.Catalogos[0].Opciones[0].Clave == "otra" ||
		entorno.composicion.Capacidades[0].OperacionClave ==
			"operacion.alterada" {
		t.Fatal("la copia comparte memoria mutable")
	}
	excesiva := entorno.composicion
	excesiva.Flujo.Fases = make(
		[]ports.FaseVisualRRHH,
		ports.MaximoFasesComposicionVisualRRHH+1,
	)
	if _, err := excesiva.Clonar(); !errors.Is(
		err, ports.ErrResultadoComposicionVisualRRHHNoConfiable,
	) {
		t.Fatalf("colección excesiva aceptada: %v", err)
	}
}

type entornoVisualRRHHPrueba struct {
	ahora       time.Time
	contexto    ports.ContextoConsultaRRHH
	vocabulario ports.VocabularioComposicionVisualRRHH
	solicitud   ports.SolicitudComposicionVisualRRHH
	capacidad   ports.CapacidadComposicionVisualRRHH
	orden       ports.OrdenConsultaComposicionVisualRRHH
	composicion ports.ComposicionVisualRRHH
}

func entornoComposicionVisualPrueba(t *testing.T) entornoVisualRRHHPrueba {
	t.Helper()
	ahora := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	contexto := contextoPuertosRRHH(t, ahora)
	vocabulario, err := ports.NuevoVocabularioComposicionVisualRRHH(
		"contratacion_temporal.composicion_visual.consultar",
		"rrhh.contratacion_temporal.tramitacion",
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudComposicionVisualRRHH(
		"flujo:rrhh:general", 3,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := ports.NuevaCapacidadComposicionVisualRRHH(
		"decision:visual:001", "correlacion:visual:001", "motivo:visual:001",
		contexto, ports.AmbitoOrganizacionRRHH, contexto.OrganizacionRef(),
		vocabulario, solicitud, ahora, ahora.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenConsultaComposicionVisualRRHH(
		contexto, capacidad, vocabulario, solicitud, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	generada := ahora.Add(time.Second)
	composicion := ports.ComposicionVisualRRHH{
		Esquema:    ports.EsquemaComposicionVisualRRHH,
		GeneradaEn: generada,
		Flujo: ports.DefinicionFlujoVisualRRHH{
			Referencia: solicitud.FlujoRef(), Version: solicitud.FlujoVersion(),
			Huella:       strings.Repeat("a", 64),
			ClaveI18n:    "ui.contratacion_temporal.flujo.general",
			PublicadoEn:  ahora.Add(-2 * time.Hour),
			VigenteDesde: ahora.Add(-time.Hour),
			VigenteHasta: ahora.Add(time.Hour),
			Fases: []ports.FaseVisualRRHH{{
				Clave: "solicitud", Orden: 1,
				ClaveI18n: "ui.contratacion_temporal.fase.solicitud",
			}},
			Tareas: []ports.TareaVisualRRHH{{
				Referencia: "tarea:solicitud:alta", FaseClave: "solicitud",
				Orden: 1, ClaveI18n: "ui.contratacion_temporal.tarea.alta",
				Paneles: []string{"panel:solicitud:datos"},
				Operaciones: []ports.OperacionVisualRRHH{{
					Clave:          "solicitud.crear",
					ClaveI18n:      "ui.contratacion_temporal.operacion.crear",
					CapacidadClave: "contratacion_temporal.solicitud.crear",
				}},
			}},
			Paneles: []ports.PanelVisualRRHH{{
				Referencia: "panel:solicitud:datos", Orden: 1,
				Tipo:      ports.PanelVisualFormulario,
				ClaveI18n: "ui.contratacion_temporal.panel.solicitud",
				Campos: []ports.CampoVisualRRHH{{
					Clave: "motivo", Orden: 1,
					ClaveI18n: "ui.contratacion_temporal.campo.motivo",
					Control:   ports.ControlVisualSeleccion, Obligatorio: true,
					CatalogoRef: "catalogo:motivos:alta", CatalogoVersion: 7,
				}},
			}},
		},
		Catalogos: []ports.CatalogoVisualRRHH{{
			Referencia: "catalogo:motivos:alta", Version: 7,
			Huella:       strings.Repeat("b", 64),
			ClaveI18n:    "ui.contratacion_temporal.catalogo.motivos",
			PublicadoEn:  ahora.Add(-2 * time.Hour),
			VigenteDesde: ahora.Add(-time.Hour),
			VigenteHasta: ahora.Add(time.Hour),
			Opciones: []ports.OpcionCatalogoVisualRRHH{{
				Clave:     "sustitucion",
				ClaveI18n: "ui.contratacion_temporal.motivo.sustitucion",
			}},
		}},
		Capacidades: []ports.CapacidadVisualConcedidaRRHH{{
			OperacionClave: "solicitud.crear",
			CapacidadClave: "contratacion_temporal.solicitud.crear",
		}},
	}
	composicion.Flujo.Huella, err =
		ports.CalcularHuellaDefinicionFlujoVisualRRHH(composicion.Flujo)
	if err != nil {
		t.Fatalf("huella de flujo: %v", err)
	}
	for indice := range composicion.Catalogos {
		composicion.Catalogos[indice].Huella, err =
			ports.CalcularHuellaCatalogoVisualRRHH(
				composicion.Catalogos[indice],
			)
		if err != nil {
			t.Fatalf("huella de catálogo: %v", err)
		}
	}
	huella, err := ports.CalcularHuellaComposicionVisualRRHH(composicion)
	if err != nil {
		t.Fatalf("huella: %v", err)
	}
	composicion.Lectura, err = ports.NuevoReciboComposicionVisualRRHH(
		"lectura:visual:001", "auditoria:visual:001",
		orden, huella, generada.Add(time.Second),
	)
	if err != nil {
		t.Fatalf("recibo: %v", err)
	}
	return entornoVisualRRHHPrueba{
		ahora: ahora, contexto: contexto, vocabulario: vocabulario,
		solicitud: solicitud, capacidad: capacidad,
		orden: orden, composicion: composicion,
	}
}
