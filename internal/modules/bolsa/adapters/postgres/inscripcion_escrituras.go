package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

const consultaPresentarInscripcion = `SELECT vec_bolsa_llamamientos.solicitar_inscripcion_v1(
	$1::text,$2::jsonb,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::bytea,
	$8::numeric,$9::numeric,$10::bytea,$11::bytea,$12::bytea,$13::bytea)`

const consultaDecidirInscripcion = `SELECT vec_bolsa_llamamientos.revisar_inscripcion_v1(
	$1::text,$2::jsonb,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::bytea,
	$8::numeric,$9::numeric,$10::bytea,$11::bytea,$12::bytea,$13::bytea)`

const consultaIncorporarInscripcion = `SELECT vec_bolsa_llamamientos.incorporar_inscripcion_v1(
	$1::text,$2::jsonb,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::bytea,
	$8::numeric,$9::numeric,$10::bytea,$11::bytea,$12::bytea,$13::bytea)`

type salidaPresentacionInscripcion struct {
	SolicitudRef     string     `json:"solicitud_ref"`
	ReciboRef        string     `json:"recibo_ref"`
	ConvocatoriaRef  string     `json:"convocatoria_ref"`
	CategoriaRef     string     `json:"categoria_ref"`
	Categoria        string     `json:"categoria"`
	DeclaracionRef   string     `json:"declaracion_ref"`
	Estado           string     `json:"estado"`
	Version          uint64     `json:"version"`
	RegistradaEn     time.Time  `json:"registrada_en"`
	DecididaEn       *time.Time `json:"decidida_en"`
	MotivoCodigo     string     `json:"motivo_codigo"`
	MotivoEtiqueta   string     `json:"motivo_etiqueta"`
	Repetida         *bool      `json:"repetida"`
	AuditoriaRef     string     `json:"auditoria_ref"`
	BasesRef         string     `json:"bases_ref"`
	BolsaRef         *string    `json:"bolsa_ref"`
	ParticipacionRef string     `json:"participacion_ref"`
	IncorporadaEn    *time.Time `json:"incorporada_en"`
}

func (r *RepositorioInscripcionesPostgreSQL) Incorporar(ctx context.Context, actor inscripcion.Actor, i inscripcion.Incorporacion) (inscripcion.Recibo, error) {
	if r == nil || valorNulo(r.interno) || i.Validar() != nil || !actor.EscrituraValida() {
		return inscripcion.Recibo{}, inscripcion.ErrSolicitudInvalida
	}
	captura, err := capturaEscrituraInscripcion(actor, inscripcion.AccionIncorporar,
		"vec_bolsa_llamamientos.inscripcion.incorporar.v1", "interna_corporativa")
	if err != nil {
		return inscripcion.Recibo{}, err
	}
	material, _, err := inscripcion.MaterialIncorporacion(i)
	if err != nil {
		return inscripcion.Recibo{}, err
	}
	recurso := actor.RecursoEscrituraCanonico
	huellaRecurso := sha256.Sum256(recurso)
	resumen := actor.MaterialEscritura.ResumenCapacidad()
	if resumen.EfectoRef() != i.SolicitudRef || resumen.EfectoHuellaSHA256() != hex.EncodeToString(huellaRecurso[:]) {
		return inscripcion.Recibo{}, inscripcion.ErrAccesoDenegado
	}
	argumentos := append([]any{string(material), captura, recurso}, argumentosV3Inscripcion(actor)...)
	var salida salidaPresentacionInscripcion
	_, err = transaccionInscripcion(ctx, r.interno, func(tx pgx.Tx) ([]byte, error) {
		var respuesta []byte
		err := tx.QueryRow(ctx, consultaIncorporarInscripcion, argumentos...).Scan(&respuesta)
		return respuesta, err
	}, func(respuesta []byte) error {
		salida = salidaPresentacionInscripcion{}
		if decodificarInscripcionEstricta(respuesta, &salida) != nil ||
			salida.SolicitudRef != i.SolicitudRef || salida.Estado != inscripcion.EstadoIncorporada ||
			salida.Version != i.VersionEsperada+1 || salida.DecididaEn == nil ||
			salida.DecididaEn.IsZero() || salida.RegistradaEn.IsZero() ||
			salida.ReciboRef == "" || salida.Categoria == "" || salida.AuditoriaRef == "" || salida.Repetida == nil ||
			salida.ParticipacionRef == "" {
			return inscripcion.ErrNoDisponible
		}
		return nil
	})
	if err != nil {
		return inscripcion.Recibo{}, err
	}
	recibo := inscripcion.Recibo{Solicitud: inscripcion.Solicitud{
		SolicitudRef: salida.SolicitudRef, ReciboRef: salida.ReciboRef,
		ConvocatoriaRef: salida.ConvocatoriaRef, CategoriaRef: salida.CategoriaRef,
		DeclaracionRef: salida.DeclaracionRef,
		BolsaRef:       salida.BolsaRef, Categoria: salida.Categoria,
		Estado: salida.Estado, Version: salida.Version,
		RegistradaEn: salida.RegistradaEn.UTC(), DecididaEn: salida.DecididaEn,
		BasesRef: salida.BasesRef, ParticipacionRef: salida.ParticipacionRef,
	}, Repetida: *salida.Repetida}
	if recibo.Solicitud.Validar() != nil {
		return inscripcion.Recibo{}, inscripcion.ErrNoDisponible
	}
	return recibo, nil
}

