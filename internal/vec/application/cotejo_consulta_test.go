package application

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	consultaCotejoPruebaSecreto         = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	consultaCotejoPruebaIndiceHistorico = "hmac-sha256:indice-cotejo-v1:1111111111111111111111111111111111111111111111111111111111111111"
	consultaCotejoPruebaIndiceActual    = "hmac-sha256:indice-cotejo-v2:2222222222222222222222222222222222222222222222222222222222222222"
	consultaCotejoPruebaProteccion      = "custodia-cotejo-consulta-001"
	consultaCotejoPruebaExpediente      = "expediente-cotejo-consulta-001"
	consultaCotejoPruebaPersona         = "persona-cotejo-consulta-001"
)

type consultaCotejoPruebaReloj struct {
	ahora time.Time
}

func (r *consultaCotejoPruebaReloj) Ahora() time.Time { return r.ahora }

type consultaCotejoPruebaRegistro struct {
	auditoria domain.AuditEntry
	evento    domain.Event
}

type consultaCotejoPruebaCodigos struct {
	ports.RepositorioCodigosCotejo
	codigo             domain.CodigoCotejo
	errorBusqueda      error
	errorAuditoria     error
	indicesConsultados [][]string
	registros          []consultaCotejoPruebaRegistro
}

func (r *consultaCotejoPruebaCodigos) BuscarCodigoCotejoPorIndices(
	_ context.Context,
	indices []string,
) (domain.CodigoCotejo, error) {
	r.indicesConsultados = append(r.indicesConsultados, append([]string(nil), indices...))
	if r.errorBusqueda != nil {
		return domain.CodigoCotejo{}, r.errorBusqueda
	}
	return r.codigo, nil
}

func (r *consultaCotejoPruebaCodigos) RegistrarConsultaCotejo(
	_ context.Context,
	auditoria domain.AuditEntry,
	evento domain.Event,
) error {
	if r.errorAuditoria != nil {
		return r.errorAuditoria
	}
	r.registros = append(r.registros, consultaCotejoPruebaRegistro{auditoria: auditoria, evento: evento})
	return nil
}

type consultaCotejoPruebaDocumentos struct {
	ports.RepositorioDocumentosLogicos
	documento domain.DocumentoLogico
	error     error
	consultas []domain.ReferenciaDocumento
}

func (r *consultaCotejoPruebaDocumentos) ObtenerDocumentoLogico(
	_ context.Context,
	referencia domain.ReferenciaDocumento,
) (domain.DocumentoLogico, error) {
	r.consultas = append(r.consultas, referencia)
	if r.error != nil {
		return domain.DocumentoLogico{}, r.error
	}
	return r.documento, nil
}

type consultaCotejoPruebaSelladorIndice struct {
	ports.SelladorIndiceCodigoCotejo
	indices   []string
	error     error
	consultas int
	secreto   ports.SecretoCodigoCotejo
}

func (s *consultaCotejoPruebaSelladorIndice) SellarIndicesConsultaCodigoCotejo(
	_ context.Context,
	secreto ports.SecretoCodigoCotejo,
) ([]string, error) {
	s.consultas++
	s.secreto = secreto
	if s.error != nil {
		return nil, s.error
	}
	return append([]string(nil), s.indices...), nil
}

type consultaCotejoPruebaAutorizador struct {
	reloj       *consultaCotejoPruebaReloj
	campos      []string
	error       error
	solicitudes []domain.SolicitudAutorizacion
}

func (a *consultaCotejoPruebaAutorizador) Exigir(
	_ context.Context,
	solicitud domain.SolicitudAutorizacion,
) (domain.DecisionAutorizacion, error) {
	a.solicitudes = append(a.solicitudes, solicitud)
	if a.error != nil {
		return domain.DecisionAutorizacion{}, a.error
	}
	ahora := a.reloj.Ahora().UTC()
	return completarDecisionAutorizacionPrueba(solicitud, domain.DecisionAutorizacion{
		DecisionRef:            "decision-cotejo-consulta-001",
		Concedida:              true,
		Codigo:                 "concedida",
		PrincipalID:            solicitud.Principal.ID,
		PerfilActivoRef:        solicitud.PerfilActivoRef,
		Accion:                 solicitud.Accion,
		RecursoRef:             solicitud.Recurso.Referencia,
		Finalidad:              solicitud.Finalidad,
		CorrelacionRef:         solicitud.CorrelacionRef,
		AsignacionRef:          "asignacion:cotejo-consulta:v1",
		AsignacionHuellaSHA256: strings.Repeat("a", 64),
		VersionRolRef:          "rol:cotejo-consulta:v1",
		VersionRolHuellaSHA256: strings.Repeat("b", 64),
		GarantiaMinima:         domain.AuthAssuranceLow,
		CamposPermitidos:       append([]string(nil), a.campos...),
		EmitidaEn:              ahora.Add(-time.Minute),
		ValidaHasta:            ahora.Add(time.Minute),
	}), nil
}

