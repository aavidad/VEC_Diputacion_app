package httpseguridad

import (
	"context"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	longitudMinimaTokenOpaco = 22
	longitudMaximaTokenOpaco = 128
)

// EstadoControlSesion es el estado persistido en control_sesion_v1. Una
// confirmacion de alta solo puede devolver el primer control activo.
type EstadoControlSesion string

const (
	EstadoControlSesionActiva   EstadoControlSesion = "activa"
	EstadoControlSesionRevocada EstadoControlSesion = "revocada"
)

func (e EstadoControlSesion) Valido() bool {
	return e == EstadoControlSesionActiva || e == EstadoControlSesionRevocada
}

// AltaSesionAtomica contiene todos los valores verificados que el registro
// necesita para escribir sesion_autenticacion_v1. Los identificadores IdP se
// conservan separados de las referencias canonicas emitidas por el registro.
// AsercionExpiraEn es el limite superior que el control no puede ampliar.
type AltaSesionAtomica struct {
	AsercionID         string
	SesionID           string
	SujetoID           string
	CuentaID           string
	CuentaOrdinariaID  string
	CuentaPrivilegiada bool
	Superficie         Superficie
	// EspacioIdentidad es el emisor HTTPS exacto, protegido y ya cotejado con
	// la configuracion. Permite que el adaptador HMAC demuestre que el dominio
	// de seudonimizacion pertenece al IdP correcto sin persistir este URL.
	EspacioIdentidad  string
	MetodoObservado   dominiovec.AuthMethod
	GarantiaObservada dominiovec.AuthAssurance
	// AutenticacionHuellaSHA256 es SHA-256 hexadecimal, sin prefijo, de los
	// bytes exactos de la copia privada de la asercion protegida que el
	// verificador acepto. Solo se incorpora al alta tras verificar con exito.
	AutenticacionHuellaSHA256 string
	// AutenticacionVerificadaEn procede del resultado protegido/verificado;
	// nunca se sustituye por el reloj local posterior.
	AutenticacionVerificadaEn time.Time
	// SesionEmitidaEn procede de AsercionProxyIdentidad.EmitidaEn y acredita
	// la emision de la sesion, no la serializacion material del token.
	SesionEmitidaEn     time.Time
	AsercionExpiraEn    time.Time
	PoliticaGarantiaRef string
	// PoliticaGarantiaHuellaSHA256 es la parte hexadecimal exacta de la huella
	// canonica sha256:... ya validada del evaluador.
	PoliticaGarantiaHuellaSHA256 string
}

// ConfirmacionAltaSesion es el recibo emitido por el registro autoritativo en
// la misma operacion que consume la asercion y crea la sesion.
type ConfirmacionAltaSesion struct {
	AutenticacionRef          string
	AsercionRef               string
	SesionRef                 string
	ControlSesionRef          string
	ControlSesionRevision     uint64
	ControlSesionEstado       EstadoControlSesion
	ControlSesionHuellaSHA256 string
	CuentaRef                 string
	CuentaOrdinariaRef        string
	SesionRevalidadaEn        time.Time
	SesionValidaHasta         time.Time
	AltaConfirmada            AltaSesionAtomica
}

// ConsultaSesionActiva reproduce exactamente las columnas autoritativas de
// sesion_autenticacion_v1 y control_sesion_v1 que deben seguir vigentes. No
// transporta identificadores IdP ni atributos declarados por el cliente.
type ConsultaSesionActiva struct {
	AutenticacionRef             string
	AutenticacionHuellaSHA256    string
	AsercionRef                  string
	SesionRef                    string
	CuentaRef                    string
	CuentaOrdinariaRef           string
	CuentaPrivilegiada           bool
	Superficie                   Superficie
	MetodoObservado              dominiovec.AuthMethod
	GarantiaObservada            dominiovec.AuthAssurance
	PoliticaGarantiaRef          string
	PoliticaGarantiaHuellaSHA256 string
	AutenticacionVerificadaEn    time.Time
	SesionEmitidaEn              time.Time
	ControlSesionRef             string
	ControlSesionRevision        uint64
	ControlSesionEstado          EstadoControlSesion
	ControlSesionHuellaSHA256    string
	SesionRevalidadaEn           time.Time
	SesionValidaHasta            time.Time
}

