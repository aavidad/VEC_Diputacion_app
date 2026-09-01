package postgres

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func evidenciaConfirmacionPostgreSQLPrueba(t *testing.T) (
	ports.EvidenciaOrdenConfirmarAltaCandidata,
	ports.DatosCandidaturaAlta,
) {
	t.Helper()
	instante := time.Date(2026, 8, 31, 10, 11, 12, 123456000, time.UTC)
	solicitud := domain.SolicitudCentro{
		CentroRef: "centro:seleccion", ContactoRef: "contacto:seleccion",
		CategoriaRef: "categoria:auxiliar", GrupoSubgrupo: "C2",
		MotivoClave: "acumulacion_tareas", Detalle: "Necesidad temporal R3B",
		Periodo: domain.PeriodoPrevisto{
			Inicio: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		},
		DocumentosAdjuntos: []string{}, Observaciones: "",
	}
	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia: "expediente:ct:r3b-postgre", OrganizacionRef: "organizacion:dipgra",
		NumeroVisible: "2026/R3B-PG",
		Flujo: domain.ReferenciaFlujo{DefinicionRef: "flujo:contratacion-temporal",
			Version: 1, HuellaSHA256: strings.Repeat("7", 64)},
		FaseInicial: "solicitud_registrada", Solicitud: solicitud,
		Actuacion: domain.DatosActuacion{
			AccionClave: "registrar_solicitud", ActorRef: "actor:rrhh:r3b",
			UnidadRef: "unidad:seleccion", ReciboRef: "recibo:alta:r3b-postgre",
			RealizadaEn: instante, FaseDestino: "solicitud_registrada",
			EstadoDestino: domain.EstadoEnCurso, DocumentosRef: []string{},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ambitos, err := ports.NuevaColeccionSellosHMAC(
		selloCandidaturaPrueba("vec.contratacion-temporal.ambito-idempotencia", 2, "a"),
		[]string{selloCandidaturaPrueba("vec.contratacion-temporal.ambito-idempotencia", 1, "b")},
	)
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := ports.NuevaColeccionSellosHMAC(
		selloCandidaturaPrueba("vec.contratacion-temporal.huella-peticion", 2, "c"),
		[]string{selloCandidaturaPrueba("vec.contratacion-temporal.huella-peticion", 1, "d")},
	)
	if err != nil {
		t.Fatal(err)
	}
	datosCandidatura := ports.DatosCandidaturaAlta{
		ReservaRef: "reserva:alta:postgre-r3b",
		Referencias: ports.ReferenciasAlta{ExpedienteRef: expediente.Referencia,
			NumeroVisible: expediente.NumeroVisible, ReciboRef: expediente.Actuaciones[0].ReciboRef},
		AmbitoIdempotenciaHMAC: ambitosActivo(t, ambitos),
		HuellaPeticionHMAC:     huellasActivo(t, huellas),
		OrganizacionRef:        expediente.OrganizacionRef, ActorRef: expediente.Actuaciones[0].ActorRef,
		PerfilRef: "perfil:rrhh:r3b", InstanteEfecto: instante,
	}
	candidatura, err := ports.NuevaCandidaturaAlta(datosCandidatura)
	if err != nil {
		t.Fatal(err)
	}
	return ports.EvidenciaOrdenConfirmarAltaCandidata{
		Expediente: expediente, AmbitosIdempotenciaHMAC: ambitos,
		HuellasPeticionHMAC: huellas, Candidatura: candidatura,
	}, datosCandidatura
}

func TestCanonConfirmacionAltaEsExactoYConMicrosegundos(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	alta, sellos, huella, err := canonConfirmacionAlta(evidencia)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(alta)
	defer borrarBytes(sellos)
	esperado := `{"esquema":"vec.contratacion-temporal.efecto-alta.v2","reserva_ref":"reserva:alta:postgre-r3b","expediente_ref":"expediente:ct:r3b-postgre","numero_visible":"2026/R3B-PG","recibo_ref":"recibo:alta:r3b-postgre","organizacion_ref":"organizacion:dipgra","actor_ref":"actor:rrhh:r3b","perfil_ref":"perfil:rrhh:r3b","version":1,"flujo":{"definicion_ref":"flujo:contratacion-temporal","version":1,"huella_sha256":"` + strings.Repeat("7", 64) + `"},"fase_actual":"solicitud_registrada","estado_actual":"en_curso","solicitud":{"centro_ref":"centro:seleccion","contacto_ref":"contacto:seleccion","categoria_ref":"categoria:auxiliar","grupo_subgrupo":"C2","motivo_clave":"acumulacion_tareas","detalle":"Necesidad temporal R3B","periodo":{"inicio":"2026-09-01","fin":"2026-09-30"},"rc":{"existe":false,"numero":"","fecha":"","importe":{"centimos":0,"moneda":"EUR"},"documento_ref":""},"documentos_adjuntos":[],"observaciones":""},"creado_en":"2026-08-31T10:11:12.123456Z","actualizado_en":"2026-08-31T10:11:12.123456Z","actuacion":{"secuencia":1,"version_expediente":1,"accion_clave":"registrar_solicitud","actor_ref":"actor:rrhh:r3b","unidad_ref":"unidad:seleccion","recibo_ref":"recibo:alta:r3b-postgre","realizada_en":"2026-08-31T10:11:12.123456Z","fase_origen":"","fase_destino":"solicitud_registrada","estado_origen":"pendiente","estado_destino":"en_curso","observaciones":"","documentos_ref":[]}}`
	if string(alta) != esperado || len(huella) != 64 {
		t.Fatalf("canon de alta divergente\nobtenido=%s\nesperado=%s\nhuella=%s", alta, esperado, huella)
	}
	esperadoSellos := `{"esquema":"vec.contratacion-temporal.sellos-hmac.v1","activo":{"generacion":2,"ambito_hmac":"` + selloCandidaturaPrueba("vec.contratacion-temporal.ambito-idempotencia", 2, "a") + `","huella_hmac":"` + selloCandidaturaPrueba("vec.contratacion-temporal.huella-peticion", 2, "c") + `"},"retenidos":[{"generacion":1,"ambito_hmac":"` + selloCandidaturaPrueba("vec.contratacion-temporal.ambito-idempotencia", 1, "b") + `","huella_hmac":"` + selloCandidaturaPrueba("vec.contratacion-temporal.huella-peticion", 1, "d") + `"}]}`
	if string(sellos) != esperadoSellos {
		t.Fatalf("canon de sellos divergente: %s", sellos)
	}
}

func TestConfirmacionAltaExigeVentanaBreveContenidaEnConcesion(t *testing.T) {
	emitidaDecision := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	registradaEn := emitidaDecision.Add(10 * time.Second)
	validaHasta := emitidaDecision.Add(90 * time.Second)
	confirmacion := puertosvec.DatosConfirmacionRegistroConcesionAutorizacionLigadaV3{
		EmitidaEn: emitidaDecision, RegistradaEn: registradaEn, ValidaHasta: validaHasta,
	}
	pruebas := []struct {
		nombre    string
		emitidaEn time.Time
		expiraEn  time.Time
		aceptada  bool
	}{
		{"concesion de 90s contiene capacidad de 5s", validaHasta.Add(-5 * time.Second), validaHasta, true},
		{"emision anterior a decision", emitidaDecision.Add(-time.Second), emitidaDecision.Add(4 * time.Second), false},
		{"emision anterior a registro", registradaEn.Add(-time.Second), registradaEn.Add(4 * time.Second), false},
		{"emision exactamente en limite final", validaHasta, validaHasta.Add(5 * time.Second), false},
		{"expiracion posterior a limite final", validaHasta.Add(-4 * time.Second), validaHasta.Add(time.Second), false},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			resumen := resumenCapacidadConfirmacionPrueba(t, prueba.emitidaEn, prueba.expiraEn)
			if obtenida := capacidadBreveContenidaEnConcesion(resumen, confirmacion); obtenida != prueba.aceptada {
				t.Fatalf("contencion temporal divergente: obtenida=%t esperada=%t", obtenida, prueba.aceptada)
			}
		})
	}
}

