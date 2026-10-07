package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/catalogoincidencias"
	"vec-diputacion-granada/internal/vec/domain"
)

const instantePrueba = "2026-10-07T12:00:00.000Z"
const correlacionPrueba = "0123456789abcdef0123456789abcdef"

func codificarLinea(t *testing.T, valor any) []byte {
	t.Helper()
	b, err := json.Marshal(valor)
	if err != nil {
		t.Fatal(err)
	}
	return append(b, '\n')
}

func accesoPrueba(t *testing.T) []byte {
	t.Helper()
	return codificarLinea(t, map[string]any{
		"time": instantePrueba, "level": "WARN", "msg": "http.server.request",
		"service.name": "vec-server", "service.version": "72cf97a1c07e",
		"deployment.environment.name": "desarrollo", "vec.superficie": "interno",
		"http.request.method": "GET", "url.path": "/api/vec/bolsa/mi-bolsa/historial",
		"http.response.status_code": 200, "http.server.request.duration": 0.4069,
		"http.response.body.size": 0, "vec.correlacion": correlacionPrueba,
		"vec.bd.consultas": 31, "vec.bd.duracion": 0.4036,
		"vec.bd.espera_conexion": 0.0033, "vec.lenta": true,
		"vec.bd.consulta_mas_lenta":          "vec_bolsa.listar_participaciones",
		"vec.bd.consulta_mas_lenta.duracion": 0.4014,
	})
}

func arranquePrueba(t *testing.T) []byte {
	t.Helper()
	return codificarLinea(t, map[string]any{
		"time": instantePrueba, "level": "ERROR", "msg": "vec.process.startup",
		"service.name": "vec-server", "service.version": "72cf97a1c07e",
		"deployment.environment.name": "desarrollo", "vec.superficie": "interno",
		"vec.arranque.fase": "composicion", "vec.arranque.resultado": "fallida",
		"vec.arranque.duracion": 0.15, "error.type": "otro",
	})
}

func incidenciaPrueba(t *testing.T) []byte {
	t.Helper()
	catalogo, err := catalogoincidencias.Predeterminado()
	if err != nil {
		t.Fatal(err)
	}
	mensaje, ok := catalogo.Plantilla(domain.IncidenciaArranqueFallido)
	if !ok {
		t.Fatal("incidencia sin texto de catálogo")
	}
	return codificarLinea(t, map[string]any{
		"esquema": domain.EsquemaIncidenciaTecnica, "instante": instantePrueba,
		"codigo": string(domain.IncidenciaArranqueFallido), "severidad": "critica",
		"componente": "servidor", "etapa": "escucha", "entorno": "desarrollo",
		"version_binario": "72cf97a1c07e", "correlacion": correlacionPrueba,
		"recuento": 2, "mensaje": mensaje,
	})
}

func resultadoPrueba(t *testing.T) []byte {
	t.Helper()
	return codificarLinea(t, map[string]any{
		"esquema": domain.EsquemaResultadoTecnico, "instante": instantePrueba,
		"resultado": "no_disponible", "nivel": "error", "componente": "servidor",
		"etapa": "escucha", "entorno": "desarrollo", "version_binario": "72cf97a1c07e",
		"correlacion": correlacionPrueba, "correlacion_ref": "correlacion_" + correlacionPrueba,
	})
}