// RegistroSesiones consume la asercion y registra la sesion atomica; al
// proyectar vuelve a comprobar cuentas activas y sesion no revocada.
type RegistroSesiones interface {
	ConsumirAsercionYRegistrar(context.Context, AltaSesionAtomica) (ConfirmacionAltaSesion, error)
	ComprobarSesionYCuentaActivas(context.Context, ConsultaSesionActiva) error
}

// Validar permite al adaptador durable rechazar el alta antes de abrir una
// transaccion. No concede autoridad: el servicio vuelve a validarla y coteja
// el recibo al regresar del puerto.
func (a AltaSesionAtomica) Validar() error { return validarAltaSesionAtomica(a) }

// Validar comprueba la estructura y la ligadura interna del recibo.
func (c ConfirmacionAltaSesion) Validar() error {
	return validarConfirmacionAltaSesion(c, c.AltaConfirmada)
}

// ValidarPara impide que un recibo valido de otra escritura confirme el alta.
func (c ConfirmacionAltaSesion) ValidarPara(alta AltaSesionAtomica) error {
	return validarConfirmacionAltaSesion(c, alta)
}

// Validar comprueba el conjunto exacto que puede cotejarse directamente con
// las dos tablas autoritativas, sin completar omisiones.
func (c ConsultaSesionActiva) Validar() error {
	if !referenciaOpacaSesionValida(c.AutenticacionRef, "aut_") ||
		!huellaSHA256SesionValida(c.AutenticacionHuellaSHA256) ||
		!referenciaOpacaSesionValida(c.AsercionRef, "ase_") ||
		!referenciaOpacaSesionValida(c.SesionRef, "ses_") ||
		!referenciaOpacaSesionValida(c.CuentaRef, "cta_") ||
		!referenciaOpacaSesionValida(c.CuentaOrdinariaRef, "cta_") ||
		!c.Superficie.Valida() || c.Superficie == SuperficiePublicaAnonima ||
		!c.MetodoObservado.Valido() || c.MetodoObservado == dominiovec.AuthMethodDemo ||
		!c.GarantiaObservada.Valida() ||
		!referenciaOpacaSesionValida(c.PoliticaGarantiaRef, "pga_") ||
		!huellaSHA256SesionValida(c.PoliticaGarantiaHuellaSHA256) ||
		!instanteSesionCanonico(c.AutenticacionVerificadaEn) ||
		!instanteSesionCanonico(c.SesionEmitidaEn) ||
		!referenciaOpacaSesionValida(c.ControlSesionRef, "cse_") ||
		c.ControlSesionRevision == 0 || c.ControlSesionEstado != EstadoControlSesionActiva ||
		!huellaSHA256SesionValida(c.ControlSesionHuellaSHA256) ||
		!instanteSesionCanonico(c.SesionRevalidadaEn) || !instanteSesionCanonico(c.SesionValidaHasta) ||
		c.AutenticacionVerificadaEn.After(c.SesionEmitidaEn) ||
		c.SesionRevalidadaEn.Before(c.AutenticacionVerificadaEn) ||
		c.SesionRevalidadaEn.Before(c.SesionEmitidaEn) ||
		!c.SesionValidaHasta.After(c.SesionRevalidadaEn) {
		return ErrSesionNoValida
	}
	if c.CuentaPrivilegiada {
		if c.Superficie != SuperficieAdministracionPrivilegiada || c.CuentaRef == c.CuentaOrdinariaRef {
			return ErrSesionNoValida
		}
		return nil
	}
	if c.Superficie == SuperficieAdministracionPrivilegiada || c.CuentaRef != c.CuentaOrdinariaRef {
		return ErrSesionNoValida
	}
	return nil
}

