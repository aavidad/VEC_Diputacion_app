package httpinterno

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	esquemaOpcionesBorradores = "vec.bolsa.borradores.opciones.v1"
	esquemaListaBorradores    = "vec.bolsa.borradores.lista.v1"
	esquemaDetalleBorrador    = "vec.bolsa.borrador.detalle.v1"
	esquemaGuardadoBorrador   = "vec.bolsa.borrador.guardado.v1"
)

var errSalidaBorradorInsegura = errors.New("bolsa http interno: salida de borrador no confiable")

type envelopeDatosBorrador[T any] struct {
	Data T `json:"data"`
}

type capacidadesGlobalesBorradorJSON struct {
	Consultar bool `json:"consultar"`
	Crear     bool `json:"crear"`
}

type capacidadesFilaBorradorJSON struct {
	Consultar  bool `json:"consultar"`
	Actualizar bool `json:"actualizar"`
}

type limiteEdicionBorradorJSON struct {
	MaximoCategorias           int `json:"maximo_categorias"`
	MaximoPlazos               int `json:"maximo_plazos"`
	MaximoRequisitos           int `json:"maximo_requisitos"`
	MaximoDocumentos           int `json:"maximo_documentos"`
	MaximoAyudas               int `json:"maximo_ayudas"`
	MaximoTitulo               int `json:"maximo_titulo"`
	MaximoResumen              int `json:"maximo_resumen"`
	MaximoDescripcion          int `json:"maximo_descripcion"`
	MaximoTituloPlazo          int `json:"maximo_titulo_plazo"`
	MaximoDescripcionPlazo     int `json:"maximo_descripcion_plazo"`
	MaximoTituloRequisito      int `json:"maximo_titulo_requisito"`
	MaximoDescripcionRequisito int `json:"maximo_descripcion_requisito"`
	MaximoPreguntaAyuda        int `json:"maximo_pregunta_ayuda"`
	MaximoRespuestaAyuda       int `json:"maximo_respuesta_ayuda"`
}

type opcionCategoriaBorradorJSON struct {
	CategoriaRef string `json:"categoria_ref"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
	Clave        string `json:"clave"`
	Etiqueta     string `json:"etiqueta"`
}

type opcionTipoBorradorJSON struct {
	TipoRef      string `json:"tipo_ref"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
	Clave        string `json:"clave"`
	Etiqueta     string `json:"etiqueta"`
}

type opcionPlantillaBorradorJSON struct {
	PlantillaRef string `json:"plantilla_ref"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
	Nombre       string `json:"nombre"`
	Descripcion  string `json:"descripcion"`
}

type opcionMotivoBorradorJSON struct {
	MotivoRef    string `json:"motivo_ref"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
	Etiqueta     string `json:"etiqueta"`
	Descripcion  string `json:"descripcion"`
}

type opcionesBorradoresJSON struct {
	Esquema     string                          `json:"esquema"`
	Categorias  []opcionCategoriaBorradorJSON   `json:"categorias"`
	Tipos       []opcionTipoBorradorJSON        `json:"tipos"`
	Plantillas  []opcionPlantillaBorradorJSON   `json:"plantillas"`
	Motivos     []opcionMotivoBorradorJSON      `json:"motivos"`
	Limites     limiteEdicionBorradorJSON       `json:"limites"`
	Capacidades capacidadesGlobalesBorradorJSON `json:"capacidades"`
}

type referenciaEstadoBorradorJSON struct {
	Referencia         string `json:"referencia"`
	Revision           int    `json:"revision"`
	HuellaEstadoSHA256 string `json:"huella_estado_sha256"`
}

type selectorListaBorradoresJSON struct {
	Limite    int    `json:"limite"`
	Cursor    string `json:"cursor,omitempty"`
	Texto     string `json:"texto,omitempty"`
	Categoria string `json:"categoria,omitempty"`
}

type paginacionBorradoresJSON struct {
	Limite          int    `json:"limite"`
	Total           int    `json:"total"`
	SiguienteCursor string `json:"siguiente_cursor,omitempty"`
}

type filaBorradorJSON struct {
	ReferenciaEstado     referenciaEstadoBorradorJSON `json:"referencia_estado"`
	ETag                 string                       `json:"etag"`
	CodigoVersionPublica string                       `json:"codigo_version_publica"`
	IdentificadorPublico string                       `json:"identificador_publico"`
	Titulo               string                       `json:"titulo"`
	Tipo                 string                       `json:"tipo"`
	Categorias           []string                     `json:"categorias"`
	ExpedienteRef        string                       `json:"expediente_ref"`
	CreadaEn             string                       `json:"creada_en"`
	ActualizadaEn        string                       `json:"actualizada_en"`
	NumeroPlazos         int                          `json:"numero_plazos"`
	NumeroRequisitos     int                          `json:"numero_requisitos"`
	NumeroDocumentos     int                          `json:"numero_documentos"`
	NumeroAyudas         int                          `json:"numero_ayudas"`
	Capacidades          capacidadesFilaBorradorJSON  `json:"capacidades"`
}

