package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func recursoDenegacionDecisionCoberturaO404EPrueba() dominiovec.RecursoAutorizable {
	return dominiovec.RecursoAutorizable{
		Referencia: "recurso:expediente:1",
		ModuloID:   "contratacion-temporal",
		Tipo:       "expediente",
		Ambitos: map[string]string{
			"organizacion_ref": "diputacion-granada",
		},
		Atributos: map[string]string{
			"clasificacion": "interno",
		},
	}
}

type filaBytesDecisionCoberturaO404EPrueba struct {
	contenido []byte
	err       error
}

func (f filaBytesDecisionCoberturaO404EPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != 1 {
		return errors.New("número de columnas inesperado")
	}
	destino, ok := destinos[0].(*[]byte)
	if !ok {
		return errors.New("destino de recibo inesperado")
	}
	*destino = append([]byte(nil), f.contenido...)
	return nil
}

func TestEjecutorDecisionCoberturaO404ENoReintentaCommitAmbiguo(
	t *testing.T,
) {
	t.Parallel()
	errCommit := errors.New("respuesta COMMIT perdida")
	tx := &transaccionPreparacionPrueba{errConfirmar: errCommit}
	iniciador := &iniciadorPreparacionPrueba{
		transacciones: []pgx.Tx{tx, &transaccionPreparacionPrueba{}},
	}
	ejecutor, err :=
		nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(iniciador)
	if err != nil {
		t.Fatalf("crear ejecutor: %v", err)
	}
	invocaciones := 0
	err = ejecutor.EjecutarSesionTCB(
		context.Background(),
		func(puerto cobertura.SesionTCBOperacionDecisionCobertura) error {
			invocaciones++
			sesion, ok := puerto.(*sesionDecisionCoberturaO404E)
			if !ok {
				t.Fatal("tipo de sesión inesperado")
			}
			sesion.mu.Lock()
			sesion.estado = estadoSesionDecisionCoberturaConsumida
			sesion.confirmada = true
			sesion.mu.Unlock()
			return nil
		},
	)
	if !errors.Is(err, errCommit) {
		t.Fatalf("error = %v; esperado %v", err, errCommit)
	}
	if invocaciones != 1 || iniciador.inicios != 1 ||
		tx.confirmaciones != 1 || tx.reversiones != 1 {
		t.Fatalf(
			"ciclo inesperado: callback=%d begin=%d commit=%d rollback=%d",
			invocaciones,
			iniciador.inicios,
			tx.confirmaciones,
			tx.reversiones,
		)
	}
	if iniciador.opciones.IsoLevel != pgx.Serializable ||
		iniciador.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("opciones transaccionales incorrectas: %+v", iniciador.opciones)
	}
}

func TestEjecutorDecisionCoberturaO404ERollbackAnteErrorCallback(
	t *testing.T,
) {
	t.Parallel()
	errCallback := errors.New("núcleo rechaza recibo")
	tx := &transaccionPreparacionPrueba{}
	iniciador := &iniciadorPreparacionPrueba{tx: tx}
	ejecutor, err :=
		nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(iniciador)
	if err != nil {
		t.Fatalf("crear ejecutor: %v", err)
	}
	err = ejecutor.EjecutarSesionTCB(
		context.Background(),
		func(cobertura.SesionTCBOperacionDecisionCobertura) error {
			return errCallback
		},
	)
	if !errors.Is(err, errCallback) ||
		tx.confirmaciones != 0 || tx.reversiones != 1 {
		t.Fatalf(
			"resultado inesperado: err=%v commit=%d rollback=%d",
			err,
			tx.confirmaciones,
			tx.reversiones,
		)
	}
}

