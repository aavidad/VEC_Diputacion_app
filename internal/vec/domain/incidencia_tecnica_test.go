package domain

import (
	"strings"
	"testing"
	"time"
)

func TestCatalogoIncidenciasTecnicasCerradoYCompleto(t *testing.T) {
	esperados := []string{
		"ARRANQUE_FALLIDO", "CATALOGO_MODULOS_INVALIDO", "HTTP_INTERNO_FALLIDO",
		"PANICO_CONTROLADO", "GOBIERNO_V3_NO_DISPONIBLE", "POSTGRES_NO_DISPONIBLE",
		"SMTP_NO_DISPONIBLE", "OSRM_NO_DISPONIBLE", "AUDITORIA_NO_REGISTRADA",
		"MODULO_WEB_NO_CARGADO", "CLIENTE_FALLO_NO_CLASIFICADO", "RECOLECCION_DEGRADADA",
		"ALERTA_NO_ENTREGADA",
	}
	codigos := CodigosIncidenciaTecnica()
	if len(codigos) != len(esperados) {
		t.Fatalf("catalogo con %d codigos, se esperaban %d", len(codigos), len(esperados))
	}
	vistos := map[CodigoIncidenciaTecnica]bool{}
	for i, codigo := range codigos {
		if string(codigo) != esperados[i] {
			t.Fatalf("codigo %d = %q, se esperaba %q", i, codigo, esperados[i])
		}
		if vistos[codigo] {
			t.Fatalf("codigo duplicado %q", codigo)
		}
		vistos[codigo] = true
		def, ok := DefinicionIncidenciaTecnicaDe(codigo)
		if !ok || def.Codigo != codigo {
			t.Fatalf("codigo %q sin definicion", codigo)
		}
		switch def.Severidad {
		case SeveridadIncidenciaAviso, SeveridadIncidenciaError, SeveridadIncidenciaCritica:
		default:
			t.Fatalf("severidad invalida en %q", codigo)
		}
		if len(def.Componentes) == 0 || len(def.Etapas) == 0 {
			t.Fatalf("%q sin componentes o etapas", codigo)
		}
		if def.Plantilla == "" || strings.ContainsAny(def.Plantilla, "%{}<>") {
			t.Fatalf("plantilla de %q no es texto fijo: %q", codigo, def.Plantilla)
		}
		// Toda combinación admitida se clasifica sin saneamiento.
		for _, componente := range def.Componentes {
			for _, etapa := range def.Etapas {
				c, saneada := ClasificarIncidenciaTecnica(SolicitudIncidenciaTecnica{Codigo: codigo, Componente: componente, Etapa: etapa})
				if saneada || c.Codigo != codigo || c.Componente != componente || c.Etapa != etapa || c.Severidad != def.Severidad {
					t.Fatalf("combinacion admitida saneada: %q/%q/%q", codigo, componente, etapa)
				}
			}
		}
	}
	codigos[0] = "ALTERADO"
	if CodigosIncidenciaTecnica()[0] != IncidenciaArranqueFallido {
		t.Fatal("la lista de codigos devuelta comparte memoria")
	}
	def, _ := DefinicionIncidenciaTecnicaDe(IncidenciaArranqueFallido)
	def.Componentes[0] = "alterado"
	otra, _ := DefinicionIncidenciaTecnicaDe(IncidenciaArranqueFallido)
	if otra.Componentes[0] != ComponenteIncidenciaServidor {
		t.Fatal("la definicion devuelta comparte memoria")
	}
	if VersionCatalogoIncidenciasTecnicas != 1 || EsquemaIncidenciaTecnica != "vec.incidencia_tecnica.v1" {
		t.Fatal("version de catalogo o esquema inesperados")
	}
}

func TestClasificarIncidenciaTecnicaSaneaLoNoCatalogado(t *testing.T) {
	casos := []SolicitudIncidenciaTecnica{
		{Codigo: "DESCONOCIDO", Componente: ComponenteIncidenciaServidor, Etapa: EtapaIncidenciaComposicion},
		{Codigo: "arranque_fallido", Componente: ComponenteIncidenciaServidor, Etapa: EtapaIncidenciaComposicion},
		{Codigo: IncidenciaArranqueFallido, Componente: "12345678Z", Etapa: EtapaIncidenciaComposicion},
		{Codigo: IncidenciaArranqueFallido, Componente: ComponenteIncidenciaServidor, Etapa: "/home/persona/secreto"},
		{Codigo: IncidenciaArranqueFallido, Componente: ComponenteIncidenciaPostgreSQL, Etapa: EtapaIncidenciaComposicion},
		{},
	}
	for _, s := range casos {
		c, saneada := ClasificarIncidenciaTecnica(s)
		if !saneada {
			t.Fatalf("no saneada: %+v", s)
		}
		if c.Codigo != IncidenciaRecoleccionDegradada || c.Componente != ComponenteIncidenciaSupervision || c.Etapa != EtapaIncidenciaValidacion || c.Severidad != SeveridadIncidenciaAviso {
			t.Fatalf("saneamiento inesperado: %+v", c)
		}
	}
}

