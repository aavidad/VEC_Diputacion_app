package ports

import (
	"strconv"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// Estas acciones pertenecen a Documentos y no heredan permisos de Usuarios,
// carga documental, baremacion ni lectura general del almacen.
const (
	AccionNegocioEscribirImagenProcesada = "documentos.imagen.almacen.escribir"
	AccionNegocioPromoverImagenProcesada = "documentos.imagen.almacen.promover"
	AccionNegocioAbrirImagenPropiaActiva = "documentos.imagen.almacen.abrir_propia"
	AccionNegocioAbrirImagenAjenaActiva  = "documentos.imagen.almacen.abrir_ajena"

	PasoAlmacenEscribirImagenProcesada PasoOperacionAlmacen = "01_escribir_imagen_procesada"
	PasoAlmacenPromoverImagenProcesada PasoOperacionAlmacen = "01_promover_imagen_procesada"
	PasoAlmacenAbrirImagenActiva       PasoOperacionAlmacen = "01_abrir_imagen_activa"

	AtributoAlmacenImagenDocumentoRef      = "almacen_imagen_documento_ref"
	AtributoAlmacenImagenActorPersonaRef   = "almacen_imagen_actor_persona_ref"
	AtributoAlmacenImagenTitularPersonaRef = "almacen_imagen_titular_persona_ref"
	AtributoAlmacenImagenAudiencia         = "almacen_imagen_audiencia"
	AtributoAlmacenImagenFinalidad         = "almacen_imagen_finalidad"
	AtributoAlmacenImagenHuellaSHA256      = "almacen_imagen_huella_sha256"
	AtributoAlmacenImagenTamano            = "almacen_imagen_tamano"
	AtributoAlmacenImagenClaveIdempotencia = "almacen_imagen_clave_idempotencia"

	audienciaImagenPersonal = "portal_personal_autenticado"
	audienciaImagenInterna  = "portal_interno_autenticado"
	finalidadImagenPropia   = "finalidad:usuarios:imagen-propia:v1"
	finalidadImagenInterna  = "finalidad:usuarios:imagen-directorio-interno:v1"
	tamanoMaximoImagen      = 2 * 1024 * 1024
)

// ImagenAlmacenVinculada procede de la reserva y de la referencia activa
// resueltas por puertos confiables. El recurso que evalua V3 debe contener
// estos mismos valores antes de emitir la decision.
type ImagenAlmacenVinculada struct {
	DocumentoRef      string
	ActorPersonaRef   string
	TitularPersonaRef string
	Audiencia         string
	Finalidad         string
	HuellaSHA256      string
	Tamano            int64
	ClaveIdempotencia string
}

type vinculoImagenAlmacen = ImagenAlmacenVinculada

func clonarVinculoImagen(i *vinculoImagenAlmacen) *vinculoImagenAlmacen {
	if i == nil {
		return nil
	}
	copia := *i
	return &copia
}

func (i ImagenAlmacenVinculada) validoPara(accion string) bool {
	if !referenciaOpacaAlmacenValida(i.DocumentoRef, 128) ||
		!referenciaOpacaAlmacenValida(i.ActorPersonaRef, 128) ||
		!referenciaOpacaAlmacenValida(i.TitularPersonaRef, 128) ||
		!esSHA256Hexadecimal(i.HuellaSHA256) || i.Tamano < 1 || i.Tamano > tamanoMaximoImagen ||
		contieneComodinContextoAlmacen(i.DocumentoRef, i.ActorPersonaRef, i.TitularPersonaRef) {
		return false
	}
	propia := i.ActorPersonaRef == i.TitularPersonaRef
	if i.Audiencia != audienciaImagenPersonal && i.Audiencia != audienciaImagenInterna {
		return false
	}
	if propia {
		if i.Finalidad != finalidadImagenPropia {
			return false
		}
	} else if i.Audiencia != audienciaImagenInterna || i.Finalidad != finalidadImagenInterna {
		return false
	}
	switch accion {
	case AccionNegocioEscribirImagenProcesada, AccionNegocioPromoverImagenProcesada:
		return propia && referenciaOpacaAlmacenValida(i.ClaveIdempotencia, 512) &&
			!contieneComodinContextoAlmacen(i.ClaveIdempotencia)
	case AccionNegocioAbrirImagenPropiaActiva:
		return propia && i.ClaveIdempotencia == ""
	case AccionNegocioAbrirImagenAjenaActiva:
		return !propia && i.ClaveIdempotencia == ""
	default:
		return false
	}
}

func nuevoContextoImagenAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	imagen ImagenAlmacenVinculada,
	verificadaEn time.Time,
	accion string,
	paso PasoOperacionAlmacen,
	accionTecnica string,
	campos []string,
	requiereObjeto bool,
) (ContextoOperacionAlmacen, error) {
	if recurso.ModuloID != "documentos" || recurso.Tipo != "imagen_usuario" ||
		recurso.Referencia != imagen.DocumentoRef || decision.Finalidad != imagen.Finalidad ||
		vinculos.CargaRef != imagen.DocumentoRef || !imagen.validoPara(accion) {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	especificacion := especificacionAutorizacionAlmacen{
		accionNegocio: accion, camposExactos: campos,
		pasos:          []pasoPlanOperacionAlmacen{{referencia: paso, accion: accionTecnica}},
		requiereObjeto: requiereObjeto, imagen: clonarVinculoImagen(&imagen),
	}
	return nuevoContextoOperacionAlmacen(decision, recurso, vinculos, verificadaEn, especificacion)
}

func NuevoContextoEscribirImagenProcesadaAlmacen(
	decision domain.DecisionAutorizacion, recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen, imagen ImagenAlmacenVinculada, verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoImagenAlmacen(decision, recurso, vinculos, imagen, verificadaEn,
		AccionNegocioEscribirImagenProcesada, PasoAlmacenEscribirImagenProcesada,
		AccionAlmacenEscribir, []string{"contenido_png256", "objeto_cuarentena"}, false)
}

func NuevoContextoPromoverImagenProcesadaAlmacen(
	decision domain.DecisionAutorizacion, recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen, imagen ImagenAlmacenVinculada, verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoImagenAlmacen(decision, recurso, vinculos, imagen, verificadaEn,
		AccionNegocioPromoverImagenProcesada, PasoAlmacenPromoverImagenProcesada,
		AccionAlmacenPromover, []string{"objeto_admitido", "estado"}, true)
}

func NuevoContextoAbrirImagenActivaAlmacen(
	decision domain.DecisionAutorizacion, recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen, imagen ImagenAlmacenVinculada, verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	accion := AccionNegocioAbrirImagenPropiaActiva
	if imagen.ActorPersonaRef != imagen.TitularPersonaRef {
		accion = AccionNegocioAbrirImagenAjenaActiva
	}
	return nuevoContextoImagenAlmacen(decision, recurso, vinculos, imagen, verificadaEn,
		accion, PasoAlmacenAbrirImagenActiva, AccionAlmacenLeer,
		[]string{"contenido_png256"}, true)
}

func recursoVinculaImagenAlmacen(
	recurso domain.RecursoAutorizable,
	v VinculosOperacionAlmacen,
	especificacion especificacionAutorizacionAlmacen,
) bool {
	i := especificacion.imagen
	if i == nil {
		return true
	}
	a := recurso.Atributos
	esperados := 6 + 7
	if i.ClaveIdempotencia != "" {
		esperados++
	}
	if especificacion.requiereObjeto {
		esperados += 2
	}
	clave, existeClave := a[AtributoAlmacenImagenClaveIdempotencia]
	if len(a) != esperados || existeClave != (i.ClaveIdempotencia != "") ||
		(existeClave && clave != i.ClaveIdempotencia) ||
		a[AtributoAlmacenImagenDocumentoRef] != i.DocumentoRef ||
		a[AtributoAlmacenImagenActorPersonaRef] != i.ActorPersonaRef ||
		a[AtributoAlmacenImagenTitularPersonaRef] != i.TitularPersonaRef ||
		a[AtributoAlmacenImagenAudiencia] != i.Audiencia ||
		a[AtributoAlmacenImagenFinalidad] != i.Finalidad ||
		a[AtributoAlmacenImagenHuellaSHA256] != i.HuellaSHA256 ||
		a[AtributoAlmacenImagenTamano] != strconv.FormatInt(i.Tamano, 10) ||
		v.CargaRef != i.DocumentoRef {
		return false
	}
	return true
}

func (c ContextoOperacionAlmacen) validarEscrituraImagen(
	clave string, zona ZonaAlmacen, mime string, tamano int64, sha string,
) error {
	if c.validarEstructura() != nil {
		return errorAutorizacionAlmacen()
	}
	i := c.datos.imagen
	if i == nil {
		return nil
	}
	if c.datos.accionNegocio != AccionNegocioEscribirImagenProcesada ||
		clave != i.ClaveIdempotencia || zona != ZonaAlmacenCuarentena ||
		mime != "image/png" || tamano != i.Tamano || sha != i.HuellaSHA256 {
		return errorAutorizacionAlmacen()
	}
	return nil
}