type consultaCotejoPruebaPuertosNoUsados struct {
	ports.CatalogoPoliticasCotejo
	ports.RepositorioGobiernoPoliticasCotejo
	ports.GeneradorValorCodigoCotejo
	ports.GeneradorIDCodigoCotejo
	ports.SelladorSolicitudCotejo
	ports.ProtectorCodigoCotejo
	ports.FuenteEvidenciaEmisionDocumento
}

type consultaCotejoPruebaEntorno struct {
	servicio    *ServicioCotejo
	reloj       *consultaCotejoPruebaReloj
	codigos     *consultaCotejoPruebaCodigos
	documentos  *consultaCotejoPruebaDocumentos
	autorizador *consultaCotejoPruebaAutorizador
	sellador    *consultaCotejoPruebaSelladorIndice
	secreto     ports.SecretoCodigoCotejo
}

func consultaCotejoPruebaNuevoEntorno(t *testing.T) *consultaCotejoPruebaEntorno {
	t.Helper()
	ahora := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	reloj := &consultaCotejoPruebaReloj{ahora: ahora}
	secreto, err := ports.NuevoSecretoCodigoCotejo(consultaCotejoPruebaSecreto)
	if err != nil {
		t.Fatalf("NuevoSecretoCodigoCotejo() error = %v", err)
	}
	documento := consultaCotejoPruebaDocumento(ahora)
	codigo := consultaCotejoPruebaCodigoActivo(ahora)
	if err := documento.Validar(); err != nil {
		t.Fatalf("DocumentoLogico.Validar() fixture error = %v", err)
	}
	if err := codigo.Validar(); err != nil {
		t.Fatalf("CodigoCotejo.Validar() fixture error = %v", err)
	}
	codigos := &consultaCotejoPruebaCodigos{codigo: codigo}
	documentos := &consultaCotejoPruebaDocumentos{documento: documento}
	autorizador := &consultaCotejoPruebaAutorizador{reloj: reloj}
	sellador := &consultaCotejoPruebaSelladorIndice{
		indices: []string{consultaCotejoPruebaIndiceActual, consultaCotejoPruebaIndiceHistorico},
	}
	noUsados := &consultaCotejoPruebaPuertosNoUsados{}
	servicio, err := NuevoServicioCotejo(
		noUsados,
		noUsados,
		codigos,
		documentos,
		autorizador,
		noUsados,
		noUsados,
		sellador,
		noUsados,
		noUsados,
		noUsados,
		reloj,
	)
	if err != nil {
		t.Fatalf("NuevoServicioCotejo() error = %v", err)
	}
	return &consultaCotejoPruebaEntorno{
		servicio: servicio, reloj: reloj, codigos: codigos, documentos: documentos,
		autorizador: autorizador, sellador: sellador, secreto: secreto,
	}
}

func consultaCotejoPruebaDocumento(ahora time.Time) domain.DocumentoLogico {
	return domain.DocumentoLogico{
		ID:       "documento-cotejo-consulta-001",
		Version:  1,
		Revision: 1,
		Plantilla: domain.ReferenciaPlantillaDocumento{
			ID:           "certificado_bolsa_consulta",
			Version:      2,
			HuellaSHA256: strings.Repeat("c", 64),
		},
		ModuloID:       "bolsa",
		TipoDocumental: "certificado",
		Clasificacion:  "datos_personales_alta",
		Relaciones: []domain.RelacionDocumento{
			{Tipo: domain.TipoRelacionPersona, Referencia: consultaCotejoPruebaPersona, Rol: "interesada"},
			{Tipo: domain.TipoRelacionExpediente, Referencia: consultaCotejoPruebaExpediente, Rol: "principal"},
		},
		Estado:           domain.EstadoDocumentoLogicoRegistrado,
		HuellaDatosHMAC:  "hmac-sha256:datos-cotejo-consulta-v1:" + strings.Repeat("d", 64),
		HuellaFuenteHMAC: "hmac-sha256:fuente-cotejo-consulta-v1:" + strings.Repeat("e", 64),
		CreadoPor:        "tecnico-rrhh-consulta-1",
		CreadoEn:         ahora.Add(-4 * time.Hour),
		CorrelacionRef:   "correlacion-documento-consulta-001",
		Motivo:           "Emision de certificado para consulta de cotejo",
		ENI: domain.MetadatosENI{
			Identificador:     "ES_L01000000_2026_DOC_CONSULTA_001",
			Organo:            "L01000000",
			Origen:            "administracion",
			EstadoElaboracion: "original",
			TipoDocumental:    "certificado",
			FechaCaptura:      ahora.Add(-4 * time.Hour),
		},
	}
}