func resumenCapacidadConfirmacionPrueba(
	t *testing.T,
	emitidaEn time.Time,
	expiraEn time.Time,
) puertosvec.ResumenCapacidadAtestacionAutorizacionV3 {
	t.Helper()
	huella := strings.Repeat("a", 64)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:ct:r9", huella, huella, "contexto:ct:r9", huella,
		ports.AccionCrearSolicitud, "efecto:ct:r9", huella,
		audienciaConfirmarAltaV1, emitidaEn, expiraEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	return resumen
}

func filaConfirmacionPostgreSQLPrueba(
	evidencia ports.EvidenciaOrdenConfirmarAltaCandidata,
) filaConfirmacionAlta {
	instante := evidencia.Expediente.ActualizadoEn.Add(time.Second)
	return filaConfirmacionAlta{
		expedienteRef: evidencia.Expediente.Referencia,
		numeroVisible: evidencia.Expediente.NumeroVisible,
		version:       1, reciboRef: evidencia.Expediente.Actuaciones[0].ReciboRef,
		auditoriaRef: "auditoria:alta:r3b", eventoRef: "evento:alta:r3b",
		confirmadaEn: instante,
	}
}

func filaSQLConfirmacionPostgreSQLPrueba(
	evidencia ports.EvidenciaOrdenConfirmarAltaCandidata,
) pgx.Row {
	fila := filaConfirmacionPostgreSQLPrueba(evidencia)
	recibo := ports.ReciboAlta{ExpedienteRef: fila.expedienteRef,
		NumeroVisible: fila.numeroVisible, Version: uint64(fila.version),
		ReciboRef: fila.reciboRef, AuditoriaRef: fila.auditoriaRef,
		EventoRef: fila.eventoRef, ConfirmadaEn: fila.confirmadaEn}
	return filaAltaCandidataPrueba{valores: []any{
		fila.expedienteRef, fila.numeroVisible, fila.version, fila.reciboRef,
		fila.auditoriaRef, fila.eventoRef, fila.confirmadaEn, huellaReciboAlta(recibo),
	}}
}

