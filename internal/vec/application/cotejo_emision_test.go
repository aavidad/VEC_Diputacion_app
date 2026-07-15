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
	emisionCotejoPruebaSecretoVisible  = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	emisionCotejoPruebaSecretoDistinto = "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ"
	emisionCotejoPruebaIndiceHistorico = "hmac-sha256:indice-cotejo-v1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	emisionCotejoPruebaIndiceNuevo     = "hmac-sha256:indice-cotejo-v2:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	emisionCotejoPruebaHuellaSolicitud = "hmac-sha256:solicitud-cotejo-v1:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

func TestReservarCodigoCotejoConfirmaDocumentoPoliticaCustodiaYSinFiltrarSecretos(t *testing.T) {
	entorno := emisionCotejoPruebaNuevoEntorno(t)
	aplicacionPolitica, err := entorno.politica.Aplicacion()
	if err != nil {
		t.Fatalf("preparar aplicacion de politica: %v", err)
	}

	resultado, err := entorno.servicio.ReservarCodigoCotejo(context.Background(), entorno.orden)
	if err != nil {
		t.Fatalf("reservar codigo de cotejo: %v", err)
	}
	if resultado.Repetida {
		t.Fatal("la primera reserva no puede marcarse como repetida")
	}
	if resultado.Codigo.Documento != entorno.orden.Documento {
		t.Fatalf("documento confirmado = %+v; esperado %+v", resultado.Codigo.Documento, entorno.orden.Documento)
	}
	if !reflect.DeepEqual(resultado.Codigo.Politica, aplicacionPolitica) {
		t.Fatalf("politica confirmada = %+v; esperada %+v", resultado.Codigo.Politica, aplicacionPolitica)
	}
	if resultado.Secreto.Revelar() != emisionCotejoPruebaSecretoVisible {
		t.Fatalf("secreto devuelto inesperado: %q", resultado.Secreto.Revelar())
	}

	if len(entorno.documentos.consultas) != 1 || entorno.documentos.consultas[0] != entorno.orden.Documento {
		t.Fatalf("consulta documental no exacta: %+v", entorno.documentos.consultas)
	}
	if len(entorno.catalogo.consultas) != 1 || entorno.catalogo.consultas[0].id != entorno.orden.PoliticaID ||
		entorno.catalogo.consultas[0].version != entorno.orden.PoliticaVersion {
		t.Fatalf("consulta de politica no exacta: %+v", entorno.catalogo.consultas)
	}
	if len(entorno.codigos.solicitudesReserva) != 1 {
		t.Fatalf("reservas persistidas = %d; esperada 1", len(entorno.codigos.solicitudesReserva))
	}
	solicitudReserva := entorno.codigos.solicitudesReserva[0]
	if solicitudReserva.Documento != entorno.documento.Referencia() || solicitudReserva.Politica != aplicacionPolitica.Referencia {
		t.Fatalf("reserva no fija documento/politica exactos: %+v", solicitudReserva)
	}
	if solicitudReserva.HuellaSolicitudHMAC != emisionCotejoPruebaHuellaSolicitud {
		t.Fatalf("HMAC de solicitud = %q", solicitudReserva.HuellaSolicitudHMAC)
	}
	if !solicitudReserva.SolicitadaEn.Equal(entorno.ahora) ||
		!solicitudReserva.ExpiraEn.Equal(entorno.ahora.Add(vigenciaReservaOperacion)) {
		t.Fatalf("ventana de reserva inesperada: %s - %s", solicitudReserva.SolicitadaEn, solicitudReserva.ExpiraEn)
	}

	if len(entorno.protector.solicitudesProteger) != 1 {
		t.Fatalf("operaciones de custodia = %d; esperada 1", len(entorno.protector.solicitudesProteger))
	}
	solicitudCustodia := entorno.protector.solicitudesProteger[0]
	if solicitudCustodia.Secreto.Revelar() != emisionCotejoPruebaSecretoVisible ||
		solicitudCustodia.IndiceCodigoHMAC != emisionCotejoPruebaIndiceHistorico {
		t.Fatalf("custodia no recibio el secreto y su indice exactos: %+v", solicitudCustodia)
	}
	if solicitudCustodia.ClaveIdempotencia != "custodia-cotejo:codigo-cotejo-prueba-001" {
		t.Fatalf("clave idempotente de custodia = %q", solicitudCustodia.ClaveIdempotencia)
	}
	if err := solicitudCustodia.ValidarEn(entorno.ahora); err != nil {
		t.Fatalf("contexto opaco de proteccion invalido: %v", err)
	}
	if len(entorno.autorizador.solicitudes) != 2 ||
		entorno.autorizador.solicitudes[0].Accion != AccionReservarCodigoCotejo ||
		entorno.autorizador.solicitudes[1].Accion != ports.AccionProtegerCodigoCotejo {
		t.Fatalf("decisiones de reserva/proteccion no son exactas y separadas: %+v", entorno.autorizador.solicitudes)
	}
	if resultado.Codigo.IndiceCodigoHMAC != emisionCotejoPruebaIndiceHistorico ||
		resultado.Codigo.ProteccionRef != entorno.protector.custodia.ProteccionRef {
		t.Fatalf("codigo no enlaza HMAC/custodia: %+v", resultado.Codigo)
	}
	if entorno.codigos.confirmaciones != 1 || entorno.codigos.huellaConfirmada != emisionCotejoPruebaHuellaSolicitud {
		t.Fatalf("confirmaciones = %d, HMAC = %q", entorno.codigos.confirmaciones, entorno.codigos.huellaConfirmada)
	}

	if _, err := json.Marshal(resultado.Secreto); !errors.Is(err, ports.ErrSerializacionCodigoCotejoProhibida) {
		t.Fatalf("serializar el secreto: error = %v; esperado %v", err, ports.ErrSerializacionCodigoCotejoProhibida)
	}
	jsonResultado, err := json.Marshal(resultado)
	if err != nil {
		t.Fatalf("serializar resultado publico: %v", err)
	}
	for _, prohibido := range []string{
		emisionCotejoPruebaSecretoVisible,
		emisionCotejoPruebaIndiceHistorico,
		entorno.protector.custodia.ProteccionRef,
	} {
		if strings.Contains(string(jsonResultado), prohibido) {
			t.Fatalf("el JSON publico contiene material prohibido %q: %s", prohibido, jsonResultado)
		}
	}

	jsonEvidencias, err := json.Marshal(struct {
		Traza  domain.AuditEntry `json:"traza"`
		Evento domain.Event      `json:"evento"`
	}{Traza: entorno.codigos.traza, Evento: entorno.codigos.evento})
	if err != nil {
		t.Fatalf("serializar auditoria y evento: %v", err)
	}
	for _, prohibido := range []string{
		emisionCotejoPruebaSecretoVisible,
		"hmac-sha256",
		entorno.protector.custodia.ProteccionRef,
	} {
		if strings.Contains(string(jsonEvidencias), prohibido) {
			t.Fatalf("auditoria/outbox contiene material prohibido %q: %s", prohibido, jsonEvidencias)
		}
	}
	if entorno.codigos.traza.Action != eventoCodigoCotejoReservado ||
		entorno.codigos.evento.Type != eventoCodigoCotejoReservado {
		t.Fatalf("auditoria/outbox de reserva incorrectos: %+v / %+v", entorno.codigos.traza, entorno.codigos.evento)
	}
}

