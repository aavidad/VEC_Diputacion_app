package ports

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultarVersionConvocatoria         = "bolsa.convocatoria.version.consultar"
	AccionConsultarVersionConFlujoConvocatoria = "bolsa.convocatoria.version_con_flujo.consultar"
	AccionCrearBorradorConvocatoria            = "bolsa.convocatoria.borrador.crear"
	AccionActualizarBorradorConvocatoria       = "bolsa.convocatoria.borrador.actualizar"
	AccionPublicarVersionConvocatoria          = "bolsa.convocatoria.version.publicar"
	AccionPublicarYSustituirConvocatoria       = "bolsa.convocatoria.version.publicar_y_sustituir"
	AccionPublicarTrasRetiradaConvocatoria     = "bolsa.convocatoria.version.publicar_tras_retirada"
	AccionRetirarVersionConvocatoria           = "bolsa.convocatoria.version.retirar"
	ModuloGobiernoConvocatorias                = "bolsa"
	TipoRecursoVersionConvocatoriaGobernada    = "version_convocatoria_gobernada"
	FinalidadConsultaInternaConvocatorias      = "consulta_interna_convocatorias"
	FinalidadGobiernoConvocatorias             = "gobierno_convocatorias"
	AtributoHuellaIntencionConvocatoria        = "huella_intencion_sha256"
	VentanaMaximaUsoAutorizacionConvocatoria   = 30 * time.Second
)

var (
	ErrAutorizacionGobiernoConvocatoriaInvalida = errors.New("bolsa: autorizacion de gobierno de convocatoria invalida")
	ErrConsultaGobiernoConvocatoriaInvalida     = errors.New("bolsa: consulta interna de convocatoria invalida")
	ErrVersionGobernadaConvocatoriaNoEncontrada = errors.New("bolsa: version gobernada de convocatoria no encontrada")
)

type especificacionAutorizacionConvocatoria struct {
	finalidad string
	campos    []string
	mutacion  bool
}

var especificacionesAutorizacionConvocatoria = map[string]especificacionAutorizacionConvocatoria{
	AccionConsultarVersionConvocatoria: {
		finalidad: FinalidadConsultaInternaConvocatorias,
		campos:    []string{"version_convocatoria"},
	},
	AccionConsultarVersionConFlujoConvocatoria: {
		finalidad: FinalidadConsultaInternaConvocatorias,
		campos:    []string{"instancia_flujo", "version_convocatoria"},
	},
	AccionCrearBorradorConvocatoria: {
		finalidad: FinalidadGobiernoConvocatorias, mutacion: true,
		campos: []string{"auditoria", "evento_outbox", "version_convocatoria"},
	},
	AccionActualizarBorradorConvocatoria: {
		finalidad: FinalidadGobiernoConvocatorias, mutacion: true,
		campos: []string{"auditoria", "evento_outbox", "version_convocatoria"},
	},
	AccionPublicarVersionConvocatoria: {
		finalidad: FinalidadGobiernoConvocatorias, mutacion: true,
		campos: []string{"auditoria", "evento_outbox", "version_convocatoria"},
	},
	AccionPublicarYSustituirConvocatoria: {
		finalidad: FinalidadGobiernoConvocatorias, mutacion: true,
		campos: []string{"auditoria", "evento_outbox", "version_publicada", "version_sustituida"},
	},
	AccionPublicarTrasRetiradaConvocatoria: {
		finalidad: FinalidadGobiernoConvocatorias, mutacion: true,
		campos: []string{"auditoria", "evento_outbox", "version_predecesora", "version_publicada"},
	},
	AccionRetirarVersionConvocatoria: {
		finalidad: FinalidadGobiernoConvocatorias, mutacion: true,
		campos: []string{"auditoria", "evento_outbox", "version_convocatoria"},
	},
}

// SelectorVersionConvocatoriaExacta impide consultas ambiguas o a «la ultima».
type SelectorVersionConvocatoriaExacta struct {
	ID        string
	Secuencia int
}

func (s SelectorVersionConvocatoriaExacta) Referencia() string {
	return s.ID + "#" + numeroDecimalConvocatoria(s.Secuencia)
}

