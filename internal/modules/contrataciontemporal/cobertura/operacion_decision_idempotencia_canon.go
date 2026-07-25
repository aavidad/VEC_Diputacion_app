package cobertura

import (
	"bytes"
	"context"
	"encoding/binary"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	dominioAmbitoOperacionDecisionCobertura    = "vec.contratacion-temporal.cobertura-decision.ambito"
	dominioSemanticaOperacionDecisionCobertura = "vec.contratacion-temporal." +
		"cobertura-decision.semantica"
	esquemaAmbitoOperacionDecisionCobertura    = "VEC-CT-COBERTURA-DECISION-AMBITO-C3-V1"
	esquemaSemanticaOperacionDecisionCobertura = "VEC-CT-COBERTURA-DECISION-SEMANTICA-C3-V1"
	maximoBytesCanonOperacionDecisionCobertura = 64 * 1024
)

// DatosIdentidadOperacionDecisionCobertura es una identidad opaca: no es DTO
// de transporte ni proyección de persistencia. Sus referencias internas solo
// se usan para construir las preimágenes dentro del orquestador.
type DatosIdentidadOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	claveIdempotencia  string
	tipo               domain.TipoDecisionCoberturaGobernada
	organizacionRef    string
	expedienteRef      string
	versionExpediente  uint64
	actorRef           string
	perfilRef          string
	accion             domain.ClaveCatalogo
	viaElegida         domain.ClaveCatalogo
	identidadSemantica domain.IdentidadSemanticaPropuestaDecisionCobertura
	motivo             domain.MotivoGobernadoDecisionCobertura
	predecesoraRef     string
	predecesoraHuella  string
}

