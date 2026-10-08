package observabilidad

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/catalogoincidencias"
	"vec-diputacion-granada/internal/vec/domain"
)

// ConfiguracionRecolector es local y explícita. La retención solo afecta al
// directorio privado de incidencias técnicas; nunca recibe auditoría funcional.
type ConfiguracionRecolector struct {
	// CatalogoIncidencias selecciona un catálogo local; vacío usa el índice
	// común de idiomas del binario. Catalogo permite inyección ya validada.
	CatalogoIncidencias string                                    `json:"catalogo_incidencias,omitempty"`
	Catalogo            *catalogoincidencias.Catalogo             `json:"-"`
	Directorio          string                                    `json:"directorio"`
	MaxLineaBytes       int                                       `json:"max_linea_bytes"`
	MaxArchivoBytes     int64                                     `json:"max_archivo_bytes"`
	MaxArchivos         int                                       `json:"max_archivos"`
	RetencionSegundos   int64                                     `json:"retencion_segundos"`
	VentanaSegundos     int64                                     `json:"ventana_alertas_segundos"`
	Umbrales            map[domain.CodigoIncidenciaTecnica]uint64 `json:"umbrales_alerta"`
	UmbralesResultado   map[domain.CodigoResultadoTecnico]uint64  `json:"umbrales_resultado,omitempty"`
}

// MetricasRecolector no contiene etiquetas ni entradas aportadas por personas.
type MetricasRecolector struct {
	Recibidas    uint64                                    `json:"recibidas"`
	Escritas     uint64                                    `json:"escritas"`
	Rechazadas   uint64                                    `json:"rechazadas"`
	Alertas      uint64                                    `json:"alertas"`
	Retirados    uint64                                    `json:"archivos_retirados"`
	PorCodigo    map[domain.CodigoIncidenciaTecnica]uint64 `json:"por_codigo"`
	PorResultado map[domain.CodigoResultadoTecnico]uint64  `json:"por_resultado"`
}

// RecolectarIncidencias consume hasta EOF. Las entradas no conformes y largas
// se cuentan y se descartan; su contenido nunca llega al almacenamiento ni al
// diagnóstico. Los fallos del destino detienen la recogida y se comunican como
// os.ErrInvalid, sin ruta ni error libre. Cada ejecución requiere un solo lector.
func RecolectarIncidencias(entrada io.Reader, alertas io.Writer, cfg ConfiguracionRecolector) (metricas MetricasRecolector, err error) {
	return recolectarIncidencias(entrada, alertas, cfg, time.Now)
}

