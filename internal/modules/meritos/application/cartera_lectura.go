package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const esquemaOrdenCartera = "vec.meritos.cartera.propia.v1"
const rolCarteraPersonal = "persona_meritos_cartera"

// SolicitudCartera recibe la identidad exclusivamente de la frontera mTLS.
// PersonaRef y el perfil nunca proceden de la URL ni del cuerpo HTTP.
type SolicitudCartera struct {
	ResultadoContexto vec.ResultadoContextoActorRegistradoV2
	Vinculo           vec.VinculoAutenticacionActorV2
	Correlacion       vec.ReferenciaCorrelacionAutorizacionV2
	Filtro            string
	Idioma            string
	Limite            int
	Cursor            string
}

type ServicioCartera struct {
	lector   ports.LectorCartera
	auditor  ports.AuditorIntentosCartera
	fuente   vecports.FuenteAutorizacion
	reloj    vecports.Reloj
	cursores *SelladorCursorCartera
}

func NuevoServicioCartera(lector ports.LectorCartera, auditor ports.AuditorIntentosCartera,
	fuente vecports.FuenteAutorizacion, reloj vecports.Reloj) (*ServicioCartera, error) {
	if nulo(lector) || nulo(auditor) || nulo(fuente) || nulo(reloj) {
		return nil, ports.ErrCarteraNoDisponible
	}
	cursores, err := NuevoSelladorCursorCartera()
	if err != nil {
		return nil, errors.Join(ports.ErrCarteraNoDisponible, err)
	}
	return &ServicioCartera{lector, auditor, fuente, reloj, cursores}, nil
}

