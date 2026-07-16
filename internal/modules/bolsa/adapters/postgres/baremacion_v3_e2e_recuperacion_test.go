package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func escribirEstadoBolsaPostgreSQLE2E(
	t *testing.T,
	ruta string,
	estado estadoBolsaPostgreSQLE2E,
	reemplazar bool,
) {
	t.Helper()
	documento, err := json.Marshal(estado)
	if err != nil {
		t.Fatalf("serializar estado E2E: %v", err)
	}
	defer borrarBytesE2E(documento)
	rutaEscritura := ruta
	banderas := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if reemplazar {
		validarFicheroEstadoBolsaPostgreSQLE2E(t, ruta)
		rutaEscritura = ruta + ".tmp"
	}
	fichero, err := os.OpenFile(rutaEscritura, banderas, 0o600)
	if err != nil {
		t.Fatalf("crear estado E2E 0600: %v", err)
	}
	cerrarConError := func(errEscritura error) {
		_ = fichero.Close()
		_ = os.Remove(rutaEscritura)
		t.Fatalf("persistir estado E2E: %v", errEscritura)
	}
	if _, err = fichero.Write(documento); err != nil {
		cerrarConError(err)
	}
	if err = fichero.Sync(); err != nil {
		cerrarConError(err)
	}
	if err = fichero.Close(); err != nil {
		_ = os.Remove(rutaEscritura)
		t.Fatalf("cerrar estado E2E: %v", err)
	}
	validarFicheroEstadoBolsaPostgreSQLE2E(t, rutaEscritura)
	if reemplazar {
		if err = os.Rename(rutaEscritura, ruta); err != nil {
			_ = os.Remove(rutaEscritura)
			t.Fatalf("reemplazar estado E2E atomicamente: %v", err)
		}
		validarFicheroEstadoBolsaPostgreSQLE2E(t, ruta)
	}
	directorio, err := os.Open(filepath.Dir(ruta))
	if err != nil {
		t.Fatalf("abrir directorio del estado E2E para fsync: %v", err)
	}
	if err = directorio.Sync(); err != nil {
		_ = directorio.Close()
		t.Fatalf("hacer durable el reemplazo del estado E2E: %v", err)
	}
	if err = directorio.Close(); err != nil {
		t.Fatalf("cerrar directorio del estado E2E: %v", err)
	}
}

func leerEstadoBolsaPostgreSQLE2E(t *testing.T, ruta string) estadoBolsaPostgreSQLE2E {
	t.Helper()
	validarFicheroEstadoBolsaPostgreSQLE2E(t, ruta)
	documento, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("leer estado E2E: %v", err)
	}
	defer borrarBytesE2E(documento)
	decodificador := json.NewDecoder(bytes.NewReader(documento))
	decodificador.DisallowUnknownFields()
	var estado estadoBolsaPostgreSQLE2E
	if err = decodificador.Decode(&estado); err != nil {
		t.Fatalf("decodificar estado E2E cerrado: %v", err)
	}
	var resto any
	if err = decodificador.Decode(&resto); !errors.Is(err, io.EOF) {
		t.Fatalf("estado E2E contiene datos adicionales: %v", err)
	}
	if estado.Esquema != "vec.pruebas.bolsa.postgresql-v3-e2e.v3" ||
		(estado.Etapa != etapaConfirmadaPendienteReplayPostgreSQLE2E &&
			estado.Etapa != etapaConfirmadaReproducidaPostgreSQLE2E) ||
		estado.Ancla.IsZero() || estado.InstanteUso.IsZero() ||
		len(estado.HuellaTokenSHA256) != sha256.Size*2 ||
		estado.VersionBase.Validar() != nil || estado.AgregadoObjetivo.Validar() != nil ||
		estado.ManifiestoHistorico.Validar() != nil || estado.Manifiesto.Validar() != nil ||
		estado.Trazabilidad.Validar() != nil || estado.AutorizacionReservaRef == "" ||
		estado.AutorizacionPrevalidacionRef == "" ||
		estado.AutorizacionConfirmacionRef == "" || estado.ConfirmadaEn.IsZero() ||
		!estado.ConfirmadaEn.Equal(estado.InstanteUso) ||
		estado.HuellaSolicitudReservaHMAC == "" ||
		estado.HuellaSolicitudConfirmacionHMAC == "" ||
		estado.ManifiestoHistorico.SelloManifiestoHMACSHA256 ==
			estado.Manifiesto.SelloManifiestoHMACSHA256 {
		t.Fatal("estado E2E semanticamente invalido")
	}
	baseHistorica, err := referenciaBaseManifiesto(estado.AgregadoObjetivo, 0)
	baseObjetivo, errObjetivo := referenciaBaseManifiesto(estado.AgregadoObjetivo, 1)
	_, errorHuellaToken := hex.DecodeString(estado.HuellaTokenSHA256)
	if errorHuellaToken != nil || err != nil || errObjetivo != nil ||
		len(estado.AgregadoObjetivo.Decisiones) != 2 ||
		baseObjetivo != estado.VersionBase ||
		estado.VersionBase.Numero != 2 || estado.ManifiestoHistorico.VersionBase != 1 ||
		estado.Manifiesto.VersionBase != estado.VersionBase.Numero ||
		estado.ManifiestoHistorico.ValidarCoberturaFirmaPara(
			baseHistorica, estado.AgregadoObjetivo.Decisiones[0].Contenido,
			estado.AgregadoObjetivo.Decisiones[0].Firma,
		) != nil || estado.Manifiesto.ValidarCoberturaFirmaPara(
		estado.VersionBase, estado.AgregadoObjetivo.Decisiones[1].Contenido,
		estado.AgregadoObjetivo.Decisiones[1].Firma,
	) != nil {
		t.Fatal("estado E2E no conserva los dos manifiestos ligados a V2/V3")
	}
	return estado
}

