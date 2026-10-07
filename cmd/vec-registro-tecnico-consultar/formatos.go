package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/shared/telemetria"
	"vec-diputacion-granada/internal/vec/adapters/catalogoincidencias"
	"vec-diputacion-granada/internal/vec/adapters/observabilidad"
	"vec-diputacion-granada/internal/vec/domain"
)

var camposAcceso = map[string]bool{
	"time": true, "level": true, "msg": true, "service.name": true, "service.version": true,
	"deployment.environment.name": true, "vec.superficie": true, "http.request.method": true,
	"url.path": true, "http.response.status_code": true, "http.server.request.duration": true,
	"http.response.body.size": true, "vec.correlacion": true, "vec.bd.consultas": true,
	"vec.bd.duracion": true, "vec.bd.espera_conexion": true, "vec.bd.error": true,
	"error.type": true, "vec.lenta": true, "vec.bd.consulta_mas_lenta": true,
	"vec.bd.consulta_mas_lenta.duracion": true, "vec.interrumpida": true, "vec.cancelada": true,
	"vec.fases": true, "vec.bd.lotes": true, "vec.bd.lote.resultados_observados": true,
	"vec.bd.lote.errores": true, "vec.bd.lote.duracion_hasta_cierre": true,
	"vec.bd.operaciones_desconocidas": true, "vec.bd.operaciones": true,
}

var camposOperacionAcceso = map[string]bool{
	"nombre": true, "n": true, "total": true, "maxima": true, "errores": true,
}

func validarOperacionesAcceso(valor json.RawMessage) error {
	var operaciones []json.RawMessage
	if err := json.Unmarshal(valor, &operaciones); err != nil {
		return os.ErrInvalid
	}
	if len(operaciones) == 0 || len(operaciones) > 17 {
		return os.ErrInvalid
	}
	vistas := make(map[string]bool, len(operaciones))
	for _, operacion := range operaciones {
		campos, err := objetoPlano(operacion)
		if err != nil {
			return os.ErrInvalid
		}
		if !soloCampos(campos, camposOperacionAcceso) {
			return os.ErrInvalid
		}
		nombre, ok := cadena(campos, "nombre")
		n, okN := entero(campos, "n", 1_000_000)
		total, okTotal := segundos(campos, "total")
		maxima, okMaxima := segundos(campos, "maxima")
		if !ok || !telemetria.NombreOperacionRegistrada(nombre) || vistas[nombre] || !okN || n == 0 || !okTotal || !okMaxima || maxima > total+0.0001 {
			return os.ErrInvalid
		}
		vistas[nombre] = true
		if bruto, presente := campos["errores"]; presente {
			errores, err := objetoPlano(bruto)
			if err != nil {
				return os.ErrInvalid
			}
			if len(errores) == 0 || len(errores) > 5 {
				return os.ErrInvalid
			}
			var totalErrores int64
			for clase, cantidad := range errores {
				if clase != "otras" {
					if _, valida := claseErrorCerrada(clase); !valida {
						return os.ErrInvalid
					}
				}
				valor, valido := entero(map[string]json.RawMessage{"valor": cantidad}, "valor", n)
				if !valido || valor == 0 {
					return os.ErrInvalid
				}
				totalErrores += valor
			}
			if totalErrores > n {
				return os.ErrInvalid
			}
		}
	}
	return nil
}