type listaBorradoresJSON struct {
	Esquema     string                          `json:"esquema"`
	Selector    selectorListaBorradoresJSON     `json:"selector"`
	Paginacion  paginacionBorradoresJSON        `json:"paginacion"`
	Capacidades capacidadesGlobalesBorradorJSON `json:"capacidades"`
	Elementos   []filaBorradorJSON              `json:"elementos"`
}

type plazoBorradorSalidaJSON struct {
	Referencia  string `json:"referencia"`
	Tipo        string `json:"tipo"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	AbreEn      string `json:"abre_en"`
	CierraEn    string `json:"cierra_en"`
}

type requisitoBorradorSalidaJSON struct {
	Referencia  string `json:"referencia"`
	Orden       int    `json:"orden"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Obligatorio bool   `json:"obligatorio"`
}

type ayudaBorradorSalidaJSON struct {
	Referencia string `json:"referencia"`
	Categoria  string `json:"categoria"`
	Orden      int    `json:"orden"`
	Pregunta   string `json:"pregunta"`
	Respuesta  string `json:"respuesta"`
}

type contenidoEditableBorradorJSON struct {
	Tipo        string                        `json:"tipo"`
	Categorias  []string                      `json:"categorias"`
	Titulo      string                        `json:"titulo"`
	Resumen     string                        `json:"resumen"`
	Descripcion string                        `json:"descripcion"`
	Plazos      []plazoBorradorSalidaJSON     `json:"plazos"`
	Requisitos  []requisitoBorradorSalidaJSON `json:"requisitos"`
	Ayuda       []ayudaBorradorSalidaJSON     `json:"ayuda"`
}

type referenciaConfiguracionBorradorJSON struct {
	Referencia   string `json:"referencia"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type documentoLecturaBorradorJSON struct {
	Rol                   string `json:"rol"`
	PublicacionRef        string `json:"publicacion_ref"`
	DocumentoRef          string `json:"documento_ref"`
	VersionDocumento      int    `json:"version_documento"`
	RepresentacionRef     string `json:"representacion_ref"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
	FirmaValidadaRef      string `json:"firma_validada_ref"`
	ReciboCustodiaRef     string `json:"recibo_custodia_ref"`
}

type configuracionLecturaBorradorJSON struct {
	Catalogos        referenciaConfiguracionBorradorJSON `json:"catalogos"`
	Calendario       referenciaConfiguracionBorradorJSON `json:"calendario"`
	ReglasBaremacion referenciaConfiguracionBorradorJSON `json:"reglas_baremacion"`
	FlujoProceso     referenciaConfiguracionBorradorJSON `json:"flujo_proceso"`
	FlujoSolicitud   referenciaConfiguracionBorradorJSON `json:"flujo_solicitud"`
	Plantilla        referenciaConfiguracionBorradorJSON `json:"plantilla"`
	Documentos       []documentoLecturaBorradorJSON      `json:"documentos"`
}

type ambitoLecturaBorradorJSON struct {
	OrganizacionRef  string `json:"organizacion_ref"`
	UnidadGestionRef string `json:"unidad_gestion_ref,omitempty"`
}

type detalleBorradorJSON struct {
	Esquema              string                           `json:"esquema"`
	ReferenciaEstado     referenciaEstadoBorradorJSON     `json:"referencia_estado"`
	ETag                 string                           `json:"etag"`
	CodigoVersionPublica string                           `json:"codigo_version_publica"`
	IdentificadorPublico string                           `json:"identificador_publico"`
	AmbitoLectura        ambitoLecturaBorradorJSON        `json:"ambito_lectura"`
	ExpedienteRef        string                           `json:"expediente_ref"`
	ContenidoEditable    contenidoEditableBorradorJSON    `json:"contenido_editable"`
	ConfiguracionLectura configuracionLecturaBorradorJSON `json:"configuracion_lectura"`
	Capacidades          capacidadesFilaBorradorJSON      `json:"capacidades"`
}

type reciboGuardadoBorradorJSON struct {
	Esquema          string                       `json:"esquema"`
	TransaccionRef   string                       `json:"transaccion_ref"`
	Accion           string                       `json:"accion"`
	ReferenciaEstado referenciaEstadoBorradorJSON `json:"referencia_estado"`
	ETag             string                       `json:"etag"`
	AuditoriaRef     string                       `json:"auditoria_ref"`
	EventoOutboxRef  string                       `json:"evento_outbox_ref"`
	ConfirmadaEn     string                       `json:"confirmada_en"`
}