func TestReservarCodigoCotejoRepetidoRecuperaYAdmiteIndiceHistorico(t *testing.T) {
	entorno := emisionCotejoPruebaNuevoEntorno(t)
	primera, err := entorno.servicio.ReservarCodigoCotejo(context.Background(), entorno.orden)
	if err != nil {
		t.Fatalf("crear reserva inicial: %v", err)
	}
	entorno.codigos.reservaFijada = &ports.ReservaEmisionCodigoCotejo{
		Repetida: true,
		Codigo:   primera.Codigo,
	}
	entorno.selladorIndice.indicesConsulta = []string{
		emisionCotejoPruebaIndiceNuevo,
		emisionCotejoPruebaIndiceHistorico,
	}
	entorno.protector.recuperacion = ports.RecuperacionCodigoCotejo{
		Secreto:      primera.Secreto,
		ConectorID:   "vault-cotejo-prueba",
		EvidenciaRef: "evidencia-recuperacion-cotejo-prueba-001",
	}
	llamadasGenerador := entorno.generador.llamadas
	llamadasProteccion := len(entorno.protector.solicitudesProteger)

	repetida, err := entorno.servicio.ReservarCodigoCotejo(context.Background(), entorno.orden)
	if err != nil {
		t.Fatalf("recuperar reserva repetida: %v", err)
	}
	if !repetida.Repetida || !reflect.DeepEqual(repetida.Codigo, primera.Codigo) ||
		repetida.Secreto.Revelar() != primera.Secreto.Revelar() {
		t.Fatalf("resultado repetido inesperado: %+v", repetida)
	}
	if entorno.generador.llamadas != llamadasGenerador || len(entorno.protector.solicitudesProteger) != llamadasProteccion {
		t.Fatal("la repeticion genero o custodio un codigo nuevo")
	}
	if len(entorno.protector.solicitudesRecuperar) != 1 {
		t.Fatalf("recuperaciones = %d; esperada 1", len(entorno.protector.solicitudesRecuperar))
	}
	solicitud := entorno.protector.solicitudesRecuperar[0]
	if solicitud.ProteccionRef != primera.Codigo.ProteccionRef ||
		solicitud.IndiceCodigoHMACEsperado != emisionCotejoPruebaIndiceHistorico {
		t.Fatalf("recuperacion no fija custodia/indice historico: %+v", solicitud)
	}
	if err := solicitud.ValidarEn(entorno.ahora); err != nil {
		t.Fatalf("contexto opaco de recuperacion invalido: %v", err)
	}
	if len(entorno.autorizador.solicitudes) != 4 ||
		entorno.autorizador.solicitudes[2].Accion != AccionReservarCodigoCotejo ||
		entorno.autorizador.solicitudes[3].Accion != ports.AccionRecuperarCodigoCotejo {
		t.Fatalf("decisiones de repeticion/recuperacion no son exactas y separadas: %+v", entorno.autorizador.solicitudes)
	}
	if entorno.selladorIndice.consultas != 1 ||
		entorno.selladorIndice.secretoConsulta.Revelar() != primera.Secreto.Revelar() {
		t.Fatalf("el secreto recuperado no se verifico: consultas=%d", entorno.selladorIndice.consultas)
	}
}