func recolectarIncidencias(entrada io.Reader, alertas io.Writer, cfg ConfiguracionRecolector, reloj func() time.Time) (metricas MetricasRecolector, err error) {
	metricas.PorCodigo = make(map[domain.CodigoIncidenciaTecnica]uint64)
	metricas.PorResultado = make(map[domain.CodigoResultadoTecnico]uint64)
	if entrada == nil || alertas == nil || reloj == nil || !configuracionRecolectorValida(cfg) {
		return metricas, os.ErrInvalid
	}
	if cfg.Catalogo != nil && cfg.CatalogoIncidencias != "" {
		return metricas, os.ErrInvalid
	}
	if cfg.Catalogo == nil {
		if cfg.CatalogoIncidencias != "" {
			cfg.Catalogo, err = catalogoincidencias.DesdeArchivo(cfg.CatalogoIncidencias)
		} else {
			cfg.Catalogo, err = catalogoincidencias.Predeterminado()
		}
		if err != nil {
			return metricas, os.ErrInvalid
		}
	}
	if !cfg.Catalogo.Valido() {
		return metricas, os.ErrInvalid
	}
	almacen, err := abrirAlmacenRecolector(cfg, reloj)
	if err != nil {
		return metricas, os.ErrInvalid
	}
	defer func() {
		metricas.Retirados = almacen.retirados
		if almacen.cerrar() != nil {
			err = os.ErrInvalid
		}
	}()
	contador := contadorAlertas{cfg: cfg, inicio: reloj(), cantidades: make(map[domain.CodigoIncidenciaTecnica]uint64), avisados: make(map[domain.CodigoIncidenciaTecnica]bool)}
	lector := bufio.NewReaderSize(entrada, cfg.MaxLineaBytes+1)
	for {
		linea, larga, fin := leerLineaRecolector(lector, cfg.MaxLineaBytes)
		if fin != nil && fin != io.EOF {
			return metricas, os.ErrInvalid
		}
		if len(linea) == 0 && !larga && fin == io.EOF {
			return metricas, nil
		}
		metricas.Recibidas++
		var incidencia lineaIncidencia
		var resultado lineaResultado
		var datos []byte
		var errorLinea error
		var esquema string
		if !larga {
			esquema, errorLinea = esquemaLineaRecolector(linea)
			if errorLinea == nil {
				switch esquema {
				case domain.EsquemaIncidenciaTecnica:
					incidencia, errorLinea = validarLineaRecolectorConCatalogo(linea, cfg.Catalogo)
					if errorLinea == nil {
						datos, errorLinea = json.Marshal(incidencia)
					}
				case domain.EsquemaResultadoTecnico:
					resultado, errorLinea = validarLineaResultadoRecolector(linea)
					if errorLinea == nil {
						datos, errorLinea = json.Marshal(resultado)
					}
				default:
					errorLinea = os.ErrInvalid
				}
			}
		}
		clear(linea)
		if larga || errorLinea != nil {
			metricas.Rechazadas++
		} else {
			if almacen.escribir(append(datos, '\n')) != nil {
				return metricas, os.ErrInvalid
			}
			metricas.Escritas++
			if esquema == domain.EsquemaIncidenciaTecnica {
				codigo := domain.CodigoIncidenciaTecnica(incidencia.Codigo)
				metricas.PorCodigo[codigo] = sumarSaturado(metricas.PorCodigo[codigo], uint64(incidencia.Recuento))
				if contador.registrar(incidencia, reloj(), alertas, &metricas) != nil {
					return metricas, os.ErrInvalid
				}
			} else {
				codigo := domain.CodigoResultadoTecnico(resultado.Resultado)
				metricas.PorResultado[codigo] = sumarSaturado(metricas.PorResultado[codigo], 1)
				if contador.registrarResultado(resultado, reloj(), alertas, &metricas) != nil {
					return metricas, os.ErrInvalid
				}
			}
		}
		if fin == io.EOF {
			return metricas, nil
		}
	}
}

func esquemaLineaRecolector(datos []byte) (string, error) {
	var cabecera struct {
		Esquema string `json:"esquema"`
	}
	if json.Unmarshal(datos, &cabecera) != nil || cabecera.Esquema == "" {
		return "", os.ErrInvalid
	}
	return cabecera.Esquema, nil
}

// Se vacía una línea larga por fragmentos acotados y se continúa con la siguiente.
func leerLineaRecolector(r *bufio.Reader, limite int) ([]byte, bool, error) {
	parte, err := r.ReadSlice('\n')
	larga := len(bytes.TrimSuffix(parte, []byte{'\n'})) > limite || err == bufio.ErrBufferFull
	if larga {
		for err == bufio.ErrBufferFull {
			clear(parte)
			parte, err = r.ReadSlice('\n')
		}
		clear(parte)
		return nil, true, err
	}
	return bytes.TrimSuffix(parte, []byte{'\n'}), false, err
}

func validarLineaRecolector(datos []byte) (lineaIncidencia, error) {
	catalogo, err := catalogoincidencias.Predeterminado()
	if err != nil {
		return lineaIncidencia{}, os.ErrInvalid
	}
	return validarLineaRecolectorConCatalogo(datos, catalogo)
}

