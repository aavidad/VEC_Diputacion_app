package xlsconvoca

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"path"
	"strconv"
	"strings"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

const maximoEntradasXLSX = 512

// El ZIP se consume entero y cada tamaño real se mide antes de interpretar XML.
// No se extrae ninguna entrada a disco ni se resuelve ningún Target externo.
func decodificarXLSX(ctx context.Context, origen io.ReadSeeker) (dominio.HojaStaging, error) {
	tamano, err := origen.Seek(0, io.SeekEnd)
	if err != nil || tamano < 4 {
		return dominio.HojaStaging{}, ErrXLSInvalido
	}
	if tamano > maximoBytesXLS {
		return dominio.HojaStaging{}, ErrLimiteXLSExcedido
	}
	if _, err := origen.Seek(0, io.SeekStart); err != nil {
		return dominio.HojaStaging{}, ErrXLSInvalido
	}
	contenido := make([]byte, int(tamano))
	if _, err := io.ReadFull(origen, contenido); err != nil {
		return dominio.HojaStaging{}, ErrXLSInvalido
	}
	archivo, err := zip.NewReader(bytes.NewReader(contenido), tamano)
	if err != nil {
		return dominio.HojaStaging{}, ErrXLSInvalido
	}
	if len(archivo.File) == 0 || len(archivo.File) > maximoEntradasXLSX {
		return dominio.HojaStaging{}, ErrLimiteXLSExcedido
	}
	var declarado uint64
	for _, entrada := range archivo.File {
		if err := ctx.Err(); err != nil {
			return dominio.HojaStaging{}, err
		}
		nombreLimpio := strings.TrimSuffix(entrada.Name, "/")
		if entrada.Flags&1 != 0 || (entrada.Method != zip.Store && entrada.Method != zip.Deflate) ||
			nombreLimpio == "" || strings.ContainsAny(entrada.Name, "\\\x00") ||
			strings.HasPrefix(entrada.Name, "/") || path.Clean(nombreLimpio) != nombreLimpio ||
			(strings.HasSuffix(entrada.Name, "/") && entrada.UncompressedSize64 != 0) {
			return dominio.HojaStaging{}, ErrXLSInvalido
		}
		if nombreXLSXPeligroso(entrada.Name) {
			return dominio.HojaStaging{}, ErrXLSInvalido
		}
		if entrada.UncompressedSize64 > maximoBytesXLS || declarado > uint64(maximoBytesXLS)-entrada.UncompressedSize64 {
			return dominio.HojaStaging{}, ErrLimiteXLSExcedido
		}
		declarado += entrada.UncompressedSize64
	}
	partes := make(map[string][]byte, len(archivo.File))
	var real int64
	for _, entrada := range archivo.File {
		if err := ctx.Err(); err != nil {
			return dominio.HojaStaging{}, err
		}
		if _, existe := partes[entrada.Name]; existe {
			return dominio.HojaStaging{}, ErrXLSInvalido
		}
		lector, err := entrada.Open()
		if err != nil {
			return dominio.HojaStaging{}, ErrXLSInvalido
		}
		var destino bytes.Buffer
		guardar := entrada.Name == "xl/workbook.xml" || entrada.Name == "xl/_rels/workbook.xml.rels" ||
			entrada.Name == "xl/sharedStrings.xml" || entrada.Name == "xl/styles.xml" ||
			entrada.Name == "[Content_Types].xml" || strings.HasSuffix(entrada.Name, ".rels") ||
			strings.HasPrefix(entrada.Name, "xl/worksheets/")
		var salida io.Writer = io.Discard
		if guardar {
			salida = &destino
		}
		restante := int64(maximoBytesXLS) - real
		n, copiaErr := io.Copy(salida, io.LimitReader(lector, restante+1))
		if copiaErr != nil || n > restante {
			lector.Close()
			if n > restante {
				return dominio.HojaStaging{}, ErrLimiteXLSExcedido
			}
			return dominio.HojaStaging{}, ErrXLSInvalido
		}
		var sobrante [1]byte
		otro, finErr := lector.Read(sobrante[:])
		cerrarErr := lector.Close()
		if otro != 0 || finErr != io.EOF || cerrarErr != nil || uint64(n) != entrada.UncompressedSize64 {
			return dominio.HojaStaging{}, ErrXLSInvalido
		}
		real += n
		if guardar {
			partes[entrada.Name] = destino.Bytes()
		} else {
			partes[entrada.Name] = nil
		}
	}
	if err := validarPartesXLSX(partes); err != nil {
		return dominio.HojaStaging{}, err
	}
	nombre, relacion, err := leerLibroXLSX(partes["xl/workbook.xml"])
	if err != nil {
		return dominio.HojaStaging{}, err
	}
	ruta, err := rutaHojaXLSX(partes["xl/_rels/workbook.xml.rels"], relacion)
	if err != nil {
		return dominio.HojaStaging{}, err
	}
	if err := verificarTipoHojaXLSX(partes["[Content_Types].xml"], ruta); err != nil {
		return dominio.HojaStaging{}, err
	}
	datosHoja, existe := partes[ruta]
	if !existe || datosHoja == nil {
		return dominio.HojaStaging{}, ErrXLSInvalido
	}
	compartidas, err := leerCompartidasXLSX(partes["xl/sharedStrings.xml"])
	if err != nil {
		return dominio.HojaStaging{}, err
	}
	estilos, err := leerEstilosXLSX(partes["xl/styles.xml"])
	if err != nil {
		return dominio.HojaStaging{}, err
	}
	cabeceras, filas, err := leerHojaXLSX(ctx, datosHoja, compartidas, estilos)
	if err != nil {
		return dominio.HojaStaging{}, err
	}
	esquema, formato, err := dominio.DetectarFormato(cabeceras)
	if err != nil {
		return dominio.HojaStaging{}, err
	}
	if err := dominio.ValidarNombreHoja(esquema, formato, nombre); err != nil {
		return dominio.HojaStaging{}, err
	}
	return dominio.HojaStaging{Esquema: esquema, Cabeceras: cabeceras, NombreHoja: nombre, Filas: filas}, nil
}