func exigirEstadoConfirmadoBolsaPostgreSQLE2E(
	t *testing.T,
	estado estadoBolsaPostgreSQLE2E,
	etapaEsperada string,
) {
	t.Helper()
	if estado.Etapa != etapaEsperada || estado.VersionConfirmada == nil {
		t.Fatal("estado confirmado carece de etapa o version durable")
	}
	versionConfirmada := *estado.VersionConfirmada
	evidencia := puertosbolsa.EvidenciaTransaccionBaremacion{
		AuditoriaRef: estado.AuditoriaRef, HuellaAuditoriaSHA256: estado.HuellaAuditoriaSHA256,
		EventoOutboxRef:          estado.EventoOutboxRef,
		HuellaEventoOutboxSHA256: estado.HuellaEventoOutboxSHA256,
		ConfirmadaEn:             estado.ConfirmadaFiableEn,
	}
	if versionConfirmada.Validar() != nil ||
		versionConfirmada.BaremacionMeritoRef != estado.AgregadoObjetivo.ID ||
		versionConfirmada.Numero != estado.VersionBase.Numero+1 || evidencia.Validar() != nil {
		t.Fatal("estado confirmado contiene referencias o evidencia invalidas")
	}
}

func validarFicheroEstadoBolsaPostgreSQLE2E(t *testing.T, ruta string) {
	t.Helper()
	informacion, err := os.Lstat(ruta)
	if err != nil {
		t.Fatalf("inspeccionar estado E2E: %v", err)
	}
	if !informacion.Mode().IsRegular() || informacion.Mode().Perm() != 0o600 {
		t.Fatalf("estado E2E no es fichero regular 0600: modo=%s", informacion.Mode())
	}
	estadistica, correcto := informacion.Sys().(*syscall.Stat_t)
	if !correcto || int(estadistica.Uid) != os.Geteuid() {
		t.Fatal("estado E2E no pertenece al uid del proceso")
	}
}

func reconstruirReservaBolsaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	entorno entornoAutorizacionBolsaPostgreSQLE2E,
	estado estadoBolsaPostgreSQLE2E,
) puertosbolsa.SolicitudReservarCambioBaremacion {
	t.Helper()
	contextoReserva := entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionReservarDecisionBaremacion,
		estado.AgregadoObjetivo.ID,
		estado.AutorizacionReservaRef, false,
		estado.InstanteUso,
	)
	if contextoReserva.Proyeccion().AutorizacionRef != estado.AutorizacionReservaRef {
		t.Fatal("la autorizacion de reserva rehidratada no coincide con el estado durable")
	}
	solicitud := sellarReservaBolsaPostgreSQLE2E(t, puertosbolsa.SolicitudReservarCambioBaremacion{
		Contexto: contextoReserva, Clase: puertosbolsa.ClaseCambioIncorporarDecision,
		ClaveIdempotencia:   "idempotencia:e2e:postgresql:v3:decision:2",
		BaremacionMeritoRef: estado.AgregadoObjetivo.ID,
		VersionEsperada:     &estado.VersionBase,
		SolicitadaEn:        estado.InstanteUso,
		ExpiraEn:            estado.InstanteUso.Add(5 * time.Minute),
	})
	if solicitud.HuellaSolicitudHMAC != estado.HuellaSolicitudReservaHMAC {
		t.Fatal("la reserva exacta reconstruida cambio su HMAC tras el reinicio")
	}
	return solicitud
}

func ejecutarFaseConfirmacionBolsaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	poolAdmin *pgxpool.Pool,
	rutaEstado string,
) {
	t.Helper()
	estado := leerEstadoBolsaPostgreSQLE2E(t, rutaEstado)
	exigirEstadoConfirmadoBolsaPostgreSQLE2E(
		t, estado, etapaConfirmadaPendienteReplayPostgreSQLE2E,
	)
	entorno := nuevoEntornoAutorizacionBolsaPostgreSQLE2E(
		t, ctx, pool, poolAdmin, estado.Ancla, false,
	)
	observadorSellos := &observadorSellosBolsaPostgreSQLE2E{}
	repositorio, err := NuevoRepositorioBaremaciones(
		pool,
		relojBolsaPostgreSQLE2E{ahora: estado.InstanteUso},
		protectorSellosBolsaPostgreSQLE2E{observador: observadorSellos},
	)
	if err != nil {
		t.Fatalf("recrear repositorio tras reinicio: %v", err)
	}
	solicitud := reconstruirReservaBolsaPostgreSQLE2E(t, ctx, entorno, estado)
	resultado, err := reservarCambioDiagnosticadoBolsaPostgreSQLE2E(
		ctx, repositorio, poolAdmin, solicitud,
	)
	if err != nil {
		t.Fatalf("reproducir resultado confirmado por reserva idempotente tras reinicio: %v", err)
	}
	if resultado.ValidarPara(solicitud) != nil || !resultado.Repetida ||
		resultado.VersionConfirmada == nil ||
		resultado.VersionConfirmada.Referencia != *estado.VersionConfirmada ||
		!observadorSellos.contieneTodos(
			estado.ManifiestoHistorico.SelloManifiestoHMACSHA256,
			estado.Manifiesto.SelloManifiestoHMACSHA256,
		) {
		t.Fatal("la reserva idempotente no reprodujo la V3 exacta ni verifico todo el archivo")
	}
	estado.Etapa = etapaConfirmadaReproducidaPostgreSQLE2E
	exigirConfirmacionV3DurableBolsaPostgreSQLE2E(t, ctx, poolAdmin, estado)
	// Estas decisiones se registran en una fase independiente y solo se
	// consumen en la fase de recuperacion. La lectura funcional nunca usa admin.
	instanteLecturas := instanteAutoritativoBolsaPostgreSQLE2E(t, ctx, pool)
	_ = entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionConsultarBaremacionVigente,
		estado.AgregadoObjetivo.ID,
		referenciaDecisionAutorizacionBolsaPostgreSQLE2E("lectura:vigente"), true,
		instanteLecturas,
	)
	_ = entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionConsultarVersionBaremacion,
		estado.AgregadoObjetivo.ID,
		referenciaDecisionAutorizacionBolsaPostgreSQLE2E("lectura:version"), true,
		instanteLecturas,
	)
	_ = entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion,
		estado.AuditoriaRef,
		referenciaDecisionAutorizacionBolsaPostgreSQLE2E("lectura:evidencia"), true,
		instanteLecturas,
	)
	escribirEstadoBolsaPostgreSQLE2E(t, rutaEstado, estado, true)
}

func ejecutarFaseRecuperacionBolsaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	poolAdmin *pgxpool.Pool,
	rutaEstado string,
) {
	t.Helper()
	estado := leerEstadoBolsaPostgreSQLE2E(t, rutaEstado)
	exigirEstadoConfirmadoBolsaPostgreSQLE2E(
		t, estado, etapaConfirmadaReproducidaPostgreSQLE2E,
	)
	versionEsperada := *estado.VersionConfirmada
	observadorSellos := &observadorSellosBolsaPostgreSQLE2E{}
	protector := protectorSellosBolsaPostgreSQLE2E{observador: observadorSellos}
	entorno := nuevoEntornoAutorizacionBolsaPostgreSQLE2E(
		t, ctx, pool, poolAdmin, estado.Ancla, false,
	)
	repositorio, err := NuevoRepositorioBaremaciones(
		pool,
		relojBolsaPostgreSQLE2E{ahora: estado.InstanteUso},
		protector,
	)
	if err != nil {
		t.Fatalf("recrear repositorio para recuperacion: %v", err)
	}
	solicitudReserva := reconstruirReservaBolsaPostgreSQLE2E(t, ctx, entorno, estado)
	repetido, err := reservarCambioDiagnosticadoBolsaPostgreSQLE2E(
		ctx, repositorio, poolAdmin, solicitudReserva,
	)
	if err != nil {
		t.Fatalf("reproducir reserva confirmada tras segundo reinicio: %v", err)
	}
	if repetido.ValidarPara(solicitudReserva) != nil || !repetido.Repetida ||
		repetido.VersionConfirmada == nil ||
		repetido.VersionConfirmada.Referencia != versionEsperada {
		t.Fatal("el segundo replay idempotente no devolvio la version original")
	}
	instanteLecturas := instanteAutoritativoBolsaPostgreSQLE2E(t, ctx, pool)
	repositorioLecturas, err := NuevoRepositorioBaremaciones(
		pool,
		relojBolsaPostgreSQLE2E{ahora: instanteLecturas},
		protector,
	)
	if err != nil {
		t.Fatalf("crear repositorio con reloj fresco para lecturas: %v", err)
	}
	contextoVigente := entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionConsultarBaremacionVigente,
		estado.AgregadoObjetivo.ID,
		referenciaDecisionAutorizacionBolsaPostgreSQLE2E("lectura:vigente"), false,
		instanteLecturas,
	)
	vigente, err := repositorioLecturas.ObtenerVersionVigente(
		ctx,
		puertosbolsa.SolicitudObtenerBaremacionVigente{
			Contexto: contextoVigente, BaremacionMeritoRef: estado.AgregadoObjetivo.ID,
		},
	)
	if err != nil {
		t.Fatalf("leer vigente por adaptador real tras reinicio: %v", err)
	}
	contextoVersion := entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionConsultarVersionBaremacion,
		estado.AgregadoObjetivo.ID,
		referenciaDecisionAutorizacionBolsaPostgreSQLE2E("lectura:version"), false,
		instanteLecturas,
	)
	version, err := repositorioLecturas.ObtenerVersion(
		ctx,
		puertosbolsa.SolicitudObtenerVersionBaremacion{
			Contexto: contextoVersion, BaremacionMeritoRef: estado.AgregadoObjetivo.ID,
			Numero: versionEsperada.Numero,
		},
	)
	if err != nil {
		t.Fatalf("leer version exacta por adaptador real tras reinicio: %v", err)
	}
	contextoEvidencia := entorno.autorizarEn(
		t, ctx, puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion,
		estado.AuditoriaRef,
		referenciaDecisionAutorizacionBolsaPostgreSQLE2E("lectura:evidencia"), false,
		instanteLecturas,
	)
	evidencia, err := repositorioLecturas.ObtenerEvidenciaTransaccion(
		ctx,
		puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion{
			Contexto: contextoEvidencia, BaremacionMeritoRef: estado.AgregadoObjetivo.ID,
			NumeroVersion: versionEsperada.Numero,
			AuditoriaRef:  estado.AuditoriaRef, EventoOutboxRef: estado.EventoOutboxRef,
		},
	)
	if err != nil {
		t.Fatalf("recuperar evidencia transaccional por adaptador real: %v", err)
	}
	if vigente.Referencia != versionEsperada || version.Referencia != versionEsperada ||
		evidencia.Version.Referencia != versionEsperada ||
		evidencia.Evidencia.AuditoriaRef != estado.AuditoriaRef ||
		evidencia.Evidencia.HuellaAuditoriaSHA256 != estado.HuellaAuditoriaSHA256 ||
		evidencia.Evidencia.EventoOutboxRef != estado.EventoOutboxRef ||
		evidencia.Evidencia.HuellaEventoOutboxSHA256 != estado.HuellaEventoOutboxSHA256 ||
		evidencia.Manifiesto == nil ||
		evidencia.Manifiesto.Referencia != estado.Manifiesto.Referencia ||
		evidencia.Manifiesto.HuellaManifiestoSHA256 != estado.Manifiesto.HuellaManifiestoSHA256 {
		t.Fatal("version/evidencia recuperada no coincide exactamente con el estado durable")
	}
	verificarManifiestoRecuperadoBolsaPostgreSQLE2E(t, estado.ManifiestoHistorico, version, 0)
	verificarManifiestoRecuperadoBolsaPostgreSQLE2E(t, *evidencia.Manifiesto, version, 1)
	if !observadorSellos.contieneTodos(
		estado.ManifiestoHistorico.SelloManifiestoHMACSHA256,
		estado.Manifiesto.SelloManifiestoHMACSHA256,
	) {
		t.Fatal("la recuperacion no verifico los dos sellos historicos del archivo V3")
	}
	exigirConfirmacionV3DurableBolsaPostgreSQLE2E(t, ctx, poolAdmin, estado)
}

