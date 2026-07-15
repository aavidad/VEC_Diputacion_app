package application

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	eventoConsultaPublicaCotejo   = domain.AccionConsultaPublicaCotejo
	eventoConsultaProtegidaCotejo = domain.AccionConsultaProtegidaCotejo
)

type EstadoConsultaCotejo string

const (
	EstadoConsultaCotejoNoDisponible           EstadoConsultaCotejo = "no_disponible"
	EstadoConsultaCotejoDisponible             EstadoConsultaCotejo = "disponible"
	EstadoConsultaCotejoRequiereIdentificacion EstadoConsultaCotejo = "requiere_identificacion"
)

type OrdenConsultaPublicaCotejo struct {
	Secreto          ports.SecretoCodigoCotejo
	CorrelacionRef   string
	OrigenTecnicoRef string
}

// ResultadoConsultaPublicaCotejo contiene una lista cerrada de campos que no
// pueden identificar por si mismos a una persona ni revelar un expediente.
type ResultadoConsultaPublicaCotejo struct {
	Estado          EstadoConsultaCotejo `json:"estado"`
	Organo          string               `json:"organo,omitempty"`
	TipoDocumental  string               `json:"tipo_documental,omitempty"`
	FechaEmision    *time.Time           `json:"fecha_emision,omitempty"`
	HuellaSHA256    string               `json:"huella_sha256,omitempty"`
	PermiteDescarga bool                 `json:"permite_descarga"`
}

func (s *ServicioCotejo) ConsultarCotejoPublico(ctx context.Context, orden OrdenConsultaPublicaCotejo) (ResultadoConsultaPublicaCotejo, error) {
	if err := ctx.Err(); err != nil {
		return ResultadoConsultaPublicaCotejo{}, err
	}
	if orden.Secreto.Validar() != nil || strings.TrimSpace(orden.CorrelacionRef) == "" ||
		strings.TrimSpace(orden.OrigenTecnicoRef) == "" {
		return ResultadoConsultaPublicaCotejo{}, ErrOrdenCotejoInvalida
	}
	codigo, encontrado, err := s.buscarCodigoCotejo(ctx, orden.Secreto)
	if err != nil {
		if errors.Is(err, ports.ErrIndicesCodigoCotejoAmbiguos) {
			errAuditoria := s.registrarConsultaCotejo(ctx, datosRegistroConsultaCotejo{
				Accion: eventoConsultaPublicaCotejo, Resultado: "integridad_no_resuelta",
				CorrelacionRef: orden.CorrelacionRef, OrigenTecnicoRef: orden.OrigenTecnicoRef,
				Instante: s.reloj.Ahora().UTC(),
			})
			return ResultadoConsultaPublicaCotejo{Estado: EstadoConsultaCotejoNoDisponible},
				errors.Join(ErrCotejoNoDisponible, err, errAuditoria)
		}
		return ResultadoConsultaPublicaCotejo{}, err
	}
	ahora := s.reloj.Ahora().UTC()
	if !encontrado {
		if err := s.registrarConsultaCotejo(ctx, datosRegistroConsultaCotejo{
			Accion: eventoConsultaPublicaCotejo, Resultado: "no_encontrado",
			CorrelacionRef: orden.CorrelacionRef, OrigenTecnicoRef: orden.OrigenTecnicoRef, Instante: ahora,
		}); err != nil {
			return ResultadoConsultaPublicaCotejo{}, err
		}
		return ResultadoConsultaPublicaCotejo{Estado: EstadoConsultaCotejoNoDisponible}, nil
	}
	resultado := ResultadoConsultaPublicaCotejo{Estado: EstadoConsultaCotejoNoDisponible}
	resultadoAuditoria := "no_disponible"
	if codigo.DisponibleEn(ahora) && codigo.VersionEmitida != nil {
		switch codigo.Politica.ClaseAcceso {
		case domain.ClaseAccesoCotejoPublico:
			resultado = construirResultadoPublicoCotejo(codigo)
			resultadoAuditoria = "disponible"
		case domain.ClaseAccesoCotejoProtegido:
			resultado.Estado = EstadoConsultaCotejoRequiereIdentificacion
			resultadoAuditoria = "requiere_identificacion"
		case domain.ClaseAccesoCotejoInterno:
			resultadoAuditoria = "interno_oculto"
		}
	}
	if err := s.registrarConsultaCotejo(ctx, datosRegistroConsultaCotejo{
		Accion: eventoConsultaPublicaCotejo, Codigo: &codigo, Resultado: resultadoAuditoria,
		CorrelacionRef: orden.CorrelacionRef, OrigenTecnicoRef: orden.OrigenTecnicoRef, Instante: ahora,
	}); err != nil {
		return ResultadoConsultaPublicaCotejo{}, err
	}
	return resultado, nil
}

