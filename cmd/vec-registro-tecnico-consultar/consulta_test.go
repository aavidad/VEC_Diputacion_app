package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestConsultaAceptaFasesCerradasYRegistroAnterior(t *testing.T) {
	var acceso map[string]any
	if err := json.Unmarshal(bytes.TrimSuffix(accesoPrueba(t), []byte{'\n'}), &acceso); err != nil {
		t.Fatal(err)
	}
	acceso["vec.fases"] = []any{
		map[string]any{"nombre": "identidad", "n": 1, "total": 0.002},
		map[string]any{"nombre": "sesion", "n": 2, "total": 0.01},
		map[string]any{"nombre": "lectura_con_auditoria", "n": 1, "total": 0.2},
	}
	contenido := append(accesoPrueba(t), codificarLinea(t, acceso)...)
	codigo, resumen, salida, diagnostico := ejecutarPrueba(t, "--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", archivoPrueba(t, contenido))
	if codigo != 0 || resumen.Validas != 2 || resumen.Rechazadas != 0 || resumen.Peticiones.Total != 2 || diagnostico != "" || strings.Contains(salida, "lectura_con_auditoria") {
		t.Fatalf("compatibilidad de acceso anterior y con fases: código=%d resumen=%+v salida=%q diagnóstico=%q", codigo, resumen, salida, diagnostico)
	}
	for _, mal := range []any{
		[]any{map[string]any{"nombre": "persona privada", "n": 1, "total": 0.1}},
		[]any{map[string]any{"nombre": "identidad", "n": 1, "total": 0.1, "dato": "persona privada"}},
		[]any{map[string]any{"nombre": "identidad", "n": 1, "total": 0.1, "errores": 1, "canceladas": 1}},
		[]any{map[string]any{"nombre": "identidad", "n": 1, "total": 0.1}, map[string]any{"nombre": "identidad", "n": 1, "total": 0.1}},
	} {
		acceso["vec.fases"] = mal
		codigo, resumen, salida, diagnostico = ejecutarPrueba(t, "--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", archivoPrueba(t, codificarLinea(t, acceso)))
		if codigo != 3 || resumen.Rechazadas != 1 || resumen.Validas != 0 || strings.Contains(salida+diagnostico, "persona privada") {
			t.Fatalf("fase libre aceptada o filtrada: código=%d resumen=%+v salida=%q diagnóstico=%q", codigo, resumen, salida, diagnostico)
		}
	}
}