func nuevaRespuestaOpcionesBorrador(
	origen gobiernoconvocatorias.OpcionesBorradores,
) (envelopeDatosBorrador[opcionesBorradoresJSON], error) {
	if !limitesBorradorValidos(origen.Limites) || len(origen.Categorias) > 10_000 || len(origen.Tipos) > 4096 ||
		len(origen.Plantillas) > 1024 || len(origen.Motivos) > 1024 {
		return envelopeDatosBorrador[opcionesBorradoresJSON]{}, errSalidaBorradorInsegura
	}
	salida := opcionesBorradoresJSON{
		Esquema:     esquemaOpcionesBorradores,
		Categorias:  make([]opcionCategoriaBorradorJSON, len(origen.Categorias)),
		Tipos:       make([]opcionTipoBorradorJSON, len(origen.Tipos)),
		Plantillas:  make([]opcionPlantillaBorradorJSON, len(origen.Plantillas)),
		Motivos:     make([]opcionMotivoBorradorJSON, len(origen.Motivos)),
		Limites:     limitesAJSON(origen.Limites),
		Capacidades: capacidadesGlobalesAJSON(origen.Capacidades),
	}
	identidades, claves := map[string]struct{}{}, map[string]struct{}{}
	huellasCatalogo := map[string]string{}
	for indice, opcion := range origen.Categorias {
		identidadCatalogo := opcion.Referencia + ":" + strconv.Itoa(opcion.Version)
		if !opcionCatalogoValida(opcion) || insertarRepetida(identidades, opcion.Referencia+":"+strconv.Itoa(opcion.Version)+":"+opcion.Clave) ||
			insertarRepetida(claves, opcion.Clave) || huellaCatalogoContradictoria(huellasCatalogo, identidadCatalogo, opcion.HuellaSHA256) {
			return envelopeDatosBorrador[opcionesBorradoresJSON]{}, errSalidaBorradorInsegura
		}
		salida.Categorias[indice] = opcionCategoriaBorradorJSON{
			CategoriaRef: opcion.Referencia, Version: opcion.Version, HuellaSHA256: opcion.HuellaSHA256,
			Clave: opcion.Clave, Etiqueta: opcion.Etiqueta,
		}
	}
	identidades, claves, huellasCatalogo = map[string]struct{}{}, map[string]struct{}{}, map[string]string{}
	for indice, opcion := range origen.Tipos {
		identidadCatalogo := opcion.Referencia + ":" + strconv.Itoa(opcion.Version)
		if !opcionCatalogoValida(opcion) || insertarRepetida(identidades, opcion.Referencia+":"+strconv.Itoa(opcion.Version)+":"+opcion.Clave) ||
			insertarRepetida(claves, opcion.Clave) || huellaCatalogoContradictoria(huellasCatalogo, identidadCatalogo, opcion.HuellaSHA256) {
			return envelopeDatosBorrador[opcionesBorradoresJSON]{}, errSalidaBorradorInsegura
		}
		salida.Tipos[indice] = opcionTipoBorradorJSON{
			TipoRef: opcion.Referencia, Version: opcion.Version, HuellaSHA256: opcion.HuellaSHA256,
			Clave: opcion.Clave, Etiqueta: opcion.Etiqueta,
		}
	}
	identidades = map[string]struct{}{}
	for indice, opcion := range origen.Plantillas {
		if !referenciaBorradorValida(opcion.Referencia, 512) || !enteroSeguroPositivo(opcion.Version) ||
			!patronHuellaBorrador.MatchString(opcion.HuellaSHA256) ||
			!cadenaBorradorValida(opcion.Nombre, 180, false, false) ||
			!cadenaBorradorValida(opcion.Descripcion, 1000, true, true) ||
			insertarRepetida(identidades, opcion.Referencia+":"+strconv.Itoa(opcion.Version)) {
			return envelopeDatosBorrador[opcionesBorradoresJSON]{}, errSalidaBorradorInsegura
		}
		salida.Plantillas[indice] = opcionPlantillaBorradorJSON{
			PlantillaRef: opcion.Referencia, Version: opcion.Version, HuellaSHA256: opcion.HuellaSHA256,
			Nombre: opcion.Nombre, Descripcion: opcion.Descripcion,
		}
	}
	identidades = map[string]struct{}{}
	for indice, opcion := range origen.Motivos {
		if !referenciaBorradorValida(opcion.Referencia, 512) || !enteroSeguroPositivo(opcion.Version) ||
			!patronHuellaBorrador.MatchString(opcion.HuellaSHA256) ||
			!cadenaBorradorValida(opcion.Etiqueta, 180, false, false) ||
			!cadenaBorradorValida(opcion.Descripcion, 1000, true, true) ||
			insertarRepetida(identidades, opcion.Referencia+":"+strconv.Itoa(opcion.Version)) {
			return envelopeDatosBorrador[opcionesBorradoresJSON]{}, errSalidaBorradorInsegura
		}
		salida.Motivos[indice] = opcionMotivoBorradorJSON{
			MotivoRef: opcion.Referencia, Version: opcion.Version, HuellaSHA256: opcion.HuellaSHA256,
			Etiqueta: opcion.Etiqueta, Descripcion: opcion.Descripcion,
		}
	}
	return envelopeDatosBorrador[opcionesBorradoresJSON]{Data: salida}, nil
}

