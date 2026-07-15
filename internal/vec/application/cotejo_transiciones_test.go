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
	transicionesCotejoPruebaSecreto             = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	transicionesCotejoPruebaIndice              = "hmac-sha256:indice_cotejo_transiciones:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	transicionesCotejoPruebaIndiceSustituto     = "hmac-sha256:indice_cotejo_transiciones:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	transicionesCotejoPruebaProteccion          = "proteccion-cotejo-transiciones-001"
	transicionesCotejoPruebaProteccionSustituto = "proteccion-cotejo-transiciones-002"
	transicionesCotejoPruebaExpediente          = "expediente-cotejo-transiciones-001"
	transicionesCotejoPruebaCodigoID            = "codigo-cotejo-transiciones-001"
	transicionesCotejoPruebaSustitutoID         = "codigo-cotejo-transiciones-002"
	transicionesCotejoPruebaRepresentacionID    = "representacion-cotejo-transiciones-001"
)

func TestTransicionesCotejoPruebaActivacionCotejaDocumentoRepresentacionYEvidenciaExactos(t *testing.T) {
	entorno := transicionesCotejoPruebaNuevoEntorno(t)
	anterior := entorno.codigos.codigos[entorno.codigo.ID]
	huellaAnterior, err := anterior.HuellaEstadoSHA256()
	if err != nil {
		t.Fatalf("calcular huella anterior: %v", err)
	}

	resultado, err := entorno.servicio.ActivarCodigoCotejo(context.Background(), transicionesCotejoPruebaOrdenActivar(entorno))
	if err != nil {
		t.Fatalf("activar codigo de cotejo: %v", err)
	}
	if resultado.Estado != domain.EstadoCodigoCotejoActivo || resultado.Revision != anterior.Revision+1 {
		t.Fatalf("estado/revision activados = %s/%d", resultado.Estado, resultado.Revision)
	}
	if resultado.VersionEmitida == nil || !reflect.DeepEqual(*resultado.VersionEmitida, entorno.evidencia.VersionEmitida) {
		t.Fatalf("version activada = %+v; evidencia exacta = %+v", resultado.VersionEmitida, entorno.evidencia.VersionEmitida)
	}
	if resultado.VersionEmitida.RepresentacionID != entorno.representacion.ID ||
		resultado.VersionEmitida.ReferenciaContenido != entorno.representacion.ReferenciaContenido ||
		resultado.VersionEmitida.HuellaContenidoSHA256 != entorno.representacion.HuellaContenidoSHA256 ||
		resultado.VersionEmitida.MIME != entorno.representacion.MIME ||
		resultado.VersionEmitida.Tamano != entorno.representacion.Tamano {
		t.Fatalf("la version activada no fija los bytes de la representacion: %+v", resultado.VersionEmitida)
	}

	if !reflect.DeepEqual(entorno.documentos.consultas, []domain.ReferenciaDocumento{entorno.documento.Referencia()}) ||
		!reflect.DeepEqual(entorno.documentos.listados, []domain.ReferenciaDocumento{entorno.documento.Referencia()}) {
		t.Fatalf("consultas documentales no exactas: documentos=%+v representaciones=%+v",
			entorno.documentos.consultas, entorno.documentos.listados)
	}
	if len(entorno.evidencias.solicitudes) != 1 {
		t.Fatalf("solicitudes de evidencia = %d; esperada 1", len(entorno.evidencias.solicitudes))
	}
	solicitudEvidencia := entorno.evidencias.solicitudes[0]
	if solicitudEvidencia.Documento != entorno.documento.Referencia() ||
		solicitudEvidencia.RepresentacionID != entorno.representacion.ID ||
		solicitudEvidencia.SolicitanteID != entorno.principal.ID ||
		solicitudEvidencia.AutorizacionRef != "decision-cotejo-transiciones-001" ||
		solicitudEvidencia.Finalidad != "emitir_documento_verificable" ||
		solicitudEvidencia.CorrelacionRef != "correlacion-activar-cotejo-transiciones-001" {
		t.Fatalf("solicitud de evidencia no exacta: %+v", solicitudEvidencia)
	}

	if len(entorno.autorizador.solicitudes) != 1 {
		t.Fatalf("solicitudes de autorizacion = %d; esperada 1", len(entorno.autorizador.solicitudes))
	}
	solicitudAutorizacion := entorno.autorizador.solicitudes[0]
	recursoEsperado := domain.RecursoAutorizable{
		Referencia: anterior.Referencia(),
		ModuloID:   anterior.ModuloID,
		Tipo:       "codigo_cotejo",
		Ambitos: map[string]string{
			"expediente":    transicionesCotejoPruebaExpediente,
			"clasificacion": anterior.Clasificacion,
		},
		Atributos: map[string]string{
			"documento_ref":   entorno.documento.ID + ":1",
			"tipo_documental": anterior.TipoDocumental,
			"estado":          string(domain.EstadoCodigoCotejoReservado),
		},
	}
	if solicitudAutorizacion.Accion != AccionActivarCodigoCotejo ||
		!reflect.DeepEqual(solicitudAutorizacion.Recurso, recursoEsperado) {
		t.Fatalf("autorizacion de activacion incorrecta: %+v", solicitudAutorizacion)
	}

	confirmacion := transicionesCotejoPruebaUnicaConfirmacion(t, entorno.codigos, "activacion")
	huellaNueva, err := resultado.HuellaEstadoSHA256()
	if err != nil {
		t.Fatalf("calcular huella nueva: %v", err)
	}
	if confirmacion.huellaAnterior != huellaAnterior || !reflect.DeepEqual(confirmacion.codigo, resultado) ||
		confirmacion.traza.Action != eventoCodigoCotejoActivado ||
		confirmacion.traza.BeforeHash != huellaAnterior || confirmacion.traza.AfterHash != huellaNueva ||
		confirmacion.traza.RuleRef != entorno.evidencia.EvidenciaRef ||
		confirmacion.traza.Metadata["representacion_id"] != entorno.representacion.ID ||
		confirmacion.traza.Metadata["firmas"] != "1" || confirmacion.traza.Metadata["sellos_tiempo"] != "1" ||
		confirmacion.traza.Metadata["registro"] != "true" ||
		confirmacion.evento.Type != eventoCodigoCotejoActivado ||
		confirmacion.evento.Payload["representacion_id"] != entorno.representacion.ID ||
		confirmacion.evento.Payload["huella_estado"] != huellaNueva {
		t.Fatalf("auditoria/outbox de activacion incorrectos: %+v / %+v", confirmacion.traza, confirmacion.evento)
	}
	transicionesCotejoPruebaExigirEvidenciasSinMaterialSensible(t, confirmacion)
}

