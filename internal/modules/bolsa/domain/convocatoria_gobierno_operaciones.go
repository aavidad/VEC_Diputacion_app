package domain

import (
	"strings"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const TipoEntidadFlujoConvocatoriaBolsa = "convocatoria_bolsa"

func (v VersionConvocatoriaGobernada) ActualizarBorrador(
	revisionEsperada int,
	contenido ContenidoPublicableConvocatoria,
	configuracion ConfiguracionFijadaConvocatoria,
	actorID, motivo string,
	instante time.Time,
) (VersionConvocatoriaGobernada, error) {
	actorID = strings.TrimSpace(actorID)
	motivo = strings.TrimSpace(motivo)
	fecha := instanteConvocatoriaCanonico(instante)
	if v.Validar() != nil || v.EstadoGobierno != EstadoGobiernoConvocatoriaBorrador ||
		revisionEsperada != v.Revision || contenido.Validar() != nil ||
		contenido.IdentificadorPublico != v.Contenido.IdentificadorPublico ||
		configuracion.ValidarPara(contenido) != nil || !referenciaOpacaValida(actorID) ||
		!textoConvocatoriaValido(motivo, 8000, true) || fecha.IsZero() ||
		fecha.Before(v.ultimaFechaGobierno()) {
		return VersionConvocatoriaGobernada{}, ErrTransicionGobiernoConvocatoria
	}
	actualizada := v
	actualizada.Contenido = contenido
	actualizada.Configuracion = configuracion
	actualizada.Revision++
	actualizada.UltimaModificacionPor = actorID
	actualizada.UltimaModificacionEn = fecha
	actualizada.MotivoModificacion = motivo
	canonica, err := actualizada.ClonarCanonico()
	if err != nil {
		return VersionConvocatoriaGobernada{}, err
	}
	huellaAnterior, errAnterior := v.huellaContenidoSinValidar()
	huellaPosterior, errPosterior := canonica.huellaContenidoSinValidar()
	if errAnterior != nil || errPosterior != nil || huellaAnterior == huellaPosterior {
		return VersionConvocatoriaGobernada{}, ErrTransicionGobiernoConvocatoria
	}
	return canonica, nil
}

func (v VersionConvocatoriaGobernada) PublicarInicial(
	actorID string,
	aprobacion EvidenciaAprobacionConvocatoria,
	dependencias EvidenciaDependenciasConvocatoria,
	motivo string,
	instante time.Time,
) (VersionConvocatoriaGobernada, error) {
	if v.Secuencia != 1 || v.VersionAnteriorRef != "" {
		return VersionConvocatoriaGobernada{}, ErrTransicionGobiernoConvocatoria
	}
	return v.publicar(actorID, aprobacion, dependencias, motivo, instante)
}

type ResultadoPublicacionSucesoraConvocatoria struct {
	Publicada   VersionConvocatoriaGobernada
	Predecesora VersionConvocatoriaGobernada
}

// PublicarSucesora devuelve las dos instantaneas que el repositorio debe
// confirmar de forma atomica. Mientras exista una publicación activa conserva la
// misma definición del flujo; cambiarla exigirá una migración expresa.
func (v VersionConvocatoriaGobernada) PublicarSucesora(
	predecesora VersionConvocatoriaGobernada,
	actorID string,
	aprobacion EvidenciaAprobacionConvocatoria,
	dependencias EvidenciaDependenciasConvocatoria,
	motivo string,
	instante time.Time,
) (ResultadoPublicacionSucesoraConvocatoria, error) {
	if v.Validar() != nil || predecesora.Validar() != nil || v.Secuencia <= 1 ||
		(predecesora.EstadoGobierno != EstadoGobiernoConvocatoriaPublicada &&
			predecesora.EstadoGobierno != EstadoGobiernoConvocatoriaRetirada) ||
		v.ID != predecesora.ID || v.Secuencia != predecesora.Secuencia+1 ||
		v.VersionAnteriorRef != predecesora.Referencia() ||
		v.Contenido.IdentificadorPublico != predecesora.Contenido.IdentificadorPublico ||
		v.CreadaEn.Before(predecesora.ultimaFechaGobierno()) ||
		v.Configuracion.FlujoProceso != predecesora.Configuracion.FlujoProceso {
		return ResultadoPublicacionSucesoraConvocatoria{}, ErrTransicionGobiernoConvocatoria
	}
	publicada, err := v.publicar(actorID, aprobacion, dependencias, motivo, instante)
	if err != nil {
		return ResultadoPublicacionSucesoraConvocatoria{}, err
	}
	anterior := predecesora
	if predecesora.EstadoGobierno == EstadoGobiernoConvocatoriaPublicada {
		anterior, err = predecesora.SustituirPor(publicada)
		if err != nil {
			return ResultadoPublicacionSucesoraConvocatoria{}, err
		}
	}
	return ResultadoPublicacionSucesoraConvocatoria{Publicada: publicada, Predecesora: anterior}, nil
}

func (v VersionConvocatoriaGobernada) publicar(
	actorID string,
	aprobacion EvidenciaAprobacionConvocatoria,
	dependencias EvidenciaDependenciasConvocatoria,
	motivo string,
	instante time.Time,
) (VersionConvocatoriaGobernada, error) {
	actorID = strings.TrimSpace(actorID)
	motivo = strings.TrimSpace(motivo)
	fecha := instanteConvocatoriaCanonico(instante)
	if v.Validar() != nil || v.EstadoGobierno != EstadoGobiernoConvocatoriaBorrador ||
		!referenciaOpacaValida(actorID) || actorID == v.CreadaPor || actorID == v.UltimaModificacionPor ||
		!textoConvocatoriaValido(motivo, 8000, true) || fecha.IsZero() || fecha.Before(v.ultimaFechaGobierno()) {
		return VersionConvocatoriaGobernada{}, ErrTransicionGobiernoConvocatoria
	}
	huellaContenido, errContenido := v.huellaContenidoSinValidar()
	huellaEstado, errEstado := v.HuellaSHA256()
	if errContenido != nil || errEstado != nil || !aprobacion.validaPara(
		"publicar", v.Referencia(), v.Revision, huellaContenido, huellaEstado, v.ultimaFechaEdicion(), fecha,
	) || aprobacion.AprobadaPor == v.CreadaPor || aprobacion.AprobadaPor == v.UltimaModificacionPor ||
		aprobacion.AprobadaPor == actorID ||
		!dependencias.validaPara(
			v.Referencia(), v.Revision, huellaContenido, huellaEstado, v.ultimaFechaEdicion(), fecha,
		) {
		return VersionConvocatoriaGobernada{}, ErrTransicionGobiernoConvocatoria
	}
	publicada := v
	publicada.EstadoGobierno = EstadoGobiernoConvocatoriaPublicada
	publicada.PublicadaPor = actorID
	publicada.PublicadaEn = fecha
	publicada.MotivoPublicacion = motivo
	publicada.AprobacionPublicacion = clonarAprobacionConvocatoria(&aprobacion)
	publicada.ComprobacionDependencias = clonarComprobacionDependencias(&dependencias)
	return publicada.ClonarCanonico()
}

func (v VersionConvocatoriaGobernada) Retirar(
	actorID string,
	aprobacion EvidenciaAprobacionConvocatoria,
	motivo string,
	instante time.Time,
) (VersionConvocatoriaGobernada, error) {
	actorID = strings.TrimSpace(actorID)
	motivo = strings.TrimSpace(motivo)
	fecha := instanteConvocatoriaCanonico(instante)
	if v.Validar() != nil || v.EstadoGobierno != EstadoGobiernoConvocatoriaPublicada ||
		!referenciaOpacaValida(actorID) || actorID == v.PublicadaPor ||
		!textoConvocatoriaValido(motivo, 8000, true) || fecha.IsZero() || fecha.Before(v.ultimaFechaGobierno()) {
		return VersionConvocatoriaGobernada{}, ErrTransicionGobiernoConvocatoria
	}
	huellaContenido, errContenido := v.huellaContenidoSinValidar()
	huellaEstado, errEstado := v.HuellaSHA256()
	if errContenido != nil || errEstado != nil || !aprobacion.validaPara(
		"retirar", v.Referencia(), v.Revision, huellaContenido, huellaEstado, v.PublicadaEn, fecha,
	) ||
		aprobacion.AprobadaPor == v.PublicadaPor || aprobacion.AprobadaPor == actorID {
		return VersionConvocatoriaGobernada{}, ErrTransicionGobiernoConvocatoria
	}
	retirada := v
	retirada.EstadoGobierno = EstadoGobiernoConvocatoriaRetirada
	retirada.RetiradaPor = actorID
	retirada.RetiradaEn = fecha
	retirada.MotivoRetirada = motivo
	retirada.AprobacionRetirada = clonarAprobacionConvocatoria(&aprobacion)
	return retirada.ClonarCanonico()
}

func (v VersionConvocatoriaGobernada) NuevaVersion(
	codigoVersionPublica string,
	contenido ContenidoPublicableConvocatoria,
	configuracion ConfiguracionFijadaConvocatoria,
	expedienteRef, actorID, motivo string,
	instante time.Time,
) (VersionConvocatoriaGobernada, error) {
	codigoVersionPublica = strings.TrimSpace(codigoVersionPublica)
	actorID = strings.TrimSpace(actorID)
	expedienteRef = strings.TrimSpace(expedienteRef)
	motivo = strings.TrimSpace(motivo)
	fecha := instanteConvocatoriaCanonico(instante)
	if v.Validar() != nil || (v.EstadoGobierno != EstadoGobiernoConvocatoriaPublicada &&
		v.EstadoGobierno != EstadoGobiernoConvocatoriaRetirada) ||
		!claveCatalogoConvocatoriaValida(codigoVersionPublica) || codigoVersionPublica == v.CodigoVersionPublica ||
		contenido.Validar() != nil || contenido.IdentificadorPublico != v.Contenido.IdentificadorPublico ||
		configuracion.ValidarPara(contenido) != nil || !referenciaOpacaValida(expedienteRef) ||
		!referenciaOpacaValida(actorID) || !textoConvocatoriaValido(motivo, 8000, true) ||
		fecha.IsZero() || fecha.Before(v.ultimaFechaGobierno()) {
		return VersionConvocatoriaGobernada{}, ErrTransicionGobiernoConvocatoria
	}
	nueva := VersionConvocatoriaGobernada{
		ID: v.ID, Secuencia: v.Secuencia + 1, CodigoVersionPublica: codigoVersionPublica,
		Revision: 1, VersionAnteriorRef: v.Referencia(), Contenido: contenido,
		Configuracion: configuracion, ExpedienteRef: expedienteRef, MotivoCreacion: motivo,
		EstadoGobierno: EstadoGobiernoConvocatoriaBorrador, CreadaPor: actorID, CreadaEn: fecha,
	}
	return nueva.ClonarCanonico()
}

// SustituirPor prepara la mutacion de la version anterior. El repositorio debe
// confirmar esta copia y la publicacion nueva en una unica transaccion.
func (v VersionConvocatoriaGobernada) SustituirPor(
	nueva VersionConvocatoriaGobernada,
) (VersionConvocatoriaGobernada, error) {
	if v.Validar() != nil || nueva.Validar() != nil ||
		v.EstadoGobierno != EstadoGobiernoConvocatoriaPublicada ||
		nueva.EstadoGobierno != EstadoGobiernoConvocatoriaPublicada || nueva.ID != v.ID ||
		nueva.Secuencia != v.Secuencia+1 || nueva.VersionAnteriorRef != v.Referencia() ||
		nueva.Contenido.IdentificadorPublico != v.Contenido.IdentificadorPublico ||
		nueva.CreadaEn.Before(v.ultimaFechaGobierno()) || nueva.PublicadaEn.Before(v.PublicadaEn) {
		return VersionConvocatoriaGobernada{}, ErrTransicionGobiernoConvocatoria
	}
	sustituida := v
	sustituida.EstadoGobierno = EstadoGobiernoConvocatoriaSustituida
	sustituida.SustituidaPorRef = nueva.Referencia()
	sustituida.SustituidaPor = nueva.PublicadaPor
	sustituida.SustituidaEn = nueva.PublicadaEn
	return sustituida.ClonarCanonico()
}

func (v VersionConvocatoriaGobernada) ProyectarPublica(
	instancia dominiovec.InstanciaFlujo,
) (Convocatoria, error) {
	actualizadaFlujoEn := instancia.CreadaEn
	if instancia.Revision > 1 {
		actualizadaFlujoEn = instancia.ActualizadaEn
	}
	actualizadaFlujoEn = instanteConvocatoriaCanonico(actualizadaFlujoEn)
	actualizadaEn := actualizadaFlujoEn
	if actualizadaEn.Before(v.PublicadaEn) {
		actualizadaEn = v.PublicadaEn
	}
	definicionRef := v.Configuracion.FlujoProceso.ReferenciaVersionada()
	estado := EstadoConvocatoria(instancia.EstadoActual)
	if v.Validar() != nil || v.EstadoGobierno != EstadoGobiernoConvocatoriaPublicada ||
		instancia.Validar() != nil || instancia.TipoEntidad != TipoEntidadFlujoConvocatoriaBolsa ||
		instancia.EntidadRef != v.ID || instancia.DefinicionRef != definicionRef ||
		instancia.DefinicionContenidoHuellaSHA256 != v.Configuracion.FlujoProceso.HuellaContenidoSHA256 ||
		!estado.IsValid() || actualizadaEn.IsZero() ||
		(v.Secuencia == 1 && actualizadaFlujoEn.Before(v.PublicadaEn)) {
		return Convocatoria{}, ErrVersionConvocatoriaGobernadaInvalida
	}
	contenido := v.Contenido
	documentos := make([]DocumentoConvocatoria, len(contenido.Documentos))
	for indice, documento := range contenido.Documentos {
		documentos[indice] = DocumentoConvocatoria{
			Referencia: documento.Referencia, Tipo: documento.Tipo, Orden: documento.Orden,
			Titulo: documento.Titulo, Descripcion: documento.Descripcion, Formato: documento.Formato,
			URL: documento.URL, PublicadoEn: v.PublicadaEn,
		}
	}
	proyeccion := Convocatoria{
		ID: v.ID, Version: v.CodigoVersionPublica, Estado: estado,
		DatosPublicos: &DatosPublicosConvocatoria{
			IdentificadorPublico: contenido.IdentificadorPublico, Tipo: contenido.Tipo,
			CatalogoCategorias: contenido.CatalogoCategorias,
			Categorias:         append([]string(nil), contenido.Categorias...), Titulo: contenido.Titulo,
			Resumen: contenido.Resumen, Descripcion: contenido.Descripcion,
			PublicadaEn: v.PublicadaEn, ActualizadaEn: actualizadaEn,
			Plazos:     append([]PlazoConvocatoria(nil), contenido.Plazos...),
			Requisitos: append([]RequisitoConvocatoria(nil), contenido.Requisitos...),
			Documentos: documentos, Ayuda: append([]AyudaConvocatoria(nil), contenido.Ayuda...),
		},
	}
	if err := proyeccion.ValidarPublicacion(); err != nil {
		return Convocatoria{}, ErrVersionConvocatoriaGobernadaInvalida
	}
	return proyeccion.Clonar(), nil
}

func (e EvidenciaAprobacionConvocatoria) validaPara(
	accion, convocatoriaRef string,
	revision int,
	huellaContenido, huellaEstado string,
	noAntes, noDespues time.Time,
) bool {
	return e.Accion == accion && referenciaOpacaValida(e.Referencia) &&
		huellaSHA256Valida(e.HuellaEvidenciaSHA256) && e.ConvocatoriaRef == convocatoriaRef &&
		e.Revision == revision && e.HuellaContenidoSHA256 == huellaContenido &&
		e.HuellaEstadoSHA256 == huellaEstado && referenciaOpacaValida(e.AprobadaPor) &&
		instanteUTCCanonico(e.AprobadaEn) && !e.AprobadaEn.Before(noAntes) && !e.AprobadaEn.After(noDespues)
}

func (e EvidenciaDependenciasConvocatoria) validaPara(
	convocatoriaRef string,
	revision int,
	huellaContenido, huellaEstado string,
	noAntes, noDespues time.Time,
) bool {
	return referenciaOpacaValida(e.Referencia) && huellaSHA256Valida(e.HuellaEvidenciaSHA256) &&
		e.ConvocatoriaRef == convocatoriaRef && e.Revision == revision &&
		e.HuellaContenidoSHA256 == huellaContenido && e.HuellaEstadoSHA256 == huellaEstado &&
		instanteUTCCanonico(e.VerificadaEn) && !e.VerificadaEn.Before(noAntes) && !e.VerificadaEn.After(noDespues) &&
		noDespues.Sub(e.VerificadaEn) <= vigenciaMaximaComprobacionDependencias
}

func (v VersionConvocatoriaGobernada) datosPublicacionPresentes() bool {
	return v.PublicadaPor != "" || !v.PublicadaEn.IsZero() || v.MotivoPublicacion != "" ||
		v.AprobacionPublicacion != nil || v.ComprobacionDependencias != nil
}

func (v VersionConvocatoriaGobernada) datosSustitucionPresentes() bool {
	return v.SustituidaPorRef != "" || v.SustituidaPor != "" || !v.SustituidaEn.IsZero()
}

func (v VersionConvocatoriaGobernada) datosRetiradaPresentes() bool {
	return v.RetiradaPor != "" || !v.RetiradaEn.IsZero() || v.MotivoRetirada != "" || v.AprobacionRetirada != nil
}

func (v VersionConvocatoriaGobernada) datosPublicacionValidos() bool {
	huellaContenido, errContenido := v.huellaContenidoSinValidar()
	huellaEstado, errEstado := v.huellaBorradorPrevioPublicacion()
	return errContenido == nil && errEstado == nil && referenciaOpacaValida(v.PublicadaPor) &&
		instanteUTCCanonico(v.PublicadaEn) &&
		v.PublicadaEn.Compare(v.ultimaFechaEdicion()) >= 0 && textoConvocatoriaValido(v.MotivoPublicacion, 8000, true) &&
		v.AprobacionPublicacion != nil && v.AprobacionPublicacion.validaPara(
		"publicar", v.Referencia(), v.Revision, huellaContenido, huellaEstado,
		v.ultimaFechaEdicion(), v.PublicadaEn,
	) && v.AprobacionPublicacion.AprobadaPor != v.CreadaPor &&
		v.AprobacionPublicacion.AprobadaPor != v.UltimaModificacionPor &&
		v.AprobacionPublicacion.AprobadaPor != v.PublicadaPor &&
		v.ComprobacionDependencias != nil && v.ComprobacionDependencias.validaPara(
		v.Referencia(), v.Revision, huellaContenido, huellaEstado, v.ultimaFechaEdicion(), v.PublicadaEn,
	)
}

func (v VersionConvocatoriaGobernada) datosSustitucionValidos() bool {
	return v.SustituidaPorRef == referenciaVersionConvocatoria(v.ID, v.Secuencia+1) &&
		referenciaOpacaValida(v.SustituidaPor) && instanteUTCCanonico(v.SustituidaEn) &&
		!v.SustituidaEn.Before(v.PublicadaEn)
}

func (v VersionConvocatoriaGobernada) datosRetiradaValidos() bool {
	huellaContenido, errContenido := v.huellaContenidoSinValidar()
	huellaEstado, errEstado := v.huellaPublicadaPreviaRetirada()
	return errContenido == nil && errEstado == nil && referenciaOpacaValida(v.RetiradaPor) &&
		v.RetiradaPor != v.PublicadaPor &&
		instanteUTCCanonico(v.RetiradaEn) && !v.RetiradaEn.Before(v.PublicadaEn) &&
		textoConvocatoriaValido(v.MotivoRetirada, 8000, true) && v.AprobacionRetirada != nil &&
		v.AprobacionRetirada.validaPara(
			"retirar", v.Referencia(), v.Revision, huellaContenido, huellaEstado, v.PublicadaEn, v.RetiradaEn,
		) &&
		v.AprobacionRetirada.AprobadaPor != v.PublicadaPor &&
		v.AprobacionRetirada.AprobadaPor != v.RetiradaPor
}

func (v VersionConvocatoriaGobernada) huellaBorradorPrevioPublicacion() (string, error) {
	base := v
	base.EstadoGobierno = EstadoGobiernoConvocatoriaBorrador
	base.PublicadaPor = ""
	base.PublicadaEn = time.Time{}
	base.MotivoPublicacion = ""
	base.AprobacionPublicacion = nil
	base.ComprobacionDependencias = nil
	base.SustituidaPorRef = ""
	base.SustituidaPor = ""
	base.SustituidaEn = time.Time{}
	base.RetiradaPor = ""
	base.RetiradaEn = time.Time{}
	base.MotivoRetirada = ""
	base.AprobacionRetirada = nil
	return base.HuellaSHA256()
}

func (v VersionConvocatoriaGobernada) huellaPublicadaPreviaRetirada() (string, error) {
	base := v
	base.EstadoGobierno = EstadoGobiernoConvocatoriaPublicada
	base.RetiradaPor = ""
	base.RetiradaEn = time.Time{}
	base.MotivoRetirada = ""
	base.AprobacionRetirada = nil
	return base.HuellaSHA256()
}

func (v VersionConvocatoriaGobernada) ultimaFechaEdicion() time.Time {
	if !v.UltimaModificacionEn.IsZero() {
		return v.UltimaModificacionEn
	}
	return v.CreadaEn
}

func (v VersionConvocatoriaGobernada) ultimaFechaGobierno() time.Time {
	if !v.RetiradaEn.IsZero() {
		return v.RetiradaEn
	}
	if !v.SustituidaEn.IsZero() {
		return v.SustituidaEn
	}
	if !v.PublicadaEn.IsZero() {
		return v.PublicadaEn
	}
	return v.ultimaFechaEdicion()
}
