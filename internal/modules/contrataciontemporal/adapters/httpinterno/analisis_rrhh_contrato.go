package httpinterno

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	MaximoCuerpoAnalisisRRHHBytes  = 64 * 1024
	MaximoAniosPeriodoAnalisisRRHH = 100
	esquemaReciboAnalisisRRHH      = "vec.contratacion-temporal.recibo-analisis-rrhh.v1"
)

var (
	errEntradaAnalisisRRHHInvalida = errors.New(
		"contratacion temporal http: entrada de analisis RRHH invalida",
	)
	errContenidoAnalisisRRHHNoValido = errors.New(
		"contratacion temporal http: contenido de analisis RRHH no valido",
	)
	errCuerpoAnalisisRRHHDemasiadoGrande = errors.New(
		"contratacion temporal http: cuerpo de analisis RRHH demasiado grande",
	)
)

type datosAnalisisRRHHJSON struct {
	ModalidadClave    string                 `json:"modalidad_clave"`
	CategoriaRef      string                 `json:"categoria_ref"`
	GrupoSubgrupo     string                 `json:"grupo_subgrupo"`
	CausaClave        string                 `json:"causa_clave"`
	Periodo           *periodoPrevistoJSON   `json:"periodo"`
	PorcentajeJornada *uint16                `json:"porcentaje_jornada"`
	EntradaRC         *entradaRCAnalisisJSON `json:"entrada_rc"`
}

