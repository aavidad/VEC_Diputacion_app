package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	transaccionbolsa "vec-diputacion-granada/internal/modules/bolsa/internal/transaccion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

var instantePostgreSQLPrueba = time.Date(2026, time.July, 15, 12, 0, 0, 123_456_000, time.UTC)

type iniciadorPostgreSQLBaremacionPrueba struct{ tx pgx.Tx }

func (i iniciadorPostgreSQLBaremacionPrueba) BeginTx(
	context.Context, pgx.TxOptions,
) (pgx.Tx, error) {
	return i.tx, nil
}

type transaccionPostgreSQLBaremacionPrueba struct {
	pgx.Tx
	fila           pgx.Row
	confirmaciones int
	reversiones    int
}

func (t *transaccionPostgreSQLBaremacionPrueba) Exec(
	context.Context, string, ...any,
) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (t *transaccionPostgreSQLBaremacionPrueba) QueryRow(
	context.Context, string, ...any,
) pgx.Row {
	return t.fila
}

func (t *transaccionPostgreSQLBaremacionPrueba) Commit(context.Context) error {
	t.confirmaciones++
	return nil
}

func (t *transaccionPostgreSQLBaremacionPrueba) Rollback(context.Context) error {
	t.reversiones++
	return nil
}

type filaPostgreSQLBaremacionPrueba struct{ valores []any }

func (f filaPostgreSQLBaremacionPrueba) Scan(destinos ...any) error {
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
			if valor == nil {
				*destino = nil
				continue
			}
			bytes, valido := valor.([]byte)
			if !valido {
				return errors.New("columna binaria invalida")
			}
			*destino = append([]byte(nil), bytes...)
		case *pgtype.Timestamptz:
			if valor == nil {
				*destino = pgtype.Timestamptz{}
				continue
			}
			instante, valido := valor.(time.Time)
			if !valido {
				return errors.New("columna temporal invalida")
			}
			*destino = pgtype.Timestamptz{Time: instante, Valid: true}
		default:
			return errors.New("destino de Scan no soportado")
		}
	}
	return nil
}

type relojPostgreSQLBaremacionPrueba struct{ instante time.Time }

func (r relojPostgreSQLBaremacionPrueba) Ahora() time.Time { return r.instante }

// Solo existe dentro de esta prueba. El constructor productivo exige que el
// ensamblado sustituya este doble por un verificador criptografico auditado.
type verificadorPostgreSQLBaremacionPrueba struct{}

func (verificadorPostgreSQLBaremacionPrueba) VerificarSelloBaremacion(
	ctx context.Context, solicitud puertosbolsa.SolicitudVerificarSelloBaremacion,
) error {
	if ctx == nil || solicitud.Validar() != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	return ctx.Err()
}