func archivoPrueba(t *testing.T, contenido []byte) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "tecnico.jsonl")
	if err := os.WriteFile(ruta, contenido, 0600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func ejecutarPrueba(t *testing.T, args ...string) (int, resumenConsulta, string, string) {
	t.Helper()
	var salida, diagnostico bytes.Buffer
	codigo := ejecutar(args, &salida, &diagnostico)
	var resumen resumenConsulta
	if salida.Len() > 0 && json.Unmarshal(salida.Bytes(), &resumen) != nil {
		t.Fatalf("salida no JSON: %q", salida.String())
	}
	return codigo, resumen, salida.String(), diagnostico.String()
}

func TestConsultaAgregaRegistrosYFiltraSinExponerRuta(t *testing.T) {
	archivo := archivoPrueba(t, bytes.Join([][]byte{accesoPrueba(t), arranquePrueba(t), incidenciaPrueba(t), resultadoPrueba(t)}, nil))
	base := []string{"--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", archivo}
	codigo, r, salida, diagnostico := ejecutarPrueba(t, base...)
	if codigo != 0 || diagnostico != "" || r.Recibidas != 4 || r.Validas != 4 || r.Seleccionadas != 4 || r.Rechazadas != 0 ||
		r.Peticiones.Total != 1 || r.Peticiones.Lentas != 1 || r.Peticiones.ConsultasBD != 31 ||
		r.Peticiones.Duracion.TotalSegundos != 0.4069 || r.Peticiones.DuracionBD.TotalSegundos != 0.4036 ||
		r.Peticiones.EsperaConexionPool.TotalSegundos != 0.0033 || r.Arranque.Fallidas != 1 ||
		r.Incidencias["ARRANQUE_FALLIDO"] != 2 || r.Resultados["no_disponible"] != 1 || r.Errores["otro"] != 1 {
		t.Fatalf("resumen inesperado: código=%d %+v; diagnóstico=%q", codigo, r, diagnostico)
	}
	if strings.Contains(salida, "/api/vec/") || strings.Contains(salida, correlacionPrueba) || strings.Contains(salida, "listar_participaciones") {
		t.Fatalf("salida contiene campos de origen: %s", salida)
	}
	codigo, r, _, _ = ejecutarPrueba(t, append(base, "--ruta", "/api/vec/bolsa/mi-bolsa/historial")...)
	if codigo != 0 || r.Seleccionadas != 1 || r.Peticiones.Total != 1 || r.Arranque.Fallidas != 0 || len(r.Incidencias) != 0 {
		t.Fatalf("filtro de ruta: %+v", r)
	}
	codigo, r, _, _ = ejecutarPrueba(t, append(base, "--codigo", "ARRANQUE_FALLIDO")...)
	if codigo != 0 || r.Seleccionadas != 1 || r.Incidencias["ARRANQUE_FALLIDO"] != 2 || r.Peticiones.Total != 0 {
		t.Fatalf("filtro de código: %+v", r)
	}
}

func TestConsultaDeclaraEntradasInvalidasSinCopiarDatos(t *testing.T) {
	secreto := "persona_sintetica_clave_irrepetible"
	contenido := append(accesoPrueba(t), []byte("{\"msg\":\"http.server.request\",\"msg\":\""+secreto+"\"}\n")...)
	contenido = append(contenido, []byte("{\"msg\":\"http.server.request\",\"identidad\":\""+secreto+"\"}\n")...)
	contenido = append(contenido, []byte(strings.Repeat("X", maxLineaBytes+1)+"\n")...)
	contenido = append(contenido, []byte("{\"password\":\""+secreto+"\"}")...)
	archivo := archivoPrueba(t, contenido)
	codigo, r, salida, diagnostico := ejecutarPrueba(t, "--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", archivo)
	if codigo != 3 || r.Validas != 1 || r.Rechazadas != 4 || r.Estado != "parcial" || diagnostico != "" ||
		strings.Contains(salida+diagnostico, secreto) {
		t.Fatalf("rechazo no visible o dato copiado: código=%d %+v salida=%q diagnóstico=%q", codigo, r, salida, diagnostico)
	}
}

func TestConsultaProyectaClaseDesconocidaSinCopiarCodigoLibre(t *testing.T) {
	var acceso map[string]any
	if err := json.Unmarshal(bytes.TrimSuffix(accesoPrueba(t), []byte{'\n'}), &acceso); err != nil {
		t.Fatal(err)
	}
	acceso["level"] = "ERROR"
	acceso["http.response.status_code"] = 503
	acceso["error.type"] = "bd_JUAN1"
	acceso["vec.bd.error"] = "bd_JUAN1"
	archivo := archivoPrueba(t, codificarLinea(t, acceso))
	codigo, r, salida, diagnostico := ejecutarPrueba(t, "--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", archivo)
	if codigo != 0 || r.Peticiones.Errores5xx != 1 || r.Errores["bd_otro"] != 1 ||
		strings.Contains(salida+diagnostico, "JUAN1") {
		t.Fatalf("clase no cerrada: código=%d %+v salida=%q diagnóstico=%q", codigo, r, salida, diagnostico)
	}
}

func TestConsultaRechazaArchivoYParametrosSinMostrarRuta(t *testing.T) {
	marcador := "nombre-privado-sintetico"
	directorio := t.TempDir()
	ruta := filepath.Join(directorio, marcador)
	if err := os.Symlink("/dev/null", ruta); err != nil {
		t.Fatal(err)
	}
	base := []string{"--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", ruta}
	codigo, _, salida, diagnostico := ejecutarPrueba(t, base...)
	if codigo != 2 || salida != "" || !strings.Contains(diagnostico, "mensaje") || strings.Contains(diagnostico, marcador) {
		t.Fatalf("archivo inseguro: código=%d salida=%q diagnóstico=%q", codigo, salida, diagnostico)
	}
	archivo := archivoPrueba(t, accesoPrueba(t))
	codigo, _, salida, diagnostico = ejecutarPrueba(t, "--desde", "2026-10-07T13:00:00Z", "--hasta", "2026-10-07T11:00:00Z", "--archivo", archivo)
	if codigo != 2 || salida != "" || !strings.Contains(diagnostico, "mensaje") {
		t.Fatalf("intervalo inválido aceptado: %d %q %q", codigo, salida, diagnostico)
	}
	codigo, _, salida, diagnostico = ejecutarPrueba(t, "--ayuda", "--idioma", "en")
	if codigo != 0 || diagnostico != "" || !strings.Contains(salida, "technical JSONL") {
		t.Fatalf("ayuda inglesa: %d %q %q", codigo, salida, diagnostico)
	}
	grande := archivoPrueba(t, nil)
	if err := os.Truncate(grande, maxArchivoBytes+1); err != nil {
		t.Fatal(err)
	}
	codigo, _, salida, diagnostico = ejecutarPrueba(t, "--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", grande)
	if codigo != 2 || salida != "" || !strings.Contains(diagnostico, "mensaje") {
		t.Fatalf("archivo truncado como éxito: %d %q %q", codigo, salida, diagnostico)
	}
	enlace := filepath.Join(t.TempDir(), "duplicado.jsonl")
	if err := os.Link(archivo, enlace); err != nil {
		t.Fatal(err)
	}
	codigo, _, salida, diagnostico = ejecutarPrueba(t, "--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", archivo, "--archivo", enlace)
	if codigo != 2 || salida != "" || !strings.Contains(diagnostico, "mensaje") {
		t.Fatalf("archivo duplicado contado: %d %q %q", codigo, salida, diagnostico)
	}
}