func TestTransicionesCotejoPruebaActivacionRechazaDivergenciasYEstadosInseguros(t *testing.T) {
	casos := []struct {
		nombre   string
		preparar func(*transicionesCotejoPruebaEntorno)
		err      error
	}{
		{
			nombre: "huella divergente",
			preparar: func(entorno *transicionesCotejoPruebaEntorno) {
				evidencia := entorno.evidencia
				evidencia.VersionEmitida.HuellaContenidoSHA256 = strings.Repeat("8", 64)
				entorno.evidencias.evidencias[transicionesCotejoPruebaClaveEvidencia{
					documento: entorno.documento.Referencia(), representacionID: entorno.representacion.ID,
				}] = evidencia
			},
			err: domain.ErrEvidenciaEmisionInvalida,
		},
		{
			nombre: "referencia de contenido divergente",
			preparar: func(entorno *transicionesCotejoPruebaEntorno) {
				evidencia := entorno.evidencia
				evidencia.VersionEmitida.ReferenciaContenido = "almacen:cotejo-transiciones:contenido-divergente"
				entorno.evidencias.evidencias[transicionesCotejoPruebaClaveEvidencia{
					documento: entorno.documento.Referencia(), representacionID: entorno.representacion.ID,
				}] = evidencia
			},
			err: domain.ErrEvidenciaEmisionInvalida,
		},
		{
			nombre: "tamano divergente",
			preparar: func(entorno *transicionesCotejoPruebaEntorno) {
				evidencia := entorno.evidencia
				evidencia.VersionEmitida.Tamano++
				entorno.evidencias.evidencias[transicionesCotejoPruebaClaveEvidencia{
					documento: entorno.documento.Referencia(), representacionID: entorno.representacion.ID,
				}] = evidencia
			},
			err: domain.ErrEvidenciaEmisionInvalida,
		},
		{
			nombre: "antivirus no limpio",
			preparar: func(entorno *transicionesCotejoPruebaEntorno) {
				representaciones := entorno.documentos.representaciones[entorno.documento.Referencia()]
				representaciones[0].EstadoAntivirus = domain.EstadoAntivirusPendiente
				entorno.documentos.representaciones[entorno.documento.Referencia()] = representaciones
			},
			err: domain.ErrEvidenciaEmisionInvalida,
		},
		{
			nombre: "estado documental incompatible",
			preparar: func(entorno *transicionesCotejoPruebaEntorno) {
				documento := entorno.documentos.documentos[entorno.documento.Referencia()]
				documento.Estado = domain.EstadoDocumentoLogicoFirmado
				entorno.documentos.documentos[entorno.documento.Referencia()] = documento
			},
			err: domain.ErrTransicionCodigoCotejo,
		},
		{
			nombre: "reserva caducada",
			preparar: func(entorno *transicionesCotejoPruebaEntorno) {
				codigo := entorno.codigos.codigos[entorno.codigo.ID]
				codigo.ReservaExpiraEn = entorno.ahora.Add(-time.Second)
				entorno.codigos.codigos[codigo.ID] = codigo
			},
			err: domain.ErrTransicionCodigoCotejo,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := transicionesCotejoPruebaNuevoEntorno(t)
			caso.preparar(entorno)

			resultado, err := entorno.servicio.ActivarCodigoCotejo(context.Background(), transicionesCotejoPruebaOrdenActivar(entorno))
			if !errors.Is(err, caso.err) {
				t.Fatalf("error = %v; esperado %v", err, caso.err)
			}
			if resultado.ID != "" || len(entorno.codigos.confirmaciones) != 0 {
				t.Fatalf("el rechazo devolvio o confirmo una activacion: %+v / %+v", resultado, entorno.codigos.confirmaciones)
			}
		})
	}
}

