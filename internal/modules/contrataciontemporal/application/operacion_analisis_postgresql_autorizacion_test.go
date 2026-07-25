package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorIntegracionAnalisisO304 struct {
	ahora         time.Time
	entrada       entradaIntegracionAnalisisO304
	motivo        dominiovec.ReferenciaEntradaCatalogo
	administrador *pgxpool.Pool
	llamadas      int
	errUltimo     error
}

func (a *autorizadorIntegracionAnalisisO304) ExigirSolicitudLigadaV3(
	ctx context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
) (
	decisionRespuesta dominiovec.DecisionAutorizacionLigadaV3,
	confirmacionRespuesta puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	errRespuesta error,
) {
	a.llamadas++
	defer func() { a.errUltimo = errRespuesta }()
	datos, err := solicitud.Datos()
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			err
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			err
	}
	ambitos := make([]dominiovec.AmbitoPerfil, 0, len(datos.Recurso.Ambitos))
	for clave, valor := range datos.Recurso.Ambitos {
		ambitos = append(ambitos, dominiovec.AmbitoPerfil{
			Clave: clave, Valores: []string{valor},
		})
	}
	rol := dominiovec.VersionRol{
		RolID: "o304_go_integracion", Version: 1,
		Nombre: "Integración Go O3-04",
		Estado: dominiovec.EstadoVersionRolPublicada,
		Concesiones: []dominiovec.ConcesionRol{{
			Accion: datos.Accion, ModuloID: datos.Recurso.ModuloID,
			TipoRecurso:      datos.Recurso.Tipo,
			Finalidades:      []string{datos.Finalidad},
			GarantiaMinima:   dominiovec.AuthAssuranceSubstantial,
			CamposPermitidos: []string{"estado"},
			Obligaciones:     []string{"auditar"},
		}},
		PublicadaPor: "autoridad-o304-go",
		PublicadaEn:  a.ahora.Add(-10 * time.Minute),
	}
	control := dominiovec.ControlVigenciaVersionRol{
		VersionRolRef: rol.Referencia(), Revision: 1,
		Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
		ActualizadoPor: rol.PublicadaPor,
		ActualizadoEn:  a.ahora.Add(-5 * time.Minute),
	}
	asignacion := dominiovec.AsignacionPerfil{
		AsignacionID:    a.entrada.AsignacionID,
		Version:         a.entrada.AsignacionVersion,
		PerfilActivoRef: vinculo.PerfilActivoRef,
		PrincipalID:     vinculo.PrincipalID,
		VersionRolRef:   rol.Referencia(),
		Estado:          dominiovec.EstadoAsignacionPerfilActiva,
		Ambitos:         ambitos,
		EmitidaPor:      "autoridad-o304-go",
		EmitidaEn:       a.ahora.Add(-10 * time.Minute),
		VigenteDesde:    a.ahora.Add(-5 * time.Minute),
		VigenteHasta:    a.ahora.Add(30 * time.Minute),
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil:              asignacion,
		VersionRol:                    rol,
		ControlVigenciaVersionRol:     control,
		Politicas:                     a.entrada.Politicas,
		RevisionCatalogoPoliticas:     a.entrada.RevisionCatalogo,
		CatalogoPoliticasHuellaSHA256: a.entrada.HuellaCatalogo,
	}
	evidencia, err := dominiovec.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea,
		"decision:o304:go-integracion",
		a.ahora, a.ahora.Add(90*time.Second),
	)
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			err
	}
	decision, err := dominiovec.NuevaDecisionAutorizacionLigadaV3(
		solicitud, evidencia,
	)
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			err
	}
	orden, err := puertosvec.
		NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
			solicitud, decision, a.motivo, resultado,
		)
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			err
	}
	confirmacion, err := puertosvec.
		RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
			ctx, registroConcesionV3Doble{registradaEn: a.ahora}, orden,
		)
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			err
	}
	if err := a.persistirAutoridad(
		ctx, decision, rol, control, asignacion,
	); err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			err
	}
	return decision, confirmacion, nil
}

