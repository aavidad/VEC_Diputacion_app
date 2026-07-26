// Package registroaccesos contiene el contrato del caso de uso T13 junto a su
// modelo de solicitud. Vive en aplicación para no ampliar ports durante
// DEC-051.
package registroaccesos

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	VersionConsultaAdministrativaAccesos uint16 = 1
	MaximoRegistrosConsultaAccesos       uint16 = 100
	MaximoIntervaloConsultaAccesos              = 31 * 24 * time.Hour

	ModuloAuditoriaConsultaAccesos = "vec.module.bolsa"
	AccionAuditoriaConsultaAccesos = "bolsa.registro_accesos.consultar"

	ModuloAutorizacionConsultaAccesos      = "bolsa"
	TipoRecursoAutorizacionConsultaAccesos = "registro_accesos_administrativo"

	esquemaHuellaConsultaAccesos = "VEC-BOLSA-CONSULTA-ADMINISTRATIVA-ACCESOS-V1"
	ambitoSeudonimizacionAccesos = "bolsa_registro_accesos_t13"
)

var camposAutorizacionConsultaAccesos = [...]string{
	"accion",
	"actor_seudonimizado",
	"expediente_ref",
	"finalidad",
	"modulo_id",
	"ocurrido_en",
	"recurso_ref",
	"resultado",
	"version_esquema",
	"version_objeto",
}

func CamposPermitidosConsultaAdministrativaAccesos() []string {
	return append([]string(nil), camposAutorizacionConsultaAccesos[:]...)
}

var (
	ErrRegistroAccesosInvalido = errors.New(
		"bolsa: registro de acceso invalido",
	)
	ErrConsultaAdministrativaAccesosDenegada = errors.New(
		"bolsa: consulta administrativa de accesos denegada",
	)
	ErrRegistroAccesosNoDisponible = errors.New(
		"bolsa: registro durable de accesos no disponible",
	)
)

// RegistroAccesosAdministrativo es el contrato exclusivo de la consulta
// paginada y minimizada. No amplía AuditStore ni concede un append genérico:
// cada efecto de otra vertical necesita su propio wrapper VEC transaccional.
type RegistroAccesosAdministrativo interface {
	ConsultarAccesosAdministrativos(
		context.Context,
		SolicitudConsultaAdministrativaAccesos,
	) (PaginaConsultaAdministrativaAccesos, error)
}

type datosEvidenciaActorConsultaAccesos struct {
	principalID string
	seudonimo   string
	ambito      string
}

// EvidenciaActorConsultaAccesos es nominal y opaca. Solo la emite la frontera
// HSM/KMS común de VEC tras calcular el HMAC del principal para el ámbito T13.
// Un texto con forma de HMAC no permite reconstruir esta capacidad.
type EvidenciaActorConsultaAccesos struct {
	datos *datosEvidenciaActorConsultaAccesos
}

