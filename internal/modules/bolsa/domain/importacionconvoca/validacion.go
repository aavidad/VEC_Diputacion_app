package importacionconvoca

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	campoFila                  = "fila"
	codigoColumnasInvalidas    = "numero_columnas_invalido"
	codigoValorRequerido       = "valor_requerido"
	codigoTipoCeldaInvalido    = "tipo_celda_invalido"
	codigoFormulaProhibida     = "formula_prohibida"
	codigoTextoInvalido        = "texto_invalido"
	codigoTextoExcesivo        = "texto_demasiado_largo"
	codigoDocumentoEnmascarado = "documento_enmascarado_invalido"
	codigoDecimalInvalido      = "decimal_invalido"
	codigoOrdenGrupoInvalido   = "orden_grupo_invalido"
	codigoTotalIncoherente     = "total_incoherente"
)

var decimalConvoca = regexp.MustCompile(`^[0-9]{1,9}([.,][0-9]{1,4})?$`)

// ValidarHoja ejecuta la zona de ensayo. Las filas validas y las incidencias
// se separan antes de que el repositorio pueda confirmar el lote.
func ValidarHoja(hoja HojaStaging) (ResultadoStaging, error) {
	if err := hoja.ValidarEstructura(); err != nil {
		return ResultadoStaging{}, err
	}
	resultado := ResultadoStaging{
		Aceptadas:   make([]FilaAceptada, 0, len(hoja.Filas)),
		Incidencias: make([]Incidencia, 0),
	}
	for _, fila := range hoja.Filas {
		if filaVacia(fila) {
			continue
		}
		resultado.FilasLeidas++
		aceptada, incidencias := validarFila(hoja.Esquema, fila)
		if len(incidencias) != 0 {
			resultado.Rechazadas++
			resultado.Incidencias = append(resultado.Incidencias, incidencias...)
			continue
		}
		resultado.Aceptadas = append(resultado.Aceptadas, aceptada)
	}
	return resultado, nil
}

func validarFila(esquema EsquemaExportacion, fila FilaStaging) (FilaAceptada, []Incidencia) {
	incidencias := make([]Incidencia, 0, 4)
	if len(fila.Celdas) > esquema.NumeroColumnas() {
		incidencias = append(incidencias, incidencia(fila.Numero, campoFila, codigoColumnasInvalidas))
	}
	documento, codigo := leerTexto(fila, 0, true, 16)
	if codigo != "" {
		incidencias = append(incidencias, incidencia(fila.Numero, "DNI/NIE", codigo))
	} else if !documentoEnmascaradoValido(documento) {
		incidencias = append(incidencias, incidencia(fila.Numero, "DNI/NIE", codigoDocumentoEnmascarado))
	}
	primerApellido, codigo := leerTexto(fila, 1, true, 120)
	if codigo != "" {
		incidencias = append(incidencias, incidencia(fila.Numero, "Primer Apellido", codigo))
	}
	segundoApellido, codigo := leerTexto(fila, 2, false, 120)
	if codigo != "" {
		incidencias = append(incidencias, incidencia(fila.Numero, "Segundo Apellido", codigo))
	}
	nombre, codigo := leerTexto(fila, 3, true, 120)
	if codigo != "" {
		incidencias = append(incidencias, incidencia(fila.Numero, "Nombre", codigo))
	}
	turno, codigo := leerTexto(fila, 4, true, 120)
	if codigo != "" {
		incidencias = append(incidencias, incidencia(fila.Numero, "Turno", codigo))
	}

	aceptada := FilaAceptada{
		Numero: fila.Numero, Esquema: esquema,
		Identidad: IdentidadEnmascarada{
			Documento: documento, PrimerApellido: primerApellido,
			SegundoApellido: segundoApellido, Nombre: nombre,
		},
		Turno: turno,
	}
	switch esquema {
	case EsquemaResumenPersona:
		validarResumen(fila, &aceptada, &incidencias)
	case EsquemaDetalleMerito:
		validarDetalle(fila, &aceptada, &incidencias)
	default:
		incidencias = append(incidencias, incidencia(fila.Numero, campoFila, codigoColumnasInvalidas))
	}
	if len(incidencias) != 0 {
		return FilaAceptada{}, incidencias
	}
	return aceptada, nil
}

func validarResumen(fila FilaStaging, aceptada *FilaAceptada, incidencias *[]Incidencia) {
	experiencia, codigoExperiencia := leerDecimal(fila, 5, true)
	formacion, codigoFormacion := leerDecimal(fila, 6, true)
	total, codigoTotal := leerDecimal(fila, 7, true)
	for _, dato := range []struct{ campo, codigo string }{
		{"Experiencia", codigoExperiencia}, {"Formacion", codigoFormacion}, {"Total", codigoTotal},
	} {
		if dato.codigo != "" {
			*incidencias = append(*incidencias, incidencia(fila.Numero, dato.campo, dato.codigo))
		}
	}
	if codigoExperiencia == "" && codigoFormacion == "" && codigoTotal == "" &&
		!sumaDecimalCoincide(experiencia, formacion, total) {
		*incidencias = append(*incidencias, incidencia(fila.Numero, "Total", codigoTotalIncoherente))
	}
	aceptada.Resumen = &ResumenPersona{Experiencia: experiencia, Formacion: formacion, Total: total}
}