func consultaCotejoPruebaCodigoActivo(ahora time.Time) domain.CodigoCotejo {
	version := domain.VersionEmitidaCotejo{
		RepresentacionID:      "representacion-cotejo-consulta-001",
		ReferenciaContenido:   "almacen:cotejo-consulta-001",
		HuellaContenidoSHA256: strings.Repeat("f", 64),
		MIME:                  "application/pdf",
		Tamano:                8_192,
		FirmaRefs:             []string{"firma-cotejo-consulta-001"},
		SelloTiempoRefs:       []string{"sello-tiempo-cotejo-consulta-001"},
		ValidacionFirmaRef:    "validacion-firma-cotejo-consulta-001",
		RegistroRef:           "registro-cotejo-consulta-001",
		EmitidaEn:             ahora.Add(-2 * time.Hour),
	}
	return domain.CodigoCotejo{
		ID:               "codigo-cotejo-consulta-001",
		Revision:         2,
		Documento:        domain.ReferenciaDocumento{ID: "documento-cotejo-consulta-001", Version: 1},
		ModuloID:         "bolsa",
		TipoDocumental:   "certificado",
		Clasificacion:    "datos_personales_alta",
		Organo:           "L01000000",
		ExpedienteRef:    consultaCotejoPruebaExpediente,
		IndiceCodigoHMAC: consultaCotejoPruebaIndiceHistorico,
		ProteccionRef:    consultaCotejoPruebaProteccion,
		VersionGenerador: "generador-cotejo-consulta-v1",
		EntropiaBits:     160,
		Politica: domain.AplicacionPoliticaCotejo{
			Referencia: domain.ReferenciaPoliticaCotejo{
				ID:           "politica_cotejo_consulta",
				Version:      3,
				HuellaSHA256: strings.Repeat("9", 64),
			},
			ClaseAcceso:              domain.ClaseAccesoCotejoPublico,
			CamposPublicos:           []domain.CampoPublicoCotejo{domain.CampoPublicoCotejoFechaEmision, domain.CampoPublicoCotejoTipoDocumental},
			PermiteDescargaDocumento: true,
			GarantiaMinima:           domain.AuthAssuranceLow,
			DiasPlazoActivacion:      7,
			DiasDisponibilidad:       30,
		},
		Estado:              domain.EstadoCodigoCotejoActivo,
		ReservadoPor:        "tecnico-rrhh-consulta-1",
		ReservadoEn:         ahora.Add(-3 * time.Hour),
		ReservaExpiraEn:     ahora.Add(-90 * time.Minute),
		MotivoReserva:       "Reserva para emitir certificado con cotejo",
		CorrelacionRef:      "correlacion-cotejo-consulta-001",
		VersionEmitida:      &version,
		ActivadoPor:         "tecnico-registro-consulta-1",
		ActivadoEn:          ahora.Add(-time.Hour),
		ActivacionRef:       "activacion-cotejo-consulta-001",
		EvidenciaEmisionRef: "evidencia-emision-cotejo-consulta-001",
		MotivoActivacion:    "Activacion tras validar la emision documental",
		DisponibleDesde:     ahora.Add(-time.Hour),
		DisponibleHasta:     ahora.Add(24 * time.Hour),
	}
}

func consultaCotejoPruebaOrdenPublica(entorno *consultaCotejoPruebaEntorno) OrdenConsultaPublicaCotejo {
	return OrdenConsultaPublicaCotejo{
		Secreto:          entorno.secreto,
		CorrelacionRef:   "correlacion-consulta-publica-001",
		OrigenTecnicoRef: "http-publico-consulta-001",
	}
}

func consultaCotejoPruebaPrincipal(sujetoRef string) domain.Principal {
	return domain.Principal{
		ID:            personaAutorizacionPrueba("persona-autenticada-consulta-001"),
		Roles:         []string{"persona_interesada"},
		AuthMethod:    domain.AuthMethodCertificate,
		AuthAssurance: domain.AuthAssuranceHigh,
		Attributes: map[string]string{
			"sujeto_activo_ref": sujetoRef,
			"persona_ref":       "persona-ref-alternativa-no-usada",
		},
	}
}

