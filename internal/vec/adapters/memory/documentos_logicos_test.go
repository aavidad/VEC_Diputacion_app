package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	huellaPlantillaLogicaPrueba = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	huellaDatosLogicaPrueba     = "hmac-sha256:documentos-1:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	huellaFuenteLogicaPrueba    = "hmac-sha256:documentos-1:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	huellaFuenteLogicaDistinta  = "hmac-sha256:documentos-1:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
)

func resultadoDocumentoLogicoMemoriaPrueba(t *testing.T, store *Store, principalID string, fecha time.Time) domain.ResultadoGeneracionDocumento {
	t.Helper()
	documento := domain.DocumentoLogico{
		ID:       "documento-logico-001",
		Version:  1,
		Revision: 1,
		Plantilla: domain.ReferenciaPlantillaDocumento{
			ID:           "contrato_bolsa",
			Version:      7,
			HuellaSHA256: huellaPlantillaLogicaPrueba,
		},
		ModuloID:       "bolsa",
		TipoDocumental: "contrato",
		Clasificacion:  "datos_personales_alta",
		Relaciones: []domain.RelacionDocumento{
			{Tipo: domain.TipoRelacionExpediente, Referencia: "expediente-042", Rol: "principal"},
			{Tipo: domain.TipoRelacionPersona, Referencia: "persona-001", Rol: "interesada"},
		},
		Estado:           domain.EstadoDocumentoLogicoBorrador,
		HuellaDatosHMAC:  huellaDatosLogicaPrueba,
		HuellaFuenteHMAC: huellaFuenteLogicaPrueba,
		CreadoPor:        principalID,
		CreadoEn:         fecha,
		CorrelacionRef:   "correlacion-logica-001",
		Motivo:           "Generacion de contrato tras llamamiento aceptado",
		ENI: domain.MetadatosENI{
			Identificador:     "documento-logico-001",
			Organo:            "ORGANO-PRUEBA",
			Origen:            "administracion",
			EstadoElaboracion: "original",
			TipoDocumental:    "contrato",
			FechaCaptura:      fecha,
		},
	}
	tipos := []struct {
		id      string
		tipo    domain.TipoRepresentacionDocumento
		formato domain.FormatoDocumento
		datos   []byte
	}{
		{id: "representacion-docx-001", tipo: domain.TipoRepresentacionTrabajo, formato: domain.FormatoDocumentoDOCX, datos: []byte("PK-documento-docx")},
		{id: "representacion-pdf-001", tipo: domain.TipoRepresentacionVisualizacion, formato: domain.FormatoDocumentoPDF, datos: []byte("%PDF-documento-pdf")},
	}
	representaciones := make([]domain.RepresentacionDocumento, 0, len(tipos))
	for _, tipo := range tipos {
		suma := sha256.Sum256(tipo.datos)
		huella := hex.EncodeToString(suma[:])
		solicitudContenido := solicitudContenidoDocumentalMemoriaPrueba(
			t, tipo.id, tipo.id, "contenido-"+tipo.id,
			tipo.formato, ports.ZonaAlmacenAdmitida, tipo.datos,
		)
		guardado, err := store.GuardarContenido(context.Background(), solicitudContenido)
		if err != nil {
			t.Fatalf("GuardarContenido() error = %v", err)
		}
		representaciones = append(representaciones, domain.RepresentacionDocumento{
			ID:                    tipo.id,
			Documento:             documento.Referencia(),
			Tipo:                  tipo.tipo,
			Formato:               tipo.formato,
			MIME:                  tipo.formato.MIME(),
			NombreFichero:         tipo.id + tipo.formato.Extension(),
			Tamano:                int64(len(tipo.datos)),
			HuellaContenidoSHA256: huella,
			HuellaFuenteHMAC:      documento.HuellaFuenteHMAC,
			ReferenciaContenido:   guardado.Referencia,
			EstadoTecnico:         domain.EstadoRepresentacionDisponible,
			EstadoAntivirus:       domain.EstadoAntivirusNoAplica,
			GeneradaPor:           principalID,
			GeneradaEn:            fecha,
		})
	}
	resultado := domain.ResultadoGeneracionDocumento{Documento: documento, Representaciones: representaciones}
	if err := resultado.Validar(); err != nil {
		t.Fatalf("resultado de prueba invalido: %v", err)
	}
	return resultado
}

