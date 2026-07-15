package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionDocumentoLogicoGenerado = "vec.documento.logico.generado"
	eventoDocumentoLogicoGenerado = "vec.documento.logico.generado"
	vigenciaReservaDocumental     = 5 * time.Minute
)

// OrdenGenerarDocumentoLogico produce una unica version administrativa con
// una o varias representaciones tecnicas. ClaveIdempotencia debe ser opaca,
// aleatoria y estable en todos los reintentos de la misma operacion.
type OrdenGenerarDocumentoLogico struct {
	Principal         domain.Principal
	PerfilActivo      string
	RepresentadoRef   string
	Finalidad         string
	Clasificacion     string
	ClaveIdempotencia string
	PlantillaID       string
	PlantillaVersion  int
	Relaciones        []domain.RelacionDocumento
	Representaciones  []domain.SolicitudRepresentacionDocumento
	Datos             map[string]string
	Motivo            string
	CorrelacionRef    string
}

func (s *ServicioDocumental) GenerarDocumentoLogico(ctx context.Context, orden OrdenGenerarDocumentoLogico) (resultado domain.ResultadoGeneracionDocumento, err error) {
	if err := ctx.Err(); err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}
	if err := orden.Principal.Validate(); err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}
	if strings.TrimSpace(orden.PerfilActivo) == "" || strings.TrimSpace(orden.Finalidad) == "" ||
		strings.TrimSpace(orden.Clasificacion) == "" ||
		orden.ClaveIdempotencia != strings.TrimSpace(orden.ClaveIdempotencia) || orden.ClaveIdempotencia == "" ||
		strings.TrimSpace(orden.PlantillaID) == "" ||
		orden.PlantillaVersion < 1 || strings.TrimSpace(orden.Motivo) == "" ||
		strings.TrimSpace(orden.CorrelacionRef) == "" || len(orden.Representaciones) == 0 {
		return domain.ResultadoGeneracionDocumento{}, ErrOrdenDocumentalInvalida
	}
	relaciones, err := domain.CanonizarRelacionesDocumento(orden.Relaciones)
	if err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}
	expedienteRef, err := referenciaExpedientePrincipal(relaciones)
	if err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}
	solicitudes, err := domain.CanonizarSolicitudesRepresentacionDocumento(orden.Representaciones)
	if err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}

	plantilla, err := s.catalogo.ObtenerPlantilla(ctx, orden.PlantillaID, orden.PlantillaVersion)
	if err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}
	if err := plantilla.Validar(); err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}
	if plantilla.Estado != domain.EstadoPlantillaPublicada {
		return domain.ResultadoGeneracionDocumento{}, domain.ErrPlantillaNoPublicada
	}
	if !orden.Principal.AuthAssurance.Cumple(plantilla.GarantiaMinima) {
		return domain.ResultadoGeneracionDocumento{}, domain.ErrGarantiaInsuficiente
	}
	for _, solicitud := range solicitudes {
		if !plantilla.AdmiteFormato(solicitud.Formato) {
			return domain.ResultadoGeneracionDocumento{}, domain.ErrFormatoDocumentoInvalido
		}
		if _, existe := s.renderizadores[solicitud.Formato]; !existe {
			return domain.ResultadoGeneracionDocumento{}, ErrRenderizadorNoDisponible
		}
	}
	contenidoFusionado, err := plantilla.Fusionar(orden.Datos)
	if err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}
	huellaPlantilla, err := plantilla.HuellaSHA256()
	if err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}
	huellaDatos, err := s.selladorDatos.SellarDatos(ctx, datosDocumentoLogicoCanonicos(plantilla, orden.Datos))
	if err != nil {
		return domain.ResultadoGeneracionDocumento{}, fmt.Errorf("sellar datos documentales: %w", err)
	}
	huellaFuente, err := s.selladorDatos.SellarDatos(ctx, fuenteDocumentoLogicoCanonica(plantilla, huellaPlantilla, huellaDatos, relaciones))
	if err != nil {
		return domain.ResultadoGeneracionDocumento{}, fmt.Errorf("sellar fuente documental: %w", err)
	}
	huellaSolicitud, err := s.selladorSolicitud.SellarSolicitudDocumento(ctx,
		solicitudDocumentoLogicoCanonica(orden, plantilla, huellaPlantilla, relaciones, solicitudes))
	if err != nil {
		return domain.ResultadoGeneracionDocumento{}, fmt.Errorf("sellar solicitud documental: %w", err)
	}
	if !huellaHMACDocumentalValida(huellaSolicitud) {
		return domain.ResultadoGeneracionDocumento{}, ErrConfirmacionContenidoInvalida
	}

	// Todas las representaciones se renderizan y validan antes de formular la
	// solicitud al PDP. Asi la decision compromete las huellas de los bytes
	// reales y nunca una estimacion, un formato pendiente o un plan parcial.
	type salidaRenderizada struct {
		solicitud domain.SolicitudRepresentacionDocumento
		contenido []byte
		tamano    int64
		huella    string
	}
	salidas := make([]salidaRenderizada, 0, len(solicitudes))
	var totalBytes int64
	for _, solicitud := range solicitudes {
		renderizador := s.renderizadores[solicitud.Formato]
		contenido, err := renderizador.Renderizar(ctx, contenidoFusionado)
		if err != nil {
			return domain.ResultadoGeneracionDocumento{}, fmt.Errorf("renderizar %s: %w", solicitud.Formato, err)
		}
		tamano := int64(len(contenido))
		if tamano < 1 || tamano > s.limiteBytes || totalBytes > s.limiteBytes-tamano {
			return domain.ResultadoGeneracionDocumento{}, ErrDocumentoDemasiadoGrande
		}
		totalBytes += tamano
		if err := renderizador.ValidarSalida(ctx, contenido); err != nil {
			return domain.ResultadoGeneracionDocumento{}, fmt.Errorf("validar salida %s: %w", solicitud.Formato, err)
		}
		suma := sha256.Sum256(contenido)
		salidas = append(salidas, salidaRenderizada{
			solicitud: solicitud, contenido: append([]byte(nil), contenido...), tamano: tamano,
			huella: hex.EncodeToString(suma[:]),
		})
	}

	instanteReserva := s.reloj.Ahora().UTC()
	if instanteReserva.IsZero() {
		return domain.ResultadoGeneracionDocumento{}, ErrConfirmacionContenidoInvalida
	}
	reserva, err := s.repositorioLogico.ReservarGeneracion(ctx, ports.SolicitudReservarGeneracionDocumento{
		ClaveIdempotencia:   strings.TrimSpace(orden.ClaveIdempotencia),
		PrincipalID:         strings.TrimSpace(orden.Principal.ID),
		HuellaSolicitudHMAC: huellaSolicitud,
		SolicitadaEn:        instanteReserva,
		ExpiraEn:            instanteReserva.Add(vigenciaReservaDocumental),
	})
	if err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}
	if reserva.Repetida {
		if reserva.Token != "" || !reserva.Resultado.Repetida || reserva.Resultado.Validar() != nil {
			return domain.ResultadoGeneracionDocumento{}, ports.ErrReservaDocumentoNoValida
		}
		// La idempotencia no conserva privilegios. Antes de devolver un resultado
		// anterior se exige de nuevo el permiso publicado de la plantilla; no se
		// ejecuta ni reserva ningun efecto de almacen.
		if _, err := s.exigirAutorizacionDocumental(ctx, orden.Principal, orden.PerfilActivo,
			plantilla.PermisoGenerar, domain.RecursoAutorizable{
				Referencia: expedienteRef,
				ModuloID:   plantilla.ModuloID,
				Tipo:       "expediente",
				Atributos: map[string]string{
					"plantilla_id": plantilla.ID, "tipo_documental": plantilla.TipoDocumental,
					"resultado_idempotente": "confirmado",
				},
			}, orden.Finalidad, orden.CorrelacionRef, orden.Motivo); err != nil {
			return domain.ResultadoGeneracionDocumento{}, err
		}
		return reserva.Resultado, nil
	}
	if strings.TrimSpace(reserva.Token) == "" {
		return domain.ResultadoGeneracionDocumento{}, ports.ErrReservaDocumentoNoValida
	}
	confirmada := false
	efectoReservado := false
	defer func() {
		if confirmada || efectoReservado {
			return
		}
		ctxLimpieza, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancelar()
		_ = s.repositorioLogico.AbandonarGeneracion(ctxLimpieza, reserva.Token)
	}()

	documentoID, err := s.generadorID.NuevoIDDocumento()
	if err != nil {
		return domain.ResultadoGeneracionDocumento{}, fmt.Errorf("generar identificador documental: %w", err)
	}
	if documentoID != strings.TrimSpace(documentoID) || documentoID == "" {
		return domain.ResultadoGeneracionDocumento{}, ErrConfirmacionContenidoInvalida
	}
	creadoEn := s.reloj.Ahora().UTC()
	if creadoEn.IsZero() || creadoEn.Before(instanteReserva) {
		return domain.ResultadoGeneracionDocumento{}, ErrConfirmacionContenidoInvalida
	}
	documento := domain.DocumentoLogico{
		ID:       documentoID,
		Version:  1,
		Revision: 1,
		Plantilla: domain.ReferenciaPlantillaDocumento{
			ID:           plantilla.ID,
			Version:      plantilla.Version,
			HuellaSHA256: huellaPlantilla,
		},
		ModuloID:         plantilla.ModuloID,
		TipoDocumental:   plantilla.TipoDocumental,
		Clasificacion:    strings.TrimSpace(orden.Clasificacion),
		Relaciones:       relaciones,
		Estado:           domain.EstadoDocumentoLogicoBorrador,
		HuellaDatosHMAC:  huellaDatos,
		HuellaFuenteHMAC: huellaFuente,
		CreadoPor:        strings.TrimSpace(orden.Principal.ID),
		CreadoEn:         creadoEn,
		CorrelacionRef:   strings.TrimSpace(orden.CorrelacionRef),
		Motivo:           strings.TrimSpace(orden.Motivo),
		ENI: domain.MetadatosENI{
			Identificador:     documentoID,
			Organo:            s.organoENI,
			Origen:            "administracion",
			EstadoElaboracion: "original",
			TipoDocumental:    plantilla.TipoDocumental,
			FechaCaptura:      creadoEn,
		},
	}
	if err := documento.Validar(); err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}
	contenidosPlan := make([]contenidoGeneracionDocumental, 0, len(salidas))
	for _, salida := range salidas {
		representacionID := referenciaRepresentacionInicial(documentoID, salida.solicitud)
		contenidosPlan = append(contenidosPlan, contenidoGeneracionDocumental{
			declaracion: ports.DeclaracionRepresentacionGeneracionDocumental{
				ReferenciaLogica:  representacionID,
				ClaveIdempotencia: "contenido-representacion:" + representacionID,
				Formato:           salida.solicitud.Formato, Zona: ports.ZonaAlmacenAdmitida,
				MIME: salida.solicitud.Formato.MIME(), Tamano: salida.tamano, HuellaSHA256: salida.huella,
			},
			contenido: salida.contenido,
		})
	}
	plan, err := s.ejecutarPlanGeneracionDocumental(ctx, solicitudPlanGeneracionDocumental{
		principal: orden.Principal, perfilActivo: orden.PerfilActivo,
		finalidad: orden.Finalidad, motivo: orden.Motivo, correlacionRef: orden.CorrelacionRef,
		clasificacion: orden.Clasificacion, plantilla: plantilla,
		recursoBase: domain.RecursoAutorizable{
			Referencia: expedienteRef, ModuloID: plantilla.ModuloID, Tipo: "expediente",
			Atributos: map[string]string{
				"plantilla_id": plantilla.ID, "tipo_documental": plantilla.TipoDocumental,
				"representaciones": strconv.Itoa(len(solicitudes)),
			},
		},
		sujetoRef:       sujetoDocumentoRef(orden.Principal.ID, orden.RepresentadoRef),
		ambitoSujetoRef: "documento_logico:" + documentoID + ":" + huellaSolicitud,
		operacionRef:    "almacenar-documento-logico:" + documentoID,
		cargaRef:        documentoID,
		efectoRef:       "efecto-generacion-documento-logico:" + huellaSolicitud,
		huellaSolicitud: huellaSolicitud, contenidos: contenidosPlan,
	})
	efectoReservado = plan.efectoReservado
	if err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}

	representaciones := make([]domain.RepresentacionDocumento, 0, len(salidas))
	evidenciasAlmacen := make([]string, 0, len(salidas))
	conectoresAlmacen := make(map[string]struct{}, len(salidas))
	versionesObjetos := make([]string, 0, len(salidas))
	for _, salida := range salidas {
		solicitud := salida.solicitud
		representacionID := referenciaRepresentacionInicial(documentoID, solicitud)
		guardado, existe := plan.pasos[representacionID]
		if !existe || guardado.objeto.Validar() != nil || guardado.evidenciaOperacion == "" {
			return domain.ResultadoGeneracionDocumento{}, ErrConfirmacionContenidoInvalida
		}
		evidenciasAlmacen = append(evidenciasAlmacen, guardado.evidenciaOperacion)
		conectoresAlmacen[guardado.conectorID] = struct{}{}
		versionesObjetos = append(versionesObjetos, guardado.objeto.Referencia+"@"+guardado.objeto.Version)
		representaciones = append(representaciones, domain.RepresentacionDocumento{
			ID:                    representacionID,
			Documento:             documento.Referencia(),
			Tipo:                  solicitud.Tipo,
			Formato:               solicitud.Formato,
			MIME:                  solicitud.Formato.MIME(),
			NombreFichero:         nombreFicheroRepresentacion(plantilla.TipoDocumental, documentoID, solicitud),
			Tamano:                salida.tamano,
			HuellaContenidoSHA256: salida.huella,
			HuellaFuenteHMAC:      huellaFuente,
			ReferenciaContenido:   guardado.objeto.Referencia,
			EstadoTecnico:         domain.EstadoRepresentacionDisponible,
			EstadoAntivirus:       domain.EstadoAntivirusNoAplica,
			GeneradaPor:           strings.TrimSpace(orden.Principal.ID),
			GeneradaEn:            creadoEn,
		})
	}
	resultado = domain.ResultadoGeneracionDocumento{Documento: documento, Representaciones: representaciones}
	if err := resultado.Validar(); err != nil {
		return domain.ResultadoGeneracionDocumento{}, err
	}
	sort.Strings(evidenciasAlmacen)
	listaConectoresAlmacen := make([]string, 0, len(conectoresAlmacen))
	for conector := range conectoresAlmacen {
		listaConectoresAlmacen = append(listaConectoresAlmacen, conector)
	}
	sort.Strings(listaConectoresAlmacen)
	sort.Strings(versionesObjetos)

	confirmadaEn := s.reloj.Ahora().UTC()
	if confirmadaEn.IsZero() || confirmadaEn.Before(creadoEn) {
		return domain.ResultadoGeneracionDocumento{}, ErrConfirmacionContenidoInvalida
	}
	referenciaDocumento := documento.ID + ":" + strconv.Itoa(documento.Version)
	traza := domain.AuditEntry{
		ActorID:              documento.CreadoPor,
		ActorProfile:         strings.TrimSpace(orden.PerfilActivo),
		ActorRoles:           append([]string(nil), orden.Principal.Roles...),
		RepresentedSubjectID: strings.TrimSpace(orden.RepresentadoRef),
		AuthMethod:           orden.Principal.AuthMethod,
		AuthAssurance:        orden.Principal.AuthAssurance,
		AuthorizationRef:     plan.decision.DecisionRef,
		Purpose:              strings.TrimSpace(orden.Finalidad),
		Action:               AccionDocumentoLogicoGenerado,
		ModuleID:             documento.ModuloID,
		SubjectRef:           referenciaDocumento,
		ObjectVersion:        documento.Version,
		ExpedienteRef:        expedienteRef,
		DocumentRef:          referenciaDocumento,
		RuleRef:              plantilla.ID + ":" + strconv.Itoa(plantilla.Version),
		Reason:               documento.Motivo,
		Result:               "correcto",
		AfterHash:            documento.HuellaFuenteHMAC,
		CorrelationRef:       documento.CorrelacionRef,
		OccurredAt:           confirmadaEn,
		Metadata: map[string]string{
			"almacen_conectores":        strings.Join(listaConectoresAlmacen, ","),
			"almacen_evidencias_refs":   strings.Join(evidenciasAlmacen, ","),
			"almacen_objetos_versiones": strings.Join(versionesObjetos, ","),
			"huella_datos_hmac":         documento.HuellaDatosHMAC,
			"huella_plantilla":          documento.Plantilla.HuellaSHA256,
			"clasificacion":             documento.Clasificacion,
			"representaciones":          strconv.Itoa(len(representaciones)),
			"solicitudes_canonicas":     serializarSolicitudesRepresentacion(solicitudes),
		},
	}
	evento := domain.Event{
		Type:       eventoDocumentoLogicoGenerado,
		ModuleID:   documento.ModuloID,
		SubjectRef: referenciaDocumento,
		ActorID:    documento.CreadoPor,
		OccurredAt: confirmadaEn,
		Payload: map[string]string{
			"documento_ref":      documento.ID,
			"documento_version":  strconv.Itoa(documento.Version),
			"huella_fuente_hmac": documento.HuellaFuenteHMAC,
			"representaciones":   strconv.Itoa(len(representaciones)),
		},
	}
	if err := s.repositorioLogico.ConfirmarGeneracionLogica(ctx, reserva.Token, huellaSolicitud, confirmadaEn, resultado, traza, evento); err != nil {
		return domain.ResultadoGeneracionDocumento{}, fmt.Errorf("confirmar generacion documental logica: %w", err)
	}
	confirmada = true
	return resultado, nil
}