func TestTransicionesCotejoPruebaRetiradaAutorizadaConfirmaAuditoriaYEvento(t *testing.T) {
	entorno := transicionesCotejoPruebaNuevoEntorno(t)
	activo := transicionesCotejoPruebaActivarFixture(t, entorno.codigo, entorno.evidencia, entorno.ahora.Add(-time.Minute))
	entorno.codigos.codigos[activo.ID] = activo
	huellaAnterior, err := activo.HuellaEstadoSHA256()
	if err != nil {
		t.Fatalf("calcular huella anterior: %v", err)
	}
	orden := OrdenRetirarCodigoCotejo{
		Principal:       entorno.principal,
		PerfilActivo:    perfilAutorizacionPrueba("cotejo-transiciones-001"),
		RepresentadoRef: "persona-representada-cotejo-transiciones-001",
		Finalidad:       "retirar_documento_verificable",
		CodigoID:        activo.ID,
		RetiradaRef:     "acuerdo-retirada-cotejo-transiciones-001",
		Motivo:          "retirada autorizada del codigo de cotejo",
		CorrelacionRef:  "correlacion-retirar-cotejo-transiciones-001",
	}

	resultado, err := entorno.servicio.RetirarCodigoCotejo(context.Background(), orden)
	if err != nil {
		t.Fatalf("retirar codigo de cotejo: %v", err)
	}
	if resultado.Estado != domain.EstadoCodigoCotejoRetirado || resultado.RetiradaRef != orden.RetiradaRef ||
		resultado.RetiradoPor != entorno.principal.ID || !resultado.RetiradoEn.Equal(entorno.ahora) {
		t.Fatalf("retirada incorrecta: %+v", resultado)
	}
	if len(entorno.autorizador.solicitudes) != 1 ||
		entorno.autorizador.solicitudes[0].Accion != AccionRetirarCodigoCotejo ||
		entorno.autorizador.solicitudes[0].Recurso.Atributos["estado"] != string(domain.EstadoCodigoCotejoActivo) {
		t.Fatalf("retirada no autorizada contra el recurso activo: %+v", entorno.autorizador.solicitudes)
	}

	confirmacion := transicionesCotejoPruebaUnicaConfirmacion(t, entorno.codigos, "retirada")
	huellaNueva, err := resultado.HuellaEstadoSHA256()
	if err != nil {
		t.Fatalf("calcular huella nueva: %v", err)
	}
	if confirmacion.huellaAnterior != huellaAnterior || confirmacion.traza.Action != eventoCodigoCotejoRetirado ||
		confirmacion.traza.RuleRef != orden.RetiradaRef || confirmacion.traza.BeforeHash != huellaAnterior ||
		confirmacion.traza.AfterHash != huellaNueva || confirmacion.evento.Type != eventoCodigoCotejoRetirado ||
		confirmacion.evento.Payload["estado"] != string(domain.EstadoCodigoCotejoRetirado) ||
		confirmacion.evento.Payload["huella_estado"] != huellaNueva {
		t.Fatalf("auditoria/outbox de retirada incorrectos: %+v / %+v", confirmacion.traza, confirmacion.evento)
	}
	transicionesCotejoPruebaExigirEvidenciasSinMaterialSensible(t, confirmacion)
}

func TestTransicionesCotejoPruebaSustitucionAutorizadaExigeMismoAmbitoYOtroDocumento(t *testing.T) {
	entorno := transicionesCotejoPruebaNuevoEntorno(t)
	anterior, sustituto := transicionesCotejoPruebaPrepararSustitucion(t, entorno)
	huellaAnterior, err := anterior.HuellaEstadoSHA256()
	if err != nil {
		t.Fatalf("calcular huella anterior: %v", err)
	}
	orden := transicionesCotejoPruebaOrdenSustituir(entorno, anterior, sustituto)

	resultado, err := entorno.servicio.SustituirCodigoCotejo(context.Background(), orden)
	if err != nil {
		t.Fatalf("sustituir codigo de cotejo: %v", err)
	}
	if resultado.Estado != domain.EstadoCodigoCotejoSustituido || resultado.SustituidoPorRef != sustituto.Referencia() ||
		resultado.Documento == sustituto.Documento || !sustituto.DisponibleEn(entorno.ahora) {
		t.Fatalf("sustitucion incorrecta: resultado=%+v sustituto=%+v", resultado, sustituto)
	}
	if len(entorno.autorizador.solicitudes) != 1 ||
		entorno.autorizador.solicitudes[0].Accion != AccionSustituirCodigoCotejo ||
		entorno.autorizador.solicitudes[0].Recurso.Atributos["sustituto_ref"] != sustituto.Referencia() {
		t.Fatalf("autorizacion de sustitucion incorrecta: %+v", entorno.autorizador.solicitudes)
	}
	if persistido := entorno.codigos.codigos[sustituto.ID]; persistido.Estado != domain.EstadoCodigoCotejoActivo {
		t.Fatalf("la sustitucion altero el codigo sustituto: %+v", persistido)
	}

	confirmacion := transicionesCotejoPruebaUnicaConfirmacion(t, entorno.codigos, "sustitucion")
	huellaNueva, err := resultado.HuellaEstadoSHA256()
	if err != nil {
		t.Fatalf("calcular huella nueva: %v", err)
	}
	if confirmacion.huellaAnterior != huellaAnterior || confirmacion.traza.Action != eventoCodigoCotejoSustituido ||
		confirmacion.traza.RuleRef != orden.SustitucionRef ||
		confirmacion.traza.Metadata["sustituto_ref"] != sustituto.Referencia() ||
		confirmacion.traza.BeforeHash != huellaAnterior || confirmacion.traza.AfterHash != huellaNueva ||
		confirmacion.evento.Type != eventoCodigoCotejoSustituido ||
		confirmacion.evento.Payload["sustituto_ref"] != sustituto.Referencia() ||
		confirmacion.evento.Payload["huella_estado"] != huellaNueva {
		t.Fatalf("auditoria/outbox de sustitucion incorrectos: %+v / %+v", confirmacion.traza, confirmacion.evento)
	}
	transicionesCotejoPruebaExigirEvidenciasSinMaterialSensible(t, confirmacion)
}