func TestClasificarIncidenciaTecnicaAcotaRecuento(t *testing.T) {
	base := SolicitudIncidenciaTecnica{Codigo: IncidenciaHTTPInternoFallido, Componente: ComponenteIncidenciaHTTP, Etapa: EtapaIncidenciaPeticion}
	for entrada, esperado := range map[uint32]uint32{0: 1, 1: 1, 7: 7, RecuentoMaximoIncidenciaTecnica + 1: RecuentoMaximoIncidenciaTecnica, ^uint32(0): RecuentoMaximoIncidenciaTecnica} {
		base.Recuento = entrada
		if c, _ := ClasificarIncidenciaTecnica(base); c.Recuento != esperado {
			t.Fatalf("recuento %d -> %d, se esperaba %d", entrada, c.Recuento, esperado)
		}
	}
}

func TestNormalizacionesDeEntornoVersionYCorrelacion(t *testing.T) {
	for entrada, esperado := range map[string]EntornoIncidenciaTecnica{
		"produccion": EntornoIncidenciaProduccion, "desarrollo": EntornoIncidenciaDesarrollo,
		"pruebas": EntornoIncidenciaPruebas, "presentacion": EntornoIncidenciaPresentacion,
		"": EntornoIncidenciaDesconocido, "PRODUCCION": EntornoIncidenciaDesconocido, "10.1.2.3": EntornoIncidenciaDesconocido,
	} {
		if obtenido := NormalizarEntornoIncidenciaTecnica(entrada); obtenido != esperado {
			t.Fatalf("entorno %q -> %q", entrada, obtenido)
		}
	}
	for entrada, valida := range map[string]bool{
		"1f5222d": true, "1f5222d7aabbccddeeff00112233445566778899": true, "v1.2.3": true, "v10.20.30": true,
		"": false, "1F5222D": false, "1f5222": false, "v1.2": false, "1.2.3": false, "v1.2.3-dev": false,
		"12345678Z": false, "/home/x/vec": false, "v1..3": false, "v1.2.": false, "v12345.1.1": false,
	} {
		obtenida := NormalizarVersionBinario(entrada)
		if valida && obtenida != entrada || !valida && obtenida != VersionBinarioDesconocida {
			t.Fatalf("version %q -> %q", entrada, obtenida)
		}
	}
	if !EsCorrelacionTecnicaValida(strings.Repeat("a1", 16)) || EsCorrelacionTecnicaValida("per_0123456789abcdef0123456789ab") || EsCorrelacionTecnicaValida(strings.Repeat("A", 32)) {
		t.Fatal("validacion de correlacion incorrecta")
	}
}

func TestNuevaIncidenciaTecnicaRevalidaCampos(t *testing.T) {
	// Una clasificación construida a mano, sin pasar por el catálogo, no
	// puede colar valores ni severidad o mensaje propios.
	manual := ClasificacionIncidenciaTecnica{
		Codigo: "persona@example.org", Severidad: "critica", Componente: "10.1.2.3",
		Etapa: "Juan Perez", Recuento: 3, Plantilla: "DNI 12345678Z",
	}
	instante := time.Date(2026, 9, 25, 10, 11, 12, 123456789, time.FixedZone("CEST", 2*3600))
	inc := NuevaIncidenciaTecnica(manual, instante, "per_123", "/home/x", "no-hex")
	if inc.Codigo != IncidenciaRecoleccionDegradada || inc.Mensaje == manual.Plantilla || inc.Severidad != SeveridadIncidenciaAviso {
		t.Fatalf("clasificacion manual no revalidada: %+v", inc)
	}
	if inc.Entorno != EntornoIncidenciaDesconocido || inc.VersionBinario != VersionBinarioDesconocida || inc.Correlacion != strings.Repeat("0", 32) {
		t.Fatalf("metadatos no saneados: %+v", inc)
	}
	if inc.Instante.Location() != time.UTC || inc.Instante.Nanosecond() != 123000000 || inc.Instante.Hour() != 8 {
		t.Fatalf("instante no normalizado a UTC y milisegundos: %v", inc.Instante)
	}
	if inc.Esquema != EsquemaIncidenciaTecnica || inc.Recuento != 3 {
		t.Fatalf("esquema o recuento inesperados: %+v", inc)
	}
}
