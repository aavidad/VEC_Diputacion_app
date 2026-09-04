package domain

import (
	"errors"
	"time"
)

const (
	AccionRegistrarFiscalizacion ClaveCatalogo = "contratacion_temporal.fiscalizacion.registrar"
	FaseFiscalizacion            ClaveFase     = "fiscalizacion"
	FaseSubsanacionUnidad        ClaveFase     = "subsanacion_unidad"
)

type ResultadoFiscalizacion string

const (
	FiscalizacionFavorable                 ResultadoFiscalizacion = "favorable"
	FiscalizacionFavorableConObservaciones ResultadoFiscalizacion = "favorable_con_observaciones"
	FiscalizacionDesfavorable              ResultadoFiscalizacion = "desfavorable"
)

func (r ResultadoFiscalizacion) Valido() bool {
	switch r {
	case FiscalizacionFavorable,
		FiscalizacionFavorableConObservaciones,
		FiscalizacionDesfavorable:
		return true
	default:
		return false
	}
}

type EstadoRetornoFiscalizacion string

const EstadoRetornoFiscalizacionPendiente EstadoRetornoFiscalizacion = "pendiente"

var ErrFiscalizacionInvalida = errors.New(
	"contratacion temporal: fiscalizacion invalida",
)

// RetornoFiscalizacionUnidad conserva el destino operativo derivado de la
// asignacion vigente. Nunca se construye con una unidad aportada por cliente.
type RetornoFiscalizacionUnidad struct {
	RetornoRef     string                     `json:"retorno_ref"`
	UnidadRef      string                     `json:"unidad_ref"`
	ResponsableRef string                     `json:"responsable_ref"`
	Estado         EstadoRetornoFiscalizacion `json:"estado"`
	CreadoEn       time.Time                  `json:"creado_en"`
}

func (r RetornoFiscalizacionUnidad) validar() error {
	if !referenciaValida(r.RetornoRef) || !referenciaValida(r.UnidadRef) ||
		!referenciaValida(r.ResponsableRef) ||
		r.Estado != EstadoRetornoFiscalizacionPendiente ||
		!instanteCanonico(r.CreadoEn) {
		return ErrFiscalizacionInvalida
	}
	return nil
}

type FiscalizacionRegistrada struct {
	FiscalizacionRef       string                         `json:"fiscalizacion_ref"`
	Resultado              ResultadoFiscalizacion         `json:"resultado"`
	UnidadFiscalizadoraRef string                         `json:"unidad_fiscalizadora_ref"`
	InformeJuridicoRef     string                         `json:"informe_juridico_ref"`
	DocumentoInformeRef    string                         `json:"documento_informe_ref"`
	Observaciones          string                         `json:"observaciones,omitempty"`
	FiscalizadaEn          time.Time                      `json:"fiscalizada_en"`
	Retorno                *RetornoFiscalizacionUnidad    `json:"retorno,omitempty"`
	ActuacionRegistro      *VinculoActuacionFiscalizacion `json:"actuacion_registro"`
}

type VinculoActuacionFiscalizacion struct {
	Secuencia              uint64                 `json:"secuencia"`
	VersionExpediente      uint64                 `json:"version_expediente"`
	AccionClave            ClaveCatalogo          `json:"accion_clave"`
	FaseDestino            ClaveFase              `json:"fase_destino"`
	EstadoDestino          EstadoOperativo        `json:"estado_destino"`
	ReciboRef              string                 `json:"recibo_ref"`
	FiscalizacionRef       string                 `json:"fiscalizacion_ref"`
	Resultado              ResultadoFiscalizacion `json:"resultado"`
	UnidadFiscalizadoraRef string                 `json:"unidad_fiscalizadora_ref"`
	InformeJuridicoRef     string                 `json:"informe_juridico_ref"`
	DocumentoInformeRef    string                 `json:"documento_informe_ref"`
	RetornoRef             string                 `json:"retorno_ref,omitempty"`
	UnidadRetornoRef       string                 `json:"unidad_retorno_ref,omitempty"`
	ResponsableRetornoRef  string                 `json:"responsable_retorno_ref,omitempty"`
}

type DatosRegistrarFiscalizacion struct {
	FiscalizacionRef       string
	Resultado              ResultadoFiscalizacion
	UnidadFiscalizadoraRef string
	Observaciones          string
	FiscalizadaEn          time.Time
	RetornoRef             string
}

