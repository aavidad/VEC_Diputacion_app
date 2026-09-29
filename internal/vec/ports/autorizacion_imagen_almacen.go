package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// Estas acciones pertenecen a Documentos y no heredan permisos de Usuarios,
// carga documental, baremacion ni lectura general del almacen.
const (
	EsquemaContextoImagenAlmacenV3       = "vec.almacen.imagen.contexto-operacion.v3"
	AccionNegocioEscribirImagenProcesada = "documentos.imagen.almacen.escribir"
	AccionNegocioPromoverImagenProcesada = "documentos.imagen.almacen.promover"
	AccionNegocioAbrirImagenPropiaActiva = "documentos.imagen.almacen.abrir_propia"
	AccionNegocioAbrirImagenAjenaActiva  = "documentos.imagen.almacen.abrir_ajena"

	PasoAlmacenEscribirImagenProcesada PasoOperacionAlmacen = "01_escribir_imagen_procesada"
	PasoAlmacenPromoverImagenProcesada PasoOperacionAlmacen = "01_promover_imagen_procesada"
	PasoAlmacenAbrirImagenActiva       PasoOperacionAlmacen = "01_abrir_imagen_activa"

	AtributoAlmacenImagenDocumentoRef         = "almacen_imagen_documento_ref"
	AtributoAlmacenImagenActorPersonaRef      = "almacen_imagen_actor_persona_ref"
	AtributoAlmacenImagenTitularPersonaRef    = "almacen_imagen_titular_persona_ref"
	AtributoAlmacenImagenAudiencia            = "almacen_imagen_audiencia"
	AtributoAlmacenImagenFinalidad            = "almacen_imagen_finalidad"
	AtributoAlmacenImagenHuellaSHA256         = "almacen_imagen_huella_sha256"
	AtributoAlmacenImagenTamano               = "almacen_imagen_tamano"
	AtributoAlmacenImagenClaveIdempotencia    = "almacen_imagen_clave_idempotencia"
	AtributoAlmacenImagenEvidenciaAnalisisRef = "almacen_imagen_evidencia_analisis_ref"

	AudienciaConsumoEscribirImagenProcesadaV3 = "vec_documentos.imagen.almacen.escribir.v1"
	AudienciaConsumoPromoverImagenProcesadaV3 = "vec_documentos.imagen.almacen.promover.v1"
	AudienciaConsumoAbrirImagenPropiaV3       = "vec_documentos.imagen.almacen.abrir_propia.v1"
	AudienciaConsumoAbrirImagenAjenaV3        = "vec_documentos.imagen.almacen.abrir_ajena.v1"

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
	DocumentoRef         string
	ActorPersonaRef      string
	TitularPersonaRef    string
	Audiencia            string
	Finalidad            string
	HuellaSHA256         string
	Tamano               int64
	ClaveIdempotencia    string
	EvidenciaAnalisisRef string
}

// La preimagen V3 completa evita degradar una concesion nominal al formato
// historico V1/V2. Documentos debe entregar el material a su consumidor SQL
// en la misma transaccion que estado, auditoria y outbox.
type AutorizacionImagenAlmacenV3 struct {
	Solicitud     domain.SolicitudAutorizacionLigadaV3
	Decision      domain.DecisionAutorizacionLigadaV3
	Confirmacion  ConfirmacionRegistroConcesionAutorizacionLigadaV3
	ContextoActor domain.ResultadoContextoActorRegistradoV2
	Motivo        domain.ReferenciaEntradaCatalogo
	Material      ExportacionMaterialConsumoAutorizacionAtestadaV3
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
		if !propia || !referenciaOpacaAlmacenValida(i.ClaveIdempotencia, 512) ||
			contieneComodinContextoAlmacen(i.ClaveIdempotencia) {
			return false
		}
		if accion == AccionNegocioPromoverImagenProcesada {
			return referenciaOpacaAlmacenValida(i.EvidenciaAnalisisRef, 512) &&
				!contieneComodinContextoAlmacen(i.EvidenciaAnalisisRef)
		}
		return i.EvidenciaAnalisisRef == ""
	case AccionNegocioAbrirImagenPropiaActiva:
		return propia && i.ClaveIdempotencia == "" && i.EvidenciaAnalisisRef == ""
	case AccionNegocioAbrirImagenAjenaActiva:
		return !propia && i.ClaveIdempotencia == "" && i.EvidenciaAnalisisRef == ""
	default:
		return false
	}
}