// Consultar evalúa el permiso actual de cada página. La lectura y su asiento
// positivo se confirman en PostgreSQL antes de devolver una sola ficha.
func (s *ServicioCartera) Consultar(ctx context.Context, q SolicitudCartera) (ports.PaginaCartera, error) {
	vacia := ports.PaginaCartera{}
	if ctx == nil || s == nil || nulo(s.lector) || nulo(s.auditor) || nulo(s.fuente) ||
		nulo(s.reloj) || s.cursores == nil {
		return vacia, ports.ErrCarteraNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	datos, err := q.Vinculo.Datos()
	if err != nil || q.ResultadoContexto.Validar() != nil ||
		q.Vinculo.ValidarPara(q.ResultadoContexto) != nil || q.Correlacion.Validar() != nil ||
		!q.Vinculo.VigenteEn(ahora, q.ResultadoContexto) ||
		datos.Superficie != vec.SuperficieAutenticacionExternaPersonalV1 ||
		datos.GarantiaObservada != vec.AuthAssuranceHigh || datos.CuentaPrivilegiada ||
		datos.PrincipalID != q.ResultadoContexto.Contexto.PersonaRef ||
		!domain.ReferenciaValida(datos.PrincipalID) {
		return vacia, ports.ErrCarteraDenegada
	}
	persona := datos.PrincipalID
	correlacion, err := q.Correlacion.ValorCanonico()
	if err != nil {
		return vacia, ports.ErrCarteraDenegada
	}
	if !filtroCarteraValido(q.Filtro) || !idiomaCarteraValido(q.Idioma) ||
		q.Limite < 1 || q.Limite > 50 || len(q.Cursor) > 2048 {
		return vacia, s.registrarFallo(ctx, q, datos, correlacion, "denegado")
	}
	corte, ultimo := ahora, ""
	if q.Cursor != "" {
		corte, ultimo, err = s.cursores.Abrir(q.Cursor, persona, q.Filtro, q.Idioma, q.Limite, ahora)
		if err != nil {
			return vacia, s.registrarFallo(ctx, q, datos, correlacion, "denegado")
		}
	}
	recurso := vec.RecursoAutorizable{Referencia: "cartera:" + persona,
		ModuloID: ports.ModuloCartera, Tipo: ports.TipoRecursoCartera,
		Ambitos:   map[string]string{"persona_ref": persona},
		Atributos: map[string]string{"propiedad": "persona"}}
	if recurso.Validar() != nil {
		return vacia, s.registrarFallo(ctx, q, datos, correlacion, "error")
	}
	instantanea, err := s.fuente.ObtenerInstantaneaAutorizacion(ctx, persona, datos.PerfilActivoRef)
	if err != nil || ctx.Err() != nil || instantanea.Validar() != nil {
		return vacia, s.registrarFallo(ctx, q, datos, correlacion, "error")
	}
	if !concesionCarteraExacta(instantanea) {
		return vacia, s.registrarFallo(ctx, q, datos, correlacion, "denegado")
	}
	ahora = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	evaluacion, err := vec.EvaluarLecturaActualAutorizacionV3(q.Vinculo, q.ResultadoContexto,
		instantanea, ports.AccionConsultarCartera, recurso, ports.FinalidadCartera,
		ports.CamposConsultaCartera(), ahora)
	if err != nil || !evaluacion.Permitido ||
		!slices.Equal(evaluacion.CamposPermitidos, ports.CamposConsultaCartera()) ||
		!slices.Equal(evaluacion.Obligaciones, []string{"auditar"}) {
		return vacia, s.registrarFallo(ctx, q, datos, correlacion, "denegado")
	}
	captura, err := capturarConsultaCartera(q, datos, evaluacion, corte, ultimo, ahora, correlacion)
	if err != nil {
		return vacia, s.registrarFallo(ctx, q, datos, correlacion, "error")
	}
	pagina, err := s.lector.LeerCartera(ctx, captura)
	if err != nil {
		resultado := "error"
		if errors.Is(err, vec.ErrAutorizacionDenegada) || errors.Is(err, ports.ErrCarteraDenegada) {
			resultado = "denegado"
		}
		return vacia, s.registrarFallo(ctx, q, datos, correlacion, resultado)
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	if pagina.Validar(corte, correlacion, q.Limite, q.Filtro, ultimo) != nil {
		return vacia, ports.ErrCarteraNoDisponible
	}
	salida := ports.PaginaCartera{Hechos: copiarFichasCartera(pagina.Hechos),
		MotivosRetirada: slices.Clone(pagina.MotivosRetirada), Total: pagina.Total,
		Filtro: q.Filtro, CorteEn: pagina.CorteEn, ReciboAcceso: pagina.ReciboAcceso}
	if pagina.CursorSiguienteHechoRef != "" {
		salida.CursorSiguiente, err = s.cursores.Sellar(persona, q.Filtro, q.Idioma, q.Limite,
			corte, pagina.CursorSiguienteHechoRef, s.reloj.Ahora().UTC())
		if err != nil {
			return vacia, ports.ErrCarteraNoDisponible
		}
	}
	return salida, nil
}

func concesionCarteraExacta(i vec.InstantaneaAutorizacion) bool {
	if i.VersionRol.RolID != rolCarteraPersonal ||
		i.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		i.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		i.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" {
		return false
	}
	n := 0
	for _, c := range i.VersionRol.Concesiones {
		if c.Accion != ports.AccionConsultarCartera {
			continue
		}
		n++
		campos := slices.Clone(c.CamposPermitidos)
		slices.Sort(campos)
		if c.ModuloID != ports.ModuloCartera || c.TipoRecurso != ports.TipoRecursoCartera ||
			!slices.Equal(c.Finalidades, []string{ports.FinalidadCartera}) ||
			!slices.Equal(campos, ports.CamposConsultaCartera()) ||
			c.GarantiaMinima != vec.AuthAssuranceHigh ||
			!slices.Equal(c.Obligaciones, []string{"auditar"}) {
			return false
		}
	}
	return n == 1
}

func copiarFichasCartera(entrada []ports.FichaCartera) []ports.FichaCartera {
	salida := slices.Clone(entrada)
	for i := range salida {
		if salida[i].Retirada != nil {
			r := *salida[i].Retirada
			salida[i].Retirada = &r
		}
	}
	return salida
}

func (s *ServicioCartera) registrarFallo(ctx context.Context, q SolicitudCartera,
	datos vec.DatosVinculoAutenticacionActorV2, correlacion, resultado string) error {
	captura, err := capturarIntentoCartera(q, datos, correlacion, resultado)
	if err != nil {
		return ports.ErrCarteraNoDisponible
	}
	auditCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancelar()
	if err := s.auditor.RegistrarIntentoCartera(auditCtx, captura); err != nil {
		return ports.ErrCarteraNoDisponible
	}
	if resultado == "denegado" {
		return ports.ErrCarteraDenegada
	}
	return ports.ErrCarteraNoDisponible
}

func identificadorRegistroCartera(intento bool) (string, error) {
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return "", err
	}
	prefijo := "lectura_"
	if intento {
		prefijo = "intento_"
	}
	return prefijo + hex.EncodeToString(aleatorio[:]), nil
}

