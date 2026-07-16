package ports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrConfirmacionGobiernoConvocatoriaInvalida   = errors.New("bolsa: confirmacion de gobierno de convocatoria invalida")
	ErrVersionGobernadaConvocatoriaYaExiste       = errors.New("bolsa: version gobernada de convocatoria ya existe")
	ErrCASVersionConvocatoriaEnConflicto          = errors.New("bolsa: revision o huella de convocatoria en conflicto")
	ErrRamaVersionConvocatoriaEnConflicto         = errors.New("bolsa: la predecesora ya tiene otra rama")
	ErrUsoAutorizacionConvocatoriaConsumido       = errors.New("bolsa: uso de autorizacion de convocatoria ya consumido")
	ErrAtestacionVerificacionConsumida            = errors.New("bolsa: atestacion de verificacion de convocatoria ya consumida")
	ErrReciboGobiernoConvocatoriaInvalido         = errors.New("bolsa: recibo de gobierno de convocatoria invalido")
	ErrSerializacionGobiernoConvocatoriaProhibida = errors.New("bolsa: serializacion de orden de gobierno de convocatoria prohibida")
)

type PreparacionTransaccionGobiernoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	Material      MaterialIntencionGobiernoConvocatoria
	Idempotencia  TestimonioIdempotenciaConvocatoria
	Autorizacion  puertosvec.EvidenciaUsoDecisionAutorizacion
	SelladoMotivo AtestacionSelladoMotivoConvocatoria
	SolicitadaEn  time.Time
}

func (p PreparacionTransaccionGobiernoConvocatoria) Validar() error {
	return p.validarEn(p.SolicitadaEn)
}

func (p PreparacionTransaccionGobiernoConvocatoria) validarEn(instante time.Time) error {
	recurso, errRecurso := RecursoAutorizableMutacionConvocatoria(p.Material)
	datosAutorizacion, errAutorizacion := p.Autorizacion.Datos()
	datosIdempotencia, errIdempotencia := p.Idempotencia.Datos()
	datosSellado, errSellado := p.SelladoMotivo.DatosParaConsumo()
	if errRecurso != nil || errAutorizacion != nil || errIdempotencia != nil ||
		errSellado != nil ||
		!instanteGobiernoConvocatoriaCanonico(p.SolicitadaEn) ||
		!instanteGobiernoConvocatoriaCanonico(instante) || instante.Before(p.SolicitadaEn) ||
		validarUsoAutorizacionConvocatoria(
			p.Autorizacion, p.Material.Accion, recurso, instante,
		) != nil || p.Idempotencia.ValidarPara(
		p.Material, datosAutorizacion.Decision.PrincipalID,
	) != nil || p.SelladoMotivo.validarParaMaterial(p.Material, instante) != nil ||
		datosSellado.PrincipalRef != datosAutorizacion.Decision.PrincipalID ||
		datosSellado.CorrelacionRef != datosAutorizacion.Decision.CorrelacionRef ||
		instante.Before(datosIdempotencia.EmitidoEn) ||
		!instante.Before(datosIdempotencia.ValidoHasta) {
		return ErrConfirmacionGobiernoConvocatoriaInvalida
	}
	return nil
}

