package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
)

const (
	rutaConfiguracionAnalisisContratacionTemporalDesarrollo = "/api/vec/contratacion-temporal/configuracion-analisis"
	esquemaConfiguracionAnalisisContratacionTemporal        = "vec.contratacion_temporal.configuracion_analisis.v1"
	artefactoAnalisisContratacionTemporalDesarrollo         = "artefacto:analisis:desarrollo:v1"
	causaAnalisisContratacionTemporalDesarrollo             = domain.ClaveCatalogo("necesidad_temporal")
	entradaRCAnalisisContratacionTemporalDesarrollo         = "rc:desarrollo:001"
	huellaEntradaRCAnalisisContratacionTemporalDesarrollo   = "3259ad24878afcc3e4cf6ad860377c7d84f6b2b5152dc61146686a8e69ee895b"
	dominioHMACAmbitoAnalisisDesarrollo                     = "vec.contratacion-temporal.analisis.ambito-idempotencia"
	dominioHMACSemanticaAnalisisDesarrollo                  = "vec.contratacion-temporal.analisis.huella-semantica"
)

var errAnalisisContratacionTemporalDesarrolloNoDisponible = errors.New(
	"contratacion temporal: analisis de desarrollo no disponible",
)

var modalidadesAnalisisContratacionTemporalDesarrollo = [...]domain.ClaveCatalogo{
	"sustitucion",
	"vacante",
	"acumulacion_tareas",
	"programa",
	"relevo",
}

func solicitudAnalisisContratacionTemporalDesarrolloValida(
	solicitud ports.SolicitudPrepararArtefactoAnalisis,
) bool {
	datos := solicitud.DatosFuncionales
	if solicitud.ArtefactoRef != artefactoAnalisisContratacionTemporalDesarrollo ||
		solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		!categoriaDeCatalogoDesarrollo(datos.CategoriaRef) ||
		!grupoSubgrupoDeCatalogoValido(datos.CategoriaRef, datos.GrupoSubgrupo) ||
		datos.CausaClave != causaAnalisisContratacionTemporalDesarrollo ||
		datos.EntradaRC.Referencia != entradaRCAnalisisContratacionTemporalDesarrollo ||
		!hmac.Equal(
			[]byte(datos.EntradaRC.HuellaSHA256),
			[]byte(huellaEntradaRCAnalisisContratacionTemporalDesarrollo),
		) {
		return false
	}
	for _, modalidad := range modalidadesAnalisisContratacionTemporalDesarrollo {
		if datos.ModalidadClave == modalidad {
			return true
		}
	}
	return false
}

type entradaRCConfiguracionAnalisisContratacionTemporalDesarrollo struct {
	Referencia   string `json:"referencia"`
	HuellaSHA256 string `json:"huella_sha256"`
	Etiqueta     string `json:"etiqueta"`
}

type configuracionAnalisisContratacionTemporalDesarrollo struct {
	SubsanacionDisponible bool                                                           `json:"subsanacion_disponible,omitempty"`
	Esquema               string                                                         `json:"esquema"`
	ArtefactoRef          string                                                         `json:"artefacto_ref"`
	Modalidades           []opcionClaveCatalogosAltaContratacionTemporalDesarrollo       `json:"modalidades"`
	Categorias            []categoriaCatalogosAltaContratacionTemporalDesarrollo         `json:"categorias"`
	Causas                []opcionClaveCatalogosAltaContratacionTemporalDesarrollo       `json:"causas"`
	EntradasRC            []entradaRCConfiguracionAnalisisContratacionTemporalDesarrollo `json:"entradas_rc"`
	MotivosRectificacion  []opcionClaveCatalogosAltaContratacionTemporalDesarrollo       `json:"motivos_rectificacion"`
}

type respuestaConfiguracionAnalisisContratacionTemporalDesarrollo struct {
	Data configuracionAnalisisContratacionTemporalDesarrollo `json:"data"`
}