type OrdenConsultaProtegidaCotejo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	Secreto        ports.SecretoCodigoCotejo
	Motivo         string
	CorrelacionRef string
}

type ResultadoConsultaProtegidaCotejo struct {
	Estado             EstadoConsultaCotejo        `json:"estado,omitempty"`
	CodigoRef          string                      `json:"codigo_ref,omitempty"`
	Documento          *domain.ReferenciaDocumento `json:"documento,omitempty"`
	ModuloID           string                      `json:"modulo_id,omitempty"`
	TipoDocumental     string                      `json:"tipo_documental,omitempty"`
	Clasificacion      string                      `json:"clasificacion,omitempty"`
	Organo             string                      `json:"organo,omitempty"`
	ExpedienteRef      string                      `json:"expediente_ref,omitempty"`
	FechaEmision       *time.Time                  `json:"fecha_emision,omitempty"`
	HuellaSHA256       string                      `json:"huella_sha256,omitempty"`
	FirmaRefs          []string                    `json:"firma_refs,omitempty"`
	SelloTiempoRefs    []string                    `json:"sello_tiempo_refs,omitempty"`
	ValidacionFirmaRef string                      `json:"validacion_firma_ref,omitempty"`
	RegistroRef        string                      `json:"registro_ref,omitempty"`
	// PermiteDescarga es un puntero para distinguir una denegacion expresa de
	// la ausencia de permiso para revelar siquiera esta capacidad. Solo se
	// proyecta cuando la decision concede el campo permite_descarga.
	PermiteDescarga *bool `json:"permite_descarga,omitempty"`
}

