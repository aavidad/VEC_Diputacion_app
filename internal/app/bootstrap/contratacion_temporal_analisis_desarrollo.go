package bootstrap

import (
	"context"
	"crypto/hmac"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

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
		datos.CategoriaRef != categoriaAltaContratacionTemporalDesarrollo ||
		datos.GrupoSubgrupo != grupoSubgrupoAltaContratacionTemporalDesarrollo ||
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
	Esquema              string                                                         `json:"esquema"`
	ArtefactoRef         string                                                         `json:"artefacto_ref"`
	Modalidades          []opcionClaveCatalogosAltaContratacionTemporalDesarrollo       `json:"modalidades"`
	Categorias           []categoriaCatalogosAltaContratacionTemporalDesarrollo         `json:"categorias"`
	Causas               []opcionClaveCatalogosAltaContratacionTemporalDesarrollo       `json:"causas"`
	EntradasRC           []entradaRCConfiguracionAnalisisContratacionTemporalDesarrollo `json:"entradas_rc"`
	MotivosRectificacion []opcionClaveCatalogosAltaContratacionTemporalDesarrollo       `json:"motivos_rectificacion"`
}

type respuestaConfiguracionAnalisisContratacionTemporalDesarrollo struct {
	Data configuracionAnalisisContratacionTemporalDesarrollo `json:"data"`
}

func nuevaRutaConfiguracionAnalisisContratacionTemporalDesarrollo() (
	vechttp.RutaExacta,
	error,
) {
	return vechttp.RutaExacta{
		Ruta:      rutaConfiguracionAnalisisContratacionTemporalDesarrollo,
		Manejador: manejadorConfiguracionAnalisisContratacionTemporalDesarrollo{},
	}, nil
}

type manejadorConfiguracionAnalisisContratacionTemporalDesarrollo struct{}

func (manejadorConfiguracionAnalisisContratacionTemporalDesarrollo) ServeHTTP(
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
	contenido, err := json.Marshal(
		respuestaConfiguracionAnalisisContratacionTemporalDesarrollo{
			Data: nuevaConfiguracionAnalisisContratacionTemporalDesarrollo(),
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

func nuevaConfiguracionAnalisisContratacionTemporalDesarrollo() configuracionAnalisisContratacionTemporalDesarrollo {
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
		Categorias: []categoriaCatalogosAltaContratacionTemporalDesarrollo{{
			Referencia: categoriaAltaContratacionTemporalDesarrollo,
			Etiqueta:   "Categoría C2",
			GruposSubgrupos: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{{
				Clave: grupoSubgrupoAltaContratacionTemporalDesarrollo, Etiqueta: "Grupo C2",
			}},
		}},
		Causas: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{{
			Clave:    string(causaAnalisisContratacionTemporalDesarrollo),
			Etiqueta: "Necesidad temporal",
		}},
		EntradasRC: []entradaRCConfiguracionAnalisisContratacionTemporalDesarrollo{{
			Referencia:   entradaRCAnalisisContratacionTemporalDesarrollo,
			HuellaSHA256: huellaEntradaRCAnalisisContratacionTemporalDesarrollo,
			Etiqueta:     "Retención de crédito sintética 001",
		}},
		MotivosRectificacion: make(
			[]opcionClaveCatalogosAltaContratacionTemporalDesarrollo,
			0,
		),
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
		resolutorPoliticaOperacionAnalisisDesarrollo{},
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

type resolutorPoliticaOperacionAnalisisDesarrollo struct{}

func (resolutorPoliticaOperacionAnalisisDesarrollo) ResolverPoliticaOperacionAnalisis(
	ctx context.Context,
	solicitud ports.SolicitudResolverPoliticaOperacionAnalisis,
) (ports.PoliticaOperacionAnalisis, error) {
	if contextoInterfazNulo(ctx) || solicitud.Validar() != nil ||
		solicitud.Operacion != ports.OperacionRegistrarAnalisis ||
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
	if politica.ValidarPara(solicitud) != nil {
		return ports.PoliticaOperacionAnalisis{},
			ports.ErrPoliticaOperacionAnalisisNoDisponible
	}
	return politica, nil
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
