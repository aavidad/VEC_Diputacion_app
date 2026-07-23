package ports

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// DatosContextoPeticionIntegracionBolsa contiene exclusivamente referencias
// resueltas por servidor. Acción, recurso, finalidad y autorización proceden
// de catálogos y autoridades comunes, nunca de campos libres del cliente.
type DatosContextoPeticionIntegracionBolsa struct {
	OperacionRef         string                               `json:"operacion_ref"`
	OrganizacionRef      string                               `json:"organizacion_ref"`
	ExpedienteRef        string                               `json:"expediente_ref"`
	VersionExpediente    uint64                               `json:"version_expediente"`
	CorrelacionRef       string                               `json:"correlacion_ref"`
	ContratoVersion      uint64                               `json:"contrato_version"`
	AutoridadSolicitante string                               `json:"autoridad_solicitante"`
	Autorizacion         ReferenciaVersionadaIntegracionBolsa `json:"autorizacion"`
	Accion               ReferenciaVersionadaIntegracionBolsa `json:"accion"`
	Recurso              ReferenciaVersionadaIntegracionBolsa `json:"recurso"`
	Finalidad            ReferenciaVersionadaIntegracionBolsa `json:"finalidad"`
	SolicitadaEn         time.Time                            `json:"solicitada_en"`
	ValidaHasta          time.Time                            `json:"valida_hasta"`
}

func (d DatosContextoPeticionIntegracionBolsa) validarEstructura() error {
	if !domain.ReferenciaOpacaValida(d.OperacionRef) ||
		!domain.ReferenciaOpacaValida(d.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(d.ExpedienteRef) ||
		!enteroSeguroBolsa(d.VersionExpediente) ||
		!domain.ReferenciaOpacaValida(d.CorrelacionRef) ||
		d.ContratoVersion != VersionContratoIntegracionBolsa ||
		!domain.ReferenciaOpacaValida(d.AutoridadSolicitante) ||
		d.Autorizacion.Validar() != nil || d.Accion.Validar() != nil ||
		d.Recurso.Validar() != nil || d.Finalidad.Validar() != nil ||
		!instanteBolsaCanonico(d.SolicitadaEn) ||
		!instanteBolsaCanonico(d.ValidaHasta) ||
		!d.ValidaHasta.After(d.SolicitadaEn) ||
		d.ValidaHasta.Sub(d.SolicitadaEn) > VigenciaMaximaPeticionIntegracionBolsa {
		return ErrPeticionIntegracionBolsaInvalida
	}
	return nil
}

func (d DatosContextoPeticionIntegracionBolsa) validarEn(instante time.Time) error {
	if d.validarEstructura() != nil || !instanteBolsaCanonico(instante) ||
		instante.Before(d.SolicitadaEn) || !instante.Before(d.ValidaHasta) {
		return ErrPeticionIntegracionBolsaInvalida
	}
	return nil
}

// RegistroContextoPeticionIntegracionBolsa es la forma serializable para un
// adaptador interno. No es autoridad: al rehidratar debe volver a verificarse.
type RegistroContextoPeticionIntegracionBolsa struct {
	Datos                DatosContextoPeticionIntegracionBolsa `json:"datos"`
	ClaveVerificacionRef string                                `json:"clave_verificacion_ref"`
	SelloHMAC            string                                `json:"sello_hmac"`
}

func (r RegistroContextoPeticionIntegracionBolsa) validarSintaxis() error {
	referencia, _, valida := descomponerSelloHMACBolsa(
		r.SelloHMAC,
		dominioSelloPeticionBolsa,
	)
	if r.Datos.validarEstructura() != nil || !valida ||
		referencia != r.ClaveVerificacionRef {
		return ErrPeticionIntegracionBolsaInvalida
	}
	return nil
}

type datosContextoPeticionIntegracionBolsa struct {
	registro RegistroContextoPeticionIntegracionBolsa
}

// ContextoPeticionIntegracionBolsa es una capacidad nominal opaca. Web,
// escritorio, CLI y MCP usan el mismo puerto, pero ninguno puede construirla
// desde JSON, cookies, cabeceras o almacenamiento de navegador.
type ContextoPeticionIntegracionBolsa struct {
	datos *datosContextoPeticionIntegracionBolsa
}

func (ContextoPeticionIntegracionBolsa) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCapacidadBolsa
}

func (*ContextoPeticionIntegracionBolsa) UnmarshalJSON([]byte) error {
	return ErrSerializacionCapacidadBolsa
}

func (c ContextoPeticionIntegracionBolsa) DatosEn(
	instante time.Time,
) (DatosContextoPeticionIntegracionBolsa, error) {
	if c.datos == nil || c.datos.registro.validarSintaxis() != nil ||
		c.datos.registro.Datos.validarEn(instante) != nil {
		return DatosContextoPeticionIntegracionBolsa{}, ErrPeticionIntegracionBolsaInvalida
	}
	return c.datos.registro.Datos, nil
}