func nombreXLSXPeligroso(nombre string) bool {
	n := strings.ToLower(nombre)
	return strings.Contains(n, "vbaproject") || strings.Contains(n, "macros") ||
		strings.HasPrefix(n, "xl/externallinks/") || strings.HasPrefix(n, "xl/embeddings/") ||
		strings.HasPrefix(n, "xl/activex/") || strings.HasPrefix(n, "xl/connections") ||
		strings.HasPrefix(n, "xl/querytables/")
}

func validarPartesXLSX(partes map[string][]byte) error {
	if partes["xl/workbook.xml"] == nil || partes["xl/_rels/workbook.xml.rels"] == nil ||
		partes["_rels/.rels"] == nil || partes["[Content_Types].xml"] == nil {
		return ErrXLSInvalido
	}
	if err := verificarRaizXLSX(partes["_rels/.rels"]); err != nil {
		return err
	}
	if err := verificarTipoLibroXLSX(partes["[Content_Types].xml"]); err != nil {
		return err
	}
	for nombre, datos := range partes {
		if nombre == "[Content_Types].xml" {
			if err := verificarTiposContenidoXLSX(datos); err != nil {
				return err
			}
		}
		if strings.HasSuffix(nombre, ".rels") {
			if err := verificarRelacionesXLSX(datos); err != nil {
				return err
			}
		}
	}
	return verificarRelacionesPartesLibroXLSX(partes)
}