func TestConfirmacionAltaReintenta40001Y40P01EnTransaccionNueva(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	for _, codigo := range []string{"40001", "40P01"} {
		t.Run(codigo, func(t *testing.T) {
			fallo := &transaccionAltaCandidataPrueba{
				fila:      filaSQLConfirmacionPostgreSQLPrueba(evidencia),
				errCommit: &pgconn.PgError{Code: codigo},
			}
			exito := &transaccionAltaCandidataPrueba{
				fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia),
			}
			base := &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{fallo, exito}}
			iniciador := &iniciadorConfirmacionPrueba{base: base}
			transaccion := &TransaccionAltasPostgreSQLCandidata{pool: iniciador}
			recibo, err := transaccion.confirmarConEntradas(
				context.Background(), evidencia, entradasConfirmarAlta{},
			)
			if err != nil || recibo.ExpedienteRef == "" || base.inicios != 2 ||
				iniciador.reconciliaciones != 0 || fallo.commits != 1 || exito.commits != 1 {
				t.Fatalf("reintento transaccional divergente: recibo=%+v err=%v inicios=%d reconciliaciones=%d commits=%d/%d",
					recibo, err, base.inicios, iniciador.reconciliaciones, fallo.commits, exito.commits)
			}
		})
	}
}

func TestConfirmacionAltaLimitaReintentosTransitoriosATres(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	transacciones := make([]pgx.Tx, maximoIntentosConfirmarAlta+1)
	pruebas := make([]*transaccionAltaCandidataPrueba, len(transacciones))
	for indice := range transacciones {
		pruebas[indice] = &transaccionAltaCandidataPrueba{
			fila:      filaSQLConfirmacionPostgreSQLPrueba(evidencia),
			errCommit: &pgconn.PgError{Code: "40001"},
		}
		transacciones[indice] = pruebas[indice]
	}
	base := &iniciadorAltaCandidataPrueba{transacciones: transacciones}
	iniciador := &iniciadorConfirmacionPrueba{base: base}
	recibo, err := (&TransaccionAltasPostgreSQLCandidata{pool: iniciador}).confirmarConEntradas(
		context.Background(), evidencia, entradasConfirmarAlta{},
	)
	if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
		recibo != (ports.ReciboAlta{}) || base.inicios != maximoIntentosConfirmarAlta ||
		iniciador.reconciliaciones != 0 ||
		pruebas[0].commits != 1 || pruebas[1].commits != 1 || pruebas[2].commits != 1 ||
		pruebas[3].commits != 0 {
		t.Fatalf("limite de reintentos divergente: recibo=%+v err=%v inicios=%d reconciliaciones=%d commits=%d/%d/%d/%d",
			recibo, err, base.inicios, iniciador.reconciliaciones, pruebas[0].commits, pruebas[1].commits,
			pruebas[2].commits, pruebas[3].commits)
	}
}