func TestReservarCodigoCotejoRepetidoConIndiceDistintoFallaCerrado(t *testing.T) {
	entorno := emisionCotejoPruebaNuevoEntorno(t)
	primera, err := entorno.servicio.ReservarCodigoCotejo(context.Background(), entorno.orden)
	if err != nil {
		t.Fatalf("crear reserva inicial: %v", err)
	}
	secretoDistinto := emisionCotejoPruebaNuevoSecreto(t, emisionCotejoPruebaSecretoDistinto)
	entorno.codigos.reservaFijada = &ports.ReservaEmisionCodigoCotejo{Repetida: true, Codigo: primera.Codigo}
	entorno.protector.recuperacion = ports.RecuperacionCodigoCotejo{
		Secreto:      secretoDistinto,
		ConectorID:   "vault-cotejo-prueba",
		EvidenciaRef: "evidencia-recuperacion-cotejo-prueba-002",
	}
	entorno.selladorIndice.indicesConsulta = []string{emisionCotejoPruebaIndiceNuevo}
	confirmaciones := entorno.codigos.confirmaciones
	llamadasGenerador := entorno.generador.llamadas

	resultado, err := entorno.servicio.ReservarCodigoCotejo(context.Background(), entorno.orden)
	if !errors.Is(err, ErrResultadoCotejoInvalido) || !errors.Is(err, ports.ErrValorCodigoCotejoNoDisponible) {
		t.Fatalf("error = %v; esperado fallo cerrado de valor no disponible", err)
	}
	if resultado.Secreto.Revelar() != "" || resultado.Codigo.ID != "" {
		t.Fatalf("el fallo devolvio material parcial: %+v", resultado)
	}
	if entorno.codigos.confirmaciones != confirmaciones || entorno.generador.llamadas != llamadasGenerador {
		t.Fatal("el fallo cerrado confirmo o genero un codigo nuevo")
	}
}

func TestReservarCodigoCotejoRechazaBajaEntropiaYAbandonaReserva(t *testing.T) {
	entorno := emisionCotejoPruebaNuevoEntorno(t)
	entorno.generador.resultado.EntropiaBits = minimoEntropiaCotejoAplicacion - 1

	resultado, err := entorno.servicio.ReservarCodigoCotejo(context.Background(), entorno.orden)
	if !errors.Is(err, ports.ErrMaterialCodigoCotejoInvalido) {
		t.Fatalf("error = %v; esperado %v", err, ports.ErrMaterialCodigoCotejoInvalido)
	}
	if resultado.Codigo.ID != "" || resultado.Secreto.Revelar() != "" {
		t.Fatalf("el fallo devolvio material parcial: %+v", resultado)
	}
	if entorno.generador.llamadas != 1 {
		t.Fatalf("llamadas al generador = %d; esperada 1", entorno.generador.llamadas)
	}
	if !reflect.DeepEqual(entorno.codigos.reservasAbandonadas, []string{"token-reserva-cotejo-prueba-001"}) {
		t.Fatalf("reservas abandonadas = %+v", entorno.codigos.reservasAbandonadas)
	}
	if entorno.selladorIndice.sellados != 0 || len(entorno.protector.solicitudesProteger) != 0 ||
		entorno.codigos.confirmaciones != 0 {
		t.Fatal("la baja entropia alcanzo indexacion, custodia o confirmacion")
	}
}