func consultaCotejoPruebaOrdenProtegida(entorno *consultaCotejoPruebaEntorno, sujetoRef string) OrdenConsultaProtegidaCotejo {
	return OrdenConsultaProtegidaCotejo{
		Principal:      consultaCotejoPruebaPrincipal(sujetoRef),
		PerfilActivo:   perfilAutorizacionPrueba("persona-consulta:v1"),
		Finalidad:      "consultar_documento_propio",
		Secreto:        entorno.secreto,
		Motivo:         "Verificacion del certificado propio",
		CorrelacionRef: "correlacion-consulta-protegida-001",
	}
}

func consultaCotejoPruebaExigirTitularidad(entorno *consultaCotejoPruebaEntorno, roles ...string) {
	entorno.codigos.codigo.Politica.ClaseAcceso = domain.ClaseAccesoCotejoProtegido
	entorno.codigos.codigo.Politica.RequiereTitularidad = true
	entorno.codigos.codigo.Politica.RolesTitularidad = append([]string(nil), roles...)
}

func TestConsultarCotejoPublicoMuestraSoloCamposPermitidosYUsaIndiceHistorico(t *testing.T) {
	entorno := consultaCotejoPruebaNuevoEntorno(t)

	resultado, err := entorno.servicio.ConsultarCotejoPublico(context.Background(), consultaCotejoPruebaOrdenPublica(entorno))
	if err != nil {
		t.Fatalf("ConsultarCotejoPublico() error = %v", err)
	}
	if resultado.Estado != EstadoConsultaCotejoDisponible || resultado.TipoDocumental != entorno.codigos.codigo.TipoDocumental ||
		resultado.FechaEmision == nil || !resultado.FechaEmision.Equal(entorno.codigos.codigo.VersionEmitida.EmitidaEn) ||
		resultado.Organo != "" || resultado.HuellaSHA256 != "" || !resultado.PermiteDescarga {
		t.Fatalf("resultado publico fuera de allowlist: %+v", resultado)
	}
	contenido, err := json.Marshal(resultado)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var campos map[string]any
	if err := json.Unmarshal(contenido, &campos); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	esperados := []string{"estado", "tipo_documental", "fecha_emision", "permite_descarga"}
	if len(campos) != len(esperados) {
		t.Fatalf("campos JSON = %v", campos)
	}
	for _, esperado := range esperados {
		if _, existe := campos[esperado]; !existe {
			t.Fatalf("falta campo publico %q en %v", esperado, campos)
		}
	}
	if entorno.sellador.consultas != 1 || entorno.sellador.secreto.Revelar() != consultaCotejoPruebaSecreto ||
		len(entorno.codigos.indicesConsultados) != 1 || !reflect.DeepEqual(entorno.codigos.indicesConsultados[0], []string{
		consultaCotejoPruebaIndiceHistorico,
		consultaCotejoPruebaIndiceActual,
	}) {
		t.Fatalf("busqueda multiclave incorrecta: %+v", entorno.codigos.indicesConsultados)
	}
}

func TestConsultaCotejoPublicaOcultaMaterialSensibleEnJSONAuditoriaYEvento(t *testing.T) {
	entorno := consultaCotejoPruebaNuevoEntorno(t)
	resultado, err := entorno.servicio.ConsultarCotejoPublico(context.Background(), consultaCotejoPruebaOrdenPublica(entorno))
	if err != nil {
		t.Fatalf("ConsultarCotejoPublico() error = %v", err)
	}
	if len(entorno.codigos.registros) != 1 {
		t.Fatalf("registros de consulta = %d", len(entorno.codigos.registros))
	}
	evidencia, err := json.Marshal(struct {
		Resultado ResultadoConsultaPublicaCotejo `json:"resultado"`
		Auditoria domain.AuditEntry              `json:"auditoria"`
		Evento    domain.Event                   `json:"evento"`
	}{
		Resultado: resultado,
		Auditoria: entorno.codigos.registros[0].auditoria,
		Evento:    entorno.codigos.registros[0].evento,
	})
	if err != nil {
		t.Fatalf("json.Marshal() evidencia error = %v", err)
	}
	for _, prohibido := range []string{
		consultaCotejoPruebaSecreto,
		consultaCotejoPruebaIndiceHistorico,
		consultaCotejoPruebaIndiceActual,
		consultaCotejoPruebaProteccion,
		consultaCotejoPruebaExpediente,
	} {
		if strings.Contains(string(evidencia), prohibido) {
			t.Fatalf("la consulta publica filtra %q: %s", prohibido, evidencia)
		}
	}
}

