package gobiernoreglasbaremo

import (
	"errors"
	"strings"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	moduloBolsaGobiernoReglas   = "bolsa"
	tipoRecursoReglasGobernadas = "version_reglas_baremo_gobernada"
	finalidadGobiernoReglas     = "gobierno_reglas_baremo"
	finalidadConsultaReglas     = "consulta_gobierno_reglas_baremo"
	perfilProteccionReglas      = "interno_alto"
	ambitoConvocatoriaRef       = "convocatoria_ref"
	ambitoExpedienteRef         = "expediente_ref"
)

var camposGobiernoReglas = []string{
	"auditoria",
	"estado_reglas_baremo",
	"salida_eventos",
}

var camposConsultaReglas = []string{"estado_reglas_baremo"}

type especificacionAutorizacion struct {
	accion    string
	finalidad string
	campos    []string
}

func especificacionPara(
	operacion OperacionGobiernoReglasBaremoV2,
) (especificacionAutorizacion, error) {
	var accion string
	switch operacion {
	case OperacionAltaBorrador:
		accion = "bolsa.reglas_baremo.borrador.crear"
	case OperacionPublicar:
		accion = "bolsa.reglas_baremo.publicar"
	case OperacionActivar:
		accion = "bolsa.reglas_baremo.activar"
	case OperacionSustituir:
		accion = "bolsa.reglas_baremo.sustituir"
	case OperacionRetirar:
		accion = "bolsa.reglas_baremo.retirar"
	case OperacionDescartar:
		accion = "bolsa.reglas_baremo.descartar"
	case OperacionConsultaExacta:
		return especificacionAutorizacion{
			accion:    "bolsa.reglas_baremo.version.consultar",
			finalidad: finalidadConsultaReglas,
			campos:    append([]string(nil), camposConsultaReglas...),
		}, nil
	default:
		return especificacionAutorizacion{}, ErrOperacionInvalida
	}
	return especificacionAutorizacion{
		accion: accion, finalidad: finalidadGobiernoReglas,
		campos: append([]string(nil), camposGobiernoReglas...),
	}, nil
}

type datosContratoAutorizacionV2 struct {
	operacion OperacionGobiernoReglasBaremoV2
	recurso   dominiovec.RecursoAutorizable
}

// alcanceAutorizacionReglasBaremoV2 solo puede derivarse de una version o una
// identidad de dominio validas. Impide elegir por texto libre los ambitos que
// reducen la concesion VEC.
type alcanceAutorizacionReglasBaremoV2 struct {
	convocatoriaRef string
	expedienteRef   string
}

func nuevoAlcanceAutorizacionDesdeVersion(
	version reglas.VersionGobernadaReglasBaremo,
) (alcanceAutorizacionReglasBaremoV2, error) {
	conjunto, err := version.Conjunto()
	if err != nil {
		return alcanceAutorizacionReglasBaremoV2{}, ErrContratoAutorizacionInvalido
	}
	return nuevoAlcanceAutorizacionDesdeIdentidad(conjunto.Identidad())
}

func nuevoAlcanceAutorizacionDesdeIdentidad(
	identidad reglas.IdentidadConjuntoReglasBaremo,
) (alcanceAutorizacionReglasBaremoV2, error) {
	validada, err := reglas.NuevaIdentidadConjuntoReglasBaremo(
		identidad.Referencia(), identidad.Version(),
		identidad.ConvocatoriaRef(), identidad.ExpedienteRef(),
	)
	if err != nil || validada.Referencia() != identidad.Referencia() ||
		validada.Version() != identidad.Version() ||
		validada.ConvocatoriaRef() != identidad.ConvocatoriaRef() ||
		validada.ExpedienteRef() != identidad.ExpedienteRef() {
		return alcanceAutorizacionReglasBaremoV2{}, ErrContratoAutorizacionInvalido
	}
	alcance := alcanceAutorizacionReglasBaremoV2{
		convocatoriaRef: identidad.ConvocatoriaRef(),
		expedienteRef:   identidad.ExpedienteRef(),
	}
	if alcance.validar() != nil {
		return alcanceAutorizacionReglasBaremoV2{}, ErrContratoAutorizacionInvalido
	}
	return alcance, nil
}