func evidenciaDocumentoLogicoMemoriaPrueba(resultado domain.ResultadoGeneracionDocumento, fecha time.Time) (domain.AuditEntry, domain.Event) {
	documento := resultado.Documento
	referencia := documento.ID + ":" + strconv.Itoa(documento.Version)
	traza := domain.AuditEntry{
		ActorID:          documento.CreadoPor,
		ActorProfile:     "tecnico_rrhh",
		AuthMethod:       domain.AuthMethodCertificate,
		AuthAssurance:    domain.AuthAssuranceHigh,
		AuthorizationRef: "decision-interna-001",
		Purpose:          "gestion_contratacion_temporal",
		Action:           "vec.documento.logico.generado",
		ModuleID:         documento.ModuloID,
		SubjectRef:       referencia,
		ObjectVersion:    documento.Version,
		DocumentRef:      referencia,
		RuleRef:          documento.Plantilla.ID + ":" + strconv.Itoa(documento.Plantilla.Version),
		Reason:           documento.Motivo,
		Result:           "correcto",
		AfterHash:        documento.HuellaFuenteHMAC,
		CorrelationRef:   documento.CorrelacionRef,
		OccurredAt:       fecha,
	}
	evento := domain.Event{
		Type:       "vec.documento.logico.generado",
		ModuleID:   documento.ModuloID,
		SubjectRef: referencia,
		ActorID:    documento.CreadoPor,
		OccurredAt: fecha,
		Payload: map[string]string{
			"documento_ref":      documento.ID,
			"documento_version":  strconv.Itoa(documento.Version),
			"huella_fuente_hmac": documento.HuellaFuenteHMAC,
			"representaciones":   strconv.Itoa(len(resultado.Representaciones)),
		},
	}
	return traza, evento
}

func solicitudReservaLogicaPrueba(principalID string, fecha time.Time) ports.SolicitudReservarGeneracionDocumento {
	return ports.SolicitudReservarGeneracionDocumento{
		ClaveIdempotencia:   "operacion-documental-00000001",
		PrincipalID:         principalID,
		HuellaSolicitudHMAC: huellaFuenteLogicaPrueba,
		SolicitadaEn:        fecha,
		ExpiraEn:            fecha.Add(5 * time.Minute),
	}
}

func TestRepositorioDocumentosLogicosConfirmaYRepiteIdempotente(t *testing.T) {
	store := NewStore()
	fecha := time.Date(2026, time.July, 14, 17, 0, 0, 0, time.UTC)
	solicitud := solicitudReservaLogicaPrueba("tecnico-rrhh-1", fecha.Add(-time.Minute))
	reserva, err := store.ReservarGeneracion(context.Background(), solicitud)
	if err != nil || reserva.Token == "" || reserva.Repetida {
		t.Fatalf("ReservarGeneracion() = %+v, %v", reserva, err)
	}
	resultado := resultadoDocumentoLogicoMemoriaPrueba(t, store, solicitud.PrincipalID, fecha)
	traza, evento := evidenciaDocumentoLogicoMemoriaPrueba(resultado, fecha)
	if err := store.ConfirmarGeneracionLogica(context.Background(), reserva.Token, solicitud.HuellaSolicitudHMAC, fecha, resultado, traza, evento); err != nil {
		t.Fatalf("ConfirmarGeneracionLogica() error = %v", err)
	}

	solicitud.SolicitadaEn = fecha.Add(time.Second)
	solicitud.ExpiraEn = solicitud.SolicitadaEn.Add(5 * time.Minute)
	repetida, err := store.ReservarGeneracion(context.Background(), solicitud)
	if err != nil || !repetida.Repetida || repetida.Token != "" || !repetida.Resultado.Repetida {
		t.Fatalf("repeticion = %+v, %v", repetida, err)
	}
	if err := repetida.Resultado.Validar(); err != nil {
		t.Fatalf("resultado repetido invalido: %v", err)
	}
	repetida.Resultado.Documento.Relaciones[0].Referencia = "alterada"
	repetida.Resultado.Representaciones[0].NombreFichero = "alterado.docx"
	guardado, err := store.ObtenerDocumentoLogico(context.Background(), resultado.Documento.Referencia())
	if err != nil || guardado.Relaciones[0].Referencia == "alterada" {
		t.Fatalf("documento guardado comparte memoria: %+v, %v", guardado, err)
	}
	representaciones, err := store.ListarRepresentacionesDocumento(context.Background(), resultado.Documento.Referencia())
	if err != nil || len(representaciones) != 2 || representaciones[0].NombreFichero == "alterado.docx" {
		t.Fatalf("representaciones guardadas = %+v, %v", representaciones, err)
	}

	conflicto := solicitud
	conflicto.HuellaSolicitudHMAC = huellaFuenteLogicaDistinta
	if _, err := store.ReservarGeneracion(context.Background(), conflicto); !errors.Is(err, ports.ErrClaveIdempotenciaReutilizada) {
		t.Fatalf("reutilizacion con otros datos: error = %v", err)
	}
	otroPrincipal := solicitud
	otroPrincipal.PrincipalID = "tecnico-rrhh-2"
	if otra, err := store.ReservarGeneracion(context.Background(), otroPrincipal); err != nil || otra.Token == "" {
		t.Fatalf("la clave no quedo aislada por principal: %+v, %v", otra, err)
	}

	auditoria, _ := store.ListAudit(context.Background(), resultado.Documento.ID+":1")
	eventos, _ := store.ListEvents(context.Background(), []string{"vec.documento.logico.generado"})
	if len(auditoria) != 1 || len(eventos) != 1 || eventos[0].Payload["auditoria_ref"] != auditoria[0].ID {
		t.Fatalf("evidencia atomica incompleta: auditoria=%+v eventos=%+v", auditoria, eventos)
	}
}