func nuevaRespuestaListaBorradores(
	origen gobiernoconvocatorias.ListaBorradores,
	solicitado gobiernoconvocatorias.SelectorListaBorradores,
) (envelopeDatosBorrador[listaBorradoresJSON], error) {
	if origen.Selector != solicitado || solicitado.Limite < 1 || solicitado.Limite > 50 ||
		origen.Total < 0 || uint64(origen.Total) > uint64(maximoEnteroSeguroJSON) ||
		len(origen.Elementos) > solicitado.Limite || len(origen.Elementos) > origen.Total ||
		((origen.Total == 0) != (len(origen.Elementos) == 0)) ||
		(origen.SiguienteCursor != "" && (!patronCursorBorrador.MatchString(origen.SiguienteCursor) ||
			origen.Total <= len(origen.Elementos))) {
		return envelopeDatosBorrador[listaBorradoresJSON]{}, errSalidaBorradorInsegura
	}
	salida := listaBorradoresJSON{
		Esquema: esquemaListaBorradores,
		Selector: selectorListaBorradoresJSON{
			Limite: solicitado.Limite, Cursor: solicitado.Cursor, Texto: solicitado.Texto, Categoria: solicitado.Categoria,
		},
		Paginacion: paginacionBorradoresJSON{
			Limite: solicitado.Limite, Total: origen.Total, SiguienteCursor: origen.SiguienteCursor,
		},
		Capacidades: capacidadesGlobalesAJSON(origen.Capacidades),
		Elementos:   make([]filaBorradorJSON, len(origen.Elementos)),
	}
	vistas := map[string]struct{}{}
	for indice, fila := range origen.Elementos {
		convertida, err := convertirFilaBorrador(fila)
		if err != nil || insertarRepetida(vistas, fila.Estado.Referencia) {
			return envelopeDatosBorrador[listaBorradoresJSON]{}, errSalidaBorradorInsegura
		}
		salida.Elementos[indice] = convertida
	}
	return envelopeDatosBorrador[listaBorradoresJSON]{Data: salida}, nil
}

func nuevaRespuestaDetalleBorrador(
	origen gobiernoconvocatorias.DetalleBorrador,
	solicitado puertosbolsa.SelectorVersionConvocatoriaExacta,
) (envelopeDatosBorrador[detalleBorradorJSON], string, error) {
	if !estadoSalidaBorradorValido(origen.Estado) || origen.Estado.Referencia != solicitado.Referencia() ||
		!patronClaveBorrador.MatchString(origen.CodigoVersionPublica) ||
		!patronIdentificadorBorrador.MatchString(origen.IdentificadorPublico) ||
		!referenciaBorradorValida(origen.Ambito.OrganizacionRef, 512) ||
		(origen.Ambito.UnidadGestionRef != "" && !referenciaBorradorValida(origen.Ambito.UnidadGestionRef, 512)) ||
		!referenciaBorradorValida(origen.ExpedienteRef, 512) {
		return envelopeDatosBorrador[detalleBorradorJSON]{}, "", errSalidaBorradorInsegura
	}
	contenido, err := convertirContenidoSalida(origen.Contenido)
	if err != nil {
		return envelopeDatosBorrador[detalleBorradorJSON]{}, "", err
	}
	configuracion, err := convertirConfiguracionSalida(origen.Configuracion)
	if err != nil {
		return envelopeDatosBorrador[detalleBorradorJSON]{}, "", err
	}
	etag := etagEstadoBorrador(origen.Estado)
	salida := detalleBorradorJSON{
		Esquema: esquemaDetalleBorrador, ReferenciaEstado: estadoAJSON(origen.Estado), ETag: etag,
		CodigoVersionPublica: origen.CodigoVersionPublica, IdentificadorPublico: origen.IdentificadorPublico,
		AmbitoLectura: ambitoLecturaBorradorJSON{
			OrganizacionRef: origen.Ambito.OrganizacionRef, UnidadGestionRef: origen.Ambito.UnidadGestionRef,
		},
		ExpedienteRef: origen.ExpedienteRef, ContenidoEditable: contenido,
		ConfiguracionLectura: configuracion, Capacidades: capacidadesFilaAJSON(origen.Capacidades),
	}
	return envelopeDatosBorrador[detalleBorradorJSON]{Data: salida}, etag, nil
}