func verificarRelacionesPartesLibroXLSX(partes map[string][]byte) error {
	dec := xml.NewDecoder(bytes.NewReader(partes["xl/_rels/workbook.xml.rels"]))
	var estilos, compartidas bool
	for {
		token, err := siguienteTokenXLSX(dec)
		if err == io.EOF {
			break
		}
		if err != nil {
			return ErrXLSInvalido
		}
		inicio, ok := token.(xml.StartElement)
		if !ok || inicio.Name.Local != "Relationship" {
			continue
		}
		tipo, destino := atributo(inicio.Attr, "Type"), atributo(inicio.Attr, "Target")
		switch {
		case strings.HasSuffix(tipo, "/styles"):
			if estilos || destino != "styles.xml" || atributo(inicio.Attr, "TargetMode") != "" {
				return ErrXLSInvalido
			}
			estilos = true
		case strings.HasSuffix(tipo, "/sharedStrings"):
			if compartidas || destino != "sharedStrings.xml" || atributo(inicio.Attr, "TargetMode") != "" {
				return ErrXLSInvalido
			}
			compartidas = true
		}
	}
	_, hayEstilos := partes["xl/styles.xml"]
	_, hayCompartidas := partes["xl/sharedStrings.xml"]
	if estilos != hayEstilos || compartidas != hayCompartidas {
		return ErrXLSInvalido
	}
	if estilos {
		if partes["xl/styles.xml"] == nil {
			return ErrXLSInvalido
		}
		if err := verificarTipoParteXLSX(partes["[Content_Types].xml"], "/xl/styles.xml",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"); err != nil {
			return err
		}
	}
	if compartidas {
		if partes["xl/sharedStrings.xml"] == nil {
			return ErrXLSInvalido
		}
		if err := verificarTipoParteXLSX(partes["[Content_Types].xml"], "/xl/sharedStrings.xml",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"); err != nil {
			return err
		}
	}
	return nil
}

func verificarRaizXLSX(datos []byte) error {
	dec := xml.NewDecoder(bytes.NewReader(datos))
	cuenta := 0
	for {
		token, err := siguienteTokenXLSX(dec)
		if err == io.EOF {
			if cuenta == 1 {
				return nil
			}
			return ErrXLSInvalido
		}
		if err != nil {
			return ErrXLSInvalido
		}
		inicio, ok := token.(xml.StartElement)
		if !ok || inicio.Name.Local != "Relationship" ||
			!strings.HasSuffix(atributo(inicio.Attr, "Type"), "/officeDocument") {
			continue
		}
		cuenta++
		if cuenta > 1 || atributo(inicio.Attr, "Target") != "xl/workbook.xml" ||
			atributo(inicio.Attr, "TargetMode") != "" {
			return ErrXLSInvalido
		}
	}
}

func verificarTipoLibroXLSX(datos []byte) error {
	return verificarTipoParteXLSX(datos, "/xl/workbook.xml",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml")
}

func verificarTipoHojaXLSX(datos []byte, ruta string) error {
	return verificarTipoParteXLSX(datos, "/"+ruta,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml")
}

func verificarTipoParteXLSX(datos []byte, nombre, esperado string) error {
	dec := xml.NewDecoder(bytes.NewReader(datos))
	coincidencias := 0
	for {
		token, err := siguienteTokenXLSX(dec)
		if err == io.EOF {
			if coincidencias == 1 {
				return nil
			}
			return ErrXLSInvalido
		}
		if err != nil {
			return ErrXLSInvalido
		}
		inicio, ok := token.(xml.StartElement)
		if !ok || inicio.Name.Local != "Override" || atributo(inicio.Attr, "PartName") != nombre {
			continue
		}
		coincidencias++
		if coincidencias > 1 || atributo(inicio.Attr, "ContentType") != esperado {
			return ErrXLSInvalido
		}
	}
}

func tipoEjecutableXLSX(valor string) bool {
	valor = strings.ToLower(valor)
	return strings.Contains(valor, "macro") || strings.Contains(valor, "vba") ||
		strings.Contains(valor, "externallink") || strings.Contains(valor, "activex")
}

func verificarTiposContenidoXLSX(datos []byte) error {
	dec := xml.NewDecoder(bytes.NewReader(datos))
	for {
		token, err := siguienteTokenXLSX(dec)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return ErrXLSInvalido
		}
		inicio, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if tipoEjecutableXLSX(atributo(inicio.Attr, "ContentType")) ||
			tipoEjecutableXLSX(atributo(inicio.Attr, "PartName")) {
			return ErrXLSInvalido
		}
	}
}

func verificarRelacionesXLSX(datos []byte) error {
	dec := xml.NewDecoder(bytes.NewReader(datos))
	for {
		token, err := siguienteTokenXLSX(dec)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return ErrXLSInvalido
		}
		if inicio, ok := token.(xml.StartElement); ok && inicio.Name.Local == "Relationship" {
			if strings.EqualFold(strings.TrimSpace(atributo(inicio.Attr, "TargetMode")), "External") ||
				tipoEjecutableXLSX(atributo(inicio.Attr, "Type")) ||
				tipoEjecutableXLSX(atributo(inicio.Attr, "Target")) {
				return ErrXLSInvalido
			}
		}
	}
}