type entradaRCAnalisisJSON struct {
	Referencia   string `json:"referencia"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type solicitudRegistroAnalisisRRHHJSON struct {
	ExpedienteRef     string                 `json:"expediente_ref"`
	VersionEsperada   uint64                 `json:"version_esperada"`
	ClaveIdempotencia string                 `json:"clave_idempotencia"`
	ArtefactoRef      string                 `json:"artefacto_ref"`
	Analisis          *datosAnalisisRRHHJSON `json:"analisis"`
}

type solicitudRectificacionAnalisisRRHHJSON struct {
	ExpedienteRef            string                 `json:"expediente_ref"`
	VersionEsperada          uint64                 `json:"version_esperada"`
	ClaveIdempotencia        string                 `json:"clave_idempotencia"`
	ArtefactoRef             string                 `json:"artefacto_ref"`
	Analisis                 *datosAnalisisRRHHJSON `json:"analisis"`
	MotivoRectificacionClave string                 `json:"motivo_rectificacion_clave"`
}

type entradaOperacionAnalisisRRHH struct {
	operacion           ports.TipoOperacionAnalisis
	expedienteRef       string
	versionEsperada     uint64
	claveIdempotencia   string
	artefactoRef        string
	datosFuncionales    ports.DatosFuncionalesOperacionAnalisis
	motivoRectificacion domain.ClaveCatalogo
}

func operacionAnalisisRRHHDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (entradaOperacionAnalisisRRHH, error) {
	if r.URL.Path == RutaRegistroAnalisisRRHH {
		var entrada solicitudRegistroAnalisisRRHHJSON
		if err := decodificarAnalisisRRHH(w, r, &entrada); err != nil {
			return entradaOperacionAnalisisRRHH{}, err
		}
		return nuevaEntradaOperacionAnalisisRRHH(
			ports.OperacionRegistrarAnalisis,
			entrada.ExpedienteRef,
			entrada.VersionEsperada,
			entrada.ClaveIdempotencia,
			entrada.ArtefactoRef,
			entrada.Analisis,
			"",
		)
	}
	var entrada solicitudRectificacionAnalisisRRHHJSON
	if err := decodificarAnalisisRRHH(w, r, &entrada); err != nil {
		return entradaOperacionAnalisisRRHH{}, err
	}
	return nuevaEntradaOperacionAnalisisRRHH(
		ports.OperacionRectificarAnalisis,
		entrada.ExpedienteRef,
		entrada.VersionEsperada,
		entrada.ClaveIdempotencia,
		entrada.ArtefactoRef,
		entrada.Analisis,
		domain.ClaveCatalogo(entrada.MotivoRectificacionClave),
	)
}

func decodificarAnalisisRRHH(
	w http.ResponseWriter,
	r *http.Request,
	destino any,
) error {
	lector := http.MaxBytesReader(w, r.Body, MaximoCuerpoAnalisisRRHHBytes+1)
	contenido, err := io.ReadAll(lector)
	if err != nil {
		var limite *http.MaxBytesError
		if errors.As(err, &limite) {
			return errCuerpoAnalisisRRHHDemasiadoGrande
		}
		return errEntradaAnalisisRRHHInvalida
	}
	if len(contenido) == 0 || !utf8.Valid(contenido) {
		return errEntradaAnalisisRRHHInvalida
	}
	if len(contenido) > MaximoCuerpoAnalisisRRHHBytes {
		return errCuerpoAnalisisRRHHDemasiadoGrande
	}
	if err := validarJSONAltaSinDuplicados(contenido); err != nil {
		if errors.Is(err, errCuerpoAltaDemasiadoGrande) {
			return errCuerpoAnalisisRRHHDemasiadoGrande
		}
		return errEntradaAnalisisRRHHInvalida
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return errEntradaAnalisisRRHHInvalida
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return errEntradaAnalisisRRHHInvalida
	}
	return nil
}

func nuevaEntradaOperacionAnalisisRRHH(
	operacion ports.TipoOperacionAnalisis,
	expedienteRef string,
	versionEsperada uint64,
	claveIdempotencia string,
	artefactoRef string,
	analisis *datosAnalisisRRHHJSON,
	motivoRectificacion domain.ClaveCatalogo,
) (entradaOperacionAnalisisRRHH, error) {
	if analisis == nil || analisis.Periodo == nil ||
		analisis.PorcentajeJornada == nil || analisis.EntradaRC == nil {
		return entradaOperacionAnalisisRRHH{},
			errContenidoAnalisisRRHHNoValido
	}
	inicio, errInicio := fechaCivilUTC(analisis.Periodo.Inicio)
	fin, errFin := fechaCivilUTC(analisis.Periodo.Fin)
	datos := ports.DatosFuncionalesOperacionAnalisis{
		ModalidadClave: domain.ClaveCatalogo(analisis.ModalidadClave),
		CategoriaRef:   analisis.CategoriaRef,
		GrupoSubgrupo:  analisis.GrupoSubgrupo,
		CausaClave:     domain.ClaveCatalogo(analisis.CausaClave),
		Periodo: domain.PeriodoPrevisto{
			Inicio: inicio,
			Fin:    fin,
		},
		PorcentajeJornada: domain.JornadaDiezmilesimas(
			*analisis.PorcentajeJornada,
		),
		EntradaRC: domain.VinculoEntradaRC{
			Referencia:   analisis.EntradaRC.Referencia,
			HuellaSHA256: analisis.EntradaRC.HuellaSHA256,
		},
	}
	if errInicio != nil || errFin != nil ||
		!operacion.Valida() ||
		!domain.ReferenciaOpacaValida(expedienteRef) ||
		!ports.VersionOperacionAnalisisConIncrementoValida(versionEsperada) ||
		!ports.ClaveIdempotenciaValida(claveIdempotencia) ||
		!domain.ReferenciaOpacaValida(artefactoRef) ||
		datos.Validar() != nil ||
		fin.After(inicio.AddDate(MaximoAniosPeriodoAnalisisRRHH, 0, 0)) {
		return entradaOperacionAnalisisRRHH{},
			errContenidoAnalisisRRHHNoValido
	}
	if operacion == ports.OperacionRegistrarAnalisis {
		if motivoRectificacion != "" {
			return entradaOperacionAnalisisRRHH{},
				errContenidoAnalisisRRHHNoValido
		}
	} else if !motivoRectificacion.Valida() {
		return entradaOperacionAnalisisRRHH{},
			errContenidoAnalisisRRHHNoValido
	}
	return entradaOperacionAnalisisRRHH{
		operacion:           operacion,
		expedienteRef:       expedienteRef,
		versionEsperada:     versionEsperada,
		claveIdempotencia:   claveIdempotencia,
		artefactoRef:        artefactoRef,
		datosFuncionales:    datos,
		motivoRectificacion: motivoRectificacion,
	}, nil
}

func (e entradaOperacionAnalisisRRHH) solicitudRegistro(
	c ContextoCanalAnalisisRRHH,
) application.SolicitudRegistrarAnalisis {
	return application.SolicitudRegistrarAnalisis{
		AutenticacionRef:  c.AutenticacionRef,
		SesionRef:         c.SesionRef,
		PerfilRef:         c.PerfilRef,
		OrganizacionRef:   c.OrganizacionRef,
		ExpedienteRef:     e.expedienteRef,
		VersionEsperada:   e.versionEsperada,
		ClaveIdempotencia: e.claveIdempotencia,
		ArtefactoRef:      e.artefactoRef,
		DatosFuncionales:  e.datosFuncionales,
	}
}

func (e entradaOperacionAnalisisRRHH) solicitudRectificacion(
	c ContextoCanalAnalisisRRHH,
) application.SolicitudRectificarAnalisis {
	return application.SolicitudRectificarAnalisis{
		AutenticacionRef:         c.AutenticacionRef,
		SesionRef:                c.SesionRef,
		PerfilRef:                c.PerfilRef,
		OrganizacionRef:          c.OrganizacionRef,
		ExpedienteRef:            e.expedienteRef,
		VersionEsperada:          e.versionEsperada,
		ClaveIdempotencia:        e.claveIdempotencia,
		ArtefactoRef:             e.artefactoRef,
		DatosFuncionales:         e.datosFuncionales,
		MotivoRectificacionClave: e.motivoRectificacion,
	}
}

func reciboAnalisisRRHHEsSeguro(
	r ports.ReciboOperacionAnalisis,
	c ContextoCanalAnalisisRRHH,
	e entradaOperacionAnalisisRRHH,
) bool {
	return r.Operacion == e.operacion &&
		r.OrganizacionRef == c.OrganizacionRef &&
		r.ExpedienteRef == e.expedienteRef &&
		r.VersionAnterior == e.versionEsperada &&
		ports.VersionOperacionAnalisisConIncrementoValida(r.VersionAnterior) &&
		r.VersionResultante == r.VersionAnterior+1 &&
		r.VersionResultante <= ports.MaximoEnteroSeguroOperacionAnalisis &&
		r.SecuenciaActuacion == r.VersionResultante &&
		r.ArtefactoRef == e.artefactoRef &&
		huellaCoberturaValida(r.ArtefactoHuellaSHA256) &&
		domain.ReferenciaOpacaValida(r.ReciboRef) &&
		domain.ReferenciaOpacaValida(r.AuditoriaRef) &&
		domain.ReferenciaOpacaValida(r.EventoRef) &&
		domain.ReferenciaOpacaValida(r.ConsumoFuentesRef) &&
		huellaCoberturaValida(r.HuellaConsumoFuentes) &&
		domain.ReferenciaOpacaValida(r.ConcesionV3DecisionRef) &&
		ports.SelloHMACSHA256Valido(r.HuellaSemanticaHMAC) &&
		ports.SelloHMACSHA256Valido(r.AmbitoConsultaHMAC) &&
		ports.SelloHMACSHA256Valido(r.HuellaConsultaHMAC) &&
		domain.InstanteUTCCanonico(r.ConfirmadaEn)
}

type envoltorioReciboAnalisisRRHH struct {
	Data reciboAnalisisRRHHJSON `json:"data"`
}

type reciboAnalisisRRHHJSON struct {
	Esquema           string `json:"esquema"`
	Operacion         string `json:"operacion"`
	ExpedienteRef     string `json:"expediente_ref"`
	VersionResultante uint64 `json:"version_resultante"`
	ReciboRef         string `json:"recibo_ref"`
	ConfirmadaEn      string `json:"confirmada_en"`
}

func responderExitoAnalisisRRHH(
	w http.ResponseWriter,
	recibo ports.ReciboOperacionAnalisis,
) {
	responderJSONCobertura(
		w,
		http.StatusCreated,
		envoltorioReciboAnalisisRRHH{Data: reciboAnalisisRRHHJSON{
			Esquema:           esquemaReciboAnalisisRRHH,
			Operacion:         string(recibo.Operacion),
			ExpedienteRef:     recibo.ExpedienteRef,
			VersionResultante: recibo.VersionResultante,
			ReciboRef:         recibo.ReciboRef,
			ConfirmadaEn: recibo.ConfirmadaEn.UTC().Format(
				time.RFC3339Nano,
			),
		}},
	)
}

func errorEntradaAnalisisRRHH(err error) errorPublicoCobertura {
	if errors.Is(err, errCuerpoAnalisisRRHHDemasiadoGrande) {
		return errorCuerpoCoberturaDemasiadoGrande
	}
	if errors.Is(err, errContenidoAnalisisRRHHNoValido) {
		return errorContenidoCoberturaInvalido
	}
	return errorPeticionCoberturaNoValida
}