func TestConsultaAceptaCamposSQLYLotesEmitidosEnPeticionFallida(t *testing.T) {
	var acceso map[string]any
	if err := json.Unmarshal(bytes.TrimSuffix(accesoPrueba(t), []byte{'\n'}), &acceso); err != nil {
		t.Fatal(err)
	}
	acceso["level"] = "ERROR"
	acceso["http.response.status_code"] = 503
	acceso["error.type"] = "bd_57014"
	acceso["vec.bd.error"] = "bd_57014"
	acceso["vec.bd.lotes"] = 1
	acceso["vec.bd.lote.resultados_observados"] = 2
	acceso["vec.bd.lote.errores"] = 1
	acceso["vec.bd.lote.duracion_hasta_cierre"] = 0.03
	acceso["vec.bd.operaciones_desconocidas"] = 1
	acceso["vec.bd.operaciones"] = []any{map[string]any{
		"nombre": "vec_usuarios.consultar_preferencias_propias_v1", "n": 1,
		"total": 0.2, "maxima": 0.2, "errores": map[string]any{"bd_57014": 1},
	}}
	acceso["vec.fases"] = []any{map[string]any{"nombre": "lectura_con_auditoria", "n": 1, "total": 0.25, "errores": 1}}
	codigo, resumen, _, diagnostico := ejecutarPrueba(t, "--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", archivoPrueba(t, codificarLinea(t, acceso)))
	if codigo != 0 || resumen.Validas != 1 || resumen.Rechazadas != 0 || resumen.Peticiones.Errores5xx != 1 || diagnostico != "" {
		t.Fatalf("campos emitidos no consultables: código=%d resumen=%+v diagnóstico=%q", codigo, resumen, diagnostico)
	}
	for _, nombre := range []string{"select", "sql", "otras"} {
		acceso["vec.bd.operaciones"] = []any{map[string]any{"nombre": nombre, "n": 1, "total": 0.1, "maxima": 0.1}}
		codigo, resumen, _, _ = ejecutarPrueba(t, "--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", archivoPrueba(t, codificarLinea(t, acceso)))
		if codigo != 0 || resumen.Validas != 1 {
			t.Fatalf("nombre emitido %q rechazado: código=%d resumen=%+v", nombre, codigo, resumen)
		}
	}
	acceso["vec.bd.operaciones"] = []any{map[string]any{"nombre": "persona_privada", "n": 1, "total": 0.1, "maxima": 0.1}}
	codigo, resumen, salida, diagnostico := ejecutarPrueba(t, "--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", archivoPrueba(t, codificarLinea(t, acceso)))
	if codigo != 3 || resumen.Rechazadas != 1 || strings.Contains(salida+diagnostico, "persona_privada") {
		t.Fatalf("operación SQL libre aceptada o copiada: código=%d resumen=%+v salida=%q diagnóstico=%q", codigo, resumen, salida, diagnostico)
	}
	acceso["vec.bd.operaciones"] = []any{map[string]any{"nombre": "vec_usuarios.consultar_preferencias_propias_v1", "n": 1,
		"total": 0.2, "maxima": 0.2, "errores": map[string]any{"persona privada": 1}}}
	codigo, resumen, salida, diagnostico = ejecutarPrueba(t, "--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", archivoPrueba(t, codificarLinea(t, acceso)))
	if codigo != 3 || resumen.Rechazadas != 1 || strings.Contains(salida+diagnostico, "persona privada") {
		t.Fatalf("clase SQL libre aceptada o copiada: código=%d resumen=%+v salida=%q diagnóstico=%q", codigo, resumen, salida, diagnostico)
	}
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
	codigo, err := ejecutar(args, &salida, &diagnostico)
	if err != nil {
		t.Fatalf("error de salida inesperado: %v", err)
	}
	var resumen resumenConsulta
	if salida.Len() > 0 && json.Unmarshal(salida.Bytes(), &resumen) != nil {
		t.Fatalf("salida no JSON: %q", salida.String())
	}
	return codigo, resumen, salida.String(), diagnostico.String()
}

type escritorFallidoConsulta struct{}

func (escritorFallidoConsulta) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

type escritorCortoConsulta struct{}

func (escritorCortoConsulta) Write(datos []byte) (int, error) { return len(datos) / 2, nil }

func TestConsultaPropagaErrorDeSalidaConClaseCerrada(t *testing.T) {
	codigo, err := ejecutar([]string{"--desde", "invalido"}, io.Discard, escritorFallidoConsulta{})
	if codigo != 2 || !errors.Is(err, errSalidaConsultaNoDisponible) || !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("fallo de diagnóstico perdido: código=%d error=%v", codigo, err)
	}
	var diagnostico bytes.Buffer
	codigo, err = ejecutar([]string{"--ayuda"}, escritorFallidoConsulta{}, &diagnostico)
	if codigo != 2 || !errors.Is(err, errSalidaConsultaNoDisponible) || !errors.Is(err, io.ErrClosedPipe) ||
		!strings.Contains(diagnostico.String(), "mensaje") || strings.Contains(diagnostico.String(), "closed pipe") {
		t.Fatalf("fallo de salida principal perdido o expuesto: código=%d error=%v diagnóstico=%q", codigo, err, diagnostico.String())
	}
	diagnostico.Reset()
	archivo := archivoPrueba(t, accesoPrueba(t))
	codigo, err = ejecutar([]string{"--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", archivo}, escritorFallidoConsulta{}, &diagnostico)
	if codigo != 2 || !errors.Is(err, errSalidaConsultaNoDisponible) || !errors.Is(err, io.ErrClosedPipe) ||
		!strings.Contains(diagnostico.String(), "mensaje") || strings.Contains(diagnostico.String(), archivo) {
		t.Fatalf("fallo del resumen perdido o expuesto: código=%d error=%v diagnóstico=%q", codigo, err, diagnostico.String())
	}
	diagnostico.Reset()
	codigo, err = ejecutar([]string{"--ayuda"}, escritorCortoConsulta{}, &diagnostico)
	if codigo != 2 || !errors.Is(err, errSalidaConsultaNoDisponible) || !errors.Is(err, io.ErrShortWrite) ||
		!strings.Contains(diagnostico.String(), "mensaje") {
		t.Fatalf("escritura parcial aceptada: código=%d error=%v diagnóstico=%q", codigo, err, diagnostico.String())
	}
}