func siguienteTokenXLSX(dec *xml.Decoder) (xml.Token, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if _, directiva := token.(xml.Directive); directiva {
		return nil, ErrXLSInvalido
	}
	return token, nil
}

func atributo(atributos []xml.Attr, nombre string) string {
	for _, a := range atributos {
		if a.Name.Local == nombre {
			return a.Value
		}
	}
	return ""
}

func leerLibroXLSX(datos []byte) (string, string, error) {
	dec := xml.NewDecoder(bytes.NewReader(datos))
	nombre, relacion, hojas := "", "", 0
	for {
		token, err := siguienteTokenXLSX(dec)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", ErrXLSInvalido
		}
		if inicio, ok := token.(xml.StartElement); ok {
			if inicio.Name.Local == "definedName" || inicio.Name.Local == "externalReferences" {
				return "", "", ErrXLSInvalido
			}
			if inicio.Name.Local == "sheet" {
				hojas++
				nombre = atributo(inicio.Attr, "name")
				relacion = atributo(inicio.Attr, "id")
			}
		}
	}
	if hojas != 1 || nombre == "" || relacion == "" || len(nombre) > maximoBytesCelda {
		return "", "", ErrXLSInvalido
	}
	return nombre, relacion, nil
}

func rutaHojaXLSX(datos []byte, id string) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(datos))
	ruta := ""
	for {
		token, err := siguienteTokenXLSX(dec)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", ErrXLSInvalido
		}
		inicio, ok := token.(xml.StartElement)
		if !ok || inicio.Name.Local != "Relationship" || atributo(inicio.Attr, "Id") != id {
			continue
		}
		if ruta != "" || !strings.HasSuffix(atributo(inicio.Attr, "Type"), "/worksheet") {
			return "", ErrXLSInvalido
		}
		destino := atributo(inicio.Attr, "Target")
		if strings.HasPrefix(destino, "/") {
			ruta = path.Clean(strings.TrimPrefix(destino, "/"))
		} else {
			ruta = path.Clean("xl/" + destino)
		}
	}
	if !strings.HasPrefix(ruta, "xl/worksheets/") || !strings.HasSuffix(ruta, ".xml") {
		return "", ErrXLSInvalido
	}
	return ruta, nil
}

func agregarTextoXLSX(destino *strings.Builder, texto []byte) error {
	if len(texto) > maximoBytesCelda-destino.Len() {
		return ErrLimiteXLSExcedido
	}
	_, _ = destino.Write(texto)
	return nil
}

func leerTextoXLSX(dec *xml.Decoder, fin string) (string, error) {
	var texto strings.Builder
	for {
		token, err := siguienteTokenXLSX(dec)
		if err != nil {
			return "", ErrXLSInvalido
		}
		switch t := token.(type) {
		case xml.CharData:
			if err := agregarTextoXLSX(&texto, t); err != nil {
				return "", err
			}
		case xml.EndElement:
			if t.Name.Local == fin {
				return texto.String(), nil
			}
		case xml.StartElement:
			return "", ErrXLSInvalido
		}
	}
}

