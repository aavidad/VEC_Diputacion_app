package bootstrap

import (
	"context"
	"fmt"
	"strings"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

// SeudonimizarPresentacionCertificado comparte la MISMA clave HSM fijada para
// el alta, con tres etiquetas nuevas de propósito. Emite solo el epoch
// primario; el adaptador y el gobierno SQL exigen el ID/version fijados y
// deniegan una rotación no coordinada, aunque el HSM conserve claves previas.
func (s *seudonimizadorSesionDesarrollo) SeudonimizarPresentacionCertificado(
	ctx context.Context,
	ids postgresidentidad.IdentificadoresPresentacionCertificado,
) (postgresidentidad.SeudonimosPresentacionCertificado, error) {
	vacia := postgresidentidad.SeudonimosPresentacionCertificado{}
	if s == nil || contextoInterfazNulo(ctx) || ctx.Err() != nil {
		return vacia, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	espacio, dominio, dominioHMAC := espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo, "vec.identidad.desarrollo.hmac.v1"
	if s.configuracion != nil {
		espacio, dominio, dominioHMAC = s.configuracion.EspacioIdentidad, s.configuracion.DominioRef, s.configuracion.DominioHMAC
	}
	if s.derivador == nil || !s.derivador.valido() || len(s.derivador.generaciones) == 0 ||
		ids.EspacioIdentidad != espacio ||
		!huellaPresentacionCertificadoValida(ids.CertificadoSHA256) ||
		!huellaPresentacionCertificadoValida(ids.CASHA256) {
		return vacia, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	espacioClave := "vec.identidad.desarrollo"
	if s.derivador.espacioSeudonimos != "" {
		espacioClave = s.derivador.espacioSeudonimos
	}
	r := postgresidentidad.SeudonimosPresentacionCertificado{
		Esquema:          postgresidentidad.EsquemaHMACSHA256V1,
		EspacioIdentidad: espacio, DominioRef: dominio,
		ClaveID:      fmt.Sprintf("%s.g%d", espacioClave, s.derivador.generaciones[0].generacion),
		ClaveVersion: uint64(s.derivador.generaciones[0].generacion),
	}
	campos := []struct {
		etiqueta, valor string
		destino         *[32]byte
	}{
		{"asercion", ids.AsercionID, &r.AsercionIDHMAC},
		{"sesion", ids.SesionIDAfirmada, &r.SesionIDHMAC},
		{"sujeto", ids.SujetoID, &r.SujetoIDHMAC},
		{"cuenta", ids.CuentaID, &r.CuentaIDHMAC},
		{"certificado_der", ids.CertificadoSHA256, &r.CertificadoDERHMAC},
		{"ca_der", ids.CASHA256, &r.CAHMAC},
		{"presentacion_nonce", ids.NonceID, &r.NonceHMAC},
	}
	for _, campo := range campos {
		if !identificadorSesionDesarrolloValido(campo.valor) {
			return vacia, httpseguridad.ErrPresentacionCertificadoNoValida
		}
		preimagen := []byte(dominioHMAC + "\x00" + espacio + "\x00" + campo.etiqueta + "\x00" + campo.valor)
		huellas, err := s.derivador.calcularHMAC(preimagen, preimagen)
		borrarBytes(preimagen)
		if err != nil || len(huellas) != len(s.derivador.generaciones) {
			borrarResultadosHMACIdempotenciaDesarrollo(huellas)
			return vacia, httpseguridad.ErrPresentacionCertificadoNoValida
		}
		*campo.destino = huellas[0].huellaSolicitud
		borrarResultadosHMACIdempotenciaDesarrollo(huellas)
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	return r, nil
}

func huellaPresentacionCertificadoValida(v string) bool {
	if !strings.HasPrefix(v, "sha256:") || len(v) != len("sha256:")+64 {
		return false
	}
	for _, c := range v[len("sha256:"):] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return strings.Trim(v[len("sha256:"):], "0") != ""
}

var _ postgresidentidad.SeudonimizadorPresentacionCertificado = (*seudonimizadorSesionDesarrollo)(nil)
