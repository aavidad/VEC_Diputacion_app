package importacionconvoca

import "time"

// EsquemaReciboImportacion identifica la salida minimizada para el consumidor
// del importador. El acta durable conserva el detalle de las incidencias.
const EsquemaReciboImportacion = "vec.bolsa.importacion-convoca.recibo.v1"

type EstadoReciboImportacion string

const (
	EstadoReciboNueva       EstadoReciboImportacion = "nueva"
	EstadoReciboReutilizada EstadoReciboImportacion = "reutilizada"
)

// ReciboImportacion permite comunicar un alta o replay sin exponer filas,
// identidad enmascarada, nombre del fichero, ruta de custodia ni actor.
// La procedencia deja explícito que el lote aún requiere conciliación.
type ReciboImportacion struct {
	Esquema                      string                  `json:"esquema"`
	Estado                       EstadoReciboImportacion `json:"estado"`
	ActaRef                      string                  `json:"acta_ref"`
	ImportacionRef               string                  `json:"importacion_ref"`
	BolsaRef                     string                  `json:"bolsa_ref"`
	CategoriaRef                 string                  `json:"categoria_ref"`
	HuellaFicheroSHA256          string                  `json:"huella_fichero_sha256"`
	RegistradaEn                 time.Time               `json:"registrada_en"`
	EsquemaExportacion           string                  `json:"esquema_exportacion"`
	FilasLeidas                  int                     `json:"filas_leidas"`
	FilasAceptadas               int                     `json:"filas_aceptadas"`
	FilasRechazadas              int                     `json:"filas_rechazadas"`
	Incidencias                  int                     `json:"incidencias"`
	Autoridad                    string                  `json:"autoridad"`
	HabilitaActosConEfectos      bool                    `json:"habilita_actos_con_efectos"`
	RequiereConfirmacionRegistro bool                    `json:"requiere_confirmacion_registro"`
}

// Recibo valida la respuesta del repositorio antes de mostrarla y conserva el
// instante del acta original en los reintentos.
func (r ResultadoImportacion) Recibo() (ReciboImportacion, error) {
	a := r.Acta
	if a.Validar() != nil {
		return ReciboImportacion{}, ErrResultadoInseguro
	}
	estado := EstadoReciboNueva
	if r.Reutilizada {
		estado = EstadoReciboReutilizada
	}
	return ReciboImportacion{
		Esquema:                      EsquemaReciboImportacion,
		Estado:                       estado,
		ActaRef:                      a.ActaRef,
		ImportacionRef:               a.ImportacionRef,
		BolsaRef:                     a.BolsaRef,
		CategoriaRef:                 a.CategoriaRef,
		HuellaFicheroSHA256:          a.HuellaFicheroSHA256,
		RegistradaEn:                 a.RegistradaEn,
		EsquemaExportacion:           string(a.Esquema),
		FilasLeidas:                  a.FilasLeidas,
		FilasAceptadas:               a.FilasAceptadas,
		FilasRechazadas:              a.FilasRechazadas,
		Incidencias:                  len(a.Incidencias),
		Autoridad:                    a.Procedencia.Autoridad,
		HabilitaActosConEfectos:      a.Procedencia.HabilitaActosConEfectos,
		RequiereConfirmacionRegistro: a.Procedencia.RequiereConfirmacionRegistro,
	}, nil
}