func (a *autorizadorIntegracionAnalisisO304) persistirAutoridad(
	ctx context.Context,
	decision dominiovec.DecisionAutorizacionLigadaV3,
	rol dominiovec.VersionRol,
	control dominiovec.ControlVigenciaVersionRol,
	asignacion dominiovec.AsignacionPerfil,
) error {
	decisionCanonica, err := dominiovec.
		RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		return err
	}
	motivoCanonico, err := dominiovec.
		RepresentacionCanonicaMotivoAutorizacionV2(a.motivo)
	if err != nil {
		return err
	}
	rolJSON, err := json.Marshal(rol)
	if err != nil {
		return err
	}
	controlJSON, err := json.Marshal(control)
	if err != nil {
		return err
	}
	asignacionJSON, err := json.Marshal(asignacion)
	if err != nil {
		return err
	}
	huellaRol, err := rol.HuellaSHA256()
	if err != nil {
		return err
	}
	huellaControl, err := control.HuellaSHA256()
	if err != nil {
		return err
	}
	huellaAsignacion, err := asignacion.HuellaSHA256()
	if err != nil {
		return err
	}
	tx, err := a.administrador.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		SET LOCAL search_path = pg_catalog;
		SET LOCAL timezone = 'UTC';
		SET LOCAL statement_timeout = '15s';
		SET LOCAL idle_in_transaction_session_timeout = '20s'`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.version_rol(
			version_rol_ref, rol_id, version, huella_sha256,
			publicada_en, documento
		) VALUES ($1,$2,$3,$4,$5,$6::jsonb)`,
		rol.Referencia(), rol.RolID, rol.Version, huellaRol,
		rol.PublicadaEn, rolJSON,
	)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.control_vigencia_version_rol(
			version_rol_ref, revision, estado, huella_sha256,
			actualizado_en, documento
		) VALUES ($1,$2,$3,$4,$5,$6::jsonb)`,
		rol.Referencia(), control.Revision, control.Estado,
		huellaControl, control.ActualizadoEn, controlJSON,
	)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
		VALUES ($1,$2,clock_timestamp(),'autoridad-o304-go',
		        'acto:control:o304:go')`,
		rol.Referencia(), control.Revision,
	)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.asignacion_perfil(
			asignacion_ref, asignacion_id, version, perfil_activo_ref,
			principal_id, version_rol_ref, huella_sha256,
			emitida_en, documento
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)`,
		asignacion.Referencia(), asignacion.AsignacionID,
		asignacion.Version, asignacion.PerfilActivoRef,
		asignacion.PrincipalID, rol.Referencia(), huellaAsignacion,
		asignacion.EmitidaEn, asignacionJSON,
	)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.asignacion_perfil_actual
		VALUES ($1,$2,clock_timestamp(),'autoridad-o304-go',
		        'acto:asignacion:o304:go')
		ON CONFLICT (perfil_activo_ref) DO UPDATE SET
			asignacion_ref=EXCLUDED.asignacion_ref,
			actualizada_en=EXCLUDED.actualizada_en,
			actualizada_por=EXCLUDED.actualizada_por,
			acto_ref=EXCLUDED.acto_ref`,
		asignacion.PerfilActivoRef, asignacion.Referencia(),
	)
	if err != nil {
		return err
	}
	var concedida bool
	err = tx.QueryRow(ctx, `
		SELECT concedida
		  FROM vec_autorizacion.registrar_decision_contexto_actor_v3(
			$1,$2,$3,$4
		  )`,
		decisionCanonica, motivoCanonico,
		a.entrada.PersonaVersion, a.entrada.PerfilVersion,
	).Scan(&concedida)
	if err != nil {
		return err
	}
	if !concedida {
		return fmt.Errorf(
			"registro de decisión O3-04 no concedió la solicitud",
		)
	}
	return tx.Commit(ctx)
}
