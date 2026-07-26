package registroaccesos

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"time"

	registroaplicacion "vec-diputacion-granada/internal/modules/bolsa/application/registroaccesos"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const maximoRespuesta = 1024 * 1024

type entradaJSON struct {
	ActorID              string            `json:"actor_id"`
	ActorProfile         string            `json:"actor_profile"`
	ActorRoles           []string          `json:"actor_roles"`
	RepresentedSubjectID string            `json:"represented_subject_id"`
	AuthMethod           string            `json:"auth_method"`
	AuthAssurance        string            `json:"auth_assurance"`
	AuthorizationRef     string            `json:"authorization_ref"`
	Purpose              string            `json:"purpose"`
	Action               string            `json:"action"`
	ModuleID             string            `json:"module_id"`
	SubjectRef           string            `json:"subject_ref"`
	ObjectVersion        int64             `json:"object_version"`
	ExpedienteRef        string            `json:"expediente_ref"`
	DocumentRef          string            `json:"document_ref"`
	RuleRef              string            `json:"rule_ref"`
	Reason               string            `json:"reason"`
	Result               string            `json:"result"`
	BeforeHash           string            `json:"before_hash"`
	AfterHash            string            `json:"after_hash"`
	CorrelationRef       string            `json:"correlation_ref"`
	Metadata             map[string]string `json:"metadata"`
	OccurredAt           time.Time         `json:"occurred_at"`
}

type entradaConfirmadaJSON struct {
	ID                 string `json:"id"`
	Seq                int64  `json:"seq"`
	IntegrityAlgorithm string `json:"integrity_algorithm"`
	PrevSignature      string `json:"prev_signature"`
	Signature          string `json:"signature"`
	entradaJSON
}

type filtroJSON struct {
	Version               uint16 `json:"version"`
	ActorSeudonimizado    string `json:"actor_seudonimizado"`
	ModuloID              string `json:"module_id"`
	Accion                string `json:"accion"`
	FinalidadAcceso       string `json:"finalidad_acceso"`
	RecursoRef            string `json:"recurso_ref"`
	ExpedienteRef         string `json:"expediente_ref"`
	Resultado             string `json:"resultado"`
	DesdeInclusive        string `json:"desde_inclusive"`
	HastaExclusive        string `json:"hasta_exclusive"`
	VersionObjeto         uint64 `json:"version_objeto"`
	Limite                uint16 `json:"limite"`
	Cursor                string `json:"cursor"`
	FinalidadDeLaConsulta string `json:"finalidad_consulta"`
}

type autorizacionJSON struct {
	Prueba           pruebaAutorizacionJSON `json:"prueba"`
	DecisionCanonica []byte                 `json:"decision_canonica"`
	RecursoCanonico  []byte                 `json:"recurso_canonico"`
}

type pruebaAutorizacionJSON struct {
	EsquemaHuella        string `json:"esquema_huella"`
	DecisionRef          string `json:"decision_ref"`
	HuellaDecisionSHA256 string `json:"huella_decision_sha256"`
	VerificadaEn         string `json:"verificada_en"`
	PrincipalRef         string `json:"principal_ref"`
}

type consultaJSON struct {
	Version      uint16           `json:"version"`
	Filtro       filtroJSON       `json:"filtro"`
	Auditoria    entradaJSON      `json:"auditoria"`
	Autorizacion autorizacionJSON `json:"autorizacion"`
}

type resumenJSON struct {
	RegistroRef        string    `json:"registro_ref"`
	ActorSeudonimizado string    `json:"actor_seudonimizado"`
	ModuloID           string    `json:"modulo_id"`
	Accion             string    `json:"accion"`
	Finalidad          string    `json:"finalidad"`
	RecursoRef         string    `json:"recurso_ref"`
	ExpedienteRef      string    `json:"expediente_ref"`
	Resultado          string    `json:"resultado"`
	VersionObjeto      uint64    `json:"version_objeto"`
	OcurridoEn         time.Time `json:"ocurrido_en"`
	VersionEsquema     uint16    `json:"version_esquema"`
}

type paginaJSON struct {
	Auditoria entradaConfirmadaJSON `json:"auditoria"`
	Registros []resumenJSON         `json:"registros"`
	Siguiente string                `json:"siguiente"`
}

func serializarEntrada(entrada vecdomain.AuditEntry) ([]byte, error) {
	if registroaplicacion.ValidarEntradaRegistroAcceso(entrada) != nil {
		return nil, registroaplicacion.ErrRegistroAccesosInvalido
	}
	contenido, err := json.Marshal(desdeEntrada(entrada))
	if err != nil {
		return nil, registroaplicacion.ErrRegistroAccesosInvalido
	}
	return contenido, nil
}