func TestRepositorioDocumentosLogicosSoloConcedeUnaReservaConcurrente(t *testing.T) {
	store := NewStore()
	fecha := time.Date(2026, time.July, 14, 18, 0, 0, 0, time.UTC)
	solicitud := solicitudReservaLogicaPrueba("tecnico-rrhh-1", fecha)
	const intentos = 32
	var grupo sync.WaitGroup
	grupo.Add(intentos)
	resultados := make(chan ports.ReservaGeneracionDocumento, intentos)
	errores := make(chan error, intentos)
	for i := 0; i < intentos; i++ {
		go func() {
			defer grupo.Done()
			reserva, err := store.ReservarGeneracion(context.Background(), solicitud)
			if err != nil {
				errores <- err
				return
			}
			resultados <- reserva
		}()
	}
	grupo.Wait()
	close(resultados)
	close(errores)
	var ganadora ports.ReservaGeneracionDocumento
	concedidas := 0
	for resultado := range resultados {
		ganadora = resultado
		concedidas++
	}
	enCurso := 0
	for err := range errores {
		if !errors.Is(err, ports.ErrGeneracionDocumentoEnCurso) {
			t.Fatalf("error concurrente inesperado: %v", err)
		}
		enCurso++
	}
	if concedidas != 1 || enCurso != intentos-1 {
		t.Fatalf("concedidas=%d en_curso=%d", concedidas, enCurso)
	}
	if err := store.AbandonarGeneracion(context.Background(), ganadora.Token); err != nil {
		t.Fatalf("AbandonarGeneracion() error = %v", err)
	}
	reintento, err := store.ReservarGeneracion(context.Background(), solicitud)
	if err != nil || reintento.Token == "" || reintento.Token == ganadora.Token {
		t.Fatalf("reintento tras abandono = %+v, %v", reintento, err)
	}
	conflicto := solicitud
	conflicto.HuellaSolicitudHMAC = huellaFuenteLogicaDistinta
	if _, err := store.ReservarGeneracion(context.Background(), conflicto); !errors.Is(err, ports.ErrClaveIdempotenciaReutilizada) {
		t.Fatalf("una clave abandonada cambio de significado: %v", err)
	}
}

func TestRepositorioDocumentosLogicosReemplazaReservaCaducadaEInvalidaTokenAnterior(t *testing.T) {
	store := NewStore()
	fecha := time.Date(2026, time.July, 14, 19, 0, 0, 0, time.UTC)
	primeraSolicitud := solicitudReservaLogicaPrueba("tecnico-rrhh-1", fecha)
	primeraSolicitud.ExpiraEn = fecha.Add(time.Minute)
	primera, err := store.ReservarGeneracion(context.Background(), primeraSolicitud)
	if err != nil {
		t.Fatalf("primera reserva: %v", err)
	}
	segundaSolicitud := primeraSolicitud
	segundaSolicitud.SolicitadaEn = fecha.Add(2 * time.Minute)
	segundaSolicitud.ExpiraEn = fecha.Add(7 * time.Minute)
	segunda, err := store.ReservarGeneracion(context.Background(), segundaSolicitud)
	if err != nil || segunda.Token == primera.Token {
		t.Fatalf("segunda reserva = %+v, %v", segunda, err)
	}
	resultado := resultadoDocumentoLogicoMemoriaPrueba(t, store, primeraSolicitud.PrincipalID, segundaSolicitud.SolicitadaEn)
	traza, evento := evidenciaDocumentoLogicoMemoriaPrueba(resultado, segundaSolicitud.SolicitadaEn)
	if err := store.ConfirmarGeneracionLogica(context.Background(), primera.Token, primeraSolicitud.HuellaSolicitudHMAC, segundaSolicitud.SolicitadaEn, resultado, traza, evento); !errors.Is(err, ports.ErrReservaDocumentoNoValida) {
		t.Fatalf("token caducado sustituido: error = %v", err)
	}
	if err := store.ConfirmarGeneracionLogica(context.Background(), segunda.Token, segundaSolicitud.HuellaSolicitudHMAC, segundaSolicitud.SolicitadaEn, resultado, traza, evento); err != nil {
		t.Fatalf("token vigente: error = %v", err)
	}
}

