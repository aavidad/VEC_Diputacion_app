package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Prueba de contrato Go↔SQL de CT115/CT116. Solo se ejecuta dentro del
// ensayo desechable (probar_ct115_ct116_cese_modificacion_pg18.sh con
// VEC_CT115_GO=1), que instala las migraciones, la fixture y los dobles de
// las fachadas AD3: prueba la transacción CT, no la criptografía V3.
func TestSeguimientoPostgreSQLContratoGoSQL(t *testing.T) {
	dsn := os.Getenv("VEC_CT115_PG_DSN")
	if dsn == "" {
		t.Skip("solo en el ensayo PostgreSQL 18 desechable de CT115/CT116")
	}
	org, expA, expB := os.Getenv("VEC_CT115_ORG"), os.Getenv("VEC_CT115_EXP_A"), os.Getenv("VEC_CT115_EXP_B")
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo, err := NuevoRepositorioOperacionSeguimientoPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	politica := func(ref string) ports.PoliticaOperacionSeguimiento {
		ahora := time.Now().UTC().Truncate(time.Microsecond)
		return ports.PoliticaOperacionSeguimiento{DefinicionRef: ref, DefinicionVersion: 1, DefinicionHuellaSHA256: strings.Repeat("c", 64),
			MotivoAutorizacion: vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_seguimiento_ct", CatalogoVersion: 1,
				CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("1", 32)},
			EvaluadaEn: ahora.Add(-time.Second), ValidaHasta: ahora.Add(4 * time.Minute)}
	}

	// ------------------------------------------------ cese
	fecha := time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC)
	cese := ports.MaterialCese{OrganizacionRef: org, ExpedienteRef: expA, ActorRef: "per_ct115_go", PerfilRef: "prf_ct115_go", VersionEsperada: 7,
		ClaveIdempotencia: "11111111-1111-4111-8111-111111111111", Datos: domain.DatosCese{CausaClave: "fin_sustitucion", FechaEfecto: fecha,
			JustificanteTipo: "comunicacion_reincorporacion", JustificanteRef: "documento:ct115:go", JustificanteSHA256: strings.Repeat("a", 64),
			Observaciones: "Reincorporación de la titular"}}
	sellosCese := sellosPrueba(t, ports.OperacionRegistrarCese, "cese-1")
	prep, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionRegistrarCese, cese, sellosCese, refsPrueba("cese-1"))
	if err != nil || prep.Confirmada || prep.IncorporacionRef == "" {
		t.Fatalf("preparar cese: %+v %v", prep, err)
	}
	instante := time.Now().UTC().Truncate(time.Microsecond)
	siguiente, err := prep.Expediente.RegistrarCese(7, cese.Datos, domain.DatosActuacion{AccionClave: domain.AccionCesarNombramiento, ActorRef: cese.ActorRef,
		UnidadRef: prep.Expediente.Asignacion.UnidadRef, ReciboRef: prep.Referencias.ReciboRef, RealizadaEn: instante, FaseDestino: domain.FaseNombramiento,
		EstadoDestino: domain.EstadoEnCurso, Observaciones: cese.Datos.Observaciones, DocumentosRef: []string{cese.Datos.JustificanteRef}})
	if err != nil {
		t.Fatal(err)
	}
	p := politica("causas_cese_contratacion_temporal")
	contexto := ports.ContextoAutorizadoSeguimiento{Ambitos: ambitosPrueba(org, expA), Atributos: map[string]string{
		"version_expediente": "7", "causa_clave": "fin_sustitucion", "fecha_efecto": "2027-02-15", "justificante_tipo": "comunicacion_reincorporacion",
		"justificante_ref": "documento:ct115:go", "justificante_sha256": strings.Repeat("a", 64), "observaciones_huella_sha256": huellaPrueba("Reincorporación de la titular"),
		"incorporacion_ref": prep.IncorporacionRef, "politica_ref": p.DefinicionRef, "politica_version": "1", "politica_huella_sha256": p.DefinicionHuellaSHA256,
		"ambito_idempotencia_hmac": prep.AmbitoIdempotenciaHMAC, "huella_peticion_hmac": prep.HuellaPeticionHMAC}}
	orden := ports.OrdenConfirmarOperacionSeguimiento{Operacion: ports.OperacionRegistrarCese, Material: cese, Preparacion: prep, Siguiente: siguiente,
		Politica: p, Contexto: contexto, InstanteEfecto: instante, Accion: domain.AccionCesarNombramiento, Finalidad: ports.FinalidadRegistrarCese,
		Audiencia: ports.AudienciaConsumoCeseV1}
	orden.Autorizacion = exportacionPrueba(t, orden, expA, cese.ActorRef, cese.PerfilRef)
	recibo, err := repo.ConfirmarOperacionSeguimiento(ctx, orden)
	if err != nil || recibo.VersionResultante != 8 || recibo.CausaClave != "fin_sustitucion" || !recibo.RegistradaEn.Equal(instante) {
		t.Fatalf("confirmar cese: %+v %v", recibo, err)
	}
	otra, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionRegistrarCese, cese, sellosCese, refsPrueba("cese-otra"))
	if err != nil || !otra.Confirmada || otra.Recibo == nil || otra.Recibo.ReciboRef != recibo.ReciboRef {
		t.Fatalf("recuperación del cese: %+v %v", otra, err)
	}
	cambiada := cese
	cambiada.Datos.CausaClave = "renuncia"
	if _, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionRegistrarCese, cambiada, sellosCese, refsPrueba("cese-cambiada")); !errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
		t.Fatalf("clave reutilizada: %v", err)
	}

	// ------------------------------------------------ cierre
	confirmada := time.Date(2027, 2, 16, 0, 0, 0, 0, time.UTC)
	cierre := ports.MaterialCierreExpediente{OrganizacionRef: org, ExpedienteRef: expA, ActorRef: "per_ct115_go", PerfilRef: "prf_ct115_go", VersionEsperada: 8,
		ClaveIdempotencia: "22222222-2222-4222-8222-222222222222", Datos: domain.DatosCierreExpediente{Condiciones: []string{"cese_registrado", "ginpix_confirmado"},
			GINPIXNumero: "GX-2027-0099", GINPIXConfirmadaEn: &confirmada}}
	prepC, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionCerrarExpediente, cierre, sellosPrueba(t, ports.OperacionCerrarExpediente, "cierre-1"), refsPrueba("cierre-1"))
	if err != nil || prepC.CeseReciboRef != recibo.ReciboRef {
		t.Fatalf("preparar cierre: %+v %v", prepC, err)
	}
	instante = time.Now().UTC().Truncate(time.Microsecond)
	siguiente, err = prepC.Expediente.CerrarTrasCese(8, cierre.Datos, domain.DatosActuacion{AccionClave: domain.AccionCerrarExpediente, ActorRef: cierre.ActorRef,
		UnidadRef: prepC.Expediente.Asignacion.UnidadRef, ReciboRef: prepC.Referencias.ReciboRef, RealizadaEn: instante, FaseDestino: domain.FaseNombramiento,
		EstadoDestino: domain.EstadoCompletado, DocumentosRef: []string{"ginpix:GX-2027-0099"}})
	if err != nil {
		t.Fatal(err)
	}
	p = politica("vec.contratacion_temporal.reglas")
	orden = ports.OrdenConfirmarOperacionSeguimiento{Operacion: ports.OperacionCerrarExpediente, Material: cierre, Preparacion: prepC, Siguiente: siguiente,
		Politica: p, InstanteEfecto: instante, Accion: domain.AccionCerrarExpediente, Finalidad: ports.FinalidadCerrarExpediente,
		Audiencia: ports.AudienciaConsumoCierreExpedienteV1, Contexto: ports.ContextoAutorizadoSeguimiento{Ambitos: ambitosPrueba(org, expA),
			Atributos: map[string]string{"version_expediente": "8", "cese_recibo_ref": recibo.ReciboRef, "condiciones": "cese_registrado,ginpix_confirmado",
				"ginpix_numero": "GX-2027-0099", "ginpix_confirmada_en": "2027-02-16", "observaciones_huella_sha256": huellaPrueba(""),
				"politica_ref": p.DefinicionRef, "politica_version": "1", "politica_huella_sha256": p.DefinicionHuellaSHA256,
				"ambito_idempotencia_hmac": prepC.AmbitoIdempotenciaHMAC, "huella_peticion_hmac": prepC.HuellaPeticionHMAC}}}
	orden.Autorizacion = exportacionPrueba(t, orden, expA, cierre.ActorRef, cierre.PerfilRef)
	reciboC, err := repo.ConfirmarOperacionSeguimiento(ctx, orden)
	if err != nil || reciboC.EstadoResultante != domain.EstadoCompletado || reciboC.CeseReciboRef != recibo.ReciboRef {
		t.Fatalf("confirmar cierre: %+v %v", reciboC, err)
	}
	estado, err := repo.ConsultarEstadoSeguimiento(ctx, org, expA)
	if err != nil || estado.Cese == nil || estado.Cierre == nil || estado.Cese.CausaClave != "fin_sustitucion" || estado.Cierre.GINPIXNumero != "GX-2027-0099" {
		t.Fatalf("estado: %+v %v", estado, err)
	}

	// ------------------------------------------------ modificación
	mod := ports.MaterialModificacionNombramiento{OrganizacionRef: org, ExpedienteRef: expB, ActorRef: "per_ct115_go", PerfilRef: "prf_ct115_go", VersionEsperada: 7,
		ClaveIdempotencia: "33333333-3333-4333-8333-333333333333", Datos: domain.DatosModificacionTrasNombramiento{MotivoClave: "cambio_jornada",
			Periodo: domain.PeriodoPrevisto{Inicio: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), Fin: time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)},
			Jornada: 5000, Coste: domain.Importe{Moneda: "EUR", Centimos: 1}, FuenteCoste: "fuente:coste:pendiente", FaseRetorno: domain.FaseFiscalizacion,
			Observaciones: "Reducción de jornada"}}
	sellosMod := sellosPrueba(t, ports.OperacionModificarTrasNombramiento, "mod-1")
	prepM, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionModificarTrasNombramiento, mod, sellosMod, refsPrueba("mod-1"))
	if err != nil || prepM.Confirmada {
		t.Fatalf("preparar modificación: %v", err)
	}
	mod.Datos.Coste, mod.Datos.FuenteCoste = domain.Importe{Moneda: "EUR", Centimos: 2000000}, "autoridad:ct:desarrollo:calculo-coste"
	instante = time.Now().UTC().Truncate(time.Microsecond)
	siguiente, err = prepM.Expediente.ModificarTrasNombramiento(7, mod.Datos, domain.DatosActuacion{AccionClave: domain.AccionModificarTrasNombramiento,
		ActorRef: mod.ActorRef, UnidadRef: prepM.Expediente.Asignacion.UnidadRef, ReciboRef: prepM.Referencias.ReciboRef, RealizadaEn: instante,
		FaseDestino: domain.FaseFiscalizacion, EstadoDestino: domain.EstadoEnCurso, Observaciones: mod.Datos.Observaciones})
	if err != nil {
		t.Fatal(err)
	}
	p = politica("vec.contratacion_temporal.reglas")
	orden = ports.OrdenConfirmarOperacionSeguimiento{Operacion: ports.OperacionModificarTrasNombramiento, Material: mod, Preparacion: prepM, Siguiente: siguiente,
		Politica: p, InstanteEfecto: instante, Accion: domain.AccionModificarTrasNombramiento, Finalidad: ports.FinalidadModificarTrasNombramiento,
		Audiencia: ports.AudienciaConsumoModificacionNombramientoV1, Contexto: ports.ContextoAutorizadoSeguimiento{Ambitos: ambitosPrueba(org, expB),
			Atributos: map[string]string{"version_expediente": "7", "motivo_clave": "cambio_jornada", "periodo_inicio": "2027-01-01", "periodo_fin": "2027-03-31",
				"porcentaje_jornada": "5000", "coste_centimos": "2000000", "fuente_coste_ref": "autoridad:ct:desarrollo:calculo-coste", "fase_retorno": "fiscalizacion",
				"estado_retorno": "en_curso", "observaciones_huella_sha256": huellaPrueba("Reducción de jornada"), "politica_ref": p.DefinicionRef,
				"politica_version": "1", "politica_huella_sha256": p.DefinicionHuellaSHA256, "ambito_idempotencia_hmac": prepM.AmbitoIdempotenciaHMAC,
				"huella_peticion_hmac": prepM.HuellaPeticionHMAC}}}
	orden.Autorizacion = exportacionPrueba(t, orden, expB, mod.ActorRef, mod.PerfilRef)
	reciboM, err := repo.ConfirmarOperacionSeguimiento(ctx, orden)
	if err != nil || reciboM.FaseResultante != domain.FaseFiscalizacion || reciboM.CosteCentimos != 2000000 {
		t.Fatalf("confirmar modificación: %+v %v", reciboM, err)
	}
	// La repetición con el coste provisional recupera el recibo original.
	mod.Datos.Coste, mod.Datos.FuenteCoste = domain.Importe{Moneda: "EUR", Centimos: 1}, "fuente:coste:pendiente"
	rec, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionModificarTrasNombramiento, mod, sellosMod, refsPrueba("mod-otra"))
	if err != nil || !rec.Confirmada || rec.Recibo == nil || rec.Recibo.ReciboRef != reciboM.ReciboRef || rec.FuenteCosteRef != "autoridad:ct:desarrollo:calculo-coste" {
		t.Fatalf("recuperación de la modificación: %+v %v", rec, err)
	}
}