func nuevaRespuestaReciboBorrador(
	origen gobiernoconvocatorias.ProyeccionReciboBorrador,
	accionEsperada string,
) (envelopeDatosBorrador[reciboGuardadoBorradorJSON], string, string, error) {
	accionInterna := puertosbolsa.AccionCrearBorradorConvocatoria
	if accionEsperada == "actualizar" {
		accionInterna = puertosbolsa.AccionActualizarBorradorConvocatoria
	}
	if (accionEsperada != "crear" && accionEsperada != "actualizar") || origen.Accion != accionInterna ||
		!estadoSalidaBorradorValido(origen.EstadoPrincipal) || !referenciaBorradorValida(origen.TransaccionRef, 512) ||
		!referenciaBorradorValida(origen.AuditoriaRef, 512) || !referenciaBorradorValida(origen.EventoOutboxRef, 512) ||
		origen.TransaccionRef == origen.AuditoriaRef || origen.TransaccionRef == origen.EventoOutboxRef ||
		origen.AuditoriaRef == origen.EventoOutboxRef ||
		!instanteSalidaBorradorValido(origen.ConfirmadaEn) {
		return envelopeDatosBorrador[reciboGuardadoBorradorJSON]{}, "", "", errSalidaBorradorInsegura
	}
	selector, err := selectorDesdeReferenciaEstado(origen.EstadoPrincipal.Referencia)
	if err != nil {
		return envelopeDatosBorrador[reciboGuardadoBorradorJSON]{}, "", "", errSalidaBorradorInsegura
	}
	etag := etagEstadoBorrador(origen.EstadoPrincipal)
	location := RutaBorradores + "/" + codificarIdentificadorRutaBorrador(selector.ID) +
		"/versiones/" + strconv.Itoa(selector.Secuencia)
	salida := reciboGuardadoBorradorJSON{
		Esquema: esquemaGuardadoBorrador, TransaccionRef: origen.TransaccionRef, Accion: accionEsperada,
		ReferenciaEstado: estadoAJSON(origen.EstadoPrincipal), ETag: etag,
		AuditoriaRef: origen.AuditoriaRef, EventoOutboxRef: origen.EventoOutboxRef,
		ConfirmadaEn: formatearInstanteBorrador(origen.ConfirmadaEn),
	}
	return envelopeDatosBorrador[reciboGuardadoBorradorJSON]{Data: salida}, etag, location, nil
}

func convertirFilaBorrador(origen gobiernoconvocatorias.FilaBorrador) (filaBorradorJSON, error) {
	if !estadoSalidaBorradorValido(origen.Estado) || !patronClaveBorrador.MatchString(origen.CodigoVersionPublica) ||
		!patronIdentificadorBorrador.MatchString(origen.IdentificadorPublico) ||
		!cadenaBorradorValida(origen.Titulo, 180, false, false) || !patronClaveBorrador.MatchString(origen.Tipo) ||
		len(origen.Categorias) < 1 || len(origen.Categorias) > 1024 || !referenciaBorradorValida(origen.ExpedienteRef, 512) ||
		!instanteSalidaBorradorValido(origen.CreadaEn) || !instanteSalidaBorradorValido(origen.ActualizadaEn) ||
		origen.ActualizadaEn.Before(origen.CreadaEn) || origen.NumeroPlazos < 0 || origen.NumeroPlazos > 64 ||
		origen.NumeroRequisitos < 0 || origen.NumeroRequisitos > 256 || origen.NumeroDocumentos < 0 ||
		origen.NumeroDocumentos > 256 || origen.NumeroAyudas < 0 || origen.NumeroAyudas > 128 {
		return filaBorradorJSON{}, errSalidaBorradorInsegura
	}
	categorias := append([]string(nil), origen.Categorias...)
	vistas := map[string]struct{}{}
	for _, categoria := range categorias {
		if !patronClaveBorrador.MatchString(categoria) || insertarRepetida(vistas, categoria) {
			return filaBorradorJSON{}, errSalidaBorradorInsegura
		}
	}
	return filaBorradorJSON{
		ReferenciaEstado: estadoAJSON(origen.Estado), ETag: etagEstadoBorrador(origen.Estado),
		CodigoVersionPublica: origen.CodigoVersionPublica, IdentificadorPublico: origen.IdentificadorPublico,
		Titulo: origen.Titulo, Tipo: origen.Tipo, Categorias: categorias, ExpedienteRef: origen.ExpedienteRef,
		CreadaEn: formatearInstanteBorrador(origen.CreadaEn), ActualizadaEn: formatearInstanteBorrador(origen.ActualizadaEn),
		NumeroPlazos: origen.NumeroPlazos, NumeroRequisitos: origen.NumeroRequisitos,
		NumeroDocumentos: origen.NumeroDocumentos, NumeroAyudas: origen.NumeroAyudas,
		Capacidades: capacidadesFilaAJSON(origen.Capacidades),
	}, nil
}