func referenciaExpedientePrincipal(relaciones []domain.RelacionDocumento) (string, error) {
	requisito := domain.RequisitoRelacionDocumento{
		Tipo:   domain.TipoRelacionExpediente,
		Rol:    "principal",
		Minimo: 1,
		Maximo: 1,
	}
	if err := domain.ValidarRequisitosRelacionesDocumento(relaciones, []domain.RequisitoRelacionDocumento{requisito}); err != nil {
		return "", err
	}
	for _, relacion := range relaciones {
		if relacion.Tipo == requisito.Tipo && relacion.Rol == requisito.Rol {
			return relacion.Referencia, nil
		}
	}
	return "", domain.ErrRequisitoRelacionDocumentoIncumplido
}

func datosDocumentoLogicoCanonicos(plantilla domain.PlantillaDocumento, datos map[string]string) []byte {
	claves := make([]string, 0, len(datos))
	for clave := range datos {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	var salida strings.Builder
	escribirCampoCanonico(&salida, "esquema", "datos-documento-logico-v1")
	escribirCampoCanonico(&salida, "plantilla_id", plantilla.ID)
	escribirCampoCanonico(&salida, "plantilla_version", strconv.Itoa(plantilla.Version))
	for _, clave := range claves {
		escribirCampoCanonico(&salida, clave, datos[clave])
	}
	return []byte(salida.String())
}

func fuenteDocumentoLogicoCanonica(
	plantilla domain.PlantillaDocumento,
	huellaPlantilla, huellaDatos string,
	relaciones []domain.RelacionDocumento,
) []byte {
	var salida strings.Builder
	escribirCampoCanonico(&salida, "esquema", "fuente-documento-logico-v1")
	escribirCampoCanonico(&salida, "plantilla_id", plantilla.ID)
	escribirCampoCanonico(&salida, "plantilla_version", strconv.Itoa(plantilla.Version))
	escribirCampoCanonico(&salida, "huella_plantilla", huellaPlantilla)
	escribirCampoCanonico(&salida, "huella_datos", huellaDatos)
	escribirCampoCanonico(&salida, "modulo_id", plantilla.ModuloID)
	escribirCampoCanonico(&salida, "tipo_documental", plantilla.TipoDocumental)
	for indice, relacion := range relaciones {
		prefijo := "relacion_" + strconv.Itoa(indice) + "_"
		escribirCampoCanonico(&salida, prefijo+"tipo", string(relacion.Tipo))
		escribirCampoCanonico(&salida, prefijo+"rol", relacion.Rol)
		escribirCampoCanonico(&salida, prefijo+"referencia", relacion.Referencia)
	}
	return []byte(salida.String())
}

func solicitudDocumentoLogicoCanonica(
	orden OrdenGenerarDocumentoLogico,
	plantilla domain.PlantillaDocumento,
	huellaPlantilla string,
	relaciones []domain.RelacionDocumento,
	solicitudes []domain.SolicitudRepresentacionDocumento,
) []byte {
	var salida strings.Builder
	escribirCampoCanonico(&salida, "esquema", "solicitud-documento-logico-v1")
	escribirCampoCanonico(&salida, "plantilla_id", plantilla.ID)
	escribirCampoCanonico(&salida, "plantilla_version", strconv.Itoa(plantilla.Version))
	escribirCampoCanonico(&salida, "huella_plantilla", huellaPlantilla)
	escribirCampoCanonico(&salida, "datos_canonicos", string(datosDocumentoLogicoCanonicos(plantilla, orden.Datos)))
	escribirCampoCanonico(&salida, "principal_id", strings.TrimSpace(orden.Principal.ID))
	escribirCampoCanonico(&salida, "perfil_activo", strings.TrimSpace(orden.PerfilActivo))
	escribirCampoCanonico(&salida, "representado_ref", strings.TrimSpace(orden.RepresentadoRef))
	escribirCampoCanonico(&salida, "finalidad", strings.TrimSpace(orden.Finalidad))
	escribirCampoCanonico(&salida, "clasificacion", strings.TrimSpace(orden.Clasificacion))
	escribirCampoCanonico(&salida, "motivo", strings.TrimSpace(orden.Motivo))
	escribirCampoCanonico(&salida, "correlacion_ref", strings.TrimSpace(orden.CorrelacionRef))
	for indice, relacion := range relaciones {
		prefijo := "relacion_" + strconv.Itoa(indice) + "_"
		escribirCampoCanonico(&salida, prefijo+"tipo", string(relacion.Tipo))
		escribirCampoCanonico(&salida, prefijo+"rol", relacion.Rol)
		escribirCampoCanonico(&salida, prefijo+"referencia", relacion.Referencia)
	}
	for indice, solicitud := range solicitudes {
		prefijo := "representacion_" + strconv.Itoa(indice) + "_"
		escribirCampoCanonico(&salida, prefijo+"tipo", string(solicitud.Tipo))
		escribirCampoCanonico(&salida, prefijo+"formato", string(solicitud.Formato))
	}
	return []byte(salida.String())
}

func referenciaRepresentacionInicial(documentoID string, solicitud domain.SolicitudRepresentacionDocumento) string {
	return documentoID + ":representacion:" + string(solicitud.Tipo) + ":" + string(solicitud.Formato)
}

func nombreFicheroRepresentacion(tipoDocumental, documentoID string, solicitud domain.SolicitudRepresentacionDocumento) string {
	return nombreFicheroDocumento(tipoDocumental+"-"+string(solicitud.Tipo), documentoID, solicitud.Formato)
}

func serializarSolicitudesRepresentacion(solicitudes []domain.SolicitudRepresentacionDocumento) string {
	valores := make([]string, 0, len(solicitudes))
	for _, solicitud := range solicitudes {
		valores = append(valores, string(solicitud.Tipo)+":"+string(solicitud.Formato))
	}
	return strings.Join(valores, ",")
}