func nuevaRutaConfiguracionAnalisisContratacionTemporalDesarrollo() (
	vechttp.RutaExacta,
	error,
) {
	return nuevaRutaConfiguracionAnalisisConSubsanacionYMotivosDesarrollo(
		false,
		fuenteMotivosRectificacionAnalisisDesarrollo{},
	)
}

func nuevaRutaConfiguracionAnalisisConSubsanacionDesarrollo(disponible bool) (vechttp.RutaExacta, error) {
	return nuevaRutaConfiguracionAnalisisConSubsanacionYMotivosDesarrollo(
		disponible,
		fuenteMotivosRectificacionAnalisisDesarrollo{},
	)
}

func nuevaRutaConfiguracionAnalisisContratacionTemporalDesarrolloConMotivos(
	motivos fuenteMotivosRectificacionAnalisisDesarrollo,
) (vechttp.RutaExacta, error) {
	return nuevaRutaConfiguracionAnalisisConSubsanacionYMotivosDesarrollo(false, motivos)
}

func nuevaRutaConfiguracionAnalisisConSubsanacionYMotivosDesarrollo(
	subsanacionDisponible bool,
	motivos fuenteMotivosRectificacionAnalisisDesarrollo,
) (vechttp.RutaExacta, error) {
	return vechttp.RutaExacta{
		Ruta: rutaConfiguracionAnalisisContratacionTemporalDesarrollo,
		Manejador: manejadorConfiguracionAnalisisContratacionTemporalDesarrollo{
			subsanacionDisponible: subsanacionDisponible,
			motivos:               motivos,
		},
	}, nil
}

type manejadorConfiguracionAnalisisContratacionTemporalDesarrollo struct {
	subsanacionDisponible bool
	motivos               fuenteMotivosRectificacionAnalisisDesarrollo
}

func (m manejadorConfiguracionAnalisisContratacionTemporalDesarrollo) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	prepararCabecerasCatalogosAltaContratacionTemporalDesarrollo(w)
	if r == nil || r.URL == nil {
		responderErrorConfiguracionAnalisisContratacionTemporalDesarrollo(
			w, r, http.StatusServiceUnavailable, "servicio_no_disponible",
		)
		return
	}
	if r.URL.Path != rutaConfiguracionAnalisisContratacionTemporalDesarrollo ||
		r.URL.RawQuery != "" || r.ContentLength != 0 ||
		len(r.TransferEncoding) != 0 ||
		cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(r.Header) {
		responderErrorConfiguracionAnalisisContratacionTemporalDesarrollo(
			w, r, http.StatusBadRequest, "solicitud_invalida",
		)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		responderErrorConfiguracionAnalisisContratacionTemporalDesarrollo(
			w, r, http.StatusMethodNotAllowed, "metodo_no_permitido",
		)
		return
	}
	configuracion := nuevaConfiguracionAnalisisContratacionTemporalDesarrollo(
		m.motivos.opciones(r.Context()),
	)
	configuracion.SubsanacionDisponible = m.subsanacionDisponible
	contenido, err := json.Marshal(
		respuestaConfiguracionAnalisisContratacionTemporalDesarrollo{
			Data: configuracion,
		},
	)
	if err != nil {
		responderErrorConfiguracionAnalisisContratacionTemporalDesarrollo(
			w, r, http.StatusServiceUnavailable, "servicio_no_disponible",
		)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(contenido)
	}
}