func serializarConsulta(
	solicitud registroaplicacion.SolicitudConsultaAdministrativaAccesos,
	datos vecports.DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	revalidadaEn time.Time,
) ([]byte, error) {
	filtro, errFiltro := solicitud.Filtro()
	auditoria, errAuditoria := solicitud.AuditoriaLectura()
	canonica, errCanonica := datos.RepresentacionCanonica()
	recurso, errRecurso := solicitud.RecursoAutorizado()
	recursoCanonico, errRecursoCanonico := json.Marshal(struct {
		Ambitos   map[string]string `json:"ambitos"`
		Atributos map[string]string `json:"atributos"`
	}{
		Ambitos: recurso.Ambitos, Atributos: recurso.Atributos,
	})
	if errFiltro != nil || errAuditoria != nil || errCanonica != nil ||
		errRecurso != nil || errRecursoCanonico != nil ||
		datos.Decision.RecursoRef != recurso.Referencia ||
		revalidadaEn.Before(datos.VerificadaEn) {
		return nil, registroaplicacion.ErrConsultaAdministrativaAccesosDenegada
	}
	carga := consultaJSON{
		Version: 1,
		Filtro: filtroJSON{
			Version: filtro.Version, ActorSeudonimizado: filtro.ActorSeudonimizado,
			ModuloID: filtro.ModuloID, Accion: filtro.Accion,
			FinalidadAcceso: filtro.FinalidadAcceso, RecursoRef: filtro.RecursoRef,
			ExpedienteRef: filtro.ExpedienteRef, Resultado: filtro.Resultado,
			DesdeInclusive: formatearInstanteFiltro(filtro.DesdeInclusive),
			HastaExclusive: formatearInstanteFiltro(filtro.HastaExclusive),
			VersionObjeto:  filtro.VersionObjeto, Limite: filtro.Limite,
			Cursor:                filtro.Cursor.Valor(),
			FinalidadDeLaConsulta: filtro.FinalidadDeLaConsulta,
		},
		Auditoria: desdeEntrada(auditoria),
		Autorizacion: autorizacionJSON{
			Prueba: pruebaAutorizacionJSON{
				EsquemaHuella:        datos.EsquemaHuella,
				DecisionRef:          datos.Decision.DecisionRef,
				HuellaDecisionSHA256: datos.HuellaDecisionSHA256,
				VerificadaEn: datos.VerificadaEn.UTC().Format(
					"2006-01-02T15:04:05.000000Z",
				),
				PrincipalRef: datos.Decision.PrincipalID,
			},
			DecisionCanonica: canonica, RecursoCanonico: recursoCanonico,
		},
	}
	contenido, err := json.Marshal(carga)
	if err != nil || len(contenido) > maximoRespuesta {
		return nil, registroaplicacion.ErrConsultaAdministrativaAccesosDenegada
	}
	return contenido, nil
}

func formatearInstanteFiltro(instante time.Time) string {
	return instante.UTC().Format("2006-01-02T15:04:05.000000Z")
}

func restaurarEntradaConfirmada(contenido []byte) (vecdomain.AuditEntry, error) {
	var carga entradaConfirmadaJSON
	if err := decodificarCerrado(contenido, &carga); err != nil {
		return vecdomain.AuditEntry{}, registroaplicacion.ErrRegistroAccesosNoDisponible
	}
	return carga.restaurar(), nil
}

func restaurarPagina(
	solicitud registroaplicacion.SolicitudConsultaAdministrativaAccesos,
	contenido []byte,
) (registroaplicacion.PaginaConsultaAdministrativaAccesos, error) {
	var carga paginaJSON
	if err := decodificarCerrado(contenido, &carga); err != nil {
		return registroaplicacion.PaginaConsultaAdministrativaAccesos{},
			registroaplicacion.ErrRegistroAccesosNoDisponible
	}
	registros := make([]registroaplicacion.ResumenAccesoAdministrativo, len(carga.Registros))
	for indice, fila := range carga.Registros {
		registros[indice] = registroaplicacion.ResumenAccesoAdministrativo{
			RegistroRef: fila.RegistroRef, ActorSeudonimizado: fila.ActorSeudonimizado,
			ModuloID: fila.ModuloID, Accion: fila.Accion, Finalidad: fila.Finalidad,
			RecursoRef: fila.RecursoRef, ExpedienteRef: fila.ExpedienteRef,
			Resultado: fila.Resultado, VersionObjeto: fila.VersionObjeto,
			OcurridoEn: fila.OcurridoEn, VersionEsquema: fila.VersionEsquema,
		}
	}
	cursor, err := registroaplicacion.NuevoCursorConsultaAdministrativaAccesos(carga.Siguiente)
	if err != nil {
		return registroaplicacion.PaginaConsultaAdministrativaAccesos{},
			registroaplicacion.ErrRegistroAccesosNoDisponible
	}
	pagina, err := registroaplicacion.NuevaPaginaConsultaAdministrativaAccesos(
		solicitud, registros, cursor, carga.Auditoria.restaurar(),
	)
	if err != nil {
		return registroaplicacion.PaginaConsultaAdministrativaAccesos{},
			registroaplicacion.ErrRegistroAccesosNoDisponible
	}
	return pagina, nil
}

