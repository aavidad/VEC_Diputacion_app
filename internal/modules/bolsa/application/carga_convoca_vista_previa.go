package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Límites de la carga desde la pantalla. El transporte corta el cuerpo antes
// de leerlo (MaximoBytesCargaConvoca más el margen del formulario) y el caso
// de uso vuelve a comprobar tamaño y número de filas antes de validar.
const (
	MaximoBytesCargaConvoca = 16 * 1024 * 1024
	MaximoFilasCargaConvoca = 20000
)

// Estados de fila, avisos y bloqueos de la vista previa: códigos estables
// que la pantalla traduce con su catálogo; nunca texto visible.
const (
	EstadoFilaCargaAceptada  = "aceptada"
	EstadoFilaCargaRechazada = "rechazada"
	AvisoIdentidadAmbigua    = ports.MotivoRevisionIdentidadAmbigua
	BloqueoEsquemaDetalle    = "esquema_detalle_meritos"
	BloqueoSinFilasAceptadas = "sin_filas_aceptadas"
	// BloqueoIdentidadNoDerivable: alguna fila aceptada no permite derivar la
	// identidad de la persona; la constitución la rechazaría entera.
	BloqueoIdentidadNoDerivable = "identidad_no_derivable"
)

var (
	ErrFicheroCargaConvocaInvalido = errors.New("bolsa: fichero de carga CONVOCA invalido")
	ErrFicheroCargaConvocaExcesivo = errors.New("bolsa: fichero de carga CONVOCA demasiado grande")
	ErrFilasCargaConvocaExcesivas  = errors.New("bolsa: demasiadas filas en la carga CONVOCA")
	ErrDecodificadorCargaRequerido = errors.New("bolsa: decodificador de carga CONVOCA requerido")
)

// DecodificadorCargaConvoca lee el libro (XLS o XLSX) ya acotado en memoria.
type DecodificadorCargaConvoca interface {
	Decodificar(context.Context, io.ReadSeeker) (importacion.HojaStaging, error)
}

// IncidenciaCargaConvoca es un error de una celda: campo estable del acta y
// código del motivo.
type IncidenciaCargaConvoca struct {
	Campo  string
	Codigo string
}

// FilaVistaPreviaCargaConvoca es una fila del fichero tal como quedaría. Las
// rechazadas solo llevan su número y sus errores; las aceptadas, los datos
// que verá la bolsa (documento enmascarado, nombre y puntuación) y la
// posición provisional.
type FilaVistaPreviaCargaConvoca struct {
	Numero          int
	Estado          string
	Posicion        int
	Documento       string
	PrimerApellido  string
	SegundoApellido string
	Nombre          string
	Experiencia     string
	Formacion       string
	Total           string
	Errores         []IncidenciaCargaConvoca
	Avisos          []string
}

// VistaPreviaCargaConvoca no deja rastro: no importa, no custodia ni escribe.
type VistaPreviaCargaConvoca struct {
	HuellaSHA256  string
	NombreFichero string
	Esquema       string
	FilasLeidas   int
	Aceptadas     int
	Rechazadas    int
	ConAvisos     int
	// Bloqueo es el motivo por el que el fichero no puede cargarse aunque se
	// acepten sus errores (vacío si puede cargarse).
	Bloqueo string
	Filas   []FilaVistaPreviaCargaConvoca
}

// PrevisualizadorCargaConvoca valida el fichero con las mismas reglas que el
// importador y ordena las filas como las ordenará la constitución.
type PrevisualizadorCargaConvoca struct {
	decodificador DecodificadorCargaConvoca
}

func NuevoPrevisualizadorCargaConvoca(decodificador DecodificadorCargaConvoca) (*PrevisualizadorCargaConvoca, error) {
	if decodificador == nil {
		return nil, ErrDecodificadorCargaRequerido
	}
	return &PrevisualizadorCargaConvoca{decodificador: decodificador}, nil
}

