package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	bolsadomain "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const recursoAuditoriaPoliticaCeseBolsa = "bolsa:politica:cese:vigente"

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararConsultaPoliticaCese(r *http.Request) (bolsaports.OrdenConsultaPoliticaCese, error) {
	if r == nil {
		return bolsaports.OrdenConsultaPoliticaCese{}, bolsahttp.ErrAutenticacionInternaAusente
	}
	actor, err := p.contextoRevalidado(r.Context())
	if err != nil {
		return bolsaports.OrdenConsultaPoliticaCese{}, vecdomain.ErrAutorizacionDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(r.Context(), p.generar)
	if err != nil {
		return bolsaports.OrdenConsultaPoliticaCese{}, err
	}
	return bolsaports.OrdenConsultaPoliticaCese{ResultadoContexto: actor.Resultado, Vinculo: actor.Vinculo,
		Correlacion: correlacion, Motivo: motivoConsultarBorradorLlamamientoBolsaDesarrollo()}, nil
}

// La sesión y la fuente central son las ya compuestas para Bolsa B103. La
// única escritura de esta lectura normativa es su apunte nominal común.
type consultaPoliticaCeseBolsaActual struct {
	preparador  *preparadorBorradorLlamamientoDesarrollo
	leer        func(context.Context) (bolsadomain.PoliticaCese, error)
	registrador vecports.RegistradorIntentosAuditoria
	proceso     string
}

func (c *consultaPoliticaCeseBolsaActual) Consultar(ctx context.Context, orden bolsaports.OrdenConsultaPoliticaCese) (politica bolsadomain.PoliticaCese, fallo error) {
	if c == nil || c.preparador == nil || c.preparador.soporte == nil || c.leer == nil ||
		c.registrador == nil || c.proceso == "" || ctx == nil || ctx.Err() != nil {
		return politica, bolsaports.ErrConsultaPoliticaCeseNoDisponible
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	datos, err := orden.Vinculo.Datos()
	if err != nil || orden.ResultadoContexto.Validar() != nil || orden.Vinculo.ValidarPara(orden.ResultadoContexto) != nil ||
		!orden.Vinculo.VigenteEn(ahora, orden.ResultadoContexto) || datos.Superficie != "interna_corporativa" || datos.CuentaPrivilegiada {
		return politica, vecdomain.ErrAutorizacionDenegada
	}
	defer func() {
		resultado := vecdomain.ResultadoIntentoAuditoriaExito
		if fallo != nil || ctx.Err() != nil {
			resultado = vecdomain.ResultadoIntentoAuditoriaError
			if errors.Is(fallo, vecdomain.ErrAutorizacionDenegada) {
				resultado = vecdomain.ResultadoIntentoAuditoriaDenegado
			}
		}
		if err := c.auditar(ctx, orden, resultado); err != nil {
			slog.Error("bolsa politica cese: auditoria no confirmada", "causa", err)
			politica, fallo = bolsadomain.PoliticaCese{}, bolsaports.ErrConsultaPoliticaCeseNoDisponible
		} else if ctx.Err() != nil {
			politica, fallo = bolsadomain.PoliticaCese{}, ctx.Err()
		}
	}()
	p := c.preparador
	if p.fuenteRespuestas == nil {
		return politica, bolsaports.ErrConsultaPoliticaCeseNoDisponible
	}
	instantanea, err := p.fuenteRespuestas.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, datos.PerfilActivoRef)
	contrato := contratoLecturaBolsa{accion: bolsaports.AccionConsultarPoliticaCese, finalidad: bolsaports.FinalidadConsultarPoliticaCese,
		tipo: bolsaports.TipoRecursoPoliticaCese, campos: []string{bolsaports.CampoPoliticaCese}}
	if err != nil {
		slog.Error("bolsa politica cese: fuente de permisos no disponible", "causa", err)
		return politica, bolsaports.ErrConsultaPoliticaCeseNoDisponible
	}
	if instantanea.Validar() != nil || !concesionLecturaBolsaExacta(instantanea, contrato) ||
		instantanea.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		instantanea.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		instantanea.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" {
		return politica, vecdomain.ErrAutorizacionDenegada
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: bolsaports.RecursoPoliticaCeseRef, ModuloID: "bolsa", Tipo: contrato.tipo,
		Ambitos: map[string]string{"unidad_ref": p.soporte.unidadRef, "ambito_ref": p.soporte.ambitoRef}}
	ahora = time.Now().UTC().Truncate(time.Microsecond)
	evaluacion, err := vecdomain.EvaluarLecturaActualAutorizacionV3(orden.Vinculo, orden.ResultadoContexto,
		instantanea, contrato.accion, recurso, contrato.finalidad, contrato.campos, ahora)
	if err != nil || !evaluacion.Permitido || !slices.Equal(evaluacion.CamposPermitidos, contrato.campos) ||
		(len(evaluacion.Obligaciones) != 0 && !slices.Equal(evaluacion.Obligaciones, []string{"auditar"})) {
		return politica, vecdomain.ErrAutorizacionDenegada
	}
	politica, err = c.leer(ctx)
	if err != nil || politica.Validar() != nil {
		slog.Error("bolsa politica cese: lectura no disponible", "causa", err)
		return bolsadomain.PoliticaCese{}, bolsaports.ErrConsultaPoliticaCeseNoDisponible
	}
	ahora = time.Now().UTC().Truncate(time.Microsecond)
	if !evaluacion.ValidaHasta.After(ahora) || !orden.Vinculo.VigenteEn(ahora, orden.ResultadoContexto) {
		return bolsadomain.PoliticaCese{}, vecdomain.ErrAutorizacionDenegada
	}
	return politica, nil
}

func (c *consultaPoliticaCeseBolsaActual) auditar(ctx context.Context, consulta bolsaports.OrdenConsultaPoliticaCese, resultado vecdomain.ResultadoIntentoAuditoria) error {
	protegido, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelar()
	ref, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return err
	}
	correlacion, err := consulta.Correlacion.ValorCanonico()
	if err != nil {
		return err
	}
	orden, err := vecports.NuevaOrdenIntentoAuditoria(ref, consulta.ResultadoContexto, consulta.Vinculo, vecdomain.DatosIntentoAuditoria{
		Accion: "lectura_bolsas_rrhh_consultar", ModuloID: "bolsa", RecursoRef: recursoAuditoriaPoliticaCeseBolsa,
		FinalidadRef: "gestion_bolsa_rrhh", Resultado: resultado, Motivo: consulta.Motivo,
		Proceso: c.proceso, Canal: "interna_corporativa", CorrelacionRef: correlacion})
	if err != nil {
		return err
	}
	acuse, err := c.registrador.AppendIntentoAuditoria(protegido, orden)
	if err != nil {
		return err
	}
	return acuse.ValidarPara(orden)
}

func nuevaRutaPoliticaCeseBolsa(preparador *preparadorBorradorLlamamientoDesarrollo,
	leer func(context.Context) (bolsadomain.PoliticaCese, error), registrador vecports.RegistradorIntentosAuditoria, proceso string,
) (vechttp.RutaExacta, error) {
	if preparador == nil || leer == nil || registrador == nil || proceso != "vec-rrhh" {
		return vechttp.RutaExacta{}, bolsaports.ErrConsultaPoliticaCeseNoDisponible
	}
	h, err := bolsahttp.NuevoHandlerPoliticaCese(preparador, &consultaPoliticaCeseBolsaActual{preparador, leer, registrador, proceso})
	return vechttp.RutaExacta{Ruta: bolsahttp.RutaPoliticaCese, Manejador: h}, err
}