// NuevaIdentidadOperacionDecisionCobertura deriva actor y perfil únicamente
// de una capacidad nominal VEC vigente y ligada a su resultado autoritativo.
// Las entradas web, escritorio, CLI o MCP no pueden suministrarlos como texto.
func NuevaIdentidadOperacionDecisionCobertura(
	claveIdempotencia string,
	tipo domain.TipoDecisionCoberturaGobernada,
	organizacionRef string,
	expedienteRef string,
	versionExpediente uint64,
	contexto ports.ContextoAutorizacionAltaV3,
	solicitudContexto ports.SolicitudResolverContextoAutorizacionAltaV3,
	autenticadaEn time.Time,
	accion domain.ClaveCatalogo,
	viaElegida domain.ClaveCatalogo,
	identidadSemantica domain.IdentidadSemanticaPropuestaDecisionCobertura,
	motivo domain.MotivoGobernadoDecisionCobertura,
	predecesoraRef string,
	predecesoraHuella string,
) (DatosIdentidadOperacionDecisionCobertura, error) {
	if contexto.ValidarPara(solicitudContexto, autenticadaEn) != nil {
		return DatosIdentidadOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	datosVinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		return DatosIdentidadOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	datos := DatosIdentidadOperacionDecisionCobertura{
		claveIdempotencia: claveIdempotencia, tipo: tipo,
		organizacionRef: organizacionRef, expedienteRef: expedienteRef,
		versionExpediente: versionExpediente,
		actorRef:          datosVinculo.PrincipalID,
		perfilRef:         datosVinculo.PerfilActivoRef,
		accion:            accion, viaElegida: viaElegida,
		identidadSemantica: identidadSemantica, motivo: motivo,
		predecesoraRef:    predecesoraRef,
		predecesoraHuella: predecesoraHuella,
	}
	if datos.Validar() != nil {
		return DatosIdentidadOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return datos, nil
}

func (d DatosIdentidadOperacionDecisionCobertura) Validar() error {
	if !ports.ClaveIdempotenciaValida(d.claveIdempotencia) ||
		!tipoDecisionCoberturaValido(d.tipo) ||
		!domain.ReferenciaOpacaValida(d.organizacionRef) ||
		!domain.ReferenciaOpacaValida(d.expedienteRef) ||
		d.versionExpediente == 0 ||
		d.versionExpediente >= MaximoEnteroSeguroOperacionDecisionCobertura ||
		!domain.ReferenciaOpacaValida(d.actorRef) ||
		!domain.ReferenciaOpacaValida(d.perfilRef) ||
		!d.viaElegida.Valida() ||
		d.identidadSemantica.Validar() != nil ||
		d.accion != accionOperacionDecisionCobertura(d.tipo) {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	motivoVacio := d.motivo == (domain.MotivoGobernadoDecisionCobertura{})
	motivoValido := motivoOperacionDecisionCoberturaValido(d.motivo)
	switch d.tipo {
	case domain.DecisionCoberturaInicial:
		if d.predecesoraRef != "" || d.predecesoraHuella != "" ||
			(!motivoVacio && !motivoValido) {
			return ErrOperacionDecisionCoberturaIdempotenteInvalida
		}
	case domain.DecisionCoberturaRectificacion:
		if !motivoValido ||
			!domain.ReferenciaOpacaValida(d.predecesoraRef) ||
			!huellaSHA256OperacionDecisionCoberturaValida(
				d.predecesoraHuella,
			) {
			return ErrOperacionDecisionCoberturaIdempotenteInvalida
		}
	default:
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return nil
}

func tipoDecisionCoberturaValido(
	tipo domain.TipoDecisionCoberturaGobernada,
) bool {
	return tipo == domain.DecisionCoberturaInicial ||
		tipo == domain.DecisionCoberturaRectificacion
}

func accionOperacionDecisionCobertura(
	tipo domain.TipoDecisionCoberturaGobernada,
) domain.ClaveCatalogo {
	if tipo == domain.DecisionCoberturaInicial {
		return domain.AccionDecidirCoberturaGobernada
	}
	if tipo == domain.DecisionCoberturaRectificacion {
		return domain.AccionRectificarCoberturaGobernada
	}
	return ""
}

func motivoOperacionDecisionCoberturaValido(
	motivo domain.MotivoGobernadoDecisionCobertura,
) bool {
	return motivo.ReferenciaCatalogo.Validar() == nil &&
		motivo.ReferenciaCatalogo.CatalogoVersion > 0 &&
		uint64(motivo.ReferenciaCatalogo.CatalogoVersion) <=
			MaximoEnteroSeguroOperacionDecisionCobertura &&
		motivo.ClaveI18n.Valida()
}

// PreimagenesOperacionDecisionCobertura nunca entrega datos funcionales al
// registro. Solo el sellador HMAC recibe copias acotadas de los bytes.
type PreimagenesOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	ambito    []byte
	semantica []byte
}

func NuevasPreimagenesOperacionDecisionCobertura(
	datos DatosIdentidadOperacionDecisionCobertura,
) (PreimagenesOperacionDecisionCobertura, error) {
	if datos.Validar() != nil {
		return PreimagenesOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	ambito := nuevoCanonOperacionDecisionCobertura()
	ambito.texto(esquemaAmbitoOperacionDecisionCobertura)
	ambito.texto(datos.claveIdempotencia)
	ambito.texto(datos.organizacionRef)
	ambito.texto(datos.expedienteRef)
	ambito.texto(datos.actorRef)
	ambito.texto(datos.perfilRef)
	bytesAmbito, err := ambito.resultado()
	if err != nil {
		return PreimagenesOperacionDecisionCobertura{}, err
	}
	semantica := nuevoCanonOperacionDecisionCobertura()
	semantica.texto(esquemaSemanticaOperacionDecisionCobertura)
	semantica.texto(datos.claveIdempotencia)
	semantica.texto(string(datos.tipo))
	semantica.texto(datos.organizacionRef)
	semantica.texto(datos.expedienteRef)
	semantica.entero(datos.versionExpediente)
	semantica.texto(datos.actorRef)
	semantica.texto(datos.perfilRef)
	semantica.texto(string(datos.accion))
	semantica.texto(string(datos.viaElegida))
	semantica.texto(datos.identidadSemantica.Referencia)
	semantica.texto(datos.identidadSemantica.HuellaSHA256)
	escribirMotivoOperacionDecisionCobertura(semantica, datos.motivo)
	semantica.texto(datos.predecesoraRef)
	semantica.texto(datos.predecesoraHuella)
	bytesSemantica, err := semantica.resultado()
	if err != nil {
		return PreimagenesOperacionDecisionCobertura{}, err
	}
	return PreimagenesOperacionDecisionCobertura{
		ambito: bytesAmbito, semantica: bytesSemantica,
	}, nil
}

func (p PreimagenesOperacionDecisionCobertura) BytesAmbito() ([]byte, error) {
	if !preimagenOperacionDecisionCoberturaValida(p.ambito) {
		return nil, ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return append([]byte(nil), p.ambito...), nil
}

func (p PreimagenesOperacionDecisionCobertura) BytesSemantica() ([]byte, error) {
	if !preimagenOperacionDecisionCoberturaValida(p.semantica) {
		return nil, ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return append([]byte(nil), p.semantica...), nil
}

type SelladorOperacionDecisionCobertura interface {
	SellarOperacionDecisionCobertura(
		context.Context,
		PreimagenesOperacionDecisionCobertura,
	) (SellosOperacionDecisionCobertura, error)
}

type SellosOperacionDecisionCobertura struct {
	AmbitosIdempotenciaHMAC ports.ColeccionSellosHMAC
	HuellasSemanticasHMAC   ports.ColeccionSellosHMAC
}

func (s SellosOperacionDecisionCobertura) Validar() error {
	_, _, err := ports.ParActivoColeccionesHMAC(
		s.AmbitosIdempotenciaHMAC,
		dominioAmbitoOperacionDecisionCobertura,
		s.HuellasSemanticasHMAC,
		dominioSemanticaOperacionDecisionCobertura,
	)
	if err != nil {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return nil
}

func (s SellosOperacionDecisionCobertura) parActivo() (
	string,
	string,
	error,
) {
	ambito, semantica, err := ports.ParActivoColeccionesHMAC(
		s.AmbitosIdempotenciaHMAC,
		dominioAmbitoOperacionDecisionCobertura,
		s.HuellasSemanticasHMAC,
		dominioSemanticaOperacionDecisionCobertura,
	)
	if err != nil {
		return "", "", ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return ambito, semantica, nil
}

func (s SellosOperacionDecisionCobertura) contienePar(
	ambito string,
	semantica string,
) bool {
	return ports.ColeccionesHMACContienenPar(
		s.AmbitosIdempotenciaHMAC,
		dominioAmbitoOperacionDecisionCobertura,
		s.HuellasSemanticasHMAC,
		dominioSemanticaOperacionDecisionCobertura,
		ambito,
		semantica,
	)
}

type canonOperacionDecisionCobertura struct {
	buffer bytes.Buffer
	err    error
}

func nuevoCanonOperacionDecisionCobertura() *canonOperacionDecisionCobertura {
	return &canonOperacionDecisionCobertura{}
}

func (c *canonOperacionDecisionCobertura) texto(valor string) {
	if c.err != nil || len(valor) > maximoBytesCanonOperacionDecisionCobertura ||
		c.buffer.Len() > maximoBytesCanonOperacionDecisionCobertura-4-len(valor) {
		c.err = ErrOperacionDecisionCoberturaIdempotenteInvalida
		return
	}
	var longitud [4]byte
	binary.BigEndian.PutUint32(longitud[:], uint32(len(valor)))
	_, _ = c.buffer.Write(longitud[:])
	_, _ = c.buffer.WriteString(valor)
}

func (c *canonOperacionDecisionCobertura) entero(valor uint64) {
	if c.err != nil || valor > MaximoEnteroSeguroOperacionDecisionCobertura {
		c.err = ErrOperacionDecisionCoberturaIdempotenteInvalida
		return
	}
	var contenido [8]byte
	binary.BigEndian.PutUint64(contenido[:], valor)
	_, _ = c.buffer.Write(contenido[:])
}

func (c *canonOperacionDecisionCobertura) resultado() ([]byte, error) {
	if c.err != nil || c.buffer.Len() == 0 ||
		c.buffer.Len() > maximoBytesCanonOperacionDecisionCobertura {
		return nil, ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return append([]byte(nil), c.buffer.Bytes()...), nil
}

func escribirMotivoOperacionDecisionCobertura(
	canon *canonOperacionDecisionCobertura,
	motivo domain.MotivoGobernadoDecisionCobertura,
) {
	canon.texto(motivo.ReferenciaCatalogo.CatalogoID)
	canon.entero(uint64(motivo.ReferenciaCatalogo.CatalogoVersion))
	canon.texto(motivo.ReferenciaCatalogo.CatalogoHuellaSHA256)
	canon.texto(motivo.ReferenciaCatalogo.EntradaClave)
	canon.texto(string(motivo.ClaveI18n))
}

func preimagenOperacionDecisionCoberturaValida(valor []byte) bool {
	return len(valor) > 0 && len(valor) <= maximoBytesCanonOperacionDecisionCobertura
}