func (s SelectorVersionConvocatoriaExacta) Validar() error {
	if !referenciaGobiernoConvocatoriaValida(s.ID) || strings.ContainsRune(s.ID, '#') ||
		s.Secuencia < 1 || !referenciaVersionGobernadaConvocatoriaValida(s.Referencia()) {
		return ErrConsultaGobiernoConvocatoriaInvalida
	}
	return nil
}

// RecursoAutorizableConsultaVersionConvocatoria construye el mismo recurso
// exacto que debe evaluar el PDP. La consulta no admite ambitos ni atributos
// declarados por el cliente.
func RecursoAutorizableConsultaVersionConvocatoria(
	selector SelectorVersionConvocatoriaExacta,
) (dominiovec.RecursoAutorizable, error) {
	if selector.Validar() != nil {
		return dominiovec.RecursoAutorizable{}, ErrConsultaGobiernoConvocatoriaInvalida
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: selector.Referencia(), ModuloID: ModuloGobiernoConvocatorias,
		Tipo: TipoRecursoVersionConvocatoriaGobernada,
	}
	if recurso.Validar() != nil {
		return dominiovec.RecursoAutorizable{}, ErrConsultaGobiernoConvocatoriaInvalida
	}
	return recurso, nil
}

// RecursoAutorizableMutacionConvocatoria liga la concesion a la preimagen
// semantica completa; una concesion no puede aplicarse a otra mutacion.
func RecursoAutorizableMutacionConvocatoria(
	material MaterialIntencionGobiernoConvocatoria,
) (dominiovec.RecursoAutorizable, error) {
	if material.Validar() != nil {
		return dominiovec.RecursoAutorizable{}, ErrAutorizacionGobiernoConvocatoriaInvalida
	}
	huella, err := material.HuellaSHA256()
	if err != nil {
		return dominiovec.RecursoAutorizable{}, ErrAutorizacionGobiernoConvocatoriaInvalida
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: material.EstadoPrincipalNuevo.Referencia,
		ModuloID:   ModuloGobiernoConvocatorias,
		Tipo:       TipoRecursoVersionConvocatoriaGobernada,
		Atributos:  map[string]string{AtributoHuellaIntencionConvocatoria: huella},
	}
	if recurso.Validar() != nil {
		return dominiovec.RecursoAutorizable{}, ErrAutorizacionGobiernoConvocatoriaInvalida
	}
	return recurso, nil
}

type SolicitudConsultaVersionConvocatoriaAutorizada struct {
	bloqueoSerializacionGobiernoConvocatoria
	Selector              SelectorVersionConvocatoriaExacta
	IncluirInstanciaFlujo bool
	Autorizacion          puertosvec.EvidenciaUsoDecisionAutorizacion
	ConsultadaEn          time.Time
}

func (s SolicitudConsultaVersionConvocatoriaAutorizada) Validar() error {
	recurso, err := RecursoAutorizableConsultaVersionConvocatoria(s.Selector)
	accion := s.accionAutorizacion()
	if err != nil || validarUsoAutorizacionConvocatoria(
		s.Autorizacion, accion, recurso, s.ConsultadaEn,
	) != nil {
		return ErrConsultaGobiernoConvocatoriaInvalida
	}
	return nil
}

func (s SolicitudConsultaVersionConvocatoriaAutorizada) accionAutorizacion() string {
	if s.IncluirInstanciaFlujo {
		return AccionConsultarVersionConFlujoConvocatoria
	}
	return AccionConsultarVersionConvocatoria
}

type ResultadoConsultaVersionConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	Version                            dominiobolsa.VersionConvocatoriaGobernada
	InstanciaFlujo                     *dominiovec.InstanciaFlujo
	HuellaVersionSHA256                string
	HuellaInstanciaFlujoSHA256         string
	AutorizacionRef                    string
	HuellaAutorizacionSHA256           string
	AtestacionAutorizacionRef          string
	HuellaAtestacionAutorizacionSHA256 string
	ConsumoAutorizacionRef             string
	AuditoriaRef                       string
	HuellaAuditoriaSHA256              string
	ConsultadaEn                       time.Time
}