func leerCadenaXLSX(dec *xml.Decoder, fin string) (string, error) {
	var cadena strings.Builder
	enFonetica := false
	for {
		token, err := siguienteTokenXLSX(dec)
		if err != nil {
			return "", ErrXLSInvalido
		}
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "rPh" {
				enFonetica = true
			} else if t.Name.Local == "t" && !enFonetica {
				texto, err := leerTextoXLSX(dec, "t")
				if err != nil {
					return "", err
				}
				if err := agregarTextoXLSX(&cadena, []byte(texto)); err != nil {
					return "", err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "rPh" {
				enFonetica = false
			}
			if t.Name.Local == fin {
				return cadena.String(), nil
			}
		}
	}
}

func leerCompartidasXLSX(datos []byte) ([]string, error) {
	if datos == nil {
		return nil, nil
	}
	dec := xml.NewDecoder(bytes.NewReader(datos))
	var cadenas []string
	for {
		token, err := siguienteTokenXLSX(dec)
		if err == io.EOF {
			return cadenas, nil
		}
		if err != nil {
			return nil, ErrXLSInvalido
		}
		inicio, ok := token.(xml.StartElement)
		if !ok || inicio.Name.Local != "si" {
			continue
		}
		valor, err := leerCadenaXLSX(dec, "si")
		if err != nil {
			return nil, err
		}
		if len(cadenas) >= maximoFilasXLS*maximoColumnasXLS {
			return nil, ErrLimiteXLSExcedido
		}
		cadenas = append(cadenas, valor)
	}
}

func numeroAtributo(atributos []xml.Attr, nombre string) (int, error) {
	valor := atributo(atributos, nombre)
	numero, err := strconv.Atoi(valor)
	if err != nil || numero < 0 {
		return 0, ErrXLSInvalido
	}
	return numero, nil
}

func leerEstilosXLSX(datos []byte) ([]bool, error) {
	if datos == nil {
		return []bool{false}, nil
	}
	dec := xml.NewDecoder(bytes.NewReader(datos))
	formatos := make(map[int]string)
	var estilos []bool
	enXfs := false
	for {
		token, err := siguienteTokenXLSX(dec)
		if err == io.EOF {
			if len(estilos) == 0 {
				return []bool{false}, nil
			}
			return estilos, nil
		}
		if err != nil {
			return nil, ErrXLSInvalido
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "cellXfs":
				enXfs = true
			case "numFmt":
				id, err := numeroAtributo(t.Attr, "numFmtId")
				if err != nil {
					return nil, err
				}
				formatos[id] = atributo(t.Attr, "formatCode")
			case "xf":
				if enXfs {
					id, err := numeroAtributo(t.Attr, "numFmtId")
					if err != nil {
						return nil, err
					}
					fecha, valido := esFormatoFechaXLSX(id, formatos)
					if !valido {
						return nil, ErrXLSInvalido
					}
					if len(estilos) >= maximoFilasXLS*maximoColumnasXLS {
						return nil, ErrLimiteXLSExcedido
					}
					estilos = append(estilos, fecha)
				}
			}
		case xml.EndElement:
			if t.Name.Local == "cellXfs" {
				enXfs = false
			}
		}
	}
}

func esFormatoFechaXLSX(id int, formatos map[int]string) (bool, bool) {
	if id >= 14 && id <= 22 || id >= 27 && id <= 36 || id >= 45 && id <= 47 ||
		id >= 50 && id <= 58 || id >= 71 && id <= 81 {
		return true, true
	}
	if id <= 13 || id >= 37 && id <= 44 || id == 48 || id == 49 {
		return false, true
	}
	if id < 164 {
		return true, true // Desconocido: tratarlo como fecha, nunca como puntos.
	}
	formato, existe := formatos[id]
	if !existe || formato == "" {
		return false, false
	}
	// Sólo aceptamos como numérico un formato que no contiene componentes de
	// fecha. Un falso positivo provoca rechazo de fila, nunca puntos inventados.
	enComillas, escapado, enCorchetes := false, false, false
	var corchete strings.Builder
	for _, r := range strings.ToLower(formato) {
		if escapado {
			escapado = false
			continue
		}
		if r == '\\' || r == '_' || r == '*' {
			escapado = true
			continue
		}
		if r == '"' {
			enComillas = !enComillas
			continue
		}
		if !enComillas && r == '[' {
			enCorchetes = true
			corchete.Reset()
			continue
		}
		if enCorchetes {
			if r == ']' {
				switch corchete.String() {
				case "h", "hh", "m", "mm", "s", "ss":
					return true, true
				}
				enCorchetes = false
			} else {
				corchete.WriteRune(r)
			}
			continue
		}
		if !enComillas && strings.ContainsRune("ydhsm", r) {
			return true, true
		}
	}
	if enCorchetes || enComillas || escapado {
		return true, true
	}
	return false, true
}