func TestSalidasPostgreSQLNoConfiablesReviertenLosSeisMetodos(t *testing.T) {
	t.Parallel()
	token, err := transaccionbolsa.GenerarTokenReserva()
	if err != nil {
		t.Fatal(err)
	}
	baremacion := baremacionPostgreSQLPrueba(t)
	reserva := puertosbolsa.SolicitudReservarCambioBaremacion{
		Contexto: contextoPostgreSQLBaremacionPrueba(
			t, puertosbolsa.AccionReservarAltaBaremacion, baremacion.ID,
		),
		Clase:               puertosbolsa.ClaseCambioAltaBaremacion,
		ClaveIdempotencia:   "alta-baremacion-postgresql-prueba",
		BaremacionMeritoRef: baremacion.ID,
		HuellaSolicitudHMAC: hmacPostgreSQLBaremacionPrueba("1"),
		SolicitadaEn:        instantePostgreSQLPrueba.Add(-time.Minute),
		ExpiraEn:            instantePostgreSQLPrueba.Add(5 * time.Minute),
	}
	confirmacion := puertosbolsa.SolicitudConfirmarCambioBaremacion{
		Contexto: contextoPostgreSQLBaremacionPrueba(
			t, puertosbolsa.AccionConfirmarAltaBaremacion, baremacion.ID,
		),
		Token: token, Clase: puertosbolsa.ClaseCambioAltaBaremacion,
		HuellaSolicitudHMAC: hmacPostgreSQLBaremacionPrueba("2"),
		Agregado:            baremacion,
		Trazabilidad: puertosbolsa.TrazabilidadCambioBaremacion{
			MotivoClave: "alta_autobaremacion",
			Motivo:      "Alta de la autobaremacion calculada oficialmente.",
		},
		ConfirmadaEn: instantePostgreSQLPrueba.Add(-30 * time.Second),
	}
	abandono := puertosbolsa.SolicitudAbandonarReservaBaremacion{
		Contexto: contextoPostgreSQLBaremacionPrueba(
			t, puertosbolsa.AccionAbandonarAltaBaremacion, baremacion.ID,
		),
		Token: token, Clase: puertosbolsa.ClaseCambioAltaBaremacion,
		BaremacionMeritoRef: baremacion.ID,
	}
	vigente := puertosbolsa.SolicitudObtenerBaremacionVigente{
		Contexto: contextoPostgreSQLBaremacionPrueba(
			t, puertosbolsa.AccionConsultarBaremacionVigente, baremacion.ID,
		),
		BaremacionMeritoRef: baremacion.ID,
	}
	version := puertosbolsa.SolicitudObtenerVersionBaremacion{
		Contexto: contextoPostgreSQLBaremacionPrueba(
			t, puertosbolsa.AccionConsultarVersionBaremacion, baremacion.ID,
		),
		BaremacionMeritoRef: baremacion.ID, Numero: 1,
	}
	evidencia := puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion{
		Contexto: contextoPostgreSQLBaremacionPrueba(
			t, puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion,
			"auditoria:postgresql:prueba",
		),
		BaremacionMeritoRef: baremacion.ID, NumeroVersion: 1,
		AuditoriaRef:    "auditoria:postgresql:prueba",
		EventoOutboxRef: "evento:postgresql:prueba",
	}

	casos := []struct {
		nombre   string
		valores  []any
		ejecutar func(*RepositorioBaremaciones) error
	}{
		{
			"reservar", []any{
				"reservada", "reserva-forjada", reserva.ExpiraEn, "", "",
				nil, nil, "", "", "", "",
			},
			func(r *RepositorioBaremaciones) error {
				_, err := r.ReservarCambio(context.Background(), reserva)
				return err
			},
		},
		{
			"confirmar", []any{
				"confirmada", "1", strings.Repeat("a", 64), []byte(`{}`),
				nil, "auditoria", strings.Repeat("b", 64),
				"evento", strings.Repeat("c", 64),
			},
			func(r *RepositorioBaremaciones) error {
				_, err := r.ConfirmarCambio(context.Background(), confirmacion)
				return err
			},
		},
		{
			"abandonar", []any{"estado_no_gobernado"},
			func(r *RepositorioBaremaciones) error {
				return r.AbandonarReserva(context.Background(), abandono)
			},
		},
		{
			"obtener vigente", []any{
				"obtenida", "1", strings.Repeat("a", 64), []byte(`{}`), nil,
				"auditoria:postgresql:prueba",
			},
			func(r *RepositorioBaremaciones) error {
				_, err := r.ObtenerVersionVigente(context.Background(), vigente)
				return err
			},
		},
		{
			"obtener version", []any{
				"obtenida", "1", strings.Repeat("a", 64), []byte(`{}`), nil,
				"auditoria:postgresql:prueba",
			},
			func(r *RepositorioBaremaciones) error {
				_, err := r.ObtenerVersion(context.Background(), version)
				return err
			},
		},
		{
			"obtener evidencia", []any{
				"obtenida", "1", strings.Repeat("a", 64), []byte(`{}`), nil,
				[]byte(`{}`), []byte(`{}`),
			},
			func(r *RepositorioBaremaciones) error {
				_, err := r.ObtenerEvidenciaTransaccion(context.Background(), evidencia)
				return err
			},
		},
	}

	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			tx := &transaccionPostgreSQLBaremacionPrueba{
				fila: filaPostgreSQLBaremacionPrueba{valores: caso.valores},
			}
			repositorio, err := nuevoRepositorioBaremaciones(
				iniciadorPostgreSQLBaremacionPrueba{tx: tx},
				relojPostgreSQLBaremacionPrueba{instante: instantePostgreSQLPrueba},
				verificadorPostgreSQLBaremacionPrueba{},
			)
			if err != nil {
				t.Fatal(err)
			}
			err = caso.ejecutar(repositorio)
			if !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) {
				t.Fatalf("error cerrado esperado, recibido: %v", err)
			}
			if tx.confirmaciones != 0 || tx.reversiones == 0 {
				t.Fatalf(
					"salida no confiable confirmada: commits=%d rollbacks=%d",
					tx.confirmaciones, tx.reversiones,
				)
			}
		})
	}
}