func TestTransicionesCotejoPruebaSustitucionRechazaSustitutoOAmbitoDivergente(t *testing.T) {
	casos := []struct {
		nombre   string
		preparar func(*testing.T, *transicionesCotejoPruebaEntorno, domain.CodigoCotejo)
	}{
		{
			nombre: "sustituto no activo",
			preparar: func(t *testing.T, entorno *transicionesCotejoPruebaEntorno, sustituto domain.CodigoCotejo) {
				documento := entorno.documentos.documentos[sustituto.Documento]
				reservado := transicionesCotejoPruebaCodigoReservado(
					t, sustituto.ID, documento, transicionesCotejoPruebaIndiceSustituto,
					transicionesCotejoPruebaProteccionSustituto, entorno.ahora,
				)
				entorno.codigos.codigos[sustituto.ID] = reservado
			},
		},
		{
			nombre: "sustituto no disponible",
			preparar: func(_ *testing.T, entorno *transicionesCotejoPruebaEntorno, sustituto domain.CodigoCotejo) {
				sustituto.DisponibleHasta = entorno.ahora
				entorno.codigos.codigos[sustituto.ID] = sustituto
			},
		},
		{
			nombre: "mismo documento",
			preparar: func(_ *testing.T, entorno *transicionesCotejoPruebaEntorno, sustituto domain.CodigoCotejo) {
				sustituto.Documento = entorno.documento.Referencia()
				entorno.codigos.codigos[sustituto.ID] = sustituto
			},
		},
		{
			nombre: "otro expediente",
			preparar: func(_ *testing.T, entorno *transicionesCotejoPruebaEntorno, sustituto domain.CodigoCotejo) {
				sustituto.ExpedienteRef = "expediente-cotejo-transiciones-divergente"
				documento := entorno.documentos.documentos[sustituto.Documento]
				documento.Relaciones[0].Referencia = sustituto.ExpedienteRef
				entorno.documentos.documentos[sustituto.Documento] = documento
				entorno.codigos.codigos[sustituto.ID] = sustituto
			},
		},
		{
			nombre: "otro tipo documental",
			preparar: func(_ *testing.T, entorno *transicionesCotejoPruebaEntorno, sustituto domain.CodigoCotejo) {
				sustituto.TipoDocumental = "resolucion"
				documento := entorno.documentos.documentos[sustituto.Documento]
				documento.TipoDocumental = sustituto.TipoDocumental
				documento.ENI.TipoDocumental = sustituto.TipoDocumental
				entorno.documentos.documentos[sustituto.Documento] = documento
				entorno.codigos.codigos[sustituto.ID] = sustituto
			},
		},
		{
			nombre: "otra clasificacion",
			preparar: func(_ *testing.T, entorno *transicionesCotejoPruebaEntorno, sustituto domain.CodigoCotejo) {
				sustituto.Clasificacion = "interno"
				documento := entorno.documentos.documentos[sustituto.Documento]
				documento.Clasificacion = sustituto.Clasificacion
				entorno.documentos.documentos[sustituto.Documento] = documento
				entorno.codigos.codigos[sustituto.ID] = sustituto
			},
		},
		{
			nombre: "otro organo",
			preparar: func(_ *testing.T, entorno *transicionesCotejoPruebaEntorno, sustituto domain.CodigoCotejo) {
				sustituto.Organo = "L02000000"
				documento := entorno.documentos.documentos[sustituto.Documento]
				documento.ENI.Organo = sustituto.Organo
				entorno.documentos.documentos[sustituto.Documento] = documento
				entorno.codigos.codigos[sustituto.ID] = sustituto
			},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := transicionesCotejoPruebaNuevoEntorno(t)
			anterior, sustituto := transicionesCotejoPruebaPrepararSustitucion(t, entorno)
			caso.preparar(t, entorno, sustituto)

			resultado, err := entorno.servicio.SustituirCodigoCotejo(
				context.Background(), transicionesCotejoPruebaOrdenSustituir(entorno, anterior, sustituto),
			)
			if !errors.Is(err, domain.ErrTransicionCodigoCotejo) {
				t.Fatalf("error = %v; esperado %v", err, domain.ErrTransicionCodigoCotejo)
			}
			if resultado.ID != "" || len(entorno.codigos.confirmaciones) != 0 {
				t.Fatalf("el rechazo devolvio o confirmo una sustitucion: %+v / %+v", resultado, entorno.codigos.confirmaciones)
			}
		})
	}
}

type transicionesCotejoPruebaReloj struct {
	ahora time.Time
}

func (r *transicionesCotejoPruebaReloj) Ahora() time.Time { return r.ahora }

type transicionesCotejoPruebaAutorizador struct {
	reloj       *transicionesCotejoPruebaReloj
	solicitudes []domain.SolicitudAutorizacion
}

func (a *transicionesCotejoPruebaAutorizador) Exigir(
	_ context.Context,
	solicitud domain.SolicitudAutorizacion,
) (domain.DecisionAutorizacion, error) {
	a.solicitudes = append(a.solicitudes, solicitud)
	ahora := a.reloj.Ahora().UTC()
	return completarDecisionAutorizacionPrueba(solicitud, domain.DecisionAutorizacion{
		DecisionRef:            "decision-cotejo-transiciones-001",
		Concedida:              true,
		Codigo:                 "concedida",
		PrincipalID:            solicitud.Principal.ID,
		PerfilActivoRef:        solicitud.PerfilActivoRef,
		Accion:                 solicitud.Accion,
		RecursoRef:             solicitud.Recurso.Referencia,
		Finalidad:              solicitud.Finalidad,
		CorrelacionRef:         solicitud.CorrelacionRef,
		AsignacionRef:          "asignacion:cotejo-transiciones:v1",
		AsignacionHuellaSHA256: strings.Repeat("1", 64),
		VersionRolRef:          "rol:cotejo-transiciones:v1",
		VersionRolHuellaSHA256: strings.Repeat("2", 64),
		GarantiaMinima:         domain.AuthAssuranceHigh,
		EmitidaEn:              ahora.Add(-time.Minute),
		ValidaHasta:            ahora.Add(time.Minute),
	}), nil
}