func TestReservarCodigoCotejoRechazaEstadoDocumentalTardioAntesDeGenerar(t *testing.T) {
	entorno := emisionCotejoPruebaNuevoEntorno(t)
	entorno.documentos.documento.Estado = domain.EstadoDocumentoLogicoRegistrado

	_, err := entorno.servicio.ReservarCodigoCotejo(context.Background(), entorno.orden)
	if !errors.Is(err, domain.ErrDocumentoLogicoInvalido) {
		t.Fatalf("error = %v; esperado %v", err, domain.ErrDocumentoLogicoInvalido)
	}
	if len(entorno.catalogo.consultas) != 0 || entorno.autorizador.llamadas != 0 ||
		entorno.generador.llamadas != 0 || len(entorno.protector.solicitudesProteger) != 0 ||
		len(entorno.codigos.solicitudesReserva) != 0 {
		t.Fatal("el estado tardio alcanzo politica, autorizacion, reserva, generador o custodia")
	}
}

func TestReservarCodigoCotejoAutorizacionDenegadaNoGeneraNiCustodia(t *testing.T) {
	entorno := emisionCotejoPruebaNuevoEntorno(t)
	entorno.autorizador.denegar = true

	_, err := entorno.servicio.ReservarCodigoCotejo(context.Background(), entorno.orden)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("error = %v; esperado %v", err, domain.ErrAutorizacionDenegada)
	}
	if entorno.autorizador.llamadas != 1 {
		t.Fatalf("llamadas de autorizacion = %d; esperada 1", entorno.autorizador.llamadas)
	}
	if entorno.generador.llamadas != 0 || len(entorno.protector.solicitudesProteger) != 0 ||
		len(entorno.protector.solicitudesRecuperar) != 0 || len(entorno.codigos.solicitudesReserva) != 0 ||
		entorno.selladorSolicitud.llamadas != 0 {
		t.Fatal("la denegacion alcanzo sellado, reserva, generador o custodia")
	}
}

func TestReservarCodigoCotejoNoReinterpretaDecisionReservaParaProteger(t *testing.T) {
	entorno := emisionCotejoPruebaNuevoEntorno(t)
	entorno.autorizador.reutilizarDecisionRef = true

	resultado, err := entorno.servicio.ReservarCodigoCotejo(context.Background(), entorno.orden)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || !reflect.DeepEqual(resultado, ResultadoReservaCodigoCotejo{}) {
		t.Fatalf("ReservarCodigoCotejo() = (%+v, %v); esperaba denegacion cerrada", resultado, err)
	}
	if len(entorno.autorizador.solicitudes) != 2 || len(entorno.protector.solicitudesProteger) != 0 ||
		entorno.codigos.confirmaciones != 0 {
		t.Fatalf("decision reutilizada alcanzo custodia/confirmacion: solicitudes=%d proteger=%d confirmar=%d",
			len(entorno.autorizador.solicitudes), len(entorno.protector.solicitudesProteger), entorno.codigos.confirmaciones)
	}
}

func TestReservarCodigoCotejoFalloTrasCustodiaNoBorraSinAutoridadTecnica(t *testing.T) {
	entorno := emisionCotejoPruebaNuevoEntorno(t)
	fallo := errors.New("fallo transaccional de confirmacion")
	entorno.codigos.falloConfirmacion = fallo

	_, err := entorno.servicio.ReservarCodigoCotejo(context.Background(), entorno.orden)
	if !errors.Is(err, fallo) {
		t.Fatalf("error = %v; esperado %v", err, fallo)
	}
	if len(entorno.protector.solicitudesProteger) != 1 || len(entorno.protector.solicitudesEliminar) != 0 {
		t.Fatalf("la limpieza sin autoridad se ejecuto: proteger=%d eliminar=%d",
			len(entorno.protector.solicitudesProteger), len(entorno.protector.solicitudesEliminar))
	}
}

type emisionCotejoPruebaEntorno struct {
	servicio          *ServicioCotejo
	ahora             time.Time
	documento         domain.DocumentoLogico
	politica          domain.PoliticaCotejo
	orden             OrdenReservarCodigoCotejo
	catalogo          *emisionCotejoPruebaCatalogo
	documentos        *emisionCotejoPruebaDocumentos
	autorizador       *emisionCotejoPruebaAutorizador
	codigos           *emisionCotejoPruebaCodigos
	generador         *emisionCotejoPruebaGenerador
	selladorIndice    *emisionCotejoPruebaSelladorIndice
	selladorSolicitud *emisionCotejoPruebaSelladorSolicitud
	protector         *emisionCotejoPruebaProtector
}