func convertirContenidoSalida(
	origen gobiernoconvocatorias.ContenidoEditableBorrador,
) (contenidoEditableBorradorJSON, error) {
	entrada := contenidoBorradorJSON{
		Tipo: origen.Tipo, Categorias: &origen.Categorias, Titulo: origen.Titulo, Resumen: origen.Resumen,
		Descripcion: origen.Descripcion, Plazos: &[]plazoBorradorJSON{}, Requisitos: &[]requisitoBorradorJSON{},
		Ayuda: &[]ayudaBorradorJSON{},
	}
	for _, plazo := range origen.Plazos {
		*entrada.Plazos = append(*entrada.Plazos, plazoBorradorJSON{
			Referencia: plazo.Referencia, Tipo: plazo.Tipo, Titulo: plazo.Titulo, Descripcion: plazo.Descripcion,
			AbreEn: formatearInstanteBorrador(plazo.AbreEn), CierraEn: formatearInstanteBorrador(plazo.CierraEn),
		})
	}
	for _, requisito := range origen.Requisitos {
		obligatorio := requisito.Obligatorio
		*entrada.Requisitos = append(*entrada.Requisitos, requisitoBorradorJSON{
			Referencia: requisito.Referencia, Orden: requisito.Orden, Titulo: requisito.Titulo,
			Descripcion: requisito.Descripcion, Obligatorio: &obligatorio,
		})
	}
	for _, ayuda := range origen.Ayuda {
		*entrada.Ayuda = append(*entrada.Ayuda, ayudaBorradorJSON{
			Referencia: ayuda.Referencia, Categoria: ayuda.Categoria, Orden: ayuda.Orden,
			Pregunta: ayuda.Pregunta, Respuesta: ayuda.Respuesta,
		})
	}
	validada, err := convertirContenidoBorrador(&entrada)
	if err != nil || !contenidosEditablesIguales(validada, origen) {
		return contenidoEditableBorradorJSON{}, errSalidaBorradorInsegura
	}
	salida := contenidoEditableBorradorJSON{
		Tipo: origen.Tipo, Categorias: append([]string(nil), origen.Categorias...), Titulo: origen.Titulo,
		Resumen: origen.Resumen, Descripcion: origen.Descripcion,
		Plazos:     make([]plazoBorradorSalidaJSON, len(origen.Plazos)),
		Requisitos: make([]requisitoBorradorSalidaJSON, len(origen.Requisitos)),
		Ayuda:      make([]ayudaBorradorSalidaJSON, len(origen.Ayuda)),
	}
	for indice, plazo := range origen.Plazos {
		salida.Plazos[indice] = plazoBorradorSalidaJSON{
			Referencia: plazo.Referencia, Tipo: plazo.Tipo, Titulo: plazo.Titulo, Descripcion: plazo.Descripcion,
			AbreEn: formatearInstanteBorrador(plazo.AbreEn), CierraEn: formatearInstanteBorrador(plazo.CierraEn),
		}
	}
	for indice, requisito := range origen.Requisitos {
		salida.Requisitos[indice] = requisitoBorradorSalidaJSON{
			Referencia: requisito.Referencia, Orden: requisito.Orden, Titulo: requisito.Titulo,
			Descripcion: requisito.Descripcion, Obligatorio: requisito.Obligatorio,
		}
	}
	for indice, ayuda := range origen.Ayuda {
		salida.Ayuda[indice] = ayudaBorradorSalidaJSON{
			Referencia: ayuda.Referencia, Categoria: ayuda.Categoria, Orden: ayuda.Orden,
			Pregunta: ayuda.Pregunta, Respuesta: ayuda.Respuesta,
		}
	}
	return salida, nil
}