type transicionesCotejoPruebaDocumentos struct {
	ports.RepositorioDocumentosLogicos
	documentos       map[domain.ReferenciaDocumento]domain.DocumentoLogico
	representaciones map[domain.ReferenciaDocumento][]domain.RepresentacionDocumento
	consultas        []domain.ReferenciaDocumento
	listados         []domain.ReferenciaDocumento
}

func (r *transicionesCotejoPruebaDocumentos) ObtenerDocumentoLogico(
	_ context.Context,
	referencia domain.ReferenciaDocumento,
) (domain.DocumentoLogico, error) {
	r.consultas = append(r.consultas, referencia)
	documento, existe := r.documentos[referencia]
	if !existe {
		return domain.DocumentoLogico{}, ports.ErrDocumentoLogicoNoEncontrado
	}
	return documento, nil
}

func (r *transicionesCotejoPruebaDocumentos) ListarRepresentacionesDocumento(
	_ context.Context,
	referencia domain.ReferenciaDocumento,
) ([]domain.RepresentacionDocumento, error) {
	r.listados = append(r.listados, referencia)
	representaciones, existe := r.representaciones[referencia]
	if !existe {
		return nil, ports.ErrRepresentacionNoEncontrada
	}
	return append([]domain.RepresentacionDocumento(nil), representaciones...), nil
}

type transicionesCotejoPruebaClaveEvidencia struct {
	documento        domain.ReferenciaDocumento
	representacionID string
}

type transicionesCotejoPruebaEvidencias struct {
	evidencias  map[transicionesCotejoPruebaClaveEvidencia]domain.EvidenciaEmisionDocumento
	solicitudes []ports.SolicitudObtenerEvidenciaEmisionDocumento
}

func (f *transicionesCotejoPruebaEvidencias) ObtenerEvidenciaEmisionDocumento(
	_ context.Context,
	solicitud ports.SolicitudObtenerEvidenciaEmisionDocumento,
) (domain.EvidenciaEmisionDocumento, error) {
	f.solicitudes = append(f.solicitudes, solicitud)
	evidencia, existe := f.evidencias[transicionesCotejoPruebaClaveEvidencia{
		documento: solicitud.Documento, representacionID: solicitud.RepresentacionID,
	}]
	if !existe {
		return domain.EvidenciaEmisionDocumento{}, ports.ErrEvidenciaEmisionNoEncontrada
	}
	return evidencia, nil
}

type transicionesCotejoPruebaConfirmacion struct {
	operacion      string
	huellaAnterior string
	codigo         domain.CodigoCotejo
	traza          domain.AuditEntry
	evento         domain.Event
}

type transicionesCotejoPruebaCodigos struct {
	ports.RepositorioCodigosCotejo
	codigos        map[string]domain.CodigoCotejo
	consultas      []string
	confirmaciones []transicionesCotejoPruebaConfirmacion
}

func (r *transicionesCotejoPruebaCodigos) ObtenerCodigoCotejo(
	_ context.Context,
	id string,
) (domain.CodigoCotejo, error) {
	r.consultas = append(r.consultas, id)
	codigo, existe := r.codigos[id]
	if !existe {
		return domain.CodigoCotejo{}, ports.ErrCodigoCotejoNoEncontrado
	}
	return codigo, nil
}

func (r *transicionesCotejoPruebaCodigos) ConfirmarActivacionCodigoCotejo(
	_ context.Context,
	huellaAnterior string,
	codigo domain.CodigoCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	return r.transicionesCotejoPruebaConfirmar("activacion", huellaAnterior, codigo, traza, evento)
}

func (r *transicionesCotejoPruebaCodigos) ConfirmarRetiradaCodigoCotejo(
	_ context.Context,
	huellaAnterior string,
	codigo domain.CodigoCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	return r.transicionesCotejoPruebaConfirmar("retirada", huellaAnterior, codigo, traza, evento)
}

func (r *transicionesCotejoPruebaCodigos) ConfirmarSustitucionCodigoCotejo(
	_ context.Context,
	huellaAnterior string,
	codigo domain.CodigoCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	return r.transicionesCotejoPruebaConfirmar("sustitucion", huellaAnterior, codigo, traza, evento)
}

func (r *transicionesCotejoPruebaCodigos) transicionesCotejoPruebaConfirmar(
	operacion, huellaAnterior string,
	codigo domain.CodigoCotejo,
	traza domain.AuditEntry,
	evento domain.Event,
) error {
	r.confirmaciones = append(r.confirmaciones, transicionesCotejoPruebaConfirmacion{
		operacion: operacion, huellaAnterior: huellaAnterior, codigo: codigo, traza: traza, evento: evento,
	})
	r.codigos[codigo.ID] = codigo
	return nil
}

type transicionesCotejoPruebaPuertosNoUsados struct {
	ports.CatalogoPoliticasCotejo
	ports.RepositorioGobiernoPoliticasCotejo
	ports.GeneradorValorCodigoCotejo
	ports.GeneradorIDCodigoCotejo
	ports.SelladorIndiceCodigoCotejo
	ports.SelladorSolicitudCotejo
	ports.ProtectorCodigoCotejo
}

type transicionesCotejoPruebaEntorno struct {
	servicio       *ServicioCotejo
	ahora          time.Time
	principal      domain.Principal
	documento      domain.DocumentoLogico
	representacion domain.RepresentacionDocumento
	evidencia      domain.EvidenciaEmisionDocumento
	codigo         domain.CodigoCotejo
	codigos        *transicionesCotejoPruebaCodigos
	documentos     *transicionesCotejoPruebaDocumentos
	autorizador    *transicionesCotejoPruebaAutorizador
	evidencias     *transicionesCotejoPruebaEvidencias
}

