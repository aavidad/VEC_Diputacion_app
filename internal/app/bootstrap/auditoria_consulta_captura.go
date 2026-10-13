package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/vec/auditoria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type capturadorAuditoriaLecturaRRHH struct{ fuente vecports.FuenteAutorizacion }

func (c capturadorAuditoriaLecturaRRHH) CapturarLecturaAuditoria(ctx context.Context, identidad auditoria.ContextoConsulta,
	f auditoria.Filtro, operacion string, busqueda auditoria.Busqueda,
) (auditoria.CapturaLecturaSesion, error) {
	vacia := auditoria.CapturaLecturaSesion{}
	if ctx == nil || ctx.Err() != nil || dependenciaAuditoriaConsultaNula(c.fuente) ||
		(operacion != "opciones" && operacion != "buscar" && operacion != "consultar") {
		return vacia, auditoria.ErrDenegada
	}
	datos, err := identidad.Vinculo.Datos()
	if err != nil || datos.Superficie != vecdomain.SuperficieAutenticacionInternaCorporativaV1 || datos.CuentaPrivilegiada {
		return vacia, auditoria.ErrDenegada
	}
	i, err := c.fuente.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, datos.PerfilActivoRef)
	if err != nil {
		return vacia, auditoria.ErrNoDisponible
	}
	if i.Validar() != nil || i.VersionRol.RolID != "consulta_auditoria_rrhh" || len(i.AsignacionPerfil.Ambitos) != 3 ||
		i.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		i.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" {
		return vacia, auditoria.ErrDenegada
	}
	ambitos := make(map[string]string, 3)
	for _, alcance := range i.AsignacionPerfil.Ambitos {
		if (alcance.Clave != "organizacion_ref" && alcance.Clave != "unidad_ref" && alcance.Clave != "ambito_ref") || len(alcance.Valores) != 1 || ambitos[alcance.Clave] != "" {
			return vacia, auditoria.ErrDenegada
		}
		ambitos[alcance.Clave] = alcance.Valores[0]
	}
	organizacion := ambitos["organizacion_ref"]
	canon, err := auditoria.CanonicoFiltro(f)
	if err != nil {
		return vacia, err
	}
	if operacion == "buscar" {
		canon += "\nvec.auditoria.busqueda.v1\n" + busqueda.Texto + "\n" + busqueda.DespuesRef
	}
	huella := sha256.Sum256([]byte(canon))
	recurso := vecdomain.RecursoAutorizable{Referencia: f.ExpedienteRef, ModuloID: auditoria.ModuloAutorizacion,
		Tipo: auditoria.TipoRecurso, Ambitos: ambitos,
		Atributos: map[string]string{"filtro_sha256": hex.EncodeToString(huella[:])}}
	e, err := vecdomain.EvaluarLecturaActualAutorizacionV3(identidad.Vinculo, identidad.Resultado, i,
		auditoria.AccionConsultar, recurso, f.FinalidadRef, auditoria.CamposLecturaSesion(), time.Now().UTC().Truncate(time.Microsecond))
	if err != nil || !e.Permitido || !slices.Equal(e.CamposPermitidos, auditoria.CamposLecturaSesion()) ||
		(len(e.Obligaciones) != 0 && !slices.Equal(e.Obligaciones, []string{"auditar"})) {
		return vacia, auditoria.ErrDenegada
	}
	antes := ""
	if !f.Antes.OcurridoEn.IsZero() {
		antes = f.Antes.OcurridoEn.UTC().Format("2006-01-02T15:04:05.000000Z")
	}
	filtro := map[string]any{"fuente": f.Fuente, "recurso_ref": f.ExpedienteRef, "actor_ref": f.ActorRef,
		"desde": f.Desde.Format("2006-01-02T15:04:05.000000Z"), "hasta": f.Hasta.Format("2006-01-02T15:04:05.000000Z"),
		"limite": strconv.FormatUint(uint64(f.Limite), 10), "antes_en": antes, "antes_fuente": f.Antes.Fuente, "antes_id": f.Antes.ID,
		"finalidad_ref": f.FinalidadRef, "motivo_ref": f.MotivoRef, "motivo_consulta": f.MotivoConsulta,
		"autor_lectura_ref": f.AutorLecturaRef, "busqueda": busqueda.Texto, "despues_ref": busqueda.DespuesRef}
	captura, err := componerCapturaLecturaActual(ctx, identidad.Vinculo, identidad.Resultado, e,
		auditoria.AccionConsultar, auditoria.ModuloAutorizacion, f.FinalidadRef, f.MotivoRef, f.ExpedienteRef,
		map[string]any{"operacion": operacion, "organizacion_ref": organizacion, "fuente": f.Fuente,
			"unidad_ref": ambitos["unidad_ref"], "ambito_ref": ambitos["ambito_ref"],
			"filtro_sha256": hex.EncodeToString(huella[:]), "filtro": filtro,
			"motivo_catalogo_ref": identidad.Motivo.Referencia(), "motivo_catalogo_sha256": identidad.Motivo.CatalogoHuellaSHA256,
			"motivo_consulta": f.MotivoConsulta})
	if err != nil {
		return vacia, err
	}
	return auditoria.CapturaLecturaSesion{ContextoCanonico: captura.contexto, VinculoCanonico: captura.vinculo,
		Orden: captura.orden, OrganizacionRef: organizacion}, nil
}