func convertirConfiguracionSalida(
	origen gobiernoconvocatorias.ConfiguracionLecturaBorrador,
) (configuracionLecturaBorradorJSON, error) {
	referencias := []gobiernoconvocatorias.ReferenciaConfiguracionLecturaBorrador{
		origen.Catalogos, origen.Calendario, origen.ReglasBaremacion,
		origen.FlujoProceso, origen.FlujoSolicitud, origen.Plantilla,
	}
	for _, referencia := range referencias {
		if !referenciaConfiguracionSalidaValida(referencia) {
			return configuracionLecturaBorradorJSON{}, errSalidaBorradorInsegura
		}
	}
	if len(origen.Documentos) < 1 || len(origen.Documentos) > 256 {
		return configuracionLecturaBorradorJSON{}, errSalidaBorradorInsegura
	}
	salida := configuracionLecturaBorradorJSON{
		Catalogos:        referenciaConfiguracionAJSON(origen.Catalogos),
		Calendario:       referenciaConfiguracionAJSON(origen.Calendario),
		ReglasBaremacion: referenciaConfiguracionAJSON(origen.ReglasBaremacion),
		FlujoProceso:     referenciaConfiguracionAJSON(origen.FlujoProceso),
		FlujoSolicitud:   referenciaConfiguracionAJSON(origen.FlujoSolicitud),
		Plantilla:        referenciaConfiguracionAJSON(origen.Plantilla),
		Documentos:       make([]documentoLecturaBorradorJSON, len(origen.Documentos)),
	}
	publicaciones, documentos, representaciones := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	firmas, custodias := map[string]struct{}{}, map[string]struct{}{}
	for indice, documento := range origen.Documentos {
		if !patronClaveBorrador.MatchString(documento.Rol) || !referenciaBorradorValida(documento.PublicacionRef, 512) ||
			!referenciaBorradorValida(documento.DocumentoRef, 512) || !enteroSeguroPositivo(documento.VersionDocumento) ||
			!referenciaBorradorValida(documento.RepresentacionRef, 512) || !patronHuellaBorrador.MatchString(documento.HuellaContenidoSHA256) ||
			!referenciaBorradorValida(documento.FirmaValidadaRef, 512) || !referenciaBorradorValida(documento.ReciboCustodiaRef, 512) ||
			insertarRepetida(publicaciones, documento.PublicacionRef) ||
			insertarRepetida(documentos, documento.DocumentoRef+"#"+strconv.Itoa(documento.VersionDocumento)) ||
			insertarRepetida(representaciones, documento.RepresentacionRef) ||
			insertarRepetida(firmas, documento.FirmaValidadaRef) || insertarRepetida(custodias, documento.ReciboCustodiaRef) {
			return configuracionLecturaBorradorJSON{}, errSalidaBorradorInsegura
		}
		salida.Documentos[indice] = documentoLecturaBorradorJSON(documento)
	}
	return salida, nil
}

func limitesBorradorValidos(l gobiernoconvocatorias.LimitesEdicionBorrador) bool {
	valores := []struct{ valor, maximo int }{
		{l.MaximoCategorias, 1024}, {l.MaximoPlazos, 64}, {l.MaximoRequisitos, 256},
		{l.MaximoDocumentos, 256}, {l.MaximoAyudas, 128}, {l.MaximoTitulo, 180},
		{l.MaximoResumen, 500}, {l.MaximoDescripcion, 12000}, {l.MaximoTituloPlazo, 180},
		{l.MaximoDescripcionPlazo, 1000}, {l.MaximoTituloRequisito, 180},
		{l.MaximoDescripcionRequisito, 3000}, {l.MaximoPreguntaAyuda, 300}, {l.MaximoRespuestaAyuda, 5000},
	}
	for _, valor := range valores {
		if valor.valor < 1 || valor.valor > valor.maximo {
			return false
		}
	}
	return true
}

func limitesAJSON(l gobiernoconvocatorias.LimitesEdicionBorrador) limiteEdicionBorradorJSON {
	return limiteEdicionBorradorJSON{
		MaximoCategorias: l.MaximoCategorias, MaximoPlazos: l.MaximoPlazos,
		MaximoRequisitos: l.MaximoRequisitos, MaximoDocumentos: l.MaximoDocumentos,
		MaximoAyudas: l.MaximoAyudas, MaximoTitulo: l.MaximoTitulo, MaximoResumen: l.MaximoResumen,
		MaximoDescripcion: l.MaximoDescripcion, MaximoTituloPlazo: l.MaximoTituloPlazo,
		MaximoDescripcionPlazo: l.MaximoDescripcionPlazo, MaximoTituloRequisito: l.MaximoTituloRequisito,
		MaximoDescripcionRequisito: l.MaximoDescripcionRequisito,
		MaximoPreguntaAyuda:        l.MaximoPreguntaAyuda, MaximoRespuestaAyuda: l.MaximoRespuestaAyuda,
	}
}

func opcionCatalogoValida(o gobiernoconvocatorias.OpcionCatalogoBorrador) bool {
	return referenciaBorradorValida(o.Referencia, 512) && enteroSeguroPositivo(o.Version) &&
		patronHuellaBorrador.MatchString(o.HuellaSHA256) && patronClaveBorrador.MatchString(o.Clave) &&
		cadenaBorradorValida(o.Etiqueta, 180, false, false)
}