func emisionCotejoPruebaNuevoEntorno(t *testing.T) *emisionCotejoPruebaEntorno {
	t.Helper()
	ahora := time.Date(2026, time.July, 14, 10, 30, 0, 0, time.UTC)
	documento := emisionCotejoPruebaDocumento(ahora)
	politica := emisionCotejoPruebaPolitica(ahora)
	secreto := emisionCotejoPruebaNuevoSecreto(t, emisionCotejoPruebaSecretoVisible)
	catalogo := &emisionCotejoPruebaCatalogo{politica: politica}
	documentos := &emisionCotejoPruebaDocumentos{documento: documento}
	autorizador := &emisionCotejoPruebaAutorizador{ahora: ahora}
	codigos := &emisionCotejoPruebaCodigos{}
	generador := &emisionCotejoPruebaGenerador{resultado: ports.ValorCodigoCotejoGenerado{
		Secreto:          secreto,
		EntropiaBits:     minimoEntropiaCotejoAplicacion,
		VersionGenerador: "generador-cotejo-prueba-v1",
	}}
	generadorID := &emisionCotejoPruebaGeneradorID{id: "codigo-cotejo-prueba-001"}
	selladorIndice := &emisionCotejoPruebaSelladorIndice{indice: emisionCotejoPruebaIndiceHistorico}
	selladorSolicitud := &emisionCotejoPruebaSelladorSolicitud{huella: emisionCotejoPruebaHuellaSolicitud}
	protector := &emisionCotejoPruebaProtector{ahora: ahora, custodia: ports.CustodiaCodigoCotejo{
		ProteccionRef: "custodia-cotejo-prueba-001",
		ConectorID:    "vault-cotejo-prueba",
		EvidenciaRef:  "evidencia-custodia-cotejo-prueba-001",
	}}
	reloj := emisionCotejoPruebaReloj{ahora: ahora}
	servicio, err := NuevoServicioCotejo(
		catalogo,
		emisionCotejoPruebaGobierno{},
		codigos,
		documentos,
		autorizador,
		generador,
		generadorID,
		selladorIndice,
		selladorSolicitud,
		protector,
		emisionCotejoPruebaEvidencias{},
		reloj,
	)
	if err != nil {
		t.Fatalf("crear servicio de cotejo: %v", err)
	}
	orden := OrdenReservarCodigoCotejo{
		Principal: domain.Principal{
			ID:            personaAutorizacionPrueba("tecnico-rrhh-cotejo-prueba"),
			DisplayName:   "Tecnico RRHH",
			Roles:         []string{"tecnico_rrhh"},
			AuthMethod:    domain.AuthMethodCertificate,
			AuthAssurance: domain.AuthAssuranceHigh,
		},
		PerfilActivo:      perfilAutorizacionPrueba("tecnico-rrhh-cotejo-prueba"),
		RepresentadoRef:   "persona-representada-cotejo-prueba",
		Finalidad:         "emitir_certificado_verificable",
		ClaveIdempotencia: "idempotencia-cotejo-prueba-001",
		Documento:         documento.Referencia(),
		PoliticaID:        politica.ID,
		PoliticaVersion:   politica.Version,
		Motivo:            "reservar codigo para el documento",
		CorrelacionRef:    "correlacion-cotejo-prueba-001",
	}
	return &emisionCotejoPruebaEntorno{
		servicio:          servicio,
		ahora:             ahora,
		documento:         documento,
		politica:          politica,
		orden:             orden,
		catalogo:          catalogo,
		documentos:        documentos,
		autorizador:       autorizador,
		codigos:           codigos,
		generador:         generador,
		selladorIndice:    selladorIndice,
		selladorSolicitud: selladorSolicitud,
		protector:         protector,
	}
}

func emisionCotejoPruebaDocumento(ahora time.Time) domain.DocumentoLogico {
	return domain.DocumentoLogico{
		ID:       "documento-cotejo-prueba-001",
		Version:  3,
		Revision: 1,
		VersionAnterior: &domain.ReferenciaDocumento{
			ID:      "documento-cotejo-prueba-001",
			Version: 2,
		},
		Plantilla: domain.ReferenciaPlantillaDocumento{
			ID:           "certificado_bolsa_cotejo",
			Version:      4,
			HuellaSHA256: strings.Repeat("d", 64),
		},
		ModuloID:       "bolsa",
		TipoDocumental: "certificado",
		Clasificacion:  "restringido",
		Relaciones: []domain.RelacionDocumento{{
			Tipo:       domain.TipoRelacionExpediente,
			Referencia: "expediente-cotejo-prueba-001",
			Rol:        "principal",
		}},
		Estado:           domain.EstadoDocumentoLogicoCerrado,
		HuellaDatosHMAC:  "hmac-sha256:datos-documento-cotejo-v1:" + strings.Repeat("e", 64),
		HuellaFuenteHMAC: "hmac-sha256:fuente-documento-cotejo-v1:" + strings.Repeat("f", 64),
		CreadoPor:        "tecnico-rrhh-cotejo-prueba",
		CreadoEn:         ahora.Add(-3 * time.Hour),
		CorrelacionRef:   "correlacion-documento-cotejo-prueba",
		Motivo:           "generar certificado para cotejo",
		ENI: domain.MetadatosENI{
			Identificador:     "ES_L01000000_2026_DOC_COTEJO_001",
			Organo:            "L01000000",
			Origen:            "administracion",
			EstadoElaboracion: "original",
			TipoDocumental:    "certificado",
			FechaCaptura:      ahora.Add(-3 * time.Hour),
		},
	}
}