func (f FiscalizacionRegistrada) Validar() error {
	if !referenciaValida(f.FiscalizacionRef) || !f.Resultado.Valido() ||
		!referenciaValida(f.UnidadFiscalizadoraRef) ||
		!referenciaValida(f.InformeJuridicoRef) ||
		!referenciaValida(f.DocumentoInformeRef) ||
		!textoValido(f.Observaciones, 2000, true) ||
		!instanteCanonico(f.FiscalizadaEn) || f.ActuacionRegistro == nil ||
		f.ActuacionRegistro.validar() != nil {
		return ErrFiscalizacionInvalida
	}
	if f.Resultado == FiscalizacionFavorable && f.Observaciones != "" {
		return ErrFiscalizacionInvalida
	}
	if (f.Resultado == FiscalizacionFavorableConObservaciones ||
		f.Resultado == FiscalizacionDesfavorable) &&
		!textoValido(f.Observaciones, 2000, false) {
		return ErrFiscalizacionInvalida
	}
	if f.Resultado == FiscalizacionDesfavorable {
		if f.Retorno == nil || f.Retorno.validar() != nil {
			return ErrFiscalizacionInvalida
		}
	} else if f.Retorno != nil {
		return ErrFiscalizacionInvalida
	}
	return nil
}

func (v VinculoActuacionFiscalizacion) validar() error {
	if v.Secuencia != 6 || v.VersionExpediente != 6 ||
		v.AccionClave != AccionRegistrarFiscalizacion ||
		!v.Resultado.Valido() || !referenciaValida(v.ReciboRef) ||
		!referenciaValida(v.FiscalizacionRef) ||
		!referenciaValida(v.UnidadFiscalizadoraRef) ||
		!referenciaValida(v.InformeJuridicoRef) ||
		!referenciaValida(v.DocumentoInformeRef) {
		return ErrFiscalizacionInvalida
	}
	if v.Resultado == FiscalizacionDesfavorable {
		if v.FaseDestino != FaseSubsanacionUnidad ||
			v.EstadoDestino != EstadoIncidencia ||
			!referenciaValida(v.RetornoRef) ||
			!referenciaValida(v.UnidadRetornoRef) ||
			!referenciaValida(v.ResponsableRetornoRef) {
			return ErrFiscalizacionInvalida
		}
		return nil
	}
	if v.FaseDestino != FaseFiscalizacion || v.EstadoDestino != EstadoEnCurso ||
		v.RetornoRef != "" || v.UnidadRetornoRef != "" ||
		v.ResponsableRetornoRef != "" {
		return ErrFiscalizacionInvalida
	}
	return nil
}

func (v VinculoActuacionFiscalizacion) correspondeA(
	actuacion Actuacion,
	fiscalizacion FiscalizacionRegistrada,
) bool {
	if v.validar() != nil || v.Secuencia != actuacion.Secuencia ||
		v.VersionExpediente != actuacion.VersionExpediente ||
		v.AccionClave != actuacion.AccionClave ||
		v.FaseDestino != actuacion.FaseDestino ||
		v.EstadoDestino != actuacion.EstadoDestino ||
		v.ReciboRef != actuacion.ReciboRef ||
		v.UnidadFiscalizadoraRef != actuacion.UnidadRef ||
		v.FiscalizacionRef != fiscalizacion.FiscalizacionRef ||
		v.Resultado != fiscalizacion.Resultado ||
		v.InformeJuridicoRef != fiscalizacion.InformeJuridicoRef ||
		v.DocumentoInformeRef != fiscalizacion.DocumentoInformeRef ||
		len(actuacion.DocumentosRef) != 1 ||
		actuacion.DocumentosRef[0] != fiscalizacion.DocumentoInformeRef {
		return false
	}
	if fiscalizacion.Retorno == nil {
		return v.RetornoRef == "" && v.UnidadRetornoRef == "" &&
			v.ResponsableRetornoRef == ""
	}
	return v.RetornoRef == fiscalizacion.Retorno.RetornoRef &&
		v.UnidadRetornoRef == fiscalizacion.Retorno.UnidadRef &&
		v.ResponsableRetornoRef == fiscalizacion.Retorno.ResponsableRef
}

