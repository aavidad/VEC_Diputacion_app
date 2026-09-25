package informejuridico

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// fuenteCampos reúne sólo lo que ya contiene el detalle autorizado. No hay
// identidad de la persona: ningún campo la aporta y las plantillas dejan su
// hueco para completarlo en el circuito competente.
type fuenteCampos struct {
	detalle    ports.DetalleExpedienteRRHH
	propuesta  ports.HitoExpedienteRRHH
	requerido  *ports.HitoExpedienteRRHH
	etiquetar  EtiquetadorReferencias
	plantillas *PlantillasBorrador
	plantilla  PlantillaBorrador
}

// camposBorrador es la lista cerrada de campos combinables. Añadir un campo
// exige código porque exige un dato nuevo del expediente; cambiar un texto,
// no.
var camposBorrador = map[string]func(f *fuenteCampos) string{
	"numero_expediente":  func(f *fuenteCampos) string { return f.detalle.Resumen.NumeroVisible },
	"expediente_ref":     func(f *fuenteCampos) string { return f.detalle.Resumen.ExpedienteRef },
	"version_expediente": func(f *fuenteCampos) string { return strconv.FormatUint(f.detalle.Resumen.Version, 10) },
	"centro":             func(f *fuenteCampos) string { return referenciaConNombre(f.etiquetar, f.detalle.Resumen.CentroRef) },
	"categoria":          func(f *fuenteCampos) string { return referenciaConNombre(f.etiquetar, f.detalle.Resumen.CategoriaRef) },
	"grupo_subgrupo":     func(f *fuenteCampos) string { return f.detalle.Solicitud.GrupoSubgrupo },
	"modalidad": func(f *fuenteCampos) string {
		return f.plantillas.etiqueta("modalidad.", string(f.detalle.Analisis.ModalidadClave))
	},
	"causa": func(f *fuenteCampos) string {
		return f.plantillas.etiqueta("causa.", string(f.detalle.Analisis.CausaClave))
	},
	"periodo_inicio": func(f *fuenteCampos) string { return f.detalle.Analisis.PeriodoInicio.Format("02/01/2006") },
	"periodo_fin":    func(f *fuenteCampos) string { return f.detalle.Analisis.PeriodoFin.Format("02/01/2006") },
	"jornada": func(f *fuenteCampos) string {
		j := f.detalle.Analisis.PorcentajeJornada
		return fmt.Sprintf("%d,%02d %%", j/100, j%100)
	},
	"resultado_rc": func(f *fuenteCampos) string { return string(f.detalle.Analisis.ResultadoRC) },
	"coste": func(f *fuenteCampos) string {
		c := f.detalle.Analisis.CostePrevisto
		if c == nil {
			return ""
		}
		return fmt.Sprintf("%d,%02d %s", c.Centimos/100, c.Centimos%100, c.Moneda)
	},
	"fuente_coste":         func(f *fuenteCampos) string { return f.detalle.Analisis.FuenteCosteRef },
	"observaciones":        func(f *fuenteCampos) string { return f.detalle.Analisis.Observaciones },
	"via_cobertura":        func(f *fuenteCampos) string { return string(f.detalle.Cobertura.ViaClave) },
	"unidad":               func(f *fuenteCampos) string { return referenciaConNombre(f.etiquetar, f.detalle.Asignacion.UnidadRef) },
	"asignada_en":          func(f *fuenteCampos) string { return instanteDocumento(f.detalle.Asignacion.AsignadaEn) },
	"comprobaciones_bolsa": comprobacionesBolsa,
	"propuesta_actuacion":  func(f *fuenteCampos) string { return strconv.FormatUint(f.propuesta.Secuencia, 10) },
	"propuesta_fecha":      func(f *fuenteCampos) string { return instanteDocumento(f.propuesta.RealizadaEn) },
	"actuacion_requerida": func(f *fuenteCampos) string {
		if f.requerido == nil {
			return ""
		}
		return strconv.FormatUint(f.requerido.Secuencia, 10)
	},
	"actuacion_requerida_fecha": func(f *fuenteCampos) string {
		if f.requerido == nil {
			return ""
		}
		return instanteDocumento(f.requerido.RealizadaEn)
	},
	"firmantes": func(f *fuenteCampos) string {
		lineas := make([]string, 0, len(f.plantilla.Firmantes))
		for _, firmante := range f.plantilla.Firmantes {
			lineas = append(lineas, "- "+firmante)
		}
		return strings.Join(lineas, "\n")
	},
	"plantilla_ref": func(f *fuenteCampos) string { return f.plantilla.Referencia },
}

func instanteDocumento(instante time.Time) string {
	return instante.UTC().Format(time.RFC3339Nano)
}

// comprobacionesBolsa devuelve una línea por comprobación, o vacío si no
// consta ninguna; la plantilla decide el encabezado y el texto alternativo.
func comprobacionesBolsa(f *fuenteCampos) string {
	comprobaciones := f.detalle.Cobertura.Comprobaciones
	lineas := make([]string, 0, len(comprobaciones))
	for _, comprobacion := range comprobaciones {
		lineas = append(lineas, "- "+f.plantillas.etiqueta("comprobacion.", string(comprobacion.Clave))+": "+
			f.plantillas.etiqueta("resultado.", string(comprobacion.Resultado)))
	}
	return strings.Join(lineas, "\n")
}

func (f *fuenteCampos) valores() map[string]string {
	valores := make(map[string]string, len(camposBorrador))
	for campo, obtener := range camposBorrador {
		valores[campo] = obtener(f)
	}
	return valores
}