func (s *ServicioCotejo) ConsultarCotejoProtegido(ctx context.Context, orden OrdenConsultaProtegidaCotejo) (ResultadoConsultaProtegidaCotejo, error) {
	if err := ctx.Err(); err != nil {
		return ResultadoConsultaProtegidaCotejo{}, err
	}
	if err := validarContextoCotejo(orden.Principal, orden.PerfilActivo, orden.Finalidad,
		orden.Motivo, orden.CorrelacionRef, domain.AuthAssuranceLow); err != nil {
		return ResultadoConsultaProtegidaCotejo{}, err
	}
	if orden.Secreto.Validar() != nil {
		return ResultadoConsultaProtegidaCotejo{}, ErrOrdenCotejoInvalida
	}
	codigo, encontrado, err := s.buscarCodigoCotejo(ctx, orden.Secreto)
	if err != nil {
		return ResultadoConsultaProtegidaCotejo{}, err
	}
	ahora := s.reloj.Ahora().UTC()
	if !encontrado || !codigo.DisponibleEn(ahora) || codigo.VersionEmitida == nil {
		var codigoAuditoria *domain.CodigoCotejo
		if encontrado {
			codigoAuditoria = &codigo
		}
		if err := s.registrarConsultaCotejo(ctx, datosRegistroConsultaCotejo{
			Principal: &orden.Principal, PerfilActivo: orden.PerfilActivo, Finalidad: orden.Finalidad,
			Accion: eventoConsultaProtegidaCotejo, Codigo: codigoAuditoria, Resultado: "no_disponible",
			CorrelacionRef: orden.CorrelacionRef, Instante: ahora,
		}); err != nil {
			return ResultadoConsultaProtegidaCotejo{}, err
		}
		return ResultadoConsultaProtegidaCotejo{}, ErrCotejoNoDisponible
	}
	if !orden.Principal.AuthAssurance.Cumple(codigo.Politica.GarantiaMinima) {
		if err := s.registrarConsultaCotejo(ctx, datosRegistroConsultaCotejo{
			Principal: &orden.Principal, PerfilActivo: orden.PerfilActivo, Finalidad: orden.Finalidad,
			Accion: eventoConsultaProtegidaCotejo, Codigo: &codigo, Resultado: "garantia_insuficiente",
			CorrelacionRef: orden.CorrelacionRef, Instante: ahora,
		}); err != nil {
			return ResultadoConsultaProtegidaCotejo{}, errors.Join(domain.ErrGarantiaInsuficiente, err)
		}
		return ResultadoConsultaProtegidaCotejo{}, domain.ErrGarantiaInsuficiente
	}
	documento, expedienteRef, err := s.documentoVinculadoCodigoCotejo(ctx, codigo)
	if err != nil {
		return ResultadoConsultaProtegidaCotejo{}, err
	}
	sujetoActivoRef := sujetoActivoCotejo(orden.Principal)
	if codigo.Politica.RequiereTitularidad && !esTitularDocumentoCotejo(documento.Relaciones, sujetoActivoRef, codigo.Politica.RolesTitularidad) {
		if err := s.registrarConsultaCotejo(ctx, datosRegistroConsultaCotejo{
			Principal: &orden.Principal, PerfilActivo: orden.PerfilActivo, Finalidad: orden.Finalidad,
			Accion: eventoConsultaProtegidaCotejo, Codigo: &codigo, Resultado: "titularidad_no_acreditada",
			CorrelacionRef: orden.CorrelacionRef, RepresentadoRef: sujetoActivoRef, Instante: ahora,
		}); err != nil {
			return ResultadoConsultaProtegidaCotejo{}, errors.Join(domain.ErrAutorizacionDenegada, err)
		}
		return ResultadoConsultaProtegidaCotejo{}, domain.ErrAutorizacionDenegada
	}
	accion := AccionConsultaProtegidaCotejo
	if codigo.Politica.ClaseAcceso == domain.ClaseAccesoCotejoInterno {
		accion = AccionRevisionInternaCotejo
	}
	recurso := recursoCodigoCotejo(codigo, expedienteRef, nil)
	if sujetoActivoRef != "" {
		recurso.Ambitos["persona"] = sujetoActivoRef
	}
	decision, err := exigirDecisionAutorizacion(ctx, s.autorizador, s.reloj,
		orden.Principal, orden.PerfilActivo, accion, recurso, orden.Finalidad, orden.CorrelacionRef, orden.Motivo,
		usoCamposDecisionConsumidos)
	if err != nil {
		if errAuditoria := s.registrarConsultaCotejo(ctx, datosRegistroConsultaCotejo{
			Principal: &orden.Principal, PerfilActivo: orden.PerfilActivo, Finalidad: orden.Finalidad,
			Accion: eventoConsultaProtegidaCotejo, Codigo: &codigo, Resultado: "autorizacion_denegada",
			CorrelacionRef: orden.CorrelacionRef, RepresentadoRef: sujetoActivoRef, Instante: ahora,
		}); errAuditoria != nil {
			return ResultadoConsultaProtegidaCotejo{}, errors.Join(err, errAuditoria)
		}
		return ResultadoConsultaProtegidaCotejo{}, err
	}
	if !camposDecisionConsultaCotejoValidos(decision.CamposPermitidos) {
		errDecision := errors.Join(domain.ErrAutorizacionDenegada, domain.ErrDecisionAutorizacionInvalida)
		if errAuditoria := s.registrarConsultaCotejo(ctx, datosRegistroConsultaCotejo{
			Principal: &orden.Principal, PerfilActivo: orden.PerfilActivo, Finalidad: orden.Finalidad,
			Accion: eventoConsultaProtegidaCotejo, Codigo: &codigo, Resultado: "proyeccion_campos_invalida",
			CorrelacionRef: orden.CorrelacionRef, AutorizacionRef: decision.DecisionRef,
			RepresentadoRef: sujetoActivoRef, Instante: s.reloj.Ahora().UTC(),
		}); errAuditoria != nil {
			return ResultadoConsultaProtegidaCotejo{}, errors.Join(errDecision, errAuditoria)
		}
		return ResultadoConsultaProtegidaCotejo{}, errDecision
	}
	resultado := construirResultadoProtegidoCotejo(codigo, decision)
	confirmadaEn := s.reloj.Ahora().UTC()
	if err := s.registrarConsultaCotejo(ctx, datosRegistroConsultaCotejo{
		Principal: &orden.Principal, PerfilActivo: orden.PerfilActivo, Finalidad: orden.Finalidad,
		Accion: eventoConsultaProtegidaCotejo, Codigo: &codigo, Resultado: "disponible",
		CorrelacionRef: orden.CorrelacionRef, AutorizacionRef: decision.DecisionRef,
		RepresentadoRef: sujetoActivoRef, Instante: confirmadaEn,
	}); err != nil {
		return ResultadoConsultaProtegidaCotejo{}, err
	}
	return resultado, nil
}

