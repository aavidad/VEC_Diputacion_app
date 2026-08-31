package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
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

func TestConfirmacionAltaReintenta40001EnTransaccionNueva(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	fallo := &transaccionAltaCandidataPrueba{fila: filaAltaCandidataPrueba{
		err: &pgconn.PgError{Code: "40001"},
	}}
	exito := &transaccionAltaCandidataPrueba{fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia)}
	iniciador := &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{fallo, exito}}
	transaccion := &TransaccionAltasPostgreSQLCandidata{pool: iniciador}
	if _, err := transaccion.confirmarConEntradas(
		context.Background(), evidencia, entradasConfirmarAlta{},
	); err != nil {
		t.Fatal(err)
	}
	if iniciador.inicios != 2 || exito.commits != 1 {
		t.Fatalf("reintento transaccional divergente: %d", iniciador.inicios)
	}
}

type errorEnvioPosiblePrueba struct{}

func (errorEnvioPosiblePrueba) Error() string { return "conexion perdida tras envio" }

type errorSeguroPrueba struct{}

func (errorSeguroPrueba) Error() string     { return "no enviado" }
func (errorSeguroPrueba) SafeToRetry() bool { return true }

func TestConfirmacionAltaReconciliaUnaVezConMismosArgumentos(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	primera := &transaccionAltaCandidataPrueba{
		fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia), errCommit: errorEnvioPosiblePrueba{},
	}
	segunda := &transaccionAltaCandidataPrueba{fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia)}
	iniciador := &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{primera, segunda}}
	transaccion := &TransaccionAltasPostgreSQLCandidata{pool: iniciador}
	entradas := entradasConfirmarAlta{alta: []byte("alta-exacta"), sellos: []byte("sellos-exactos")}
	if _, err := transaccion.confirmarConEntradas(context.Background(), evidencia, entradas); err != nil {
		t.Fatal(err)
	}
	if iniciador.inicios != 2 || len(primera.argumentos) != 12 || len(segunda.argumentos) != 12 ||
		string(primera.argumentos[10].([]byte)) != string(segunda.argumentos[10].([]byte)) ||
		string(primera.argumentos[11].([]byte)) != string(segunda.argumentos[11].([]byte)) {
		t.Fatal("la reconciliacion no reutilizo exactamente las doce entradas")
	}
}

func TestConfirmacionAltaSegundaAmbiguedadEsIndeterminada(t *testing.T) {
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	primera := &transaccionAltaCandidataPrueba{fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia), errCommit: errorEnvioPosiblePrueba{}}
	segunda := &transaccionAltaCandidataPrueba{fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia), errCommit: errorEnvioPosiblePrueba{}}
	transaccion := &TransaccionAltasPostgreSQLCandidata{pool: &iniciadorAltaCandidataPrueba{
		transacciones: []pgx.Tx{primera, segunda},
	}}
	_, err := transaccion.confirmarConEntradas(context.Background(), evidencia, entradasConfirmarAlta{})
	if !errors.Is(err, ports.ErrResultadoAltaIndeterminado) {
		t.Fatalf("segunda ambiguedad no normalizada: %v", err)
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
		{"envio seguro", &transaccionAltaCandidataPrueba{fila: filaAltaCandidataPrueba{err: errorSeguroPrueba{}}}},
		{"commit rollback", &transaccionAltaCandidataPrueba{fila: filaSQLConfirmacionPostgreSQLPrueba(evidencia), errCommit: pgx.ErrTxCommitRollback}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			iniciador := &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{caso.tx}}
			transaccion := &TransaccionAltasPostgreSQLCandidata{pool: iniciador}
			_, err := transaccion.confirmarConEntradas(context.Background(), evidencia, entradasConfirmarAlta{})
			if !errors.Is(err, ports.ErrPersistenciaNoDisponible) || iniciador.inicios != 1 {
				t.Fatalf("se reconcilio un resultado determinado: %v, %d", err, iniciador.inicios)
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