func ambitosPrueba(org, exp string) map[string]string {
	return map[string]string{"organizacion_ref": org, "expediente_ref": exp, "fase_previa": "nombramiento", "estado_previo": "en_curso"}
}

func huellaPrueba(texto string) string {
	h := sha256.Sum256([]byte(texto))
	return hex.EncodeToString(h[:])
}

func refsPrueba(sufijo string) ports.ReferenciasEfectoSeguimiento {
	return ports.ReferenciasEfectoSeguimiento{ReservaRef: "reserva:go:" + sufijo, ReciboRef: "recibo:go:" + sufijo, EventoRef: "evento:go:" + sufijo}
}

func sellosPrueba(t *testing.T, operacion, semilla string) ports.SellosOperacionSeguimiento {
	t.Helper()
	ambito, huella, _ := ports.DominiosHMACOperacionSeguimiento(operacion)
	a, errA := ports.NuevaColeccionSellosHMAC("hmac-sha256:"+ambito+"/v1:"+huellaPrueba("a"+semilla), nil)
	h, errH := ports.NuevaColeccionSellosHMAC("hmac-sha256:"+huella+"/v1:"+huellaPrueba("h"+semilla), nil)
	if errA != nil || errH != nil {
		t.Fatal(errA, errH)
	}
	return ports.SellosOperacionSeguimiento{Ambitos: a, Huellas: h}
}

