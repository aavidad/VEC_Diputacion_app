package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/contactopropio"
	"vec-diputacion-granada/internal/modules/usuarios"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	seguridaddoc "vec-diputacion-granada/internal/vec/adapters/documentos/seguridad"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	vecapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

const (
	contactoPGAdminDSN    = "VEC_F2_CONTACTO_PG_ADMIN_DSN"
	contactoPGContextoDSN = "VEC_F2_CONTACTO_PG_CONTEXTO_DSN"
	contactoPGFuenteDSN   = "VEC_F2_CONTACTO_PG_FUENTE_DSN"
	contactoPGRegistroDSN = "VEC_F2_CONTACTO_PG_REGISTRO_DSN"
	contactoPGMotivosDSN  = "VEC_F2_CONTACTO_PG_MOTIVOS_DSN"
	contactoPGWriterDSN   = "VEC_F2_CONTACTO_PG_WRITER_DSN"
)

type sesionContactoPGPrueba struct {
	vinculo   domain.VinculoAutenticacionActorV2
	resultado domain.ResultadoContextoActorRegistradoV2
}

func (s sesionContactoPGPrueba) ResolverContactoPropio(context.Context) (domain.VinculoAutenticacionActorV2, domain.ResultadoContextoActorRegistradoV2, error) {
	return s.vinculo, s.resultado, nil
}

func abrirPoolContactoPGPrueba(t *testing.T, ctx context.Context, variable, rol string) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv(variable)
	if dsn == "" {
		t.Fatal("falta DSN de ensayo F2 aislado")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("pool F2 no disponible")
	}
	t.Cleanup(pool.Close)
	var base, login string
	var version int
	if err := pool.QueryRow(ctx, `SELECT current_database(),session_user,current_setting('server_version_num')::int`).Scan(&base, &login, &version); err != nil || !strings.HasPrefix(base, "vec_f2_") || version < 180000 || version >= 190000 || login != rol {
		t.Fatal("pool F2 fuera de contenedor nominal PostgreSQL 18")
	}
	if rol != "postgres" {
		var exclusiva bool
		if err := pool.QueryRow(ctx, `SELECT session_user=current_user AND
            (SELECT count(*)=1 FROM pg_auth_members WHERE member=session_user::regrole) AND
            (SELECT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
               FROM pg_roles WHERE rolname=session_user)`).Scan(&exclusiva); err != nil || !exclusiva {
			t.Fatal("LOGIN F2 no tiene membresía nominal exclusiva")
		}
	}
	return pool
}

func configuracionMaterialContactoPGPrueba(t *testing.T) config.Config {
	t.Helper()
	directorio := os.Getenv("VEC_F2_CONTACTO_MATERIAL_EFIMERO")
	if directorio == "" {
		cfg, _ := generarMaterialDesarrolloPrueba(t)
		return cfg
	}
	raiz, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal("raíz de ensayo F2 no disponible")
	}
	return config.Config{
		Address:                   "127.0.0.1:0",
		ExecutionProfile:          config.ExecutionProfileDevelopment,
		AuthMode:                  config.AuthModeDevelopment,
		DevelopmentGuard:          config.DevelopmentGuardAcknowledgement,
		DevelopmentMaterialDir:    directorio,
		PersonalCatalogPath:       "memory",
		BolsaPublicSourcePath:     filepath.Join(raiz, config.DefaultBolsaPublicSourcePath),
		BolsaCategoriesSourcePath: filepath.Join(raiz, config.DefaultBolsaCategoriesSourcePath),
	}.Normalize()
}