func (c ContextoPeticionIntegracionBolsa) datosDurables() (
	DatosContextoPeticionIntegracionBolsa,
	error,
) {
	if c.datos == nil || c.datos.registro.validarSintaxis() != nil {
		return DatosContextoPeticionIntegracionBolsa{}, ErrPeticionIntegracionBolsaInvalida
	}
	return c.datos.registro.Datos, nil
}

func (c ContextoPeticionIntegracionBolsa) Registro() (
	RegistroContextoPeticionIntegracionBolsa,
	error,
) {
	if _, err := c.datosDurables(); err != nil {
		return RegistroContextoPeticionIntegracionBolsa{}, err
	}
	return c.datos.registro, nil
}

type SelladorPeticionIntegracionBolsa interface {
	SellarDatos(context.Context, []byte) (string, error)
}

// EmisorContextoPeticionIntegracionBolsa es una dependencia TCB de salida. La
// aplicación recibe el contexto opaco, no el sellador.
type EmisorContextoPeticionIntegracionBolsa struct {
	autoridadRef string
	claveRef     string
	sellador     SelladorPeticionIntegracionBolsa
}

func NuevoEmisorContextoPeticionIntegracionBolsa(
	autoridadRef string,
	claveActivaRef string,
	sellador SelladorPeticionIntegracionBolsa,
) (*EmisorContextoPeticionIntegracionBolsa, error) {
	referencia, _, valida := descomponerSelloHMACBolsa(
		"hmac-sha256:"+claveActivaRef+":"+digestSintacticoNoNuloBolsa(),
		dominioSelloPeticionBolsa,
	)
	if !domain.ReferenciaOpacaValida(autoridadRef) || !valida ||
		referencia != claveActivaRef || dependenciaIntegracionBolsaNula(sellador) {
		return nil, ErrPeticionIntegracionBolsaInvalida
	}
	return &EmisorContextoPeticionIntegracionBolsa{
		autoridadRef: autoridadRef,
		claveRef:     claveActivaRef,
		sellador:     sellador,
	}, nil
}

func (e *EmisorContextoPeticionIntegracionBolsa) Emitir(
	ctx context.Context,
	datos DatosContextoPeticionIntegracionBolsa,
	instante time.Time,
) (ContextoPeticionIntegracionBolsa, error) {
	if ctx == nil || e == nil || dependenciaIntegracionBolsaNula(e.sellador) ||
		datos.AutoridadSolicitante != e.autoridadRef ||
		datos.validarEn(instante) != nil {
		return ContextoPeticionIntegracionBolsa{}, ErrPeticionIntegracionBolsaInvalida
	}
	if err := ctx.Err(); err != nil {
		return ContextoPeticionIntegracionBolsa{}, err
	}
	material := materialContextoPeticionBolsa(datos)
	defer borrarBytesIntegracionBolsa(material)
	sello, err := e.sellador.SellarDatos(ctx, material)
	if err != nil {
		if ctx.Err() != nil {
			return ContextoPeticionIntegracionBolsa{}, ctx.Err()
		}
		return ContextoPeticionIntegracionBolsa{}, ErrIntegracionBolsaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ContextoPeticionIntegracionBolsa{}, err
	}
	referencia, _, valida := descomponerSelloHMACBolsa(
		sello,
		dominioSelloPeticionBolsa,
	)
	if !valida || referencia != e.claveRef {
		return ContextoPeticionIntegracionBolsa{}, ErrPeticionIntegracionBolsaInvalida
	}
	registro := RegistroContextoPeticionIntegracionBolsa{
		Datos: datos, ClaveVerificacionRef: referencia, SelloHMAC: sello,
	}
	return contextoDesdeRegistroVerificado(registro), nil
}

// AutenticadorContextoPeticionIntegracionBolsa pertenece a la frontera
// receptora. Rehidrata el registro únicamente después de verificar autoridad,
// generación y MAC; nunca recibe capacidad de firma.
type AutenticadorContextoPeticionIntegracionBolsa struct {
	autoridadEsperada string
	anillo            anilloVerificacionHMACBolsa
	verificador       VerificadorHMACIntegracionBolsa
}

func NuevoAutenticadorContextoPeticionIntegracionBolsa(
	autoridadEsperada string,
	claveActivaRef string,
	clavesRetenidas []string,
	verificador VerificadorHMACIntegracionBolsa,
) (*AutenticadorContextoPeticionIntegracionBolsa, error) {
	anillo, err := nuevoAnilloVerificacionHMACBolsa(
		dominioSelloPeticionBolsa,
		claveActivaRef,
		clavesRetenidas,
	)
	if err != nil || !domain.ReferenciaOpacaValida(autoridadEsperada) ||
		dependenciaIntegracionBolsaNula(verificador) {
		return nil, ErrPeticionIntegracionBolsaInvalida
	}
	return &AutenticadorContextoPeticionIntegracionBolsa{
		autoridadEsperada: autoridadEsperada,
		anillo:            anillo,
		verificador:       verificador,
	}, nil
}