// exportacionPrueba compone un material estructuralmente válido cuya decisión
// liga el recurso exacto; la huella del contexto la calcula el dominio VEC,
// de modo que la prueba coteja también la canonización Go frente a CT115.
func exportacionPrueba(t *testing.T, o ports.OrdenConfirmarOperacionSeguimiento, exp, actor, perfil string) vp.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	recurso := vd.RecursoAutorizable{Referencia: exp, ModuloID: ports.ModuloContratacion, Tipo: tipoRecursoOperacionSeguimiento(o.Operacion),
		Ambitos: o.Contexto.Ambitos, Atributos: o.Contexto.Atributos}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decisionRef := "decision:go:" + strconv.FormatInt(time.Now().UnixNano(), 10)
	decision, _ := json.Marshal(map[string]string{"decision_ref": decisionRef, "accion": string(o.Accion), "finalidad": o.Finalidad,
		"modulo_id": ports.ModuloContratacion, "tipo_recurso": recurso.Tipo, "recurso_ref": exp, "principal_id": actor, "perfil_activo_ref": perfil,
		"contexto_recurso_huella_sha256": huella})
	hd := sha256.Sum256(decision)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionRef, hex.EncodeToString(hd[:]), strings.Repeat("e", 64), "contexto:go",
		strings.Repeat("f", 64), string(o.Accion), exp, huella, o.Audiencia, ahora, ahora.Add(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	publica, _, _ := ed25519.GenerateKey(rand.Reader)
	spki, _ := x509.MarshalPKIXPublicKey(publica)
	capacidad := []byte(strings.Repeat("c", 600))
	x, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(capacidad, resumen, decision, []byte("motivo"), []byte("contexto"), 1, 1,
		[]byte("payload"), []byte("sobre"), []byte("evidencia"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return x
}