func nuevoContextoImagenAlmacen(
	autorizacion AutorizacionImagenAlmacenV3,
	vinculos VinculosOperacionAlmacen,
	imagen ImagenAlmacenVinculada,
	verificadaEn time.Time,
	accion string,
	paso PasoOperacionAlmacen,
	accionTecnica string,
	campos []string,
	requiereObjeto bool,
) (ContextoOperacionAlmacen, error) {
	datosSolicitud, err := autorizacion.Solicitud.Datos()
	if err != nil {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	recurso := datosSolicitud.Recurso
	resumen := autorizacion.Material.ResumenCapacidad()
	audiencia := audienciaConsumoImagenV3(accion)
	vinculo, errVinculo := datosSolicitud.VinculoAutenticacionActor.Datos()
	correlacion, errCorrelacion := datosSolicitud.Correlacion.ValorCanonico()
	huellaRecurso, errHuella := recurso.HuellaContextoAutorizacionSHA256()
	concedida, _, errDecision := autorizacion.Decision.Resultado()
	emitida, validaHasta, errVentana := autorizacion.Decision.VentanaValidez()
	if recurso.ModuloID != "documentos" || recurso.Tipo != "imagen_usuario" ||
		recurso.Referencia != imagen.DocumentoRef || datosSolicitud.Finalidad != imagen.Finalidad ||
		datosSolicitud.Accion != accion || vinculos.CargaRef != imagen.DocumentoRef ||
		!imagen.validoPara(accion) || errVinculo != nil || errCorrelacion != nil || errHuella != nil ||
		errDecision != nil || errVentana != nil || !concedida || vinculo.PrincipalID != imagen.ActorPersonaRef ||
		resumen.Operacion() != accion || resumen.EfectoRef() != recurso.Referencia ||
		resumen.EfectoHuellaSHA256() != huellaRecurso || resumen.AudienciaConsumo() != audiencia ||
		verificadaEn.Before(emitida) || verificadaEn.Before(resumen.EmitidaEn()) ||
		!verificadaEn.Before(validaHasta) || !verificadaEn.Before(resumen.ExpiraEn()) ||
		!MaterialAtestadoLigadoV3(autorizacion.Solicitud, autorizacion.Decision,
			autorizacion.Confirmacion, autorizacion.ContextoActor, autorizacion.Motivo,
			autorizacion.Material, audiencia) ||
		!camposDecisionImagenV3(autorizacion.Decision, campos) {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	especificacion := especificacionAutorizacionAlmacen{
		accionNegocio: accion, camposExactos: campos,
		pasos:          []pasoPlanOperacionAlmacen{{referencia: paso, accion: accionTecnica}},
		requiereObjeto: requiereObjeto, imagen: clonarVinculoImagen(&imagen),
	}
	if !especificacion.valida() || !vinculos.validosPara(especificacion) ||
		!recursoVinculaOperacionAlmacen(recurso, vinculos, especificacion) {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	material, err := clonarMaterialImagenV3(autorizacion.Material)
	if err != nil {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	pasos := clonarPasosOperacionAlmacen(especificacion.pasos)
	d := &datosContextoOperacionAlmacen{
		esquema:      EsquemaContextoImagenAlmacenV3,
		operacionRef: vinculos.OperacionRef, correlacionRef: correlacion,
		autorizacionRef: resumen.DecisionRef(), finalidad: imagen.Finalidad,
		clasificacion: vinculos.Clasificacion, accionNegocio: accion,
		accionTecnica: accionTecnica, cargaRef: vinculos.CargaRef,
		sujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		recursoRef:          recurso.Referencia, moduloID: recurso.ModuloID, tipoRecurso: recurso.Tipo,
		huellaRecursoSHA256: huellaRecurso, huellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		efectoRef: vinculos.EfectoRef,
		pasoRef:   paso, objetoVinculado: vinculos.ObjetoVinculado,
		huellaDecisionSHA256: resumen.DecisionHuellaSHA256(),
		verificadaEn:         verificadaEn, validaHasta: resumen.ExpiraEn(), pasos: pasos,
		imagen: clonarVinculoImagen(&imagen), materialImagenV3: &material,
	}
	d.huellaPlanEfectoSHA256 = huellaPlanImagenV3(d)
	c := ContextoOperacionAlmacen{datos: d}
	if c.validarEstructura() != nil {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	return c, nil
}

func NuevoContextoEscribirImagenProcesadaAlmacen(
	autorizacion AutorizacionImagenAlmacenV3,
	vinculos VinculosOperacionAlmacen, imagen ImagenAlmacenVinculada, verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoImagenAlmacen(autorizacion, vinculos, imagen, verificadaEn,
		AccionNegocioEscribirImagenProcesada, PasoAlmacenEscribirImagenProcesada,
		AccionAlmacenEscribir, []string{"contenido_png256", "objeto_cuarentena"}, false)
}

func NuevoContextoPromoverImagenProcesadaAlmacen(
	autorizacion AutorizacionImagenAlmacenV3,
	vinculos VinculosOperacionAlmacen, imagen ImagenAlmacenVinculada, verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoImagenAlmacen(autorizacion, vinculos, imagen, verificadaEn,
		AccionNegocioPromoverImagenProcesada, PasoAlmacenPromoverImagenProcesada,
		AccionAlmacenPromover, []string{"objeto_admitido", "estado"}, true)
}

func NuevoContextoAbrirImagenActivaAlmacen(
	autorizacion AutorizacionImagenAlmacenV3,
	vinculos VinculosOperacionAlmacen, imagen ImagenAlmacenVinculada, verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	accion := AccionNegocioAbrirImagenPropiaActiva
	if imagen.ActorPersonaRef != imagen.TitularPersonaRef {
		accion = AccionNegocioAbrirImagenAjenaActiva
	}
	return nuevoContextoImagenAlmacen(autorizacion, vinculos, imagen, verificadaEn,
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
	if i.EvidenciaAnalisisRef != "" {
		esperados++
	}
	if especificacion.requiereObjeto {
		esperados += 2
	}
	clave, existeClave := a[AtributoAlmacenImagenClaveIdempotencia]
	evidencia, existeEvidencia := a[AtributoAlmacenImagenEvidenciaAnalisisRef]
	if len(a) != esperados || existeClave != (i.ClaveIdempotencia != "") ||
		(existeClave && clave != i.ClaveIdempotencia) ||
		existeEvidencia != (i.EvidenciaAnalisisRef != "") ||
		(existeEvidencia && evidencia != i.EvidenciaAnalisisRef) ||
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

func audienciaConsumoImagenV3(accion string) string {
	switch accion {
	case AccionNegocioEscribirImagenProcesada:
		return AudienciaConsumoEscribirImagenProcesadaV3
	case AccionNegocioPromoverImagenProcesada:
		return AudienciaConsumoPromoverImagenProcesadaV3
	case AccionNegocioAbrirImagenPropiaActiva:
		return AudienciaConsumoAbrirImagenPropiaV3
	case AccionNegocioAbrirImagenAjenaActiva:
		return AudienciaConsumoAbrirImagenAjenaV3
	}
	return ""
}

// Solo inspecciona los campos del documento canónico ya verificado. Nunca
// reconstruye una capacidad a partir de JSON ni sustituye el consumo V3.
func camposDecisionImagenV3(decision domain.DecisionAutorizacionLigadaV3, esperados []string) bool {
	canon, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		return false
	}
	var campos struct {
		CamposPermitidos []string `json:"campos_permitidos"`
		Obligaciones     []string `json:"obligaciones"`
	}
	if json.Unmarshal(canon, &campos) != nil || len(campos.Obligaciones) != 0 {
		return false
	}
	return camposAutorizacionExactos(campos.CamposPermitidos, esperados)
}

func clonarMaterialImagenV3(origen ExportacionMaterialConsumoAutorizacionAtestadaV3) (
	ExportacionMaterialConsumoAutorizacionAtestadaV3, error,
) {
	if origen.ValidarEstructura() != nil {
		return ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errorAutorizacionAlmacen()
	}
	return NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		origen.CapacidadCanonica(), origen.ResumenCapacidad(), origen.DecisionCanonica(),
		origen.MotivoCanonico(), origen.ContextoActorCanonico(), origen.PersonaVersion(),
		origen.PerfilVersion(), origen.PayloadVECAD3(), origen.SobreCOSESign1(),
		origen.EvidenciaVerificacion(), origen.RaizPublicaSPKI(),
	)
}

// MaterialAutorizacionImagenV3 entrega una copia defensiva al consumidor
// durable de Documentos. El contexto de imagen no expone evidencia V1.
func (c ContextoOperacionAlmacen) MaterialAutorizacionImagenV3() (
	ExportacionMaterialConsumoAutorizacionAtestadaV3, error,
) {
	if c.validarEstructura() != nil || c.datos.imagen == nil || c.datos.materialImagenV3 == nil {
		return ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errorAutorizacionAlmacen()
	}
	return clonarMaterialImagenV3(*c.datos.materialImagenV3)
}

func huellaPlanImagenV3(d *datosContextoOperacionAlmacen) string {
	if d == nil || d.imagen == nil || d.materialImagenV3 == nil {
		return ""
	}
	huellaMaterial, err := d.materialImagenV3.HuellaConjuntoSHA256()
	if err != nil {
		return ""
	}
	i := d.imagen
	valores := []string{
		d.esquema, huellaMaterial, d.autorizacionRef, d.huellaDecisionSHA256,
		d.operacionRef, d.correlacionRef, d.accionNegocio, d.recursoRef,
		d.huellaRecursoSHA256, d.finalidad, d.clasificacion, d.cargaRef,
		d.sujetoSeudonimoHMAC, d.huellaSolicitudHMAC, d.efectoRef,
		d.objetoVinculado.Referencia, d.objetoVinculado.Version,
		i.DocumentoRef, i.ActorPersonaRef, i.TitularPersonaRef, i.Audiencia,
		i.Finalidad, i.HuellaSHA256, strconv.FormatInt(i.Tamano, 10),
		i.ClaveIdempotencia, i.EvidenciaAnalisisRef,
	}
	for _, paso := range d.pasos {
		valores = append(valores, string(paso.referencia), paso.accion)
	}
	var canon strings.Builder
	for _, valor := range valores {
		canon.WriteString(strconv.Itoa(len(valor)))
		canon.WriteByte(':')
		canon.WriteString(valor)
		canon.WriteByte('\n')
	}
	suma := sha256.Sum256([]byte(canon.String()))
	return hex.EncodeToString(suma[:])
}

func (c ContextoOperacionAlmacen) validarEstructuraImagenV3() error {
	d := c.datos
	if d == nil || d.esquema != EsquemaContextoImagenAlmacenV3 ||
		d.imagen == nil || d.materialImagenV3 == nil ||
		d.materialImagenV3.ValidarEstructura() != nil ||
		!d.imagen.validoPara(d.accionNegocio) ||
		d.moduloID != "documentos" || d.tipoRecurso != "imagen_usuario" ||
		d.recursoRef != d.imagen.DocumentoRef || d.cargaRef != d.imagen.DocumentoRef ||
		d.finalidad != d.imagen.Finalidad || d.evidencia.datos != nil {
		return ErrAutorizacionAlmacenInvalida
	}
	resumen := d.materialImagenV3.ResumenCapacidad()
	if resumen.DecisionRef() != d.autorizacionRef ||
		resumen.DecisionHuellaSHA256() != d.huellaDecisionSHA256 ||
		resumen.Operacion() != d.accionNegocio ||
		resumen.EfectoRef() != d.recursoRef ||
		resumen.EfectoHuellaSHA256() != d.huellaRecursoSHA256 ||
		resumen.AudienciaConsumo() != audienciaConsumoImagenV3(d.accionNegocio) ||
		d.verificadaEn.Before(resumen.EmitidaEn()) ||
		!d.verificadaEn.Before(resumen.ExpiraEn()) ||
		!d.validaHasta.Equal(resumen.ExpiraEn()) ||
		len(d.pasos) != 1 || d.pasos[0].referencia != d.pasoRef ||
		d.pasos[0].accion != d.accionTecnica ||
		huellaPlanImagenV3(d) != d.huellaPlanEfectoSHA256 {
		return ErrAutorizacionAlmacenInvalida
	}
	return nil
}

// validarPromocionImagen impide que una capacidad nominal S3 se reutilice
// con otra clave o con una evidencia de análisis distinta de la aprobada.
func (c ContextoOperacionAlmacen) validarPromocionImagen(clave, evidencia string) error {
	if c.validarEstructura() != nil {
		return errorAutorizacionAlmacen()
	}
	if c.datos.imagen == nil {
		return nil
	}
	i := c.datos.imagen
	if c.datos.accionNegocio != AccionNegocioPromoverImagenProcesada ||
		clave != i.ClaveIdempotencia || evidencia != i.EvidenciaAnalisisRef {
		return errorAutorizacionAlmacen()
	}
	return nil
}