func MaterialAltaBorradorConvocatoria(
	version dominiobolsa.VersionConvocatoriaGobernada,
	predecesora *ReferenciaEstadoVersionConvocatoria,
	versionPredecesora *dominiobolsa.VersionConvocatoriaGobernada,
	motivo AtestacionSelladoMotivoConvocatoria,
) (MaterialIntencionGobiernoConvocatoria, error) {
	nuevo, err := estadoVersionConvocatoria(version)
	hmacMotivo, huellaSolicitudMotivo, errAtestacion := motivo.material()
	representacionMotivo, errMotivo := hmacMotivo.representacionMaterial()
	material := MaterialIntencionGobiernoConvocatoria{
		Esquema: "bolsa.convocatoria.intencion.v1", Accion: AccionCrearBorradorConvocatoria,
		EstadoPrincipalNuevo: nuevo, EstadoRelacionadoEsperado: clonarEstadoVersion(predecesora),
		DominioCriptograficoMotivo:  hmacMotivo.DominioCriptografico,
		GeneracionClaveMotivo:       hmacMotivo.GeneracionClave,
		HuellaSolicitudMotivoSHA256: huellaSolicitudMotivo,
		HuellaMotivoHMACSHA256:      representacionMotivo,
	}
	relacionValida := version.Secuencia == 1 && predecesora == nil &&
		versionPredecesora == nil && version.VersionAnteriorRef == ""
	if version.Secuencia > 1 && predecesora != nil && versionPredecesora != nil {
		estadoPredecesora, errEstado := estadoVersionConvocatoria(*versionPredecesora)
		relacionValida = errEstado == nil && estadoPredecesora == *predecesora &&
			predecesoraExactaParaBorradorSucesor(version, *versionPredecesora)
	}
	if err != nil || errAtestacion != nil || errMotivo != nil || version.Validar() != nil ||
		version.EstadoGobierno != dominiobolsa.EstadoGobiernoConvocatoriaBorrador ||
		!relacionValida || material.Validar() != nil || !motivo.coincideMaterial(material) {
		return MaterialIntencionGobiernoConvocatoria{}, ErrMaterialIntencionConvocatoriaInvalido
	}
	return material, nil
}

func MaterialActualizacionBorradorConvocatoria(
	esperada ReferenciaEstadoVersionConvocatoria,
	version dominiobolsa.VersionConvocatoriaGobernada,
	motivo AtestacionSelladoMotivoConvocatoria,
) (MaterialIntencionGobiernoConvocatoria, error) {
	return materialCambioSimpleConvocatoria(
		AccionActualizarBorradorConvocatoria, esperada, version,
		dominiobolsa.EstadoGobiernoConvocatoriaBorrador, 1, motivo,
	)
}

// MaterialPublicacionConvocatoria exige la predecesora para toda secuencia
// posterior a la primera. Publicada pasa a sustituida; retirada se relee y se
// devuelve sin alteracion. En ambos casos el repositorio bloquea ambas filas.
func MaterialPublicacionConvocatoria(
	esperada ReferenciaEstadoVersionConvocatoria,
	version dominiobolsa.VersionConvocatoriaGobernada,
	predecesoraEsperada *ReferenciaEstadoVersionConvocatoria,
	predecesoraResultado *dominiobolsa.VersionConvocatoriaGobernada,
	motivo AtestacionSelladoMotivoConvocatoria,
) (MaterialIntencionGobiernoConvocatoria, error) {
	nuevo, errNuevo := estadoVersionConvocatoria(version)
	hmacMotivo, huellaSolicitudMotivo, errAtestacion := motivo.material()
	representacionMotivo, errMotivo := hmacMotivo.representacionMaterial()
	material := MaterialIntencionGobiernoConvocatoria{
		Esquema: "bolsa.convocatoria.intencion.v1", Accion: AccionPublicarVersionConvocatoria,
		EstadoPrincipalEsperado: clonarEstadoVersion(&esperada), EstadoPrincipalNuevo: nuevo,
		DominioCriptograficoMotivo:  hmacMotivo.DominioCriptografico,
		GeneracionClaveMotivo:       hmacMotivo.GeneracionClave,
		HuellaSolicitudMotivoSHA256: huellaSolicitudMotivo,
		HuellaMotivoHMACSHA256:      representacionMotivo,
	}
	if version.Secuencia == 1 {
		if predecesoraEsperada != nil || predecesoraResultado != nil {
			return MaterialIntencionGobiernoConvocatoria{}, ErrMaterialIntencionConvocatoriaInvalido
		}
	} else {
		if predecesoraEsperada == nil || predecesoraResultado == nil ||
			predecesoraResultado.Validar() != nil ||
			predecesoraResultado.Referencia() != version.VersionAnteriorRef ||
			predecesoraEsperada.Referencia != version.VersionAnteriorRef ||
			!parejaPublicacionSucesoraExacta(version, *predecesoraResultado) {
			return MaterialIntencionGobiernoConvocatoria{}, ErrMaterialIntencionConvocatoriaInvalido
		}
		estadoPredecesora, err := estadoVersionConvocatoria(*predecesoraResultado)
		if err != nil {
			return MaterialIntencionGobiernoConvocatoria{}, ErrMaterialIntencionConvocatoriaInvalido
		}
		material.EstadoRelacionadoEsperado = clonarEstadoVersion(predecesoraEsperada)
		material.EstadoRelacionadoNuevo = clonarEstadoVersion(&estadoPredecesora)
		switch predecesoraResultado.EstadoGobierno {
		case dominiobolsa.EstadoGobiernoConvocatoriaSustituida:
			material.Accion = AccionPublicarYSustituirConvocatoria
		case dominiobolsa.EstadoGobiernoConvocatoriaRetirada:
			material.Accion = AccionPublicarTrasRetiradaConvocatoria
		default:
			return MaterialIntencionGobiernoConvocatoria{}, ErrMaterialIntencionConvocatoriaInvalido
		}
	}
	if errNuevo != nil || errAtestacion != nil || errMotivo != nil || version.Validar() != nil ||
		version.EstadoGobierno != dominiobolsa.EstadoGobiernoConvocatoriaPublicada ||
		esperada.Referencia != version.Referencia() || esperada.Revision != version.Revision ||
		material.Validar() != nil || !motivo.coincideMaterial(material) {
		return MaterialIntencionGobiernoConvocatoria{}, ErrMaterialIntencionConvocatoriaInvalido
	}
	return material, nil
}