func TestEjecutorLecturaDecisionCoberturaO404EEsPrimarioSoloLectura(
	t *testing.T,
) {
	t.Parallel()
	tx := &transaccionPreparacionPrueba{}
	iniciador := &iniciadorPreparacionPrueba{tx: tx}
	ejecutor, err :=
		nuevoEjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL(
			iniciador,
		)
	if err != nil {
		t.Fatalf("crear ejecutor: %v", err)
	}
	err = ejecutor.EjecutarLecturaPrimariaTCB(
		context.Background(),
		func(puerto cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura) error {
			sesion, ok := puerto.(*sesionLecturaPrimariaDecisionCoberturaO404E)
			if !ok {
				t.Fatal("tipo de sesión inesperado")
			}
			sesion.mu.Lock()
			sesion.usada = true
			sesion.mu.Unlock()
			return nil
		},
	)
	if err != nil {
		t.Fatalf("ejecutar lectura: %v", err)
	}
	if iniciador.inicios != 1 || tx.confirmaciones != 1 ||
		iniciador.opciones.IsoLevel != pgx.Serializable ||
		iniciador.opciones.AccessMode != pgx.ReadOnly {
		t.Fatalf(
			"lectura no es SERIALIZABLE READ ONLY: opciones=%+v begin=%d commit=%d",
			iniciador.opciones,
			iniciador.inicios,
			tx.confirmaciones,
		)
	}
}

func TestSesionDecisionCoberturaO404EConfirmaUnaSolaVez(
	t *testing.T,
) {
	t.Parallel()
	reciboJSON := []byte(`{
		"esquema":"` + esquemaReciboDecisionCoberturaO404E + `",
		"recibo_ref":"rec","reserva_ref":"res","auditoria_ref":"aud",
		"correlacion_vec_ref":"cor","decision_vec_ref":"dec",
		"decision_vec_huella_sha256":"` + strings.Repeat("a", 64) + `",
		"codigo_probatorio_vec":"denegada_por_politica",
		"concedida_vec":false,"revision_cercado":2,
		"ambito_idempotencia_hmac":"hmac-sha256:vec.ct.ambito/v1:` +
		strings.Repeat("b", 64) + `",
		"huella_semantica_hmac":"hmac-sha256:vec.ct.semantica/v1:` +
		strings.Repeat("c", 64) + `",
		"confirmada_en":"2026-07-26T10:00:00Z",
		"aplicada":false,"denegada_vec":true,
		"decision_cobertura_ref":"","decision_cobertura_huella_sha256":"",
		"version_resultante":0,"evento_ref":"","actuacion_ref":""
	}`)
	tx := &transaccionPreparacionPrueba{
		fila: filaBytesDecisionCoberturaO404EPrueba{contenido: reciboJSON},
	}
	sesion := nuevaSesionDecisionCoberturaO404E(
		tx,
		context.Background(),
		nuevaGuardiaCicloDecisionCoberturaO404E(),
	)
	sesion.estado = estadoSesionDecisionCoberturaLista
	sesion.rama = cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada
	recurso := recursoDenegacionDecisionCoberturaO404EPrueba()
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("huella recurso: %v", err)
	}
	sesion.carga = cargaConfirmarDecisionCoberturaO404E{
		Esquema: esquemaCargaDecisionCoberturaO404E,
		Rama:    cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada,
		DecisionVEC: decisionVECDecisionCoberturaO404E{
			DecisionCanonica: []byte{1},
			MotivoCanonico:   []byte{2},
		},
		Denegacion: &denegacionDecisionCoberturaO404E{
			RecursoRef:          recurso.Referencia,
			RecursoModulo:       recurso.ModuloID,
			RecursoTipo:         recurso.Tipo,
			Ambitos:             recurso.Ambitos,
			Atributos:           recurso.Atributos,
			RecursoHuellaSHA256: huellaRecurso,
		},
		ConsumosC1: []consumoC1DecisionCoberturaO404E{},
	}
	recibo, err := sesion.Confirmar(context.Background())
	if err != nil {
		t.Fatalf("confirmar: %v", err)
	}
	if recibo.ReciboRef != "rec" || !recibo.DenegadaVEC ||
		!sesion.confirmada ||
		!strings.Contains(tx.consulta, funcionConfirmarDecisionCoberturaO404E) {
		t.Fatalf(
			"confirmación inesperada: recibo=%+v consulta=%s",
			recibo,
			tx.consulta,
		)
	}
	if _, err := sesion.Confirmar(context.Background()); err == nil {
		t.Fatal("la sesión permitió una segunda confirmación")
	}
}