func NuevaEvidenciaActorConsultaAccesos(
	ctx context.Context,
	principalID string,
	seudonimizador vecports.SeudonimizadorSujetoAlmacen,
) (EvidenciaActorConsultaAccesos, error) {
	if ctx == nil || ctx.Err() != nil || seudonimizador == nil ||
		!textoAccesoValido(principalID, 512, false) {
		return EvidenciaActorConsultaAccesos{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	solicitud, err := vecports.NuevaSolicitudSeudonimizarSujetoAlmacen(
		principalID, ambitoSeudonimizacionAccesos,
	)
	if err != nil {
		return EvidenciaActorConsultaAccesos{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	seudonimo, err := seudonimizador.SeudonimizarSujetoAlmacen(
		ctx, solicitud,
	)
	if err != nil || !ActorSeudonimizadoValido(seudonimo) {
		return EvidenciaActorConsultaAccesos{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	return EvidenciaActorConsultaAccesos{
		datos: &datosEvidenciaActorConsultaAccesos{
			principalID: principalID, seudonimo: seudonimo,
			ambito: ambitoSeudonimizacionAccesos,
		},
	}, nil
}

func (e EvidenciaActorConsultaAccesos) validarPara(
	principalID, seudonimo string,
) error {
	if e.datos == nil || e.datos.principalID != principalID ||
		e.datos.seudonimo != seudonimo ||
		e.datos.ambito != ambitoSeudonimizacionAccesos ||
		!ActorSeudonimizadoValido(e.datos.seudonimo) {
		return ErrConsultaAdministrativaAccesosDenegada
	}
	return nil
}

func (EvidenciaActorConsultaAccesos) String() string {
	return "[EVIDENCIA-ACTOR-SEUDONIMIZADO-T13-OPACA]"
}

// CursorConsultaAdministrativaAccesos es opaco: su valor identifica una
// prueba encadenada ya existente, pero no revela la secuencia interna.
type CursorConsultaAdministrativaAccesos struct {
	valor string
}

func NuevoCursorConsultaAdministrativaAccesos(
	valor string,
) (CursorConsultaAdministrativaAccesos, error) {
	cursor := CursorConsultaAdministrativaAccesos{valor: valor}
	if valor != "" && !cursor.valido() {
		return CursorConsultaAdministrativaAccesos{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	return cursor, nil
}

func (c CursorConsultaAdministrativaAccesos) Valor() string {
	if !c.valido() {
		return ""
	}
	return c.valor
}

func (c CursorConsultaAdministrativaAccesos) EsCero() bool {
	return c.valor == ""
}

func (c CursorConsultaAdministrativaAccesos) valido() bool {
	if c.valor == "" {
		return true
	}
	const prefijo = "cursor:v1:"
	if len(c.valor) != len(prefijo)+sha256.Size*2 ||
		!strings.HasPrefix(c.valor, prefijo) {
		return false
	}
	return esHexadecimalMinusculo(strings.TrimPrefix(c.valor, prefijo))
}

func (CursorConsultaAdministrativaAccesos) String() string {
	return "[CURSOR-CONSULTA-ACCESOS-OPACO]"
}

func (c CursorConsultaAdministrativaAccesos) GoString() string {
	return c.String()
}

func (c CursorConsultaAdministrativaAccesos) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, c.String())
}

func (c CursorConsultaAdministrativaAccesos) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

// FiltroConsultaAdministrativaAccesos exige un intervalo cerrado y al menos
// un ancla de alta selectividad: actor seudonimizado, recurso o expediente.
// Los restantes campos son filtros exactos; nunca aceptan comodines.
type FiltroConsultaAdministrativaAccesos struct {
	Version               uint16
	ActorSeudonimizado    string
	ModuloID              string
	Accion                string
	FinalidadAcceso       string
	RecursoRef            string
	ExpedienteRef         string
	Resultado             string
	DesdeInclusive        time.Time
	HastaExclusive        time.Time
	VersionObjeto         uint64
	Limite                uint16
	Cursor                CursorConsultaAdministrativaAccesos
	FinalidadDeLaConsulta string
}

func (f FiltroConsultaAdministrativaAccesos) Validar() error {
	if f.Version != VersionConsultaAdministrativaAccesos ||
		f.Limite == 0 || f.Limite > MaximoRegistrosConsultaAccesos ||
		!instanteConsultaAccesosValido(f.DesdeInclusive) ||
		!instanteConsultaAccesosValido(f.HastaExclusive) ||
		!f.HastaExclusive.After(f.DesdeInclusive) ||
		f.HastaExclusive.Sub(f.DesdeInclusive) >
			MaximoIntervaloConsultaAccesos ||
		f.VersionObjeto > 9007199254740991 ||
		!f.Cursor.valido() ||
		!identificadorAccesoValido(f.FinalidadDeLaConsulta, 128) ||
		(f.ActorSeudonimizado == "" &&
			f.RecursoRef == "" &&
			f.ExpedienteRef == "") ||
		(f.ActorSeudonimizado != "" &&
			!ActorSeudonimizadoValido(f.ActorSeudonimizado)) ||
		!filtroExactoAccesoValido(f.ModuloID, 128) ||
		!filtroExactoAccesoValido(f.Accion, 160) ||
		!filtroExactoAccesoValido(f.FinalidadAcceso, 128) ||
		!filtroExactoAccesoValido(f.RecursoRef, 512) ||
		!filtroExactoAccesoValido(f.ExpedienteRef, 512) ||
		!resultadoFiltroAccesoValido(f.Resultado) {
		return ErrConsultaAdministrativaAccesosDenegada
	}
	return nil
}

func filtroExactoAccesoValido(valor string, maximo int) bool {
	return valor == "" ||
		(textoAccesoValido(valor, maximo, false) &&
			!strings.ContainsAny(valor, "*?"))
}

func resultadoFiltroAccesoValido(resultado string) bool {
	return resultado == "" || resultadoAccesoValido(resultado)
}

// SolicitudConsultaAdministrativaAccesos liga el filtro a la traza que debe
// quedar confirmada antes de que el adaptador lea o devuelva una sola fila.
type SolicitudConsultaAdministrativaAccesos struct {
	filtro       FiltroConsultaAdministrativaAccesos
	auditoria    vecdomain.AuditEntry
	autorizacion vecports.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	actor        EvidenciaActorConsultaAccesos
	huella       string
}

func NuevaSolicitudConsultaAdministrativaAccesos(
	filtro FiltroConsultaAdministrativaAccesos,
	auditoria vecdomain.AuditEntry,
	actor EvidenciaActorConsultaAccesos,
	autorizacion vecports.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
) (SolicitudConsultaAdministrativaAccesos, error) {
	huella, err := HuellaFiltroConsultaAdministrativaAccesos(filtro)
	if err != nil ||
		!auditoriaConsultaAccesosExacta(filtro, huella, auditoria) ||
		validarAutorizacionConsultaAccesos(
			filtro, huella, auditoria, actor, autorizacion,
			auditoria.OccurredAt,
		) != nil {
		return SolicitudConsultaAdministrativaAccesos{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	return SolicitudConsultaAdministrativaAccesos{
		filtro: filtro, auditoria: clonarEntradaAuditoriaAccesos(auditoria),
		autorizacion: autorizacion, actor: actor, huella: huella,
	}, nil
}

func (s SolicitudConsultaAdministrativaAccesos) Filtro() (
	FiltroConsultaAdministrativaAccesos,
	error,
) {
	if s.validar() != nil {
		return FiltroConsultaAdministrativaAccesos{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	return s.filtro, nil
}

func (s SolicitudConsultaAdministrativaAccesos) AuditoriaLectura() (
	vecdomain.AuditEntry,
	error,
) {
	if s.validar() != nil {
		return vecdomain.AuditEntry{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	return clonarEntradaAuditoriaAccesos(s.auditoria), nil
}

// RevalidarAutorizacion debe invocarse inmediatamente antes del efecto. La
// capacidad V2 procede del PDP; el AuditEntry solo es evidencia y nunca
// autoridad para conceder la consulta.
func (s SolicitudConsultaAdministrativaAccesos) RevalidarAutorizacion(
	instante time.Time,
) (
	vecports.DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	error,
) {
	if s.validar() != nil ||
		validarAutorizacionConsultaAccesos(
			s.filtro, s.huella, s.auditoria, s.actor, s.autorizacion,
			instante,
		) != nil {
		return vecports.DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	datos, err := s.autorizacion.Datos()
	if err != nil {
		return vecports.DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	return datos, nil
}

func (s SolicitudConsultaAdministrativaAccesos) RecursoAutorizado() (
	vecdomain.RecursoAutorizable,
	error,
) {
	if s.validar() != nil {
		return vecdomain.RecursoAutorizable{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	return RecursoAutorizableConsultaAdministrativaAccesos(
		s.filtro, s.actor,
	)
}

func (s SolicitudConsultaAdministrativaAccesos) validar() error {
	huella, err := HuellaFiltroConsultaAdministrativaAccesos(s.filtro)
	if err != nil || huella != s.huella ||
		!auditoriaConsultaAccesosExacta(s.filtro, huella, s.auditoria) ||
		validarAutorizacionConsultaAccesos(
			s.filtro, huella, s.auditoria, s.actor, s.autorizacion,
			s.auditoria.OccurredAt,
		) != nil {
		return ErrConsultaAdministrativaAccesosDenegada
	}
	return nil
}

// ResumenAccesoAdministrativo omite perfil, roles, identidad representada,
// autenticación, metadatos, hashes de negocio, motivo y referencias
// documentales. Contiene solo las coordenadas exigidas por T13.
type ResumenAccesoAdministrativo struct {
	RegistroRef        string
	ActorSeudonimizado string
	ModuloID           string
	Accion             string
	Finalidad          string
	RecursoRef         string
	ExpedienteRef      string
	Resultado          string
	VersionObjeto      uint64
	OcurridoEn         time.Time
	VersionEsquema     uint16
}

func (r ResumenAccesoAdministrativo) Validar() error {
	if !textoAccesoValido(r.RegistroRef, 160, false) ||
		!ActorSeudonimizadoValido(r.ActorSeudonimizado) ||
		!identificadorAccesoValido(r.ModuloID, 128) ||
		!identificadorAccesoValido(r.Accion, 160) ||
		!identificadorAccesoValido(r.Finalidad, 128) ||
		!textoAccesoValido(r.RecursoRef, 512, false) ||
		(r.ExpedienteRef != "" &&
			!textoAccesoValido(r.ExpedienteRef, 512, false)) ||
		!resultadoAccesoValido(r.Resultado) ||
		r.VersionObjeto > 9007199254740991 ||
		!instanteConsultaAccesosValido(r.OcurridoEn) ||
		r.VersionEsquema != VersionConsultaAdministrativaAccesos {
		return ErrRegistroAccesosInvalido
	}
	return nil
}

type PaginaConsultaAdministrativaAccesos struct {
	Registros           []ResumenAccesoAdministrativo
	Siguiente           CursorConsultaAdministrativaAccesos
	AuditoriaConfirmada vecdomain.AuditEntry
}

func NuevaPaginaConsultaAdministrativaAccesos(
	solicitud SolicitudConsultaAdministrativaAccesos,
	registros []ResumenAccesoAdministrativo,
	siguiente CursorConsultaAdministrativaAccesos,
	auditoriaConfirmada vecdomain.AuditEntry,
) (PaginaConsultaAdministrativaAccesos, error) {
	filtro, errFiltro := solicitud.Filtro()
	auditoriaSolicitada, errAuditoria := solicitud.AuditoriaLectura()
	if errFiltro != nil || errAuditoria != nil ||
		len(registros) > int(filtro.Limite) ||
		!siguiente.valido() ||
		!entradaConfirmadaCoincide(
			auditoriaSolicitada,
			auditoriaConfirmada,
		) {
		return PaginaConsultaAdministrativaAccesos{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	copia := make([]ResumenAccesoAdministrativo, len(registros))
	copy(copia, registros)
	for _, registro := range copia {
		if registro.Validar() != nil ||
			!registroCumpleFiltroConsultaAccesos(registro, filtro) {
			return PaginaConsultaAdministrativaAccesos{},
				ErrRegistroAccesosInvalido
		}
	}
	return PaginaConsultaAdministrativaAccesos{
		Registros:           copia,
		Siguiente:           siguiente,
		AuditoriaConfirmada: clonarEntradaAuditoriaAccesos(auditoriaConfirmada),
	}, nil
}

// ValidarEntradaRegistroAcceso cierra la parte de AuditEntry que este almacén
// acepta. Las identidades deben llegar seudonimizadas y todos los campos de
// integridad son siempre producidos por PostgreSQL.
func ValidarEntradaRegistroAcceso(entrada vecdomain.AuditEntry) error {
	if entrada.ID != "" || entrada.Seq != 0 ||
		entrada.IntegrityAlgorithm != "" ||
		entrada.PrevSignature != "" || entrada.Signature != "" ||
		!ActorSeudonimizadoValido(entrada.ActorID) ||
		(entrada.ActorProfile != "" &&
			!textoAccesoValido(entrada.ActorProfile, 160, false)) ||
		(entrada.RepresentedSubjectID != "" &&
			!ActorSeudonimizadoValido(entrada.RepresentedSubjectID)) ||
		!identificadorAccesoValido(entrada.Purpose, 128) ||
		!identificadorAccesoValido(entrada.Action, 160) ||
		!identificadorAccesoValido(entrada.ModuleID, 128) ||
		!textoAccesoValido(entrada.SubjectRef, 512, false) ||
		entrada.ObjectVersion < 0 ||
		uint64(entrada.ObjectVersion) > 9007199254740991 ||
		(entrada.ExpedienteRef != "" &&
			!textoAccesoValido(entrada.ExpedienteRef, 512, false)) ||
		(entrada.DocumentRef != "" &&
			!textoAccesoValido(entrada.DocumentRef, 512, false)) ||
		(entrada.RuleRef != "" &&
			!textoAccesoValido(entrada.RuleRef, 512, false)) ||
		(entrada.Reason != "" &&
			!identificadorAccesoValido(entrada.Reason, 160)) ||
		!resultadoAccesoValido(entrada.Result) ||
		!huellaOpcionalAccesoValida(entrada.BeforeHash) ||
		!huellaOpcionalAccesoValida(entrada.AfterHash) ||
		!textoAccesoValido(entrada.CorrelationRef, 160, false) ||
		!instanteConsultaAccesosValido(entrada.OccurredAt) ||
		len(entrada.ActorRoles) > 16 || len(entrada.Metadata) > 16 ||
		(entrada.AuthorizationRef != "" &&
			!textoAccesoValido(entrada.AuthorizationRef, 160, false)) {
		return ErrRegistroAccesosInvalido
	}
	if entrada.AuthMethod != "" && !entrada.AuthMethod.Valido() {
		return ErrRegistroAccesosInvalido
	}
	if entrada.AuthAssurance != "" && !entrada.AuthAssurance.Valida() {
		return ErrRegistroAccesosInvalido
	}
	roles := append([]string(nil), entrada.ActorRoles...)
	sort.Strings(roles)
	for indice, rol := range roles {
		if !identificadorAccesoValido(rol, 128) ||
			(indice > 0 && roles[indice-1] == rol) ||
			rol != entrada.ActorRoles[indice] {
			return ErrRegistroAccesosInvalido
		}
	}
	for clave, valor := range entrada.Metadata {
		if !identificadorAccesoValido(clave, 128) ||
			!textoAccesoValido(valor, 256, true) {
			return ErrRegistroAccesosInvalido
		}
	}
	return nil
}

func ActorSeudonimizadoValido(actor string) bool {
	partes := strings.Split(actor, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" ||
		len(partes[1]) == 0 || len(partes[1]) > 64 ||
		len(partes[2]) != sha256.Size*2 {
		return false
	}
	for indice, caracter := range partes[1] {
		if (indice == 0 && (caracter < 'a' || caracter > 'z')) ||
			(indice > 0 &&
				(caracter < 'a' || caracter > 'z') &&
				(caracter < '0' || caracter > '9') &&
				caracter != '.' && caracter != '_' && caracter != '-') {
			return false
		}
	}
	return esHexadecimalMinusculo(partes[2]) &&
		partes[2] != strings.Repeat("0", 64)
}

func HuellaFiltroConsultaAdministrativaAccesos(
	filtro FiltroConsultaAdministrativaAccesos,
) (string, error) {
	if filtro.Validar() != nil {
		return "", ErrConsultaAdministrativaAccesosDenegada
	}
	canon := strings.Join([]string{
		esquemaHuellaConsultaAccesos,
		fmt.Sprintf("%d", filtro.Version),
		filtro.ActorSeudonimizado,
		filtro.ModuloID,
		filtro.Accion,
		filtro.FinalidadAcceso,
		filtro.RecursoRef,
		filtro.ExpedienteRef,
		filtro.Resultado,
		formatearInstanteFiltroConsultaAccesos(filtro.DesdeInclusive),
		formatearInstanteFiltroConsultaAccesos(filtro.HastaExclusive),
		fmt.Sprintf("%d", filtro.VersionObjeto),
		fmt.Sprintf("%d", filtro.Limite),
		filtro.Cursor.Valor(),
		filtro.FinalidadDeLaConsulta,
	}, "\x00")
	suma := sha256.Sum256([]byte(canon))
	return hex.EncodeToString(suma[:]), nil
}

func RecursoAutorizableConsultaAdministrativaAccesos(
	filtro FiltroConsultaAdministrativaAccesos,
	actor EvidenciaActorConsultaAccesos,
) (vecdomain.RecursoAutorizable, error) {
	huella, err := HuellaFiltroConsultaAdministrativaAccesos(filtro)
	if err != nil || actor.datos == nil ||
		actor.validarPara(actor.datos.principalID, actor.datos.seudonimo) != nil {
		return vecdomain.RecursoAutorizable{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	recurso := vecdomain.RecursoAutorizable{
		Referencia: "consulta-accesos:sha256:" + huella,
		ModuloID:   ModuloAutorizacionConsultaAccesos,
		Tipo:       TipoRecursoAutorizacionConsultaAccesos,
		Ambitos: map[string]string{
			"version_esquema": fmt.Sprintf("%d", filtro.Version),
		},
		Atributos: atributosAutorizacionFiltroConsultaAccesos(
			filtro, actor.datos.seudonimo,
		),
	}
	if recurso.Validar() != nil {
		return vecdomain.RecursoAutorizable{},
			ErrConsultaAdministrativaAccesosDenegada
	}
	return recurso, nil
}

func validarAutorizacionConsultaAccesos(
	filtro FiltroConsultaAdministrativaAccesos,
	huella string,
	auditoria vecdomain.AuditEntry,
	actor EvidenciaActorConsultaAccesos,
	autorizacion vecports.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	instante time.Time,
) error {
	recurso, errRecurso := RecursoAutorizableConsultaAdministrativaAccesos(
		filtro, actor,
	)
	datos, errDatos := autorizacion.Datos()
	if errRecurso != nil || errDatos != nil ||
		autorizacion.ValidarEn(instante) != nil {
		return ErrConsultaAdministrativaAccesosDenegada
	}
	decision := datos.Decision
	huellaContexto, errHuella := recurso.HuellaContextoAutorizacionSHA256()
	vinculo, errVinculo := decision.VinculoAutenticacionActor.Datos()
	if errHuella != nil || errVinculo != nil ||
		actor.validarPara(decision.PrincipalID, auditoria.ActorID) != nil ||
		recurso.Referencia != "consulta-accesos:sha256:"+huella ||
		decision.Accion != AccionAuditoriaConsultaAccesos ||
		decision.RecursoRef != recurso.Referencia ||
		decision.ModuloID != recurso.ModuloID ||
		decision.TipoRecurso != recurso.Tipo ||
		decision.ContextoRecursoHuellaSHA256 != huellaContexto ||
		decision.Finalidad != filtro.FinalidadDeLaConsulta ||
		decision.GarantiaMinima != vecdomain.AuthAssuranceHigh ||
		vinculo.GarantiaObservada != vecdomain.AuthAssuranceHigh ||
		(vinculo.Superficie !=
			vecdomain.SuperficieAutenticacionInternaCorporativaV1 &&
			vinculo.Superficie !=
				vecdomain.SuperficieAutenticacionAdministracionPrivilegiadaV1) ||
		vinculo.MetodoObservado != auditoria.AuthMethod ||
		vinculo.GarantiaObservada != auditoria.AuthAssurance ||
		decision.DecisionRef != auditoria.AuthorizationRef ||
		decision.CorrelacionRef != auditoria.CorrelationRef ||
		len(decision.Obligaciones) != 0 ||
		!listasAccesoIguales(
			decision.CamposPermitidos,
			camposAutorizacionConsultaAccesos[:],
		) {
		return ErrConsultaAdministrativaAccesosDenegada
	}
	return nil
}

func listasAccesoIguales(primera, segunda []string) bool {
	if len(primera) != len(segunda) {
		return false
	}
	for indice := range primera {
		if primera[indice] != segunda[indice] {
			return false
		}
	}
	return true
}

func resultadoAccesoValido(resultado string) bool {
	switch resultado {
	case "permitido", "denegado", "error":
		return true
	default:
		return false
	}
}

func instanteConsultaAccesosValido(instante time.Time) bool {
	return !instante.IsZero() &&
		instante.Location() == time.UTC &&
		instante.Nanosecond()%1000 == 0 &&
		instante.Year() >= 1 && instante.Year() <= 9999
}

func identificadorAccesoValido(valor string, maximo int) bool {
	return textoAccesoValido(valor, maximo, false) &&
		!strings.ContainsAny(valor, "*?")
}

func textoAccesoValido(
	valor string,
	maximo int,
	permiteEspacios bool,
) bool {
	if valor == "" || valor != strings.TrimSpace(valor) ||
		len(valor) > maximo || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) ||
			(!permiteEspacios && unicode.IsSpace(caracter)) {
			return false
		}
	}
	return true
}

func huellaOpcionalAccesoValida(huella string) bool {
	if huella == "" {
		return true
	}
	if len(huella) != sha256.Size*2 || huella == strings.Repeat("0", 64) {
		return false
	}
	return esHexadecimalMinusculo(huella)
}

func huellaAccesoRequeridaValida(huella string) bool {
	if len(huella) != sha256.Size*2 {
		return false
	}
	return esHexadecimalMinusculo(huella)
}

func entradaConfirmadaCoincide(
	solicitada vecdomain.AuditEntry,
	confirmada vecdomain.AuditEntry,
) bool {
	return ValidarEntradaRegistroAcceso(solicitada) == nil &&
		confirmada.ID != "" &&
		confirmada.Seq > 0 &&
		confirmada.IntegrityAlgorithm == "sha256-chain-v1" &&
		huellaAccesoRequeridaValida(confirmada.Signature) &&
		((confirmada.Seq == 1 &&
			confirmada.PrevSignature == strings.Repeat("0", 64)) ||
			(confirmada.Seq > 1 &&
				huellaAccesoRequeridaValida(confirmada.PrevSignature))) &&
		confirmada.ActorID == solicitada.ActorID &&
		confirmada.ActorProfile == solicitada.ActorProfile &&
		strings.Join(confirmada.ActorRoles, "\x00") ==
			strings.Join(solicitada.ActorRoles, "\x00") &&
		confirmada.RepresentedSubjectID == solicitada.RepresentedSubjectID &&
		confirmada.AuthMethod == solicitada.AuthMethod &&
		confirmada.AuthAssurance == solicitada.AuthAssurance &&
		confirmada.AuthorizationRef == solicitada.AuthorizationRef &&
		confirmada.Purpose == solicitada.Purpose &&
		confirmada.Action == solicitada.Action &&
		confirmada.ModuleID == solicitada.ModuleID &&
		confirmada.SubjectRef == solicitada.SubjectRef &&
		confirmada.ObjectVersion == solicitada.ObjectVersion &&
		confirmada.ExpedienteRef == solicitada.ExpedienteRef &&
		confirmada.DocumentRef == solicitada.DocumentRef &&
		confirmada.RuleRef == solicitada.RuleRef &&
		confirmada.Reason == solicitada.Reason &&
		confirmada.Result == solicitada.Result &&
		confirmada.BeforeHash == solicitada.BeforeHash &&
		confirmada.AfterHash == solicitada.AfterHash &&
		confirmada.CorrelationRef == solicitada.CorrelationRef &&
		confirmada.OccurredAt.Equal(solicitada.OccurredAt) &&
		mapasAccesoIguales(confirmada.Metadata, solicitada.Metadata)
}

func mapasAccesoIguales(primero, segundo map[string]string) bool {
	if len(primero) != len(segundo) {
		return false
	}
	for clave, valor := range primero {
		if segundo[clave] != valor {
			return false
		}
	}
	return true
}

func clonarEntradaAuditoriaAccesos(
	entrada vecdomain.AuditEntry,
) vecdomain.AuditEntry {
	entrada.ActorRoles = append([]string(nil), entrada.ActorRoles...)
	if entrada.Metadata != nil {
		entrada.Metadata = make(map[string]string, len(entrada.Metadata))
		for clave, valor := range entrada.Metadata {
			entrada.Metadata[clave] = valor
		}
	}
	return entrada
}