func MaterialRetiradaConvocatoria(
	esperada ReferenciaEstadoVersionConvocatoria,
	version dominiobolsa.VersionConvocatoriaGobernada,
	motivo AtestacionSelladoMotivoConvocatoria,
) (MaterialIntencionGobiernoConvocatoria, error) {
	return materialCambioSimpleConvocatoria(
		AccionRetirarVersionConvocatoria, esperada, version,
		dominiobolsa.EstadoGobiernoConvocatoriaRetirada, 0, motivo,
	)
}

func materialCambioSimpleConvocatoria(
	accion string,
	esperada ReferenciaEstadoVersionConvocatoria,
	version dominiobolsa.VersionConvocatoriaGobernada,
	estado dominiobolsa.EstadoGobiernoConvocatoria,
	aumentoRevision int,
	motivo AtestacionSelladoMotivoConvocatoria,
) (MaterialIntencionGobiernoConvocatoria, error) {
	nuevo, err := estadoVersionConvocatoria(version)
	hmacMotivo, huellaSolicitudMotivo, errAtestacion := motivo.material()
	representacionMotivo, errMotivo := hmacMotivo.representacionMaterial()
	material := MaterialIntencionGobiernoConvocatoria{
		Esquema: "bolsa.convocatoria.intencion.v1", Accion: accion,
		EstadoPrincipalEsperado: clonarEstadoVersion(&esperada), EstadoPrincipalNuevo: nuevo,
		DominioCriptograficoMotivo:  hmacMotivo.DominioCriptografico,
		GeneracionClaveMotivo:       hmacMotivo.GeneracionClave,
		HuellaSolicitudMotivoSHA256: huellaSolicitudMotivo,
		HuellaMotivoHMACSHA256:      representacionMotivo,
	}
	if err != nil || errAtestacion != nil || errMotivo != nil || version.Validar() != nil || version.EstadoGobierno != estado ||
		esperada.Referencia != version.Referencia() ||
		version.Revision != esperada.Revision+aumentoRevision || material.Validar() != nil ||
		!motivo.coincideMaterial(material) {
		return MaterialIntencionGobiernoConvocatoria{}, ErrMaterialIntencionConvocatoriaInvalido
	}
	return material, nil
}

type ConfirmacionAltaBorradorConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	Version             dominiobolsa.VersionConvocatoriaGobernada
	PredecesoraEsperada *ReferenciaEstadoVersionConvocatoria
	Predecesora         *dominiobolsa.VersionConvocatoriaGobernada
	Transaccion         PreparacionTransaccionGobiernoConvocatoria
}

