package ginpixfichero

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var ErrPreparacionExportacionInvalida = errors.New(
	"contratacion temporal: preparacion de exportacion ginpix invalida",
)

// MetadatosPreparacionExportacion conserva solo las ligaduras opacas y
// versiones necesarias para entregar el contenido a capacidades comunes.
type MetadatosPreparacionExportacion struct {
	VersionExpediente    uint64
	ExpedienteRef        string
	IncorporacionRef     string
	ProcedenciaModeloRef string
	CorrelacionRef       string
	IdempotenciaRef      string
	MapeoRef             string
	MapeoVersion         uint64
	ProcedenciaMapeoRef  string
}

type datosPreparacionExportacion struct {
	contenido       []byte
	huellaSHA256    string
	metadatosOpacos MetadatosPreparacionExportacion
}

// PreparacionExportacion es un sobre inmutable de contenido. No escribe,
// firma, entrega ni confirma el fichero.
type PreparacionExportacion struct {
	datos *datosPreparacionExportacion
}

// PrepararExportacion fija los bytes exactos producidos por Codificar y su
// huella antes de que una capacidad común pueda consumirlos.
func PrepararExportacion(carga domain.CargaMapeadaGINPIX) (PreparacionExportacion, error) {
	if carga.Validar() != nil {
		return PreparacionExportacion{}, ErrPreparacionExportacionInvalida
	}
	contenido, err := Codificar(carga)
	if err != nil {
		return PreparacionExportacion{}, ErrPreparacionExportacionInvalida
	}
	if len(contenido) == 0 || len(contenido) > MaximoBytesFicheroGINPIX {
		return PreparacionExportacion{}, ErrPreparacionExportacionInvalida
	}
	datosCarga := carga.Datos()
	return prepararContenidoExportacion(contenido, MetadatosPreparacionExportacion{
		VersionExpediente:    datosCarga.VersionExpediente,
		ExpedienteRef:        datosCarga.ExpedienteRef,
		IncorporacionRef:     datosCarga.IncorporacionRef,
		ProcedenciaModeloRef: datosCarga.ProcedenciaModeloRef,
		CorrelacionRef:       datosCarga.CorrelacionRef,
		IdempotenciaRef:      datosCarga.IdempotenciaRef,
		MapeoRef:             datosCarga.MapeoRef,
		MapeoVersion:         datosCarga.MapeoVersion,
		ProcedenciaMapeoRef:  datosCarga.ProcedenciaMapeoRef,
	})
}

func prepararContenidoExportacion(
	contenido []byte,
	metadatos MetadatosPreparacionExportacion,
) (PreparacionExportacion, error) {
	// La cota se comprueba antes de calcular la huella o crear la copia que
	// pasa a ser propiedad privada de la preparación.
	if len(contenido) == 0 || len(contenido) > MaximoBytesFicheroGINPIX ||
		!metadatos.validar() {
		return PreparacionExportacion{}, ErrPreparacionExportacionInvalida
	}
	suma := sha256.Sum256(contenido)
	preparacion := PreparacionExportacion{datos: &datosPreparacionExportacion{
		contenido:       append([]byte(nil), contenido...),
		huellaSHA256:    hex.EncodeToString(suma[:]),
		metadatosOpacos: metadatos,
	}}
	if preparacion.Validar() != nil {
		return PreparacionExportacion{}, ErrPreparacionExportacionInvalida
	}
	return preparacion, nil
}

func (p PreparacionExportacion) Validar() error {
	if p.datos == nil || len(p.datos.contenido) == 0 ||
		len(p.datos.contenido) > MaximoBytesFicheroGINPIX ||
		!p.datos.metadatosOpacos.validar() {
		return ErrPreparacionExportacionInvalida
	}
	suma := sha256.Sum256(p.datos.contenido)
	if p.datos.huellaSHA256 != hex.EncodeToString(suma[:]) {
		return ErrPreparacionExportacionInvalida
	}
	return nil
}

func (p PreparacionExportacion) Contenido() ([]byte, error) {
	if p.Validar() != nil {
		return nil, ErrPreparacionExportacionInvalida
	}
	return append([]byte(nil), p.datos.contenido...), nil
}

// ContenidoParaFirma devuelve los mismos bytes del fichero mediante una copia
// independiente. No selecciona algoritmo, proveedor, clave ni certificado.
func (p PreparacionExportacion) ContenidoParaFirma() ([]byte, error) {
	return p.Contenido()
}

func (p PreparacionExportacion) HuellaSHA256() (string, error) {
	if p.Validar() != nil {
		return "", ErrPreparacionExportacionInvalida
	}
	return p.datos.huellaSHA256, nil
}

func (p PreparacionExportacion) Metadatos() (MetadatosPreparacionExportacion, error) {
	if p.Validar() != nil {
		return MetadatosPreparacionExportacion{}, ErrPreparacionExportacionInvalida
	}
	return p.datos.metadatosOpacos, nil
}

func (m MetadatosPreparacionExportacion) validar() bool {
	return m.VersionExpediente > 0 && m.ExpedienteRef != "" &&
		m.IncorporacionRef != "" && m.ProcedenciaModeloRef != "" &&
		m.CorrelacionRef != "" && m.IdempotenciaRef != "" &&
		m.MapeoRef != "" && m.MapeoVersion > 0 &&
		m.ProcedenciaMapeoRef != ""
}