// Sólo se habilita desde el runner de esquema sintético, en contenedor PG18
// desechable sin red. ContextoActor, PDP, registro V3, COSE, confianza y
// consumidor SQL son los adaptadores reales; la identidad inicial es fixture.
func TestContactoPropioPG18MaterialFirmadoYConsumoNominal(t *testing.T) {
	if os.Getenv("VEC_F2_CONTACTO_PG18_DESECHABLE") != "1" {
		t.Skip("requiere runner F2 PG18 desechable")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancelar()
	admin := abrirPoolContactoPGPrueba(t, ctx, contactoPGAdminDSN, "postgres")
	contexto := abrirPoolContactoPGPrueba(t, ctx, contactoPGContextoDSN, "vec_contacto_f2_contexto_login")
	fuente := abrirPoolContactoPGPrueba(t, ctx, contactoPGFuenteDSN, "vec_contacto_f2_fuente_login")
	registro := abrirPoolContactoPGPrueba(t, ctx, contactoPGRegistroDSN, "vec_contacto_f2_registro_login")
	motivos := abrirPoolContactoPGPrueba(t, ctx, contactoPGMotivosDSN, "vec_contacto_f2_motivos_login")
	writer := abrirPoolContactoPGPrueba(t, ctx, contactoPGWriterDSN, "vec_contacto_f2_login")
	var vacia bool
	if err := admin.QueryRow(ctx, `SELECT NOT EXISTS(SELECT 1 FROM vec_contacto_usuario_v1.versiones)
        AND NOT EXISTS(SELECT 1 FROM vec_contacto_usuario_v1.actual)
        AND NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version)
        AND to_regclass('vec_contacto_usuario_v1.operaciones') IS NULL
        AND to_regprocedure('vec_contacto_usuario_v1.registrar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NOT NULL`).Scan(&vacia); err != nil || !vacia {
		t.Fatal("preimagen F2 de prueba contaminada")
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	vinculo, resultado := contextoRegistradoContactoPGPrueba(t, ctx, contexto, ahora)
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(resultado.Contexto.Principal.ID, resultado.Contexto.PerfilActivoRef, ahora,
		"contacto_propio_f2_prueba", "Contacto propio sintético", "contacto-propio-f2-prueba",
		[]domain.ConcesionRol{
			{Accion: vecapp.AccionAltaContactoUsuario, ModuloID: usuarios.ModuleID, TipoRecurso: "contacto_usuario", Finalidades: []string{contactopropio.FinalidadRegistro}, GarantiaMinima: domain.AuthAssuranceHigh},
			{Accion: vecapp.AccionConsultarContactoUsuario, ModuloID: usuarios.ModuleID, TipoRecurso: "contacto_usuario", Finalidades: []string{contactopropio.FinalidadRegistro}, GarantiaMinima: domain.AuthAssuranceHigh},
		},
		[]domain.AmbitoPerfil{{Clave: "persona_ref", Valores: []string{resultado.Contexto.PersonaRef}}})
	if err != nil {
		t.Fatal("rol de contacto sintético inválido")
	}
	autoridad := autoridadPostgreSQLDesarrollo{pool: admin, vinculo: vinculo, prefijoBloqueo: "vec:f2:prueba:autorizacion:", actoControlRol: "acto:f2:prueba:control", actoAsignacion: "acto:f2:prueba:asignacion", actoSesion: "acto:f2:prueba:sesion"}
	instantanea, err = autoridad.prepararInstantanea(ctx, instantanea, true)
	if err != nil || autoridad.publicarInstantanea(ctx, instantanea) != nil {
		t.Fatal("autoridad PostgreSQL no publicó permiso nominal")
	}
	motivo := motivoContactoPropioDesarrollo("contacto-propio-pg-prueba")
	motivoRecibo := motivoContactoPropioDesarrollo("contacto-recibo-pg-prueba")
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, admin, []domain.ReferenciaEntradaCatalogo{motivo, motivoRecibo}, ahora); err != nil {
		t.Fatal("motivo gobernado de contacto no disponible")
	}
	cfg := configuracionMaterialContactoPGPrueba(t)
	seguridad, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal("seguridad de ensayo no disponible")
	}
	t.Cleanup(seguridad.derivadorIdempotencia.borrar)
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(seguridad.derivadorIdempotencia, ahora)
	if err != nil {
		t.Fatal("material de firma de ensayo no disponible")
	}
	t.Cleanup(material.borrarCopiasEfimeras)
	material.audienciaConsumo = contactopropio.AudienciaRegistro
	material.capacidad, err = confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(material.claveHMACID, material.claveHMACVersion, material.claveHMAC, material.emisorID, material.audienciaConsumo,
		confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, material.validaDesde, material.validaHasta, time.Time{}, material.claveHMACRevision, material.claveHMACHuella)
	if err != nil || publicarGobiernoContactoPGPrueba(ctx, admin, material) != nil {
		t.Fatal("gobierno V3 de contacto no disponible")
	}
	materialRecibo := material
	semillaRecibo := sha256.Sum256(append(append([]byte(nil), material.claveHMAC...), []byte("vec.contacto.recibo.prueba.v1")...))
	materialRecibo.claveHMAC = append([]byte(nil), semillaRecibo[:]...)
	t.Cleanup(func() { clear(materialRecibo.claveHMAC) })
	materialRecibo.claveHMACID = "clave:capacidad:contacto-recibo:prueba"
	materialRecibo.claveHMACVersion = 2
	materialRecibo.claveHMACRevision = 2
	materialRecibo.claveHMACOrden = 2
	materialRecibo.emisorID = "emisor:contacto-recibo:prueba"
	materialRecibo.audienciaConsumo = vecapp.AudienciaConsultaReciboContactoUsuario
	huellaSecretoRecibo := sha256.Sum256(materialRecibo.claveHMAC)
	materialRecibo.claveHMACSecreto = hex.EncodeToString(huellaSecretoRecibo[:])
	hGobierno := sha256.New()
	_, _ = hGobierno.Write([]byte("vec.ct.desarrollo.capacidad-v3.gobierno.v1\x00"))
	_, _ = hGobierno.Write(materialRecibo.claveHMAC)
	materialRecibo.claveHMACHuella = hex.EncodeToString(hGobierno.Sum(nil))
	materialRecibo.capacidad, err = confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(materialRecibo.claveHMACID, materialRecibo.claveHMACVersion,
		materialRecibo.claveHMAC, materialRecibo.emisorID, materialRecibo.audienciaConsumo,
		confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, materialRecibo.validaDesde, materialRecibo.validaHasta,
		time.Time{}, materialRecibo.claveHMACRevision, materialRecibo.claveHMACHuella)
	if err != nil || publicarGobiernoContactoPGPrueba(ctx, admin, materialRecibo) != nil {
		t.Fatal("gobierno V3 de recibo no disponible")
	}
	almacenFuente, err := vecpg.NuevoAlmacenAutorizacion(fuente)
	if err != nil {
		t.Fatal("fuente de autorización no disponible")
	}
	almacenRegistro, err := vecpg.NuevoAlmacenAutorizacion(registro)
	if err != nil {
		t.Fatal("registro de autorización no disponible")
	}
	validador, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivos, motivo.CatalogoID)
	if err != nil {
		t.Fatal("validador de motivo no disponible")
	}
	reloj := relojContratacionTemporalDesarrollo{}
	pdp, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(almacenFuente, almacenRegistro, almacenRegistro, validador, reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		t.Fatal("PDP PostgreSQL no disponible")
	}
	proveedor, err := nuevoProveedorMaterialAutorizacionBaseDesarrollo(material, reloj)
	if err != nil {
		t.Fatal("atestación COSE de contacto no disponible")
	}
	emisor, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(pdp, proveedor.atestador, proveedor.confianza, proveedor.emisor)
	if err != nil {
		t.Fatal("emisor V3 de contacto no disponible")
	}
	proveedorRecibo, err := nuevoProveedorMaterialAutorizacionBaseDesarrollo(materialRecibo, reloj)
	if err != nil {
		t.Fatal("atestación COSE de recibo no disponible")
	}
	emisorRecibo, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(pdp, proveedorRecibo.atestador, proveedorRecibo.confianza, proveedorRecibo.emisor)
	if err != nil {
		t.Fatal("emisor V3 de recibo no disponible")
	}
	clave := derivarClaveDesarrollo(seguridad.emisorKMS.claveEnvoltura, "vec.contacto.usuario.auditoria.prueba.v1")
	seudonimizador, err := seguridaddoc.NuevoSelladorHMAC("contacto_pg_prueba_v1", clave[:])
	borrarBytes(clave[:])
	if err != nil {
		t.Fatal("seudonimizador de contacto no disponible")
	}
	servicio, err := contactopropio.NuevoServicio(contactopropio.Dependencias{Sesion: sesionContactoPGPrueba{vinculo, resultado}, FuenteAutorizacion: almacenFuente, Emisor: emisor,
		Seudonimizador: seudonimizador, Protector: seguridad.emisorKMS, Huellas: huellasContactoDesarrollo{derivador: seguridad.derivadorIdempotencia}, PoolEscritor: writer,
		Reloj: reloj, GeneradorCorrelacion: seguridadvec.GeneradorReferenciasCriptograficas{}, PerfilPropioRef: resultado.Contexto.PerfilActivoRef,
		Motivo: motivo, AmbitosRecurso: map[string]string{"persona_ref": resultado.Contexto.PersonaRef}})
	if err != nil {
		t.Fatal("servicio de contacto no disponible")
	}
	recibo, err := servicio.Guardar(ctx, "persona@example.test", 0)
	if err != nil || recibo.ReplayConfirmado || recibo.SujetoRef != resultado.Contexto.PersonaRef || recibo.Version != 1 || recibo.EvidenciaCentral.Referencia == "" {
		t.Fatal("POST contacto firmado no confirmó en PostgreSQL")
	}
	conRecibos, err := contactopropio.NuevoServicioConRecibos(servicio, contactopropio.DependenciasConsultaRecibo{PoolConsulta: writer, Emisor: emisorRecibo, Motivo: motivoRecibo})
	if err != nil {
		t.Fatal("servicio de recibo propio no disponible")
	}
	consultado, err := conRecibos.ConsultarRecibo(ctx, 1)
	if err != nil || !consultado.Encontrado || consultado.Version != 1 || consultado.ReciboOriginal.EvidenciaCentral.Referencia != recibo.EvidenciaCentral.Referencia {
		t.Fatal("GET autorizado no recuperó recibo original")
	}
	repetido, err := servicio.Guardar(ctx, "persona@example.test", 0)
	if err != nil || !repetido.ReplayConfirmado || repetido.EvidenciaCentral.Referencia != recibo.EvidenciaCentral.Referencia || repetido.ConsumoRef != recibo.ConsumoRef {
		t.Fatal("replay semántico no conservó recibo/contacto")
	}
	var versiones, actuales, outbox int
	if err := admin.QueryRow(ctx, `SELECT (SELECT count(*) FROM vec_contacto_usuario_v1.versiones),
        (SELECT count(*) FROM vec_contacto_usuario_v1.actual),
        (SELECT count(*) FROM vec_contacto_usuario_v1.outbox)`).Scan(&versiones, &actuales, &outbox); err != nil || versiones != 1 || actuales != 1 || outbox != 1 {
		t.Fatal("contacto confirmado duplicó o perdió efectos")
	}
}

