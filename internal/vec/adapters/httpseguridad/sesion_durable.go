package httpseguridad

import (
	"context"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"
)

const (
	longitudMinimaTokenOpaco = 22
	longitudMaximaTokenOpaco = 128
)

// ConfirmacionAltaSesion es el recibo emitido por el registro autoritativo en
// la misma operacion que consume la asercion y crea la sesion.
type ConfirmacionAltaSesion struct {
	AutenticacionRef   string
	AsercionRef        string
	SesionRef          string
	ControlSesionRef   string
	CuentaRef          string
	CuentaOrdinariaRef string
	AltaConfirmada     AltaSesionAtomica
}

// ConsultaSesionActiva identifica de forma completa la sesion que se proyecta.
type ConsultaSesionActiva struct {
	AutenticacionRef, AsercionRef, SesionRef, ControlSesionRef  string
	CuentaRef, CuentaOrdinariaRef                               string
	AsercionID, SesionID, SujetoID, CuentaID, CuentaOrdinariaID string
	CuentaPrivilegiada                                          bool
	Superficie                                                  Superficie
	EmitidaEn, ExpiraEn                                         time.Time
	PoliticaRef, HuellaPolitica                                 string
}

// RegistroSesiones consume la asercion y registra la sesion atomica; al
// proyectar vuelve a comprobar cuentas activas y sesion no revocada.
type RegistroSesiones interface {
	ConsumirAsercionYRegistrar(context.Context, AltaSesionAtomica) (ConfirmacionAltaSesion, error)
	ComprobarSesionYCuentaActivas(context.Context, ConsultaSesionActiva) error
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
	if !referenciaOpacaSesionValida(confirmacion.AutenticacionRef, "aut_") ||
		!referenciaOpacaSesionValida(confirmacion.AsercionRef, "ase_") ||
		!referenciaOpacaSesionValida(confirmacion.SesionRef, "ses_") ||
		!referenciaOpacaSesionValida(confirmacion.ControlSesionRef, "cse_") ||
		!referenciaOpacaSesionValida(confirmacion.CuentaRef, "cta_") ||
		!referenciaOpacaSesionValida(confirmacion.CuentaOrdinariaRef, "cta_") ||
		!altasSesionCoinciden(confirmacion.AltaConfirmada, alta) || referenciaProcedeDeEntrada(confirmacion, alta) {
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
		primera.EmitidaEn.Equal(segunda.EmitidaEn) && primera.ExpiraEn.Equal(segunda.ExpiraEn) &&
		primera.PoliticaRef == segunda.PoliticaRef && primera.HuellaPolitica == segunda.HuellaPolitica
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
		asercionID: estado.asercionID, emisor: estado.emisor, audiencia: estado.audiencia,
		sujetoID: estado.sujetoID, cuentaID: estado.cuenta.ID,
		cuentaOrdinariaID: estado.cuenta.CuentaOrdinariaID, cuentaPrivilegiada: estado.cuenta.Privilegiada,
		sesionID: estado.sesionID, superficie: estado.superficie, metodoPrimario: estado.metodoPrimario,
		garantia: estado.garantia, emitidaEn: estado.emitidaEn, noAntesDe: estado.noAntesDe, expiraEn: estado.expiraEn,
		politicaGarantiaRef: estado.politicaGarantiaRef, huellaPolitica: estado.huellaPolitica,
		huellaConfiguracion: "sha256:" + hex.EncodeToString(huellaConfiguracion[:]),
		canalVinculadoRef:   estado.canalVinculadoRef, factores: factores,
	}
}