func contextoPostgreSQLBaremacionPrueba(
	t *testing.T,
	accion puertosbolsa.AccionOperacionBaremacion,
	recursoRef string,
) puertosbolsa.ContextoOperacionBaremacion {
	t.Helper()
	campos, existe := puertosbolsa.CamposRequeridosOperacionBaremacion(accion)
	if !existe {
		t.Fatalf("accion sin campos: %s", accion)
	}
	clase, existe := puertosbolsa.ClaseRecursoRequeridaOperacionBaremacion(accion)
	if !existe {
		t.Fatalf("accion sin recurso: %s", accion)
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: recursoRef, ModuloID: "bolsa", Tipo: string(clase),
		Ambitos: map[string]string{"sujeto_ref": "sujeto:postgresql:prueba"},
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	actor, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		instantePostgreSQLPrueba,
		"per_postgresql_baremacion_prueba_001",
		"prf_postgresql_baremacion_prueba_001",
		dominiovec.AuthMethodCertificate,
		dominiovec.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	datosVinculo, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "decision:" + strings.ReplaceAll(string(accion), ".", "-"),
		Concedida:   true, Codigo: "concedida",
		PrincipalID: actor.Principal.ID, PerfilActivoRef: actor.PerfilActivoRef,
		Accion: string(accion), RecursoRef: recursoRef,
		ModuloID: "bolsa", TipoRecurso: string(clase),
		ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad:                   "gestion_bolsa", CorrelacionRef: "correlacion:postgresql:prueba",
		VinculoAutenticacionActor:             vinculo,
		AsignacionRef:                         "asignacion:postgresql:prueba:v1",
		AsignacionHuellaSHA256:                strings.Repeat("1", 64),
		VersionRolRef:                         "rol:postgresql:prueba:v1",
		VersionRolHuellaSHA256:                strings.Repeat("2", 64),
		ControlVigenciaVersionRolRef:          "rol:postgresql:prueba:v1",
		ControlVigenciaVersionRolRevision:     1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("3", 64),
		RevisionCatalogoPoliticas:             1,
		CatalogoPoliticasHuellaSHA256:         huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256:       map[string]string{},
		GarantiaMinima:                        dominiovec.AuthAssuranceHigh,
		CamposPermitidos:                      campos,
		EmitidaEn:                             instantePostgreSQLPrueba.Add(-time.Minute),
		ValidaHasta:                           instantePostgreSQLPrueba.Add(4 * time.Minute),
	}
	contexto, err := puertosbolsa.NuevaAutorizacionOperacionBaremacion(
		decision,
		puertosbolsa.VinculoAutenticacionBaremacion{
			SujetoRef:                 "sujeto:postgresql:prueba",
			Metodo:                    datosVinculo.MetodoObservado,
			Garantia:                  datosVinculo.GarantiaObservada,
			AutenticacionRef:          datosVinculo.AutenticacionRef,
			SesionRef:                 datosVinculo.SesionRef,
			SesionEmitidaEn:           datosVinculo.SesionEmitidaEn,
			SesionValidaHasta:         datosVinculo.SesionValidaHasta,
			VinculoAutenticacionActor: vinculo,
		},
		instantePostgreSQLPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	return contexto
}

func baremacionPostgreSQLPrueba(t *testing.T) dominiobolsa.BaremacionMerito {
	t.Helper()
	criterio := dominiobolsa.ReferenciaCriterio{
		ProcesoRef: "proceso:postgresql:prueba", Clave: "experiencia.publica",
		Version: 1, HuellaSHA256: strings.Repeat("a", 64),
		PuntosMaximos: 10 * dominiobolsa.UnidadesPorPunto,
		ReglaCalculo: dominiobolsa.ReferenciaReglaCalculo{
			Clave: "experiencia_publica_dias", Version: 1,
			HuellaSHA256: strings.Repeat("b", 64),
		},
	}
	evidencias := []dominiobolsa.EvidenciaMerito{{
		Referencia: dominiobolsa.ReferenciaEvidencia{
			DocumentoRef: "documento:postgresql:prueba", VersionDocumento: 1,
			RepresentacionRef: "representacion:postgresql:prueba",
			HuellaSHA256:      strings.Repeat("c", 64),
		},
	}}
	calculo := dominiobolsa.CalculoOficialBaremacion{
		CalculoRef: "calculo:postgresql:prueba",
		ProcesoRef: criterio.ProcesoRef, SolicitudRef: "solicitud:postgresql:prueba",
		SujetoRef:           "sujeto:postgresql:prueba",
		BaremacionMeritoRef: "baremacion:postgresql:prueba",
		Criterio:            criterio, Regla: criterio.ReglaCalculo, Evidencias: evidencias,
		EntradaRef:            "entrada:postgresql:prueba",
		HuellaEntradaSHA256:   strings.Repeat("d", 64),
		PuntosCalculados:      4 * dominiobolsa.UnidadesPorPunto,
		DesgloseRef:           "desglose:postgresql:prueba",
		HuellaDesgloseSHA256:  strings.Repeat("e", 64),
		ResultadoRef:          "resultado:postgresql:prueba",
		HuellaResultadoSHA256: strings.Repeat("f", 64),
		MotorCalculoRef:       "motor:postgresql:prueba", VersionMotorCalculo: "v1",
		EvidenciaEjecucionRef: "ejecucion:postgresql:prueba",
		HuellaEjecucionSHA256: strings.Repeat("0", 64),
		CalculadoEn:           instantePostgreSQLPrueba.Add(-20 * time.Minute),
	}
	baremacion, err := dominiobolsa.NuevaBaremacionMerito(
		dominiobolsa.AltaMeritoBaremable{
			ID: "baremacion:postgresql:prueba", ProcesoRef: criterio.ProcesoRef,
			SolicitudRef: "solicitud:postgresql:prueba",
			SujetoRef:    "sujeto:postgresql:prueba", Criterio: criterio,
			EvidenciasIniciales: evidencias,
			PuntosDeclarados:    5 * dominiobolsa.UnidadesPorPunto,
			CalculoOficial:      calculo,
			CreadaEn:            instantePostgreSQLPrueba.Add(-10 * time.Minute),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return baremacion
}

func hmacPostgreSQLBaremacionPrueba(caracter string) string {
	return "hmac-sha256:postgresql_prueba:" + strings.Repeat(caracter, 64)
}
