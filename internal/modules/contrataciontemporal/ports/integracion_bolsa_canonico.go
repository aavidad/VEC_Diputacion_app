package ports

import (
	"bytes"
	"strconv"
	"time"
)

const esquemaCanonicoIntegracionBolsa = "vec.contratacion-temporal.integracion-bolsa.v1"

type constructorCanonicoBolsa struct {
	contenido bytes.Buffer
}

func nuevoCanonicoBolsa(tipo string) *constructorCanonicoBolsa {
	constructor := &constructorCanonicoBolsa{}
	constructor.campo("esquema", esquemaCanonicoIntegracionBolsa)
	constructor.campo("tipo", tipo)
	return constructor
}

// campo usa longitud UTF-8 decimal + dos puntos + bytes. El nombre también se
// incluye, de modo que el formato sea inequívoco y reproducible fuera de Go.
func (c *constructorCanonicoBolsa) campo(nombre, valor string) {
	c.contenido.WriteString(strconv.Itoa(len(nombre)))
	c.contenido.WriteByte(':')
	c.contenido.WriteString(nombre)
	c.contenido.WriteString(strconv.Itoa(len(valor)))
	c.contenido.WriteByte(':')
	c.contenido.WriteString(valor)
}

func (c *constructorCanonicoBolsa) entero(nombre string, valor uint64) {
	c.campo(nombre, strconv.FormatUint(valor, 10))
}

func (c *constructorCanonicoBolsa) booleano(nombre string, valor bool) {
	c.campo(nombre, strconv.FormatBool(valor))
}

func (c *constructorCanonicoBolsa) instante(nombre string, valor time.Time) {
	c.campo(nombre, valor.Format(time.RFC3339Nano))
}

func (c *constructorCanonicoBolsa) referencia(
	prefijo string,
	valor ReferenciaVersionadaIntegracionBolsa,
) {
	c.campo(prefijo+"_ref", valor.Referencia)
	c.entero(prefijo+"_version", valor.Version)
	c.campo(prefijo+"_huella_sha256", valor.HuellaSHA256)
}

func (c *constructorCanonicoBolsa) contexto(
	valor DatosContextoPeticionIntegracionBolsa,
) {
	c.campo("operacion_ref", valor.OperacionRef)
	c.campo("organizacion_ref", valor.OrganizacionRef)
	c.campo("expediente_ref", valor.ExpedienteRef)
	c.entero("version_expediente", valor.VersionExpediente)
	c.campo("correlacion_ref", valor.CorrelacionRef)
	c.entero("contrato_version", valor.ContratoVersion)
	c.campo("autoridad_solicitante", valor.AutoridadSolicitante)
	c.referencia("autorizacion", valor.Autorizacion)
	c.referencia("accion", valor.Accion)
	c.referencia("recurso", valor.Recurso)
	c.referencia("finalidad", valor.Finalidad)
	c.instante("solicitada_en", valor.SolicitadaEn)
	c.instante("peticion_valida_hasta", valor.ValidaHasta)
}

func (c *constructorCanonicoBolsa) contextoCapacidad(
	valor ContextoPeticionIntegracionBolsa,
) {
	datos, err := valor.datosDurables()
	if err != nil {
		c.campo("contexto_invalido", "1")
		return
	}
	c.contexto(datos)
}

func (c *constructorCanonicoBolsa) procedencia(valor ProcedenciaIntegracionBolsa) {
	c.campo("autoridad_ref", valor.AutoridadRef)
	c.campo("respuesta_ref", valor.RespuestaRef)
	c.entero("procedencia_contrato_version", valor.ContratoVersion)
	c.referencia("fuente", valor.Fuente)
	c.campo("evidencia_ref", valor.Evidencia.EvidenciaRef)
	c.campo("clave_verificacion_ref", valor.Evidencia.ClaveVerificacionRef)
	c.instante("evidencia_emitida_en", valor.Evidencia.EmitidaEn)
	c.instante("evidencia_valida_hasta", valor.Evidencia.ValidaHasta)
	c.instante("evidencia_retener_hasta", valor.Evidencia.RetenerHasta)
}

func (c *constructorCanonicoBolsa) bytes() []byte {
	return append([]byte(nil), c.contenido.Bytes()...)
}
