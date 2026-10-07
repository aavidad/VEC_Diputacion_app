package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteCatalogoGobiernoReferenciaPrueba struct{}

func (fuenteCatalogoGobiernoReferenciaPrueba) ObtenerCatalogoAccionesAdministracionV1(
	context.Context, string, int, string,
) (domain.CatalogoAccionesAdministracionV1, error) {
	return domain.CatalogoAccionesAdministracionV1{}, nil
}

type emisorGobiernoReferenciaPrueba struct {
	t     *testing.T
	ahora time.Time
}

func (e emisorGobiernoReferenciaPrueba) EmitirGobiernoRolNuevo(_ context.Context,
	actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles,
	instantanea domain.InstantaneaAutorizacion, efecto Efecto,
) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.t.Helper()
	recurso, err := RecursoGobiernoRolNuevo(efecto, instantanea.AsignacionPerfil)
	if err != nil {
		e.t.Fatalf("recurso de prueba invalido: %v", err)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		e.t.Fatalf("huella de recurso de prueba invalida: %v", err)
	}
	resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:gobierno:prueba", strings.Repeat("a", 64), strings.Repeat("b", 64),
		evidencia.ResultadoContexto.RegistroContextoRef, evidencia.ResultadoContexto.HuellaSHA256,
		efecto.Accion, recurso.Referencia, huella, efecto.Audiencia, e.ahora, e.ahora.Add(4*time.Second),
	)
	if err != nil {
		e.t.Fatalf("resumen V3 de prueba invalido: %v", err)
	}
	raiz, err := x509.MarshalPKIXPublicKey(ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)))
	if err != nil {
		e.t.Fatal(err)
	}
	material, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		make([]byte, 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"),
		actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion,
		[]byte("payload"), []byte("sobre"), []byte("evidencia"), raiz,
	)
	if err != nil {
		e.t.Fatalf("material V3 de prueba invalido: %v", err)
	}
	return material, nil
}

type txGobiernoReferenciaPrueba struct {
	pgx.Tx
	fila                          pgx.Row
	consultas, commits, rollbacks int
}

func (t *txGobiernoReferenciaPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (t *txGobiernoReferenciaPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	t.consultas++
	return t.fila
}

func (t *txGobiernoReferenciaPrueba) Commit(context.Context) error {
	t.commits++
	return nil
}

func (t *txGobiernoReferenciaPrueba) Rollback(context.Context) error {
	t.rollbacks++
	return nil
}

type poolGobiernoReferenciaPrueba struct {
	tx        *txGobiernoReferenciaPrueba
	comienzos int
	opciones  pgx.TxOptions
}

func (p *poolGobiernoReferenciaPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	p.comienzos++
	p.opciones = opciones
	return p.tx, nil
}

func (*poolGobiernoReferenciaPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaFalsa{err: errors.New("consulta fuera de la transaccion")}
}

func TestCerrarGobiernoRolPorReferenciaRechazaConfirmacionFuturaAntesCommit(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	futuro := ahora.Add(time.Second)
	solicitud, respuesta := cierreGobiernoReferenciaPrueba(t, ahora, futuro)
	b, err := json.Marshal(respuesta)
	if err != nil {
		t.Fatal(err)
	}
	tx := &txGobiernoReferenciaPrueba{fila: filaFalsa{dato: b}}
	pool := &poolGobiernoReferenciaPrueba{tx: tx}
	autoridad := &AutoridadGobiernoRolNuevo{
		pool:   pool,
		fuente: fuenteCatalogoGobiernoReferenciaPrueba{},
		emisor: emisorGobiernoReferenciaPrueba{t: t, ahora: ahora},
		reloj:  relojFijo(ahora),
	}

	cierre, err := autoridad.CerrarGobiernoRolPorReferencia(context.Background(), solicitud)
	if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
		t.Fatalf("confirmacion futura aceptada: cierre=%+v err=%v", cierre, err)
	}
	if cierre.OperacionRef != "" || cierre.Recibo != nil {
		t.Fatalf("se devolvio salida provisional: %+v", cierre)
	}
	if pool.comienzos != 1 || tx.consultas != 1 || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("frontera transaccional: comienzos=%d consultas=%d commits=%d rollbacks=%d",
			pool.comienzos, tx.consultas, tx.commits, tx.rollbacks)
	}
	if pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("opciones transaccionales inesperadas: %+v", pool.opciones)
	}
}