func TestConsultarCotejoPublicoSeparaProtegidoEInterno(t *testing.T) {
	casos := []struct {
		nombre    string
		clase     domain.ClaseAccesoCotejo
		esperado  EstadoConsultaCotejo
		auditoria string
	}{
		{"protegido requiere identificacion", domain.ClaseAccesoCotejoProtegido, EstadoConsultaCotejoRequiereIdentificacion, "requiere_identificacion"},
		{"interno permanece oculto", domain.ClaseAccesoCotejoInterno, EstadoConsultaCotejoNoDisponible, "interno_oculto"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := consultaCotejoPruebaNuevoEntorno(t)
			entorno.codigos.codigo.Politica.ClaseAcceso = caso.clase
			if caso.clase == domain.ClaseAccesoCotejoInterno {
				entorno.codigos.codigo.Politica.CamposPublicos = nil
			}
			resultado, err := entorno.servicio.ConsultarCotejoPublico(context.Background(), consultaCotejoPruebaOrdenPublica(entorno))
			if err != nil {
				t.Fatalf("ConsultarCotejoPublico() error = %v", err)
			}
			if resultado.Estado != caso.esperado || resultado.Organo != "" || resultado.TipoDocumental != "" ||
				resultado.FechaEmision != nil || resultado.HuellaSHA256 != "" || resultado.PermiteDescarga {
				t.Fatalf("resultado = %+v", resultado)
			}
			if len(entorno.codigos.registros) != 1 || entorno.codigos.registros[0].auditoria.Result != caso.auditoria {
				t.Fatalf("auditoria = %+v", entorno.codigos.registros)
			}
		})
	}
}

func TestConsultarCotejoPublicoNoExponeRetiradoNiCaducado(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*consultaCotejoPruebaEntorno)
	}{
		{"retirado", func(entorno *consultaCotejoPruebaEntorno) {
			retirado, err := entorno.codigos.codigo.Retirar(
				"responsable-rrhh-consulta-1",
				"retirada-cotejo-consulta-001",
				"Documento retirado de la consulta",
				entorno.reloj.Ahora().Add(-time.Minute),
			)
			if err != nil {
				t.Fatalf("CodigoCotejo.Retirar() error = %v", err)
			}
			entorno.codigos.codigo = retirado
		}},
		{"caducado", func(entorno *consultaCotejoPruebaEntorno) {
			entorno.codigos.codigo.DisponibleHasta = entorno.reloj.Ahora()
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := consultaCotejoPruebaNuevoEntorno(t)
			caso.mutar(entorno)
			resultado, err := entorno.servicio.ConsultarCotejoPublico(context.Background(), consultaCotejoPruebaOrdenPublica(entorno))
			if err != nil {
				t.Fatalf("ConsultarCotejoPublico() error = %v", err)
			}
			if resultado != (ResultadoConsultaPublicaCotejo{Estado: EstadoConsultaCotejoNoDisponible}) {
				t.Fatalf("resultado = %+v", resultado)
			}
			if len(entorno.codigos.registros) != 1 || entorno.codigos.registros[0].auditoria.Result != "no_disponible" {
				t.Fatalf("auditoria = %+v", entorno.codigos.registros)
			}
		})
	}
}

func TestConsultarCotejoProtegidoAplicaCamposYTitularidadDelServidor(t *testing.T) {
	entorno := consultaCotejoPruebaNuevoEntorno(t)
	consultaCotejoPruebaExigirTitularidad(entorno, "representante", "interesada")
	entorno.autorizador.campos = []string{
		campoConsultaCotejoEstado,
		campoConsultaCotejoCodigoRef,
		campoConsultaCotejoFechaEmision,
		campoConsultaCotejoFirmaRefs,
		campoConsultaCotejoTipoDocumental,
	}
	orden := consultaCotejoPruebaOrdenProtegida(entorno, consultaCotejoPruebaPersona)

	resultado, err := entorno.servicio.ConsultarCotejoProtegido(context.Background(), orden)
	if err != nil {
		t.Fatalf("ConsultarCotejoProtegido() error = %v", err)
	}
	if resultado.Estado != EstadoConsultaCotejoDisponible || resultado.CodigoRef != entorno.codigos.codigo.Referencia() ||
		resultado.TipoDocumental != entorno.codigos.codigo.TipoDocumental || resultado.FechaEmision == nil ||
		!resultado.FechaEmision.Equal(entorno.codigos.codigo.VersionEmitida.EmitidaEn) ||
		!reflect.DeepEqual(resultado.FirmaRefs, entorno.codigos.codigo.VersionEmitida.FirmaRefs) {
		t.Fatalf("campos concedidos incorrectos: %+v", resultado)
	}
	if resultado.Documento != nil || resultado.ModuloID != "" || resultado.Clasificacion != "" ||
		resultado.Organo != "" || resultado.ExpedienteRef != "" || resultado.HuellaSHA256 != "" ||
		len(resultado.SelloTiempoRefs) != 0 || resultado.ValidacionFirmaRef != "" || resultado.RegistroRef != "" ||
		resultado.PermiteDescarga != nil {
		t.Fatalf("se expusieron campos no concedidos: %+v", resultado)
	}
	if len(entorno.documentos.consultas) != 1 || len(entorno.autorizador.solicitudes) != 1 {
		t.Fatalf("consultas documental/autorizacion = %d/%d", len(entorno.documentos.consultas), len(entorno.autorizador.solicitudes))
	}
	solicitud := entorno.autorizador.solicitudes[0]
	if solicitud.Accion != AccionConsultaProtegidaCotejo || solicitud.Recurso.Ambitos["persona"] != consultaCotejoPruebaPersona {
		t.Fatalf("contexto de autorizacion incorrecto: %+v", solicitud)
	}
	registro := entorno.codigos.registros[0]
	if registro.auditoria.RepresentedSubjectID != consultaCotejoPruebaPersona ||
		registro.auditoria.AuthorizationRef != "decision-cotejo-consulta-001" || registro.auditoria.Result != "disponible" {
		t.Fatalf("auditoria protegida incorrecta: %+v", registro.auditoria)
	}
}