type errorEnvioPosiblePrueba struct{}

func (errorEnvioPosiblePrueba) Error() string { return "conexion perdida tras envio" }

type errorSeguroPrueba struct{}

func (errorSeguroPrueba) Error() string     { return "no enviado" }
func (errorSeguroPrueba) SafeToRetry() bool { return true }

type iniciadorConfirmacionPrueba struct {
	base             *iniciadorAltaCandidataPrueba
	reconciliaciones int
}

func (i *iniciadorConfirmacionPrueba) BeginTx(
	ctx context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	if _, existe := ctx.Deadline(); existe {
		i.reconciliaciones++
	}
	return i.base.BeginTx(ctx, opciones)
}

func TestConfirmacionAltaReconciliaUnaVezConMismosArgumentos(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	for _, caso := range []struct {
		nombre string
		causa  error
	}{
		{"08007", &pgconn.PgError{Code: "08007"}},
		{"transporte posiblemente enviado", errorEnvioPosiblePrueba{}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			primera := &transaccionAltaCandidataPrueba{
				fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia), errCommit: caso.causa,
			}
			segunda := &transaccionAltaCandidataPrueba{fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia)}
			base := &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{primera, segunda}}
			iniciador := &iniciadorConfirmacionPrueba{base: base}
			transaccion := &TransaccionAltasPostgreSQLCandidata{pool: iniciador}
			entradas := entradasConfirmarAlta{alta: []byte("alta-exacta"), sellos: []byte("sellos-exactos")}
			if _, err := transaccion.confirmarConEntradas(context.Background(), evidencia, entradas); err != nil {
				t.Fatal(err)
			}
			if base.inicios != 2 || iniciador.reconciliaciones != 1 ||
				primera.commits != 1 || segunda.commits != 1 ||
				len(primera.argumentos) != 12 || len(segunda.argumentos) != 12 ||
				string(primera.argumentos[10].([]byte)) != string(segunda.argumentos[10].([]byte)) ||
				string(primera.argumentos[11].([]byte)) != string(segunda.argumentos[11].([]byte)) {
				t.Fatal("la reconciliacion no reutilizo una vez las doce entradas")
			}
		})
	}
}

func TestConfirmacionAltaSegundaAmbiguedadEsIndeterminada(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	primera := &transaccionAltaCandidataPrueba{fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia), errCommit: errorEnvioPosiblePrueba{}}
	segunda := &transaccionAltaCandidataPrueba{fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia), errCommit: errorEnvioPosiblePrueba{}}
	base := &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{primera, segunda}}
	iniciador := &iniciadorConfirmacionPrueba{base: base}
	transaccion := &TransaccionAltasPostgreSQLCandidata{pool: iniciador}
	recibo, err := transaccion.confirmarConEntradas(context.Background(), evidencia, entradasConfirmarAlta{})
	if !errors.Is(err, ports.ErrResultadoAltaIndeterminado) ||
		recibo != (ports.ReciboAlta{}) || base.inicios != 2 ||
		iniciador.reconciliaciones != 1 || primera.commits != 1 || segunda.commits != 1 {
		t.Fatalf("segunda ambiguedad no normalizada: recibo=%+v err=%v inicios=%d reconciliaciones=%d commits=%d/%d",
			recibo, err, base.inicios, iniciador.reconciliaciones, primera.commits, segunda.commits)
	}
}