func desdeEntrada(entrada vecdomain.AuditEntry) entradaJSON {
	roles := append([]string(nil), entrada.ActorRoles...)
	if roles == nil {
		roles = []string{}
	}
	metadatos := make(map[string]string, len(entrada.Metadata))
	for clave, valor := range entrada.Metadata {
		metadatos[clave] = valor
	}
	return entradaJSON{
		ActorID: entrada.ActorID, ActorProfile: entrada.ActorProfile,
		ActorRoles: roles, RepresentedSubjectID: entrada.RepresentedSubjectID,
		AuthMethod: string(entrada.AuthMethod), AuthAssurance: string(entrada.AuthAssurance),
		AuthorizationRef: entrada.AuthorizationRef, Purpose: entrada.Purpose,
		Action: entrada.Action, ModuleID: entrada.ModuleID, SubjectRef: entrada.SubjectRef,
		ObjectVersion: int64(entrada.ObjectVersion), ExpedienteRef: entrada.ExpedienteRef,
		DocumentRef: entrada.DocumentRef, RuleRef: entrada.RuleRef, Reason: entrada.Reason,
		Result: entrada.Result, BeforeHash: entrada.BeforeHash, AfterHash: entrada.AfterHash,
		CorrelationRef: entrada.CorrelationRef, Metadata: metadatos, OccurredAt: entrada.OccurredAt,
	}
}

func (c entradaConfirmadaJSON) restaurar() vecdomain.AuditEntry {
	return vecdomain.AuditEntry{
		ID: c.ID, Seq: c.Seq, ActorID: c.ActorID, ActorProfile: c.ActorProfile,
		ActorRoles:           append([]string(nil), c.ActorRoles...),
		RepresentedSubjectID: c.RepresentedSubjectID,
		AuthMethod:           vecdomain.AuthMethod(c.AuthMethod),
		AuthAssurance:        vecdomain.AuthAssurance(c.AuthAssurance),
		AuthorizationRef:     c.AuthorizationRef, Purpose: c.Purpose, Action: c.Action,
		ModuleID: c.ModuleID, SubjectRef: c.SubjectRef, ObjectVersion: int(c.ObjectVersion),
		ExpedienteRef: c.ExpedienteRef, DocumentRef: c.DocumentRef, RuleRef: c.RuleRef,
		Reason: c.Reason, Result: c.Result, BeforeHash: c.BeforeHash, AfterHash: c.AfterHash,
		CorrelationRef: c.CorrelationRef, Metadata: clonarMapa(c.Metadata),
		OccurredAt: c.OccurredAt, IntegrityAlgorithm: c.IntegrityAlgorithm,
		PrevSignature: c.PrevSignature, Signature: c.Signature,
	}
}

func entradaConfirmadaEquivalente(
	solicitada, confirmada vecdomain.AuditEntry,
) bool {
	solicitada.ID = confirmada.ID
	solicitada.Seq = confirmada.Seq
	solicitada.IntegrityAlgorithm = confirmada.IntegrityAlgorithm
	solicitada.PrevSignature = confirmada.PrevSignature
	solicitada.Signature = confirmada.Signature
	return confirmada.ID != "" && confirmada.Seq > 0 &&
		confirmada.IntegrityAlgorithm == "sha256-chain-v1" &&
		reflect.DeepEqual(solicitada, confirmada)
}

func clonarMapa(origen map[string]string) map[string]string {
	destino := make(map[string]string, len(origen))
	for clave, valor := range origen {
		destino[clave] = valor
	}
	return destino
}

func decodificarCerrado(contenido []byte, destino any) error {
	if len(contenido) == 0 || len(contenido) > maximoRespuesta {
		return io.ErrUnexpectedEOF
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}
