// Package xlsconvoca adapta libros BIFF8 y OOXML a la zona de ensayo T17.
// No abre rutas ni conserva el fichero: recibe un io.ReadSeeker ya acotado.
package xlsconvoca

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/nkiri/xls"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

const (
	maximoBytesXLS    = 16 * 1024 * 1024
	maximoFilasXLS    = 100_001
	maximoColumnasXLS = 32
	maximoBytesCelda  = 64 * 1024
)

var (
	ErrXLSInvalido       = errors.New("bolsa: exportacion Convoca XLS invalida")
	ErrLimiteXLSExcedido = errors.New("bolsa: limite de exportacion Convoca XLS excedido")
)

var firmaContenedorOLE2 = [8]byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}

// Lector admite un tope de filas propio (cabecera incluida) por debajo del
// máximo absoluto del adaptador; sin tope usa ese máximo.
type Lector struct{ maximoFilas int }

func NuevoLector() *Lector { return &Lector{} }

// NuevoLectorConLimiteFilas corta la lectura en cuanto la hoja declara o
// alcanza más filas que el tope (cabecera incluida), antes de leer sus celdas.
func NuevoLectorConLimiteFilas(filas int) *Lector { return &Lector{maximoFilas: filas} }

func (l *Lector) limiteFilas() int {
	if l == nil || l.maximoFilas <= 0 || l.maximoFilas > maximoFilasXLS {
		return maximoFilasXLS
	}
	return l.maximoFilas
}

func (l *Lector) Decodificar(
	ctx context.Context,
	origen io.ReadSeeker,
) (hoja dominio.HojaStaging, err error) {
	if err := ctx.Err(); err != nil {
		return dominio.HojaStaging{}, err
	}
	if origen == nil {
		return dominio.HojaStaging{}, ErrXLSInvalido
	}
	defer func() {
		if recover() != nil {
			hoja = dominio.HojaStaging{}
			err = ErrXLSInvalido
		}
	}()
	esXLSX, err := validarContenedor(origen)
	if err != nil {
		return dominio.HojaStaging{}, err
	}
	if esXLSX {
		return decodificarXLSX(ctx, origen, l.limiteFilas())
	}
	libro, err := xls.Read(origen)
	if err != nil || libro == nil || libro.SheetCount() != 1 {
		return dominio.HojaStaging{}, ErrXLSInvalido
	}
	hojaXLS := libro.Sheet(0)
	if hojaXLS == nil || hojaXLS.RowCount() < 1 || hojaXLS.RowCount() > l.limiteFilas() {
		if hojaXLS != nil && hojaXLS.RowCount() > l.limiteFilas() {
			return dominio.HojaStaging{}, ErrLimiteXLSExcedido
		}
		return dominio.HojaStaging{}, ErrXLSInvalido
	}
	cabeceras, err := leerCabeceras(hojaXLS.Row(0))
	if err != nil {
		return dominio.HojaStaging{}, err
	}
	esquema, formato, err := dominio.DetectarFormato(cabeceras)
	if err != nil {
		return dominio.HojaStaging{}, err
	}
	// El nombre de hoja se comprueba antes de leer filas: en el formato real
	// de CONVOCA forma parte del esquema cerrado.
	nombreHoja := hojaXLS.Name()
	if len(nombreHoja) > maximoBytesCelda {
		return dominio.HojaStaging{}, ErrXLSInvalido
	}
	if err := dominio.ValidarNombreHoja(esquema, formato, nombreHoja); err != nil {
		return dominio.HojaStaging{}, err
	}
	filas := make([]dominio.FilaStaging, 0, hojaXLS.RowCount()-1)
	for numero := 1; numero < hojaXLS.RowCount(); numero++ {
		if numero%256 == 0 {
			if err := ctx.Err(); err != nil {
				return dominio.HojaStaging{}, err
			}
		}
		fila, err := leerFila(hojaXLS.Row(numero), numero+1)
		if err != nil {
			return dominio.HojaStaging{}, err
		}
		filas = append(filas, fila)
	}
	return dominio.HojaStaging{
		Esquema: esquema, Cabeceras: cabeceras, NombreHoja: nombreHoja, Filas: filas,
	}, nil
}

func validarContenedor(origen io.ReadSeeker) (bool, error) {
	tamano, err := origen.Seek(0, io.SeekEnd)
	if err != nil || tamano < 4 {
		return false, ErrXLSInvalido
	}
	if tamano > maximoBytesXLS {
		return false, ErrLimiteXLSExcedido
	}
	if _, err := origen.Seek(0, io.SeekStart); err != nil {
		return false, ErrXLSInvalido
	}
	var firma [8]byte
	n, err := io.ReadFull(origen, firma[:])
	if err != nil && err != io.ErrUnexpectedEOF {
		return false, ErrXLSInvalido
	}
	if _, err := origen.Seek(0, io.SeekStart); err != nil {
		return false, ErrXLSInvalido
	}
	if n == len(firma) && firma == firmaContenedorOLE2 {
		return false, nil
	}
	if string(firma[:4]) == "PK\x03\x04" {
		return true, nil
	}
	return false, ErrXLSInvalido
}

func leerCabeceras(fila *xls.Row) ([]string, error) {
	if fila == nil || fila.CellCount() < 1 || fila.CellCount() > maximoColumnasXLS {
		return nil, ErrXLSInvalido
	}
	cabeceras := make([]string, fila.CellCount())
	for columna := 0; columna < fila.CellCount(); columna++ {
		celda := fila.Cell(columna)
		if celda == nil || celda.Type != xls.CellTypeString ||
			len(celda.Value()) > maximoBytesCelda {
			return nil, ErrXLSInvalido
		}
		cabeceras[columna] = celda.Value()
	}
	return cabeceras, nil
}

func leerFila(fila *xls.Row, numero int) (dominio.FilaStaging, error) {
	if fila == nil {
		return dominio.FilaStaging{Numero: numero}, nil
	}
	if fila.CellCount() > maximoColumnasXLS {
		return dominio.FilaStaging{}, ErrLimiteXLSExcedido
	}
	celdas := make([]dominio.CeldaStaging, fila.CellCount())
	for columna := 0; columna < fila.CellCount(); columna++ {
		celda := fila.Cell(columna)
		if celda == nil {
			celdas[columna] = dominio.CeldaStaging{Tipo: dominio.CeldaVacia}
			continue
		}
		if len(celda.Value()) > maximoBytesCelda {
			return dominio.FilaStaging{}, ErrLimiteXLSExcedido
		}
		tipo, err := convertirTipo(celda.Type)
		if err != nil {
			return dominio.FilaStaging{}, err
		}
		celdas[columna] = dominio.CeldaStaging{Tipo: tipo, Valor: celda.Value()}
	}
	return dominio.FilaStaging{Numero: numero, Celdas: celdas}, nil
}

func convertirTipo(tipo xls.CellType) (dominio.TipoCelda, error) {
	switch tipo {
	case xls.CellTypeEmpty:
		return dominio.CeldaVacia, nil
	case xls.CellTypeString:
		return dominio.CeldaTexto, nil
	case xls.CellTypeNumber:
		return dominio.CeldaNumero, nil
	case xls.CellTypeFormula:
		return dominio.CeldaFormula, nil
	case xls.CellTypeError:
		return dominio.CeldaError, nil
	case xls.CellTypeBool:
		return dominio.CeldaLogica, nil
	case xls.CellTypeDate:
		return dominio.CeldaFecha, nil
	default:
		return "", fmt.Errorf("%w: tipo de celda desconocido", ErrXLSInvalido)
	}
}
