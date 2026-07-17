package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

var instanteLlamamientoPostgreSQLPrueba = time.Date(2026, 7, 17, 8, 0, 0, 123_456_000, time.UTC)

type relojLlamamientoPostgreSQLPrueba struct{ instante time.Time }

func (r relojLlamamientoPostgreSQLPrueba) Ahora() time.Time { return r.instante }

type iniciadorLlamamientoPostgreSQLPrueba struct {
	tx       pgx.Tx
	opciones pgx.TxOptions
	inicios  int
}

func (i *iniciadorLlamamientoPostgreSQLPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	i.opciones = opciones
	i.inicios++
	return i.tx, nil
}

type transaccionLlamamientoPostgreSQLPrueba struct {
	pgx.Tx
	fila           pgx.Row
	consulta       string
	confirmaciones int
	reversiones    int
}

type filaLlamamientoPostgreSQLPrueba struct{ valores []any }

func (f filaLlamamientoPostgreSQLPrueba) Scan(destinos ...any) error {
	if len(destinos) != len(f.valores) {
		return errors.New("cantidad de columnas inesperada")
	}
	for indice, valor := range f.valores {
		switch destino := destinos[indice].(type) {
		case *string:
			texto, valido := valor.(string)
			if !valido {
				return errors.New("columna de texto invalida")
			}
			*destino = texto
		case *[]byte:
			contenido, valido := valor.([]byte)
			if !valido {
				return errors.New("columna binaria invalida")
			}
			*destino = append([]byte(nil), contenido...)
		case *pgtype.Timestamptz:
			instante, valido := valor.(time.Time)
			if !valido {
				return errors.New("columna temporal invalida")
			}
			*destino = pgtype.Timestamptz{Time: instante, Valid: true}
		default:
			return errors.New("destino de columna no soportado")
		}
	}
	return nil
}