func (r ResultadoConsultaVersionConvocatoria) ValidarPara(
	s SolicitudConsultaVersionConvocatoriaAutorizada,
) error {
	datos, err := s.Autorizacion.Datos()
	recurso, errRecurso := RecursoAutorizableConsultaVersionConvocatoria(s.Selector)
	huellaVersion, errHuellaVersion := r.Version.HuellaSHA256()
	if s.Validar() != nil || err != nil || errRecurso != nil ||
		errHuellaVersion != nil ||
		validarUsoAutorizacionConvocatoria(
			s.Autorizacion, s.accionAutorizacion(), recurso, r.ConsultadaEn,
		) != nil || r.Version.Validar() != nil ||
		r.Version.ID != s.Selector.ID || r.Version.Secuencia != s.Selector.Secuencia ||
		r.HuellaVersionSHA256 != huellaVersion ||
		!huellaInstanciaConsultaConvocatoriaValida(r.InstanciaFlujo, r.HuellaInstanciaFlujoSHA256) ||
		!presenciaInstanciaFlujoConvocatoriaValida(
			r.Version, r.InstanciaFlujo, s.IncluirInstanciaFlujo,
		) ||
		r.AutorizacionRef != datos.Decision.DecisionRef ||
		r.HuellaAutorizacionSHA256 != datos.HuellaDecisionSHA256 ||
		!referenciaGobiernoConvocatoriaValida(r.AtestacionAutorizacionRef) ||
		!huellaGobiernoConvocatoriaValida(r.HuellaAtestacionAutorizacionSHA256) ||
		!referenciaGobiernoConvocatoriaValida(r.ConsumoAutorizacionRef) ||
		!referenciaGobiernoConvocatoriaValida(r.AuditoriaRef) ||
		!huellaGobiernoConvocatoriaValida(r.HuellaAuditoriaSHA256) ||
		!referenciasGobiernoConvocatoriaDistintas(
			r.AutorizacionRef, r.AtestacionAutorizacionRef,
			r.ConsumoAutorizacionRef, r.AuditoriaRef,
		) ||
		!instanteGobiernoConvocatoriaCanonico(r.ConsultadaEn) ||
		r.ConsultadaEn.Before(s.ConsultadaEn) {
		return ErrConsultaGobiernoConvocatoriaInvalida
	}
	return nil
}

func huellaInstanciaConsultaConvocatoriaValida(
	instancia *dominiovec.InstanciaFlujo,
	huellaEsperada string,
) bool {
	if instancia == nil {
		return huellaEsperada == ""
	}
	huella, err := instancia.HuellaSHA256()
	return err == nil && huellaEsperada == huella
}

func (r ResultadoConsultaVersionConvocatoria) Clonar() (
	ResultadoConsultaVersionConvocatoria,
	error,
) {
	version, err := r.Version.ClonarCanonico()
	if err != nil {
		return ResultadoConsultaVersionConvocatoria{}, ErrConsultaGobiernoConvocatoriaInvalida
	}
	r.Version = version
	if r.InstanciaFlujo != nil {
		instancia := *r.InstanciaFlujo
		r.InstanciaFlujo = &instancia
	}
	return r, nil
}

// ConsultaGobiernoConvocatorias debe leer exactamente una version y registrar
// la lectura junto con el uso de autorizacion en la misma transaccion. La
// preimagen auditada incluye HuellaVersionSHA256 y, cuando exista,
// HuellaInstanciaFlujoSHA256; la huella del registro no es decorativa.
type ConsultaGobiernoConvocatorias interface {
	ObtenerVersionExacta(
		context.Context,
		SolicitudConsultaVersionConvocatoriaAutorizada,
	) (ResultadoConsultaVersionConvocatoria, error)
}