func capturaBaseCartera(q SolicitudCartera, datos vec.DatosVinculoAutenticacionActorV2,
	correlacion string, intento bool) (map[string]any, []byte, []byte, error) {
	id, err := identificadorRegistroCartera(intento)
	if err != nil {
		return nil, nil, nil, err
	}
	contexto := slices.Clone(q.ResultadoContexto.RepresentacionCanonica)
	vinculo, err := json.Marshal(q.Vinculo)
	if err != nil || len(contexto) == 0 {
		return nil, nil, nil, ports.ErrCarteraNoDisponible
	}
	huella := sha256.Sum256(contexto)
	base := map[string]any{
		"esquema": esquemaOrdenCartera, "persona_ref": datos.PrincipalID,
		"intento_ref": id, "registro_contexto_ref": datos.RegistroContextoRef,
		"contexto_sha256":    hex.EncodeToString(huella[:]),
		"procedencia_sha256": datos.ManifiestoProcedenciaHuellaSHA256,
		"autenticacion_ref":  datos.AutenticacionRef, "sesion_ref": datos.SesionRef,
		"autenticacion_sha256": datos.AutenticacionHuellaSHA256,
		"accion":               ports.AccionConsultarCartera, "modulo_id": ports.ModuloCartera,
		"tipo_recurso":  ports.TipoRecursoCartera,
		"recurso_ref":   "cartera:" + datos.PrincipalID,
		"finalidad_ref": ports.FinalidadCartera, "proceso": "vec-server",
		"canal":           string(vec.SuperficieAutenticacionExternaPersonalV1),
		"correlacion_ref": correlacion,
	}
	return base, contexto, vinculo, nil
}

func capturarIntentoCartera(q SolicitudCartera, datos vec.DatosVinculoAutenticacionActorV2,
	correlacion, resultado string) (ports.CapturaLecturaCartera, error) {
	base, contexto, vinculo, err := capturaBaseCartera(q, datos, correlacion, true)
	if err != nil {
		return ports.CapturaLecturaCartera{}, err
	}
	base["resultado"] = resultado
	base["motivo_ref"] = "lectura_cartera_meritos_" + resultado
	recurso := sha256.Sum256([]byte(datos.PrincipalID))
	base["recurso_ref"] = "cartera:" + hex.EncodeToString(recurso[:])
	b, err := json.Marshal(base)
	if err != nil {
		return ports.CapturaLecturaCartera{}, err
	}
	return ports.NuevaCapturaLecturaCartera(datos.PrincipalID, correlacion,
		contexto, vinculo, b, datos.SesionValidaHasta)
}

func capturarConsultaCartera(q SolicitudCartera, datos vec.DatosVinculoAutenticacionActorV2,
	e vec.ResultadoLecturaActualAutorizacionV3, corte time.Time, ultimo string,
	ahora time.Time, correlacion string) (ports.CapturaLecturaCartera, error) {
	base, contexto, vinculo, err := capturaBaseCartera(q, datos, correlacion, false)
	if err != nil {
		return ports.CapturaLecturaCartera{}, err
	}
	if !e.ValidaHasta.After(ahora) || !datos.SesionValidaHasta.After(ahora) {
		return ports.CapturaLecturaCartera{}, ports.ErrCarteraDenegada
	}
	base["limite"], base["filtro"], base["idioma"] = q.Limite, q.Filtro, q.Idioma
	base["cursor_hecho_ref"], base["corte_en"] = ultimo, corte.UTC().Truncate(time.Microsecond)
	base["campos_permitidos"] = e.CamposPermitidos
	base["resultado"], base["motivo_ref"] = "obtenida", "lectura_cartera_meritos_correcta"
	base["lectura_revision_permisos"] = strconv.FormatUint(e.RevisionPermisos, 10)
	base["lectura_instantanea_sha256"] = e.HuellaInstantaneaSHA256
	base["perfil_activo_ref"] = datos.PerfilActivoRef
	base["asignacion_ref"], base["asignacion_version"] = e.AsignacionRef, e.AsignacionVersion
	base["asignacion_huella_sha256"] = e.AsignacionHuellaSHA256
	base["version_rol_ref"], base["version_rol_version"] = e.VersionRolRef, e.VersionRolVersion
	base["version_rol_huella_sha256"] = e.VersionRolHuellaSHA256
	base["control_version_rol_ref"] = e.ControlVersionRolRef
	base["revision_control_version_rol"] = e.RevisionControlVersionRol
	base["control_version_rol_huella_sha256"] = e.ControlVersionRolHuellaSHA256
	base["revision_catalogo_politicas"] = e.RevisionCatalogoPoliticas
	base["catalogo_politicas_huella_sha256"] = e.CatalogoPoliticasHuellaSHA256
	base["politicas_evaluadas"] = e.PoliticasEvaluadas
	base["emitida_en"], base["valida_hasta"] = e.EvaluadaEn, e.ValidaHasta
	b, err := json.Marshal(base)
	if err != nil {
		return ports.CapturaLecturaCartera{}, err
	}
	hasta := e.ValidaHasta
	if datos.SesionValidaHasta.Before(hasta) {
		hasta = datos.SesionValidaHasta
	}
	return ports.NuevaCapturaLecturaCartera(datos.PrincipalID, correlacion,
		contexto, vinculo, b, hasta)
}