func cierreGobiernoReferenciaPrueba(t *testing.T, ahora, confirmadoEn time.Time) (
	domain.SolicitudCierreGobiernoRolPorReferencia, cierreGobiernoRolRespuesta,
) {
	t.Helper()
	base, _, _, _ := contratoV2Prueba(t)
	base.InstantaneaAutorizacion.AsignacionPerfil.Ambitos = []domain.AmbitoPerfil{
		{Clave: "organizacion_ref", Valores: []string{"organizacion:prueba"}},
		{Clave: "unidad_ref", Valores: []string{"unidad:prueba"}},
	}
	if err := base.InstantaneaAutorizacion.Validar(); err != nil {
		t.Fatalf("instantanea de prueba invalida: %v", err)
	}
	motivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_administracion",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: strings.Repeat("c", 64),
		EntradaClave:         "gobierno_rol_nuevo",
	}
	concesion := domain.ConcesionRol{
		Accion:         "bolsa.carga_convoca.confirmar",
		ModuloID:       "bolsa",
		TipoRecurso:    "carga_convoca",
		Finalidades:    []string{"cargar_bolsa"},
		GarantiaMinima: domain.AuthAssuranceHigh,
	}
	definicion := domain.DefinicionVersionPerfilGobernado{
		RolID: "nuevo_sintetico", Version: 1, Nombre: "Nuevo sintetico",
		Concesiones: []domain.ConcesionRol{concesion},
	}
	plan := domain.PlanGobiernoPerfil{
		Operacion:             domain.OperacionCrearPerfilGobernado,
		CatalogoRef:           "catalogo:acciones:administracion",
		CatalogoVersion:       1,
		CatalogoHuellaSHA256:  strings.Repeat("d", 64),
		VersionRolObjetivoRef: definicion.Referencia(),
		DefinicionNueva:       &definicion,
		Selecciones: []domain.SeleccionAccionAdministracionV1{{
			EntradaRef: "entrada:accion:gobierno", EntradaVersion: 1,
			EntradaHuellaSHA256: strings.Repeat("e", 64),
		}},
		Motivo: motivo, ReferenciaActo: "resolucion:gobierno:prueba",
	}
	material := domain.MaterialPropuestaGobiernoPerfil{
		OperacionRef:         "propuesta_admin:" + strings.Repeat("1", 32),
		ProponentePersonaRef: "per_" + strings.Repeat("2", 22),
		PerfilActivoRef:      "prf_" + strings.Repeat("3", 22),
		AsignacionPerfilRef:  "asignacion:proponente:v1",
		Plan:                 plan,
	}
	huella, err := material.HuellaSHA256()
	if err != nil {
		t.Fatalf("material de propuesta invalido: %v", err)
	}
	solicitud := domain.SolicitudCierreGobiernoRolPorReferencia{
		OperacionRef:            "cierre_admin:" + strings.Repeat("4", 32),
		PropuestaRef:            material.OperacionRef,
		PropuestaHuellaSHA256:   huella,
		Aprobador:               base.Actor,
		Evidencia:               base.Evidencia,
		InstantaneaAutorizacion: base.InstantaneaAutorizacion,
		Decision:                domain.DecisionAprobarPropuestaPerfil,
		Motivo:                  motivo,
		CorrelacionRef:          "correlacion_" + strings.Repeat("5", 32),
	}
	completa, err := solicitud.CompletarCierreGobiernoRolConMaterial(material)
	if err != nil {
		t.Fatalf("solicitud de cierre invalida: %v", err)
	}
	version := domain.VersionRol{
		RolID: definicion.RolID, Version: definicion.Version, Nombre: definicion.Nombre,
		Estado: domain.EstadoVersionRolPublicada, Concesiones: definicion.Concesiones,
		PublicadaPor: solicitud.Aprobador.PersonaRef, PublicadaEn: confirmadoEn,
	}
	control := domain.ControlVigenciaVersionRol{
		VersionRolRef: version.Referencia(), Revision: 1,
		Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
		ActualizadoPor: solicitud.Aprobador.PersonaRef, ActualizadoEn: confirmadoEn,
	}
	respuesta := cierreGobiernoRolRespuesta{
		Estado:                "permitido",
		OperacionRef:          solicitud.OperacionRef,
		PropuestaHuellaSHA256: huella,
		Decision:              solicitud.Decision,
		ConfirmadoEn:          confirmadoEn,
		AuditoriaAccesoRef:    "auditoria:acceso:gobierno:prueba",
		Recibo: reciboGobiernoRolRespuesta{
			ActoRef:             "acto_admin:" + strings.Repeat("6", 32),
			ReciboRef:           "recibo_admin:" + strings.Repeat("7", 32),
			ActorPersonaRef:     solicitud.Aprobador.PersonaRef,
			PerfilActivoRef:     solicitud.Aprobador.PerfilActivoRef,
			AsignacionPerfilRef: solicitud.InstantaneaAutorizacion.AsignacionPerfil.Referencia(),
			CorrelacionRef:      solicitud.CorrelacionRef,
			Motivo:              solicitud.Motivo,
			AuditoriaRef:        "auditoria:gobierno:prueba",
			VersionRol:          version,
			ControlPosterior:    control,
		},
	}
	canon, err := json.Marshal(material)
	if err != nil {
		t.Fatal(err)
	}
	respuesta.MaterialCanon = string(canon)
	respuesta.Cierre = domain.CierreGobiernoPerfil{
		OperacionRef: respuesta.OperacionRef, Material: material,
		PropuestaHuellaSHA256: respuesta.PropuestaHuellaSHA256,
		Decision:              respuesta.Decision,
		ConfirmadoEn:          respuesta.ConfirmadoEn,
		AuditoriaAccesoRef:    respuesta.AuditoriaAccesoRef,
		Recibo:                respuesta.Recibo.Dominio(),
	}
	if err := respuesta.Cierre.ValidarPara(completa); err != nil {
		t.Fatalf("cierre SQL de control no supera el dominio: %v", err)
	}
	if !confirmadoEn.After(ahora) {
		t.Fatal("el escenario no contiene una confirmacion futura")
	}
	return solicitud, respuesta
}