func validarMedidasSQLAcceso(objeto map[string]json.RawMessage, consultas int64, lenta bool, estado int) error {
	lotes, conLotes := objeto["vec.bd.lotes"]
	_, conResultados := objeto["vec.bd.lote.resultados_observados"]
	_, conErrores := objeto["vec.bd.lote.errores"]
	_, conDuracion := objeto["vec.bd.lote.duracion_hasta_cierre"]
	if conLotes != conResultados || conLotes != conErrores || conLotes != conDuracion {
		return os.ErrInvalid
	}
	if conLotes {
		n, ok := entero(map[string]json.RawMessage{"lotes": lotes}, "lotes", 1_000_000)
		observados, okObservados := entero(objeto, "vec.bd.lote.resultados_observados", 1_000_000)
		errores, okErrores := entero(objeto, "vec.bd.lote.errores", 1_000_000)
		_, okDuracion := segundos(objeto, "vec.bd.lote.duracion_hasta_cierre")
		if !ok || n == 0 || !okObservados || !okErrores || !okDuracion || errores > observados+n {
			return os.ErrInvalid
		}
	}
	if _, existe := objeto["vec.bd.operaciones_desconocidas"]; existe {
		n, ok := entero(objeto, "vec.bd.operaciones_desconocidas", consultas)
		if !ok || n == 0 || !(lenta || estado >= 400) {
			return os.ErrInvalid
		}
	}
	if operaciones, existe := objeto["vec.bd.operaciones"]; existe {
		if !(lenta || estado >= 400) {
			return os.ErrInvalid
		}
		if err := validarOperacionesAcceso(operaciones); err != nil {
			return err
		}
	}
	return nil
}

var nombresFaseAcceso = map[string]bool{
	"identidad": true, "sesion": true, "contexto": true, "v3": true,
	"lectura_con_auditoria": true, "auditoria_denegacion": true,
}

var camposFaseAcceso = map[string]bool{
	"nombre": true, "n": true, "total": true, "errores": true, "canceladas": true,
}

func validarFasesAcceso(valor json.RawMessage) error {
	var fases []json.RawMessage
	if err := json.Unmarshal(valor, &fases); err != nil {
		return os.ErrInvalid
	}
	if len(fases) == 0 || len(fases) > len(nombresFaseAcceso) {
		return os.ErrInvalid
	}
	vistas := make(map[string]bool, len(fases))
	for _, fase := range fases {
		campos, err := objetoPlano(fase)
		if err != nil {
			return os.ErrInvalid
		}
		if !soloCampos(campos, camposFaseAcceso) {
			return os.ErrInvalid
		}
		nombre, ok := cadena(campos, "nombre")
		n, okN := entero(campos, "n", 1_000_000)
		_, okTotal := segundos(campos, "total")
		if !ok || !nombresFaseAcceso[nombre] || vistas[nombre] || !okN || n == 0 || !okTotal {
			return os.ErrInvalid
		}
		vistas[nombre] = true
		var fallos, canceladas int64
		if _, presente := campos["errores"]; presente {
			fallos, ok = entero(campos, "errores", n)
			if !ok || fallos == 0 {
				return os.ErrInvalid
			}
		}
		if _, presente := campos["canceladas"]; presente {
			canceladas, ok = entero(campos, "canceladas", n)
			if !ok || canceladas == 0 {
				return os.ErrInvalid
			}
		}
		if fallos+canceladas > n {
			return os.ErrInvalid
		}
	}
	return nil
}

var camposArranque = map[string]bool{
	"time": true, "level": true, "msg": true, "service.name": true, "service.version": true,
	"deployment.environment.name": true, "vec.superficie": true, "vec.arranque.fase": true,
	"vec.arranque.resultado": true, "vec.arranque.duracion": true, "error.type": true,
}

func validarRegistro(linea []byte, catalogo *catalogoincidencias.Catalogo) (registroConsulta, error) {
	objeto, err := objetoPlano(linea)
	if err != nil {
		return registroConsulta{}, os.ErrInvalid
	}
	_, tieneEsquema := objeto["esquema"]
	_, tieneMensaje := objeto["msg"]
	if tieneEsquema == tieneMensaje {
		return registroConsulta{}, os.ErrInvalid
	}
	if tieneEsquema {
		proyeccion, err := observabilidad.ValidarRegistroConsulta(linea, catalogo)
		if err != nil {
			return registroConsulta{}, os.ErrInvalid
		}
		r := registroConsulta{instante: proyeccion.Instante, codigo: proyeccion.Codigo,
			resultado: proyeccion.Resultado, recuento: proyeccion.Recuento}
		switch proyeccion.Esquema {
		case domain.EsquemaIncidenciaTecnica:
			r.familia = "incidencia"
		case domain.EsquemaResultadoTecnico:
			r.familia = "resultado"
		default:
			return registroConsulta{}, os.ErrInvalid
		}
		return r, nil
	}
	mensaje, ok := cadena(objeto, "msg")
	if !ok {
		return registroConsulta{}, os.ErrInvalid
	}
	switch mensaje {
	case "http.server.request":
		return validarAcceso(objeto)
	case "vec.process.startup":
		return validarArranque(objeto)
	}
	return registroConsulta{}, os.ErrInvalid
}