func transicionesCotejoPruebaNuevoEntorno(t *testing.T) *transicionesCotejoPruebaEntorno {
	t.Helper()
	ahora := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	documento := transicionesCotejoPruebaDocumento(
		"documento-cotejo-transiciones-001", transicionesCotejoPruebaExpediente,
		"certificado", "restringido", "L01000000", ahora,
	)
	representacion := transicionesCotejoPruebaRepresentacion(
		documento, transicionesCotejoPruebaRepresentacionID, strings.Repeat("3", 64), ahora,
	)
	evidencia := transicionesCotejoPruebaEvidencia(documento, representacion, ahora)
	codigo := transicionesCotejoPruebaCodigoReservado(
		t, transicionesCotejoPruebaCodigoID, documento, transicionesCotejoPruebaIndice,
		transicionesCotejoPruebaProteccion, ahora,
	)
	transicionesCotejoPruebaValidarFixtures(t, documento, representacion, evidencia, codigo)

	reloj := &transicionesCotejoPruebaReloj{ahora: ahora}
	codigos := &transicionesCotejoPruebaCodigos{codigos: map[string]domain.CodigoCotejo{codigo.ID: codigo}}
	documentos := &transicionesCotejoPruebaDocumentos{
		documentos:       map[domain.ReferenciaDocumento]domain.DocumentoLogico{documento.Referencia(): documento},
		representaciones: map[domain.ReferenciaDocumento][]domain.RepresentacionDocumento{documento.Referencia(): {representacion}},
	}
	autorizador := &transicionesCotejoPruebaAutorizador{reloj: reloj}
	evidencias := &transicionesCotejoPruebaEvidencias{
		evidencias: map[transicionesCotejoPruebaClaveEvidencia]domain.EvidenciaEmisionDocumento{
			{documento: documento.Referencia(), representacionID: representacion.ID}: evidencia,
		},
	}
	noUsados := &transicionesCotejoPruebaPuertosNoUsados{}
	servicio, err := NuevoServicioCotejo(
		noUsados, noUsados, codigos, documentos, autorizador, noUsados, noUsados,
		noUsados, noUsados, noUsados, evidencias, reloj,
	)
	if err != nil {
		t.Fatalf("crear servicio de cotejo: %v", err)
	}
	return &transicionesCotejoPruebaEntorno{
		servicio: servicio, ahora: ahora, principal: transicionesCotejoPruebaPrincipal(),
		documento: documento, representacion: representacion, evidencia: evidencia, codigo: codigo,
		codigos: codigos, documentos: documentos, autorizador: autorizador, evidencias: evidencias,
	}
}

func transicionesCotejoPruebaDocumento(
	id, expedienteRef, tipoDocumental, clasificacion, organo string,
	ahora time.Time,
) domain.DocumentoLogico {
	return domain.DocumentoLogico{
		ID:       id,
		Version:  1,
		Revision: 1,
		Plantilla: domain.ReferenciaPlantillaDocumento{
			ID: "certificado_cotejo_transiciones", Version: 1, HuellaSHA256: strings.Repeat("4", 64),
		},
		ModuloID:       "bolsa",
		TipoDocumental: tipoDocumental,
		Clasificacion:  clasificacion,
		Relaciones: []domain.RelacionDocumento{{
			Tipo: domain.TipoRelacionExpediente, Referencia: expedienteRef, Rol: "principal",
		}},
		Estado:           domain.EstadoDocumentoLogicoRegistrado,
		HuellaDatosHMAC:  "hmac-sha256:datos_cotejo_transiciones:" + strings.Repeat("5", 64),
		HuellaFuenteHMAC: "hmac-sha256:fuente_cotejo_transiciones:" + strings.Repeat("6", 64),
		CreadoPor:        "tecnico-rrhh-cotejo-transiciones-001",
		CreadoEn:         ahora.Add(-4 * time.Hour),
		CorrelacionRef:   "correlacion-documento-cotejo-transiciones-001",
		Motivo:           "generacion de certificado verificable para la bolsa",
		ENI: domain.MetadatosENI{
			Identificador:     "ES_L01000000_2026_DOC_COTEJO_TRANSICIONES",
			Organo:            organo,
			Origen:            "administracion",
			EstadoElaboracion: "original",
			TipoDocumental:    tipoDocumental,
			FechaCaptura:      ahora.Add(-4 * time.Hour),
		},
	}
}

func transicionesCotejoPruebaRepresentacion(
	documento domain.DocumentoLogico,
	id, huella string,
	ahora time.Time,
) domain.RepresentacionDocumento {
	return domain.RepresentacionDocumento{
		ID:                    id,
		Documento:             documento.Referencia(),
		Tipo:                  domain.TipoRepresentacionFirma,
		Formato:               domain.FormatoDocumentoPDF,
		MIME:                  domain.FormatoDocumentoPDF.MIME(),
		NombreFichero:         id + ".pdf",
		Tamano:                12_345,
		HuellaContenidoSHA256: huella,
		HuellaFuenteHMAC:      documento.HuellaFuenteHMAC,
		ReferenciaContenido:   "almacen:cotejo-transiciones:" + id,
		EstadoTecnico:         domain.EstadoRepresentacionDisponible,
		EstadoAntivirus:       domain.EstadoAntivirusLimpio,
		GeneradaPor:           "servicio-firma-cotejo-transiciones-001",
		GeneradaEn:            ahora.Add(-10 * time.Minute),
		DerivadaDeRef:         "representacion-origen-" + id,
	}
}