// CoincideExactamente compara dos proyecciones ya validadas sin depender de
// la representacion interna de time.Time.
func (c ConsultaSesionActiva) CoincideExactamente(otra ConsultaSesionActiva) bool {
	return c.Validar() == nil && otra.Validar() == nil &&
		c.AutenticacionRef == otra.AutenticacionRef &&
		c.AutenticacionHuellaSHA256 == otra.AutenticacionHuellaSHA256 && c.AsercionRef == otra.AsercionRef &&
		c.SesionRef == otra.SesionRef && c.CuentaRef == otra.CuentaRef &&
		c.CuentaOrdinariaRef == otra.CuentaOrdinariaRef && c.CuentaPrivilegiada == otra.CuentaPrivilegiada &&
		c.Superficie == otra.Superficie && c.MetodoObservado == otra.MetodoObservado &&
		c.GarantiaObservada == otra.GarantiaObservada && c.PoliticaGarantiaRef == otra.PoliticaGarantiaRef &&
		c.PoliticaGarantiaHuellaSHA256 == otra.PoliticaGarantiaHuellaSHA256 &&
		c.AutenticacionVerificadaEn.Equal(otra.AutenticacionVerificadaEn) &&
		c.SesionEmitidaEn.Equal(otra.SesionEmitidaEn) && c.ControlSesionRef == otra.ControlSesionRef &&
		c.ControlSesionRevision == otra.ControlSesionRevision && c.ControlSesionEstado == otra.ControlSesionEstado &&
		c.ControlSesionHuellaSHA256 == otra.ControlSesionHuellaSHA256 &&
		c.SesionRevalidadaEn.Equal(otra.SesionRevalidadaEn) && c.SesionValidaHasta.Equal(otra.SesionValidaHasta)
}

func (IdentidadSesion) MarshalJSON() ([]byte, error) {
	return []byte(`{"identidad_sesion":"[CONFIDENCIAL]"}`), nil
}
func (IdentidadSesion) MarshalText() ([]byte, error) { return []byte(identidadSesionRedactada), nil }
func (*IdentidadSesion) UnmarshalText([]byte) error  { return ErrIdentidadNoSerializable }
func (IdentidadSesion) MarshalBinary() ([]byte, error) {
	return []byte(identidadSesionRedactada), nil
}
func (*IdentidadSesion) UnmarshalBinary([]byte) error { return ErrIdentidadNoSerializable }
func (IdentidadSesion) GobEncode() ([]byte, error)    { return []byte(identidadSesionRedactada), nil }
func (*IdentidadSesion) GobDecode([]byte) error       { return ErrIdentidadNoSerializable }
func (*IdentidadSesion) UnmarshalJSON([]byte) error   { return ErrIdentidadNoSerializable }
func (IdentidadSesion) LogValue() slog.Value          { return slog.StringValue(identidadSesionRedactada) }

func validarConfirmacionAltaSesion(confirmacion ConfirmacionAltaSesion, alta AltaSesionAtomica) error {
	if validarAltaSesionAtomica(alta) != nil ||
		!referenciaOpacaSesionValida(confirmacion.AutenticacionRef, "aut_") ||
		!referenciaOpacaSesionValida(confirmacion.AsercionRef, "ase_") ||
		!referenciaOpacaSesionValida(confirmacion.SesionRef, "ses_") ||
		!referenciaOpacaSesionValida(confirmacion.ControlSesionRef, "cse_") ||
		!referenciaOpacaSesionValida(confirmacion.CuentaRef, "cta_") ||
		!referenciaOpacaSesionValida(confirmacion.CuentaOrdinariaRef, "cta_") ||
		confirmacion.ControlSesionRevision != 1 ||
		confirmacion.ControlSesionEstado != EstadoControlSesionActiva ||
		!huellaSHA256SesionValida(confirmacion.ControlSesionHuellaSHA256) ||
		!instanteSesionCanonico(confirmacion.SesionRevalidadaEn) ||
		!instanteSesionCanonico(confirmacion.SesionValidaHasta) ||
		confirmacion.SesionRevalidadaEn.Before(alta.AutenticacionVerificadaEn) ||
		confirmacion.SesionRevalidadaEn.Before(alta.SesionEmitidaEn) ||
		!confirmacion.SesionValidaHasta.After(confirmacion.SesionRevalidadaEn) ||
		confirmacion.SesionValidaHasta.After(alta.AsercionExpiraEn) ||
		!altasSesionCoinciden(confirmacion.AltaConfirmada, alta) ||
		referenciaProcedeDeEntrada(confirmacion, alta) {
		return ErrSesionNoValida
	}
	if alta.CuentaPrivilegiada {
		if alta.CuentaOrdinariaID == "" || confirmacion.CuentaRef == confirmacion.CuentaOrdinariaRef {
			return ErrSesionNoValida
		}
		return nil
	}
	if alta.CuentaOrdinariaID != "" || confirmacion.CuentaRef != confirmacion.CuentaOrdinariaRef {
		return ErrSesionNoValida
	}
	return nil
}