func TestContratoJSONConsultaPrimariaDecisionCoberturaO404EEsEstable(
	t *testing.T,
) {
	t.Parallel()
	// Claves cerradas acordadas con SQL: esquema y las ocho coordenadas C,
	// más la huella privada que solo empuja la sesión nominal.
	carga := consultaPrimariaDecisionCoberturaO404E{
		Esquema:         esquemaConsultaPrimariaDecisionCoberturaO404E,
		OrganizacionRef: "org", ExpedienteRef: "exp",
		VersionExpediente: 7, ReservaRef: "res", ReciboRef: "rec",
		CorrelacionVECRef: "cor", DecisionVECRef: "dec",
		RevisionCercado: 11, HuellaOrdenSHA256: strings.Repeat("a", 64),
	}
	contenido, err := json.Marshal(carga)
	if err != nil {
		t.Fatalf("codificar: %v", err)
	}
	esperado := `{"esquema":"` + esquemaConsultaPrimariaDecisionCoberturaO404E +
		`","organizacion_ref":"org","expediente_ref":"exp",` +
		`"version_expediente":7,"reserva_ref":"res","recibo_ref":"rec",` +
		`"correlacion_vec_ref":"cor","decision_vec_ref":"dec",` +
		`"revision_cercado":11,"huella_orden_sha256":"` +
		strings.Repeat("a", 64) + `"}`
	if string(contenido) != esperado {
		t.Fatalf("contrato JSON cambió:\n%s\n!=\n%s", contenido, esperado)
	}
}

func TestPropuestaDecisionCoberturaO404ESiempreIncluyeVia(
	t *testing.T,
) {
	t.Parallel()
	proyeccion := nuevaPublicacionPropuestaDecisionCoberturaO404E(
		domain.PublicacionPropuestaDecisionCobertura{},
	)
	contenido, err := json.Marshal(proyeccion)
	if err != nil {
		t.Fatalf("codificar propuesta: %v", err)
	}
	if !strings.Contains(string(contenido), `"via_propuesta":""`) {
		t.Fatalf("via_propuesta ausente de la forma cerrada: %s", contenido)
	}
}

func TestReciboDecisionCoberturaO404ERechazaCamposDesconocidos(
	t *testing.T,
) {
	t.Parallel()
	contenido := []byte(`{
		"esquema":"` + esquemaReciboDecisionCoberturaO404E + `",
		"recibo_ref":"rec","reserva_ref":"res","auditoria_ref":"aud",
		"correlacion_vec_ref":"cor","decision_vec_ref":"dec",
		"decision_vec_huella_sha256":"` + strings.Repeat("a", 64) + `",
		"codigo_probatorio_vec":"denegada_por_politica",
		"concedida_vec":false,"revision_cercado":2,
		"ambito_idempotencia_hmac":"hmac-sha256:vec.ct.ambito/v1:` +
		strings.Repeat("b", 64) + `",
		"huella_semantica_hmac":"hmac-sha256:vec.ct.semantica/v1:` +
		strings.Repeat("c", 64) + `",
		"confirmada_en":"2026-07-26T10:00:00Z",
		"aplicada":false,"denegada_vec":true,
		"decision_cobertura_ref":"","decision_cobertura_huella_sha256":"",
		"version_resultante":0,"evento_ref":"","actuacion_ref":"",
		"campo_no_acordado":true
	}`)
	if _, err := decodificarReciboDecisionCoberturaO404E(contenido); err == nil {
		t.Fatal("un campo desconocido fue aceptado")
	}
}

func TestResultadoPrimarioAusenteSoloPublicaInstante(
	t *testing.T,
) {
	t.Parallel()
	instante := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	contenido := []byte(`{"esquema":"` +
		esquemaResultadoPrimarioDecisionCoberturaO404E +
		`","encontrado":false,"consulta":null,"recibo":null,` +
		`"observada_en_primario":"` + instante.Format(time.RFC3339) + `"}`)
	resultado, err :=
		decodificarResultadoPrimarioDecisionCoberturaO404E(contenido)
	if err != nil {
		t.Fatalf("decodificar ausencia: %v", err)
	}
	if resultado.Encontrado ||
		!resultado.ObservadaEnPrimario.Equal(instante) ||
		resultado.Coordenadas.OrganizacionRef != "" ||
		resultado.HuellaOrdenSHA256 != "" ||
		resultado.Recibo.ReciboRef != "" {
		t.Fatalf("la ausencia filtró datos: %+v", resultado)
	}
}