// NombreFicheroCargaConvocaValido admite solo un nombre simple .xls o .xlsx.
func NombreFicheroCargaConvocaValido(nombre string) bool {
	if nombre == "" || len(nombre) > 255 || !utf8.ValidString(nombre) || strings.TrimSpace(nombre) != nombre ||
		strings.ContainsAny(nombre, `/\`) || filepath.Base(nombre) != nombre {
		return false
	}
	for _, r := range nombre {
		// También los de formato (U+202E y similares), que disfrazan el nombre.
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	extension := strings.ToLower(filepath.Ext(nombre))
	return len(nombre) > len(extension) && (extension == ".xls" || extension == ".xlsx")
}

func (p *PrevisualizadorCargaConvoca) Previsualizar(ctx context.Context, nombre string, contenido []byte) (VistaPreviaCargaConvoca, error) {
	if ctx == nil || p == nil || p.decodificador == nil {
		return VistaPreviaCargaConvoca{}, ErrDecodificadorCargaRequerido
	}
	if err := ctx.Err(); err != nil {
		return VistaPreviaCargaConvoca{}, err
	}
	if len(contenido) > MaximoBytesCargaConvoca {
		return VistaPreviaCargaConvoca{}, ErrFicheroCargaConvocaExcesivo
	}
	if len(contenido) == 0 || !NombreFicheroCargaConvocaValido(nombre) {
		return VistaPreviaCargaConvoca{}, ErrFicheroCargaConvocaInvalido
	}
	hoja, err := p.decodificador.Decodificar(ctx, bytes.NewReader(contenido))
	if err != nil {
		if ctx.Err() != nil {
			return VistaPreviaCargaConvoca{}, ctx.Err()
		}
		return VistaPreviaCargaConvoca{}, errors.Join(ErrFicheroCargaConvocaInvalido, err)
	}
	if len(hoja.Filas) > MaximoFilasCargaConvoca {
		return VistaPreviaCargaConvoca{}, ErrFilasCargaConvocaExcesivas
	}
	staging, err := importacion.ValidarHoja(hoja)
	if err != nil {
		return VistaPreviaCargaConvoca{}, errors.Join(ErrFicheroCargaConvocaInvalido, err)
	}
	suma := sha256.Sum256(contenido)
	vista := VistaPreviaCargaConvoca{
		HuellaSHA256: hex.EncodeToString(suma[:]), NombreFichero: nombre, Esquema: string(hoja.Esquema),
		FilasLeidas: staging.FilasLeidas, Aceptadas: len(staging.Aceptadas), Rechazadas: staging.Rechazadas,
	}
	switch {
	case hoja.Esquema != importacion.EsquemaResumenPersona:
		vista.Bloqueo = BloqueoEsquemaDetalle
	case len(staging.Aceptadas) == 0:
		vista.Bloqueo = BloqueoSinFilasAceptadas
	}
	var derivable bool
	vista.Filas, derivable = filasVistaPrevia(staging)
	if vista.Bloqueo == "" && !derivable {
		vista.Bloqueo = BloqueoIdentidadNoDerivable
	}
	for _, fila := range vista.Filas {
		if len(fila.Avisos) > 0 {
			vista.ConAvisos++
		}
	}
	return vista, nil
}

// filasVistaPrevia devuelve primero las aceptadas en el orden de la bolsa y
// después las rechazadas por número de fila.
func filasVistaPrevia(staging importacion.ResultadoStaging) ([]FilaVistaPreviaCargaConvoca, bool) {
	aceptadas := append([]importacion.FilaAceptada(nil), staging.Aceptadas...)
	constitucion.OrdenarFilasConstitucion(aceptadas)
	ambiguas, derivable := filasIdentidadAmbigua(aceptadas)
	filas := make([]FilaVistaPreviaCargaConvoca, 0, len(aceptadas)+staging.Rechazadas)
	for i, a := range aceptadas {
		fila := FilaVistaPreviaCargaConvoca{
			Numero: a.Numero, Estado: EstadoFilaCargaAceptada, Posicion: i + 1,
			Documento: a.Identidad.Documento, PrimerApellido: a.Identidad.PrimerApellido,
			SegundoApellido: a.Identidad.SegundoApellido, Nombre: a.Identidad.Nombre,
		}
		if a.Resumen != nil {
			fila.Experiencia, fila.Formacion, fila.Total = a.Resumen.Experiencia, a.Resumen.Formacion, a.Resumen.Total
		}
		if ambiguas[a.Numero] {
			fila.Avisos = []string{AvisoIdentidadAmbigua}
		}
		filas = append(filas, fila)
	}
	porFila := map[int]int{}
	for _, inc := range staging.Incidencias {
		indice, existe := porFila[inc.Fila]
		if !existe {
			indice = len(filas)
			porFila[inc.Fila] = indice
			filas = append(filas, FilaVistaPreviaCargaConvoca{Numero: inc.Fila, Estado: EstadoFilaCargaRechazada})
		}
		filas[indice].Errores = append(filas[indice].Errores, IncidenciaCargaConvoca{Campo: inc.Campo, Codigo: inc.Codigo})
	}
	return filas, derivable
}

// filasIdentidadAmbigua marca las filas que la constitución dejará pendientes
// de revisión: comparten documento enmascarado y nombre con otra fila. El
// booleano es falso si alguna fila no permite derivar la identidad.
func filasIdentidadAmbigua(filas []importacion.FilaAceptada) (map[int]bool, bool) {
	porClave := map[string][]int{}
	derivable := true
	for _, fila := range filas {
		clave, err := constitucion.ClaveIdentidadCandidato(fila.Identidad)
		if err != nil {
			derivable = false
			continue
		}
		porClave[clave] = append(porClave[clave], fila.Numero)
	}
	ambiguas := map[int]bool{}
	for _, numeros := range porClave {
		if len(numeros) > 1 {
			for _, n := range numeros {
				ambiguas[n] = true
			}
		}
	}
	return ambiguas, derivable
}