func transicionesCotejoPruebaEvidencia(
	documento domain.DocumentoLogico,
	representacion domain.RepresentacionDocumento,
	ahora time.Time,
) domain.EvidenciaEmisionDocumento {
	return domain.EvidenciaEmisionDocumento{
		Documento: documento.Referencia(),
		VersionEmitida: domain.VersionEmitidaCotejo{
			RepresentacionID:      representacion.ID,
			ReferenciaContenido:   representacion.ReferenciaContenido,
			HuellaContenidoSHA256: representacion.HuellaContenidoSHA256,
			MIME:                  representacion.MIME,
			Tamano:                representacion.Tamano,
			FirmaRefs:             []string{"firma-cotejo-transiciones-001"},
			SelloTiempoRefs:       []string{"sello-tiempo-cotejo-transiciones-001"},
			ValidacionFirmaRef:    "validacion-firma-cotejo-transiciones-001",
			RegistroRef:           "registro-cotejo-transiciones-001",
			EmitidaEn:             ahora.Add(-5 * time.Minute),
		},
		Apta:         true,
		EvidenciaRef: "evidencia-emision-cotejo-transiciones-001",
	}
}

func transicionesCotejoPruebaCodigoReservado(
	t *testing.T,
	id string,
	documento domain.DocumentoLogico,
	indice, proteccion string,
	ahora time.Time,
) domain.CodigoCotejo {
	t.Helper()
	codigo := domain.CodigoCotejo{
		ID:               id,
		Revision:         1,
		Documento:        documento.Referencia(),
		ModuloID:         documento.ModuloID,
		TipoDocumental:   documento.TipoDocumental,
		Clasificacion:    documento.Clasificacion,
		Organo:           documento.ENI.Organo,
		ExpedienteRef:    documento.Relaciones[0].Referencia,
		IndiceCodigoHMAC: indice,
		ProteccionRef:    proteccion,
		VersionGenerador: "generador-cotejo-transiciones-v1",
		EntropiaBits:     160,
		Politica: domain.AplicacionPoliticaCotejo{
			Referencia: domain.ReferenciaPoliticaCotejo{
				ID: "politica_cotejo_transiciones", Version: 1, HuellaSHA256: strings.Repeat("7", 64),
			},
			ClaseAcceso:              domain.ClaseAccesoCotejoProtegido,
			CamposPublicos:           []domain.CampoPublicoCotejo{domain.CampoPublicoCotejoOrgano},
			PermiteDescargaDocumento: true,
			RequiereFirma:            true,
			RequiereSelloTiempo:      true,
			RequiereRegistro:         true,
			GarantiaMinima:           domain.AuthAssuranceHigh,
			DiasPlazoActivacion:      5,
			DiasDisponibilidad:       30,
		},
		Estado:          domain.EstadoCodigoCotejoReservado,
		ReservadoPor:    personaAutorizacionPrueba("tecnico-rrhh-cotejo-transiciones-001"),
		ReservadoEn:     ahora.Add(-2 * time.Hour),
		ReservaExpiraEn: ahora.Add(10 * time.Minute),
		MotivoReserva:   "reserva para certificado verificable",
		CorrelacionRef:  "correlacion-reserva-cotejo-transiciones-001",
	}
	if err := codigo.Validar(); err != nil {
		t.Fatalf("codigo reservado de prueba invalido: %v", err)
	}
	return codigo
}

func transicionesCotejoPruebaPrincipal() domain.Principal {
	return domain.Principal{
		ID:            personaAutorizacionPrueba("tecnico-rrhh-cotejo-transiciones-001"),
		DisplayName:   "Tecnico RRHH",
		Roles:         []string{"tecnico_rrhh"},
		AuthMethod:    domain.AuthMethodCertificate,
		AuthAssurance: domain.AuthAssuranceHigh,
	}
}

func transicionesCotejoPruebaOrdenActivar(entorno *transicionesCotejoPruebaEntorno) OrdenActivarCodigoCotejo {
	return OrdenActivarCodigoCotejo{
		Principal:        entorno.principal,
		PerfilActivo:     perfilAutorizacionPrueba("cotejo-transiciones-001"),
		RepresentadoRef:  "persona-representada-cotejo-transiciones-001",
		Finalidad:        "emitir_documento_verificable",
		CodigoID:         entorno.codigo.ID,
		RepresentacionID: entorno.representacion.ID,
		ActivacionRef:    "activacion-cotejo-transiciones-001",
		Motivo:           "activar codigo tras cotejar la evidencia interna",
		CorrelacionRef:   "correlacion-activar-cotejo-transiciones-001",
	}
}

func transicionesCotejoPruebaActivarFixture(
	t *testing.T,
	codigo domain.CodigoCotejo,
	evidencia domain.EvidenciaEmisionDocumento,
	instante time.Time,
) domain.CodigoCotejo {
	t.Helper()
	activo, err := codigo.Activar(
		"tecnico-registro-cotejo-transiciones-001", "activacion-fixture-"+codigo.ID,
		"activacion previa valida para probar la transicion", evidencia, instante,
	)
	if err != nil {
		t.Fatalf("activar fixture de cotejo: %v", err)
	}
	return activo
}

