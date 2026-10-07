package adminperfiles

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"time"

	h "vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
)

// La aserción es local a una petición: el token aleatorio nunca sale del
// proceso y enlaza la observación F con el ServicioIdentidad ya existente.
type asercionInterna struct {
	clave     [32]byte
	cuenta    CuentaADMIN
	observada ObservacionADMIN
	config    h.ConfiguracionSuperficie
	contenido h.AsercionProxyIdentidad
}

func (a *asercionInterna) Verificar(ctx context.Context, dato []byte) (h.AsercionProxyIdentidad, error) {
	if a == nil || ctx == nil || ctx.Err() != nil || len(dato) != len(a.clave) ||
		subtle.ConstantTimeCompare(dato, a.clave[:]) != 1 {
		return h.AsercionProxyIdentidad{}, h.ErrAsercionNoValida
	}
	return a.contenido, nil
}

func (a *asercionInterna) Evaluar(ctx context.Context, entrada h.EntradaEvaluacionGarantia) (h.ResultadoEvaluacionGarantia, error) {
	if a == nil || ctx == nil || ctx.Err() != nil || entrada.Superficie != h.SuperficieAdministracionPrivilegiada ||
		entrada.Emisor != a.config.EmisorIdentidad || entrada.ACRVerificado != a.cuenta.PoliticaGarantiaRef ||
		entrada.SujetoID != a.cuenta.SujetoID || entrada.CuentaID != a.cuenta.CuentaID ||
		entrada.MetodoPrimario != h.MetodoCertificado || len(entrada.Factores) != 1 ||
		entrada.Factores[0] != a.contenido.Factores[0] {
		return h.ResultadoEvaluacionGarantia{}, h.ErrAsercionNoValida
	}
	return h.ResultadoEvaluacionGarantia{Garantia: domain.AuthAssuranceHigh,
		PoliticaRef:    a.cuenta.PoliticaGarantiaRef,
		HuellaPolitica: "sha256:" + a.cuenta.PoliticaGarantiaHuellaSHA256}, nil
}

func (a *asercionInterna) preparar(c CuentaADMIN, observada ObservacionADMIN, config h.ConfiguracionSuperficie,
	canal string, ahora, hasta time.Time) {
	a.cuenta, a.observada, a.config = c, observada, config
	sesion := sha256.Sum256(append([]byte("admin-perfiles-sesion-v1:"), a.clave[:]...))
	a.contenido = h.AsercionProxyIdentidad{
		ID: hex.EncodeToString(a.clave[:]), SesionID: hex.EncodeToString(sesion[:]),
		Emisor: config.EmisorIdentidad, Audiencia: config.Audiencia,
		Superficie: h.SuperficieAdministracionPrivilegiada,
		SujetoID:   c.SujetoID,
		Cuenta: h.CuentaAcceso{ID: c.CuentaID, CuentaOrdinariaID: c.CuentaOrdinariaID,
			SujetoVinculadoID: c.SujetoID, Privilegiada: true},
		CanalVinculadoRef: canal, AutenticacionVerificadaEn: observada.AutenticacionVerificadaEn,
		EmitidaEn: ahora, NoAntesDe: ahora, ExpiraEn: hasta,
		MetodoPrimario: h.MetodoCertificado, ACRVerificado: c.PoliticaGarantiaRef,
		Factores: []h.FactorAutenticacion{{Metodo: h.MetodoCertificado,
			SujetoVinculadoID: c.SujetoID, CredencialRef: "certificado:sha256:" + observada.CertificadoSHA256,
			EvidenciaRef: canal, GrupoCriptograficoRef: "certificado:sha256:" + observada.CertificadoSHA256,
			VerificadoEn: ahora}},
	}
}