func contextoRegistradoContactoPGPrueba(t *testing.T, ctx context.Context, pool *pgxpool.Pool, ahora time.Time) (domain.VinculoAutenticacionActorV2, domain.ResultadoContextoActorRegistradoV2) {
	t.Helper()
	reloj := relojContratacionTemporalDesarrollo{}
	resolutor, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pool)
	if err != nil {
		t.Fatal("LOGIN de ContextoActor no acreditado")
	}
	servicio, err := vecapp.NuevoServicioContextoActorProductivoV2(resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
	if err != nil {
		t.Fatal("servicio ContextoActor no disponible")
	}
	autoridad, err := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicio)
	if err != nil {
		t.Fatal("autoridad ContextoActor no disponible")
	}
	cuenta := domain.CuentaAutenticadaContextoActor{CuentaRef: "cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa", Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh}
	solicitud := domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: "prf_sintetico_cccccccccccccccccccccccc"}
	confirmacion, err := servicio.ResolverRegistrado(ctx, solicitud)
	if err != nil {
		t.Fatal("ContextoActor no resolvió fixture maestra")
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{RegistroContextoRef: confirmacion.RegistroContextoRef, Contexto: confirmacion.Contexto,
		RepresentacionCanonica: confirmacion.RepresentacionCanonica, HuellaSHA256: confirmacion.HuellaSHA256,
		ManifiestoProcedenciaCanonico: confirmacion.ManifiestoProcedenciaCanonico, ManifiestoProcedenciaHuellaSHA256: confirmacion.ManifiestoProcedenciaHuellaSHA256,
		AutoridadEfectiva: confirmacion.AutoridadEfectiva, ResueltoEnAutoritativo: confirmacion.ResueltoEnAutoritativo}
	base := "contacto-f2-pg-sintetico"
	aut := domain.AutenticacionRevalidadaV1{AutenticacionRef: referenciaAltaContratacionTemporalDesarrollo("aut_", base+"aut"), AutenticacionHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(base + "aut"),
		AsercionRef: referenciaAltaContratacionTemporalDesarrollo("ase_", base+"ase"), SesionRef: referenciaAltaContratacionTemporalDesarrollo("ses_", base+"ses"),
		ControlSesionRef: referenciaAltaContratacionTemporalDesarrollo("cse_", base+"cse"), ControlSesionRevision: 1, ControlSesionHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(base + "cse"),
		CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: domain.SuperficieAutenticacionExternaPersonalV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia, PoliticaGarantiaRef: referenciaAltaContratacionTemporalDesarrollo("pga_", base+"pga"), PoliticaGarantiaHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(base + "pga"),
		AutenticacionVerificadaEn: ahora.Add(-2 * time.Minute), SesionEmitidaEn: ahora.Add(-2 * time.Minute), SesionRevalidadaEn: ahora.Add(-time.Minute), SesionValidaHasta: ahora.Add(20 * time.Minute)}
	vinculo, fresco, err := domain.CrearVinculoAutenticacionActorV2ConResultado(ctx, revalidadorAutenticacionAltaContratacionTemporalDesarrollo{valor: aut},
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: aut.AutenticacionRef, SesionRef: aut.SesionRef}, autoridad, solicitud, reloj)
	if err != nil || vinculo.ValidarPara(fresco) != nil || fresco.Contexto.PersonaRef != resultado.Contexto.PersonaRef {
		t.Fatal("vínculo V2 no usa ContextoActor PostgreSQL")
	}
	return vinculo, fresco
}