func TestConfirmacionAlta08007EsAmbigua(t *testing.T) {
	causa := &pgconn.PgError{Code: "08007"}
	if errorConfirmacionAmbiguo(causa) ||
		!errorConfirmacionAmbiguo(marcarEnvioPosibleConfirmacion(causa)) {
		t.Fatal("08007 solo debe abrir reconciliacion tras invocar Commit")
	}
}

func TestConfirmacionAltaSafeToRetryYCommitRollbackNoReconcilian(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	casos := []struct {
		nombre string
		tx     *transaccionAltaCandidataPrueba
	}{
		{"envio seguro", &transaccionAltaCandidataPrueba{fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia), errCommit: errorSeguroPrueba{}}},
		{"commit rollback", &transaccionAltaCandidataPrueba{fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia), errCommit: pgx.ErrTxCommitRollback}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			base := &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{caso.tx}}
			iniciador := &iniciadorConfirmacionPrueba{base: base}
			transaccion := &TransaccionAltasPostgreSQLCandidata{pool: iniciador}
			recibo, err := transaccion.confirmarConEntradas(context.Background(), evidencia, entradasConfirmarAlta{})
			if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
				recibo != (ports.ReciboAlta{}) || base.inicios != 1 ||
				iniciador.reconciliaciones != 0 || caso.tx.commits != 1 {
				t.Fatalf("se reconcilio un resultado determinado: recibo=%+v err=%v inicios=%d reconciliaciones=%d commits=%d",
					recibo, err, base.inicios, iniciador.reconciliaciones, caso.tx.commits)
			}
		})
	}
}

func TestConfirmacionAltaErrorDeConsultaNoReconcilia(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	tx := &transaccionAltaCandidataPrueba{fila: filaAltaCandidataPrueba{
		err: errorEnvioPosiblePrueba{},
	}}
	iniciador := &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{tx}}
	transaccion := &TransaccionAltasPostgreSQLCandidata{pool: iniciador}

	_, err := transaccion.confirmarConEntradas(
		context.Background(), evidencia, entradasConfirmarAlta{},
	)
	if !errors.Is(err, ports.ErrPersistenciaNoDisponible) {
		t.Fatalf("error previo a commit no normalizado: %v", err)
	}
	if iniciador.inicios != 1 || tx.commits != 0 {
		t.Fatalf("error previo a commit reconciliado: inicios=%d commits=%d", iniciador.inicios, tx.commits)
	}
}

func TestConfirmacionAltaReciboOHashAdulteradoNoHaceCommit(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	casos := []struct {
		nombre string
		mutar  func([]any)
	}{
		{"fila", func(valores []any) { valores[0] = "expediente:ct:r3b:otro" }},
		{"recibo", func(valores []any) { valores[3] = "recibo:alta:r3b:otro" }},
		{"huella", func(valores []any) { valores[7] = strings.Repeat("e", 64) }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			original := filaSQLConfirmacionPostgreSQLPrueba(evidencia).(filaAltaCandidataPrueba).valores
			valores := append([]any(nil), original...)
			caso.mutar(valores)
			tx := &transaccionAltaCandidataPrueba{fila: filaAltaCandidataPrueba{valores: valores}}
			transaccion := &TransaccionAltasPostgreSQLCandidata{pool: &iniciadorAltaCandidataPrueba{
				transacciones: []pgx.Tx{tx},
			}}
			_, err := transaccion.confirmarConEntradas(context.Background(), evidencia, entradasConfirmarAlta{})
			if !errors.Is(err, ports.ErrResultadoAltaNoConfiable) || tx.commits != 0 {
				t.Fatalf("resultado adulterado aceptado: %v, commits=%d", err, tx.commits)
			}
		})
	}
}