func validarUsoAutorizacionConvocatoria(
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacion,
	accion string,
	recurso dominiovec.RecursoAutorizable,
	instante time.Time,
) error {
	especificacion, existe := especificacionesAutorizacionConvocatoria[accion]
	datos, errDatos := evidencia.Datos()
	huellaContexto, errHuella := recurso.HuellaContextoAutorizacionSHA256()
	decision := datos.Decision
	datosVinculo, errVinculo := decision.VinculoAutenticacionActor.Datos()
	superficieInterna := datosVinculo.Superficie == dominiovec.SuperficieAutenticacionInternaCorporativaV1 &&
		!datosVinculo.CuentaPrivilegiada
	superficiePrivilegiada := datosVinculo.Superficie == dominiovec.SuperficieAutenticacionAdministracionPrivilegiadaV1 &&
		datosVinculo.CuentaPrivilegiada
	if !existe || !instanteGobiernoConvocatoriaCanonico(instante) || recurso.Validar() != nil ||
		errDatos != nil || errHuella != nil || errVinculo != nil || evidencia.ValidarEn(instante) != nil ||
		decision.Accion != accion || decision.RecursoRef != recurso.Referencia ||
		decision.ModuloID != ModuloGobiernoConvocatorias ||
		decision.TipoRecurso != TipoRecursoVersionConvocatoriaGobernada ||
		decision.ContextoRecursoHuellaSHA256 != huellaContexto ||
		decision.Finalidad != especificacion.finalidad ||
		!mismosCamposConvocatoria(decision.CamposPermitidos, especificacion.campos) ||
		len(decision.Obligaciones) != 0 || decision.GarantiaMinima != dominiovec.AuthAssuranceHigh ||
		datosVinculo.GarantiaObservada != dominiovec.AuthAssuranceHigh ||
		datosVinculo.MetodoObservado == dominiovec.AuthMethodDemo ||
		(!superficieInterna && !superficiePrivilegiada) ||
		instante.Sub(datos.VerificadaEn) > VentanaMaximaUsoAutorizacionConvocatoria {
		return ErrAutorizacionGobiernoConvocatoriaInvalida
	}
	if especificacion.mutacion {
		if len(recurso.Ambitos) != 0 || len(recurso.Atributos) != 1 ||
			!huellaGobiernoConvocatoriaValida(recurso.Atributos[AtributoHuellaIntencionConvocatoria]) {
			return ErrAutorizacionGobiernoConvocatoriaInvalida
		}
	} else if len(recurso.Ambitos) != 0 || len(recurso.Atributos) != 0 {
		return ErrAutorizacionGobiernoConvocatoriaInvalida
	}
	return nil
}

func mismosCamposConvocatoria(recibidos, esperados []string) bool {
	if len(recibidos) != len(esperados) || len(recibidos) == 0 {
		return false
	}
	a := append([]string(nil), recibidos...)
	b := append([]string(nil), esperados...)
	sort.Strings(a)
	sort.Strings(b)
	return strings.Join(a, "\x00") == strings.Join(b, "\x00")
}

func instanciaFlujoConvocatoriaExacta(
	version dominiobolsa.VersionConvocatoriaGobernada,
	instancia dominiovec.InstanciaFlujo,
) bool {
	huella, err := instancia.HuellaSHA256()
	_ = huella
	configuracion := version.Configuracion.FlujoProceso
	return err == nil && instancia.TipoEntidad == dominiobolsa.TipoEntidadFlujoConvocatoriaBolsa &&
		instancia.ID == version.InstanciaFlujoRef && instancia.EntidadRef == version.ID &&
		instancia.DefinicionRef == configuracion.ID+":"+numeroDecimalConvocatoria(configuracion.Version) &&
		instancia.DefinicionContenidoHuellaSHA256 == configuracion.HuellaContenidoSHA256
}

func presenciaInstanciaFlujoConvocatoriaValida(
	version dominiobolsa.VersionConvocatoriaGobernada,
	instancia *dominiovec.InstanciaFlujo,
	incluir bool,
) bool {
	if version.EstadoGobierno == dominiobolsa.EstadoGobiernoConvocatoriaBorrador {
		return !incluir && instancia == nil
	}
	return incluir && instancia != nil && instanciaFlujoConvocatoriaExacta(version, *instancia)
}