func emisionCotejoPruebaPolitica(ahora time.Time) domain.PoliticaCotejo {
	return domain.PoliticaCotejo{
		ID:                       "certificados_bolsa_cotejo",
		Version:                  7,
		Revision:                 2,
		VersionAnteriorRef:       "politica-cotejo:certificados_bolsa_cotejo:v6",
		Nombre:                   "Cotejo de certificados de bolsa",
		Descripcion:              "Politica publicada para verificar certificados de bolsa",
		Modulos:                  []string{"bolsa"},
		TiposDocumentales:        []string{"certificado"},
		Clasificaciones:          []string{"restringido"},
		ClaseAcceso:              domain.ClaseAccesoCotejoProtegido,
		CamposPublicos:           []domain.CampoPublicoCotejo{domain.CampoPublicoCotejoOrgano},
		PermiteDescargaDocumento: false,
		RequiereTitularidad:      false,
		RequiereFirma:            false,
		RequiereSelloTiempo:      false,
		RequiereRegistro:         false,
		GarantiaMinima:           domain.AuthAssuranceHigh,
		DiasPlazoActivacion:      30,
		DiasDisponibilidad:       365,
		Estado:                   domain.EstadoPoliticaCotejoPublicada,
		FuenteRef:                "normativa-cotejo-prueba-2026",
		MotivoCreacion:           "permitir verificacion protegida",
		CreadaPor:                "responsable-rrhh-cotejo-prueba",
		CreadaEn:                 ahora.Add(-2 * time.Hour),
		PublicadaPor:             "responsable-seguridad-cotejo-prueba",
		PublicadaEn:              ahora.Add(-time.Hour),
		AprobacionRef:            "aprobacion-cotejo-prueba-001",
		MotivoPublicacion:        "politica revisada y aprobada",
	}
}

func emisionCotejoPruebaNuevoSecreto(t *testing.T, valor string) ports.SecretoCodigoCotejo {
	t.Helper()
	secreto, err := ports.NuevoSecretoCodigoCotejo(valor)
	if err != nil {
		t.Fatalf("crear secreto de cotejo de prueba: %v", err)
	}
	return secreto
}

type emisionCotejoPruebaConsultaPolitica struct {
	id      string
	version int
}

type emisionCotejoPruebaCatalogo struct {
	politica  domain.PoliticaCotejo
	consultas []emisionCotejoPruebaConsultaPolitica
}

func (c *emisionCotejoPruebaCatalogo) ObtenerPoliticaCotejo(_ context.Context, id string, version int) (domain.PoliticaCotejo, error) {
	c.consultas = append(c.consultas, emisionCotejoPruebaConsultaPolitica{id: id, version: version})
	return c.politica, nil
}

func (c *emisionCotejoPruebaCatalogo) ListarVersionesPoliticaCotejo(context.Context, string) ([]domain.PoliticaCotejo, error) {
	return []domain.PoliticaCotejo{c.politica}, nil
}

type emisionCotejoPruebaGobierno struct{}

func (emisionCotejoPruebaGobierno) ConfirmarAltaBorradorPoliticaCotejo(context.Context, domain.PoliticaCotejo, domain.AuditEntry, domain.Event) error {
	return nil
}

func (emisionCotejoPruebaGobierno) ConfirmarActualizacionBorradorPoliticaCotejo(context.Context, string, domain.PoliticaCotejo, domain.AuditEntry, domain.Event) error {
	return nil
}

func (emisionCotejoPruebaGobierno) ConfirmarPublicacionPoliticaCotejo(context.Context, string, domain.PoliticaCotejo, domain.AuditEntry, domain.Event) error {
	return nil
}

func (emisionCotejoPruebaGobierno) ConfirmarRetiradaPoliticaCotejo(context.Context, string, domain.PoliticaCotejo, domain.AuditEntry, domain.Event) error {
	return nil
}

type emisionCotejoPruebaDocumentos struct {
	documento domain.DocumentoLogico
	consultas []domain.ReferenciaDocumento
}

func (*emisionCotejoPruebaDocumentos) ReservarGeneracion(context.Context, ports.SolicitudReservarGeneracionDocumento) (ports.ReservaGeneracionDocumento, error) {
	return ports.ReservaGeneracionDocumento{}, nil
}

func (*emisionCotejoPruebaDocumentos) ConfirmarGeneracionLogica(context.Context, string, string, time.Time, domain.ResultadoGeneracionDocumento, domain.AuditEntry, domain.Event) error {
	return nil
}

func (*emisionCotejoPruebaDocumentos) AbandonarGeneracion(context.Context, string) error { return nil }