func TestConsultarCotejoProtegidoDeniegaCampoNoReconocido(t *testing.T) {
	entorno := consultaCotejoPruebaNuevoEntorno(t)
	consultaCotejoPruebaExigirTitularidad(entorno, "interesada")
	entorno.autorizador.campos = []string{
		campoConsultaCotejoEstado,
		campoConsultaCotejoCodigoRef,
		campoConsultaCotejoTipoDocumental,
		"campo_futuro_no_implementado",
	}

	resultado, err := entorno.servicio.ConsultarCotejoProtegido(
		context.Background(),
		consultaCotejoPruebaOrdenProtegida(entorno, consultaCotejoPruebaPersona),
	)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || !reflect.DeepEqual(resultado, ResultadoConsultaProtegidaCotejo{}) {
		t.Fatalf("ConsultarCotejoProtegido() = (%+v, %v)", resultado, err)
	}
	if len(entorno.codigos.registros) != 1 ||
		entorno.codigos.registros[0].auditoria.Result != "proyeccion_campos_invalida" {
		t.Fatalf("rechazo no auditado: %+v", entorno.codigos.registros)
	}
}

func TestConsultarCotejoProtegidoDeniegaListaVaciaOCamposBaseAusentes(t *testing.T) {
	casos := []struct {
		nombre string
		campos []string
	}{
		{nombre: "lista vacia"},
		{nombre: "falta estado", campos: []string{campoConsultaCotejoCodigoRef, campoConsultaCotejoTipoDocumental}},
		{nombre: "falta codigo ref", campos: []string{campoConsultaCotejoEstado, campoConsultaCotejoTipoDocumental}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := consultaCotejoPruebaNuevoEntorno(t)
			consultaCotejoPruebaExigirTitularidad(entorno, "interesada")
			entorno.autorizador.campos = append([]string(nil), caso.campos...)

			resultado, err := entorno.servicio.ConsultarCotejoProtegido(
				context.Background(),
				consultaCotejoPruebaOrdenProtegida(entorno, consultaCotejoPruebaPersona),
			)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) ||
				!errors.Is(err, domain.ErrDecisionAutorizacionInvalida) ||
				!reflect.DeepEqual(resultado, ResultadoConsultaProtegidaCotejo{}) {
				t.Fatalf("ConsultarCotejoProtegido() = (%+v, %v)", resultado, err)
			}
			if len(entorno.codigos.registros) != 1 ||
				entorno.codigos.registros[0].auditoria.Result != "proyeccion_campos_invalida" {
				t.Fatalf("rechazo no auditado: %+v", entorno.codigos.registros)
			}
		})
	}
}