func (c ConfirmacionAltaBorradorConvocatoria) Validar() error {
	material, err := MaterialAltaBorradorConvocatoria(
		c.Version, c.PredecesoraEsperada, c.Predecesora, c.Transaccion.SelladoMotivo,
	)
	if err != nil || !materialesIntencionConvocatoriaIguales(material, c.Transaccion.Material) ||
		c.Transaccion.Validar() != nil {
		return ErrConfirmacionGobiernoConvocatoriaInvalida
	}
	return nil
}

func (ConfirmacionAltaBorradorConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionGobiernoConvocatoriaProhibida
}
func (ConfirmacionAltaBorradorConvocatoria) String() string {
	return "[ORDEN-ALTA-CONVOCATORIA-INTERNA]"
}
func (c ConfirmacionAltaBorradorConvocatoria) GoString() string { return c.String() }
func (c ConfirmacionAltaBorradorConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c ConfirmacionAltaBorradorConvocatoria) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

func (c ConfirmacionAltaBorradorConvocatoria) ValidarRecibo(
	recibo ReciboGobiernoConvocatoria,
) error {
	if c.Validar() != nil || recibo.ValidarPara(c.Transaccion) != nil {
		return ErrReciboGobiernoConvocatoriaInvalido
	}
	return nil
}

type ConfirmacionActualizacionBorradorConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	Version     dominiobolsa.VersionConvocatoriaGobernada
	Esperada    ReferenciaEstadoVersionConvocatoria
	Transaccion PreparacionTransaccionGobiernoConvocatoria
}

func (c ConfirmacionActualizacionBorradorConvocatoria) Validar() error {
	material, err := MaterialActualizacionBorradorConvocatoria(
		c.Esperada, c.Version, c.Transaccion.SelladoMotivo,
	)
	if err != nil || !materialesIntencionConvocatoriaIguales(material, c.Transaccion.Material) ||
		c.Transaccion.Validar() != nil {
		return ErrConfirmacionGobiernoConvocatoriaInvalida
	}
	return nil
}

func (ConfirmacionActualizacionBorradorConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionGobiernoConvocatoriaProhibida
}
func (ConfirmacionActualizacionBorradorConvocatoria) String() string {
	return "[ORDEN-ACTUALIZACION-CONVOCATORIA-INTERNA]"
}
func (c ConfirmacionActualizacionBorradorConvocatoria) GoString() string { return c.String() }
func (c ConfirmacionActualizacionBorradorConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c ConfirmacionActualizacionBorradorConvocatoria) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

func (c ConfirmacionActualizacionBorradorConvocatoria) ValidarRecibo(
	recibo ReciboGobiernoConvocatoria,
) error {
	if c.Validar() != nil || recibo.ValidarPara(c.Transaccion) != nil {
		return ErrReciboGobiernoConvocatoriaInvalido
	}
	return nil
}

type ConfirmacionPublicacionConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	VersionPublicada     dominiobolsa.VersionConvocatoriaGobernada
	PublicadaEsperada    ReferenciaEstadoVersionConvocatoria
	PredecesoraResultado *dominiobolsa.VersionConvocatoriaGobernada
	PredecesoraEsperada  *ReferenciaEstadoVersionConvocatoria
	Dependencias         AtestacionDependenciasConvocatoria
	Aprobacion           AtestacionAprobacionConvocatoria
	Transaccion          PreparacionTransaccionGobiernoConvocatoria
}

