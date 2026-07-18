package almacen

import (
	"fmt"
	"io"
	"log/slog"
	"time"
)

// PasoOperacionAlmacen identifica un paso cerrado comprometido por el nucleo.
type PasoOperacionAlmacen string

const (
	PasoPrepararCargaDirecta  PasoOperacionAlmacen = "01_preparar_carga_directa"
	PasoAbandonarCargaDirecta PasoOperacionAlmacen = "02_abandonar_carga_directa"
	PasoConfirmarCargaDirecta PasoOperacionAlmacen = "01_confirmar_carga_directa"
	PasoLeerParaAnalisis      PasoOperacionAlmacen = "01_leer_para_analisis"
	PasoAnalizarContenido     PasoOperacionAlmacen = "02_analizar_contenido"
	PasoPromover              PasoOperacionAlmacen = "01_promover"
	PasoCustodiarDecision     PasoOperacionAlmacen = "01_custodiar_decision"
	PasoCustodiarFirmado      PasoOperacionAlmacen = "01_custodiar_documento_firmado"
	PasoRetenerFirmado        PasoOperacionAlmacen = "01_retener_documento_firmado"
)

// ProyeccionContextoOperacionAlmacen es una copia defensiva que no permite
// reconstruir la capacidad opaca de la que procede.
type ProyeccionContextoOperacionAlmacen struct {
	Esquema                string
	OperacionRef           string
	CorrelacionRef         string
	AutorizacionRef        string
	Finalidad              string
	Clasificacion          string
	AccionNegocio          string
	AccionTecnica          string
	CargaRef               string
	SujetoSeudonimoHMAC    string
	RecursoRef             string
	ModuloID               string
	TipoRecurso            string
	HuellaRecursoSHA256    string
	HuellaSolicitudHMAC    string
	EfectoRef              string
	HuellaPlanEfectoSHA256 string
	HuellaManifiestoSHA256 string
	HuellaPasoSHA256       string
	PasoRef                PasoOperacionAlmacen
	ObjetoVinculado        ReferenciaObjetoAlmacen
	HuellaDecisionSHA256   string
	VerificadaEn           time.Time
	ValidaHasta            time.Time
}

func (ProyeccionContextoOperacionAlmacen) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionContextoAlmacenProhibida
}

func (*ProyeccionContextoOperacionAlmacen) UnmarshalJSON([]byte) error {
	return ErrSerializacionContextoAlmacenProhibida
}

func (ProyeccionContextoOperacionAlmacen) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionContextoAlmacenProhibida
}

func (*ProyeccionContextoOperacionAlmacen) UnmarshalText([]byte) error {
	return ErrSerializacionContextoAlmacenProhibida
}

func (ProyeccionContextoOperacionAlmacen) String() string {
	return "[PROYECCION-CONTEXTO-OPERACION-ALMACEN-INTERNA]"
}

func (p ProyeccionContextoOperacionAlmacen) GoString() string { return p.String() }

func (p ProyeccionContextoOperacionAlmacen) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}

func (p ProyeccionContextoOperacionAlmacen) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