func TestConsultarCotejoProtegidoProyeccionMinimaExacta(t *testing.T) {
	entorno := consultaCotejoPruebaNuevoEntorno(t)
	consultaCotejoPruebaExigirTitularidad(entorno, "interesada")
	entorno.autorizador.campos = []string{campoConsultaCotejoEstado, campoConsultaCotejoCodigoRef}

	resultado, err := entorno.servicio.ConsultarCotejoProtegido(
		context.Background(),
		consultaCotejoPruebaOrdenProtegida(entorno, consultaCotejoPruebaPersona),
	)
	if err != nil {
		t.Fatalf("ConsultarCotejoProtegido() error = %v", err)
	}
	if resultado.Estado != EstadoConsultaCotejoDisponible ||
		resultado.CodigoRef != entorno.codigos.codigo.Referencia() || resultado.PermiteDescarga != nil {
		t.Fatalf("proyeccion minima = %+v", resultado)
	}
	contenido, err := json.Marshal(resultado)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var campos map[string]any
	if err := json.Unmarshal(contenido, &campos); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(campos) != 2 || campos[campoConsultaCotejoEstado] != string(EstadoConsultaCotejoDisponible) ||
		campos[campoConsultaCotejoCodigoRef] != entorno.codigos.codigo.Referencia() {
		t.Fatalf("campos de proyeccion minima = %v", campos)
	}
}

func TestConsultarCotejoProtegidoProyeccionCompletaExacta(t *testing.T) {
	entorno := consultaCotejoPruebaNuevoEntorno(t)
	consultaCotejoPruebaExigirTitularidad(entorno, "interesada")
	entorno.autorizador.campos = []string{
		campoConsultaCotejoEstado,
		campoConsultaCotejoCodigoRef,
		campoConsultaCotejoDocumentoRef,
		campoConsultaCotejoModuloID,
		campoConsultaCotejoTipoDocumental,
		campoConsultaCotejoClasificacion,
		campoConsultaCotejoOrgano,
		campoConsultaCotejoExpedienteRef,
		campoConsultaCotejoFechaEmision,
		campoConsultaCotejoHuellaSHA256,
		campoConsultaCotejoFirmaRefs,
		campoConsultaCotejoSelloTiempoRefs,
		campoConsultaCotejoValidacionFirmaRef,
		campoConsultaCotejoRegistroRef,
		campoConsultaCotejoPermiteDescarga,
		campoConsultaCotejoDescarga,
	}

	resultado, err := entorno.servicio.ConsultarCotejoProtegido(
		context.Background(),
		consultaCotejoPruebaOrdenProtegida(entorno, consultaCotejoPruebaPersona),
	)
	if err != nil {
		t.Fatalf("ConsultarCotejoProtegido() error = %v", err)
	}
	if resultado.Estado != EstadoConsultaCotejoDisponible ||
		resultado.CodigoRef != entorno.codigos.codigo.Referencia() || resultado.Documento == nil ||
		resultado.ModuloID != entorno.codigos.codigo.ModuloID ||
		resultado.TipoDocumental != entorno.codigos.codigo.TipoDocumental ||
		resultado.Clasificacion != entorno.codigos.codigo.Clasificacion ||
		resultado.Organo != entorno.codigos.codigo.Organo ||
		resultado.ExpedienteRef != entorno.codigos.codigo.ExpedienteRef || resultado.FechaEmision == nil ||
		resultado.HuellaSHA256 != entorno.codigos.codigo.VersionEmitida.HuellaContenidoSHA256 ||
		!reflect.DeepEqual(resultado.FirmaRefs, entorno.codigos.codigo.VersionEmitida.FirmaRefs) ||
		!reflect.DeepEqual(resultado.SelloTiempoRefs, entorno.codigos.codigo.VersionEmitida.SelloTiempoRefs) ||
		resultado.ValidacionFirmaRef != entorno.codigos.codigo.VersionEmitida.ValidacionFirmaRef ||
		resultado.RegistroRef != entorno.codigos.codigo.VersionEmitida.RegistroRef ||
		resultado.PermiteDescarga == nil || !*resultado.PermiteDescarga {
		t.Fatalf("proyeccion completa = %+v", resultado)
	}
	contenido, err := json.Marshal(resultado)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var campos map[string]any
	if err := json.Unmarshal(contenido, &campos); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(campos) != 15 {
		t.Fatalf("campos de proyeccion completa = %v", campos)
	}
}