func (d *emisionCotejoPruebaDocumentos) ObtenerDocumentoLogico(_ context.Context, referencia domain.ReferenciaDocumento) (domain.DocumentoLogico, error) {
	d.consultas = append(d.consultas, referencia)
	return d.documento, nil
}

func (*emisionCotejoPruebaDocumentos) ListarRepresentacionesDocumento(context.Context, domain.ReferenciaDocumento) ([]domain.RepresentacionDocumento, error) {
	return nil, nil
}

type emisionCotejoPruebaAutorizador struct {
	ahora                 time.Time
	denegar               bool
	reutilizarDecisionRef bool
	llamadas              int
	solicitudes           []domain.SolicitudAutorizacion
}

func (a *emisionCotejoPruebaAutorizador) Exigir(_ context.Context, solicitud domain.SolicitudAutorizacion) (domain.DecisionAutorizacion, error) {
	a.llamadas++
	a.solicitudes = append(a.solicitudes, solicitud)
	if a.denegar {
		return domain.DecisionAutorizacion{}, domain.ErrAutorizacionDenegada
	}
	decisionRef := "decision-cotejo-prueba:" + solicitud.Accion
	if a.reutilizarDecisionRef {
		decisionRef = "decision-cotejo-prueba:reutilizada"
	}
	return completarDecisionAutorizacionPrueba(solicitud, domain.DecisionAutorizacion{
		DecisionRef:            decisionRef,
		Concedida:              true,
		Codigo:                 "concedida",
		PrincipalID:            strings.TrimSpace(solicitud.Principal.ID),
		PerfilActivoRef:        strings.TrimSpace(solicitud.PerfilActivoRef),
		Accion:                 strings.TrimSpace(solicitud.Accion),
		RecursoRef:             strings.TrimSpace(solicitud.Recurso.Referencia),
		Finalidad:              strings.TrimSpace(solicitud.Finalidad),
		CorrelacionRef:         strings.TrimSpace(solicitud.CorrelacionRef),
		AsignacionRef:          "asignacion-cotejo-prueba-v1",
		AsignacionHuellaSHA256: strings.Repeat("1", 64),
		VersionRolRef:          "rol-cotejo-prueba-v1",
		VersionRolHuellaSHA256: strings.Repeat("2", 64),
		GarantiaMinima:         domain.AuthAssuranceHigh,
		EmitidaEn:              a.ahora.Add(-time.Minute),
		ValidaHasta:            a.ahora.Add(time.Minute),
	}), nil
}

type emisionCotejoPruebaCodigos struct {
	reservaFijada       *ports.ReservaEmisionCodigoCotejo
	solicitudesReserva  []ports.SolicitudReservarEmisionCodigoCotejo
	reservasAbandonadas []string
	confirmaciones      int
	huellaConfirmada    string
	codigo              domain.CodigoCotejo
	traza               domain.AuditEntry
	evento              domain.Event
	falloConfirmacion   error
}

func (r *emisionCotejoPruebaCodigos) ReservarEmisionCodigoCotejo(_ context.Context, solicitud ports.SolicitudReservarEmisionCodigoCotejo) (ports.ReservaEmisionCodigoCotejo, error) {
	r.solicitudesReserva = append(r.solicitudesReserva, solicitud)
	if r.reservaFijada != nil {
		return *r.reservaFijada, nil
	}
	return ports.ReservaEmisionCodigoCotejo{Token: "token-reserva-cotejo-prueba-001"}, nil
}

func (r *emisionCotejoPruebaCodigos) ConfirmarReservaCodigoCotejo(_ context.Context, _ string, huella string, _ time.Time, codigo domain.CodigoCotejo, traza domain.AuditEntry, evento domain.Event) error {
	r.confirmaciones++
	r.huellaConfirmada = huella
	r.codigo = codigo
	r.traza = traza
	r.evento = evento
	return r.falloConfirmacion
}

func (r *emisionCotejoPruebaCodigos) AbandonarReservaCodigoCotejo(_ context.Context, token string) error {
	r.reservasAbandonadas = append(r.reservasAbandonadas, token)
	return nil
}

func (r *emisionCotejoPruebaCodigos) ObtenerCodigoCotejo(context.Context, string) (domain.CodigoCotejo, error) {
	return r.codigo, nil
}

func (r *emisionCotejoPruebaCodigos) ObtenerCodigoCotejoPorDocumento(context.Context, domain.ReferenciaDocumento) (domain.CodigoCotejo, error) {
	return r.codigo, nil
}

func (r *emisionCotejoPruebaCodigos) BuscarCodigoCotejoPorIndices(context.Context, []string) (domain.CodigoCotejo, error) {
	return r.codigo, nil
}