func huellaCatalogoContradictoria(vistas map[string]string, identidad, huella string) bool {
	anterior, existe := vistas[identidad]
	vistas[identidad] = huella
	return existe && anterior != huella
}

func estadoAJSON(estado puertosbolsa.ReferenciaEstadoVersionConvocatoria) referenciaEstadoBorradorJSON {
	return referenciaEstadoBorradorJSON{
		Referencia: estado.Referencia, Revision: estado.Revision, HuellaEstadoSHA256: estado.HuellaEstadoSHA256,
	}
}

func estadoSalidaBorradorValido(estado puertosbolsa.ReferenciaEstadoVersionConvocatoria) bool {
	if estado.Validar() != nil || !enteroSeguroPositivo(estado.Revision) {
		return false
	}
	_, err := selectorDesdeReferenciaEstado(estado.Referencia)
	return err == nil
}

func etagEstadoBorrador(estado puertosbolsa.ReferenciaEstadoVersionConvocatoria) string {
	return fmt.Sprintf(`"vec-borrador-v1.r%d.sha256-%s"`, estado.Revision, estado.HuellaEstadoSHA256)
}

func selectorDesdeReferenciaEstado(referencia string) (puertosbolsa.SelectorVersionConvocatoriaExacta, error) {
	posicion := strings.LastIndexByte(referencia, '#')
	if posicion <= 0 || posicion == len(referencia)-1 {
		return puertosbolsa.SelectorVersionConvocatoriaExacta{}, errSalidaBorradorInsegura
	}
	secuencia, err := strconv.Atoi(referencia[posicion+1:])
	selector := puertosbolsa.SelectorVersionConvocatoriaExacta{ID: referencia[:posicion], Secuencia: secuencia}
	if err != nil || !identificadorRutaBorradorValido(selector.ID) ||
		strconv.Itoa(secuencia) != referencia[posicion+1:] || selector.Validar() != nil {
		return puertosbolsa.SelectorVersionConvocatoriaExacta{}, errSalidaBorradorInsegura
	}
	return selector, nil
}

func instanteSalidaBorradorValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Location() == time.UTC && instante.Nanosecond()%1000 == 0
}

func formatearInstanteBorrador(instante time.Time) string {
	return instante.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)
}

func capacidadesGlobalesAJSON(
	origen gobiernoconvocatorias.CapacidadesGlobalesBorrador,
) capacidadesGlobalesBorradorJSON {
	return capacidadesGlobalesBorradorJSON{Consultar: origen.Consultar, Crear: origen.Crear}
}

func capacidadesFilaAJSON(
	origen gobiernoconvocatorias.CapacidadesFilaBorrador,
) capacidadesFilaBorradorJSON {
	return capacidadesFilaBorradorJSON{Consultar: origen.Consultar, Actualizar: origen.Actualizar}
}

func referenciaConfiguracionSalidaValida(
	origen gobiernoconvocatorias.ReferenciaConfiguracionLecturaBorrador,
) bool {
	return referenciaBorradorValida(origen.Referencia, 512) && enteroSeguroPositivo(origen.Version) &&
		patronHuellaBorrador.MatchString(origen.HuellaSHA256)
}

func referenciaConfiguracionAJSON(
	origen gobiernoconvocatorias.ReferenciaConfiguracionLecturaBorrador,
) referenciaConfiguracionBorradorJSON {
	return referenciaConfiguracionBorradorJSON{
		Referencia: origen.Referencia, Version: origen.Version, HuellaSHA256: origen.HuellaSHA256,
	}
}

func contenidosEditablesIguales(
	a, b gobiernoconvocatorias.ContenidoEditableBorrador,
) bool {
	ordenar := func(c gobiernoconvocatorias.ContenidoEditableBorrador) gobiernoconvocatorias.ContenidoEditableBorrador {
		c.Categorias = append([]string(nil), c.Categorias...)
		c.Plazos = append([]dominiobolsa.PlazoConvocatoria(nil), c.Plazos...)
		c.Requisitos = append([]dominiobolsa.RequisitoConvocatoria(nil), c.Requisitos...)
		c.Ayuda = append([]dominiobolsa.AyudaConvocatoria(nil), c.Ayuda...)
		sort.Strings(c.Categorias)
		sort.Slice(c.Plazos, func(i, j int) bool { return c.Plazos[i].Referencia < c.Plazos[j].Referencia })
		sort.Slice(c.Requisitos, func(i, j int) bool { return c.Requisitos[i].Orden < c.Requisitos[j].Orden })
		sort.Slice(c.Ayuda, func(i, j int) bool { return c.Ayuda[i].Orden < c.Ayuda[j].Orden })
		return c
	}
	return reflect.DeepEqual(ordenar(a), ordenar(b))
}
