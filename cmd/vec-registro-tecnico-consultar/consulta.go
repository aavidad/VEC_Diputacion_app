package main

import (
	"bufio"
	"io"
	"math"
	"os"
	"strings"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/catalogoincidencias"
	"vec-diputacion-granada/internal/vec/domain"
)

const (
	maxArchivosConsulta = 8
	maxArchivoBytes     = 16 << 20
	maxLineaBytes       = 16 << 10
	maximoSegundos      = 3600
)

type tiemposConsulta struct {
	TotalSegundos  float64 `json:"total_segundos"`
	MediaSegundos  float64 `json:"media_segundos"`
	MaximoSegundos float64 `json:"maximo_segundos"`
}

type peticionesConsulta struct {
	Total              uint64          `json:"total"`
	Errores5xx         uint64          `json:"errores_5xx"`
	Lentas             uint64          `json:"lentas"`
	ConsultasBD        int64           `json:"consultas_bd"`
	Duracion           tiemposConsulta `json:"duracion"`
	DuracionBD         tiemposConsulta `json:"duracion_bd"`
	EsperaConexionPool tiemposConsulta `json:"espera_conexion_pool"`
}

type arranqueConsulta struct {
	Preparadas uint64          `json:"preparadas"`
	Fallidas   uint64          `json:"fallidas"`
	Duracion   tiemposConsulta `json:"duracion"`
}

type resumenConsulta struct {
	Estado        string             `json:"estado"`
	Mensaje       string             `json:"mensaje"`
	Desde         string             `json:"desde"`
	Hasta         string             `json:"hasta"`
	Archivos      int                `json:"archivos"`
	BytesPrefijo  int64              `json:"bytes_prefijo"`
	Recibidas     uint64             `json:"recibidas"`
	Validas       uint64             `json:"validas"`
	Seleccionadas uint64             `json:"seleccionadas"`
	Filtradas     uint64             `json:"filtradas"`
	Rechazadas    uint64             `json:"rechazadas"`
	Incidencias   map[string]uint64  `json:"incidencias_por_codigo"`
	Resultados    map[string]uint64  `json:"resultados_por_tipo"`
	Errores       map[string]uint64  `json:"errores_por_clase"`
	Peticiones    peticionesConsulta `json:"peticiones"`
	Arranque      arranqueConsulta   `json:"arranque"`
}

type registroConsulta struct {
	familia    string
	instante   time.Time
	codigo     domain.CodigoIncidenciaTecnica
	resultado  domain.CodigoResultadoTecnico
	recuento   uint32
	ruta       string
	estado     int
	duracion   float64
	duracionBD float64
	esperaPool float64
	consultas  int64
	lenta      bool
	fallida    bool
	claseError string
}

func consultarArchivos(op opcionesConsulta, catalogo *catalogoincidencias.Catalogo) (resumenConsulta, error) {
	r := resumenConsulta{
		Desde: op.Desde.Format(time.RFC3339Nano), Hasta: op.Hasta.Format(time.RFC3339Nano),
		Archivos: len(op.Archivos), Incidencias: map[string]uint64{}, Resultados: map[string]uint64{},
		Errores: map[string]uint64{},
	}
	vistos := make(map[[2]uint64]bool, len(op.Archivos))
	for _, ruta := range op.Archivos {
		if err := consultarArchivo(ruta, op, catalogo, &r, vistos); err != nil {
			return resumenConsulta{}, os.ErrInvalid
		}
	}
	if r.Peticiones.Total > 0 {
		r.Peticiones.Duracion.MediaSegundos = redondear(r.Peticiones.Duracion.TotalSegundos / float64(r.Peticiones.Total))
		r.Peticiones.DuracionBD.MediaSegundos = redondear(r.Peticiones.DuracionBD.TotalSegundos / float64(r.Peticiones.Total))
		r.Peticiones.EsperaConexionPool.MediaSegundos = redondear(r.Peticiones.EsperaConexionPool.TotalSegundos / float64(r.Peticiones.Total))
	}
	if total := r.Arranque.Preparadas + r.Arranque.Fallidas; total > 0 {
		r.Arranque.Duracion.MediaSegundos = redondear(r.Arranque.Duracion.TotalSegundos / float64(total))
	}
	return r, nil
}

func consultarArchivo(ruta string, op opcionesConsulta, catalogo *catalogoincidencias.Catalogo, r *resumenConsulta, vistos map[[2]uint64]bool) (resultado error) {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0) // #nosec G304 -- ruta local explícita; sin enlace de hoja, fichero regular y lectura acotada.
	if err != nil {
		return os.ErrInvalid
	}
	defer func() {
		if f.Close() != nil {
			resultado = os.ErrInvalid
		}
	}()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maxArchivoBytes {
		return os.ErrInvalid
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return os.ErrInvalid
	}
	identidad := [2]uint64{uint64(stat.Dev), stat.Ino}
	if vistos[identidad] {
		return os.ErrInvalid // Evita contar dos veces un mismo archivo enlazado.
	}
	vistos[identidad] = true
	return consultarPrefijo(f, info.Size(), op, catalogo, r)
}