func (r *RepositorioInscripcionesPostgreSQL) Decidir(ctx context.Context, actor inscripcion.Actor, d inscripcion.Decision) (inscripcion.Recibo, error) {
	if r == nil || valorNulo(r.interno) || d.Validar() != nil || !actor.EscrituraValida() {
		return inscripcion.Recibo{}, inscripcion.ErrSolicitudInvalida
	}
	captura, err := capturaEscrituraInscripcion(actor, inscripcion.AccionDecidir,
		"vec_bolsa_llamamientos.inscripcion.revisar.v1", "interna_corporativa")
	if err != nil {
		return inscripcion.Recibo{}, err
	}
	material, _, err := inscripcion.MaterialDecision(d)
	if err != nil {
		return inscripcion.Recibo{}, err
	}
	recurso := actor.RecursoEscrituraCanonico
	huellaRecurso := sha256.Sum256(recurso)
	resumen := actor.MaterialEscritura.ResumenCapacidad()
	if resumen.EfectoRef() != d.SolicitudRef || resumen.EfectoHuellaSHA256() != hex.EncodeToString(huellaRecurso[:]) {
		return inscripcion.Recibo{}, inscripcion.ErrAccesoDenegado
	}
	argumentos := append([]any{string(material), captura, recurso}, argumentosV3Inscripcion(actor)...)
	var salida salidaPresentacionInscripcion
	_, err = transaccionInscripcion(ctx, r.interno, func(tx pgx.Tx) ([]byte, error) {
		var respuesta []byte
		err := tx.QueryRow(ctx, consultaDecidirInscripcion, argumentos...).Scan(&respuesta)
		return respuesta, err
	}, func(respuesta []byte) error {
		salida = salidaPresentacionInscripcion{}
		estado := inscripcion.EstadoAdmitidaAConvocatoria
		if d.Tipo == "rechazar" {
			estado = inscripcion.EstadoRechazada
		}
		if decodificarInscripcionEstricta(respuesta, &salida) != nil ||
			salida.SolicitudRef != d.SolicitudRef || salida.Estado != estado ||
			salida.Version != d.VersionEsperada+1 || salida.DecididaEn == nil ||
			salida.DecididaEn.IsZero() || salida.RegistradaEn.IsZero() ||
			salida.ReciboRef == "" || salida.Categoria == "" || salida.AuditoriaRef == "" || salida.Repetida == nil ||
			(d.Tipo == "rechazar" && (salida.MotivoCodigo != d.MotivoCodigo || salida.MotivoEtiqueta == "")) ||
			(d.Tipo == "admitir" && salida.MotivoCodigo != "") {
			return inscripcion.ErrNoDisponible
		}
		return nil
	})
	if err != nil {
		return inscripcion.Recibo{}, err
	}
	recibo := inscripcion.Recibo{Solicitud: inscripcion.Solicitud{
		SolicitudRef: salida.SolicitudRef, ReciboRef: salida.ReciboRef,
		ConvocatoriaRef: salida.ConvocatoriaRef, CategoriaRef: salida.CategoriaRef,
		DeclaracionRef: salida.DeclaracionRef,
		Categoria:      salida.Categoria, Estado: salida.Estado, Version: salida.Version,
		RegistradaEn: salida.RegistradaEn.UTC(), DecididaEn: salida.DecididaEn,
		BasesRef: salida.BasesRef, MotivoCodigo: salida.MotivoCodigo,
		MotivoEtiqueta: salida.MotivoEtiqueta,
	}, Repetida: *salida.Repetida}
	if recibo.Solicitud.Validar() != nil {
		return inscripcion.Recibo{}, inscripcion.ErrNoDisponible
	}
	return recibo, nil
}