func TestConsultarCotejoProtegidoDescargaExigeCampoExactoYPolitica(t *testing.T) {
	casos := []struct {
		nombre            string
		campos            []string
		politicaPermite   bool
		esperaIndicador   bool
		esperaDescargable bool
	}{
		{
			nombre:          "politica sin capacidad de descarga",
			campos:          []string{campoConsultaCotejoEstado, campoConsultaCotejoCodigoRef, campoConsultaCotejoPermiteDescarga},
			politicaPermite: true, esperaIndicador: true,
		},
		{
			nombre:          "capacidad no revela indicador no concedido",
			campos:          []string{campoConsultaCotejoEstado, campoConsultaCotejoCodigoRef, campoConsultaCotejoDescarga},
			politicaPermite: true,
		},
		{
			nombre:          "campo y capacidad no superan politica",
			campos:          []string{campoConsultaCotejoEstado, campoConsultaCotejoCodigoRef, campoConsultaCotejoPermiteDescarga, campoConsultaCotejoDescarga},
			politicaPermite: false, esperaIndicador: true,
		},
		{
			nombre:          "campo capacidad y politica conceden",
			campos:          []string{campoConsultaCotejoEstado, campoConsultaCotejoCodigoRef, campoConsultaCotejoPermiteDescarga, campoConsultaCotejoDescarga},
			politicaPermite: true, esperaIndicador: true, esperaDescargable: true,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := consultaCotejoPruebaNuevoEntorno(t)
			consultaCotejoPruebaExigirTitularidad(entorno, "interesada")
			entorno.codigos.codigo.Politica.PermiteDescargaDocumento = caso.politicaPermite
			entorno.autorizador.campos = append([]string(nil), caso.campos...)

			resultado, err := entorno.servicio.ConsultarCotejoProtegido(
				context.Background(),
				consultaCotejoPruebaOrdenProtegida(entorno, consultaCotejoPruebaPersona),
			)
			if err != nil {
				t.Fatalf("ConsultarCotejoProtegido() error = %v", err)
			}
			if (resultado.PermiteDescarga != nil) != caso.esperaIndicador {
				t.Fatalf("PermiteDescarga = %v; espera indicador = %t", resultado.PermiteDescarga, caso.esperaIndicador)
			}
			if resultado.PermiteDescarga != nil && *resultado.PermiteDescarga != caso.esperaDescargable {
				t.Fatalf("PermiteDescarga = %t; esperado = %t", *resultado.PermiteDescarga, caso.esperaDescargable)
			}
		})
	}
}

func TestConsultarCotejoProtegidoNoTitularFallaCerrado(t *testing.T) {
	casos := []struct {
		nombre string
		sujeto string
		roles  []string
	}{
		{"persona distinta", "persona-cotejo-consulta-ajena", []string{"interesada"}},
		{"rol documental no permitido", consultaCotejoPruebaPersona, []string{"representante"}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := consultaCotejoPruebaNuevoEntorno(t)
			consultaCotejoPruebaExigirTitularidad(entorno, caso.roles...)
			resultado, err := entorno.servicio.ConsultarCotejoProtegido(
				context.Background(),
				consultaCotejoPruebaOrdenProtegida(entorno, caso.sujeto),
			)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || !reflect.DeepEqual(resultado, ResultadoConsultaProtegidaCotejo{}) {
				t.Fatalf("ConsultarCotejoProtegido() = (%+v, %v)", resultado, err)
			}
			if len(entorno.autorizador.solicitudes) != 0 || len(entorno.codigos.registros) != 1 ||
				entorno.codigos.registros[0].auditoria.Result != "titularidad_no_acreditada" {
				t.Fatalf("efectos del rechazo = autorizaciones:%d registros:%+v", len(entorno.autorizador.solicitudes), entorno.codigos.registros)
			}
		})
	}
}

func TestConsultaCotejoFallaCerradoSiNoPuedeAuditar(t *testing.T) {
	falloAuditoria := errors.New("auditoria de consulta no disponible")
	t.Run("publica", func(t *testing.T) {
		entorno := consultaCotejoPruebaNuevoEntorno(t)
		entorno.codigos.errorAuditoria = falloAuditoria
		resultado, err := entorno.servicio.ConsultarCotejoPublico(context.Background(), consultaCotejoPruebaOrdenPublica(entorno))
		if !errors.Is(err, falloAuditoria) || !reflect.DeepEqual(resultado, ResultadoConsultaPublicaCotejo{}) {
			t.Fatalf("ConsultarCotejoPublico() = (%+v, %v)", resultado, err)
		}
	})

	t.Run("protegida", func(t *testing.T) {
		entorno := consultaCotejoPruebaNuevoEntorno(t)
		consultaCotejoPruebaExigirTitularidad(entorno, "interesada")
		entorno.autorizador.campos = []string{
			campoConsultaCotejoEstado,
			campoConsultaCotejoCodigoRef,
			campoConsultaCotejoTipoDocumental,
		}
		entorno.codigos.errorAuditoria = falloAuditoria
		resultado, err := entorno.servicio.ConsultarCotejoProtegido(
			context.Background(),
			consultaCotejoPruebaOrdenProtegida(entorno, consultaCotejoPruebaPersona),
		)
		if !errors.Is(err, falloAuditoria) || !reflect.DeepEqual(resultado, ResultadoConsultaProtegidaCotejo{}) {
			t.Fatalf("ConsultarCotejoProtegido() = (%+v, %v)", resultado, err)
		}
	})
}