func (e Expediente) RegistrarFiscalizacion(
	versionEsperada uint64,
	datos DatosRegistrarFiscalizacion,
	actuacion DatosActuacion,
) (Expediente, error) {
	if e.Validar() != nil || versionEsperada != 5 || e.Version != 5 ||
		e.Asignacion == nil || e.InformeJuridico == nil || e.Fiscalizacion != nil ||
		e.FaseActual != FaseInformeJuridico || e.EstadoActual != EstadoEnCurso ||
		!referenciaValida(datos.FiscalizacionRef) || !datos.Resultado.Valido() ||
		!referenciaValida(datos.UnidadFiscalizadoraRef) ||
		!textoValido(datos.Observaciones, 2000, true) ||
		!instanteCanonico(datos.FiscalizadaEn) || actuacion.validar() != nil ||
		actuacion.AccionClave != AccionRegistrarFiscalizacion ||
		actuacion.UnidadRef != datos.UnidadFiscalizadoraRef ||
		!actuacion.RealizadaEn.Equal(datos.FiscalizadaEn) ||
		actuacion.Observaciones != datos.Observaciones ||
		len(actuacion.DocumentosRef) != 1 ||
		actuacion.DocumentosRef[0] != e.InformeJuridico.DocumentoRef {
		return Expediente{}, ErrTransicionInvalida
	}

	fiscalizacion := FiscalizacionRegistrada{
		FiscalizacionRef:       datos.FiscalizacionRef,
		Resultado:              datos.Resultado,
		UnidadFiscalizadoraRef: datos.UnidadFiscalizadoraRef,
		InformeJuridicoRef:     e.InformeJuridico.InformeRef,
		DocumentoInformeRef:    e.InformeJuridico.DocumentoRef,
		Observaciones:          datos.Observaciones,
		FiscalizadaEn:          datos.FiscalizadaEn,
	}

	faseDestino := FaseFiscalizacion
	estadoDestino := EstadoEnCurso
	if datos.Resultado == FiscalizacionFavorable {
		if datos.Observaciones != "" || datos.RetornoRef != "" {
			return Expediente{}, ErrTransicionInvalida
		}
	} else if datos.Resultado == FiscalizacionFavorableConObservaciones {
		if !textoValido(datos.Observaciones, 2000, false) || datos.RetornoRef != "" {
			return Expediente{}, ErrTransicionInvalida
		}
	} else {
		if !textoValido(datos.Observaciones, 2000, false) ||
			!referenciaValida(datos.RetornoRef) {
			return Expediente{}, ErrTransicionInvalida
		}
		faseDestino = FaseSubsanacionUnidad
		estadoDestino = EstadoIncidencia
		fiscalizacion.Retorno = &RetornoFiscalizacionUnidad{
			RetornoRef: datos.RetornoRef, UnidadRef: e.Asignacion.UnidadRef,
			ResponsableRef: e.Asignacion.ResponsableRef,
			Estado:         EstadoRetornoFiscalizacionPendiente, CreadoEn: datos.FiscalizadaEn,
		}
	}
	if actuacion.FaseDestino != faseDestino ||
		actuacion.EstadoDestino != estadoDestino {
		return Expediente{}, ErrTransicionInvalida
	}

	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	vinculo := nuevoVinculoActuacionFiscalizacion(actuacion, fiscalizacion)
	fiscalizacion.ActuacionRegistro = &vinculo
	siguiente.Fiscalizacion = &fiscalizacion
	return siguiente.confirmarTransicion(actuacion)
}

func nuevoVinculoActuacionFiscalizacion(
	actuacion DatosActuacion,
	fiscalizacion FiscalizacionRegistrada,
) VinculoActuacionFiscalizacion {
	vinculo := VinculoActuacionFiscalizacion{
		Secuencia: 6, VersionExpediente: 6,
		AccionClave: actuacion.AccionClave, FaseDestino: actuacion.FaseDestino,
		EstadoDestino: actuacion.EstadoDestino, ReciboRef: actuacion.ReciboRef,
		FiscalizacionRef:       fiscalizacion.FiscalizacionRef,
		Resultado:              fiscalizacion.Resultado,
		UnidadFiscalizadoraRef: fiscalizacion.UnidadFiscalizadoraRef,
		InformeJuridicoRef:     fiscalizacion.InformeJuridicoRef,
		DocumentoInformeRef:    fiscalizacion.DocumentoInformeRef,
	}
	if fiscalizacion.Retorno != nil {
		vinculo.RetornoRef = fiscalizacion.Retorno.RetornoRef
		vinculo.UnidadRetornoRef = fiscalizacion.Retorno.UnidadRef
		vinculo.ResponsableRetornoRef = fiscalizacion.Retorno.ResponsableRef
	}
	return vinculo
}

func fiscalizacionLigadaAActuacion(
	fiscalizacion *FiscalizacionRegistrada,
	actuaciones []Actuacion,
) bool {
	if fiscalizacion == nil || fiscalizacion.ActuacionRegistro == nil ||
		fiscalizacion.ActuacionRegistro.validar() != nil ||
		fiscalizacion.ActuacionRegistro.Secuencia > uint64(len(actuaciones)) {
		return false
	}
	actuacion := actuaciones[fiscalizacion.ActuacionRegistro.Secuencia-1]
	return fiscalizacion.ActuacionRegistro.correspondeA(actuacion, *fiscalizacion) &&
		fiscalizacion.FiscalizadaEn.Equal(actuacion.RealizadaEn) &&
		fiscalizacion.Observaciones == actuacion.Observaciones
}

func (f FiscalizacionRegistrada) clonar() FiscalizacionRegistrada {
	if f.Retorno != nil {
		retorno := *f.Retorno
		f.Retorno = &retorno
	}
	if f.ActuacionRegistro != nil {
		vinculo := *f.ActuacionRegistro
		f.ActuacionRegistro = &vinculo
	}
	return f
}