func nuevaConfiguracionAnalisisContratacionTemporalDesarrollo(
	motivos []opcionClaveCatalogosAltaContratacionTemporalDesarrollo,
) configuracionAnalisisContratacionTemporalDesarrollo {
	modalidades := make(
		[]opcionClaveCatalogosAltaContratacionTemporalDesarrollo,
		0,
		len(modalidadesAnalisisContratacionTemporalDesarrollo),
	)
	etiquetas := [...]string{
		"Sustitución",
		"Vacante",
		"Acumulación de tareas",
		"Programa",
		"Relevo",
	}
	for indice, modalidad := range modalidadesAnalisisContratacionTemporalDesarrollo {
		modalidades = append(
			modalidades,
			opcionClaveCatalogosAltaContratacionTemporalDesarrollo{
				Clave: string(modalidad), Etiqueta: etiquetas[indice],
			},
		)
	}
	return configuracionAnalisisContratacionTemporalDesarrollo{
		Esquema:      esquemaConfiguracionAnalisisContratacionTemporal,
		ArtefactoRef: artefactoAnalisisContratacionTemporalDesarrollo,
		Modalidades:  modalidades,
		Categorias:   categoriasSinteticasDesarrollo,
		Causas: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{{
			Clave:    string(causaAnalisisContratacionTemporalDesarrollo),
			Etiqueta: "Necesidad temporal",
		}},
		EntradasRC: []entradaRCConfiguracionAnalisisContratacionTemporalDesarrollo{{
			Referencia:   entradaRCAnalisisContratacionTemporalDesarrollo,
			HuellaSHA256: huellaEntradaRCAnalisisContratacionTemporalDesarrollo,
			Etiqueta:     "Retención de crédito sintética 001",
		}},
		MotivosRectificacion: motivos,
	}
}