func (a alcanceAutorizacionReglasBaremoV2) validar() error {
	recurso := dominiovec.RecursoAutorizable{
		Referencia: "alcance:reglas-baremo",
		ModuloID:   moduloBolsaGobiernoReglas,
		Tipo:       tipoRecursoReglasGobernadas,
		Ambitos: map[string]string{
			ambitoConvocatoriaRef: a.convocatoriaRef,
			ambitoExpedienteRef:   a.expedienteRef,
		},
		Atributos: map[string]string{},
	}
	return recurso.Validar()
}

// ContratoAutorizacionV2 contiene exclusivamente el recurso y la politica
// cerrados que el futuro servicio pasara a la fachada VEC ligada a solicitud.
// No expone acciones o finalidades como parametros de texto.
type ContratoAutorizacionV2 struct {
	bloqueoSerializacion
	datos *datosContratoAutorizacionV2
}

func nuevoContratoAutorizacionV2(
	operacion OperacionGobiernoReglasBaremoV2,
	huellaEstadoSHA256 string,
	alcance alcanceAutorizacionReglasBaremoV2,
) (ContratoAutorizacionV2, error) {
	especificacion, err := especificacionPara(operacion)
	if err != nil || !huellaSHA256Valida(huellaEstadoSHA256) || alcance.validar() != nil {
		return ContratoAutorizacionV2{}, errors.Join(ErrContratoAutorizacionInvalido, err)
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: "reglas-baremo:" + huellaEstadoSHA256,
		ModuloID:   moduloBolsaGobiernoReglas,
		Tipo:       tipoRecursoReglasGobernadas,
		Ambitos: map[string]string{
			ambitoConvocatoriaRef: alcance.convocatoriaRef,
			ambitoExpedienteRef:   alcance.expedienteRef,
		},
		Atributos: map[string]string{},
	}
	if recurso.Validar() != nil {
		return ContratoAutorizacionV2{}, ErrContratoAutorizacionInvalido
	}
	_, err = aplicacionvec.NuevaPoliticaUsoDecisionAutorizacion(
		especificacion.accion,
		moduloBolsaGobiernoReglas,
		tipoRecursoReglasGobernadas,
		especificacion.finalidad,
		especificacion.campos,
		aplicacionvec.PerfilProteccionUsoAutorizacionInternoAlto,
	)
	if err != nil {
		return ContratoAutorizacionV2{}, errors.Join(ErrContratoAutorizacionInvalido, err)
	}
	return ContratoAutorizacionV2{datos: &datosContratoAutorizacionV2{
		operacion: operacion,
		recurso:   recurso,
	}}, nil
}

// Recurso devuelve una copia sin mapas compartidos. Sus dos ambitos proceden
// de la identidad tipada del plan o descriptor. En consultas, el adaptador
// debe cotejarlos con la fila antes de devolver datos.
func (c ContratoAutorizacionV2) Recurso() (dominiovec.RecursoAutorizable, error) {
	if c.validar() != nil {
		return dominiovec.RecursoAutorizable{}, ErrContratoAutorizacionInvalido
	}
	return clonarRecurso(c.datos.recurso), nil
}