type celdaXLSX struct {
	Tipo    string
	Estilo  string
	Valor   string
	Formula *string
	Inline  string
}

func leerHojaXLSX(ctx context.Context, datos []byte, compartidas []string, estilos []bool) ([]string, []dominio.FilaStaging, error) {
	dec := xml.NewDecoder(bytes.NewReader(datos))
	var cabeceras []string
	var filas []dominio.FilaStaging
	ultimo, numero, ultimaColumna := 0, 0, 0
	var celdas []dominio.CeldaStaging
	for {
		token, err := siguienteTokenXLSX(dec)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, ErrXLSInvalido
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				if numero != 0 {
					return nil, nil, ErrXLSInvalido
				}
				if err := ctx.Err(); err != nil {
					return nil, nil, err
				}
				numero = ultimo + 1
				if ref := atributo(t.Attr, "r"); ref != "" {
					numero, err = strconv.Atoi(ref)
					if err != nil || numero <= ultimo || numero < 1 {
						return nil, nil, ErrXLSInvalido
					}
				}
				if numero > maximoFilasXLS {
					return nil, nil, ErrLimiteXLSExcedido
				}
				celdas, ultimaColumna = nil, 0
			case "c":
				if numero == 0 {
					return nil, nil, ErrXLSInvalido
				}
				columna := ultimaColumna + 1
				if ref := atributo(t.Attr, "r"); ref != "" {
					var filaRef int
					columna, filaRef, err = coordenadaXLSX(ref)
					if err != nil {
						return nil, nil, err
					}
					if columna < 1 || filaRef != numero || columna <= ultimaColumna {
						return nil, nil, ErrXLSInvalido
					}
				}
				if columna > maximoColumnasXLS {
					return nil, nil, ErrLimiteXLSExcedido
				}
				celda, err := leerCeldaXLSX(dec, t)
				if err != nil {
					return nil, nil, err
				}
				valor, err := convertirCeldaXLSX(celda, compartidas, estilos)
				if err != nil {
					return nil, nil, err
				}
				for len(celdas) < columna {
					celdas = append(celdas, dominio.CeldaStaging{Tipo: dominio.CeldaVacia})
				}
				celdas[columna-1] = valor
				ultimaColumna = columna
			}
		case xml.EndElement:
			if t.Name.Local != "row" {
				continue
			}
			if numero == 0 {
				return nil, nil, ErrXLSInvalido
			}
			if numero == 1 {
				if len(celdas) == 0 {
					return nil, nil, ErrXLSInvalido
				}
				cabeceras = make([]string, len(celdas))
				for i, celda := range celdas {
					if celda.Tipo != dominio.CeldaTexto {
						return nil, nil, ErrXLSInvalido
					}
					cabeceras[i] = celda.Valor
				}
			} else if cabeceras == nil {
				return nil, nil, ErrXLSInvalido
			} else {
				filas = append(filas, dominio.FilaStaging{Numero: numero, Celdas: celdas})
			}
			ultimo, numero = numero, 0
		}
	}
	if cabeceras == nil || numero != 0 {
		return nil, nil, ErrXLSInvalido
	}
	return cabeceras, filas, nil
}

