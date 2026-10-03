package auditoria

import (
	"context"
	"errors"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ConfiguracionIntentosConsulta procede de composición y catálogos confiables.
// Los recursos son referencias opacas acreditadas por la raíz, no datos HTTP.
type ConfiguracionIntentosConsulta struct {
	Proceso, Canal                string
	RecursoCTRef, RecursoBolsaRef string
	FinalidadRef                  string
	Motivo                        vecdomain.ReferenciaEntradaCatalogo
	Plazo                         time.Duration
}

type intentosConsulta struct {
	registrador vecports.RegistradorIntentosAuditoria
	config      ConfiguracionIntentosConsulta
}

// NuevoServicioConIntentos exige el registrador común sin conceder permisos.
// Cada acceso permitido conserva la auditoría de su consumo V3 original.
func NuevoServicioConIntentos(emisor EmisorMaterialV3, ct, bolsa FuenteAuditoria,
	registrador vecports.RegistradorIntentosAuditoria, cfg ConfiguracionIntentosConsulta,
) (*Servicio, error) {
	if dependenciaNula(registrador) || cfg.Plazo <= 0 || cfg.Plazo > 30*time.Second ||
		(cfg.Canal != "interna_corporativa" && cfg.Canal != "administracion_privilegiada") {
		return nil, ErrNoDisponible
	}
	for _, recurso := range []string{cfg.RecursoCTRef, cfg.RecursoBolsaRef} {
		datos := vecdomain.DatosIntentoAuditoria{
			Accion: AccionConsultar, ModuloID: ModuloAutorizacion, RecursoRef: recurso,
			FinalidadRef: cfg.FinalidadRef, Motivo: cfg.Motivo,
			Proceso: cfg.Proceso, Canal: cfg.Canal, Resultado: vecdomain.ResultadoIntentoAuditoriaError,
			CorrelacionRef: "correlacion_11111111111111111111111111111111",
		}
		if datos.Validar() != nil {
			return nil, ErrNoDisponible
		}
	}
	s, err := NuevoServicio(emisor, ct, bolsa)
	if err != nil {
		return nil, err
	}
	s.intentos = &intentosConsulta{registrador: registrador, config: cfg}
	return s, nil
}

type intentoConsulta struct {
	referencia string
	contexto   ContextoConsulta
	datos      vecdomain.DatosIntentoAuditoria
}

func (i *intentosConsulta) preparar(p Peticion) (intentoConsulta, error) {
	c := p.Contexto
	v, err := c.Vinculo.Datos()
	if err != nil || c.Resultado.Validar() != nil || c.Vinculo.ValidarPara(c.Resultado) != nil ||
		c.Correlacion.Validar() != nil || c.Motivo != i.config.Motivo ||
		string(v.Superficie) != i.config.Canal || p.Filtro.FinalidadRef != i.config.FinalidadRef ||
		p.Filtro.MotivoRef != i.config.Motivo.Referencia() {
		return intentoConsulta{}, ErrDenegada
	}
	recurso := i.config.RecursoCTRef
	switch p.Filtro.Fuente {
	case "ct":
	case "bolsa":
		recurso = i.config.RecursoBolsaRef
	default:
		return intentoConsulta{}, ErrDenegada
	}
	if p.Filtro.ExpedienteRef != recurso {
		return intentoConsulta{}, ErrDenegada
	}
	correlacion, err := c.Correlacion.ValorCanonico()
	if err != nil {
		return intentoConsulta{}, ErrDenegada
	}
	ref, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return intentoConsulta{}, ErrNoDisponible
	}
	c.Resultado, err = c.Resultado.Clonar()
	if err != nil {
		return intentoConsulta{}, ErrDenegada
	}
	return intentoConsulta{referencia: ref, contexto: c, datos: vecdomain.DatosIntentoAuditoria{
		Accion: AccionConsultar, ModuloID: ModuloAutorizacion, RecursoRef: recurso,
		FinalidadRef: i.config.FinalidadRef, Motivo: i.config.Motivo,
		Proceso: i.config.Proceso, Canal: i.config.Canal, CorrelacionRef: correlacion,
	}}, nil
}

type falloConsultaAuditado struct {
	causa error
	acuse vecports.AcuseIntentoAuditoria
}

func (f *falloConsultaAuditado) Error() string { return f.causa.Error() }
func (f *falloConsultaAuditado) Unwrap() error { return f.causa }

// AcuseIntentoConsulta entrega solo el recibo validado tras COMMIT. Una
// respuesta fallida sin acuse no implica que el registro haya sido confirmado.
func AcuseIntentoConsulta(fallo error) (vecports.AcuseIntentoAuditoria, bool) {
	var auditado *falloConsultaAuditado
	if !errors.As(fallo, &auditado) || auditado == nil {
		return vecports.AcuseIntentoAuditoria{}, false
	}
	return auditado.acuse, true
}

func (i *intentosConsulta) registrar(ctx context.Context, intento intentoConsulta, fallo error) (vecports.AcuseIntentoAuditoria, error) {
	intento.datos.Resultado = vecdomain.ResultadoIntentoAuditoriaError
	if errors.Is(fallo, ErrDenegada) {
		intento.datos.Resultado = vecdomain.ResultadoIntentoAuditoriaDenegado
	}
	// Error es el resultado observado. Un COMMIT ambiguo nunca acredita rollback.
	orden, err := vecports.NuevaOrdenIntentoAuditoria(intento.referencia,
		intento.contexto.Resultado, intento.contexto.Vinculo, intento.datos)
	if err != nil {
		return vecports.AcuseIntentoAuditoria{}, ErrNoDisponible
	}
	registroCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), i.config.Plazo)
	defer cancelar()
	for n := 0; n < 2; n++ {
		acuse, err := i.registrador.AppendIntentoAuditoria(registroCtx, orden)
		if err == nil {
			if acuse.ValidarPara(orden) != nil {
				return vecports.AcuseIntentoAuditoria{}, ErrNoDisponible
			}
			return acuse, nil
		}
		if !errors.Is(err, vecports.ErrIntentoAuditoriaNoDisponible) || registroCtx.Err() != nil {
			return vecports.AcuseIntentoAuditoria{}, ErrNoDisponible
		}
	}
	return vecports.AcuseIntentoAuditoria{}, ErrNoDisponible
}
