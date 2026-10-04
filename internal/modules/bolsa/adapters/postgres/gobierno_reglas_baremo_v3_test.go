package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	ports "vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Dobles de transporte: estos tests no prueban firmas ni consumo PostgreSQL.
type filaGobiernoV3Prueba struct{ scan func(...any) error }

func (f filaGobiernoV3Prueba) Scan(d ...any) error { return f.scan(d...) }

type txGobiernoV3Prueba struct {
	acreditado                      bool
	codigo                          string
	canon, recibo, acceso           []byte
	replay                          bool
	commits, rollbacks, operaciones int
	args                            []any
	falloCommit, falloOperacion     error
	falloAcreditacion               error
}

func (t *txGobiernoV3Prueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("SET"), nil
}
func (t *txGobiernoV3Prueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	if q == acreditarGobiernoReglasV3SQL {
		return filaGobiernoV3Prueba{func(d ...any) error {
			if t.falloAcreditacion != nil {
				return t.falloAcreditacion
			}
			*d[0].(*bool) = t.acreditado
			return nil
		}}
	}
	t.operaciones++
	t.args = args
	return filaGobiernoV3Prueba{func(d ...any) error {
		if t.falloOperacion != nil {
			return t.falloOperacion
		}
		*d[0].(*string) = t.codigo
		*d[1].(*[]byte) = bytes.Clone(t.canon)
		*d[2].(*[]byte) = bytes.Clone(t.recibo)
		*d[3].(*[]byte) = bytes.Clone(t.acceso)
		*d[4].(*bool) = t.replay
		return nil
	}}
}

func TestGobiernoReglasV3PreflightNoResuelveNamespacePrivado(t *testing.T) {
	// La resolución regprocedure/regnamespace fuerza USAGE aunque sólo se
	// quiera observar metadatos. El LOGIN sólo ejecuta la operación de Bolsa.
	for _, prohibido := range []string{"to_regprocedure", "::regprocedure", "::regnamespace", " LIKE ", " ILIKE "} {
		if strings.Contains(acreditarGobiernoReglasV3SQL, prohibido) {
			t.Fatalf("preflight depende de resolución o coincidencia parcial: %s", prohibido)
		}
	}
	for _, requerido := range []string{
		"JOIN pg_catalog.pg_namespace ns ON ns.oid=p.pronamespace",
		"ns.nspname='vec_autorizacion_atestada_v3'",
		"p.proname='registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada'::pg_catalog.name",
		"p.proargtypes=ARRAY[t.b,t.b,t.b,t.b,t.n,t.n,t.b,t.b,t.b,t.b]::pg_catalog.oidvector",
		"p.proargnames[11:17]=ARRAY['decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo']",
		"o.rolname='vec_autorizacion_atestada_v3_propietario'",
		"NOT pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')",
	} {
		if !strings.Contains(acreditarGobiernoReglasV3SQL, requerido) {
			t.Fatalf("preflight perdió comprobación nominal: %s", requerido)
		}
	}
}

func TestGobiernoReglasV3PreflightDeniegaAntesDelNegocio(t *testing.T) {
	for _, caso := range []string{"metadata_incompatible", "error_catalogo"} {
		t.Run(caso, func(t *testing.T) {
			tx := &txGobiernoV3Prueba{}
			if caso == "error_catalogo" {
				tx.falloAcreditacion = &pgconn.PgError{Code: "42501", Message: "detalle privado de esquema"}
			}
			_, err := repoPGGobiernoPrueba(tx).abrirGobiernoReglasV3(context.Background())
			if !errors.Is(err, app.ErrGobiernoV3NoDisponible) || strings.Contains(fmt.Sprint(err), "privado") ||
				tx.operaciones != 0 || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatalf("preflight incompatible no cerró la transacción: %v", err)
			}
		})
	}
}
func (t *txGobiernoV3Prueba) Commit(context.Context) error   { t.commits++; return t.falloCommit }
func (t *txGobiernoV3Prueba) Rollback(context.Context) error { t.rollbacks++; return nil }
func repoPGGobiernoPrueba(tx *txGobiernoV3Prueba) *RepositorioGobiernoReglasBaremoV3PostgreSQL {
	return &RepositorioGobiernoReglasBaremoV3PostgreSQL{rol: "vec_bolsa_reglas_baremo_ejecutor_gobierno", iniciar: func(context.Context) (transaccionGobiernoReglasV3, error) { return tx, nil }}
}

