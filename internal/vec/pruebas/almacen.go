// Package pruebas contiene fabricas exclusivas para dobles automatizados. No
// debe importarse desde composicion productiva.
package pruebas

import (
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// NuevoContextoAlmacen crea una capacidad real de la lista positiva para una
// prueba externa al paquete ports. No existe modo generico: una accion sin
// fabrica productiva devuelve error. El objeto es obligatorio para lectura,
// promocion y retencion, y debe ser cero para el resto.
func NuevoContextoAlmacen(
	instante time.Time,
	sufijo string,
	accionTecnica string,
	objeto ports.ReferenciaObjetoAlmacen,
) (ports.ContextoOperacionAlmacen, error) {
	if instante.IsZero() || sufijo == "" || sufijo != strings.TrimSpace(sufijo) {
		return ports.ContextoOperacionAlmacen{}, ports.ErrAutorizacionAlmacenInvalida
	}
	instante = instante.UTC().Truncate(time.Microsecond)
	vinculo, err := NuevoVinculoGenerico(instante)
	if err != nil {
		return ports.ContextoOperacionAlmacen{}, err
	}
	datosActor, err := vinculo.Datos()
	if err != nil {
		return ports.ContextoOperacionAlmacen{}, err
	}

	accionNegocio, campos, requiereObjeto, pasoDerivado, err := especificacionAlmacenPrueba(accionTecnica)
	if err != nil {
		return ports.ContextoOperacionAlmacen{}, err
	}
	vinculos := ports.VinculosOperacionAlmacen{
		OperacionRef:        "operacion:prueba:" + sufijo,
		CargaRef:            "carga:prueba:" + sufijo,
		Clasificacion:       "datos_personales_alta",
		SujetoSeudonimoHMAC: "hmac-sha256:sujeto_prueba_v1:" + strings.Repeat("a", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_prueba_v1:" + strings.Repeat("b", 64),
		EfectoRef:           "efecto:prueba:" + sufijo,
	}
	if requiereObjeto {
		if objeto.Validar() != nil {
			return ports.ContextoOperacionAlmacen{}, ports.ErrAutorizacionAlmacenInvalida
		}
		vinculos.ObjetoVinculado = objeto
	} else if objeto != (ports.ReferenciaObjetoAlmacen{}) {
		return ports.ContextoOperacionAlmacen{}, ports.ErrAutorizacionAlmacenInvalida
	}
	atributos := map[string]string{
		ports.AtributoAlmacenOperacionRef:        vinculos.OperacionRef,
		ports.AtributoAlmacenCargaRef:            vinculos.CargaRef,
		ports.AtributoAlmacenClasificacion:       vinculos.Clasificacion,
		ports.AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		ports.AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		ports.AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
	}
	if requiereObjeto {
		atributos[ports.AtributoAlmacenObjetoRef] = objeto.Referencia
		atributos[ports.AtributoAlmacenObjetoVersion] = objeto.Version
	}
	recurso := domain.RecursoAutorizable{
		Referencia: "recurso:prueba:" + sufijo,
		ModuloID:   "bolsa",
		Tipo:       "documento_bolsa",
		Ambitos:    map[string]string{"sujeto_ref": datosActor.PrincipalID},
		Atributos:  atributos,
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return ports.ContextoOperacionAlmacen{}, err
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		return ports.ContextoOperacionAlmacen{}, err
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef:     "decision:almacen:prueba:" + sufijo,
		Concedida:       true,
		Codigo:          "concedida",
		PrincipalID:     datosActor.PrincipalID,
		PerfilActivoRef: datosActor.PerfilActivoRef,
		Accion:          accionNegocio, RecursoRef: recurso.Referencia, ModuloID: recurso.ModuloID,
		TipoRecurso: recurso.Tipo, ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad: "prueba_contrato_almacen", CorrelacionRef: "correlacion:prueba:" + sufijo,
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             "asignacion:prueba:" + sufijo, AsignacionHuellaSHA256: strings.Repeat("c", 64),
		VersionRolRef: "rol:prueba:v1", VersionRolHuellaSHA256: strings.Repeat("d", 64),
		ControlVigenciaVersionRolRef: "rol:prueba:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("e", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		GarantiaMinima:   domain.AuthAssuranceSubstantial,
		CamposPermitidos: append([]string(nil), campos...),
		EmitidaEn:        instante.Add(-time.Minute), ValidaHasta: instante.Add(time.Minute),
	}

	var contexto ports.ContextoOperacionAlmacen
	switch accionNegocio {
	case ports.AccionNegocioPrepararCargaDocumental:
		contexto, err = ports.NuevoContextoPrepararCargaDirectaAlmacen(decision, recurso, vinculos, instante)
	case ports.AccionNegocioConfirmarCargaDocumental:
		contexto, err = ports.NuevoContextoConfirmarCargaDirectaAlmacen(decision, recurso, vinculos, instante)
	case ports.AccionNegocioAnalizarCargaDocumental:
		contexto, err = ports.NuevoContextoAnalizarCargaDocumentalAlmacen(decision, recurso, vinculos, instante)
	case ports.AccionNegocioPromoverCargaDocumental:
		contexto, err = ports.NuevoContextoPromoverCargaDocumentalAlmacen(decision, recurso, vinculos, instante)
	case ports.AccionNegocioCustodiarDecisionBaremacion:
		contexto, err = ports.NuevoContextoCustodiarDecisionBaremacionAlmacen(decision, recurso, vinculos, instante)
	case ports.AccionNegocioCustodiarDocumentoFirmado:
		contexto, err = ports.NuevoContextoCustodiarDocumentoFirmadoAlmacen(decision, recurso, vinculos, instante)
	case ports.AccionNegocioRetenerDocumentoFirmado:
		contexto, err = ports.NuevoContextoRetenerDocumentoFirmadoAlmacen(decision, recurso, vinculos, instante)
	default:
		err = ports.ErrAutorizacionAlmacenInvalida
	}
	if err != nil {
		return ports.ContextoOperacionAlmacen{}, err
	}
	if pasoDerivado != "" {
		return contexto.DerivarPaso(pasoDerivado)
	}
	return contexto, nil
}

func especificacionAlmacenPrueba(accion string) (
	accionNegocio string,
	campos []string,
	requiereObjeto bool,
	pasoDerivado ports.PasoOperacionAlmacen,
	err error,
) {
	switch accion {
	case ports.AccionAlmacenPrepararCargaDirecta:
		return ports.AccionNegocioPrepararCargaDocumental,
			[]string{"clasificacion", "contenido", "huella_sha256", "mime", "tamano"}, false, "", nil
	case ports.AccionAlmacenAbandonarCargaDirecta:
		return ports.AccionNegocioPrepararCargaDocumental,
			[]string{"clasificacion", "contenido", "huella_sha256", "mime", "tamano"}, false,
			ports.PasoAlmacenAbandonarCargaDirecta, nil
	case ports.AccionAlmacenConfirmarCargaDirecta:
		return ports.AccionNegocioConfirmarCargaDocumental,
			[]string{"contenido_cuarentena", "estado"}, false, "", nil
	case ports.AccionAlmacenLeer:
		return ports.AccionNegocioAnalizarCargaDocumental,
			[]string{"analisis_seguridad", "estado"}, true, "", nil
	case ports.AccionAlmacenAnalizarContenido:
		return ports.AccionNegocioAnalizarCargaDocumental,
			[]string{"analisis_seguridad", "estado"}, true, ports.PasoAlmacenAnalizarContenido, nil
	case ports.AccionAlmacenPromover:
		return ports.AccionNegocioPromoverCargaDocumental,
			[]string{"contenido_admitido", "estado"}, true, "", nil
	case ports.AccionAlmacenEscribir:
		return ports.AccionNegocioCustodiarDecisionBaremacion,
			[]string{"documento_custodiado", "evidencia_custodia"}, false, "", nil
	case ports.AccionAlmacenAplicarRetencion:
		return ports.AccionNegocioRetenerDocumentoFirmado,
			[]string{"documento_firmado.retencion", "evidencia_retencion"}, true, "", nil
	default:
		return "", nil, false, "", errors.Join(
			domain.ErrAutorizacionDenegada,
			ports.ErrAutorizacionAlmacenInvalida,
		)
	}
}
