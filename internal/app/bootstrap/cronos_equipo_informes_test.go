package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
	cronosports "vec-diputacion-granada/internal/modules/cronos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestEquipoInformesApagadoNoExigeDependenciasNiPublicaRutas(t *testing.T) {
	if err := montarEquipoInformesCronos(context.Background(), dependenciasEquipoInformesCronos{}); err != nil {
		t.Fatalf("montaje apagado: %v", err)
	}
	proveedores, err := publicarMaterialEquipoInformesCronosEnLote(
		context.Background(), config.Config{}, nil,
		materialAtestacionContratacionTemporalDesarrollo{}, relojContratacionTemporalDesarrollo{},
	)
	if err != nil || proveedores != [4]*proveedorMaterialAltaContratacionTemporalDesarrollo{} {
		t.Fatalf("publicación apagada: %v", err)
	}
}

func configEquipoInformesPrueba() configuracionEquipoInformesCronos {
	var c configuracionEquipoInformesCronos
	c.Version, c.Autoridad = 1, AutoridadNoAutoritativa
	c.Roles.JefaturaVersionRolRef = "rol:cronos_jefatura:v1"
	c.Roles.RRHHVersionRolRef = "rol:cronos_rrhh:v1"
	c.Equipos = []equipoPublicadoCronosC10{{
		EquipoRef: "equipo:uno", Nombre: "Equipo de ensayo",
		VersionRef: "foto:uno:v1", FuenteRef: "fuente:personal:v1",
		ReciboFuenteRef: "recibo:personal:uno",
	}}
	m := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_cronos", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("d", 64),
		EntradaClave:         "motivo_" + strings.Repeat("5", 32)}
	c.Motivos.PresenciaJefatura, c.Motivos.PresenciaRRHH = m, m
	c.Motivos.IncidenciasJefatura, c.Motivos.IncidenciasRRHH = m, m
	c.Motivos.InformePropio, c.Motivos.InformeJefatura, c.Motivos.InformeRRHH = m, m, m
	return c
}

func TestConfigEquipoInformesNoConcedeRutasArbitrarias(t *testing.T) {
	c := configEquipoInformesPrueba()
	if err := validarConfiguracionEquipoInformesCronos(c); err != nil {
		t.Fatalf("configuración válida rechazada: %v", err)
	}
	c.Equipos = append(c.Equipos, c.Equipos[0])
	if validarConfiguracionEquipoInformesCronos(c) == nil {
		t.Fatal("acepta equipo duplicado")
	}
	c = configEquipoInformesPrueba()
	c.Equipos[0].EquipoRef = "../otro"
	if validarConfiguracionEquipoInformesCronos(c) == nil {
		t.Fatal("acepta ruta ajena")
	}
	c = configEquipoInformesPrueba()
	c.Roles.RRHHVersionRolRef = c.Roles.JefaturaVersionRolRef
	if validarConfiguracionEquipoInformesCronos(c) == nil {
		t.Fatal("fusiona rol de jefatura y RRHH")
	}
}

func TestRolEquipoInformesExigeFuentePositivaDeDosConsultas(t *testing.T) {
	concesiones := []core.ConcesionRol{
		{Accion: cronosapp.AccionPresenciaEquipo, ModuloID: "cronos", TipoRecurso: "presencia_equipo",
			Finalidades: []string{"consultar_presencia_equipo"}, CamposPermitidos: []string{"empleado_etiqueta"}},
		{Accion: cronosapp.AccionIncidenciasEquipo, ModuloID: "cronos", TipoRecurso: "incidencias_equipo",
			Finalidades: []string{"consultar_incidencias_equipo"}, CamposPermitidos: []string{"empleado_etiqueta"}},
	}
	if !rolEquipoInformesTieneConsultas(concesiones) {
		t.Fatal("rechaza concesiones exactas")
	}
	concesiones[1].CamposPermitidos = []string{"persona_ref"}
	if rolEquipoInformesTieneConsultas(concesiones) {
		t.Fatal("acepta rol sin etiqueta nominal")
	}
}

func TestCatalogosDocumentoEquipoInformesReales(t *testing.T) {
	catalogos := make(map[string][]byte)
	for _, idioma := range []string{"es", "en"} {
		ruta := filepath.Join("..", "..", "..", "web", "static", "textos", idioma, "cronos-equipo-informes-c10.json")
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		catalogos[idioma] = contenido
	}
	r, err := renderizadoresEquipoInformesCronos(catalogos)
	if err != nil || len(r.porIdioma) != 2 {
		t.Fatalf("catálogos PDF/CSV ausentes: %v", err)
	}
	ctx := context.WithValue(context.Background(), idiomaDocumentoEquipoInformesContexto{}, "en")
	if _, err := r.RenderizarInforme(ctx, cronosdomain.InformeCSV,
		cronosports.PreparacionInformeC12{Idioma: "es"}); err == nil {
		t.Fatal("renderiza con idioma distinto del recibo de preparación")
	}
}

func TestDescriptoresEquipoInformesUsanGobiernoComun(t *testing.T) {
	descriptores := descriptoresMaterialEquipoInformesCronosDesarrollo()
	if len(descriptores) != 4 {
		t.Fatal("audiencias C10–C12 incompletas")
	}
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptores[:]); err != nil {
		t.Fatalf("descriptores fuera del gobierno común: %v", err)
	}
}

func TestOfertaInformeExigeConcesionDeExportacion(t *testing.T) {
	concesiones := []core.ConcesionRol{{
		Accion:   cronosapp.AccionExportarInformeEquipoC12,
		ModuloID: "cronos", TipoRecurso: "informe_equipo",
		Finalidades:      []string{cronosapp.FinalidadExportarInformeEquipoC12},
		CamposPermitidos: []string{"archivo", "recibo"},
	}}
	if !rolTieneExportacionEquipoInformes(concesiones, false) ||
		rolTieneExportacionEquipoInformes(concesiones, true) {
		t.Fatal("mezcla exportación de equipo y propia")
	}
	concesiones[0].CamposPermitidos = []string{"archivo"}
	if rolTieneExportacionEquipoInformes(concesiones, false) {
		t.Fatal("ofrece informe sin permiso del recibo")
	}
}