func TestConsultaUsaCatalogoComunDeIdiomas(t *testing.T) {
	porDefecto, err := cargarTextos("")
	if err != nil || !strings.Contains(porDefecto.Ayuda, "Indique") {
		t.Fatal("idioma por defecto del índice común no disponible")
	}
	ingles, err := cargarTextos("en")
	if err != nil || !strings.Contains(ingles.Ayuda, "Provide") {
		t.Fatal("catálogo inglés no disponible")
	}
	if _, err := cargarTextos("zz"); err == nil {
		t.Fatal("idioma ajeno al índice aceptado")
	}
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

func TestConsultaClasificaInstanteInvalidoSinExponerEntrada(t *testing.T) {
	var acceso map[string]any
	if err := json.Unmarshal(bytes.TrimSuffix(accesoPrueba(t), []byte{'\n'}), &acceso); err != nil {
		t.Fatal(err)
	}
	marcador := "dato_temporal_sintetico"
	acceso["time"] = marcador
	archivo := archivoPrueba(t, codificarLinea(t, acceso))
	codigo, r, salida, diagnostico := ejecutarPrueba(t, "--desde", "2026-10-07T11:00:00Z", "--hasta", "2026-10-07T13:00:00Z", "--archivo", archivo)
	if codigo != 3 || r.Rechazadas != 1 || r.Rechazos["instante_invalido"] != 1 || r.Validas != 0 ||
		strings.Contains(salida+diagnostico, marcador) {
		t.Fatalf("fecha inválida sin causa cerrada: código=%d %+v salida=%q diagnóstico=%q", codigo, r, salida, diagnostico)
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

func TestConsultaLeePrefijoInicialAnteAppendYDetectaTruncamiento(t *testing.T) {
	catalogo, err := catalogoincidencias.Predeterminado()
	if err != nil {
		t.Fatal(err)
	}
	base := accesoPrueba(t)
	ruta := archivoPrueba(t, base)
	f, err := os.Open(ruta)
	if err != nil {
		t.Fatal(err)
	}
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	anexo, err := os.OpenFile(ruta, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := anexo.Write(base); err != nil {
		t.Fatal(err)
	}
	if err := anexo.Close(); err != nil {
		t.Fatal(err)
	}
	op := opcionesConsulta{Desde: time.Date(2026, 10, 7, 11, 0, 0, 0, time.UTC), Hasta: time.Date(2026, 10, 7, 13, 0, 0, 0, time.UTC)}
	var resumen resumenConsulta
	if err := consultarPrefijo(f, info.Size(), op, catalogo, &resumen); err != nil ||
		resumen.Recibidas != 1 || resumen.Seleccionadas != 1 || resumen.BytesPrefijo != info.Size() {
		t.Fatalf("append alteró el prefijo: error=%v resumen=%+v", err, resumen)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	f, err = os.Open(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(ruta, 0); err != nil {
		t.Fatal(err)
	}
	if err := consultarPrefijo(f, info.Size(), op, catalogo, &resumen); err == nil {
		t.Fatal("truncamiento del prefijo presentado como consulta completa")
	}
}

func TestConsultaPrefijoConLineaFinalIncompletaEsParcial(t *testing.T) {
	catalogo, err := catalogoincidencias.Predeterminado()
	if err != nil {
		t.Fatal(err)
	}
	ruta := archivoPrueba(t, append(accesoPrueba(t), []byte("{\"msg\":\"http.server.request\"")...))
	f, err := os.Open(ruta)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	anexo, err := os.OpenFile(ruta, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := anexo.Write([]byte("}\n")); err != nil {
		t.Fatal(err)
	}
	if err := anexo.Close(); err != nil {
		t.Fatal(err)
	}
	op := opcionesConsulta{Desde: time.Date(2026, 10, 7, 11, 0, 0, 0, time.UTC), Hasta: time.Date(2026, 10, 7, 13, 0, 0, 0, time.UTC)}
	var resumen resumenConsulta
	if err := consultarPrefijo(f, info.Size(), op, catalogo, &resumen); err != nil ||
		resumen.Validas != 1 || resumen.Rechazadas != 1 || resumen.BytesPrefijo != info.Size() {
		t.Fatalf("línea final incompleta no visible: error=%v resumen=%+v", err, resumen)
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
