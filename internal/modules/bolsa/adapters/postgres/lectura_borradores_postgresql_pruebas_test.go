package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

var instanteLecturaBorradorPostgreSQLPrueba = time.Date(2026, 7, 18, 9, 0, 0, 20_000_000, time.UTC)

type generadorCorrelacionLecturaPostgreSQLPrueba struct{ referencia string }

func (g generadorCorrelacionLecturaPostgreSQLPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return g.referencia, nil
}

func solicitudLecturaBorradorPostgreSQLPrueba(
	t *testing.T, accion string,
) (gobiernoconvocatorias.ContextoOperacionBorrador, gobiernoconvocatorias.CapacidadLecturaBorrador) {
	t.Helper()
	correlacionRef := "correlacion_0123456789abcdef0123456789abcdef"
	actor, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		instanteLecturaBorradorPostgreSQLPrueba, "per_0123456789abcdefghijkl",
		"prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate,
		dominiovec.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	contexto := gobiernoconvocatorias.ContextoOperacionBorrador{
		Actor: actor, Vinculo: vinculo, CorrelacionRef: correlacionRef,
	}
	organizacion, unidad := "org_diputaciongranada", "uni_seleccionexterna"
	referencia, tipo := "proceso:bolsa:auxiliar-2026-1#1", puertosbolsa.TipoRecursoVersionConvocatoriaGobernada
	if accion == gobiernoconvocatorias.AccionListarBorradoresGobernados {
		referencia, tipo = "borradores:"+organizacion, gobiernoconvocatorias.TipoColeccionBorradoresGobernados
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: referencia, ModuloID: puertosbolsa.ModuloGobiernoConvocatorias,
		Tipo: tipo, Ambitos: map[string]string{
			"organizacion_ref": organizacion, "unidad_gestion_ref": unidad,
		}, Atributos: map[string]string{},
	}
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_rrhh", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("9", 64),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(), generadorCorrelacionLecturaPostgreSQLPrueba{correlacionRef},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV2(
		dominiovec.DatosSolicitudAutorizacionLigadaV2{
			ContextoActor: actor, VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo: motivo, Accion: accion, Recurso: recurso,
			Finalidad:   gobiernoconvocatorias.FinalidadLecturaBorradoresGobernada,
			Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaSolicitud, err := dominiovec.HuellaSHA256SolicitudAutorizacionV2(solicitud)
	huellaMotivo, errMotivo := dominiovec.HuellaSHA256MotivoAutorizacionV2(motivo)
	huellaRecurso, errRecurso := recurso.HuellaContextoAutorizacionSHA256()
	huellaCatalogo, errCatalogo := dominiovec.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if errors.Join(err, errMotivo, errRecurso, errCatalogo) != nil {
		t.Fatal(errors.Join(err, errMotivo, errRecurso, errCatalogo))
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "decision:lectura-borrador:postgresql:0001", Concedida: true, Codigo: "concedida",
		PrincipalID: actor.PersonaRef, PerfilActivoRef: actor.PerfilActivoRef,
		Accion: accion, RecursoRef: referencia, ModuloID: puertosbolsa.ModuloGobiernoConvocatorias,
		TipoRecurso: tipo, ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad:              gobiernoconvocatorias.FinalidadLecturaBorradoresGobernada,
		CorrelacionRef:         correlacionRef,
		EsquemaHuellaSolicitud: dominiovec.EsquemaHuellaSolicitudAutorizacionV2,
		SolicitudHuellaSHA256:  huellaSolicitud,
		EsquemaHuellaMotivo:    dominiovec.EsquemaHuellaMotivoAutorizacionV2,
		MotivoHuellaSHA256:     huellaMotivo, VinculoAutenticacionActor: vinculo,
		AsignacionRef: "asignacion:lectura-borrador:v1", AsignacionHuellaSHA256: strings.Repeat("1", 64),
		VersionRolRef: "rol:lectura-borrador:v1", VersionRolHuellaSHA256: strings.Repeat("2", 64),
		ControlVigenciaVersionRolRef: "rol:lectura-borrador:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("3", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		CamposPermitidos: []string{"version_convocatoria"},
		EmitidaEn:        instanteLecturaBorradorPostgreSQLPrueba.Add(-time.Second),
		ValidaHasta:      instanteLecturaBorradorPostgreSQLPrueba.Add(time.Minute),
	}
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
		decision, instanteLecturaBorradorPostgreSQLPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	return contexto, gobiernoconvocatorias.CapacidadLecturaBorrador{
		Solicitud: solicitud, Evidencia: evidencia, Motivo: motivo, Recurso: recurso,
		OrganizacionRef: organizacion, UnidadGestionRef: unidad,
		AtestacionRef: "atestacion:lectura-borrador:postgresql:0001", VersionAtestacion: 1,
		EstadoAtestacion: "activa", HuellaAtestacionSHA256: strings.Repeat("4", 64),
	}
}

type iniciadorLecturaBorradorPostgreSQLPrueba struct {
	tx       pgx.Tx
	inicios  int
	opciones pgx.TxOptions
}

func (i *iniciadorLecturaBorradorPostgreSQLPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	i.inicios++
	i.opciones = opciones
	return i.tx, nil
}

type transaccionLecturaBorradorPostgreSQLPrueba struct {
	pgx.Tx
	fila            pgx.Row
	consulta        string
	configuraciones int
	confirmaciones  int
	reversiones     int
	cerrada         bool
}

func (t *transaccionLecturaBorradorPostgreSQLPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	t.configuraciones++
	return pgconn.NewCommandTag("SELECT 1"), nil
}
func (t *transaccionLecturaBorradorPostgreSQLPrueba) QueryRow(_ context.Context, consulta string, _ ...any) pgx.Row {
	t.consulta = consulta
	return t.fila
}
func (t *transaccionLecturaBorradorPostgreSQLPrueba) Commit(context.Context) error {
	t.confirmaciones++
	t.cerrada = true
	return nil
}
func (t *transaccionLecturaBorradorPostgreSQLPrueba) Rollback(context.Context) error {
	if !t.cerrada {
		t.reversiones++
		t.cerrada = true
	}
	return nil
}

type filaLecturaBorradorPostgreSQLPrueba struct{ fila filaBorradorCifrado }

func (f filaLecturaBorradorPostgreSQLPrueba) Scan(destinos ...any) error {
	if len(destinos) != 15 {
		return errors.New("columnas de borrador inesperadas")
	}
	binarios := []struct {
		indice int
		valor  []byte
	}{{0, f.fila.metadatos}, {1, f.fila.aad}, {3, f.fila.perfil}, {7, f.fila.envuelto}, {10, f.fila.nonce}, {11, f.fila.cifrado}, {13, f.fila.atestacion}, {14, f.fila.procedencia}}
	for _, binario := range binarios {
		*(destinos[binario.indice].(*[]byte)) = append([]byte(nil), binario.valor...)
	}
	*(destinos[2].(*string)) = f.fila.huellaAAD
	*(destinos[4].(*string)) = f.fila.esqEnv
	*(destinos[5].(*string)) = f.fila.clave
	*(destinos[6].(*int64)) = f.fila.versionClave
	*(destinos[8].(*string)) = f.fila.huellaEnv
	*(destinos[9].(*string)) = f.fila.esqSobre
	*(destinos[12].(*string)) = f.fila.huellaSobre
	return nil
}

type descifradorBloqueadoLecturaPostgreSQLPrueba struct {
	tx        *transaccionLecturaBorradorPostgreSQLPrueba
	iniciado  chan struct{}
	vioCommit bool
}

func (d *descifradorBloqueadoLecturaPostgreSQLPrueba) DescifrarBorrador(
	ctx context.Context, _ gobiernoconvocatorias.SolicitudDescifradoBorradorDurable,
) (gobiernoconvocatorias.ResultadoDescifradoBorradorDurable, error) {
	d.vioCommit = d.tx == nil || (d.tx.cerrada && d.tx.confirmaciones == 1 && d.tx.reversiones == 0)
	if d.iniciado != nil {
		select {
		case <-d.iniciado:
		default:
			close(d.iniciado)
		}
	}
	<-ctx.Done()
	return gobiernoconvocatorias.ResultadoDescifradoBorradorDurable{}, ctx.Err()
}

func firmaAtestacionLecturaPostgreSQLPrueba([]byte) ([]byte, error) {
	return []byte("firma-atestacion-kms-lectura-postgresql"), nil
}

func bytesJSONLecturaPostgreSQLPrueba(t *testing.T, valor any) []byte {
	t.Helper()
	contenido, err := json.Marshal(valor)
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

func huellaLecturaPostgreSQLPrueba(datos []byte) string {
	suma := sha256.Sum256(datos)
	return hex.EncodeToString(suma[:])
}

const aadLecturaPostgreSQLPrueba = `{"esquema":"bolsa.convocatoria.borrador.aad.v1","version_ref":"proceso:bolsa:auxiliar-2026-1#1","version_revision":1,"huella_version_sha256":"e01dd122f3253ef5d8a3b703a761ef20b2d2adb58e710b44f22f6faef76b9959","esquema_material":"bolsa.convocatoria.intencion.v2","huella_material_sha256":"26e5644ee191ea7d0d4f248a37f257bb374b58c5b37b3682de8a8b68d7ab0a39","perfil_cifrado_ref":"perfil:cifrado:borradores:v1","perfil_cifrado_version":1,"huella_perfil_cifrado_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","algoritmo_aead":"A256GCM","algoritmo_envoltura_clave":"A256KW","evidencia_perfil_ref":"evidencia:resolucion-perfil-cifrado:001","evidencia_perfil_version":1,"huella_evidencia_perfil_sha256":"12a2dfb98bc4a93490d886e29ea955206ecea48f6abc3cc07d7878eee696bcab","decision_politica_ref":"decision:politica-cifrado:borradores:001","decision_politica_version":1,"huella_decision_politica_sha256":"088e07100bbdd18f6d0a2cae71d630bbe386f8083f8c5ce09a7956214b2e569c","localizador_esquema":2,"localizador_dominio":"localizador","localizador_clave_ref":"clave:hmac:convocatorias:localizador:v3","localizador_generacion":3,"localizador_hmac_sha256":"21a19f3f0f62d2b33d347cf9c6f64db62ef6d4c156ccae13aacab673095823fe","huella_solicitud_esquema":2,"huella_solicitud_dominio":"huella_solicitud","huella_solicitud_clave_ref":"clave:hmac:convocatorias:huella:v3","huella_solicitud_generacion":3,"huella_solicitud_hmac_sha256":"f456f3f649430b0ce1348db1f3d96a27593c6282c0d562ab8d3620299b2c9927","revision_diario":1,"cercado_diario":1,"arrendamiento_inicia_en":"2026-07-18T09:00:00.007Z","arrendamiento_vence_en":"2026-07-18T09:02:00.007Z","atestacion_sellado_ref":"atestacion:motivo:001","atestacion_sellado_version":1,"huella_atestacion_sellado_sha256":"5555555555555555555555555555555555555555555555555555555555555555","token_consumo_sellado_ref":"consumo:motivo:001","huella_correlacion_sha256":"b7f3f6944cba8392fdc5245ffb3ea2def4fc6e7ca132ffd635e894b3c11eba93","procedencia_esquema":"vec.acto.procedencia.v1","perfil_ejecucion":"pruebas","autoridad_acto":"autoritativo","proveedor_procedencia_ref":"proveedor-pruebas","migrable_produccion":true}`

func filaCifradaLecturaPostgreSQLPrueba(t *testing.T) filaBorradorCifrado {
	t.Helper()
	aad := []byte(strings.ReplaceAll(aadLecturaPostgreSQLPrueba, `\"`, `"`))
	huellaAAD := huellaLecturaPostgreSQLPrueba(aad)
	perfil, err := gobiernoconvocatorias.NuevoPerfilCifradoBorrador(
		"perfil:cifrado:borradores:v1", 1, strings.Repeat("a", 64), "A256GCM", "A256KW",
	)
	procedencia, errProcedencia := gobiernoconvocatorias.NuevaProcedenciaActoBorrador(
		"pruebas", gobiernoconvocatorias.AutoridadActoAutoritativa, "proveedor-pruebas", true,
	)
	envoltura, errEnvoltura := gobiernoconvocatorias.NuevaEnvolturaClaveKMSBorrador(
		perfil, "clave:kms:borradores:v1", 1, []byte("0123456789abcdef"), huellaAAD,
	)
	sobre, errSobre := gobiernoconvocatorias.NuevoSobreCifradoAEADBorrador(
		perfil, []byte("012345678901"), []byte("0123456789abcdef"), huellaAAD,
	)
	if errors.Join(err, errProcedencia, errEnvoltura, errSobre) != nil {
		t.Fatal(errors.Join(err, errProcedencia, errEnvoltura, errSobre))
	}
	esqEnv, _ := envoltura.EsquemaParaPersistencia()
	_, clave, versionClave, envuelto, _, huellaEnv, _ := envoltura.DatosParaPersistencia()
	esqSobre, _ := sobre.EsquemaParaPersistencia()
	_, nonce, cifrado, _, huellaSobre, _ := sobre.DatosParaPersistencia()
	atestacion, err := gobiernoconvocatorias.NuevaAtestacionKMSBorrador(
		"atestacion:kms:borrador:lectura:001", 1, perfil, clave, versionClave,
		huellaAAD, huellaEnv, huellaSobre, "verificador:kms:lectura:001", procedencia,
		"Ed25519", strings.Repeat("d", 64), instanteLecturaBorradorPostgreSQLPrueba,
		instanteLecturaBorradorPostgreSQLPrueba.Add(4*time.Minute), firmaAtestacionLecturaPostgreSQLPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, algoritmo, verificador, huellaClave, firma, err := atestacion.DatosParaVerificacionFirma()
	if err != nil {
		t.Fatal(err)
	}
	perfilSQL := perfilCifradoReciboPostgreSQL{
		Referencia: perfil.Referencia, Version: perfil.Version,
		HuellaContenidoSHA256: perfil.HuellaContenidoSHA256,
		AlgoritmoAEAD:         perfil.AlgoritmoAEAD, AlgoritmoEnvolturaClave: perfil.AlgoritmoEnvolturaClave,
	}
	procedenciaSQL := procedenciaReciboPostgreSQL{
		Esquema: procedencia.Esquema, PerfilEjecucion: procedencia.PerfilEjecucion,
		Autoridad: procedencia.Autoridad, ProveedorRef: procedencia.ProveedorRef,
		MigrableProduccion: procedencia.MigrableProduccion,
	}
	atestacionSQL := atestacionKMSBorradorPostgreSQL{
		Esquema: atestacion.Esquema, AtestacionRef: atestacion.AtestacionRef,
		VersionAtestacion: atestacion.VersionAtestacion, Estado: atestacion.Estado,
		Perfil: perfilSQL, ClaveMaestraRef: clave, VersionClave: versionClave,
		HuellaAAD: huellaAAD, HuellaEnvolturaSHA256: huellaEnv, HuellaSobreSHA256: huellaSobre,
		VerificadorRef: atestacion.VerificadorRef, Procedencia: procedenciaSQL,
		Firma: firmaEvidenciaReciboPostgreSQL{
			AlgoritmoFirma: algoritmo, VerificadorRef: verificador,
			HuellaClavePublicaSHA256: huellaClave,
			HuellaPreimagenSHA256:    atestacion.Firma.HuellaPreimagenSHA256,
			FirmaBase64URL:           base64.RawURLEncoding.EncodeToString(firma),
		}, EmitidaEn: atestacion.EmitidaEn.UTC().Format(formatoInstanteMicrosegundo),
		ValidaHasta: atestacion.ValidaHasta.UTC().Format(formatoInstanteMicrosegundo),
	}
	estado := puertosbolsa.ReferenciaEstadoVersionConvocatoria{
		Referencia: "proceso:bolsa:auxiliar-2026-1#1", Revision: 1,
		HuellaEstadoSHA256: "e01dd122f3253ef5d8a3b703a761ef20b2d2adb58e710b44f22f6faef76b9959",
	}
	metadatos := metadatosBorrador{Estado: estado, ETag: `"1-` + estado.HuellaEstadoSHA256 + `"`}
	return filaBorradorCifrado{
		metadatos: bytesJSONLecturaPostgreSQLPrueba(t, metadatos), aad: aad,
		huellaAAD: huellaAAD, perfil: bytesJSONLecturaPostgreSQLPrueba(t, perfilSQL),
		esqEnv: esqEnv, clave: clave, versionClave: int64(versionClave), envuelto: envuelto,
		huellaEnv: huellaEnv, esqSobre: esqSobre, nonce: nonce, cifrado: cifrado,
		huellaSobre: huellaSobre, atestacion: bytesJSONLecturaPostgreSQLPrueba(t, atestacionSQL),
		procedencia: bytesJSONLecturaPostgreSQLPrueba(t, procedenciaSQL),
	}
}