func responderErrorConfiguracionAnalisisContratacionTemporalDesarrollo(
	w http.ResponseWriter,
	r *http.Request,
	estado int,
	codigo string,
) {
	contenido, err := json.Marshal(map[string]any{
		"error": map[string]string{
			"codigo": codigo,
			"clave_i18n": "api.contratacion_temporal.configuracion_analisis.error." +
				codigo,
		},
	})
	if err != nil {
		contenido = []byte(`{"error":{"codigo":"servicio_no_disponible"}}`)
		estado = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	if r == nil || r.Method != http.MethodHead {
		_, _ = w.Write(contenido)
	}
}

func nuevasDependenciasAnalisisContratacionTemporalDesarrollo(
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
	motivos fuenteMotivosRectificacionAnalisisDesarrollo,
) (*application.ServicioOperacionAnalisis, error) {
	if alta == nil || alta.soporte == nil || alta.autorizador == nil ||
		alta.postgresql.ejecucion == nil ||
		derivador == nil || !derivador.valido() {
		return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	artefactos, err := nuevoPreparadorFuentesAnalisisContratacionTemporalDesarrollo(
		derivador,
		reloj,
	)
	if err != nil {
		return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	sellador, err := nuevoSelladorHMACOperacionAnalisisDesarrollo(derivador)
	if err != nil {
		return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	preparaciones, err :=
		postgrescontratacion.NuevoPreparadorOperacionAnalisisPostgreSQL(
			alta.postgresql.ejecucion,
		)
	if err != nil {
		return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	transaccion, err :=
		postgrescontratacion.NuevaTransaccionOperacionesAnalisisPostgreSQL(
			alta.postgresql.ejecucion,
		)
	if err != nil {
		return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	servicio, err := application.NuevoServicioOperacionAnalisis(
		alta.soporte,
		artefactos,
		sellador,
		preparaciones,
		resolutorPoliticaOperacionAnalisisDesarrollo{motivos: motivos},
		seguridadvec.GeneradorReferenciasCriptograficas{},
		alta.autorizador,
		reloj,
		transaccion,
	)
	if err != nil {
		return nil, errAnalisisContratacionTemporalDesarrolloNoDisponible
	}
	return servicio, nil
}

type resolutorPoliticaOperacionAnalisisDesarrollo struct {
	motivos fuenteMotivosRectificacionAnalisisDesarrollo
}

func (r resolutorPoliticaOperacionAnalisisDesarrollo) ResolverPoliticaOperacionAnalisis(
	ctx context.Context,
	solicitud ports.SolicitudResolverPoliticaOperacionAnalisis,
) (ports.PoliticaOperacionAnalisis, error) {
	if contextoInterfazNulo(ctx) || solicitud.Validar() != nil ||
		solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		solicitud.ArtefactoRef != artefactoAnalisisContratacionTemporalDesarrollo ||
		solicitud.Flujo.DefinicionRef != "flujo:ct:desarrollo" ||
		solicitud.Flujo.Version != 1 ||
		!hmac.Equal(
			[]byte(solicitud.Flujo.HuellaSHA256),
			[]byte(huellaAltaContratacionTemporalDesarrollo("flujo")),
		) ||
		solicitud.FasePrevia != domain.ClaveFase("solicitud") ||
		solicitud.EstadoPrevio != domain.EstadoEnCurso {
		return ports.PoliticaOperacionAnalisis{},
			ports.ErrPoliticaOperacionAnalisisNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.PoliticaOperacionAnalisis{}, errors.Join(
			ports.ErrPoliticaOperacionAnalisisNoDisponible,
			err,
		)
	}
	politica := ports.PoliticaOperacionAnalisis{
		Operacion:             solicitud.Operacion,
		OrganizacionRef:       solicitud.OrganizacionRef,
		ExpedienteRef:         solicitud.ExpedienteRef,
		VersionExpediente:     solicitud.VersionExpediente,
		FasePrevia:            solicitud.FasePrevia,
		EstadoPrevio:          solicitud.EstadoPrevio,
		ActorRef:              solicitud.ActorRef,
		ArtefactoRef:          solicitud.ArtefactoRef,
		ArtefactoHuellaSHA256: solicitud.ArtefactoHuellaSHA256,
		DefinicionRef:         "politica:ct:desarrollo:analisis:v1",
		Version:               1,
		HuellaSHA256:          huellaAltaContratacionTemporalDesarrollo("politica-analisis"),
		Accion:                domain.ClaveCatalogo(ports.AccionRegistrarAnalisis),
		Finalidad:             finalidadAnalisisContratacionTemporalDesarrollo,
		UnidadRef:             unidadCoberturaContratacionTemporalDesarrollo,
		MotivoAutorizacion:    referenciaMotivoAutorizacionAnalisisDesarrollo("registro"),
		EvaluadaEn:            solicitud.Instante,
	}
	if solicitud.Operacion == ports.OperacionRectificarAnalisis {
		motivo, err := r.motivos.resolverMotivo(ctx, solicitud.MotivoRectificacionClave)
		if err != nil {
			return ports.PoliticaOperacionAnalisis{}, ports.ErrPoliticaOperacionAnalisisNoDisponible
		}
		huella, err := huellaPoliticaRectificacionAnalisisDesarrollo(motivo)
		if err != nil {
			return ports.PoliticaOperacionAnalisis{}, ports.ErrPoliticaOperacionAnalisisNoDisponible
		}
		politica.Accion = domain.ClaveCatalogo(ports.AccionRectificarAnalisis)
		politica.MotivoAutorizacion = referenciaMotivoAutorizacionAnalisisDesarrollo("rectificacion")
		politica.ExigeActorDistinto = true
		politica.ActorAnalisisAnteriorRef = solicitud.ActorAnalisisAnteriorRef
		politica.MotivoRectificacion = motivo
		politica.HuellaSHA256 = huella
	}
	if politica.ValidarPara(solicitud) != nil {
		return ports.PoliticaOperacionAnalisis{},
			ports.ErrPoliticaOperacionAnalisisNoDisponible
	}
	return politica, nil
}

const identificadorCanonPoliticaRectificacionAnalisisDesarrolloV1 = "vec.contratacion-temporal.analisis.politica-rectificacion.v1"

// canonPoliticaRectificacionAnalisisDesarrolloV1 fija el formato de la
// definición efectiva. Es una estructura cerrada, sin mapas ni fmt, para que
// la huella sea estable y evidencie el motivo publicado que habilita el acto.
type canonPoliticaRectificacionAnalisisDesarrolloV1 struct {
	Canon                        string `json:"canon"`
	DefinicionRef                string `json:"definicion_ref"`
	Version                      uint64 `json:"version"`
	Operacion                    string `json:"operacion"`
	Accion                       string `json:"accion"`
	Finalidad                    string `json:"finalidad"`
	UnidadRef                    string `json:"unidad_ref"`
	ExigeActorDistinto           bool   `json:"exige_actor_distinto"`
	MotivoAutorizacionCatalogoID string `json:"motivo_autorizacion_catalogo_id"`
	MotivoAutorizacionVersion    int    `json:"motivo_autorizacion_version"`
	MotivoAutorizacionHuella     string `json:"motivo_autorizacion_huella_sha256"`
	MotivoAutorizacionEntrada    string `json:"motivo_autorizacion_entrada"`
	CatalogoID                   string `json:"catalogo_id"`
	CatalogoVersion              int    `json:"catalogo_version"`
	CatalogoHuellaSHA256         string `json:"catalogo_huella_sha256"`
	EntradaClave                 string `json:"entrada_clave"`
	ClaveMensajeI18N             string `json:"clave_mensaje_i18n"`
	VigenteDesde                 string `json:"vigente_desde"`
	VigenteHasta                 string `json:"vigente_hasta,omitempty"`
}

func huellaPoliticaRectificacionAnalisisDesarrollo(
	motivo ports.MotivoRectificacionGobernado,
) (string, error) {
	if motivo.ValidarPara(domain.ClaveCatalogo(motivo.ReferenciaCatalogo.EntradaClave)) != nil {
		return "", ports.ErrPoliticaOperacionAnalisisNoDisponible
	}
	motivoAutorizacion := referenciaMotivoAutorizacionAnalisisDesarrollo("rectificacion")
	if motivoAutorizacion.Validar() != nil {
		return "", ports.ErrPoliticaOperacionAnalisisNoDisponible
	}
	canon := canonPoliticaRectificacionAnalisisDesarrolloV1{
		Canon:         identificadorCanonPoliticaRectificacionAnalisisDesarrolloV1,
		DefinicionRef: "politica:ct:desarrollo:analisis:v1", Version: 1,
		Operacion:                    string(ports.OperacionRectificarAnalisis),
		Accion:                       string(ports.AccionRectificarAnalisis),
		Finalidad:                    string(finalidadAnalisisContratacionTemporalDesarrollo),
		UnidadRef:                    unidadCoberturaContratacionTemporalDesarrollo,
		ExigeActorDistinto:           true,
		MotivoAutorizacionCatalogoID: motivoAutorizacion.CatalogoID,
		MotivoAutorizacionVersion:    motivoAutorizacion.CatalogoVersion,
		MotivoAutorizacionHuella:     motivoAutorizacion.CatalogoHuellaSHA256,
		MotivoAutorizacionEntrada:    motivoAutorizacion.EntradaClave,
		CatalogoID:                   motivo.ReferenciaCatalogo.CatalogoID,
		CatalogoVersion:              motivo.ReferenciaCatalogo.CatalogoVersion,
		CatalogoHuellaSHA256:         motivo.ReferenciaCatalogo.CatalogoHuellaSHA256,
		EntradaClave:                 motivo.ReferenciaCatalogo.EntradaClave,
		ClaveMensajeI18N:             string(motivo.ClaveMensajeI18N),
		VigenteDesde:                 motivo.VigenteDesde.Format(time.RFC3339Nano),
		VigenteHasta:                 fechaCanonicaMotivoRectificacionAnalisis(motivo.VigenteHasta),
	}
	serializado, err := json.Marshal(canon)
	if err != nil {
		return "", ports.ErrPoliticaOperacionAnalisisNoDisponible
	}
	suma := sha256.Sum256(serializado)
	return hex.EncodeToString(suma[:]), nil
}

func fechaCanonicaMotivoRectificacionAnalisis(instante time.Time) string {
	if instante.IsZero() {
		return ""
	}
	return instante.Format(time.RFC3339Nano)
}

type selladorHMACOperacionAnalisisDesarrollo struct {
	derivador *derivadorIdentidadOperacionDesarrollo
}

func nuevoSelladorHMACOperacionAnalisisDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
) (*selladorHMACOperacionAnalisisDesarrollo, error) {
	if derivador == nil || !derivador.valido() {
		return nil, ports.ErrOperacionAnalisisInvalida
	}
	return &selladorHMACOperacionAnalisisDesarrollo{derivador: derivador}, nil
}

func (s *selladorHMACOperacionAnalisisDesarrollo) SellarOperacionAnalisis(
	ctx context.Context,
	preimagenes ports.PreimagenesOperacionAnalisis,
) (ports.SellosOperacionAnalisis, error) {
	vacios := ports.SellosOperacionAnalisis{}
	if s == nil || s.derivador == nil || !s.derivador.valido() ||
		contextoInterfazNulo(ctx) {
		return vacios, ports.ErrOperacionAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacios, errors.Join(ports.ErrOperacionAnalisisInvalida, err)
	}
	ambito, err := preimagenes.BytesAmbito()
	if err != nil {
		return vacios, ports.ErrOperacionAnalisisInvalida
	}
	defer borrarBytes(ambito)
	semantica, err := preimagenes.BytesSemantica()
	if err != nil {
		return vacios, ports.ErrOperacionAnalisisInvalida
	}
	defer borrarBytes(semantica)
	resultados, err := s.derivador.calcularHMAC(ambito, semantica)
	if err != nil {
		return vacios, ports.ErrOperacionAnalisisInvalida
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	ambitos, err := nuevaColeccionHMACOperacionAnalisisDesarrollo(
		resultados,
		dominioHMACAmbitoAnalisisDesarrollo,
		true,
	)
	if err != nil {
		return vacios, err
	}
	huellas, err := nuevaColeccionHMACOperacionAnalisisDesarrollo(
		resultados,
		dominioHMACSemanticaAnalisisDesarrollo,
		false,
	)
	if err != nil {
		return vacios, err
	}
	if err := ctx.Err(); err != nil {
		return vacios, errors.Join(ports.ErrOperacionAnalisisInvalida, err)
	}
	sellos := ports.SellosOperacionAnalisis{
		AmbitosIdempotenciaHMAC: ambitos,
		HuellasSemanticasHMAC:   huellas,
	}
	if sellos.Validar() != nil {
		return vacios, ports.ErrOperacionAnalisisInvalida
	}
	return sellos, nil
}

func nuevaColeccionHMACOperacionAnalisisDesarrollo(
	resultados []resultadoHMACIdempotenciaDesarrollo,
	dominio string,
	usarAmbito bool,
) (ports.ColeccionSellosHMAC, error) {
	if len(resultados) < minimoGeneracionesIdempotenciaDesarrollo ||
		len(resultados) > maximoGeneracionesIdempotenciaDesarrollo {
		return ports.ColeccionSellosHMAC{}, ports.ErrOperacionAnalisisInvalida
	}
	sellos := make([]string, len(resultados))
	for indice := range resultados {
		valor := resultados[indice].huellaSolicitud[:]
		if usarAmbito {
			valor = resultados[indice].localizador[:]
		}
		sellos[indice] = fmt.Sprintf(
			"hmac-sha256:%s/v%d:%s",
			dominio,
			resultados[indice].generacion,
			hex.EncodeToString(valor),
		)
	}
	coleccion, err := ports.NuevaColeccionSellosHMAC(sellos[0], sellos[1:])
	if err != nil {
		return ports.ColeccionSellosHMAC{}, ports.ErrOperacionAnalisisInvalida
	}
	return coleccion, nil
}

var (
	_ ports.ResolutorPoliticaOperacionAnalisis = resolutorPoliticaOperacionAnalisisDesarrollo{}
	_ ports.SelladorOperacionAnalisis          = (*selladorHMACOperacionAnalisisDesarrollo)(nil)
)
