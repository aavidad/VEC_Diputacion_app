package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// EmisorEvidenciaIntegracionBolsa pertenece exclusivamente al adaptador de
// salida confiable del proveedor. No crea efectos ni acredita persistencia:
// el adaptador solo debe pasar recibos que su repositorio haya confirmado.
// Reutiliza exactamente el canon que consume VerificadorEvidenciaIntegracionBolsa.
type EmisorEvidenciaIntegracionBolsa struct {
	autoridad string
	clave     string
	sellador  SelladorPeticionIntegracionBolsa
}

func NuevoEmisorEvidenciaIntegracionBolsa(autoridad, clave string, sellador SelladorPeticionIntegracionBolsa) (*EmisorEvidenciaIntegracionBolsa, error) {
	if _, err := nuevoAnilloVerificacionHMACBolsa(dominioSelloRespuestaBolsa, clave, nil); err != nil ||
		dependenciaIntegracionBolsaNula(sellador) {
		return nil, ErrEvidenciaBolsaNoAutenticada
	}
	if !domain.ReferenciaOpacaValida(autoridad) {
		return nil, ErrEvidenciaBolsaNoAutenticada
	}
	return &EmisorEvidenciaIntegracionBolsa{autoridad: autoridad, clave: clave, sellador: sellador}, nil
}

func (e *EmisorEvidenciaIntegracionBolsa) preparar(p *ProcedenciaIntegracionBolsa) error {
	if e == nil || dependenciaIntegracionBolsaNula(e.sellador) || p == nil || p.AutoridadRef != e.autoridad {
		return ErrEvidenciaBolsaNoAutenticada
	}
	p.Evidencia.ClaveVerificacionRef = e.clave
	// El sello no forma parte de su propio canon. Este valor solo permite
	// validar los demás campos antes de firmar; nunca sale como resultado.
	p.Evidencia.SelloHMAC = "hmac-sha256:" + e.clave + ":" + digestSintacticoNoNuloBolsa()
	return nil
}

func (e *EmisorEvidenciaIntegracionBolsa) sellar(ctx context.Context, tipo, peticion string, entrada, salida []byte, p *ProcedenciaIntegracionBolsa) error {
	defer borrarBytesIntegracionBolsa(entrada)
	defer borrarBytesIntegracionBolsa(salida)
	if ctx == nil || e == nil || ctx.Err() != nil || !p.validarNominal() {
		return ErrEvidenciaBolsaNoAutenticada
	}
	evidencia := nuevaEvidenciaDurableBolsa(tipo, peticion, entrada, salida, *p)
	material := materialAutenticacionRespuestaBolsa(evidencia, salida)
	defer borrarBytesIntegracionBolsa(material)
	sello, err := e.sellador.SellarDatos(ctx, material)
	referencia, _, valida := descomponerSelloHMACBolsa(sello, dominioSelloRespuestaBolsa)
	if err != nil || ctx.Err() != nil || !valida || referencia != e.clave {
		return ErrEvidenciaBolsaNoAutenticada
	}
	p.Evidencia.SelloHMAC = sello
	return nil
}

func (e *EmisorEvidenciaIntegracionBolsa) FirmarDisponibilidad(ctx context.Context, solicitud SolicitudDisponibilidadBolsa, resultado ResultadoDisponibilidadBolsa, instante time.Time) (ResultadoDisponibilidadBolsa, error) {
	if e.preparar(&resultado.Procedencia) != nil || resultado.ValidarParaEn(solicitud, instante) != nil {
		return ResultadoDisponibilidadBolsa{}, ErrRespuestaBolsaNoConfiable
	}
	datos, _ := solicitud.Contexto.DatosEn(instante)
	if err := e.sellar(ctx, "disponibilidad_volatil", datos.OperacionRef,
		materialSolicitudDisponibilidadBolsa(solicitud), materialDisponibilidadBolsa(solicitud, resultado), &resultado.Procedencia); err != nil {
		return ResultadoDisponibilidadBolsa{}, err
	}
	return resultado, nil
}

func (e *EmisorEvidenciaIntegracionBolsa) FirmarOrden(ctx context.Context, comando ComandoPrepararOrdenBolsa, recibo ReciboOrdenBolsa, instante time.Time) (ReciboOrdenBolsa, error) {
	if e.preparar(&recibo.Procedencia) != nil || recibo.ValidarParaEn(comando, instante) != nil {
		return ReciboOrdenBolsa{}, ErrRespuestaBolsaNoConfiable
	}
	datos, _ := comando.Contexto.DatosEn(instante)
	if err := e.sellar(ctx, "recibo_orden", datos.OperacionRef,
		materialComandoOrdenBolsa(comando), materialReciboOrdenBolsa(comando, recibo), &recibo.Procedencia); err != nil {
		return ReciboOrdenBolsa{}, err
	}
	return recibo, nil
}

func (e *EmisorEvidenciaIntegracionBolsa) FirmarLlamamiento(ctx context.Context, comando ComandoSolicitarLlamamientoBolsa, recibo ReciboSolicitudLlamamientoBolsa, instante time.Time) (ReciboSolicitudLlamamientoBolsa, error) {
	if e.preparar(&recibo.Procedencia) != nil || recibo.ValidarParaEn(comando, instante) != nil {
		return ReciboSolicitudLlamamientoBolsa{}, ErrRespuestaBolsaNoConfiable
	}
	datos, _ := comando.DatosEn(instante)
	contexto, _ := datos.Contexto.DatosEn(instante)
	if err := e.sellar(ctx, "recibo_llamamiento", contexto.OperacionRef,
		materialComandoLlamamientoBolsa(comando), materialReciboLlamamientoBolsa(comando, recibo), &recibo.Procedencia); err != nil {
		return ReciboSolicitudLlamamientoBolsa{}, err
	}
	return recibo, nil
}