func ordenPGGobiernoPrueba(t *testing.T, ahora time.Time, decision string) (ports.OrdenAltaBorradorReglasV3, app.MaterialGobiernoV3) {
	t.Helper()
	datos, err := os.ReadFile("../../application/simulacionbaremo/testdata/reglas_a.json")
	if err != nil {
		t.Fatal(err)
	}
	conjunto, err := reglas.RestaurarConjuntoReglasBaremoConHuellaSHA256(datos, shaPGGobiernoV3(datos))
	if err != nil {
		t.Fatal(err)
	}
	ref, err := reglas.NuevaReferenciaVersionada("motivos_autorizacion", 1, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	motivo, err := reglas.NuevoMotivoCatalogadoReglasBaremo(ref, "motivo_"+strings.Repeat("d", 32))
	if err != nil {
		t.Fatal(err)
	}
	persona := "per_" + strings.Repeat("a", 24)
	perfil := "prf_" + strings.Repeat("b", 24)
	version, err := reglas.NuevaVersionGobernadaReglasBaremo(conjunto, persona, motivo, ahora)
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := version.RepresentacionCanonica()
	estado, _ := version.VinculoEstado()
	id := conjunto.Identidad()
	motivoVEC := vd.ReferenciaEntradaCatalogo{CatalogoID: ref.Referencia(), CatalogoVersion: 1, CatalogoHuellaSHA256: ref.HuellaSHA256(), EntradaClave: motivo.Clave()}
	motivoCanon, err := vd.RepresentacionCanonicaMotivoAutorizacionV2(motivoVEC)
	if err != nil {
		t.Fatal(err)
	}
	contenido := estado.Contenido()
	m := app.MaterialGobiernoV3{Esquema: "vec.bolsa.gobierno-borrador.material.v3", Operacion: "alta_borrador", Accion: "bolsa.reglas_baremo.borrador.crear", ModuloID: "bolsa", TipoRecurso: "intencion_gobierno_reglas_baremo", Finalidad: "gobierno_reglas_baremo", PersonaRef: persona, PerfilRef: perfil, ConvocatoriaRef: id.ConvocatoriaRef(), ExpedienteRef: id.ExpedienteRef(), Estado: app.EstadoMaterialGobiernoV3{Referencia: contenido.Referencia(), Version: contenido.Version(), HuellaContenidoSHA256: contenido.HuellaSHA256(), Revision: 1, HuellaEstadoSHA256: estado.HuellaEstadoSHA256()}, VersionCanonica: canon, ClaveOperacion: strings.Repeat("a", 32), HuellaSolicitudSHA256: strings.Repeat("c", 64), MotivoCanonico: motivoCanon, SolicitadaEn: ahora.Format("2006-01-02T15:04:05.000000Z")}
	m.HuellaSolicitudSHA256, err = app.HuellaSolicitudAltaBorradorV3(persona, app.PeticionAltaBorradorV3{Conjunto: conjunto, Motivo: motivo, ClaveOperacion: m.ClaveOperacion})
	if err != nil {
		t.Fatal(err)
	}
	material, _ := json.Marshal(m)
	v3 := exportacionPGGobiernoPrueba(t, m, material, ahora, decision)
	return ports.OrdenAltaBorradorReglasV3{MaterialCanonico: material, HuellaMaterialSHA256: shaPGGobiernoV3(material), VersionCanonica: canon, HuellaVersionSHA256: shaPGGobiernoV3(canon), EstadoPropuesto: estado, ClaveOperacion: m.ClaveOperacion, HuellaSolicitudSHA256: m.HuellaSolicitudSHA256, Autorizacion: v3}, m
}
func exportacionPGGobiernoPrueba(t *testing.T, m app.MaterialGobiernoV3, material []byte, ahora time.Time, decision string) vp.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	cuenta := vd.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + strings.Repeat("a", 24), Metodo: vd.AuthMethodCertificate, Garantia: vd.AuthAssuranceHigh}
	actor, err := vd.NuevoContextoActor(cuenta, vd.InstantaneaContextoActor{VinculoRef: "vca_" + strings.Repeat("a", 24), VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: m.PersonaRef, PersonaVersion: 1, PerfilActivoRef: m.PerfilRef, PerfilVersion: 1, Estado: vd.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}, ahora)
	if err != nil {
		t.Fatal(err)
	}
	contexto, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	ref, tipo := "reglas-baremo:"+m.Estado.HuellaEstadoSHA256, "version_reglas_baremo_gobernada"
	if m.Operacion == "alta_borrador" {
		ref, tipo = "intencion-reglas-baremo:"+m.HuellaSolicitudSHA256, "intencion_gobierno_reglas_baremo"
	}
	recurso := vd.RecursoAutorizable{Referencia: ref, ModuloID: "bolsa", Tipo: tipo, Ambitos: map[string]string{"convocatoria_ref": m.ConvocatoriaRef, "expediente_ref": m.ExpedienteRef}, Atributos: map[string]string{"material_sha256": shaPGGobiernoV3(material)}}
	hash, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3(decision, shaPGGobiernoV3([]byte(decision)), shaPGGobiernoV3(m.MotivoCanonico), "rca:fixture", shaPGGobiernoV3(contexto), m.Accion, ref, hash, app.AudienciaGobiernoBorradorReglasV3, ahora, ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	v, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte(decision), m.MotivoCanonico, contexto, 1, 1, []byte("payload_fixture"), []byte("sobre_fixture"), []byte("evidencia_fixture"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func accesoPGGobiernoPrueba(v vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, ahora time.Time) accesoPGGobiernoV3 {
	r := v.ResumenCapacidad()
	return accesoPGGobiernoV3{DecisionRef: r.DecisionRef(), DecisionSHA: r.DecisionHuellaSHA256(), EfectoRef: r.EfectoRef(), EfectoSHA: r.EfectoHuellaSHA256(), ConsumoSHA: shaPGGobiernoV3([]byte(r.DecisionRef() + "consumo")), AuditoriaRef: "audit:" + r.DecisionRef(), ConsumidaEn: ahora}
}
func respuestaPGGobiernoPrueba(t *testing.T, o ports.OrdenAltaBorradorReglasV3, m app.MaterialGobiernoV3, ahora time.Time) *txGobiernoV3Prueba {
	t.Helper()
	acceso := accesoPGGobiernoPrueba(o.Autorizacion, ahora)
	recibo := reciboPGGobiernoV3{ReciboRef: "recibo:fixture", Clave: o.ClaveOperacion, HuellaSolicitud: o.HuellaSolicitudSHA256, Estado: m.Estado, TransaccionRef: "tx:fixture", AuditoriaRef: acceso.AuditoriaRef, OutboxRef: "outbox:fixture", ConsumoOriginal: acceso, ConfirmadaEn: ahora}
	a, _ := json.Marshal(acceso)
	r, _ := json.Marshal(recibo)
	return &txGobiernoV3Prueba{acreditado: true, codigo: "creado", canon: bytes.Clone(o.VersionCanonica), recibo: r, acceso: a}
}

func TestGobiernoReglasV3PostgresAltaYReplayCanonOriginal(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 8, 0, 0, 123456000, time.UTC)
	original, m := ordenPGGobiernoPrueba(t, ahora, "decision:original")
	tx := respuestaPGGobiernoPrueba(t, original, m, ahora)
	repo := repoPGGobiernoPrueba(tx)
	resultado, err := repo.ConfirmarAltaBorrador(context.Background(), original)
	if err != nil {
		t.Fatal(err)
	}
	if tx.commits != 1 || tx.operaciones != 1 || len(tx.args) != 11 || !bytes.Equal(tx.args[0].([]byte), original.MaterialCanonico) || !bytes.Equal(resultado.Recibo.VersionCanonica, original.VersionCanonica) {
		t.Fatal("no conservó bytea/retorno exactos")
	}
	nuevo, m2 := ordenPGGobiernoPrueba(t, ahora.Add(time.Second), "decision:reintento")
	tx2 := respuestaPGGobiernoPrueba(t, original, m, ahora)
	tx2.codigo, tx2.replay = "replay", true
	tx2.acceso, _ = json.Marshal(accesoPGGobiernoPrueba(nuevo.Autorizacion, ahora.Add(time.Second)))
	replay, err := repoPGGobiernoPrueba(tx2).ConfirmarAltaBorrador(context.Background(), nuevo)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replay || !replay.Recibo.ConfirmadaEn.Equal(ahora) || !bytes.Equal(replay.Recibo.VersionCanonica, original.VersionCanonica) || bytes.Equal(replay.Recibo.VersionCanonica, m2.VersionCanonica) || tx2.commits != 1 {
		t.Fatal("replay adoptó fecha o canon de propuesta nueva")
	}
}

func TestGobiernoReglasV3PostgresRollbackAntesDeCommit(t *testing.T) {
	for _, caso := range []string{"canon_corrupto", "huella_material", "otra_huella_recibo", "sin_outbox", "acceso_ajeno", "sin_consumidor", "contrato_json", "intencion_inventada", "commit_ambiguo"} {
		t.Run(caso, func(t *testing.T) {
			ahora := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
			o, m := ordenPGGobiernoPrueba(t, ahora, "decision:fixture")
			tx := respuestaPGGobiernoPrueba(t, o, m, ahora)
			switch caso {
			case "canon_corrupto":
				tx.canon = append(tx.canon, ' ')
			case "huella_material":
				o.HuellaMaterialSHA256 = strings.Repeat("f", 64)
			case "otra_huella_recibo":
				var r reciboPGGobiernoV3
				_ = json.Unmarshal(tx.recibo, &r)
				r.Estado.HuellaEstadoSHA256 = strings.Repeat("f", 64)
				tx.recibo, _ = json.Marshal(r)
			case "sin_outbox":
				var r reciboPGGobiernoV3
				_ = json.Unmarshal(tx.recibo, &r)
				r.OutboxRef = ""
				tx.recibo, _ = json.Marshal(r)
			case "acceso_ajeno":
				var a accesoPGGobiernoV3
				_ = json.Unmarshal(tx.acceso, &a)
				a.DecisionRef = "decision:otra"
				tx.acceso, _ = json.Marshal(a)
			case "sin_consumidor":
				tx.acreditado = false
			case "contrato_json":
				tx.recibo = append([]byte(`{"desconocido":1,`), tx.recibo[1:]...)

			case "intencion_inventada":
				m.HuellaSolicitudSHA256 = strings.Repeat("f", 64)
				o.MaterialCanonico, _ = json.Marshal(m)
				o.HuellaMaterialSHA256 = shaPGGobiernoV3(o.MaterialCanonico)
				o.HuellaSolicitudSHA256 = m.HuellaSolicitudSHA256
				o.Autorizacion = exportacionPGGobiernoPrueba(t, m, o.MaterialCanonico, ahora, "decision:intencion_falsa")
				tx = respuestaPGGobiernoPrueba(t, o, m, ahora)
			case "commit_ambiguo":
				tx.falloCommit = errors.New("respuesta privada perdida")
			}
			_, err := repoPGGobiernoPrueba(tx).ConfirmarAltaBorrador(context.Background(), o)
			if err == nil {
				t.Fatal("aceptó retorno inválido")
			}
			if caso == "commit_ambiguo" {
				if !errors.Is(err, ErrConfirmacionGobiernoReglasV3Incierta) || tx.commits != 1 {
					t.Fatal("COMMIT ambiguo no reconciliable")
				}
			} else if tx.commits != 0 {
				t.Fatal("confirmó antes de cotejar")
			}
			if caso != "huella_material" && tx.rollbacks == 0 {
				t.Fatal("no revirtió la transacción")
			}
		})
	}
}

func TestGobiernoReglasV3PostgresLecturaAusenteAutorizada(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	alta, m := ordenPGGobiernoPrueba(t, ahora, "decision:alta")
	m.Operacion, m.Accion, m.Finalidad, m.TipoRecurso = "consultar_exacta", "bolsa.reglas_baremo.version.consultar", "consulta_gobierno_reglas_baremo", "version_reglas_baremo_gobernada"
	m.VersionCanonica = nil
	m.ClaveOperacion, m.HuellaSolicitudSHA256 = "", ""
	datos, _ := json.Marshal(m)
	v := exportacionPGGobiernoPrueba(t, m, datos, ahora, "decision:lectura")
	acceso, _ := json.Marshal(accesoPGGobiernoPrueba(v, ahora))
	tx := &txGobiernoV3Prueba{acreditado: true, codigo: "no_encontrada", acceso: acceso}
	id := reglas.IdentidadConjuntoReglasBaremo{}
	conjunto, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(alta.VersionCanonica, alta.HuellaVersionSHA256)
	if err != nil {
		t.Fatal(err)
	}
	c, err := conjunto.Conjunto()
	if err != nil {
		t.Fatal(err)
	}
	id = c.Identidad()
	orden := ports.OrdenConsultaGobiernoReglasV3{Selector: ports.SelectorGobiernoReglasV3{Identidad: id, Estado: alta.EstadoPropuesto}, MaterialCanonico: datos, HuellaMaterialSHA256: shaPGGobiernoV3(datos), Autorizacion: v}
	_, err = repoPGGobiernoPrueba(tx).ObtenerExacta(context.Background(), orden)
	if !errors.Is(err, ports.ErrReglasBaremoNoEncontradas) || tx.commits != 1 || tx.operaciones != 1 {
		t.Fatalf("lectura no auditó ausencia: %v", err)
	}
	var material app.MaterialGobiernoV3
	_ = json.Unmarshal(tx.args[0].([]byte), &material)
	if material.Operacion != "consultar_exacta" || len(material.VersionCanonica) != 0 {
		t.Fatal("GET se convirtió en alta")
	}
	if _, err := NuevoRepositorioGobiernoReglasBaremoV3PostgreSQL(context.Background(), nil, "rol:inventado"); err == nil {
		t.Fatal("constructor sin infraestructura abrió runtime")
	}
	if got := errorGobiernoReglasV3(&pgconn.PgError{Code: "23505", Message: "dato privado"}); !errors.Is(got, ports.ErrClaveIdempotenciaReglasReutilizada) || strings.Contains(fmt.Sprint(got), "privado") {
		t.Fatal("error no nominal")
	}
}

func TestGobiernoReglasV3PostgresRecuperaReciboConSelectorConocido(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	alta, m := ordenPGGobiernoPrueba(t, ahora, "decision:original")
	tx := respuestaPGGobiernoPrueba(t, alta, m, ahora)
	version, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(alta.VersionCanonica, alta.HuellaVersionSHA256)
	if err != nil {
		t.Fatal(err)
	}
	conjunto, err := version.Conjunto()
	if err != nil {
		t.Fatal(err)
	}
	m.Operacion, m.Accion, m.Finalidad, m.TipoRecurso = "recuperar_recibo", "bolsa.reglas_baremo.recibo.consultar", "consulta_gobierno_reglas_baremo", "version_reglas_baremo_gobernada"
	m.VersionCanonica = nil
	m.SolicitadaEn = ahora.Add(time.Second).Format("2006-01-02T15:04:05.000000Z")
	material, _ := json.Marshal(m)
	v := exportacionPGGobiernoPrueba(t, m, material, ahora.Add(time.Second), "decision:recuperacion")
	tx.codigo = "recuperado"
	tx.acceso, _ = json.Marshal(accesoPGGobiernoPrueba(v, ahora.Add(time.Second)))
	orden := ports.OrdenConsultaGobiernoReglasV3{Selector: ports.SelectorGobiernoReglasV3{Identidad: conjunto.Identidad(), Estado: alta.EstadoPropuesto}, MaterialCanonico: material, HuellaMaterialSHA256: shaPGGobiernoV3(material), ClaveOperacion: m.ClaveOperacion, HuellaSolicitudSHA256: m.HuellaSolicitudSHA256, Autorizacion: v}
	recuperado, err := repoPGGobiernoPrueba(tx).RecuperarRecibo(context.Background(), orden)
	if err != nil {
		t.Fatal(err)
	}
	if !recuperado.Existe || !recuperado.Recibo.ConfirmadaEn.Equal(ahora) || !bytes.Equal(recuperado.Recibo.VersionCanonica, alta.VersionCanonica) || recuperado.Acceso.DecisionRef == recuperado.Recibo.ConsumoOriginal.DecisionRef || tx.commits != 1 {
		t.Fatal("recuperación no conservó recibo histórico y acceso actual")
	}
}

func TestGobiernoReglasV3PostgresDeniegaCruceAccionesDeLecturaAntesDeSQL(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	alta, original := ordenPGGobiernoPrueba(t, ahora, "decision:original")
	version, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(alta.VersionCanonica, alta.HuellaVersionSHA256)
	if err != nil {
		t.Fatal(err)
	}
	conjunto, err := version.Conjunto()
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range []string{"consultar_exacta", "recuperar_recibo"} {
		t.Run(op, func(t *testing.T) {
			m := original
			m.Operacion, m.Accion, m.Finalidad, m.TipoRecurso = op, "bolsa.reglas_baremo.version.consultar", "consulta_gobierno_reglas_baremo", "version_reglas_baremo_gobernada"
			m.VersionCanonica = nil
			if op == "consultar_exacta" {
				m.Accion = "bolsa.reglas_baremo.recibo.consultar"
				m.ClaveOperacion, m.HuellaSolicitudSHA256 = "", ""
			}
			material, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			v := exportacionPGGobiernoPrueba(t, m, material, ahora, "decision:cruzada")
			orden := ports.OrdenConsultaGobiernoReglasV3{Selector: ports.SelectorGobiernoReglasV3{Identidad: conjunto.Identidad(), Estado: alta.EstadoPropuesto},
				MaterialCanonico: material, HuellaMaterialSHA256: shaPGGobiernoV3(material), ClaveOperacion: m.ClaveOperacion, HuellaSolicitudSHA256: m.HuellaSolicitudSHA256, Autorizacion: v}
			tx := &txGobiernoV3Prueba{}
			repo := repoPGGobiernoPrueba(tx)
			if op == "consultar_exacta" {
				_, err = repo.ObtenerExacta(context.Background(), orden)
			} else {
				_, err = repo.RecuperarRecibo(context.Background(), orden)
			}
			if !errors.Is(err, app.ErrGobiernoV3Prohibido) || tx.operaciones != 0 || tx.commits != 0 || tx.rollbacks != 0 {
				t.Fatalf("acción cruzada llegó a SQL: %v", err)
			}
		})
	}
}