func (c ConfirmacionPublicacionConvocatoria) Validar() error {
	material, err := MaterialPublicacionConvocatoria(
		c.PublicadaEsperada, c.VersionPublicada, c.PredecesoraEsperada,
		c.PredecesoraResultado, c.Transaccion.SelladoMotivo,
	)
	datosDependencias, errDependencias := c.Dependencias.DatosParaConsumo()
	datosAprobacion, errAprobacion := c.Aprobacion.DatosParaConsumo()
	if err != nil || !materialesIntencionConvocatoriaIguales(material, c.Transaccion.Material) ||
		c.Transaccion.Validar() != nil ||
		errDependencias != nil || errAprobacion != nil ||
		!atestacionVerificacionVigenteEn(
			datosDependencias.AtestacionEmitidaEn, datosDependencias.AtestacionValidaHasta,
			c.Transaccion.SolicitadaEn,
		) || !atestacionVerificacionVigenteEn(
		datosAprobacion.AtestacionEmitidaEn, datosAprobacion.AtestacionValidaHasta,
		c.Transaccion.SolicitadaEn,
	) || datosDependencias.TokenConsumoRef == datosAprobacion.TokenConsumoRef ||
		datosDependencias.AtestacionRef == datosAprobacion.AtestacionRef ||
		datosDependencias.RevisionVersion != c.PublicadaEsperada.Revision ||
		datosDependencias.HuellaEstadoVersionSHA256 != c.PublicadaEsperada.HuellaEstadoSHA256 ||
		datosAprobacion.RevisionVersion != c.PublicadaEsperada.Revision ||
		datosAprobacion.HuellaEstadoVersionSHA256 != c.PublicadaEsperada.HuellaEstadoSHA256 ||
		c.VersionPublicada.ComprobacionDependencias == nil ||
		*c.VersionPublicada.ComprobacionDependencias != datosDependencias.Evidencia ||
		c.VersionPublicada.AprobacionPublicacion == nil ||
		*c.VersionPublicada.AprobacionPublicacion != datosAprobacion.Evidencia {
		return ErrConfirmacionGobiernoConvocatoriaInvalida
	}
	return nil
}

func (ConfirmacionPublicacionConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionGobiernoConvocatoriaProhibida
}
func (ConfirmacionPublicacionConvocatoria) String() string {
	return "[ORDEN-PUBLICACION-CONVOCATORIA-INTERNA]"
}
func (c ConfirmacionPublicacionConvocatoria) GoString() string { return c.String() }
func (c ConfirmacionPublicacionConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c ConfirmacionPublicacionConvocatoria) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

func (c ConfirmacionPublicacionConvocatoria) ValidarRecibo(
	recibo ReciboGobiernoConvocatoria,
) error {
	datosDependencias, errDependencias := c.Dependencias.DatosParaConsumo()
	datosAprobacion, errAprobacion := c.Aprobacion.DatosParaConsumo()
	esperadoDependencias := reciboConsumoDependencias(datosDependencias)
	esperadoAprobacion := reciboConsumoAprobacion(datosAprobacion)
	if c.Validar() != nil || errDependencias != nil || errAprobacion != nil ||
		recibo.ValidarPara(c.Transaccion) != nil || recibo.ConsumoDependencias == nil ||
		recibo.ConsumoAprobacion == nil || *recibo.ConsumoDependencias != esperadoDependencias ||
		*recibo.ConsumoAprobacion != esperadoAprobacion ||
		!atestacionVerificacionVigenteEn(
			datosDependencias.AtestacionEmitidaEn, datosDependencias.AtestacionValidaHasta,
			recibo.ConfirmadaEn,
		) || !atestacionVerificacionVigenteEn(
		datosAprobacion.AtestacionEmitidaEn, datosAprobacion.AtestacionValidaHasta,
		recibo.ConfirmadaEn,
	) {
		return ErrReciboGobiernoConvocatoriaInvalido
	}
	return nil
}

type ConfirmacionRetiradaConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	Version     dominiobolsa.VersionConvocatoriaGobernada
	Esperada    ReferenciaEstadoVersionConvocatoria
	Aprobacion  AtestacionAprobacionConvocatoria
	Transaccion PreparacionTransaccionGobiernoConvocatoria
}