func transicionesCotejoPruebaPrepararSustitucion(
	t *testing.T,
	entorno *transicionesCotejoPruebaEntorno,
) (domain.CodigoCotejo, domain.CodigoCotejo) {
	t.Helper()
	anterior := transicionesCotejoPruebaActivarFixture(t, entorno.codigo, entorno.evidencia, entorno.ahora.Add(-time.Minute))
	entorno.codigos.codigos[anterior.ID] = anterior

	documento := transicionesCotejoPruebaDocumento(
		"documento-cotejo-transiciones-002", transicionesCotejoPruebaExpediente,
		anterior.TipoDocumental, anterior.Clasificacion, anterior.Organo, entorno.ahora,
	)
	representacion := transicionesCotejoPruebaRepresentacion(
		documento, "representacion-cotejo-transiciones-002", strings.Repeat("9", 64), entorno.ahora,
	)
	evidencia := transicionesCotejoPruebaEvidencia(documento, representacion, entorno.ahora)
	evidencia.EvidenciaRef = "evidencia-emision-cotejo-transiciones-002"
	evidencia.VersionEmitida.FirmaRefs = []string{"firma-cotejo-transiciones-002"}
	evidencia.VersionEmitida.SelloTiempoRefs = []string{"sello-tiempo-cotejo-transiciones-002"}
	evidencia.VersionEmitida.ValidacionFirmaRef = "validacion-firma-cotejo-transiciones-002"
	evidencia.VersionEmitida.RegistroRef = "registro-cotejo-transiciones-002"
	reservado := transicionesCotejoPruebaCodigoReservado(
		t, transicionesCotejoPruebaSustitutoID, documento, transicionesCotejoPruebaIndiceSustituto,
		transicionesCotejoPruebaProteccionSustituto, entorno.ahora,
	)
	sustituto := transicionesCotejoPruebaActivarFixture(t, reservado, evidencia, entorno.ahora.Add(-time.Minute))
	transicionesCotejoPruebaValidarFixtures(t, documento, representacion, evidencia, sustituto)

	entorno.documentos.documentos[documento.Referencia()] = documento
	entorno.documentos.representaciones[documento.Referencia()] = []domain.RepresentacionDocumento{representacion}
	entorno.evidencias.evidencias[transicionesCotejoPruebaClaveEvidencia{
		documento: documento.Referencia(), representacionID: representacion.ID,
	}] = evidencia
	entorno.codigos.codigos[sustituto.ID] = sustituto
	return anterior, sustituto
}

func transicionesCotejoPruebaOrdenSustituir(
	entorno *transicionesCotejoPruebaEntorno,
	anterior, sustituto domain.CodigoCotejo,
) OrdenSustituirCodigoCotejo {
	return OrdenSustituirCodigoCotejo{
		Principal:       entorno.principal,
		PerfilActivo:    perfilAutorizacionPrueba("cotejo-transiciones-001"),
		RepresentadoRef: "persona-representada-cotejo-transiciones-001",
		Finalidad:       "sustituir_documento_verificable",
		CodigoID:        anterior.ID,
		SustitutoID:     sustituto.ID,
		SustitucionRef:  "acuerdo-sustitucion-cotejo-transiciones-001",
		Motivo:          "sustitucion autorizada por una nueva version documental",
		CorrelacionRef:  "correlacion-sustituir-cotejo-transiciones-001",
	}
}

func transicionesCotejoPruebaValidarFixtures(
	t *testing.T,
	documento domain.DocumentoLogico,
	representacion domain.RepresentacionDocumento,
	evidencia domain.EvidenciaEmisionDocumento,
	codigo domain.CodigoCotejo,
) {
	t.Helper()
	if err := documento.Validar(); err != nil {
		t.Fatalf("documento de prueba invalido: %v", err)
	}
	if err := representacion.ValidarPertenencia(documento); err != nil {
		t.Fatalf("representacion de prueba invalida: %v", err)
	}
	if err := evidencia.Validar(); err != nil {
		t.Fatalf("evidencia de prueba invalida: %v", err)
	}
	if err := codigo.Validar(); err != nil {
		t.Fatalf("codigo de prueba invalido: %v", err)
	}
}

func transicionesCotejoPruebaUnicaConfirmacion(
	t *testing.T,
	repositorio *transicionesCotejoPruebaCodigos,
	operacion string,
) transicionesCotejoPruebaConfirmacion {
	t.Helper()
	if len(repositorio.confirmaciones) != 1 {
		t.Fatalf("confirmaciones = %d; esperada 1", len(repositorio.confirmaciones))
	}
	confirmacion := repositorio.confirmaciones[0]
	if confirmacion.operacion != operacion {
		t.Fatalf("operacion confirmada = %q; esperada %q", confirmacion.operacion, operacion)
	}
	return confirmacion
}

func transicionesCotejoPruebaExigirEvidenciasSinMaterialSensible(
	t *testing.T,
	confirmacion transicionesCotejoPruebaConfirmacion,
) {
	t.Helper()
	contenido, err := json.Marshal(struct {
		Traza  domain.AuditEntry `json:"traza"`
		Evento domain.Event      `json:"evento"`
	}{Traza: confirmacion.traza, Evento: confirmacion.evento})
	if err != nil {
		t.Fatalf("serializar auditoria/outbox: %v", err)
	}
	texto := string(contenido)
	for _, prohibido := range []string{
		transicionesCotejoPruebaSecreto,
		transicionesCotejoPruebaIndice,
		transicionesCotejoPruebaIndiceSustituto,
		transicionesCotejoPruebaProteccion,
		transicionesCotejoPruebaProteccionSustituto,
	} {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("auditoria/outbox contiene material sensible %q: %s", prohibido, texto)
		}
	}
	textoMinusculas := strings.ToLower(texto)
	for _, prohibido := range []string{"hmac-sha256", "indice_codigo_hmac", "proteccion_ref"} {
		if strings.Contains(textoMinusculas, prohibido) {
			t.Fatalf("auditoria/outbox contiene material sensible %q: %s", prohibido, texto)
		}
	}
}