func TestConfirmacionAltaCommitConfirmadoVenceCancelacion(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	tx := &transaccionAltaCandidataPrueba{
		fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia), alCommit: cancelar,
	}
	transaccion := &TransaccionAltasPostgreSQLCandidata{pool: &iniciadorAltaCandidataPrueba{
		transacciones: []pgx.Tx{tx},
	}}
	if _, err := transaccion.confirmarConEntradas(ctx, evidencia, entradasConfirmarAlta{}); err != nil {
		t.Fatalf("commit confirmado convertido en cancelacion: %v", err)
	}
}

func leerJSONPublicoR3B(t *testing.T, ruta string, destino any) {
	t.Helper()
	contenido, err := os.ReadFile(ruta)
	if ruta == "" || err != nil || json.Unmarshal(contenido, destino) != nil {
		t.Fatalf("vector publico R3B invalido: %v", err)
	}
}

func decodificarPublicoR3B(t *testing.T, valor string) []byte {
	t.Helper()
	contenido, err := base64.StdEncoding.DecodeString(valor)
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

func instantePublicoR3B(t *testing.T, valor string) time.Time {
	t.Helper()
	instante, err := time.Parse(time.RFC3339Nano, valor)
	if err != nil {
		t.Fatal(err)
	}
	return instante.UTC()
}

func expedientePublicoR3B(t *testing.T, efecto efectoAltaCanonico) domain.Expediente {
	t.Helper()
	fecha := func(valor string) time.Time {
		instante, err := time.Parse("2006-01-02", valor)
		if err != nil {
			t.Fatal(err)
		}
		return instante
	}
	if efecto.Solicitud.RC.Existe {
		t.Fatal("el vector neutral no debe declarar RC")
	}
	instante := instantePublicoR3B(t, efecto.CreadoEn)
	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia: efecto.ExpedienteRef, OrganizacionRef: efecto.OrganizacionRef,
		NumeroVisible: efecto.NumeroVisible,
		Flujo: domain.ReferenciaFlujo{DefinicionRef: efecto.Flujo.DefinicionRef,
			Version: efecto.Flujo.Version, HuellaSHA256: efecto.Flujo.HuellaSHA256},
		FaseInicial: domain.ClaveFase(efecto.FaseActual),
		Solicitud: domain.SolicitudCentro{
			CentroRef: efecto.Solicitud.CentroRef, ContactoRef: efecto.Solicitud.ContactoRef,
			CategoriaRef: efecto.Solicitud.CategoriaRef, GrupoSubgrupo: efecto.Solicitud.GrupoSubgrupo,
			MotivoClave: domain.ClaveCatalogo(efecto.Solicitud.MotivoClave), Detalle: efecto.Solicitud.Detalle,
			Periodo:            domain.PeriodoPrevisto{Inicio: fecha(efecto.Solicitud.Periodo.Inicio), Fin: fecha(efecto.Solicitud.Periodo.Fin)},
			DocumentosAdjuntos: efecto.Solicitud.DocumentosAdjuntos, Observaciones: efecto.Solicitud.Observaciones,
		},
		Actuacion: domain.DatosActuacion{
			AccionClave: domain.ClaveCatalogo(efecto.Actuacion.AccionClave), ActorRef: efecto.ActorRef,
			UnidadRef: efecto.Actuacion.UnidadRef, ReciboRef: efecto.ReciboRef, RealizadaEn: instante,
			FaseDestino: domain.ClaveFase(efecto.FaseActual), EstadoDestino: domain.EstadoOperativo(efecto.EstadoActual),
			Observaciones: efecto.Actuacion.Observaciones, DocumentosRef: efecto.Actuacion.DocumentosRef,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return expediente
}

func coleccionesPublicasR3B(t *testing.T, sellos sellosAltaCanonicos) (ports.ColeccionSellosHMAC, ports.ColeccionSellosHMAC) {
	t.Helper()
	ambitosRetenidos := make([]string, len(sellos.Retenidos))
	huellasRetenidas := make([]string, len(sellos.Retenidos))
	for indice, sello := range sellos.Retenidos {
		ambitosRetenidos[indice], huellasRetenidas[indice] = sello.AmbitoHMAC, sello.HuellaHMAC
	}
	ambitos, errAmbitos := ports.NuevaColeccionSellosHMAC(sellos.Activo.AmbitoHMAC, ambitosRetenidos)
	huellas, errHuellas := ports.NuevaColeccionSellosHMAC(sellos.Activo.HuellaHMAC, huellasRetenidas)
	if errAmbitos != nil || errHuellas != nil {
		t.Fatalf("sellos publicos invalidos: %v/%v", errAmbitos, errHuellas)
	}
	return ambitos, huellas
}

func materialPublicoR3B(t *testing.T, entrada entradaPublicaR3B, bundle bundlePublicoR3B, efecto efectoAltaCanonico, sellos sellosAltaCanonicos) (dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
	t.Helper()
	var plantilla decisionPlantillaPublicaR3B
	var motivo motivoPublicoR3B
	if json.Unmarshal(decodificarPublicoR3B(t, entrada.DecisionPlantillaB64), &plantilla) != nil ||
		json.Unmarshal(decodificarPublicoR3B(t, entrada.MotivoB64), &motivo) != nil ||
		!bytes.Equal(decodificarPublicoR3B(t, entrada.AltaB64), decodificarPublicoR3B(t, bundle.AltaB64)) ||
		!bytes.Equal(decodificarPublicoR3B(t, entrada.SellosB64), decodificarPublicoR3B(t, bundle.SellosB64)) {
		t.Fatal("entrada y bundle publicos divergentes")
	}
	actor, err := dominiovec.RehidratarContextoActorVinculadoV2(decodificarPublicoR3B(t, entrada.ContextoB64))
	if err != nil {
		t.Fatal(err)
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: plantilla.VinculoAutenticacionActor.RegistroContextoRef,
		Contexto:            actor, RepresentacionCanonica: decodificarPublicoR3B(t, entrada.ContextoB64),
		HuellaSHA256:                      plantilla.VinculoAutenticacionActor.ContextoActorHuellaSHA256,
		ManifiestoProcedenciaCanonico:     decodificarPublicoR3B(t, entrada.ManifiestoB64),
		ManifiestoProcedenciaHuellaSHA256: entrada.ManifiestoHuellaSHA256,
		AutoridadEfectiva:                 dominiovec.AutoridadProcedenciaContextoActorV1(entrada.AutoridadEfectiva),
		ResueltoEnAutoritativo:            instantePublicoR3B(t, entrada.ResueltoEn),
	}
	ahora := instantePublicoR3B(t, entrada.Ahora)
	autoridad := autoridadPublicaR3B{autenticacion: plantilla.VinculoAutenticacionActor.Autenticacion(), contexto: resultado, ahora: ahora, correlacion: plantilla.CorrelacionRef}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(context.Background(), autoridad,
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: plantilla.VinculoAutenticacionActor.AutenticacionRef, SesionRef: plantilla.VinculoAutenticacionActor.SesionRef},
		autoridad, dominiovec.SolicitudContextoActor{Cuenta: dominiovec.CuentaAutenticadaContextoActor{
			CuentaRef: actor.Instantanea.CuentaRef, Metodo: actor.Principal.AuthMethod, Garantia: actor.Principal.AuthAssurance}, PerfilActivoRef: actor.PerfilActivoRef}, autoridad)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), autoridad)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo.Referencia,
		Accion: plantilla.Accion, Finalidad: plantilla.Finalidad, Correlacion: correlacion,
		Recurso: dominiovec.RecursoAutorizable{Referencia: plantilla.RecursoRef, ModuloID: plantilla.ModuloID,
			Tipo: plantilla.TipoRecurso, Ambitos: map[string]string{"organizacion_ref": efecto.OrganizacionRef, "centro_ref": efecto.Solicitud.CentroRef, "categoria_ref": efecto.Solicitud.CategoriaRef},
			Atributos: map[string]string{ports.AtributoHuellaEfectoAltaSHA256: entrada.EfectoHuellaSHA256, "flujo_ref": efecto.Flujo.DefinicionRef, "flujo_version": "1", "flujo_huella_sha256": efecto.Flujo.HuellaSHA256, ports.AtributoHuellaPeticionHMACActiva: sellos.Activo.HuellaHMAC}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var rol dominiovec.VersionRol
	var control dominiovec.ControlVigenciaVersionRol
	var asignacion dominiovec.AsignacionPerfil
	if json.Unmarshal(bundle.VersionRolDocumento, &rol) != nil || json.Unmarshal(bundle.ControlRolDocumento, &control) != nil || json.Unmarshal(bundle.AsignacionDocumento, &asignacion) != nil {
		t.Fatal("instantanea publica invalida")
	}
	evidencia, err := dominiovec.NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, dominiovec.InstantaneaAutorizacion{AsignacionPerfil: asignacion, VersionRol: rol, ControlVigenciaVersionRol: control, Politicas: entrada.Politicas, RevisionCatalogoPoliticas: entrada.RevisionCatalogo, CatalogoPoliticasHuellaSHA256: entrada.HuellaCatalogoSHA256}, plantilla.DecisionRef, ahora, ahora.Add(90*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	decision, err := dominiovec.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	decisionCanonica, errCanon := dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil || errCanon != nil || !bytes.Equal(decisionCanonica, decodificarPublicoR3B(t, bundle.DecisionB64)) {
		t.Fatalf("decision publica divergente: %v/%v", err, errCanon)
	}
	ordenRegistro, err := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, motivo.Referencia, resultado)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := puertosvec.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Background(), autoridad, ordenRegistro)
	if err != nil {
		t.Fatal(err)
	}
	capacidadBytes := decodificarPublicoR3B(t, bundle.CapacidadB64)
	var capacidad capacidadPublicaR3B
	if json.Unmarshal(capacidadBytes, &capacidad) != nil {
		t.Fatal("capacidad publica invalida")
	}
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(capacidad.DecisionRef, capacidad.HuellaDecisionSHA256, capacidad.HuellaMotivoSHA256, capacidad.ContextoRef, capacidad.HuellaContextoSHA256, capacidad.Operacion, capacidad.EfectoRef, capacidad.HuellaEfectoSHA256, capacidad.AudienciaConsumo, instantePublicoR3B(t, capacidad.EmitidaEn), instantePublicoR3B(t, capacidad.ExpiraEn))
	if err != nil {
		t.Fatal(err)
	}
	material, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(capacidadBytes, resumen, decodificarPublicoR3B(t, bundle.DecisionB64), decodificarPublicoR3B(t, bundle.MotivoB64), decodificarPublicoR3B(t, bundle.ContextoB64), bundle.PersonaVersion, bundle.PerfilVersion, decodificarPublicoR3B(t, bundle.PayloadB64), decodificarPublicoR3B(t, bundle.COSEB64), decodificarPublicoR3B(t, bundle.EvidenciaB64), decodificarPublicoR3B(t, bundle.SPKIB64))
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, decision, confirmacion, material
}

func estadoConfirmacionPublicaR3B(t *testing.T, ctx context.Context, admin interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, expedienteRef string) string {
	t.Helper()
	var estado string
	err := admin.QueryRow(ctx, `SELECT concat_ws('|',
		(SELECT instante_efecto::text FROM vec_contratacion_temporal.candidatura_alta_tecnica WHERE expediente_ref=$1),
		(SELECT md5(to_jsonb(c)::text) FROM vec_contratacion_temporal.confirmacion_agregado_alta c WHERE expediente_ref=$1),
		(SELECT count(*) FROM vec_contratacion_temporal.expediente_alta WHERE expediente_ref=$1),
		(SELECT count(*) FROM vec_contratacion_temporal.auditoria_alta WHERE expediente_ref=$1),
		(SELECT count(*) FROM vec_contratacion_temporal.outbox_alta WHERE expediente_ref=$1),
		(SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 c
		  JOIN vec_contratacion_temporal.confirmacion_agregado_alta a USING(decision_ref)
		 WHERE a.expediente_ref=$1))`, expedienteRef).Scan(&estado)
	if err != nil {
		t.Fatal(err)
	}
	return estado
}