func (c ConfirmacionRetiradaConvocatoria) Validar() error {
	material, err := MaterialRetiradaConvocatoria(
		c.Esperada, c.Version, c.Transaccion.SelladoMotivo,
	)
	datosAprobacion, errAprobacion := c.Aprobacion.DatosParaConsumo()
	if err != nil || !materialesIntencionConvocatoriaIguales(material, c.Transaccion.Material) ||
		c.Transaccion.Validar() != nil ||
		errAprobacion != nil || !atestacionVerificacionVigenteEn(
		datosAprobacion.AtestacionEmitidaEn, datosAprobacion.AtestacionValidaHasta,
		c.Transaccion.SolicitadaEn,
	) || datosAprobacion.RevisionVersion != c.Esperada.Revision ||
		datosAprobacion.HuellaEstadoVersionSHA256 != c.Esperada.HuellaEstadoSHA256 ||
		c.Version.AprobacionRetirada == nil ||
		*c.Version.AprobacionRetirada != datosAprobacion.Evidencia {
		return ErrConfirmacionGobiernoConvocatoriaInvalida
	}
	return nil
}

func (ConfirmacionRetiradaConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionGobiernoConvocatoriaProhibida
}
func (ConfirmacionRetiradaConvocatoria) String() string {
	return "[ORDEN-RETIRADA-CONVOCATORIA-INTERNA]"
}
func (c ConfirmacionRetiradaConvocatoria) GoString() string { return c.String() }
func (c ConfirmacionRetiradaConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c ConfirmacionRetiradaConvocatoria) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

func (c ConfirmacionRetiradaConvocatoria) ValidarRecibo(
	recibo ReciboGobiernoConvocatoria,
) error {
	datosAprobacion, errAprobacion := c.Aprobacion.DatosParaConsumo()
	esperadoAprobacion := reciboConsumoAprobacion(datosAprobacion)
	if c.Validar() != nil || errAprobacion != nil || recibo.ValidarPara(c.Transaccion) != nil ||
		recibo.ConsumoAprobacion == nil || *recibo.ConsumoAprobacion != esperadoAprobacion ||
		!atestacionVerificacionVigenteEn(
			datosAprobacion.AtestacionEmitidaEn, datosAprobacion.AtestacionValidaHasta,
			recibo.ConfirmadaEn,
		) {
		return ErrReciboGobiernoConvocatoriaInvalido
	}
	return nil
}

// RepositorioGobiernoConvocatorias es la barrera durable, no un simple DAO.
// Dentro de la MISMA transaccion debe bloquear y releer las filas afectadas,
// comparar revision+huella, revalidar la decision registrada y su instantanea
// de politicas y una atestacion PDP registrada/COSE cuya procedencia ya haya
// sido verificada, releer el testimonio registrado del protector idempotente,
// y verificar las atestaciones de sellado HSM/KMS, aprobacion y dependencias.
// El indice (principal, HMAC idempotente) devuelve el recibo previo si coincide
// la intencion y rechaza su reutilizacion si difiere. Decision y tokens de
// sellado/verificacion se consumen una sola vez. Solo despues
// confirma agregado(s), auditoria encadenada y outbox en un COMMIT indivisible.
// Una validacion previa de la aplicacion nunca sustituye estas comprobaciones.
// ConfirmarAlta bloquea y relee la predecesora de secuencia >1; publicacion
// vuelve a bloquear ambas versiones y evita que dos ramas la reclamen.
// La composicion productiva permanece NO-GO hasta disponer de ese adaptador
// durable en el mismo TCB/BD; EvidenciaUsoDecisionAutorizacion por si sola no
// acredita la procedencia del PDP.
type RepositorioGobiernoConvocatorias interface {
	ConfirmarAltaBorrador(context.Context, ConfirmacionAltaBorradorConvocatoria) (ReciboGobiernoConvocatoria, error)
	ConfirmarActualizacionBorrador(context.Context, ConfirmacionActualizacionBorradorConvocatoria) (ReciboGobiernoConvocatoria, error)
	ConfirmarPublicacion(context.Context, ConfirmacionPublicacionConvocatoria) (ReciboGobiernoConvocatoria, error)
	ConfirmarRetirada(context.Context, ConfirmacionRetiradaConvocatoria) (ReciboGobiernoConvocatoria, error)
}