func (s *ServicioCotejo) buscarCodigoCotejo(ctx context.Context, secreto ports.SecretoCodigoCotejo) (domain.CodigoCotejo, bool, error) {
	indices, err := s.selladorIndice.SellarIndicesConsultaCodigoCotejo(ctx, secreto)
	if err != nil {
		return domain.CodigoCotejo{}, false, err
	}
	indices, err = canonizarIndicesConsultaCotejo(indices)
	if err != nil {
		return domain.CodigoCotejo{}, false, err
	}
	codigo, err := s.codigos.BuscarCodigoCotejoPorIndices(ctx, indices)
	if errors.Is(err, ports.ErrCodigoCotejoNoEncontrado) {
		return domain.CodigoCotejo{}, false, nil
	}
	if err != nil {
		return domain.CodigoCotejo{}, false, err
	}
	canonico, err := codigo.ClonarCanonico()
	if err != nil {
		return domain.CodigoCotejo{}, false, errors.Join(ErrResultadoCotejoInvalido, err)
	}
	if !contieneHuellaCotejoConstante(indices, canonico.IndiceCodigoHMAC) {
		return domain.CodigoCotejo{}, false, ErrResultadoCotejoInvalido
	}
	return canonico, true, nil
}

func canonizarIndicesConsultaCotejo(indices []string) ([]string, error) {
	if len(indices) == 0 || len(indices) > 16 {
		return nil, ErrResultadoCotejoInvalido
	}
	canonicos := append([]string(nil), indices...)
	sort.Strings(canonicos)
	for indice, valor := range canonicos {
		if !huellaHMACCotejoValida(valor) || (indice > 0 && valor == canonicos[indice-1]) {
			return nil, ErrResultadoCotejoInvalido
		}
	}
	return canonicos, nil
}

func construirResultadoPublicoCotejo(codigo domain.CodigoCotejo) ResultadoConsultaPublicaCotejo {
	resultado := ResultadoConsultaPublicaCotejo{
		Estado:          EstadoConsultaCotejoDisponible,
		PermiteDescarga: codigo.Politica.PermiteDescargaDocumento && !codigo.Politica.RequiereTitularidad,
	}
	version := codigo.VersionEmitida
	if version == nil {
		return ResultadoConsultaPublicaCotejo{Estado: EstadoConsultaCotejoNoDisponible}
	}
	if codigo.TieneCampoPublico(domain.CampoPublicoCotejoOrgano) {
		resultado.Organo = codigo.Organo
	}
	if codigo.TieneCampoPublico(domain.CampoPublicoCotejoTipoDocumental) {
		resultado.TipoDocumental = codigo.TipoDocumental
	}
	if codigo.TieneCampoPublico(domain.CampoPublicoCotejoFechaEmision) {
		fecha := version.EmitidaEn
		resultado.FechaEmision = &fecha
	}
	if codigo.TieneCampoPublico(domain.CampoPublicoCotejoHuellaSHA256) {
		resultado.HuellaSHA256 = version.HuellaContenidoSHA256
	}
	return resultado
}