// Politica devuelve la politica opaca e inmutable de la fachada VEC.
func (c ContratoAutorizacionV2) Politica() (
	aplicacionvec.PoliticaUsoDecisionAutorizacion,
	error,
) {
	if c.validar() != nil {
		return aplicacionvec.PoliticaUsoDecisionAutorizacion{}, ErrContratoAutorizacionInvalido
	}
	especificacion, err := especificacionPara(c.datos.operacion)
	if err != nil {
		return aplicacionvec.PoliticaUsoDecisionAutorizacion{}, ErrContratoAutorizacionInvalido
	}
	politica, err := aplicacionvec.NuevaPoliticaUsoDecisionAutorizacion(
		especificacion.accion,
		moduloBolsaGobiernoReglas,
		tipoRecursoReglasGobernadas,
		especificacion.finalidad,
		especificacion.campos,
		aplicacionvec.PerfilProteccionUsoAutorizacionInternoAlto,
	)
	if err != nil {
		return aplicacionvec.PoliticaUsoDecisionAutorizacion{}, ErrContratoAutorizacionInvalido
	}
	return politica, nil
}

func (c ContratoAutorizacionV2) validar() error {
	if c.datos == nil || !c.datos.operacion.valida() || c.datos.recurso.Validar() != nil ||
		len(c.datos.recurso.Ambitos) != 2 || len(c.datos.recurso.Atributos) != 0 ||
		c.datos.recurso.Ambitos[ambitoConvocatoriaRef] == "" ||
		c.datos.recurso.Ambitos[ambitoExpedienteRef] == "" {
		return ErrContratoAutorizacionInvalido
	}
	especificacion, err := especificacionPara(c.datos.operacion)
	if err != nil || c.datos.recurso.ModuloID != moduloBolsaGobiernoReglas ||
		c.datos.recurso.Tipo != tipoRecursoReglasGobernadas ||
		!strings.HasPrefix(c.datos.recurso.Referencia, "reglas-baremo:") ||
		!huellaSHA256Valida(c.datos.recurso.Referencia[len("reglas-baremo:"):]) {
		return ErrContratoAutorizacionInvalido
	}
	// La politica es opaca por diseño. Se reconstruye para comprobar que la
	// especificacion cerrada sigue siendo aceptada por el contrato VEC actual.
	_, err = aplicacionvec.NuevaPoliticaUsoDecisionAutorizacion(
		especificacion.accion,
		moduloBolsaGobiernoReglas,
		tipoRecursoReglasGobernadas,
		especificacion.finalidad,
		especificacion.campos,
		aplicacionvec.PerfilProteccionUsoAutorizacionInternoAlto,
	)
	return err
}

func clonarRecurso(origen dominiovec.RecursoAutorizable) dominiovec.RecursoAutorizable {
	resultado := origen
	resultado.Ambitos = make(map[string]string, len(origen.Ambitos))
	for clave, valor := range origen.Ambitos {
		resultado.Ambitos[clave] = valor
	}
	resultado.Atributos = make(map[string]string, len(origen.Atributos))
	for clave, valor := range origen.Atributos {
		resultado.Atributos[clave] = valor
	}
	return resultado
}

// DescriptorConsultaExactaReglasBaremoV2 declara el vinculo historico y el
// alcance minimo solicitado antes de leer la fila. No demuestra existencia ni
// concede acceso: el adaptador durable debe cotejar contenido, revision,
// huella e identidad contra una misma fila antes de devolverla.
type DescriptorConsultaExactaReglasBaremoV2 struct {
	bloqueoSerializacion
	vinculo   *reglas.VinculoEstadoReglasBaremo
	identidad *reglas.IdentidadConjuntoReglasBaremo
}

// nuevoDescriptorConsultaExactaReglasBaremoV2 queda deliberadamente dentro de
// application. El futuro caso de uso solo lo sellara despues de resolver en el
// servidor un indice o recibo durable; nunca desde campos enviados por cliente.
func nuevoDescriptorConsultaExactaReglasBaremoV2(
	vinculo reglas.VinculoEstadoReglasBaremo,
	identidad reglas.IdentidadConjuntoReglasBaremo,
) (DescriptorConsultaExactaReglasBaremoV2, error) {
	alcance, err := nuevoAlcanceAutorizacionDesdeIdentidad(identidad)
	contenido := vinculo.Contenido()
	if err != nil || !vinculoEstadoValido(vinculo) || alcance.validar() != nil ||
		contenido.Referencia() != identidad.Referencia() ||
		contenido.Version() != identidad.Version() {
		return DescriptorConsultaExactaReglasBaremoV2{}, ErrConsultaExactaInvalida
	}
	copiaVinculo, copiaIdentidad := vinculo, identidad
	return DescriptorConsultaExactaReglasBaremoV2{
		vinculo: &copiaVinculo, identidad: &copiaIdentidad,
	}, nil
}