func validarAltaSesionAtomica(alta AltaSesionAtomica) error {
	if !idAltaSesionCanonico(alta.AsercionID, false) || !idAltaSesionCanonico(alta.SesionID, false) ||
		!idAltaSesionCanonico(alta.SujetoID, false) || !idAltaSesionCanonico(alta.CuentaID, true) ||
		!alta.Superficie.Valida() || alta.Superficie == SuperficiePublicaAnonima ||
		validarEmisorConfigurado(alta.EspacioIdentidad) != nil ||
		!alta.MetodoObservado.Valido() ||
		alta.MetodoObservado == dominiovec.AuthMethodDemo || !alta.GarantiaObservada.Valida() ||
		!huellaSHA256SesionValida(alta.AutenticacionHuellaSHA256) ||
		!referenciaOpacaSesionValida(alta.PoliticaGarantiaRef, "pga_") ||
		!huellaSHA256SesionValida(alta.PoliticaGarantiaHuellaSHA256) ||
		!instanteSesionCanonico(alta.AutenticacionVerificadaEn) ||
		!instanteSesionCanonico(alta.SesionEmitidaEn) ||
		!instanteSesionCanonico(alta.AsercionExpiraEn) ||
		alta.AutenticacionVerificadaEn.After(alta.SesionEmitidaEn) ||
		!alta.AsercionExpiraEn.After(alta.SesionEmitidaEn) ||
		alta.AsercionExpiraEn.Sub(alta.SesionEmitidaEn) > duracionLimiteAsercion {
		return ErrSesionNoValida
	}
	if alta.CuentaPrivilegiada {
		if alta.Superficie != SuperficieAdministracionPrivilegiada ||
			!idAltaSesionCanonico(alta.CuentaOrdinariaID, true) ||
			alta.CuentaOrdinariaID == alta.CuentaID {
			return ErrSesionNoValida
		}
		return nil
	}
	if alta.Superficie == SuperficieAdministracionPrivilegiada || alta.CuentaOrdinariaID != "" {
		return ErrSesionNoValida
	}
	return nil
}

func idAltaSesionCanonico(valor string, minusculas bool) bool {
	canonico, err := canonicalizarID(valor, longitudMaximaID, minusculas)
	return err == nil && canonico == valor
}

func referenciaOpacaSesionValida(valor, prefijo string) bool {
	if len(valor) < len(prefijo)+longitudMinimaTokenOpaco ||
		len(valor) > len(prefijo)+longitudMaximaTokenOpaco || !strings.HasPrefix(valor, prefijo) {
		return false
	}
	for indice := len(prefijo); indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
			(caracter >= '0' && caracter <= '9') || caracter == '_' || caracter == '-' {
			continue
		}
		return false
	}
	return true
}