func construirResultadoProtegidoCotejo(codigo domain.CodigoCotejo, decision domain.DecisionAutorizacion) ResultadoConsultaProtegidaCotejo {
	resultado := ResultadoConsultaProtegidaCotejo{
		Estado:    EstadoConsultaCotejoDisponible,
		CodigoRef: codigo.Referencia(),
	}
	version := codigo.VersionEmitida
	if version == nil {
		return ResultadoConsultaProtegidaCotejo{Estado: EstadoConsultaCotejoNoDisponible}
	}
	permite := func(campo string) bool {
		return contieneCampoDecisionCotejo(decision.CamposPermitidos, campo)
	}
	if permite(campoConsultaCotejoDocumentoRef) {
		referencia := codigo.Documento
		resultado.Documento = &referencia
	}
	if permite(campoConsultaCotejoModuloID) {
		resultado.ModuloID = codigo.ModuloID
	}
	if permite(campoConsultaCotejoTipoDocumental) {
		resultado.TipoDocumental = codigo.TipoDocumental
	}
	if permite(campoConsultaCotejoClasificacion) {
		resultado.Clasificacion = codigo.Clasificacion
	}
	if permite(campoConsultaCotejoOrgano) {
		resultado.Organo = codigo.Organo
	}
	if permite(campoConsultaCotejoExpedienteRef) {
		resultado.ExpedienteRef = codigo.ExpedienteRef
	}
	if permite(campoConsultaCotejoFechaEmision) {
		fecha := version.EmitidaEn
		resultado.FechaEmision = &fecha
	}
	if permite(campoConsultaCotejoHuellaSHA256) {
		resultado.HuellaSHA256 = version.HuellaContenidoSHA256
	}
	if permite(campoConsultaCotejoFirmaRefs) {
		resultado.FirmaRefs = append([]string(nil), version.FirmaRefs...)
	}
	if permite(campoConsultaCotejoSelloTiempoRefs) {
		resultado.SelloTiempoRefs = append([]string(nil), version.SelloTiempoRefs...)
	}
	if permite(campoConsultaCotejoValidacionFirmaRef) {
		resultado.ValidacionFirmaRef = version.ValidacionFirmaRef
	}
	if permite(campoConsultaCotejoRegistroRef) {
		resultado.RegistroRef = version.RegistroRef
	}
	if permite(campoConsultaCotejoPermiteDescarga) {
		permiteDescarga := codigo.Politica.PermiteDescargaDocumento && permite(campoConsultaCotejoDescarga)
		resultado.PermiteDescarga = &permiteDescarga
	}
	return resultado
}

const (
	campoConsultaCotejoEstado             = "estado"
	campoConsultaCotejoCodigoRef          = "codigo_ref"
	campoConsultaCotejoDocumentoRef       = "documento_ref"
	campoConsultaCotejoModuloID           = "modulo_id"
	campoConsultaCotejoTipoDocumental     = "tipo_documental"
	campoConsultaCotejoClasificacion      = "clasificacion"
	campoConsultaCotejoOrgano             = "organo"
	campoConsultaCotejoExpedienteRef      = "expediente_ref"
	campoConsultaCotejoFechaEmision       = "fecha_emision"
	campoConsultaCotejoHuellaSHA256       = "huella_sha256"
	campoConsultaCotejoFirmaRefs          = "firma_refs"
	campoConsultaCotejoSelloTiempoRefs    = "sello_tiempo_refs"
	campoConsultaCotejoValidacionFirmaRef = "validacion_firma_ref"
	campoConsultaCotejoRegistroRef        = "registro_ref"
	campoConsultaCotejoPermiteDescarga    = "permite_descarga"
	// descarga es la capacidad positiva de obtener los bytes. No se infiere de
	// permite_descarga: el indicador solo puede ser verdadero si ambos campos
	// exactos y la politica documental la conceden.
	campoConsultaCotejoDescarga = "descarga"
)

func contieneCampoDecisionCotejo(campos []string, buscado string) bool {
	for _, campo := range campos {
		if campo == buscado {
			return true
		}
	}
	return false
}

func camposDecisionConsultaCotejoValidos(campos []string) bool {
	if len(campos) == 0 {
		return false
	}
	tieneEstado := false
	tieneCodigoRef := false
	for _, campo := range campos {
		switch campo {
		case campoConsultaCotejoEstado:
			tieneEstado = true
		case campoConsultaCotejoCodigoRef:
			tieneCodigoRef = true
		case campoConsultaCotejoDocumentoRef, campoConsultaCotejoModuloID,
			campoConsultaCotejoTipoDocumental, campoConsultaCotejoClasificacion,
			campoConsultaCotejoOrgano, campoConsultaCotejoExpedienteRef,
			campoConsultaCotejoFechaEmision, campoConsultaCotejoHuellaSHA256,
			campoConsultaCotejoFirmaRefs, campoConsultaCotejoSelloTiempoRefs,
			campoConsultaCotejoValidacionFirmaRef, campoConsultaCotejoRegistroRef,
			campoConsultaCotejoPermiteDescarga, campoConsultaCotejoDescarga:
			// Campo consumido de forma expresa por construirResultadoProtegidoCotejo.
		default:
			return false
		}
	}
	// Una respuesta disponible siempre revela estos dos campos. Por tanto son
	// parte obligatoria de la concesion, nunca valores implicitos del caso de uso.
	return tieneEstado && tieneCodigoRef
}

func sujetoActivoCotejo(principal domain.Principal) string {
	if referencia := strings.TrimSpace(principal.Attributes["sujeto_activo_ref"]); referencia != "" {
		return referencia
	}
	return strings.TrimSpace(principal.Attributes["persona_ref"])
}