func (d DescriptorConsultaExactaReglasBaremoV2) validar() error {
	if d.vinculo == nil || d.identidad == nil {
		return ErrConsultaExactaInvalida
	}
	alcance, err := nuevoAlcanceAutorizacionDesdeIdentidad(*d.identidad)
	contenido := d.vinculo.Contenido()
	if err != nil || !vinculoEstadoValido(*d.vinculo) || alcance.validar() != nil ||
		contenido.Referencia() != d.identidad.Referencia() ||
		contenido.Version() != d.identidad.Version() {
		return ErrConsultaExactaInvalida
	}
	return nil
}

// ConsultaExactaReglasBaremoV2 impide sustituir una seleccion historica por
// "la vigente" sin exigir haber leido antes la version completa.
type ConsultaExactaReglasBaremoV2 struct {
	bloqueoSerializacion
	descriptor *DescriptorConsultaExactaReglasBaremoV2
}

// nuevaConsultaExactaReglasBaremoV2 no es una fabrica de frontera. El adaptador
// recibe la consulta ya sellada y solo puede leer Vinculo e Identidad.
func nuevaConsultaExactaReglasBaremoV2(
	descriptor DescriptorConsultaExactaReglasBaremoV2,
) (ConsultaExactaReglasBaremoV2, error) {
	if descriptor.validar() != nil {
		return ConsultaExactaReglasBaremoV2{}, ErrConsultaExactaInvalida
	}
	copiaVinculo, copiaIdentidad := *descriptor.vinculo, *descriptor.identidad
	copia := DescriptorConsultaExactaReglasBaremoV2{
		vinculo: &copiaVinculo, identidad: &copiaIdentidad,
	}
	return ConsultaExactaReglasBaremoV2{descriptor: &copia}, nil
}

func (c ConsultaExactaReglasBaremoV2) Vinculo() (
	reglas.VinculoEstadoReglasBaremo,
	error,
) {
	if c.descriptor == nil || c.descriptor.validar() != nil {
		return reglas.VinculoEstadoReglasBaremo{}, ErrConsultaExactaInvalida
	}
	return *c.descriptor.vinculo, nil
}

// Identidad devuelve la identidad opaca que el adaptador debe cotejar en la
// misma fila que Vinculo. Se devuelve por valor y no comparte estado mutable.
func (c ConsultaExactaReglasBaremoV2) Identidad() (
	reglas.IdentidadConjuntoReglasBaremo,
	error,
) {
	if c.descriptor == nil || c.descriptor.validar() != nil {
		return reglas.IdentidadConjuntoReglasBaremo{}, ErrConsultaExactaInvalida
	}
	return *c.descriptor.identidad, nil
}

func (c ConsultaExactaReglasBaremoV2) ContratoAutorizacionV2() (
	ContratoAutorizacionV2,
	error,
) {
	vinculo, err := c.Vinculo()
	if err != nil {
		return ContratoAutorizacionV2{}, err
	}
	alcance, err := nuevoAlcanceAutorizacionDesdeIdentidad(*c.descriptor.identidad)
	if err != nil {
		return ContratoAutorizacionV2{}, ErrConsultaExactaInvalida
	}
	return nuevoContratoAutorizacionV2(
		OperacionConsultaExacta,
		vinculo.HuellaEstadoSHA256(),
		alcance,
	)
}