func publicarGobiernoContactoPGPrueba(ctx context.Context, pool *pgxpool.Pool, m materialAtestacionContratacionTemporalDesarrollo) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario`); err != nil {
		return err
	}
	consultas := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
            (clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
            VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'acto:f2:prueba:clave')`, []any{m.claveHMACID, m.claveHMACVersion, m.claveHMACRevision, m.claveHMACHuella, m.claveHMAC, m.claveHMACSecreto, m.emisorID, m.audienciaConsumo, m.validaDesde, m.validaHasta}},
		{`INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision(orden,clave_id,version,establecida_en,acto_ref)
            VALUES($1,$2,$3,$4,'acto:f2:prueba:puntero-clave')`, []any{m.claveHMACOrden, m.claveHMACID, m.claveHMACVersion, m.validaDesde}},
		{`INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
            (clave_id,version,clave_publica_spki,huella_spki_sha256,valida_desde,valida_hasta,suite,audiencia_despliegue,acto_ref)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,'acto:f2:prueba:raiz') ON CONFLICT DO NOTHING`, []any{m.claveID, m.claveVersion, m.spki, m.spkiHuella, m.validaDesde, m.validaHasta, confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, audienciaAtestacionContratacionTemporalDesarrollo}},
		{`INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version(revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
			VALUES($1,$2,$3,$4,$5,'acto:f2:prueba:configuracion') ON CONFLICT DO NOTHING`, []any{m.configuracionRef, m.configuracionOrden, m.configuracionHuella, m.publicadaEn, m.expiraEn}},
		{`INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz(configuracion_revision,raiz_clave_id,raiz_version) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, []any{m.configuracionRef, m.claveID, m.claveVersion}},
		{`INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual(orden,configuracion_revision,establecida_en,acto_ref)
			VALUES($1,$2,$3,'acto:f2:prueba:puntero-configuracion') ON CONFLICT DO NOTHING`, []any{m.configuracionOrden, m.configuracionRef, m.publicadaEn}},
	}
	for _, q := range consultas {
		if _, err = tx.Exec(ctx, q.sql, q.args...); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