func objetoPlano(linea []byte) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(linea))
	inicio, err := d.Token()
	if err != nil || inicio != json.Delim('{') {
		return nil, os.ErrInvalid
	}
	objeto := make(map[string]json.RawMessage)
	for d.More() {
		token, err := d.Token()
		clave, ok := token.(string)
		if err != nil || !ok || len(clave) > 80 {
			return nil, os.ErrInvalid
		}
		if _, repetida := objeto[clave]; repetida {
			return nil, os.ErrInvalid
		}
		var valor json.RawMessage
		if d.Decode(&valor) != nil {
			return nil, os.ErrInvalid
		}
		objeto[clave] = valor
	}
	fin, err := d.Token()
	if err != nil || fin != json.Delim('}') || d.Decode(new(any)) != io.EOF {
		return nil, os.ErrInvalid
	}
	return objeto, nil
}

func soloCampos(objeto map[string]json.RawMessage, permitidos map[string]bool) bool {
	for clave := range objeto {
		if !permitidos[clave] {
			return false
		}
	}
	return true
}

func cadena(objeto map[string]json.RawMessage, clave string) (string, bool) {
	valor, ok := objeto[clave]
	if !ok || bytes.Equal(valor, []byte("null")) {
		return "", false
	}
	var texto string
	if json.Unmarshal(valor, &texto) != nil || texto == "" {
		return "", false
	}
	return texto, true
}

func opcionalCadena(objeto map[string]json.RawMessage, clave string) (string, bool) {
	if _, existe := objeto[clave]; !existe {
		return "", true
	}
	return cadena(objeto, clave)
}

func entero(objeto map[string]json.RawMessage, clave string, maximo int64) (int64, bool) {
	valor, ok := objeto[clave]
	if !ok || bytes.Equal(valor, []byte("null")) {
		return 0, false
	}
	var numero int64
	if json.Unmarshal(valor, &numero) != nil || numero < 0 || numero > maximo {
		return 0, false
	}
	return numero, true
}

func segundos(objeto map[string]json.RawMessage, clave string) (float64, bool) {
	valor, ok := objeto[clave]
	if !ok || bytes.Equal(valor, []byte("null")) {
		return 0, false
	}
	var numero float64
	if json.Unmarshal(valor, &numero) != nil || math.IsNaN(numero) || math.IsInf(numero, 0) || numero < 0 || numero > maximoSegundos {
		return 0, false
	}
	return redondear(numero), true
}

func opcionalSegundos(objeto map[string]json.RawMessage, clave string) (float64, bool) {
	if _, existe := objeto[clave]; !existe {
		return 0, true
	}
	return segundos(objeto, clave)
}

func opcionalVerdadero(objeto map[string]json.RawMessage, clave string) (bool, bool) {
	valor, existe := objeto[clave]
	if !existe {
		return false, true
	}
	return bytes.Equal(valor, []byte("true")), bytes.Equal(valor, []byte("true"))
}

var errInstanteTecnicoInvalido = errors.New("instante_tecnico_invalido")

func cabeceraTecnica(objeto map[string]json.RawMessage) (time.Time, string, error) {
	fecha, okFecha := cadena(objeto, "time")
	nivel, okNivel := cadena(objeto, "level")
	servicio, okServicio := cadena(objeto, "service.name")
	version, okVersion := cadena(objeto, "service.version")
	entorno, okEntorno := cadena(objeto, "deployment.environment.name")
	superficie, okSuperficie := cadena(objeto, "vec.superficie")
	instante, err := time.Parse(time.RFC3339Nano, fecha)
	if err != nil || !okFecha || instante.IsZero() {
		return time.Time{}, "", errInstanteTecnicoInvalido
	}
	if !okNivel || !okServicio || !okVersion || !okEntorno || !okSuperficie ||
		(nivel != "INFO" && nivel != "WARN" && nivel != "ERROR") ||
		(servicio != "vec-server" && servicio != "vec-admin" && servicio != "vec-publico") ||
		version != domain.NormalizarVersionBinario(version) ||
		entorno != string(domain.NormalizarEntornoIncidenciaTecnica(entorno)) ||
		!superficieValida(superficie) {
		return time.Time{}, "", os.ErrInvalid
	}
	return instante.UTC(), nivel, nil
}