func (*emisionCotejoPruebaCodigos) ConfirmarActivacionCodigoCotejo(context.Context, string, domain.CodigoCotejo, domain.AuditEntry, domain.Event) error {
	return nil
}

func (*emisionCotejoPruebaCodigos) ConfirmarRetiradaCodigoCotejo(context.Context, string, domain.CodigoCotejo, domain.AuditEntry, domain.Event) error {
	return nil
}

func (*emisionCotejoPruebaCodigos) ConfirmarSustitucionCodigoCotejo(context.Context, string, domain.CodigoCotejo, domain.AuditEntry, domain.Event) error {
	return nil
}

func (*emisionCotejoPruebaCodigos) RegistrarConsultaCotejo(context.Context, domain.AuditEntry, domain.Event) error {
	return nil
}

type emisionCotejoPruebaGenerador struct {
	resultado ports.ValorCodigoCotejoGenerado
	llamadas  int
}

func (g *emisionCotejoPruebaGenerador) GenerarValorCodigoCotejo(context.Context) (ports.ValorCodigoCotejoGenerado, error) {
	g.llamadas++
	return g.resultado, nil
}

type emisionCotejoPruebaGeneradorID struct{ id string }

func (g *emisionCotejoPruebaGeneradorID) NuevoIDCodigoCotejo() (string, error) { return g.id, nil }

type emisionCotejoPruebaSelladorIndice struct {
	indice          string
	indicesConsulta []string
	sellados        int
	consultas       int
	secretoSellado  ports.SecretoCodigoCotejo
	secretoConsulta ports.SecretoCodigoCotejo
}

func (s *emisionCotejoPruebaSelladorIndice) SellarIndiceCodigoCotejo(_ context.Context, secreto ports.SecretoCodigoCotejo) (string, error) {
	s.sellados++
	s.secretoSellado = secreto
	return s.indice, nil
}

func (s *emisionCotejoPruebaSelladorIndice) SellarIndicesConsultaCodigoCotejo(_ context.Context, secreto ports.SecretoCodigoCotejo) ([]string, error) {
	s.consultas++
	s.secretoConsulta = secreto
	return append([]string(nil), s.indicesConsulta...), nil
}

type emisionCotejoPruebaSelladorSolicitud struct {
	huella    string
	llamadas  int
	contenido []byte
}

func (s *emisionCotejoPruebaSelladorSolicitud) SellarSolicitudCotejo(_ context.Context, contenido []byte) (string, error) {
	s.llamadas++
	s.contenido = append([]byte(nil), contenido...)
	return s.huella, nil
}

type emisionCotejoPruebaProtector struct {
	ahora                time.Time
	custodia             ports.CustodiaCodigoCotejo
	recuperacion         ports.RecuperacionCodigoCotejo
	solicitudesProteger  []ports.SolicitudProtegerCodigoCotejo
	solicitudesRecuperar []ports.SolicitudRecuperarCodigoCotejo
	solicitudesEliminar  []ports.SolicitudEliminarCodigoCotejoHuerfano
}

func (p *emisionCotejoPruebaProtector) ProtegerCodigoCotejo(_ context.Context, solicitud ports.SolicitudProtegerCodigoCotejo) (ports.CustodiaCodigoCotejo, error) {
	if err := solicitud.ValidarEn(p.ahora); err != nil {
		return ports.CustodiaCodigoCotejo{}, err
	}
	p.solicitudesProteger = append(p.solicitudesProteger, solicitud)
	return p.custodia, nil
}

func (p *emisionCotejoPruebaProtector) RecuperarCodigoCotejo(_ context.Context, solicitud ports.SolicitudRecuperarCodigoCotejo) (ports.RecuperacionCodigoCotejo, error) {
	if err := solicitud.ValidarEn(p.ahora); err != nil {
		return ports.RecuperacionCodigoCotejo{}, err
	}
	p.solicitudesRecuperar = append(p.solicitudesRecuperar, solicitud)
	return p.recuperacion, nil
}

func (p *emisionCotejoPruebaProtector) EliminarCodigoCotejoHuerfano(_ context.Context, solicitud ports.SolicitudEliminarCodigoCotejoHuerfano) error {
	if err := solicitud.ValidarEn(p.ahora); err != nil {
		return err
	}
	p.solicitudesEliminar = append(p.solicitudesEliminar, solicitud)
	return nil
}

type emisionCotejoPruebaEvidencias struct{}

func (emisionCotejoPruebaEvidencias) ObtenerEvidenciaEmisionDocumento(context.Context, ports.SolicitudObtenerEvidenciaEmisionDocumento) (domain.EvidenciaEmisionDocumento, error) {
	return domain.EvidenciaEmisionDocumento{}, nil
}

type emisionCotejoPruebaReloj struct{ ahora time.Time }

func (r emisionCotejoPruebaReloj) Ahora() time.Time { return r.ahora }