func (t *transaccionLlamamientoPostgreSQLPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (t *transaccionLlamamientoPostgreSQLPrueba) QueryRow(_ context.Context, consulta string, _ ...any) pgx.Row {
	t.consulta = consulta
	return t.fila
}

func (t *transaccionLlamamientoPostgreSQLPrueba) Commit(context.Context) error {
	t.confirmaciones++
	return nil
}

func (t *transaccionLlamamientoPostgreSQLPrueba) Rollback(context.Context) error {
	t.reversiones++
	return nil
}

func TestTransaccionLlamamientoPostgreSQLValidaComandoYPermaneceCerrada(t *testing.T) {
	propuesta, evidencia, comando := propuestaYEvidenciaLlamamientoPostgreSQLPrueba(t)
	recibo := reciboLlamamientoPostgreSQLPrueba(t, propuesta, evidencia)
	tx := &transaccionLlamamientoPostgreSQLPrueba{fila: filaReciboLlamamientoPostgreSQLPrueba(recibo)}
	iniciador := &iniciadorLlamamientoPostgreSQLPrueba{tx: tx}
	repositorio, err := nuevaTransaccionPropuestasLlamamientoPostgreSQL(
		iniciador, relojLlamamientoPostgreSQLPrueba{instanteLlamamientoPostgreSQLPrueba.Add(time.Second)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err = repositorio.GuardarPropuestaLlamamiento(context.Background(), comando); !errors.Is(err, puertosbolsa.ErrPersistenciaPropuestaNoDisponible) {
		t.Fatalf("el contrato antiguo no quedo cerrado: %v", err)
	}
	if iniciador.inicios != 0 || tx.confirmaciones != 0 || tx.consulta != "" {
		t.Fatalf("el cierre alcanzo PostgreSQL: inicios=%d commits=%d consulta=%q", iniciador.inicios, tx.confirmaciones, tx.consulta)
	}
}

func TestReciboLlamamientoPostgreSQLRespuestaManipuladaNoValida(t *testing.T) {
	propuesta, evidencia, _ := propuestaYEvidenciaLlamamientoPostgreSQLPrueba(t)
	datos, err := evidencia.Datos()
	if err != nil {
		t.Fatal(err)
	}
	documento, err := json.Marshal(propuesta)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		mutar  func(*reciboLlamamientoPostgreSQLV1)
	}{
		{"decision", func(r *reciboLlamamientoPostgreSQLV1) { r.DecisionRef = "decision:ajena" }},
		{"documento", func(r *reciboLlamamientoPostgreSQLV1) { r.PropuestaCanonica[0] ^= 1 }},
		{"huella auditoria", func(r *reciboLlamamientoPostgreSQLV1) { r.HuellaAuditoriaSHA256 = strings.Repeat("0", 64) }},
		{"consumo", func(r *reciboLlamamientoPostgreSQLV1) { r.ConsumoCanonico = append(r.ConsumoCanonico, '\n') }},
		{"evento", func(r *reciboLlamamientoPostgreSQLV1) { r.EventoRef = "evento:ajeno" }},
		{"instante", func(r *reciboLlamamientoPostgreSQLV1) {
			r.ConfirmadaEn.Time = instanteLlamamientoPostgreSQLPrueba.Add(5 * time.Minute)
		}},
		{"clave duplicada consumo", func(r *reciboLlamamientoPostgreSQLV1) {
			r.ConsumoCanonico = anteponerClaveDuplicadaLlamamientoPrueba(r.ConsumoCanonico)
			r.HuellaConsumoSHA256 = huellaBytesPostgreSQLLlamamiento(r.ConsumoCanonico)
		}},
		{"clave duplicada auditoria", func(r *reciboLlamamientoPostgreSQLV1) {
			r.RegistroAuditoria = anteponerClaveDuplicadaLlamamientoPrueba(r.RegistroAuditoria)
			r.HuellaAuditoriaSHA256 = huellaBytesPostgreSQLLlamamiento(r.RegistroAuditoria)
		}},
		{"clave duplicada outbox", func(r *reciboLlamamientoPostgreSQLV1) {
			r.EventoCanonico = anteponerClaveDuplicadaLlamamientoPrueba(r.EventoCanonico)
			r.HuellaEventoSHA256 = huellaBytesPostgreSQLLlamamiento(r.EventoCanonico)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			recibo := reciboLlamamientoPostgreSQLPrueba(t, propuesta, evidencia)
			caso.mutar(&recibo)
			err := recibo.validar(propuesta, datos, documento)
			if !errors.Is(err, puertosbolsa.ErrPersistenciaPropuestaNoDisponible) {
				t.Fatalf("respuesta manipulada aceptada: %v", err)
			}
		})
	}
}

func TestJSONLlamamientoNoAmbiguoRechazaDuplicadoAnidado(t *testing.T) {
	contenido := []byte(`{"objeto":{"clave":1,"clave":2}}`)
	if err := validarJSONLlamamientoNoAmbiguo(contenido); err == nil {
		t.Fatal("se acepto una clave duplicada dentro de un objeto anidado")
	}
}

func anteponerClaveDuplicadaLlamamientoPrueba(contenido []byte) []byte {
	duplicado := make([]byte, 0, len(contenido)+24)
	duplicado = append(duplicado, []byte(`{"esquema":"invalido",`)...)
	return append(duplicado, contenido[1:]...)
}

func TestTransaccionLlamamientoPostgreSQLEvidenciaYCierreFallanSeguro(t *testing.T) {
	propuesta, _, comando := propuestaYEvidenciaLlamamientoPostgreSQLPrueba(t)
	instantanea, _, _, err := comando.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := puertosbolsa.NuevoComandoGuardarPropuestaLlamamiento(
		instantanea, propuesta, puertosvec.EvidenciaUsoDecisionAutorizacion{},
	); !errors.Is(err, puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("evidencia cero admitida por el comando: %v", err)
	}
	tx := &transaccionLlamamientoPostgreSQLPrueba{fila: filaReciboLlamamientoPostgreSQLPrueba(reciboLlamamientoPostgreSQLV1{})}
	repositorio, _ := nuevaTransaccionPropuestasLlamamientoPostgreSQL(
		&iniciadorLlamamientoPostgreSQLPrueba{tx: tx},
		relojLlamamientoPostgreSQLPrueba{instanteLlamamientoPostgreSQLPrueba.Add(time.Second)},
	)
	if err := repositorio.GuardarPropuestaLlamamiento(context.Background(), puertosbolsa.ComandoGuardarPropuestaLlamamiento{}); !errors.Is(err, puertosbolsa.ErrComandoGuardarPropuestaLlamamientoInvalido) || tx.confirmaciones != 0 {
		t.Fatalf("comando cero admitido: %v", err)
	}
	if _, err := nuevaTransaccionPropuestasLlamamientoPostgreSQL(
		nil, relojLlamamientoPostgreSQLPrueba{instanteLlamamientoPostgreSQLPrueba},
	); !errors.Is(err, puertosbolsa.ErrPersistenciaPropuestaNoDisponible) {
		t.Fatalf("pool nulo admitido: %v", err)
	}
	if err := errorPostgreSQLLlamamiento(context.Background(), &pgconn.PgError{Code: "PBL03"}); !errors.Is(err, puertosbolsa.ErrDecisionAutorizacionLlamamientoUsada) ||
		!errors.Is(err, puertosvec.ErrDecisionAutorizacionConsumida) {
		t.Fatalf("replay no preserva contrato: %v", err)
	}
	if err := errorPostgreSQLLlamamiento(context.Background(), &pgconn.PgError{
		Code: "23505", ConstraintName: "propuesta_necesidad_unica",
	}); !errors.Is(err, puertosbolsa.ErrNecesidadLlamamientoYaPropuesta) {
		t.Fatalf("carrera de necesidad no preserva contrato: %v", err)
	}
}

func filaReciboLlamamientoPostgreSQLPrueba(r reciboLlamamientoPostgreSQLV1) pgx.Row {
	return filaLlamamientoPostgreSQLPrueba{valores: []any{
		r.Resultado, r.PropuestaRef, r.HuellaPropuestaSHA256,
		r.PropuestaCanonica, r.HuellaDocumentoSHA256,
		r.DecisionRef, r.HuellaDecisionSHA256,
		r.AtestacionRef, r.AtestacionCanonica, r.HuellaAtestacionSHA256,
		r.ConsumoRef, r.ConsumoCanonico, r.HuellaConsumoSHA256,
		r.AuditoriaRef, r.RegistroAuditoria, r.HuellaAuditoriaSHA256,
		r.EventoRef, r.EventoCanonico, r.HuellaEventoSHA256, r.ConfirmadaEn.Time,
	}}
}

func reciboLlamamientoPostgreSQLPrueba(
	t *testing.T,
	propuesta dominiobolsa.PropuestaLlamamiento,
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacion,
) reciboLlamamientoPostgreSQLV1 {
	t.Helper()
	datos, err := evidencia.Datos()
	if err != nil {
		t.Fatal(err)
	}
	documento, _ := json.Marshal(propuesta)
	confirmadaEn := instanteLlamamientoPostgreSQLPrueba.Add(time.Second)
	formato := confirmadaEn.Format(formatoInstanteMicrosegundo)
	atestacion := []byte(`{"esquema":"atestacion-prueba"}`)
	recibo := reciboLlamamientoPostgreSQLV1{
		Resultado: "confirmada", PropuestaRef: propuesta.PropuestaRef,
		HuellaPropuestaSHA256: propuesta.HuellaContenidoSHA256, PropuestaCanonica: documento,
		HuellaDocumentoSHA256: huellaBytesPostgreSQLLlamamiento(documento),
		DecisionRef:           datos.Decision.DecisionRef, HuellaDecisionSHA256: datos.HuellaDecisionSHA256,
		AtestacionRef: "atestacion:llamamiento:prueba", AtestacionCanonica: atestacion,
		HuellaAtestacionSHA256: huellaBytesPostgreSQLLlamamiento(atestacion),
		ConsumoRef:             "consumo:llamamiento:prueba", AuditoriaRef: "auditoria:llamamiento:prueba",
		EventoRef: "evento:llamamiento:prueba", ConfirmadaEn: pgtypeTimestamptzPrueba(confirmadaEn),
	}
	consumo := consumoLlamamientoPostgreSQLV1{
		Esquema: "vec.bolsa.llamamiento.consumo.v1", ConsumoRef: recibo.ConsumoRef,
		DecisionRef: recibo.DecisionRef, PrincipalRef: datos.Decision.PrincipalID,
		PropuestaRef: propuesta.PropuestaRef, NecesidadRef: propuesta.NecesidadRef,
		VersionNecesidad: propuesta.VersionNecesidad, HuellaNecesidadSHA256: propuesta.HuellaNecesidadSHA256,
		HuellaPropuestaSHA256: propuesta.HuellaContenidoSHA256, HuellaDocumentoSHA256: recibo.HuellaDocumentoSHA256,
		AtestacionRef: recibo.AtestacionRef, HuellaAtestacionSHA256: recibo.HuellaAtestacionSHA256,
		ConsumidaEn: formato,
	}
	recibo.ConsumoCanonico, _ = json.Marshal(consumo)
	recibo.HuellaConsumoSHA256 = huellaBytesPostgreSQLLlamamiento(recibo.ConsumoCanonico)
	auditoria := auditoriaLlamamientoPostgreSQLV1{
		Esquema: "vec.bolsa.llamamiento.auditoria.v1", AuditoriaRef: recibo.AuditoriaRef, Secuencia: 1,
		HuellaAnteriorSHA256: strings.Repeat("0", 64), ConsumoRef: recibo.ConsumoRef,
		DecisionRef: recibo.DecisionRef, PropuestaRef: recibo.PropuestaRef,
		HuellaPropuestaSHA256: recibo.HuellaPropuestaSHA256, HuellaConsumoSHA256: recibo.HuellaConsumoSHA256,
		RegistradaEn: formato,
	}
	recibo.RegistroAuditoria, _ = json.Marshal(auditoria)
	recibo.HuellaAuditoriaSHA256 = huellaBytesPostgreSQLLlamamiento(recibo.RegistroAuditoria)
	evento := eventoLlamamientoPostgreSQLV1{
		Esquema: "vec.bolsa.llamamiento.outbox.v1", EventoRef: recibo.EventoRef,
		Tipo: "bolsa.llamamiento.propuesta_confirmada.v1", AgregadoRef: recibo.PropuestaRef,
		HuellaPropuestaSHA256: recibo.HuellaPropuestaSHA256, AuditoriaRef: recibo.AuditoriaRef,
		HuellaAuditoriaSHA256: recibo.HuellaAuditoriaSHA256, EmitidoEn: formato,
	}
	recibo.EventoCanonico, _ = json.Marshal(evento)
	recibo.HuellaEventoSHA256 = huellaBytesPostgreSQLLlamamiento(recibo.EventoCanonico)
	return recibo
}

func pgtypeTimestamptzPrueba(instante time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: instante, Valid: true}
}

func propuestaYEvidenciaLlamamientoPostgreSQLPrueba(
	t *testing.T,
) (
	dominiobolsa.PropuestaLlamamiento,
	puertosvec.EvidenciaUsoDecisionAutorizacion,
	puertosbolsa.ComandoGuardarPropuestaLlamamiento,
) {
	t.Helper()
	instante := instanteLlamamientoPostgreSQLPrueba
	bolsa, err := dominiobolsa.NuevaBolsaConstituida(dominiobolsa.AltaBolsaConstituida{
		BolsaRef: "bolsa:postgresql:1", Version: 1, ProcesoRef: "proceso:postgresql:1",
		CategoriaRef: "categoria:postgresql:1", ListadoDefinitivoRef: "listado:postgresql:1",
		VersionListado: 1, HuellaListadoSHA256: huellaLlamamientoPostgreSQLPrueba('a'),
		ResolucionConstitucionRef: "resolucion:postgresql:1",
		HuellaResolucionSHA256:    huellaLlamamientoPostgreSQLPrueba('b'),
		ConstituidaEn:             instante.Add(-48 * time.Hour), VigenteDesde: instante.Add(-24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	huellaBolsa, _ := bolsa.HuellaCanonicaSHA256()
	necesidad, err := dominiobolsa.NuevaNecesidadCobertura(dominiobolsa.AltaNecesidadCobertura{
		NecesidadRef: "necesidad:postgresql:1", Version: 1, BolsaRef: bolsa.BolsaRef,
		VersionBolsa: bolsa.Version, HuellaBolsaSHA256: huellaBolsa, CategoriaRef: bolsa.CategoriaRef,
		PuestoRef: "puesto:postgresql:1", UnidadRef: "unidad:postgresql:1",
		TipoCoberturaRef: "tipo:postgresql:1", NumeroPuestos: 1,
		InicioPrevisto: instante.Add(time.Hour), FinPrevisto: instante.Add(30 * 24 * time.Hour),
		CreadaEn: instante.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	politica, err := dominiobolsa.NuevaReferenciaPoliticaLlamamiento(dominiobolsa.ReferenciaPoliticaLlamamiento{
		PoliticaRef: "politica:postgresql:1", Clave: "politica_postgresql_gobernada", Version: 1,
		HuellaSHA256: huellaLlamamientoPostgreSQLPrueba('c'), PublicadaEn: instante.Add(-48 * time.Hour),
		VigenteDesde: instante.Add(-24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	participacion, err := dominiobolsa.NuevaParticipacionBolsa(dominiobolsa.AltaParticipacionBolsa{
		ParticipacionRef: "participacion:postgresql:1", BolsaRef: bolsa.BolsaRef,
		SujetoRef: "sujeto:postgresql:1", Version: 1, AltaEn: instante.Add(-12 * time.Hour),
		Situaciones: []dominiobolsa.SituacionParticipacionBolsa{{
			Secuencia: 1, EstadoClave: "estado_postgresql_gobernado", EstadoVersion: 1,
			HuellaEstadoSHA256: huellaLlamamientoPostgreSQLPrueba('d'), CausaClave: "causa_postgresql_gobernada",
			CausaVersion: 1, HuellaCausaSHA256: huellaLlamamientoPostgreSQLPrueba('e'),
			DecisionRef: "decision:situacion:postgresql:1", HuellaDecisionSHA256: huellaLlamamientoPostgreSQLPrueba('f'),
			Desde: instante.Add(-12 * time.Hour),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	instantanea, err := dominiobolsa.NuevaInstantaneaOrdenBolsa(dominiobolsa.AltaInstantaneaOrdenBolsa{
		InstantaneaRef: "instantanea:postgresql:1", Version: 1, Bolsa: bolsa,
		ReferidaEn: instante, GeneradaEn: instante,
		Entradas: []dominiobolsa.EntradaOrdenBolsa{{Orden: 1, Participacion: participacion}},
	})
	if err != nil {
		t.Fatal(err)
	}
	situacion, _ := participacion.SituacionVigenteEn(instante)
	huellaNecesidad, _ := necesidad.HuellaCanonicaSHA256()
	evaluacion := dominiobolsa.EvaluacionParticipacionLlamamiento{
		ParticipacionRef: participacion.ParticipacionRef, SujetoRef: participacion.SujetoRef, Orden: 1,
		SituacionSecuencia: situacion.Secuencia, EstadoClave: situacion.EstadoClave,
		EstadoVersion: situacion.EstadoVersion, HuellaEstadoSHA256: situacion.HuellaEstadoSHA256,
		NecesidadRef: necesidad.NecesidadRef, VersionNecesidad: necesidad.Version,
		HuellaNecesidadSHA256: huellaNecesidad, InstantaneaRef: instantanea.InstantaneaRef,
		VersionInstantanea: instantanea.Version, HuellaInstantaneaSHA256: instantanea.HuellaContenidoSHA256,
		PoliticaRef: politica.PoliticaRef, VersionPolitica: politica.Version,
		HuellaPoliticaSHA256: politica.HuellaSHA256, Resultado: dominiobolsa.ResultadoElegible,
		Motivos: []dominiobolsa.MotivoEvaluacionLlamamiento{{
			Clave: "resultado_postgresql", ReglaRef: "regla:postgresql:1", VersionRegla: 1,
			HuellaReglaSHA256: huellaLlamamientoPostgreSQLPrueba('1'),
		}},
		EntradaEvaluacionRef:   "recibo:entrada:postgresql:1",
		HuellaEntradaSHA256:    huellaLlamamientoPostgreSQLPrueba('2'),
		ResultadoEvaluacionRef: "recibo:resultado:postgresql:1",
		HuellaResultadoSHA256:  huellaLlamamientoPostgreSQLPrueba('3'), EvaluadaEn: instante,
	}
	propuesta, err := dominiobolsa.ProponerPrimerLlamamiento(dominiobolsa.OrdenProponerPrimerLlamamiento{
		PropuestaRef: "propuesta:postgresql:1", Bolsa: bolsa, Necesidad: necesidad,
		Instantanea: instantanea, Politica: politica,
		Evaluaciones: []dominiobolsa.EvaluacionParticipacionLlamamiento{evaluacion}, GeneradaEn: instante,
	})
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := pruebasvec.NuevoVinculoGenerico(instante)
	if err != nil {
		t.Fatal(err)
	}
	datosVinculo, _ := vinculo.Datos()
	recurso := dominiovec.RecursoAutorizable{
		Referencia: propuesta.NecesidadRef, ModuloID: puertosbolsa.ModuloLlamamientos,
		Tipo:    puertosbolsa.TipoRecursoNecesidad,
		Ambitos: map[string]string{"categoria_ref": bolsa.CategoriaRef, "unidad_ref": necesidad.UnidadRef},
	}
	huellaRecurso, _ := recurso.HuellaContextoAutorizacionSHA256()
	politicaSeguridad := "politica:seguridad:postgresql:1"
	huellas := map[string]string{politicaSeguridad: huellaLlamamientoPostgreSQLPrueba('4')}
	huellaCatalogo, _ := dominiovec.HuellaEvidenciasCatalogoPoliticasAutorizacion([]string{politicaSeguridad}, huellas)
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "decision:postgresql:1", Concedida: true, Codigo: "concedida",
		PrincipalID: datosVinculo.PrincipalID, PerfilActivoRef: datosVinculo.PerfilActivoRef,
		Accion: puertosbolsa.AccionProponerLlamamiento, RecursoRef: propuesta.NecesidadRef,
		ModuloID: puertosbolsa.ModuloLlamamientos, TipoRecurso: puertosbolsa.TipoRecursoNecesidad,
		ContextoRecursoHuellaSHA256: huellaRecurso, Finalidad: puertosbolsa.FinalidadProponerLlamamiento,
		CorrelacionRef: "correlacion:postgresql:1", VinculoAutenticacionActor: vinculo,
		AsignacionRef: "asignacion:postgresql:1", AsignacionHuellaSHA256: huellaLlamamientoPostgreSQLPrueba('5'),
		VersionRolRef: "rol:postgresql:1", VersionRolHuellaSHA256: huellaLlamamientoPostgreSQLPrueba('6'),
		ControlVigenciaVersionRolRef: "rol:postgresql:1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: huellaLlamamientoPostgreSQLPrueba('7'),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasRefs: []string{politicaSeguridad}, PoliticasEvaluadasHuellasSHA256: huellas,
		PoliticasRefs: []string{politicaSeguridad}, PoliticasHuellasSHA256: huellas,
		GarantiaMinima: dominiovec.AuthAssuranceHigh, EmitidaEn: instante.Add(-time.Second),
		ValidaHasta: instante.Add(2 * time.Minute),
	}
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacion(decision, instante)
	if err != nil {
		t.Fatal(err)
	}
	comando, err := puertosbolsa.NuevoComandoGuardarPropuestaLlamamiento(instantanea, propuesta, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	return propuesta, evidencia, comando
}

func huellaLlamamientoPostgreSQLPrueba(caracter byte) string {
	return strings.Repeat(string(caracter), 64)
}