func consultarPrefijo(f io.ReaderAt, tamanoInicial int64, op opcionesConsulta, catalogo *catalogoincidencias.Catalogo, r *resumenConsulta) error {
	// La consulta fija el prefijo existente al abrir. Un append posterior
	// pertenece a la siguiente consulta y no invalida este resultado.
	seccion := io.NewSectionReader(f, 0, tamanoInicial)
	lector := bufio.NewReaderSize(seccion, maxLineaBytes+2)
	for {
		linea, err := lector.ReadSlice('\n')
		if err == bufio.ErrBufferFull {
			r.Recibidas++
			r.Rechazadas++
			for err == bufio.ErrBufferFull {
				clear(linea)
				linea, err = lector.ReadSlice('\n')
			}
			clear(linea)
			if err != nil && err != io.EOF {
				return os.ErrInvalid
			}
			if err == io.EOF {
				break
			}
			continue
		}
		if err == io.EOF && len(linea) == 0 {
			break
		}
		r.Recibidas++
		if err != nil || len(linea) == 0 || len(linea)-1 > maxLineaBytes {
			r.Rechazadas++
			clear(linea)
			if err == io.EOF {
				break
			}
			if err != nil {
				return os.ErrInvalid
			}
			continue
		}
		registro, err := validarRegistro(linea[:len(linea)-1], catalogo)
		clear(linea)
		if err != nil {
			r.Rechazadas++
			continue
		}
		r.Validas++
		if registro.instante.Before(op.Desde) || !registro.instante.Before(op.Hasta) ||
			op.Codigo != "" && (registro.familia != "incidencia" || registro.codigo != op.Codigo) ||
			op.Ruta != "" && (registro.familia != "peticion" || registro.ruta != op.Ruta) {
			r.Filtradas++
			continue
		}
		r.Seleccionadas++
		r.sumar(registro)
	}
	leidos, err := seccion.Seek(0, io.SeekCurrent)
	if err != nil || leidos != tamanoInicial {
		return os.ErrInvalid // El prefijo se truncó durante la lectura.
	}
	r.BytesPrefijo += leidos
	return nil
}

func (r *resumenConsulta) sumar(e registroConsulta) {
	switch e.familia {
	case "incidencia":
		r.Incidencias[string(e.codigo)] += uint64(e.recuento)
	case "resultado":
		r.Resultados[string(e.resultado)]++
	case "peticion":
		r.Peticiones.Total++
		if e.estado >= 500 {
			r.Peticiones.Errores5xx++
		}
		if e.lenta {
			r.Peticiones.Lentas++
		}
		r.Peticiones.ConsultasBD += e.consultas
		r.Peticiones.Duracion.sumar(e.duracion)
		r.Peticiones.DuracionBD.sumar(e.duracionBD)
		r.Peticiones.EsperaConexionPool.sumar(e.esperaPool)
	case "arranque":
		if e.fallida {
			r.Arranque.Fallidas++
		} else {
			r.Arranque.Preparadas++
		}
		r.Arranque.Duracion.sumar(e.duracion)
	}
	if e.claseError != "" {
		r.Errores[e.claseError]++
	}
}

func (t *tiemposConsulta) sumar(segundos float64) {
	t.TotalSegundos = redondear(t.TotalSegundos + segundos)
	if segundos > t.MaximoSegundos {
		t.MaximoSegundos = segundos
	}
}

func redondear(valor float64) float64 { return math.Round(valor*1e4) / 1e4 }

func rutaFija(ruta string) bool {
	if len(ruta) > 256 || !strings.HasPrefix(ruta, "/") || strings.ContainsAny(ruta, "?#\\\r\n") {
		return false
	}
	if ruta == "/" {
		return true
	}
	segmentos := strings.Split(strings.TrimPrefix(ruta, "/"), "/")
	for i, segmento := range segmentos {
		if segmento == "" && i == len(segmentos)-1 {
			continue // La raíz y algunas rutas HTML terminan en barra.
		}
		if segmento == "{valor}" || segmento == "{mas}" {
			continue
		}
		if len(segmento) == 0 || len(segmento) > 40 {
			return false
		}
		if len(segmento) >= 2 && len(segmento) <= 3 && segmento[0] == 'v' && strings.Trim(segmento[1:], "0123456789") == "" {
			continue
		}
		if strings.Trim(segmento, "abcdefghijklmnopqrstuvwxyz-_") != "" {
			return false
		}
	}
	return true
}