func (a *AutenticadorContextoPeticionIntegracionBolsa) Reautenticar(
	ctx context.Context,
	registro RegistroContextoPeticionIntegracionBolsa,
	instante time.Time,
) (ContextoPeticionIntegracionBolsa, error) {
	return a.reautenticar(ctx, registro, instante, true)
}

// reautenticarDurable verifica la procedencia histórica del contexto sin
// reabrir su ventana de uso. La capacidad devuelta sigue rechazando DatosEn
// para un instante caducado y no se expone fuera de este puerto.
func (a *AutenticadorContextoPeticionIntegracionBolsa) reautenticarDurable(
	ctx context.Context,
	registro RegistroContextoPeticionIntegracionBolsa,
	instante time.Time,
) (ContextoPeticionIntegracionBolsa, error) {
	return a.reautenticar(ctx, registro, instante, false)
}

func (a *AutenticadorContextoPeticionIntegracionBolsa) reautenticar(
	ctx context.Context,
	registro RegistroContextoPeticionIntegracionBolsa,
	instante time.Time,
	exigirFrescura bool,
) (ContextoPeticionIntegracionBolsa, error) {
	datosValidos := registro.Datos.validarEstructura() == nil &&
		instanteBolsaCanonico(instante) &&
		!instante.Before(registro.Datos.SolicitadaEn)
	if exigirFrescura {
		datosValidos = registro.Datos.validarEn(instante) == nil
	}
	if ctx == nil || a == nil || dependenciaIntegracionBolsaNula(a.verificador) ||
		registro.validarSintaxis() != nil ||
		!datosValidos ||
		registro.Datos.AutoridadSolicitante != a.autoridadEsperada ||
		!a.anillo.contiene(registro.ClaveVerificacionRef) {
		return ContextoPeticionIntegracionBolsa{}, ErrPeticionIntegracionBolsaInvalida
	}
	if err := ctx.Err(); err != nil {
		return ContextoPeticionIntegracionBolsa{}, err
	}
	material := materialContextoPeticionBolsa(registro.Datos)
	defer borrarBytesIntegracionBolsa(material)
	if err := a.verificador.VerificarDatos(
		ctx,
		registro.ClaveVerificacionRef,
		material,
		registro.SelloHMAC,
	); err != nil {
		if ctx.Err() != nil {
			return ContextoPeticionIntegracionBolsa{}, ctx.Err()
		}
		return ContextoPeticionIntegracionBolsa{}, ErrPeticionIntegracionBolsaInvalida
	}
	if err := ctx.Err(); err != nil {
		return ContextoPeticionIntegracionBolsa{}, err
	}
	return contextoDesdeRegistroVerificado(registro), nil
}

func contextoDesdeRegistroVerificado(
	registro RegistroContextoPeticionIntegracionBolsa,
) ContextoPeticionIntegracionBolsa {
	return ContextoPeticionIntegracionBolsa{
		datos: &datosContextoPeticionIntegracionBolsa{registro: registro},
	}
}

func materialContextoPeticionBolsa(
	datos DatosContextoPeticionIntegracionBolsa,
) []byte {
	c := nuevoCanonicoBolsa("contexto-peticion-autenticado")
	c.campo("operacion_ref", datos.OperacionRef)
	c.campo("organizacion_ref", datos.OrganizacionRef)
	c.campo("expediente_ref", datos.ExpedienteRef)
	c.entero("version_expediente", datos.VersionExpediente)
	c.campo("correlacion_ref", datos.CorrelacionRef)
	c.entero("contrato_version", datos.ContratoVersion)
	c.campo("autoridad_solicitante", datos.AutoridadSolicitante)
	c.referencia("autorizacion", datos.Autorizacion)
	c.referencia("accion", datos.Accion)
	c.referencia("recurso", datos.Recurso)
	c.referencia("finalidad", datos.Finalidad)
	c.instante("solicitada_en", datos.SolicitadaEn)
	c.instante("valida_hasta", datos.ValidaHasta)
	return c.bytes()
}

func registrosContextoIguales(
	primero ContextoPeticionIntegracionBolsa,
	segundo ContextoPeticionIntegracionBolsa,
) bool {
	a, errA := primero.Registro()
	b, errB := segundo.Registro()
	return errA == nil && errB == nil && a.Datos == b.Datos &&
		a.ClaveVerificacionRef == b.ClaveVerificacionRef &&
		hmac.Equal([]byte(a.SelloHMAC), []byte(b.SelloHMAC))
}

func borrarBytesIntegracionBolsa(contenido []byte) {
	for indice := range contenido {
		contenido[indice] = 0
	}
}

func digestSintacticoNoNuloBolsa() string {
	return "1111111111111111111111111111111111111111111111111111111111111111"
}

var _ json.Marshaler = ContextoPeticionIntegracionBolsa{}