func superficieValida(valor string) bool {
	switch valor {
	case "interno", "externo", "integrada", "administracion", "publica":
		return true
	}
	return false
}

func validarAcceso(objeto map[string]json.RawMessage) (registroConsulta, error) {
	if !soloCampos(objeto, camposAcceso) {
		return registroConsulta{}, os.ErrInvalid
	}
	if fases, presente := objeto["vec.fases"]; presente {
		if err := validarFasesAcceso(fases); err != nil {
			return registroConsulta{}, err
		}
	}
	instante, nivel, errCabecera := cabeceraTecnica(objeto)
	if errCabecera != nil {
		return registroConsulta{}, errCabecera
	}
	metodo, okMetodo := cadena(objeto, "http.request.method")
	ruta, okRuta := cadena(objeto, "url.path")
	estado, okEstado := entero(objeto, "http.response.status_code", 599)
	duracion, okDuracion := segundos(objeto, "http.server.request.duration")
	_, okTamano := entero(objeto, "http.response.body.size", 1<<40)
	consultas, okConsultas := entero(objeto, "vec.bd.consultas", 1_000_000)
	duracionBD, okBD := segundos(objeto, "vec.bd.duracion")
	espera, okEspera := opcionalSegundos(objeto, "vec.bd.espera_conexion")
	lenta, okLenta := opcionalVerdadero(objeto, "vec.lenta")
	_, okInterrumpida := opcionalVerdadero(objeto, "vec.interrumpida")
	correlacion, okCorrelacion := opcionalCadena(objeto, "vec.correlacion")
	if !okMetodo || !okRuta || !okEstado || !okDuracion || !okTamano || !okConsultas ||
		!okBD || !okEspera || !okLenta || !okInterrumpida || !okCorrelacion ||
		estado < 100 || !metodoValido(metodo) ||
		(estado >= 400 && estado < 500 && ruta != "{oculto}") ||
		(estado < 400 || estado >= 500) && !rutaFija(ruta) ||
		(correlacion != "" && !domain.EsCorrelacionTecnicaValida(correlacion)) {
		return registroConsulta{}, os.ErrInvalid
	}
	if err := validarMedidasSQLAcceso(objeto, consultas, lenta, int(estado)); err != nil {
		return registroConsulta{}, err
	}
	if estado >= 500 && nivel != "ERROR" || estado < 500 && lenta && nivel != "WARN" ||
		estado < 500 && !lenta && nivel != "INFO" {
		return registroConsulta{}, os.ErrInvalid
	}
	if sql, existe := objeto["vec.bd.consulta_mas_lenta"]; existe {
		var operacion string
		if json.Unmarshal(sql, &operacion) != nil || !operacionSQLFija(operacion) || !lenta {
			return registroConsulta{}, os.ErrInvalid
		}
		if _, ok := segundos(objeto, "vec.bd.consulta_mas_lenta.duracion"); !ok {
			return registroConsulta{}, os.ErrInvalid
		}
	} else if _, existe := objeto["vec.bd.consulta_mas_lenta.duracion"]; existe {
		return registroConsulta{}, os.ErrInvalid
	}
	if cancelada, existe := objeto["vec.cancelada"]; existe {
		var valor string
		if json.Unmarshal(cancelada, &valor) != nil || (valor != "cliente" && valor != "plazo") {
			return registroConsulta{}, os.ErrInvalid
		}
	}
	claseBD, okBDClase := opcionalCadena(objeto, "vec.bd.error")
	if !okBDClase {
		return registroConsulta{}, os.ErrInvalid
	}
	if claseBD != "" {
		var valida bool
		claseBD, valida = claseErrorCerrada(claseBD)
		if !valida {
			return registroConsulta{}, os.ErrInvalid
		}
	}
	claseHTTP, okHTTPClase := opcionalCadena(objeto, "error.type")
	if !okHTTPClase || estado >= 500 && claseHTTP == "" || estado < 500 && claseHTTP != "" {
		return registroConsulta{}, os.ErrInvalid
	}
	if claseHTTP != "" {
		var valida bool
		claseHTTP, valida = claseErrorCerrada(claseHTTP)
		if !valida {
			return registroConsulta{}, os.ErrInvalid
		}
	}
	if estado >= 500 && (claseBD != "" && claseHTTP != claseBD ||
		claseBD == "" && claseHTTP != strconv.FormatInt(estado, 10)) {
		return registroConsulta{}, os.ErrInvalid
	}
	clase := claseHTTP
	if clase == "" {
		clase = claseBD
	}
	return registroConsulta{familia: "peticion", instante: instante, ruta: ruta, estado: int(estado),
		duracion: duracion, duracionBD: duracionBD, esperaPool: espera,
		consultas: consultas, lenta: lenta, claseError: clase}, nil
}