func altasSesionCoinciden(primera, segunda AltaSesionAtomica) bool {
	return primera.AsercionID == segunda.AsercionID && primera.SesionID == segunda.SesionID &&
		primera.SujetoID == segunda.SujetoID && primera.CuentaID == segunda.CuentaID &&
		primera.CuentaOrdinariaID == segunda.CuentaOrdinariaID &&
		primera.CuentaPrivilegiada == segunda.CuentaPrivilegiada && primera.Superficie == segunda.Superficie &&
		primera.EspacioIdentidad == segunda.EspacioIdentidad &&
		primera.MetodoObservado == segunda.MetodoObservado && primera.GarantiaObservada == segunda.GarantiaObservada &&
		primera.AutenticacionHuellaSHA256 == segunda.AutenticacionHuellaSHA256 &&
		primera.AutenticacionVerificadaEn.Equal(segunda.AutenticacionVerificadaEn) &&
		primera.SesionEmitidaEn.Equal(segunda.SesionEmitidaEn) &&
		primera.AsercionExpiraEn.Equal(segunda.AsercionExpiraEn) &&
		primera.PoliticaGarantiaRef == segunda.PoliticaGarantiaRef &&
		primera.PoliticaGarantiaHuellaSHA256 == segunda.PoliticaGarantiaHuellaSHA256
}

func huellaSHA256SesionValida(valor string) bool {
	if len(valor) != 64 || strings.Trim(valor, "0") == "" {
		return false
	}
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

func instanteSesionCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 && instante.Nanosecond()%1_000 == 0
}

func referenciaProcedeDeEntrada(confirmacion ConfirmacionAltaSesion, alta AltaSesionAtomica) bool {
	referencias := []string{confirmacion.AutenticacionRef, confirmacion.AsercionRef, confirmacion.SesionRef,
		confirmacion.ControlSesionRef, confirmacion.CuentaRef, confirmacion.CuentaOrdinariaRef}
	entradas := []string{alta.AsercionID, alta.SesionID, alta.SujetoID, alta.CuentaID, alta.CuentaOrdinariaID}
	for _, referencia := range referencias {
		for _, entrada := range entradas {
			if entrada != "" && referencia == entrada {
				return true
			}
		}
	}
	return false
}

func nuevoContextoAuditoria(
	estado estadoIdentidadSesion,
	confirmacion ConfirmacionAltaSesion,
	huellaConfiguracion [32]byte,
) ContextoAuditoriaAutenticada {
	factores := make([]ResumenFactorAuditoria, 0, len(estado.factores))
	for _, factor := range estado.factores {
		factores = append(factores, ResumenFactorAuditoria{
			Metodo: factor.Metodo, EvidenciaRef: factor.EvidenciaRef,
			GrupoCriptograficoRef: factor.GrupoCriptograficoRef, VerificadoEn: factor.VerificadoEn,
		})
	}
	return ContextoAuditoriaAutenticada{
		autenticacionRef: confirmacion.AutenticacionRef, asercionRef: confirmacion.AsercionRef,
		sesionRef: confirmacion.SesionRef, controlSesionRef: confirmacion.ControlSesionRef,
		cuentaRef: confirmacion.CuentaRef, cuentaOrdinariaRef: confirmacion.CuentaOrdinariaRef,
		autenticacionHuellaSHA256: estado.autenticacionHuellaSHA256,
		controlSesionRevision:     confirmacion.ControlSesionRevision,
		controlSesionEstado:       confirmacion.ControlSesionEstado,
		controlSesionHuellaSHA256: confirmacion.ControlSesionHuellaSHA256,
		sesionRevalidadaEn:        confirmacion.SesionRevalidadaEn,
		sesionValidaHasta:         confirmacion.SesionValidaHasta,
		emisor:                    estado.emisor, audiencia: estado.audiencia,
		cuentaPrivilegiada: estado.cuenta.Privilegiada,
		superficie:         estado.superficie, metodoPrimario: estado.metodoPrimario,
		metodoObservado: estado.metodoObservado, autenticacionVerificadaEn: estado.autenticacionVerificadaEn,
		garantia: estado.garantia, emitidaEn: estado.emitidaEn, noAntesDe: estado.noAntesDe, expiraEn: estado.expiraEn,
		politicaGarantiaRef: estado.politicaGarantiaRef, huellaPolitica: estado.huellaPolitica,
		huellaConfiguracion: "sha256:" + hex.EncodeToString(huellaConfiguracion[:]),
		canalVinculadoRef:   estado.canalVinculadoRef, factores: factores,
	}
}