func esTitularDocumentoCotejo(relaciones []domain.RelacionDocumento, sujetoRef string, roles []string) bool {
	if sujetoRef == "" || len(roles) == 0 {
		return false
	}
	rolesPermitidos := make(map[string]struct{}, len(roles))
	for _, rol := range roles {
		rolesPermitidos[rol] = struct{}{}
	}
	for _, relacion := range relaciones {
		if relacion.Tipo != domain.TipoRelacionPersona || relacion.Referencia != sujetoRef {
			continue
		}
		if _, permitido := rolesPermitidos[relacion.Rol]; permitido {
			return true
		}
	}
	return false
}

type datosRegistroConsultaCotejo struct {
	Principal        *domain.Principal
	PerfilActivo     string
	Finalidad        string
	Accion           string
	Codigo           *domain.CodigoCotejo
	Resultado        string
	CorrelacionRef   string
	OrigenTecnicoRef string
	AutorizacionRef  string
	RepresentadoRef  string
	Instante         time.Time
}

func (s *ServicioCotejo) registrarConsultaCotejo(ctx context.Context, datos datosRegistroConsultaCotejo) error {
	actorID := "publico-anonimo"
	perfilActivo := strings.TrimSpace(datos.PerfilActivo)
	finalidad := strings.TrimSpace(datos.Finalidad)
	if datos.Principal == nil {
		perfilActivo = "publico"
		finalidad = "verificacion_documental_publica"
	}
	moduloID := moduloNucleoDocumental
	sujetoRef := "cotejo:consulta-no-resuelta"
	version := 0
	documentoRef := ""
	expedienteRef := ""
	metadatos := map[string]string{"resultado_consulta": datos.Resultado}
	if strings.TrimSpace(datos.OrigenTecnicoRef) != "" {
		metadatos["origen_tecnico_ref"] = strings.TrimSpace(datos.OrigenTecnicoRef)
	}
	if datos.Codigo != nil {
		moduloID = datos.Codigo.ModuloID
		sujetoRef = datos.Codigo.Referencia()
		version = datos.Codigo.Revision
		// La traza publica se vincula al identificador interno del cotejo, pero
		// no replica referencias de documento o expediente. Un revisor con
		// acceso interno puede resolverlas desde el agregado protegido.
		if datos.Principal != nil {
			documentoRef = referenciaDocumentoCotejo(datos.Codigo.Documento)
			expedienteRef = datos.Codigo.ExpedienteRef
		}
		metadatos["estado_codigo"] = string(datos.Codigo.Estado)
		metadatos["clase_acceso"] = string(datos.Codigo.Politica.ClaseAcceso)
	}
	traza := domain.AuditEntry{
		ActorID:              actorID,
		ActorProfile:         perfilActivo,
		RepresentedSubjectID: strings.TrimSpace(datos.RepresentadoRef),
		AuthorizationRef:     strings.TrimSpace(datos.AutorizacionRef),
		Purpose:              finalidad,
		Action:               datos.Accion,
		ModuleID:             moduloID,
		SubjectRef:           sujetoRef,
		ObjectVersion:        version,
		ExpedienteRef:        expedienteRef,
		DocumentRef:          documentoRef,
		Reason:               "consulta de cotejo documental",
		Result:               datos.Resultado,
		CorrelationRef:       strings.TrimSpace(datos.CorrelacionRef),
		Metadata:             metadatos,
		OccurredAt:           datos.Instante.UTC(),
	}
	if datos.Principal != nil {
		actorID = strings.TrimSpace(datos.Principal.ID)
		traza.ActorID = actorID
		traza.ActorRoles = append([]string(nil), datos.Principal.Roles...)
		traza.AuthMethod = datos.Principal.AuthMethod
		traza.AuthAssurance = datos.Principal.AuthAssurance
	}
	evento := domain.Event{
		Type:       datos.Accion,
		ModuleID:   moduloID,
		SubjectRef: sujetoRef,
		ActorID:    actorID,
		OccurredAt: datos.Instante.UTC(),
		Payload: map[string]string{
			"resultado_consulta": datos.Resultado,
			"autenticada":        strconv.FormatBool(datos.Principal != nil),
		},
	}
	return s.codigos.RegistrarConsultaCotejo(ctx, traza, evento)
}