func validarArranque(objeto map[string]json.RawMessage) (registroConsulta, error) {
	if !soloCampos(objeto, camposArranque) {
		return registroConsulta{}, os.ErrInvalid
	}
	instante, nivel, errCabecera := cabeceraTecnica(objeto)
	if errCabecera != nil {
		return registroConsulta{}, errCabecera
	}
	servicio, okServicio := cadena(objeto, "service.name")
	fase, okFase := cadena(objeto, "vec.arranque.fase")
	resultado, okResultado := cadena(objeto, "vec.arranque.resultado")
	duracion, okDuracion := segundos(objeto, "vec.arranque.duracion")
	clase, okClase := opcionalCadena(objeto, "error.type")
	if !okServicio || !okFase || !okResultado || !okDuracion || !okClase ||
		(servicio != "vec-server" && servicio != "vec-admin") ||
		(fase != "configuracion" && fase != "composicion" && fase != "escucha") ||
		(resultado != "preparada" && resultado != "fallida") ||
		(resultado == "fallida" && (nivel != "ERROR" || clase == "")) ||
		(resultado == "preparada" && (nivel != "INFO" || clase != "")) {
		return registroConsulta{}, os.ErrInvalid
	}
	if clase != "" {
		var valida bool
		clase, valida = claseErrorCerrada(clase)
		if !valida {
			return registroConsulta{}, os.ErrInvalid
		}
	}
	return registroConsulta{familia: "arranque", instante: instante, fallida: resultado == "fallida",
		duracion: duracion, claseError: clase}, nil
}

func metodoValido(valor string) bool {
	switch valor {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "_OTHER":
		return true
	}
	return false
}

func operacionSQLFija(valor string) bool {
	if len(valor) == 0 || len(valor) > 127 {
		return false
	}
	for _, c := range valor {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '.' {
			continue
		}
		return false
	}
	return true
}

func claseErrorCerrada(valor string) (string, bool) {
	switch valor {
	case "configuracion", "cancelada", "plazo_vencido", "red", "otro", "conexion_cancelada",
		"conexion_plazo_vencido", "conexion_red", "conexion_otro":
		return valor, true
	}
	if len(valor) == 3 && valor[0] == '5' && valor[1] >= '0' && valor[1] <= '9' && valor[2] >= '0' && valor[2] <= '9' {
		return valor, true
	}
	prefijo := "bd_"
	conexion := false
	if strings.HasPrefix(valor, "conexion_bd_") {
		prefijo, conexion = "conexion_bd_", true
	}
	if !strings.HasPrefix(valor, prefijo) || len(valor) != len(prefijo)+5 {
		return "", false
	}
	codigo := strings.TrimPrefix(valor, prefijo)
	if strings.Trim(codigo, "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "" {
		return "", false
	}
	switch codigo {
	case "42501", "57014", "53300", "40001", "40P01", "08006", "23505":
		return valor, true
	}
	if conexion {
		return "conexion_bd_otro", true
	}
	return "bd_otro", true
}