func TestRepositorioDocumentosLogicosNoConfirmaReservaCaducadaSinSustituta(t *testing.T) {
	fecha := time.Date(2026, time.July, 14, 19, 30, 0, 0, time.UTC)
	casos := []struct {
		nombre         string
		desplazamiento time.Duration
	}{
		{nombre: "en el limite exacto", desplazamiento: 0},
		{nombre: "despues del limite", desplazamiento: time.Second},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			store := NewStore()
			solicitud := solicitudReservaLogicaPrueba("tecnico-rrhh-1", fecha)
			solicitud.ExpiraEn = fecha.Add(time.Minute)
			reserva, err := store.ReservarGeneracion(context.Background(), solicitud)
			if err != nil {
				t.Fatalf("reservar: %v", err)
			}

			confirmadaEn := solicitud.ExpiraEn.Add(caso.desplazamiento)
			resultado := resultadoDocumentoLogicoMemoriaPrueba(t, store, solicitud.PrincipalID, confirmadaEn)
			traza, evento := evidenciaDocumentoLogicoMemoriaPrueba(resultado, confirmadaEn)
			err = store.ConfirmarGeneracionLogica(
				context.Background(), reserva.Token, solicitud.HuellaSolicitudHMAC,
				confirmadaEn, resultado, traza, evento,
			)
			if !errors.Is(err, ports.ErrReservaDocumentoNoValida) {
				t.Fatalf("confirmacion caducada en %s: error = %v", confirmadaEn, err)
			}
			if _, err := store.ObtenerDocumentoLogico(context.Background(), resultado.Documento.Referencia()); !errors.Is(err, ports.ErrDocumentoLogicoNoEncontrado) {
				t.Fatalf("la confirmacion caducada persistio el documento: %v", err)
			}
			auditoria, _ := store.ListAudit(context.Background(), resultado.Documento.ID+":1")
			eventos, _ := store.ListEvents(context.Background(), []string{"vec.documento.logico.generado"})
			if len(auditoria) != 0 || len(eventos) != 0 {
				t.Fatalf("la confirmacion caducada persistio evidencia: %+v %+v", auditoria, eventos)
			}
		})
	}
}

func TestRepositorioDocumentosLogicosNoPersisteConfirmacionParcial(t *testing.T) {
	store := NewStore()
	fecha := time.Date(2026, time.July, 14, 20, 0, 0, 0, time.UTC)
	solicitud := solicitudReservaLogicaPrueba("tecnico-rrhh-1", fecha.Add(-time.Minute))
	reserva, err := store.ReservarGeneracion(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("reserva: %v", err)
	}
	resultado := resultadoDocumentoLogicoMemoriaPrueba(t, store, solicitud.PrincipalID, fecha)
	traza, evento := evidenciaDocumentoLogicoMemoriaPrueba(resultado, fecha)
	traza.ModuleID = "personal"
	if err := store.ConfirmarGeneracionLogica(context.Background(), reserva.Token, solicitud.HuellaSolicitudHMAC, fecha, resultado, traza, evento); !errors.Is(err, domain.ErrDocumentoLogicoInvalido) {
		t.Fatalf("evidencia falsa: error = %v", err)
	}
	if _, err := store.ObtenerDocumentoLogico(context.Background(), resultado.Documento.Referencia()); !errors.Is(err, ports.ErrDocumentoLogicoNoEncontrado) {
		t.Fatalf("la confirmacion parcial persistio documento: %v", err)
	}
	auditoria, _ := store.ListAudit(context.Background(), resultado.Documento.ID+":1")
	eventos, _ := store.ListEvents(context.Background(), []string{"vec.documento.logico.generado"})
	if len(auditoria) != 0 || len(eventos) != 0 {
		t.Fatalf("la confirmacion parcial persistio evidencia: %+v %+v", auditoria, eventos)
	}
}
