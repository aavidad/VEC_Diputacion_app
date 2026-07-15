package memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type usoAutorizacionDocumento struct {
	EfectoRef      string
	HuellaEfecto   string
	HuellaDecision string
}

const prefijoObjetoContenidoDocumentalMemoria = "objeto:documental:memoria:v1:"

func (s *Store) ConfirmarAltaBorradorPlantilla(ctx context.Context, plantilla domain.PlantillaDocumento, traza domain.AuditEntry, evento domain.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := plantilla.Validar(); err != nil {
		return err
	}
	if plantilla.Estado != domain.EstadoPlantillaBorrador || !trazaPlantillaValida(plantilla, traza, evento) {
		return domain.ErrPlantillaDocumentoInvalida
	}
	clave := clavePlantilla(plantilla.ID, plantilla.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, existe := s.plantillas[clave]; existe {
		return ports.ErrVersionPlantillaYaExiste
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.plantillas[clave] = clonarPlantilla(plantilla)
	trazaConfirmada := s.appendAuditLocked(traza)
	evento.Payload = cloneStringMap(evento.Payload)
	if evento.Payload == nil {
		evento.Payload = map[string]string{}
	}
	evento.Payload["auditoria_ref"] = trazaConfirmada.ID
	s.appendEventLocked(evento)
	return nil
}

func (s *Store) ConfirmarPublicacionPlantilla(ctx context.Context, huellaBorradorEsperada string, publicada domain.PlantillaDocumento, traza domain.AuditEntry, evento domain.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := publicada.Validar(); err != nil {
		return err
	}
	if publicada.Estado != domain.EstadoPlantillaPublicada || traza.BeforeHash != huellaBorradorEsperada ||
		!trazaPlantillaValida(publicada, traza, evento) {
		return domain.ErrPlantillaDocumentoInvalida
	}
	base := publicada
	base.Estado = domain.EstadoPlantillaBorrador
	base.PublicadaPor = ""
	base.PublicadaEn = time.Time{}
	base.AprobacionRef = ""
	base.MotivoPublicacion = ""
	huellaBase, err := base.HuellaSHA256()
	if err != nil || huellaBase != huellaBorradorEsperada {
		return ports.ErrHuellaContenidoNoCoincide
	}

	clave := clavePlantilla(publicada.ID, publicada.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	actual, existe := s.plantillas[clave]
	if !existe {
		return ports.ErrPlantillaDocumentoNoEncontrada
	}
	if actual.Estado != domain.EstadoPlantillaBorrador {
		return domain.ErrPlantillaDocumentoInvalida
	}
	huellaActual, err := actual.HuellaSHA256()
	if err != nil || huellaActual != huellaBorradorEsperada {
		return ports.ErrHuellaContenidoNoCoincide
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.plantillas[clave] = clonarPlantilla(publicada)
	trazaConfirmada := s.appendAuditLocked(traza)
	evento.Payload = cloneStringMap(evento.Payload)
	if evento.Payload == nil {
		evento.Payload = map[string]string{}
	}
	evento.Payload["auditoria_ref"] = trazaConfirmada.ID
	s.appendEventLocked(evento)
	return nil
}

func (s *Store) ObtenerPlantilla(ctx context.Context, id string, version int) (domain.PlantillaDocumento, error) {
	if err := ctx.Err(); err != nil {
		return domain.PlantillaDocumento{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return domain.PlantillaDocumento{}, err
	}
	plantilla, existe := s.plantillas[clavePlantilla(id, version)]
	if !existe {
		return domain.PlantillaDocumento{}, ports.ErrPlantillaDocumentoNoEncontrada
	}
	return clonarPlantilla(plantilla), nil
}

func (s *Store) ListarPlantillas(ctx context.Context, moduloID string) ([]domain.PlantillaDocumento, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if moduloID == "" || moduloID != strings.TrimSpace(moduloID) || strings.ContainsRune(moduloID, '*') {
		return nil, domain.ErrPermissionDenied
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resultado := make([]domain.PlantillaDocumento, 0, len(s.plantillas))
	for _, plantilla := range s.plantillas {
		if plantilla.ModuloID == moduloID {
			resultado = append(resultado, clonarPlantilla(plantilla))
		}
	}
	sort.Slice(resultado, func(i, j int) bool {
		if resultado[i].ID != resultado[j].ID {
			return resultado[i].ID < resultado[j].ID
		}
		return resultado[i].Version < resultado[j].Version
	})
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return resultado, nil
}

func (s *Store) GuardarContenido(ctx context.Context, solicitud ports.SolicitudGuardarContenido) (ports.ContenidoDocumentoGuardado, error) {
	if err := ctx.Err(); err != nil {
		return ports.ContenidoDocumentoGuardado{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ContenidoDocumentoGuardado{}, err
	}
	proyeccion, err := solicitud.Contexto.Proyeccion()
	if err != nil {
		return ports.ContenidoDocumentoGuardado{}, err
	}
	suma := sha256.Sum256(solicitud.Contenido)
	huella := hex.EncodeToString(suma[:])
	if huella != solicitud.HuellaSHA256 {
		return ports.ContenidoDocumentoGuardado{}, ports.ErrHuellaContenidoNoCoincide
	}
	huellaSolicitud := huellaSolicitudContenidoDocumentalMemoria(solicitud, proyeccion)
	referencia := prefijoObjetoContenidoDocumentalMemoria + huellaSolicitud
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return ports.ContenidoDocumentoGuardado{}, err
	}
	ahora := time.Now().UTC()
	// Esta es la ultima comprobacion antes de observar o modificar el estado
	// que materializa el efecto. ValidarEn coteja tambien el paso exacto del
	// manifiesto; una capacidad generica nunca alcanza esta seccion critica.
	if err := solicitud.ValidarEn(ahora); err != nil {
		return ports.ContenidoDocumentoGuardado{}, err
	}
	vinculo, repetida := s.idempotenciasContenido[solicitud.ClaveIdempotencia]
	if repetida {
		if vinculo.HuellaSolicitudSHA256 != huellaSolicitud || vinculo.Referencia != referencia {
			return ports.ContenidoDocumentoGuardado{}, ports.ErrIdempotenciaAlmacenReutilizada
		}
		existente, existe := s.contenidos[vinculo.Referencia]
		if !existe {
			return ports.ContenidoDocumentoGuardado{}, ports.ErrIntegridadObjetoAlmacen
		}
		sumaExistente := sha256.Sum256(existente.Datos)
		if !strings.EqualFold(hex.EncodeToString(sumaExistente[:]), huella) || existente.MIME != solicitud.MIME ||
			existente.Zona != solicitud.Zona || !bytes.Equal(existente.Datos, solicitud.Contenido) {
			return ports.ContenidoDocumentoGuardado{}, ports.ErrIntegridadObjetoAlmacen
		}
	} else {
		if _, colision := s.contenidos[referencia]; colision {
			return ports.ContenidoDocumentoGuardado{}, ports.ErrIntegridadObjetoAlmacen
		}
		if err := ctx.Err(); err != nil {
			return ports.ContenidoDocumentoGuardado{}, err
		}
		s.contenidos[referencia] = objetoContenidoDocumento{
			MIME:  solicitud.MIME,
			Zona:  solicitud.Zona,
			Datos: append([]byte(nil), solicitud.Contenido...),
		}
		s.idempotenciasContenido[solicitud.ClaveIdempotencia] = idempotenciaContenidoDocumento{
			HuellaSolicitudSHA256: huellaSolicitud,
			Referencia:            referencia,
		}
	}
	evidencia := ports.EvidenciaOperacionAlmacen{
		Referencia: "evidencia-memoria-legada:" + huella, ConectorID: "memoria-legada",
		EsquemaContexto: proyeccion.Esquema, AccionNegocio: proyeccion.AccionNegocio,
		Accion: proyeccion.AccionTecnica, EfectoRef: proyeccion.EfectoRef,
		HuellaPlanEfectoSHA256: proyeccion.HuellaPlanEfectoSHA256,
		HuellaManifiestoSHA256: proyeccion.HuellaManifiestoSHA256,
		HuellaPasoSHA256:       proyeccion.HuellaPasoSHA256, PasoRef: proyeccion.PasoRef,
		HuellaDecisionSHA256: proyeccion.HuellaDecisionSHA256,
		Objeto:               ports.ReferenciaObjetoAlmacen{Referencia: referencia, Version: "1"},
		OperacionRef:         proyeccion.OperacionRef, CorrelacionRef: proyeccion.CorrelacionRef,
		AutorizacionRef: proyeccion.AutorizacionRef, Finalidad: proyeccion.Finalidad,
		Clasificacion: proyeccion.Clasificacion, RealizadaEn: ahora, CargaRef: proyeccion.CargaRef,
		SujetoSeudonimoHMAC: proyeccion.SujetoSeudonimoHMAC, RecursoRef: proyeccion.RecursoRef,
		ModuloID: proyeccion.ModuloID, HuellaSolicitudHMAC: proyeccion.HuellaSolicitudHMAC,
		ReintentoIdempotente: repetida,
	}
	guardado := ports.ContenidoDocumentoGuardado{
		ReferenciaLogica:   solicitud.DocumentoID,
		Referencia:         referencia,
		Version:            "1",
		ConectorID:         evidencia.ConectorID,
		Zona:               solicitud.Zona,
		MIME:               solicitud.MIME,
		HuellaSHA256:       huella,
		Tamano:             solicitud.Tamano,
		EvidenciaOperacion: evidencia,
	}
	if err := guardado.ValidarContra(solicitud); err != nil {
		return ports.ContenidoDocumentoGuardado{}, ports.ErrIntegridadObjetoAlmacen
	}
	return guardado, nil
}

// referenciaContenidoDocumentalMemoria crea una identidad opaca estable para
// el paso exacto. No es una ruta ni una URL y no expone la clave idempotente,
// la referencia logica o metadatos personales en texto claro.
func huellaSolicitudContenidoDocumentalMemoria(
	solicitud ports.SolicitudGuardarContenido,
	proyeccion ports.ProyeccionContextoOperacionAlmacen,
) string {
	valores := []string{
		"vec.memoria.contenido-documental.v1", solicitud.ClaveIdempotencia,
		solicitud.DocumentoID, string(solicitud.Zona), solicitud.MIME,
		strconv.FormatInt(solicitud.Tamano, 10), solicitud.HuellaSHA256,
		proyeccion.EfectoRef, proyeccion.HuellaPlanEfectoSHA256,
		proyeccion.HuellaManifiestoSHA256, proyeccion.HuellaPasoSHA256,
	}
	calculador := sha256.New()
	for _, valor := range valores {
		_, _ = calculador.Write([]byte(strconv.Itoa(len(valor))))
		_, _ = calculador.Write([]byte{':'})
		_, _ = calculador.Write([]byte(valor))
		_, _ = calculador.Write([]byte{'\n'})
	}
	return hex.EncodeToString(calculador.Sum(nil))
}

func (s *Store) LeerContenido(ctx context.Context, solicitud ports.SolicitudLeerContenido) (ports.ContenidoDocumentoLeido, error) {
	if err := ctx.Err(); err != nil {
		return ports.ContenidoDocumentoLeido{}, err
	}
	ahora := time.Now().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenLeer, ahora); err != nil {
		return ports.ContenidoDocumentoLeido{}, err
	}
	proyeccion, err := solicitud.Contexto.Proyeccion()
	if err != nil {
		return ports.ContenidoDocumentoLeido{}, err
	}
	if !solicitud.Zona.Valida() || strings.TrimSpace(solicitud.Referencia) == "" {
		return ports.ContenidoDocumentoLeido{}, ports.ErrSolicitudAlmacenInvalida
	}
	if proyeccion.ObjetoVinculado.Referencia != strings.TrimSpace(solicitud.Referencia) ||
		proyeccion.ObjetoVinculado.Version != "1" {
		return ports.ContenidoDocumentoLeido{}, ports.ErrSolicitudAlmacenInvalida
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return ports.ContenidoDocumentoLeido{}, err
	}
	ahora = time.Now().UTC()
	if err := solicitud.Contexto.ValidarParaEn(ports.AccionAlmacenLeer, ahora); err != nil {
		return ports.ContenidoDocumentoLeido{}, err
	}
	referencia := strings.TrimSpace(solicitud.Referencia)
	contenido, existe := s.contenidos[referencia]
	if !existe {
		return ports.ContenidoDocumentoLeido{}, ports.ErrContenidoDocumentoNoEncontrado
	}
	if contenido.Zona != solicitud.Zona {
		return ports.ContenidoDocumentoLeido{}, ports.ErrTransicionZonaAlmacenNoPermitida
	}
	if solicitud.Limite <= 0 || int64(len(contenido.Datos)) > solicitud.Limite {
		return ports.ContenidoDocumentoLeido{}, ports.ErrLimiteLecturaExcedido
	}
	suma := sha256.Sum256(contenido.Datos)
	huella := hex.EncodeToString(suma[:])
	evidencia := ports.EvidenciaOperacionAlmacen{
		Referencia: "evidencia-lectura-memoria-legada:" + huella, ConectorID: "memoria-legada",
		EsquemaContexto: proyeccion.Esquema, AccionNegocio: proyeccion.AccionNegocio,
		Accion: proyeccion.AccionTecnica, EfectoRef: proyeccion.EfectoRef,
		HuellaPlanEfectoSHA256: proyeccion.HuellaPlanEfectoSHA256,
		HuellaManifiestoSHA256: proyeccion.HuellaManifiestoSHA256,
		HuellaPasoSHA256:       proyeccion.HuellaPasoSHA256, PasoRef: proyeccion.PasoRef,
		HuellaDecisionSHA256: proyeccion.HuellaDecisionSHA256,
		Objeto:               ports.ReferenciaObjetoAlmacen{Referencia: referencia, Version: "1"},
		OperacionRef:         proyeccion.OperacionRef, CorrelacionRef: proyeccion.CorrelacionRef,
		AutorizacionRef: proyeccion.AutorizacionRef, Finalidad: proyeccion.Finalidad,
		Clasificacion: proyeccion.Clasificacion, RealizadaEn: ahora, CargaRef: proyeccion.CargaRef,
		SujetoSeudonimoHMAC: proyeccion.SujetoSeudonimoHMAC, RecursoRef: proyeccion.RecursoRef,
		ModuloID: proyeccion.ModuloID, HuellaSolicitudHMAC: proyeccion.HuellaSolicitudHMAC,
	}
	if err := ctx.Err(); err != nil {
		return ports.ContenidoDocumentoLeido{}, err
	}
	return ports.ContenidoDocumentoLeido{
		Contenido:          append([]byte(nil), contenido.Datos...),
		ConectorID:         evidencia.ConectorID,
		Zona:               solicitud.Zona,
		HuellaSHA256:       huella,
		Tamano:             int64(len(contenido.Datos)),
		EvidenciaOperacion: evidencia,
	}, nil
}

func (s *Store) ConfirmarGeneracion(
	ctx context.Context,
	documento domain.DocumentoGenerado,
	traza domain.AuditEntry,
	evento domain.Event,
	evidencia ports.EvidenciaUsoDecisionAutorizacion,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Se trabaja con una instantanea defensiva. El puerto no conserva mapas o
	// slices propiedad del llamador dentro del estado confirmado.
	documento = clonarDocumento(documento)
	traza = cloneAuditEntry(traza)
	evento = cloneEvent(evento)
	if err := documento.Validar(); err != nil {
		return err
	}
	if !trazaDocumentoValida(documento, traza, evento) {
		return domain.ErrDocumentoInvalido
	}
	datosEvidencia, err := evidencia.Datos()
	if err != nil {
		return err
	}
	efectoRef := "documento-generado:" + documento.ID
	huellaEfecto, err := huellaEfectoDocumento(documento, traza, evento, datosEvidencia)
	if err != nil {
		return errors.Join(domain.ErrDocumentoInvalido, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	plantilla, existe := s.plantillas[clavePlantilla(documento.PlantillaID, documento.PlantillaVersion)]
	if !existe || !vinculoAutorizacionDocumentoValido(documento, traza, plantilla, datosEvidencia) {
		return errorVinculoAutorizacionDocumento(nil)
	}
	// Documento.GeneradoEn representa el reloj transaccional del adaptador en
	// memoria. Un adaptador duradero debe usar la hora de la propia base de datos
	// dentro de la transaccion, nunca una fecha aportada por el cliente.
	if err := evidencia.ValidarEn(documento.GeneradoEn); err != nil {
		return err
	}
	if uso, consumida := s.usosAutorizacionDoc[datosEvidencia.Decision.DecisionRef]; consumida {
		if uso.EfectoRef == efectoRef && uso.HuellaEfecto == huellaEfecto &&
			uso.HuellaDecision == datosEvidencia.HuellaDecisionSHA256 {
			if _, confirmada := s.documentos[documento.ID]; confirmada {
				// Reintento idempotente: el primer intento ya confirmo el agregado,
				// la auditoria y el evento. No se vuelve a escribir ninguno.
				return nil
			}
		}
		return errorVinculoAutorizacionDocumento(ports.ErrDecisionAutorizacionConsumida)
	}
	if _, existe := s.documentos[documento.ID]; existe {
		return ports.ErrDocumentoYaExiste
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.documentos[documento.ID] = clonarDocumento(documento)
	s.usosAutorizacionDoc[datosEvidencia.Decision.DecisionRef] = usoAutorizacionDocumento{
		EfectoRef: efectoRef, HuellaEfecto: huellaEfecto, HuellaDecision: datosEvidencia.HuellaDecisionSHA256,
	}
	trazaConfirmada := s.appendAuditLocked(traza)
	evento.Payload = cloneStringMap(evento.Payload)
	if evento.Payload == nil {
		evento.Payload = map[string]string{}
	}
	evento.Payload["auditoria_ref"] = trazaConfirmada.ID
	s.appendEventLocked(evento)
	return nil
}

func huellaEfectoDocumento(
	documento domain.DocumentoGenerado,
	traza domain.AuditEntry,
	evento domain.Event,
	evidencia ports.DatosEvidenciaUsoDecisionAutorizacion,
) (string, error) {
	contenido, err := json.Marshal(struct {
		Esquema         string                   `json:"esquema"`
		Documento       domain.DocumentoGenerado `json:"documento"`
		Traza           domain.AuditEntry        `json:"traza"`
		Evento          domain.Event             `json:"evento"`
		EsquemaDecision string                   `json:"esquema_decision"`
		HuellaDecision  string                   `json:"huella_decision"`
	}{
		Esquema:         "vec.documento.generado.efecto.v1",
		Documento:       clonarDocumento(documento),
		Traza:           cloneAuditEntry(traza),
		Evento:          cloneEvent(evento),
		EsquemaDecision: evidencia.EsquemaHuella,
		HuellaDecision:  evidencia.HuellaDecisionSHA256,
	})
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func vinculoAutorizacionDocumentoValido(
	documento domain.DocumentoGenerado,
	traza domain.AuditEntry,
	plantilla domain.PlantillaDocumento,
	evidencia ports.DatosEvidenciaUsoDecisionAutorizacion,
) bool {
	decision := evidencia.Decision
	if plantilla.Validar() != nil || plantilla.Estado != domain.EstadoPlantillaPublicada ||
		strings.ContainsRune(plantilla.PermisoGenerar, '*') ||
		documento.PlantillaID != plantilla.ID || documento.PlantillaVersion != plantilla.Version ||
		documento.ModuloID != plantilla.ModuloID || documento.TipoDocumental != plantilla.TipoDocumental ||
		!plantilla.AdmiteFormato(documento.Formato) ||
		decision.DecisionRef != traza.AuthorizationRef ||
		decision.PrincipalID != documento.GeneradoPor || decision.PrincipalID != traza.ActorID ||
		decision.PerfilActivoRef != traza.ActorProfile ||
		decision.Accion != plantilla.PermisoGenerar ||
		decision.RecursoRef != documento.ExpedienteRef ||
		decision.ModuloID != documento.ModuloID || decision.TipoRecurso != "expediente" ||
		decision.Finalidad != traza.Purpose || decision.CorrelacionRef != documento.CorrelacionRef ||
		decision.CorrelacionRef != traza.CorrelationRef ||
		!domain.CumpleGarantiaAutenticacion(traza.AuthAssurance, decision.GarantiaMinima) ||
		!domain.CumpleGarantiaAutenticacion(traza.AuthAssurance, plantilla.GarantiaMinima) {
		return false
	}
	// Compatibilidad cerrada del contrato anterior: la decision historica
	// comprometia exactamente este recurso de dos atributos. Una decision V1
	// ligada a manifiesto, efecto y pasos produce necesariamente otra huella y
	// se deniega. Este repositorio no intenta reconstruirla desde Metadata: la
	// generacion compuesta solo podra confirmarse cuando el puerto aporte la
	// atestacion durable tipada de su reserva y de todos sus pasos.
	recurso := domain.RecursoAutorizable{
		Referencia: documento.ExpedienteRef,
		ModuloID:   plantilla.ModuloID,
		Tipo:       "expediente",
		Atributos: map[string]string{
			"plantilla_id":    plantilla.ID,
			"tipo_documental": plantilla.TipoDocumental,
		},
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	return err == nil && decision.ContextoRecursoHuellaSHA256 == huellaContexto
}

func errorVinculoAutorizacionDocumento(causa error) error {
	return errors.Join(
		domain.ErrAutorizacionDenegada,
		domain.ErrDecisionAutorizacionInvalida,
		ports.ErrEvidenciaUsoDecisionAutorizacionInvalida,
		causa,
	)
}

func (s *Store) ObtenerDocumento(ctx context.Context, id string) (domain.DocumentoGenerado, error) {
	if err := ctx.Err(); err != nil {
		return domain.DocumentoGenerado{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return domain.DocumentoGenerado{}, err
	}
	documento, existe := s.documentos[strings.TrimSpace(id)]
	if !existe {
		return domain.DocumentoGenerado{}, ports.ErrDocumentoNoEncontrado
	}
	return clonarDocumento(documento), nil
}

func (s *Store) ListarDocumentosExpediente(ctx context.Context, expedienteRef string) ([]domain.DocumentoGenerado, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if expedienteRef == "" || expedienteRef != strings.TrimSpace(expedienteRef) || strings.ContainsRune(expedienteRef, '*') {
		return nil, domain.ErrPermissionDenied
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resultado := make([]domain.DocumentoGenerado, 0)
	for _, documento := range s.documentos {
		if documento.ExpedienteRef == expedienteRef {
			resultado = append(resultado, clonarDocumento(documento))
		}
	}
	sort.Slice(resultado, func(i, j int) bool {
		if !resultado[i].GeneradoEn.Equal(resultado[j].GeneradoEn) {
			return resultado[i].GeneradoEn.Before(resultado[j].GeneradoEn)
		}
		return resultado[i].ID < resultado[j].ID
	})
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return resultado, nil
}

func clavePlantilla(id string, version int) string {
	return strings.TrimSpace(id) + ":" + strconv.Itoa(version)
}

func clonarPlantilla(plantilla domain.PlantillaDocumento) domain.PlantillaDocumento {
	plantilla.Parrafos = append([]string(nil), plantilla.Parrafos...)
	plantilla.Campos = append([]domain.CampoPlantillaDocumento(nil), plantilla.Campos...)
	plantilla.Formatos = append([]domain.FormatoDocumento(nil), plantilla.Formatos...)
	return plantilla
}

func clonarDocumento(documento domain.DocumentoGenerado) domain.DocumentoGenerado {
	documento.FirmaRefs = append([]string(nil), documento.FirmaRefs...)
	return documento
}

func trazaPlantillaValida(plantilla domain.PlantillaDocumento, traza domain.AuditEntry, evento domain.Event) bool {
	referencia := clavePlantilla(plantilla.ID, plantilla.Version)
	huella, err := plantilla.HuellaSHA256()
	if err != nil {
		return false
	}
	accion := "vec.documentos.plantilla.borrador.creado"
	actor := plantilla.CreadaPor
	antes := ""
	regla := ""
	fecha := plantilla.CreadaEn
	if plantilla.Estado == domain.EstadoPlantillaPublicada {
		accion = "vec.documentos.plantilla.publicada"
		actor = plantilla.PublicadaPor
		regla = plantilla.AprobacionRef
		fecha = plantilla.PublicadaEn
		if strings.TrimSpace(traza.BeforeHash) == "" {
			return false
		}
		antes = traza.BeforeHash
	}
	return strings.TrimSpace(traza.ActorID) != "" && traza.ActorID == actor &&
		strings.TrimSpace(traza.ActorProfile) != "" &&
		strings.TrimSpace(traza.AuthorizationRef) != "" && strings.TrimSpace(traza.Purpose) != "" &&
		traza.Action == accion && traza.ModuleID == plantilla.ModuloID && traza.ObjectVersion == plantilla.Version &&
		traza.RuleRef == regla && traza.BeforeHash == antes && traza.AfterHash == huella && traza.Result == "correcto" &&
		strings.TrimSpace(traza.Reason) != "" && strings.TrimSpace(traza.CorrelationRef) != "" &&
		traza.OccurredAt.Equal(fecha) && traza.SubjectRef == referencia &&
		evento.Type == accion && evento.SubjectRef == referencia && evento.ActorID == actor &&
		evento.ModuleID == plantilla.ModuloID && evento.OccurredAt.Equal(fecha) &&
		evento.Payload["plantilla_id"] == plantilla.ID &&
		evento.Payload["plantilla_version"] == strconv.Itoa(plantilla.Version) &&
		evento.Payload["estado"] == string(plantilla.Estado) && evento.Payload["huella_sha256"] == huella
}

func trazaDocumentoValida(documento domain.DocumentoGenerado, traza domain.AuditEntry, evento domain.Event) bool {
	referenciaPlantilla := clavePlantilla(documento.PlantillaID, documento.PlantillaVersion)
	estadoEsperado := domain.EstadoDocumentoGenerado
	if documento.Formato == domain.FormatoDocumentoDOCX {
		estadoEsperado = domain.EstadoDocumentoBorrador
	}
	return documento.Version == 1 && documento.Estado == estadoEsperado &&
		!strings.ContainsRune(documento.ID, '*') && !strings.ContainsRune(documento.ReferenciaContenido, '*') &&
		documento.EstadoAntivirus == domain.EstadoAntivirusNoAplica && len(documento.FirmaRefs) == 0 &&
		documento.RegistroRef == "" && documento.CSV == "" &&
		instanteEfectoDocumentalCanonico(documento.GeneradoEn) &&
		instanteEfectoDocumentalCanonico(documento.ENI.FechaCaptura) &&
		instanteEfectoDocumentalCanonico(traza.OccurredAt) &&
		instanteEfectoDocumentalCanonico(evento.OccurredAt) &&
		documento.ENI.Identificador == documento.ID &&
		documento.ENI.TipoDocumental == documento.TipoDocumental &&
		documento.ENI.FechaCaptura.Equal(documento.GeneradoEn) &&
		traza.ID == "" && traza.Seq == 0 && traza.IntegrityAlgorithm == "" &&
		traza.PrevSignature == "" && traza.Signature == "" && evento.ID == "" &&
		traza.ActorID == documento.GeneradoPor && strings.TrimSpace(traza.ActorProfile) != "" &&
		traza.AuthMethod.Valido() && traza.AuthAssurance.Valida() &&
		strings.TrimSpace(traza.AuthorizationRef) != "" && strings.TrimSpace(traza.Purpose) != "" &&
		traza.Action == "vec.documento.generado" && traza.ModuleID == documento.ModuloID &&
		traza.SubjectRef == documento.ID && traza.ObjectVersion == documento.Version &&
		traza.ExpedienteRef == documento.ExpedienteRef && traza.DocumentRef == documento.ID &&
		traza.RuleRef == referenciaPlantilla && strings.TrimSpace(traza.Reason) == documento.Motivo &&
		traza.Result == "correcto" && traza.BeforeHash == "" && traza.AfterHash == documento.HuellaSHA256 &&
		traza.CorrelationRef == documento.CorrelacionRef && traza.OccurredAt.Equal(documento.GeneradoEn) &&
		evento.Type == "vec.documento.generado" && evento.ModuleID == documento.ModuloID &&
		evento.SubjectRef == documento.ID && evento.ActorID == documento.GeneradoPor &&
		evento.OccurredAt.Equal(documento.GeneradoEn) &&
		evento.Payload["documento_ref"] == documento.ID &&
		evento.Payload["expediente_ref"] == documento.ExpedienteRef &&
		evento.Payload["formato"] == string(documento.Formato) &&
		evento.Payload["huella_sha256"] == documento.HuellaSHA256 &&
		len(evento.Payload) == 4 && metadatosTrazaDocumentoValidos(documento, traza.Metadata)
}

func instanteEfectoDocumentalCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC && instante.Nanosecond()%1_000 == 0
}

func metadatosTrazaDocumentoValidos(documento domain.DocumentoGenerado, metadatos map[string]string) bool {
	return len(metadatos) == 8 &&
		strings.TrimSpace(metadatos["almacen_conector"]) != "" &&
		strings.TrimSpace(metadatos["almacen_evidencia_ref"]) != "" &&
		metadatos["formato"] == string(documento.Formato) &&
		metadatos["huella_datos_hmac"] == documento.HuellaDatosHMAC &&
		metadatos["mime"] == documento.MIME &&
		metadatos["plantilla_id"] == documento.PlantillaID &&
		metadatos["plantilla_version"] == strconv.Itoa(documento.PlantillaVersion) &&
		metadatos["tamano"] == strconv.FormatInt(documento.Tamano, 10)
}

var _ ports.CatalogoPlantillasDocumento = (*Store)(nil)
var _ ports.AlmacenContenidoDocumento = (*Store)(nil)
var _ ports.RepositorioDocumentos = (*Store)(nil)
var _ ports.RepositorioGobiernoPlantillasDocumento = (*Store)(nil)
