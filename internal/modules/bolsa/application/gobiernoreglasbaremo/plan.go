package gobiernoreglasbaremo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	esquemaPlanCambioV2     = "vec.bolsa.gobierno-reglas-baremo.plan-cambio.v2"
	prefijoIntencionV2      = "intencion:reglas-baremo:v2:"
	prefijoAtestacionV2     = "atestacion:reglas-baremo:v2:"
	versionReferenciaV2     = 2
	maximoBytesPlanCambioV2 = 8 * 1024 * 1024
)

// IntencionGobiernoReglasBaremoV2 conserva una identidad idempotente declarada.
// Su validacion sintactica no le otorga autoridad ni demuestra unicidad. La
// generacion segura y su persistencia corresponden al servicio y al adaptador.
type IntencionGobiernoReglasBaremoV2 struct {
	bloqueoSerializacion
	referencia reglas.ReferenciaVersionada
}

func NuevaIntencionGobiernoReglasBaremoV2(
	referencia reglas.ReferenciaVersionada,
) (IntencionGobiernoReglasBaremoV2, error) {
	if !referenciaTecnicaV2Valida(referencia, prefijoIntencionV2) {
		return IntencionGobiernoReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	return IntencionGobiernoReglasBaremoV2{referencia: referencia}, nil
}

func (i IntencionGobiernoReglasBaremoV2) Referencia() (
	reglas.ReferenciaVersionada,
	error,
) {
	if !referenciaTecnicaV2Valida(i.referencia, prefijoIntencionV2) {
		return reglas.ReferenciaVersionada{}, ErrPlanCambioInvalido
	}
	return i.referencia, nil
}

// VinculoEvidenciaTransicionReglasBaremoV2 solo se construye desde evidencia
// de dominio estructuralmente valida. Liga operacion, CAS y referencia exacta,
// pero no es autoridad ni capacidad; el plan coteja despues su incorporacion.
type VinculoEvidenciaTransicionReglasBaremoV2 struct {
	bloqueoSerializacion
	operacion    OperacionGobiernoReglasBaremoV2
	atestacion   reglas.ReferenciaVersionada
	cas          reglas.VinculoEstadoReglasBaremo
	aprobacion   reglas.AtestacionAprobacionFirmadaReglasBaremo
	dependencias reglas.AtestacionDependenciasVigentesReglasBaremo
	autoridad    reglas.AtestacionAutoridadReglasBaremo
}

func NuevoVinculoEvidenciaPublicacionReglasBaremoV2(
	evidencia reglas.AtestacionAprobacionFirmadaReglasBaremo,
) (VinculoEvidenciaTransicionReglasBaremoV2, error) {
	return validarNuevoVinculoEvidencia(VinculoEvidenciaTransicionReglasBaremoV2{
		operacion:  OperacionPublicar,
		atestacion: evidencia.Atestacion(),
		cas:        evidencia.Vinculo(),
		aprobacion: evidencia,
	})
}

func NuevoVinculoEvidenciaActivacionReglasBaremoV2(
	evidencia reglas.AtestacionDependenciasVigentesReglasBaremo,
) (VinculoEvidenciaTransicionReglasBaremoV2, error) {
	return validarNuevoVinculoEvidencia(VinculoEvidenciaTransicionReglasBaremoV2{
		operacion:    OperacionActivar,
		atestacion:   evidencia.Atestacion(),
		cas:          evidencia.Vinculo(),
		dependencias: evidencia,
	})
}

func NuevoVinculoEvidenciaTerminalReglasBaremoV2(
	evidencia reglas.AtestacionAutoridadReglasBaremo,
) (VinculoEvidenciaTransicionReglasBaremoV2, error) {
	var operacion OperacionGobiernoReglasBaremoV2
	switch evidencia.Accion() {
	case reglas.AccionSustituirReglasBaremo:
		operacion = OperacionSustituir
	case reglas.AccionRetirarReglasBaremo:
		operacion = OperacionRetirar
	case reglas.AccionDescartarReglasBaremo:
		operacion = OperacionDescartar
	default:
		return VinculoEvidenciaTransicionReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	return validarNuevoVinculoEvidencia(VinculoEvidenciaTransicionReglasBaremoV2{
		operacion:  operacion,
		atestacion: evidencia.Atestacion(),
		cas:        evidencia.Vinculo(),
		autoridad:  evidencia,
	})
}

func validarNuevoVinculoEvidencia(
	vinculo VinculoEvidenciaTransicionReglasBaremoV2,
) (VinculoEvidenciaTransicionReglasBaremoV2, error) {
	if vinculo.validar() != nil {
		return VinculoEvidenciaTransicionReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	return vinculo, nil
}

func (v VinculoEvidenciaTransicionReglasBaremoV2) Referencia() (
	reglas.ReferenciaVersionada,
	error,
) {
	if v.validar() != nil {
		return reglas.ReferenciaVersionada{}, ErrPlanCambioInvalido
	}
	return v.atestacion, nil
}

func (v VinculoEvidenciaTransicionReglasBaremoV2) validar() error {
	if !referenciaTecnicaV2Valida(v.atestacion, prefijoAtestacionV2) ||
		!vinculoEstadoValido(v.cas) {
		return ErrPlanCambioInvalido
	}
	switch v.operacion {
	case OperacionPublicar:
		if referenciasIguales(v.aprobacion.Atestacion(), v.atestacion) &&
			vinculosIguales(v.aprobacion.Vinculo(), v.cas) {
			return nil
		}
	case OperacionActivar:
		if referenciasIguales(v.dependencias.Atestacion(), v.atestacion) &&
			vinculosIguales(v.dependencias.Vinculo(), v.cas) {
			return nil
		}
	case OperacionSustituir, OperacionRetirar, OperacionDescartar:
		accionEsperada := reglas.AccionSustituirReglasBaremo
		if v.operacion == OperacionRetirar {
			accionEsperada = reglas.AccionRetirarReglasBaremo
		}
		if v.operacion == OperacionDescartar {
			accionEsperada = reglas.AccionDescartarReglasBaremo
		}
		if referenciasIguales(v.autoridad.Atestacion(), v.atestacion) &&
			vinculosIguales(v.autoridad.Vinculo(), v.cas) &&
			v.autoridad.Accion() == accionEsperada {
			return nil
		}
	}
	return ErrPlanCambioInvalido
}

type DatosNuevoPlanCambioReglasBaremoV2 struct {
	bloqueoSerializacion
	Operacion        OperacionGobiernoReglasBaremoV2
	Intencion        IntencionGobiernoReglasBaremoV2
	CASEsperado      *reglas.VinculoEstadoReglasBaremo
	VersionResultado reglas.VersionGobernadaReglasBaremo
	VinculoEvidencia *VinculoEvidenciaTransicionReglasBaremoV2
	ContextoActor    dominiovec.ContextoActor
	ReferenciaMotivo dominiovec.ReferenciaEntradaCatalogo
	Correlacion      dominiovec.ReferenciaCorrelacionAutorizacionV2
	// InstanteTransicion es el instante de negocio incorporado a la version.
	// El plan solo puede validar su forma y coherencia interna: la frontera
	// ejecutora debe obtenerlo o cotejarlo contra un reloj confiable y aplicar
	// su politica de frescura antes del COMMIT.
	InstanteTransicion time.Time
}

type datosPlanCambioReglasBaremoV2 struct {
	operacion              OperacionGobiernoReglasBaremoV2
	intencion              IntencionGobiernoReglasBaremoV2
	cas                    reglas.VinculoEstadoReglasBaremo
	tieneCAS               bool
	versionCanonica        []byte
	huellaVersionSHA256    string
	vinculoResultado       reglas.VinculoEstadoReglasBaremo
	vinculoEvidencia       VinculoEvidenciaTransicionReglasBaremoV2
	tieneVinculoEvidencia  bool
	principalRef           string
	referenciaMotivo       dominiovec.ReferenciaEntradaCatalogo
	motivoCanonico         []byte
	huellaMotivoSHA256     string
	correlacion            dominiovec.ReferenciaCorrelacionAutorizacionV2
	instanteTransicion     time.Time
	componentes            []ComponenteEscrituraReglasBaremoV2
	representacionCanonica []byte
	huellaPlanSHA256       string
}

// PlanCambioReglasBaremoV2 es un manifiesto de efecto de negocio propuesto. No
// es ejecutable, una autorizacion, una atestacion ni un recibo: su consumidor
// debe obtener y consumir una decision VEC V2 y volver a cotejar la evidencia
// mediante un verificador confiable antes de cualquier persistencia durable.
type PlanCambioReglasBaremoV2 struct {
	bloqueoSerializacion
	datos *datosPlanCambioReglasBaremoV2
}

func NuevoPlanCambioReglasBaremoV2(
	datos DatosNuevoPlanCambioReglasBaremoV2,
) (PlanCambioReglasBaremoV2, error) {
	if !datos.Operacion.esCambio() ||
		!instanteUTCMicrosegundos(datos.InstanteTransicion) ||
		datos.ContextoActor.Validar() != nil ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(datos.ReferenciaMotivo) ||
		datos.Correlacion.Validar() != nil {
		return PlanCambioReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	if _, err := datos.Intencion.Referencia(); err != nil {
		return PlanCambioReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	canonico, huella, vinculo, err := canonizarVersionResultado(datos.VersionResultado)
	if err != nil {
		return PlanCambioReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	motivoCanonico, err := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
		datos.ReferenciaMotivo,
	)
	if err != nil {
		return PlanCambioReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	huellaMotivo, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(datos.ReferenciaMotivo)
	if err != nil {
		return PlanCambioReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	interno := &datosPlanCambioReglasBaremoV2{
		operacion:           datos.Operacion,
		intencion:           datos.Intencion,
		versionCanonica:     append([]byte(nil), canonico...),
		huellaVersionSHA256: huella,
		vinculoResultado:    vinculo,
		principalRef:        datos.ContextoActor.Principal.ID,
		referenciaMotivo:    datos.ReferenciaMotivo,
		motivoCanonico:      append([]byte(nil), motivoCanonico...),
		huellaMotivoSHA256:  huellaMotivo,
		correlacion:         datos.Correlacion,
		instanteTransicion:  datos.InstanteTransicion,
		componentes:         componentesEscrituraFijos(),
	}
	if datos.CASEsperado != nil {
		interno.cas, interno.tieneCAS = *datos.CASEsperado, true
	}
	if datos.VinculoEvidencia != nil {
		interno.vinculoEvidencia, interno.tieneVinculoEvidencia = *datos.VinculoEvidencia, true
	}
	plan := PlanCambioReglasBaremoV2{datos: interno}
	if plan.validarEstructura() != nil {
		return PlanCambioReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	representacion, err := plan.representacionSinCotejo()
	if err != nil || len(representacion) == 0 || len(representacion) > maximoBytesPlanCambioV2 {
		return PlanCambioReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	suma := sha256.Sum256(representacion)
	plan.datos.representacionCanonica = append([]byte(nil), representacion...)
	plan.datos.huellaPlanSHA256 = hex.EncodeToString(suma[:])
	if plan.validar() != nil {
		return PlanCambioReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	return plan, nil
}

type DatosPlanCambioReglasBaremoV2 struct {
	bloqueoSerializacion
	Operacion             OperacionGobiernoReglasBaremoV2
	Intencion             IntencionGobiernoReglasBaremoV2
	CASEsperado           reglas.VinculoEstadoReglasBaremo
	TieneCAS              bool
	VersionResultado      reglas.VersionGobernadaReglasBaremo
	VersionCanonica       []byte
	HuellaVersionSHA256   string
	VinculoResultado      reglas.VinculoEstadoReglasBaremo
	VinculoEvidencia      VinculoEvidenciaTransicionReglasBaremoV2
	TieneVinculoEvidencia bool
	PrincipalRef          string
	ReferenciaMotivo      dominiovec.ReferenciaEntradaCatalogo
	MotivoCanonico        []byte
	HuellaMotivoSHA256    string
	Correlacion           dominiovec.ReferenciaCorrelacionAutorizacionV2
	InstanteTransicion    time.Time
	Componentes           []ComponenteEscrituraReglasBaremoV2
}

func (p PlanCambioReglasBaremoV2) Datos() (DatosPlanCambioReglasBaremoV2, error) {
	if p.validar() != nil {
		return DatosPlanCambioReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	version, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(
		p.datos.versionCanonica,
		p.datos.huellaVersionSHA256,
	)
	if err != nil {
		return DatosPlanCambioReglasBaremoV2{}, ErrPlanCambioInvalido
	}
	return DatosPlanCambioReglasBaremoV2{
		Operacion:             p.datos.operacion,
		Intencion:             p.datos.intencion,
		CASEsperado:           p.datos.cas,
		TieneCAS:              p.datos.tieneCAS,
		VersionResultado:      version,
		VersionCanonica:       append([]byte(nil), p.datos.versionCanonica...),
		HuellaVersionSHA256:   p.datos.huellaVersionSHA256,
		VinculoResultado:      p.datos.vinculoResultado,
		VinculoEvidencia:      p.datos.vinculoEvidencia,
		TieneVinculoEvidencia: p.datos.tieneVinculoEvidencia,
		PrincipalRef:          p.datos.principalRef,
		ReferenciaMotivo:      p.datos.referenciaMotivo,
		MotivoCanonico:        append([]byte(nil), p.datos.motivoCanonico...),
		HuellaMotivoSHA256:    p.datos.huellaMotivoSHA256,
		Correlacion:           p.datos.correlacion,
		InstanteTransicion:    p.datos.instanteTransicion,
		Componentes:           append([]ComponenteEscrituraReglasBaremoV2(nil), p.datos.componentes...),
	}, nil
}

func (p PlanCambioReglasBaremoV2) RepresentacionCanonica() ([]byte, error) {
	if p.validar() != nil {
		return nil, ErrPlanCambioInvalido
	}
	return append([]byte(nil), p.datos.representacionCanonica...), nil
}

func (p PlanCambioReglasBaremoV2) HuellaSHA256() (string, error) {
	if p.validar() != nil {
		return "", ErrPlanCambioInvalido
	}
	return p.datos.huellaPlanSHA256, nil
}

func (p PlanCambioReglasBaremoV2) ContratoAutorizacionV2() (
	ContratoAutorizacionV2,
	error,
) {
	if p.validar() != nil {
		return ContratoAutorizacionV2{}, ErrPlanCambioInvalido
	}
	version, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(
		p.datos.versionCanonica,
		p.datos.huellaVersionSHA256,
	)
	if err != nil {
		return ContratoAutorizacionV2{}, ErrPlanCambioInvalido
	}
	alcance, err := nuevoAlcanceAutorizacionDesdeVersion(version)
	if err != nil {
		return ContratoAutorizacionV2{}, ErrPlanCambioInvalido
	}
	return nuevoContratoAutorizacionV2(
		p.datos.operacion,
		p.datos.vinculoResultado.HuellaEstadoSHA256(),
		alcance,
	)
}

func (p PlanCambioReglasBaremoV2) validar() error {
	if p.validarEstructura() != nil || p.datos.huellaPlanSHA256 == "" ||
		len(p.datos.representacionCanonica) == 0 ||
		len(p.datos.representacionCanonica) > maximoBytesPlanCambioV2 {
		return ErrPlanCambioInvalido
	}
	representacion, err := p.representacionSinCotejo()
	if err != nil || !bytes.Equal(representacion, p.datos.representacionCanonica) {
		return ErrPlanCambioInvalido
	}
	suma := sha256.Sum256(representacion)
	if hex.EncodeToString(suma[:]) != p.datos.huellaPlanSHA256 {
		return ErrPlanCambioInvalido
	}
	return nil
}

func (p PlanCambioReglasBaremoV2) validarEstructura() error {
	if p.datos == nil || !p.datos.operacion.esCambio() ||
		!instanteUTCMicrosegundos(p.datos.instanteTransicion) ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(p.datos.referenciaMotivo) ||
		p.datos.correlacion.Validar() != nil ||
		!componentesExactos(p.datos.componentes) {
		return ErrPlanCambioInvalido
	}
	if _, err := p.datos.intencion.Referencia(); err != nil {
		return ErrPlanCambioInvalido
	}
	version, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(
		p.datos.versionCanonica,
		p.datos.huellaVersionSHA256,
	)
	if err != nil || !vinculoEstadoValido(p.datos.vinculoResultado) {
		return ErrPlanCambioInvalido
	}
	instanteVersion, err := version.InstanteUltimaActuacion()
	if err != nil || !instanteVersion.Equal(p.datos.instanteTransicion) {
		return ErrPlanCambioInvalido
	}
	actorVersion, err := version.ActorUltimaActuacion()
	if err != nil || actorVersion != p.datos.principalRef {
		return ErrPlanCambioInvalido
	}
	motivoVersion, err := version.MotivoUltimaActuacion()
	if err != nil || !motivoDominioCoincideConVEC(motivoVersion, p.datos.referenciaMotivo) {
		return ErrPlanCambioInvalido
	}
	motivoCanonico, err := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
		p.datos.referenciaMotivo,
	)
	if err != nil || !bytes.Equal(motivoCanonico, p.datos.motivoCanonico) {
		return ErrPlanCambioInvalido
	}
	huellaMotivo, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(
		p.datos.referenciaMotivo,
	)
	if err != nil || huellaMotivo != p.datos.huellaMotivoSHA256 {
		return ErrPlanCambioInvalido
	}
	vinculo, err := version.VinculoEstado()
	if err != nil || !vinculosIguales(vinculo, p.datos.vinculoResultado) {
		return ErrPlanCambioInvalido
	}
	estado, revision, existe := p.datos.operacion.estadoResultado()
	if !existe || version.Estado() != estado || version.Revision() != revision {
		return ErrPlanCambioInvalido
	}
	if p.datos.operacion == OperacionAltaBorrador {
		if p.datos.tieneCAS || p.datos.tieneVinculoEvidencia {
			return ErrPlanCambioInvalido
		}
		return nil
	}
	if !p.datos.tieneCAS || !p.datos.tieneVinculoEvidencia ||
		!vinculoEstadoValido(p.datos.cas) ||
		p.datos.cas.Revision()+1 != p.datos.vinculoResultado.Revision() {
		return ErrPlanCambioInvalido
	}
	if p.datos.vinculoEvidencia.validar() != nil ||
		p.datos.vinculoEvidencia.operacion != p.datos.operacion ||
		!vinculosIguales(p.datos.vinculoEvidencia.cas, p.datos.cas) ||
		!p.datos.vinculoEvidencia.incorporadaEn(version) {
		return ErrPlanCambioInvalido
	}
	if !referenciasIguales(
		p.datos.cas.Contenido(),
		p.datos.vinculoResultado.Contenido(),
	) {
		return ErrPlanCambioInvalido
	}
	return nil
}

type materialPlanCambioV2 struct {
	Esquema                      string                `json:"esquema"`
	Operacion                    string                `json:"operacion"`
	Intencion                    materialReferenciaV2  `json:"intencion"`
	CASEsperado                  *materialVinculoV2    `json:"cas_esperado"`
	VersionResultadoCanonica     []byte                `json:"version_resultado_canonica"`
	HuellaVersionResultadoSHA256 string                `json:"huella_version_resultado_sha256"`
	VinculoResultado             materialVinculoV2     `json:"vinculo_resultado"`
	VinculoEvidencia             *materialReferenciaV2 `json:"vinculo_evidencia"`
	PrincipalRef                 string                `json:"principal_ref"`
	MotivoCanonico               []byte                `json:"motivo_canonico"`
	HuellaMotivoSHA256           string                `json:"huella_motivo_sha256"`
	CorrelacionRef               string                `json:"correlacion_ref"`
	InstanteTransicion           string                `json:"instante_transicion"`
	Accion                       string                `json:"accion"`
	ModuloID                     string                `json:"modulo_id"`
	TipoRecurso                  string                `json:"tipo_recurso"`
	PerfilProteccion             string                `json:"perfil_proteccion"`
	RecursoRef                   string                `json:"recurso_ref"`
	ConvocatoriaRef              string                `json:"convocatoria_ref"`
	ExpedienteRef                string                `json:"expediente_ref"`
	HuellaContextoRecursoSHA256  string                `json:"huella_contexto_recurso_sha256"`
	Finalidad                    string                `json:"finalidad"`
	Campos                       []string              `json:"campos"`
	RequisitosEjecucion          []string              `json:"requisitos_ejecucion"`
	Componentes                  []string              `json:"componentes"`
}

type materialReferenciaV2 struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type materialVinculoV2 struct {
	Contenido          materialReferenciaV2 `json:"contenido"`
	Revision           uint64               `json:"revision"`
	HuellaEstadoSHA256 string               `json:"huella_estado_sha256"`
}

func (p PlanCambioReglasBaremoV2) representacionSinCotejo() ([]byte, error) {
	nombreOperacion, err := p.datos.operacion.nombreCanonico()
	if err != nil {
		return nil, err
	}
	intencion, err := p.datos.intencion.Referencia()
	if err != nil {
		return nil, err
	}
	correlacion, err := p.datos.correlacion.ValorCanonico()
	if err != nil {
		return nil, err
	}
	version, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(
		p.datos.versionCanonica,
		p.datos.huellaVersionSHA256,
	)
	if err != nil {
		return nil, err
	}
	alcance, err := nuevoAlcanceAutorizacionDesdeVersion(version)
	if err != nil {
		return nil, err
	}
	contrato, err := nuevoContratoAutorizacionV2(
		p.datos.operacion,
		p.datos.vinculoResultado.HuellaEstadoSHA256(),
		alcance,
	)
	if err != nil {
		return nil, err
	}
	recurso, err := contrato.Recurso()
	if err != nil {
		return nil, err
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return nil, err
	}
	especificacion, err := especificacionPara(p.datos.operacion)
	if err != nil {
		return nil, err
	}
	componentes := make([]string, len(p.datos.componentes))
	for indice, componente := range p.datos.componentes {
		componentes[indice], err = componente.nombreCanonico()
		if err != nil {
			return nil, err
		}
	}
	material := materialPlanCambioV2{
		Esquema:                      esquemaPlanCambioV2,
		Operacion:                    nombreOperacion,
		Intencion:                    materialReferencia(intencion),
		VersionResultadoCanonica:     append([]byte(nil), p.datos.versionCanonica...),
		HuellaVersionResultadoSHA256: p.datos.huellaVersionSHA256,
		VinculoResultado:             materialVinculo(p.datos.vinculoResultado),
		PrincipalRef:                 p.datos.principalRef,
		MotivoCanonico:               append([]byte(nil), p.datos.motivoCanonico...),
		HuellaMotivoSHA256:           p.datos.huellaMotivoSHA256,
		CorrelacionRef:               correlacion,
		InstanteTransicion:           p.datos.instanteTransicion.Format("2006-01-02T15:04:05.000000Z"),
		Accion:                       especificacion.accion,
		ModuloID:                     moduloBolsaGobiernoReglas,
		TipoRecurso:                  tipoRecursoReglasGobernadas,
		PerfilProteccion:             perfilProteccionReglas,
		RecursoRef:                   recurso.Referencia,
		ConvocatoriaRef:              recurso.Ambitos[ambitoConvocatoriaRef],
		ExpedienteRef:                recurso.Ambitos[ambitoExpedienteRef],
		HuellaContextoRecursoSHA256:  huellaContexto,
		Finalidad:                    especificacion.finalidad,
		Campos:                       append([]string(nil), especificacion.campos...),
		RequisitosEjecucion:          requisitosEjecucionFijos(),
		Componentes:                  componentes,
	}
	if p.datos.tieneCAS {
		cas := materialVinculo(p.datos.cas)
		material.CASEsperado = &cas
	}
	if p.datos.tieneVinculoEvidencia {
		vinculoEvidencia, _ := p.datos.vinculoEvidencia.Referencia()
		valor := materialReferencia(vinculoEvidencia)
		material.VinculoEvidencia = &valor
	}
	return json.Marshal(material)
}

func canonizarVersionResultado(
	version reglas.VersionGobernadaReglasBaremo,
) ([]byte, string, reglas.VinculoEstadoReglasBaremo, error) {
	canonico, err := version.RepresentacionCanonica()
	if err != nil {
		return nil, "", reglas.VinculoEstadoReglasBaremo{}, err
	}
	huella, err := version.HuellaSHA256()
	if err != nil {
		return nil, "", reglas.VinculoEstadoReglasBaremo{}, err
	}
	restaurada, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(
		canonico,
		huella,
	)
	if err != nil {
		return nil, "", reglas.VinculoEstadoReglasBaremo{}, err
	}
	vinculo, err := restaurada.VinculoEstado()
	return canonico, huella, vinculo, err
}

func materialReferencia(referencia reglas.ReferenciaVersionada) materialReferenciaV2 {
	return materialReferenciaV2{
		Referencia:   referencia.Referencia(),
		Version:      referencia.Version(),
		HuellaSHA256: referencia.HuellaSHA256(),
	}
}

func materialVinculo(vinculo reglas.VinculoEstadoReglasBaremo) materialVinculoV2 {
	return materialVinculoV2{
		Contenido:          materialReferencia(vinculo.Contenido()),
		Revision:           vinculo.Revision(),
		HuellaEstadoSHA256: vinculo.HuellaEstadoSHA256(),
	}
}

func (v VinculoEvidenciaTransicionReglasBaremoV2) incorporadaEn(
	version reglas.VersionGobernadaReglasBaremo,
) bool {
	if v.validar() != nil {
		return false
	}
	switch v.operacion {
	case OperacionPublicar:
		return version.IncorporaAprobacionExacta(v.aprobacion)
	case OperacionActivar:
		return version.IncorporaDependenciasExactas(v.dependencias)
	case OperacionSustituir, OperacionRetirar, OperacionDescartar:
		return version.IncorporaAutoridadExacta(v.autoridad)
	default:
		return false
	}
}

func motivoDominioCoincideConVEC(
	motivo reglas.MotivoCatalogadoReglasBaremo,
	referencia dominiovec.ReferenciaEntradaCatalogo,
) bool {
	if !dominiovec.ReferenciaMotivoAutorizacionV2Valida(referencia) {
		return false
	}
	catalogo := motivo.Catalogo()
	return referencia.CatalogoVersion > 0 &&
		catalogo.Referencia() == referencia.CatalogoID &&
		catalogo.Version() == uint64(referencia.CatalogoVersion) &&
		catalogo.HuellaSHA256() == referencia.CatalogoHuellaSHA256 &&
		motivo.Clave() == referencia.EntradaClave
}

func requisitosEjecucionFijos() []string {
	return []string{
		"alcance_resuelto_servidor",
		"cotejo_evidencia_verificador_confiable",
		"decision_vec_v2_consumible",
		"consumo_atestado_vec_ad2_mismo_commit",
		"commit_serializable_atomico",
		"reloj_autoritativo_frescura_cotejada",
		"recibo_durable_reconciliable",
	}
}

func referenciaTecnicaV2Valida(
	referencia reglas.ReferenciaVersionada,
	prefijo string,
) bool {
	if referencia.Version() != versionReferenciaV2 ||
		!strings.HasPrefix(referencia.Referencia(), prefijo) ||
		!huellaSHA256Valida(referencia.HuellaSHA256()) {
		return false
	}
	sufijo := strings.TrimPrefix(referencia.Referencia(), prefijo)
	return sufijo == referencia.HuellaSHA256()
}

func vinculoEstadoValido(vinculo reglas.VinculoEstadoReglasBaremo) bool {
	contenido := vinculo.Contenido()
	if contenido.Referencia() == "" || contenido.Version() == 0 ||
		!huellaSHA256Valida(contenido.HuellaSHA256()) ||
		vinculo.Revision() == 0 || !huellaSHA256Valida(vinculo.HuellaEstadoSHA256()) {
		return false
	}
	reconstruido, err := reglas.NuevoVinculoEstadoReglasBaremo(
		contenido,
		vinculo.Revision(),
		vinculo.HuellaEstadoSHA256(),
	)
	return err == nil && vinculosIguales(vinculo, reconstruido)
}

func referenciasIguales(a, b reglas.ReferenciaVersionada) bool {
	return a.Referencia() == b.Referencia() && a.Version() == b.Version() &&
		a.HuellaSHA256() == b.HuellaSHA256()
}

func vinculosIguales(a, b reglas.VinculoEstadoReglasBaremo) bool {
	return referenciasIguales(a.Contenido(), b.Contenido()) &&
		a.Revision() == b.Revision() &&
		a.HuellaEstadoSHA256() == b.HuellaEstadoSHA256()
}

func componentesExactos(componentes []ComponenteEscrituraReglasBaremoV2) bool {
	esperados := componentesEscrituraFijos()
	if len(componentes) != len(esperados) {
		return false
	}
	for indice := range esperados {
		if componentes[indice] != esperados[indice] {
			return false
		}
	}
	return true
}

func instanteUTCMicrosegundos(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func huellaSHA256Valida(valor string) bool {
	if len(valor) != sha256.Size*2 {
		return false
	}
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}