func leerCeldaXLSX(dec *xml.Decoder, inicio xml.StartElement) (celdaXLSX, error) {
	c := celdaXLSX{Tipo: atributo(inicio.Attr, "t"), Estilo: atributo(inicio.Attr, "s")}
	for {
		token, err := siguienteTokenXLSX(dec)
		if err != nil {
			return celdaXLSX{}, ErrXLSInvalido
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "v":
				c.Valor, err = leerTextoXLSX(dec, "v")
			case "f":
				var formula string
				formula, err = leerTextoXLSX(dec, "f")
				c.Formula = &formula
			case "is":
				c.Inline, err = leerCadenaXLSX(dec, "is")
			default:
				return celdaXLSX{}, ErrXLSInvalido
			}
			if err != nil {
				return celdaXLSX{}, err
			}
		case xml.EndElement:
			if t.Name.Local == "c" {
				return c, nil
			}
		}
	}
}

func coordenadaXLSX(referencia string) (int, int, error) {
	columna, i := 0, 0
	for i < len(referencia) && referencia[i] >= 'A' && referencia[i] <= 'Z' {
		columna = columna*26 + int(referencia[i]-'A') + 1
		i++
		if columna > maximoColumnasXLS {
			return 0, 0, ErrLimiteXLSExcedido
		}
	}
	if i == 0 || i == len(referencia) {
		return 0, 0, ErrXLSInvalido
	}
	fila, err := strconv.Atoi(referencia[i:])
	if err != nil || fila < 1 {
		return 0, 0, ErrXLSInvalido
	}
	return columna, fila, nil
}

func convertirCeldaXLSX(c celdaXLSX, compartidas []string, estilos []bool) (dominio.CeldaStaging, error) {
	if c.Formula != nil {
		if !formulaInternaXLSX(*c.Formula) {
			return dominio.CeldaStaging{}, ErrXLSInvalido
		}
		return dominio.CeldaStaging{Tipo: dominio.CeldaFormula, Valor: *c.Formula}, nil
	}
	estilo := 0
	if c.Estilo != "" {
		var err error
		estilo, err = strconv.Atoi(c.Estilo)
		if err != nil || estilo < 0 || estilo >= len(estilos) {
			return dominio.CeldaStaging{}, ErrXLSInvalido
		}
	}
	tipo, valor := dominio.CeldaVacia, c.Valor
	switch c.Tipo {
	case "s":
		indice, err := strconv.Atoi(c.Valor)
		if err != nil || indice < 0 || indice >= len(compartidas) {
			return dominio.CeldaStaging{}, ErrXLSInvalido
		}
		tipo, valor = dominio.CeldaTexto, compartidas[indice]
	case "inlineStr":
		valor = c.Inline
		tipo = dominio.CeldaTexto
	case "str":
		tipo = dominio.CeldaTexto
	case "", "n":
		if valor != "" {
			tipo = dominio.CeldaNumero
			if estilos[estilo] {
				tipo = dominio.CeldaFecha
			}
		}
	case "b":
		tipo = dominio.CeldaLogica
	case "e":
		tipo = dominio.CeldaError
	case "d":
		tipo = dominio.CeldaFecha
	default:
		return dominio.CeldaStaging{}, ErrXLSInvalido
	}
	if len(valor) > maximoBytesCelda {
		return dominio.CeldaStaging{}, ErrLimiteXLSExcedido
	}
	return dominio.CeldaStaging{Tipo: tipo, Valor: valor}, nil
}

// Sólo referencias de la propia hoja y operaciones inertes llegan al dominio.
// Las fórmulas son rechazadas allí como incidencia, nunca evaluadas aquí.
func formulaInternaXLSX(formula string) bool {
	if formula == "" {
		return true // <f t="shared" si="…"/> también es una fórmula inerte.
	}
	for _, r := range formula {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' ||
			strings.ContainsRune("_.$+-*/():,= ", r) {
			continue
		}
		return false
	}
	mayusculas := strings.ToUpper(formula)
	for _, nombre := range []string{"WEBSERVICE", "HYPERLINK", "RTD", "DDE", "FILTERXML", "CALL", "EXEC", "REGISTER.ID", "CUBE"} {
		if strings.Contains(mayusculas, nombre) {
			return false
		}
	}
	return true
}
