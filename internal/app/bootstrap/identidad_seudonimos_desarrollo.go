package bootstrap

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

const espacioIdentidadSesionDesarrollo = "https://localhost/vec/desarrollo/identidad"

var dominioIdentidadSesionDesarrollo = referenciaAltaContratacionTemporalDesarrollo("idh_", espacioIdentidadSesionDesarrollo)

// ConfiguracionSeudonimosSesionPrivada selecciona un dominio aprovisionado.
// El formato HMAC de origen es el cargador privado existente. Producción
// corporativa debe inyectar su broker por el puerto SeudonimizadorAlta.
type ConfiguracionSeudonimosSesionPrivada struct {
	DirectorioMaterial, RutaConfiguracionHMAC               string
	EspacioIdentidad, DominioRef, EspacioClave, DominioHMAC string
	IncluirCuentaOrdinaria                                  bool
}

func NuevoSeudonimizadorSesionDesdeArchivo(cfg ConfiguracionSeudonimosSesionPrivada) (postgresidentidad.SeudonimizadorAlta, func(), error) {
	for _, v := range []string{cfg.EspacioIdentidad, cfg.DominioRef, cfg.EspacioClave, cfg.DominioHMAC} {
		if !identificadorSesionDesarrolloValido(v) {
			return nil, nil, httpseguridad.ErrSesionNoValida
		}
	}
	material, err := cargarMaterialIdempotenciaDesarrollo(cfg.DirectorioMaterial, cfg.RutaConfiguracionHMAC)
	defer material.borrar()
	if err != nil {
		return nil, nil, httpseguridad.ErrSesionNoValida
	}
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&material)
	if err != nil {
		return nil, nil, httpseguridad.ErrSesionNoValida
	}
	derivador.espacioSeudonimos = cfg.EspacioClave
	s := &seudonimizadorSesionDesarrollo{derivador: derivador, configuracion: &cfg}
	cerrar := func() { s.mu.Lock(); defer s.mu.Unlock(); s.derivador.borrar() }
	return s, cerrar, nil
}

type seudonimizadorSesionDesarrollo struct {
	mu            sync.RWMutex
	configuracion *ConfiguracionSeudonimosSesionPrivada
	derivador     *derivadorIdentidadOperacionDesarrollo
}

func (*seudonimizadorSesionDesarrollo) String() string { return "[SEUDONIMIZADOR-SESION-PRIVADO]" }
func (s *seudonimizadorSesionDesarrollo) Format(f fmt.State, _ rune) {
	_, _ = io.WriteString(f, s.String())
}
func (s *seudonimizadorSesionDesarrollo) LogValue() slog.Value { return slog.StringValue(s.String()) }

func (s *seudonimizadorSesionDesarrollo) SeudonimizarAlta(ctx context.Context, ids postgresidentidad.IdentificadoresAlta) (postgresidentidad.SeudonimosAlta, error) {
	vacio := postgresidentidad.SeudonimosAlta{}
	if s == nil {
		return vacio, httpseguridad.ErrSesionNoValida
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	espacio, dominio, dominioHMAC := espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo, "vec.identidad.desarrollo.hmac.v1"
	ordinaria := false
	if s.configuracion != nil {
		espacio, dominio, dominioHMAC = s.configuracion.EspacioIdentidad, s.configuracion.DominioRef, s.configuracion.DominioHMAC
		ordinaria = s.configuracion.IncluirCuentaOrdinaria
	}
	if s == nil || s.derivador == nil || !s.derivador.valido() || contextoInterfazNulo(ctx) || ctx.Err() != nil ||
		ids.EspacioIdentidad != espacio || (!ordinaria && ids.CuentaOrdinariaID != "") {
		return vacio, httpseguridad.ErrSesionNoValida
	}
	espacioClave := "vec.identidad.desarrollo"
	if s.derivador.espacioSeudonimos != "" {
		espacioClave = s.derivador.espacioSeudonimos
	}
	resultado := postgresidentidad.SeudonimosAlta{
		Esquema:          postgresidentidad.EsquemaHMACSHA256V1,
		EspacioIdentidad: espacio, DominioRef: dominio,
		ClaveID:      fmt.Sprintf("%s.g%d", espacioClave, s.derivador.generaciones[0].generacion),
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
	if ordinaria {
		campos = append(campos, struct {
			etiqueta, valor string
			destino         *[32]byte
		}{"cuenta_ordinaria", ids.CuentaOrdinariaID, &resultado.CuentaOrdinariaIDHMAC})
	}
	for _, campo := range campos {
		if !identificadorSesionDesarrolloValido(campo.valor) {
			return vacio, httpseguridad.ErrSesionNoValida
		}
		preimagen := []byte(dominioHMAC + "\x00" + espacio + "\x00" + campo.etiqueta + "\x00" + campo.valor)
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