func validarDetalle(fila FilaStaging, aceptada *FilaAceptada, incidencias *[]Incidencia) {
	grupo, codigoGrupo := leerTexto(fila, 5, true, 120)
	descripcionGrupo, codigoDescripcionGrupo := leerTexto(fila, 6, true, 500)
	ordenGrupo, codigoOrden := leerOrdenGrupo(fila, 7)
	descripcionMerito, codigoDescripcionMerito := leerTexto(fila, 8, true, 1000)
	puntosAuto, codigoAuto := leerDecimal(fila, 9, false)
	puntosTribunal, codigoTribunal := leerDecimal(fila, 10, false)
	motivo, codigoMotivo := leerTexto(fila, 11, false, 1000)
	for _, dato := range []struct{ campo, codigo string }{
		{"Grupo", codigoGrupo}, {"Descripcion del grupo", codigoDescripcionGrupo},
		{"Orden grupo", codigoOrden}, {"Descripcion del merito", codigoDescripcionMerito},
		{"Puntos autobaremacion", codigoAuto}, {"Puntos tribunal", codigoTribunal},
		{"Motivo", codigoMotivo},
	} {
		if dato.codigo != "" {
			*incidencias = append(*incidencias, incidencia(fila.Numero, dato.campo, dato.codigo))
		}
	}
	aceptada.Detalle = &DetalleMerito{
		Grupo: grupo, DescripcionGrupo: descripcionGrupo, OrdenGrupo: ordenGrupo,
		DescripcionMerito:              descripcionMerito,
		PuntosAutobaremacionHistoricos: puntosAuto,
		PuntosTribunal:                 puntosTribunal, Motivo: motivo,
	}
}

func leerTexto(fila FilaStaging, columna int, requerido bool, maximo int) (string, string) {
	celda := obtenerCelda(fila, columna)
	if celda.Tipo == CeldaFormula {
		return "", codigoFormulaProhibida
	}
	if celda.Tipo != CeldaVacia && celda.Tipo != CeldaTexto {
		return "", codigoTipoCeldaInvalido
	}
	valor := norm.NFC.String(strings.TrimSpace(celda.Valor))
	if valor == "" {
		if requerido {
			return "", codigoValorRequerido
		}
		return "", ""
	}
	if !utf8.ValidString(valor) || textoPeligroso(valor) {
		return "", codigoTextoInvalido
	}
	if utf8.RuneCountInString(valor) > maximo {
		return "", codigoTextoExcesivo
	}
	return valor, ""
}

func leerDecimal(fila FilaStaging, columna int, requerido bool) (string, string) {
	celda := obtenerCelda(fila, columna)
	if celda.Tipo == CeldaFormula {
		return "", codigoFormulaProhibida
	}
	if celda.Tipo != CeldaVacia && celda.Tipo != CeldaTexto && celda.Tipo != CeldaNumero {
		return "", codigoTipoCeldaInvalido
	}
	valor := strings.TrimSpace(celda.Valor)
	if valor == "" {
		if requerido {
			return "", codigoValorRequerido
		}
		return "", ""
	}
	if !decimalConvoca.MatchString(valor) {
		return "", codigoDecimalInvalido
	}
	return normalizarDecimal(valor), ""
}

func leerOrdenGrupo(fila FilaStaging, columna int) (uint32, string) {
	celda := obtenerCelda(fila, columna)
	if celda.Tipo == CeldaFormula {
		return 0, codigoFormulaProhibida
	}
	if celda.Tipo != CeldaTexto && celda.Tipo != CeldaNumero {
		if celda.Tipo == CeldaVacia {
			return 0, codigoValorRequerido
		}
		return 0, codigoTipoCeldaInvalido
	}
	valor := strings.TrimSpace(celda.Valor)
	if valor == "" || strings.IndexFunc(valor, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return 0, codigoOrdenGrupoInvalido
	}
	orden, err := strconv.ParseUint(valor, 10, 32)
	if err != nil || orden == 0 {
		return 0, codigoOrdenGrupoInvalido
	}
	return uint32(orden), ""
}

func obtenerCelda(fila FilaStaging, columna int) CeldaStaging {
	if columna < 0 || columna >= len(fila.Celdas) {
		return CeldaStaging{Tipo: CeldaVacia}
	}
	return fila.Celdas[columna]
}

func filaVacia(fila FilaStaging) bool {
	for _, celda := range fila.Celdas {
		if celda.Tipo != CeldaVacia || strings.TrimSpace(celda.Valor) != "" {
			return false
		}
	}
	return true
}

func documentoEnmascaradoValido(valor string) bool {
	if len(valor) != 9 || valor[:3] != "***" || valor[7:] != "**" {
		return false
	}
	for _, b := range []byte(valor[3:7]) {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

func textoPeligroso(valor string) bool {
	for _, r := range valor {
		if unicode.IsControl(r) {
			return true
		}
	}
	primero := valor[0]
	if primero == '=' || primero == '+' || primero == '@' {
		return true
	}
	return primero == '-' && len(valor) > 1 &&
		((valor[1] >= '0' && valor[1] <= '9') || valor[1] == '=')
}

func normalizarDecimal(valor string) string {
	valor = strings.ReplaceAll(valor, ",", ".")
	partes := strings.SplitN(valor, ".", 2)
	entero := strings.TrimLeft(partes[0], "0")
	if entero == "" {
		entero = "0"
	}
	if len(partes) == 1 {
		return entero
	}
	fraccion := strings.TrimRight(partes[1], "0")
	if fraccion == "" {
		return entero
	}
	return entero + "." + fraccion
}

func sumaDecimalCoincide(a, b, total string) bool {
	ra, oka := new(big.Rat).SetString(a)
	rb, okb := new(big.Rat).SetString(b)
	rt, okt := new(big.Rat).SetString(total)
	return oka && okb && okt && new(big.Rat).Add(ra, rb).Cmp(rt) == 0
}

func incidencia(fila int, campo, codigo string) Incidencia {
	return Incidencia{Fila: fila, Campo: campo, Codigo: codigo}
}