func (r *RepositorioInscripcionesPostgreSQL) Presentar(ctx context.Context, actor inscripcion.Actor, p inscripcion.Presentacion) (inscripcion.Recibo, error) {
	if r == nil || p.Validar() != nil || !actor.EscrituraValida() {
		return inscripcion.Recibo{}, inscripcion.ErrSolicitudInvalida
	}
	var ejecutor iniciadorTransacciones
	switch actor.Canal {
	case "externa_personal":
		ejecutor = r.externo
	case "interna_corporativa":
		ejecutor = r.interno
	default:
		return inscripcion.Recibo{}, inscripcion.ErrAccesoDenegado
	}
	if valorNulo(ejecutor) {
		return inscripcion.Recibo{}, inscripcion.ErrNoDisponible
	}
	audiencia := "vec_bolsa_llamamientos.inscripcion.presentar.v1"
	if actor.Canal == "interna_corporativa" {
		audiencia = "vec_bolsa_llamamientos.inscripcion.presentar_empleado.v1"
	}
	captura, err := capturaEscrituraInscripcion(actor, inscripcion.AccionPresentar,
		audiencia, actor.Canal)
	if err != nil {
		return inscripcion.Recibo{}, err
	}
	material, _, err := inscripcion.MaterialPresentacion(p)
	if err != nil {
		return inscripcion.Recibo{}, err
	}
	referencia, err := inscripcion.ReferenciaSolicitud(actor.PersonaRef, p)
	if err != nil {
		return inscripcion.Recibo{}, err
	}
	recurso := actor.RecursoEscrituraCanonico
	huellaRecurso := sha256.Sum256(recurso)
	resumen := actor.MaterialEscritura.ResumenCapacidad()
	if resumen.EfectoRef() != referencia || resumen.EfectoHuellaSHA256() != hex.EncodeToString(huellaRecurso[:]) {
		return inscripcion.Recibo{}, inscripcion.ErrAccesoDenegado
	}
	argumentos := append([]any{string(material), captura, recurso}, argumentosV3Inscripcion(actor)...)
	var salida salidaPresentacionInscripcion
	_, err = transaccionInscripcion(ctx, ejecutor, func(tx pgx.Tx) ([]byte, error) {
		var respuesta []byte
		err := tx.QueryRow(ctx, consultaPresentarInscripcion, argumentos...).Scan(&respuesta)
		return respuesta, err
	}, func(respuesta []byte) error {
		salida = salidaPresentacionInscripcion{}
		if decodificarInscripcionEstricta(respuesta, &salida) != nil ||
			salida.SolicitudRef != referencia || salida.ConvocatoriaRef != p.ConvocatoriaRef ||
			salida.CategoriaRef != p.CategoriaRef || salida.Estado != inscripcion.EstadoPendiente ||
			salida.Version != 1 || salida.ReciboRef == "" || salida.AuditoriaRef == "" || salida.Repetida == nil ||
			salida.RegistradaEn.IsZero() || salida.Categoria == "" {
			return inscripcion.ErrNoDisponible
		}
		return nil
	})
	if err != nil {
		return inscripcion.Recibo{}, err
	}
	recibo := inscripcion.Recibo{Solicitud: inscripcion.Solicitud{
		SolicitudRef: salida.SolicitudRef, ReciboRef: salida.ReciboRef,
		ConvocatoriaRef: salida.ConvocatoriaRef, CategoriaRef: salida.CategoriaRef,
		DeclaracionRef: salida.DeclaracionRef,
		Categoria:      salida.Categoria, Estado: salida.Estado, Version: salida.Version,
		RegistradaEn: salida.RegistradaEn.UTC(), BasesRef: salida.BasesRef,
	}, Repetida: *salida.Repetida}
	if recibo.Solicitud.Validar() != nil {
		return inscripcion.Recibo{}, inscripcion.ErrNoDisponible
	}
	return recibo, nil
}