func validarLineaRecolectorConCatalogo(datos []byte, catalogo *catalogoincidencias.Catalogo) (lineaIncidencia, error) {
	var l lineaIncidencia
	if clavesUnicasRecolector(datos) != nil {
		return l, os.ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if d.Decode(&l) != nil || d.Decode(new(any)) != io.EOF {
		return l, os.ErrInvalid
	}
	instante, err := time.Parse(formatoInstante, l.Instante)
	clasificacion, saneada := domain.ClasificarIncidenciaTecnica(domain.SolicitudIncidenciaTecnica{
		Codigo: domain.CodigoIncidenciaTecnica(l.Codigo), Componente: domain.ComponenteIncidenciaTecnica(l.Componente), Etapa: domain.EtapaIncidenciaTecnica(l.Etapa), Recuento: l.Recuento,
	})
	plantilla, ok := catalogo.Plantilla(clasificacion.Codigo)
	if !ok || err != nil || instante.IsZero() || saneada || l.Esquema != domain.EsquemaIncidenciaTecnica || l.Recuento != clasificacion.Recuento ||
		l.Severidad != string(clasificacion.Severidad) || l.Mensaje != plantilla ||
		l.Entorno != string(domain.NormalizarEntornoIncidenciaTecnica(l.Entorno)) ||
		l.VersionBinario != domain.NormalizarVersionBinario(l.VersionBinario) || !domain.EsCorrelacionTecnicaValida(l.Correlacion) {
		return lineaIncidencia{}, os.ErrInvalid
	}
	// Se conserva una proyección reconstruida; nunca el JSON original.
	inc := domain.NuevaIncidenciaTecnica(clasificacion, instante, domain.EntornoIncidenciaTecnica(l.Entorno), l.VersionBinario, l.Correlacion)
	return lineaIncidencia{Esquema: inc.Esquema, Instante: inc.Instante.Format(formatoInstante), Codigo: string(inc.Codigo), Severidad: string(inc.Severidad), Componente: string(inc.Componente), Etapa: string(inc.Etapa), Entorno: string(inc.Entorno), VersionBinario: inc.VersionBinario, Correlacion: inc.Correlacion, Recuento: inc.Recuento, Mensaje: plantilla}, nil
}

func clavesUnicasRecolector(datos []byte) error {
	d := json.NewDecoder(bytes.NewReader(datos))
	inicio, err := d.Token()
	if err != nil || inicio != json.Delim('{') {
		return os.ErrInvalid
	}
	vistas := make(map[string]bool)
	for d.More() {
		token, err := d.Token()
		clave, ok := token.(string)
		if err != nil || !ok || vistas[clave] || !claveIncidenciaRecolectorValida(clave) {
			return os.ErrInvalid
		}
		vistas[clave] = true
		var valor json.RawMessage
		if d.Decode(&valor) != nil {
			return os.ErrInvalid
		}
	}
	fin, err := d.Token()
	if err != nil || fin != json.Delim('}') || d.Decode(new(any)) != io.EOF {
		return os.ErrInvalid
	}
	return nil
}

func claveIncidenciaRecolectorValida(clave string) bool {
	switch clave {
	case "esquema", "instante", "codigo", "severidad", "componente", "etapa", "entorno", "version_binario", "correlacion", "recuento", "mensaje", "resultado", "nivel", "correlacion_ref":
		return true
	default:
		return false
	}
}

func configuracionRecolectorValida(c ConfiguracionRecolector) bool {
	// Estos topes acotan memoria y operaciones del proceso; no son plazos de
	// conservación ni valores predeterminados de la política de Sistemas.
	if c.Directorio == "" || c.MaxLineaBytes < 512 || c.MaxLineaBytes > 1<<20 || c.MaxArchivoBytes < int64(c.MaxLineaBytes)+1 || c.MaxArchivos < 2 || c.MaxArchivos > 1024 || c.RetencionSegundos <= 0 || c.RetencionSegundos > int64((time.Duration(1<<63-1))/time.Second) || c.VentanaSegundos <= 0 || c.VentanaSegundos > c.RetencionSegundos || len(c.Umbrales) == 0 {
		return false
	}
	for codigo, umbral := range c.Umbrales {
		if _, ok := domain.DefinicionIncidenciaTecnicaDe(codigo); !ok || umbral == 0 {
			return false
		}
	}
	for resultado, umbral := range c.UmbralesResultado {
		if (resultado != domain.ResultadoTecnicoDenegado && resultado != domain.ResultadoTecnicoNoDisponible) || umbral == 0 {
			return false
		}
	}
	return true
}

func sumarSaturado(a, b uint64) uint64 {
	if ^uint64(0)-a < b {
		return ^uint64(0)
	}
	return a + b
}
