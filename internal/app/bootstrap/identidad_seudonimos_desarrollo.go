package bootstrap

import (
	"context"
	"fmt"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

const espacioIdentidadSesionDesarrollo = "https://localhost/vec/desarrollo/identidad"

var dominioIdentidadSesionDesarrollo = referenciaAltaContratacionTemporalDesarrollo("idh_", espacioIdentidadSesionDesarrollo)

// Adaptación explícita del material local de desarrollo. No es un HSM/KMS
// corporativo y solo se construye dentro de la composición de desarrollo.
// Las cinco etiquetas no comparten preimagen con idempotencia ni con Bolsa.
type seudonimizadorSesionDesarrollo struct {
	derivador *derivadorIdentidadOperacionDesarrollo
}

func (s *seudonimizadorSesionDesarrollo) SeudonimizarAlta(ctx context.Context, ids postgresidentidad.IdentificadoresAlta) (postgresidentidad.SeudonimosAlta, error) {
	vacio := postgresidentidad.SeudonimosAlta{}
	if s == nil || s.derivador == nil || !s.derivador.valido() || contextoInterfazNulo(ctx) || ctx.Err() != nil ||
		ids.EspacioIdentidad != espacioIdentidadSesionDesarrollo || ids.CuentaOrdinariaID != "" {
		return vacio, httpseguridad.ErrSesionNoValida
	}
	resultado := postgresidentidad.SeudonimosAlta{
		Esquema:          postgresidentidad.EsquemaHMACSHA256V1,
		EspacioIdentidad: espacioIdentidadSesionDesarrollo, DominioRef: dominioIdentidadSesionDesarrollo,
		ClaveID:      fmt.Sprintf("vec.identidad.desarrollo.g%d", s.derivador.generaciones[0].generacion),
		ClaveVersion: uint64(s.derivador.generaciones[0].generacion),
	}
	campos := []struct {
		etiqueta, valor string
		destino         *[32]byte
	}{
		{"asercion", ids.AsercionID, &resultado.AsercionIDHMAC},
		{"sesion", ids.SesionID, &resultado.SesionIDHMAC},
		{"sujeto", ids.SujetoID, &resultado.SujetoIDHMAC},
		{"cuenta", ids.CuentaID, &resultado.CuentaIDHMAC},
	}
	for _, campo := range campos {
		if !identificadorSesionDesarrolloValido(campo.valor) {
			return vacio, httpseguridad.ErrSesionNoValida
		}
		preimagen := []byte("vec.identidad.desarrollo.hmac.v1\x00" + espacioIdentidadSesionDesarrollo + "\x00" + campo.etiqueta + "\x00" + campo.valor)
		huellas, err := s.derivador.calcularHMAC(preimagen, preimagen)
		borrarBytes(preimagen)
		if err != nil || len(huellas) == 0 {
			borrarResultadosHMACIdempotenciaDesarrollo(huellas)
			return vacio, httpseguridad.ErrSesionNoValida
		}
		*campo.destino = huellas[0].huellaSolicitud
		borrarResultadosHMACIdempotenciaDesarrollo(huellas)
	}
	return resultado, nil
}

func identificadorSesionDesarrolloValido(valor string) bool {
	if len(valor) == 0 || len(valor) > 512 {
		return false
	}
	for _, caracter := range valor {
		if caracter < 33 || caracter > 126 {
			return false
		}
	}
	return true
}