func verificarManifiestoRecuperadoBolsaPostgreSQLE2E(
	t *testing.T,
	manifiesto puertosbolsa.ManifiestoProbatorioBaremacion,
	version puertosbolsa.VersionBaremacion,
	indiceDecision int,
) {
	t.Helper()
	if len(manifiesto.Autorizaciones) != 18 || len(manifiesto.Evidencias) != 16 ||
		manifiesto.Validar() != nil {
		t.Fatal("manifiesto recuperado no conserva cobertura exacta 18/16")
	}
	if indiceDecision < 0 || indiceDecision >= len(version.Agregado.Decisiones) {
		t.Fatal("version recuperada carece de la decision tecnica esperada")
	}
	decision := version.Agregado.Decisiones[indiceDecision]
	base := puertosbolsa.ReferenciaVersionBaremacion{
		BaremacionMeritoRef: manifiesto.BaremacionMeritoRef,
		Numero:              manifiesto.VersionBase,
		HuellaEstadoSHA256:  manifiesto.HuellaVersionBaseSHA256,
	}
	if err := manifiesto.ValidarCoberturaFirmaPara(base, decision.Contenido, decision.Firma); err != nil {
		t.Fatalf("manifiesto recuperado no cubre la firma: %v", err)
	}
	artefactos, err := puertosbolsa.ArtefactosCanonicosManifiestoProbatorioBaremacion(manifiesto)
	if err != nil {
		t.Fatalf("reconstruir artefactos canonicos recuperados: %v", err)
	}
	contenido := artefactos.ContenidoSinHuella.Revelar()
	representacion := artefactos.RepresentacionSellada.Revelar()
	preimagen := artefactos.PreimagenHMAC.Revelar()
	defer borrarBytesE2E(contenido, representacion, preimagen)
	if huellaSHA256BytesE2E(contenido) != manifiesto.HuellaManifiestoSHA256 {
		t.Fatal("contenido canonico recuperado no reproduce la huella del manifiesto")
	}
	protector := protectorSellosBolsaPostgreSQLE2E{}
	if err = protector.VerificarSelloBaremacion(context.Background(), puertosbolsa.SolicitudVerificarSelloBaremacion{
		Finalidad:              puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3,
		RepresentacionCanonica: artefactos.RepresentacionSellada,
		SelloHMAC:              manifiesto.SelloManifiestoHMACSHA256,
	}); err != nil {
		t.Fatalf("sello del manifiesto recuperado no autentico: %v", err)
	}
	material, err := (puertosbolsa.SolicitudVerificarSelloBaremacion{
		Finalidad:              puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3,
		RepresentacionCanonica: artefactos.RepresentacionSellada,
		SelloHMAC:              manifiesto.SelloManifiestoHMACSHA256,
	}).MaterialCanonicoHMAC()
	if err != nil {
		t.Fatalf("reconstruir preimagen HMAC durable: %v", err)
	}
	materialCanonico := material.Revelar()
	defer borrarBytesE2E(materialCanonico)
	if !bytes.Equal(preimagen, materialCanonico) {
		t.Fatalf("preimagen HMAC durable divergente: %v", err)
	}
}
