package adminperfiles

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/ports"
)

type txSelectorAuditadoPrueba struct {
	pgx.Tx
	bruto                  string
	errConsulta, errCommit error
	carrerasPendientes     int
	carrerasCommit         int
	auditoriaCruzada       bool
	args                   []any
	commits, rollbacks     int
}

func (t *txSelectorAuditadoPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *txSelectorAuditadoPrueba) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	t.args = append([]any(nil), args...)
	return filaSelectorAuditadaPrueba{t}
}
func (t *txSelectorAuditadoPrueba) Commit(context.Context) error {
	t.commits++
	if t.carrerasCommit > 0 {
		t.carrerasCommit--
		return &pgconn.PgError{Code: "40001", Message: "detalle interno sintético"}
	}
	return t.errCommit
}
func (t *txSelectorAuditadoPrueba) Rollback(context.Context) error { t.rollbacks++; return nil }

type filaSelectorAuditadaPrueba struct{ tx *txSelectorAuditadoPrueba }

func (f filaSelectorAuditadaPrueba) Scan(destinos ...any) error {
	if f.tx.carrerasPendientes > 0 {
		f.tx.carrerasPendientes--
		return &pgconn.PgError{Code: "40001", Message: "detalle interno sintético"}
	}
	if f.tx.errConsulta != nil {
		return f.tx.errConsulta
	}
	if len(destinos) == 2 && len(f.tx.args) == 0 {
		*destinos[0].(*bool) = true
		*destinos[1].(*string) = "vec-admin.preperfil"
		return nil
	}
	if len(destinos) != 2 || len(f.tx.args) < 11 {
		return errors.New("fila sintética incompatible")
	}
	evento := f.tx.args[len(f.tx.args)-2].(string)
	ref := "aud_v3_p_" + strings.TrimPrefix(evento, "evento_")
	if f.tx.auditoriaCruzada {
		ref = "aud_v3_p_" + strings.Repeat("0", 32)
	}
	*destinos[0].(*[]byte) = []byte(f.tx.bruto)
	*destinos[1].(*string) = ref
	return nil
}

type poolSelectorAuditadoPrueba struct {
	tx       *txSelectorAuditadoPrueba
	opciones pgx.TxOptions
	plazos   []time.Time
	sinPlazo bool
	inicios  int
}

func (p *poolSelectorAuditadoPrueba) BeginTx(ctx context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	p.opciones = opciones
	p.inicios++
	if limite, ok := ctx.Deadline(); ok {
		p.plazos = append(p.plazos, limite)
	} else {
		p.sinPlazo = true
	}
	return p.tx, nil
}
func (*poolSelectorAuditadoPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaContextoFalsa{}
}

func escenarioSelectorAuditadoPrueba(t *testing.T, tx *txSelectorAuditadoPrueba) (*seleccionAuditadaPostgreSQL, context.Context, ObservacionADMIN, *poolSelectorAuditadoPrueba) {
	t.Helper()
	ahora := time.Date(2026, 10, 3, 14, 0, 0, 0, time.UTC)
	o := ObservacionADMIN{Entorno: "desarrollo", Host: "admin.example.invalid", Audiencia: "vec.admin.selector.v1", CertificadoSHA256: strings.Repeat("a", 64), CASHA256: strings.Repeat("b", 64), AutenticacionVerificadaEn: ahora.Add(-time.Minute), RevocacionVerificadaEn: ahora, CRLVigenteHasta: ahora.Add(time.Minute), CertificadoVigenteHasta: ahora.Add(time.Hour)}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	pool := &poolSelectorAuditadoPrueba{tx: tx}
	return &seleccionAuditadaPostgreSQL{base: &PostgreSQL{pool: pool, reloj: relojSeleccion{ahora}}}, ctx, o, pool
}

func TestListadoAuditadoNoEntregaDatosSinCommitYAcusePropio(t *testing.T) {
	const documento = `{"revision":0,"perfil_activo_ref":"","perfiles":[{"perfil_ref":"prf_aaaaaaaaaaaaaaaaaaaaaa","rol_version_ref":"rol:administracion_perfiles:v4","clave_i18n":"administracion.perfiles.rol","categoria_admin":"aplicacion"}]}`
	for _, caso := range []struct {
		nombre      string
		falloCommit error
		cruzada     bool
		esperado    bool
	}{
		{"confirmado", nil, false, true}, {"COMMIT incierto", errors.New("resultado no confirmado"), false, false}, {"asiento cruzado", nil, true, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &txSelectorAuditadoPrueba{bruto: documento, errCommit: caso.falloCommit, auditoriaCruzada: caso.cruzada}
			s, ctx, o, pool := escenarioSelectorAuditadoPrueba(t, tx)
			r, err := s.ListarPropiosAuditadosADMIN(ctx, o)
			if caso.esperado {
				if err != nil || len(r.Propios.Perfiles) != 1 || r.AuditoriaComunRef == "" || tx.commits != 1 {
					t.Fatalf("confirmación sintética inesperada: %v", err)
				}
				correlacion, _ := ports.CorrelacionIncidenciasPeticion(ctx)
				if tx.args[len(tx.args)-1] != "correlacion_"+correlacion || pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
					t.Fatal("se perdió correlación o aislamiento")
				}
			} else if err == nil || len(r.Propios.Perfiles) != 0 || r.AuditoriaComunRef != "" {
				t.Fatal("salió información sin confirmación y acuse propio")
			}
		})
	}
}

func TestSelectorDenegadoConfirmaAuditoriaAntesDeDevolverError(t *testing.T) {
	for _, fallo := range []error{nil, errors.New("COMMIT incierto"), &pgconn.PgError{Code: "42501", Message: "detalle interno sintético"}} {
		tx := &txSelectorAuditadoPrueba{bruto: `{"estado":"denegado","motivo_ref":"seleccion_revision_obsoleta"}`, errCommit: fallo}
		s, ctx, o, _ := escenarioSelectorAuditadoPrueba(t, tx)
		r, err := s.SeleccionarPerfilAuditadoADMIN(ctx, o, "prf_aaaaaaaaaaaaaaaaaaaaaa", 0)
		if err == nil || r.AuditoriaComunRef != "" || r.Seleccion.PerfilActivoRef != "" || tx.commits != 1 {
			t.Fatal("la denegación perdió su transacción común o entregó selección")
		}
		if (fallo == nil) != errors.Is(err, api.ErrConflictoEstado) {
			t.Fatal("se confundió COMMIT incierto con una revisión obsoleta")
		}
	}
}

func TestSelectorConflictoTecnicoNoEsCASObsoleto(t *testing.T) {
	for _, codigo := range []string{"40001", "40P01"} {
		tx := &txSelectorAuditadoPrueba{errConsulta: &pgconn.PgError{Code: codigo, Message: "detalle interno sintético"}}
		s, ctx, o, _ := escenarioSelectorAuditadoPrueba(t, tx)
		// El plazo corta los reintentos; agotados, el conflicto sigue sin ser CAS.
		ctx, cancelar := context.WithTimeout(ctx, 150*time.Millisecond)
		_, err := s.ListarPropiosAuditadosADMIN(ctx, o)
		cancelar()
		if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || errors.Is(err, api.ErrConflictoEstado) || tx.commits != 0 || tx.rollbacks < 2 {
			t.Fatalf("%s: se convirtió un conflicto técnico en CAS denegado o no se reintentó (rollbacks=%d)", codigo, tx.rollbacks)
		}
	}
}

// Varias peticiones simultáneas de la misma página chocan en la cadena común:
// la transacción abortada se repite entera y la lectura termina confirmada.
func TestSelectorRepiteTransaccionTrasCarreraSerializable(t *testing.T) {
	const documento = `{"revision":0,"perfil_activo_ref":"","perfiles":[{"perfil_ref":"prf_aaaaaaaaaaaaaaaaaaaaaa","rol_version_ref":"rol:administracion_perfiles:v4","clave_i18n":"administracion.perfiles.rol","categoria_admin":"aplicacion"}]}`
	tx := &txSelectorAuditadoPrueba{bruto: documento, carrerasPendientes: 2}
	s, ctx, o, _ := escenarioSelectorAuditadoPrueba(t, tx)
	r, err := s.ListarPropiosAuditadosADMIN(ctx, o)
	if err != nil || len(r.Propios.Perfiles) != 1 || r.AuditoriaComunRef == "" || tx.commits != 1 || tx.rollbacks != 3 {
		t.Fatalf("la carrera no se repitió con transacción nueva: err=%v commits=%d rollbacks=%d", err, tx.commits, tx.rollbacks)
	}
}

func TestPreflightSelectorAuditadoUsaTransaccionYCommit(t *testing.T) {
	for _, fallo := range []error{nil, errors.New("COMMIT incierto")} {
		tx := &txSelectorAuditadoPrueba{errCommit: fallo}
		s, ctx, _, pool := escenarioSelectorAuditadoPrueba(t, tx)
		err := acreditarSeleccionAuditada(ctx, s.base)
		if (err == nil) != (fallo == nil) || tx.commits != 1 || pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
			t.Fatal("se publicó el proveedor sin preflight transaccional confirmado")
		}
	}
}

func TestSeleccionAuditadaAdmiteUTCPostgreSQLYReeleccion(t *testing.T) {
	for _, caso := range []struct {
		nombre, fecha string
		revision      uint64
		valida        bool
	}{
		{"elección nueva UTC SQL", "2026-10-03T14:00:00+00:00", 0, true},
		{"reelección conserva fecha", "2026-10-03T13:59:00+00:00", 1, true},
		{"precisión inválida", "2026-10-03T14:00:00.000000001+00:00", 0, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &txSelectorAuditadoPrueba{bruto: `{"perfil_ref":"prf_aaaaaaaaaaaaaaaaaaaaaa","seleccion_revision":1,"seleccionada_en":"` + caso.fecha + `"}`}
			s, ctx, o, _ := escenarioSelectorAuditadoPrueba(t, tx)
			r, err := s.SeleccionarPerfilAuditadoADMIN(ctx, o, "prf_aaaaaaaaaaaaaaaaaaaaaa", caso.revision)
			if (err == nil) != caso.valida {
				t.Fatalf("compatibilidad de selección incorrecta: %v", err)
			}
			if caso.valida {
				fecha, _ := time.Parse(time.RFC3339Nano, caso.fecha)
				if !r.Seleccion.Valida() || !r.Seleccion.SeleccionadaEn.Equal(fecha) || r.Seleccion.Revision != 1 || tx.commits != 1 {
					t.Fatal("se perdió fecha, revisión o acuse de la selección")
				}
			} else if tx.commits != 0 || r.AuditoriaComunRef != "" {
				t.Fatal("se confirmó precisión no admitida")
			}
		})
	}
}

// vec-admin no acota el contexto de la petición: el selector pone su propio
// plazo a toda la operación, reintentos incluidos.
func TestSelectorAcotaReintentosConPlazoPropio(t *testing.T) {
	const documento = `{"revision":0,"perfil_activo_ref":"","perfiles":[{"perfil_ref":"prf_aaaaaaaaaaaaaaaaaaaaaa","rol_version_ref":"rol:administracion_perfiles:v4","clave_i18n":"administracion.perfiles.rol","categoria_admin":"aplicacion"}]}`
	tx := &txSelectorAuditadoPrueba{bruto: documento, carrerasPendientes: 1}
	s, ctx, o, pool := escenarioSelectorAuditadoPrueba(t, tx)
	antes := time.Now()
	if _, err := s.ListarPropiosAuditadosADMIN(ctx, o); err != nil {
		t.Fatal(err)
	}
	if pool.sinPlazo || len(pool.plazos) != 2 || pool.plazos[0] != pool.plazos[1] || pool.plazos[0].After(antes.Add(plazoSelectorAuditado+time.Second)) || plazoSelectorAuditado > 15*time.Second {
		t.Fatalf("los intentos no comparten un plazo acotado: %v sin_plazo=%v", pool.plazos, pool.sinPlazo)
	}
}

// Un 40001 en el COMMIT garantiza que no se aplicó nada: se repite entero.
func TestSelectorRepiteCommitAbortadoPorSerializacion(t *testing.T) {
	const documento = `{"revision":0,"perfil_activo_ref":"","perfiles":[{"perfil_ref":"prf_aaaaaaaaaaaaaaaaaaaaaa","rol_version_ref":"rol:administracion_perfiles:v4","clave_i18n":"administracion.perfiles.rol","categoria_admin":"aplicacion"}]}`
	tx := &txSelectorAuditadoPrueba{bruto: documento, carrerasCommit: 1}
	s, ctx, o, _ := escenarioSelectorAuditadoPrueba(t, tx)
	r, err := s.ListarPropiosAuditadosADMIN(ctx, o)
	if err != nil || len(r.Propios.Perfiles) != 1 || r.AuditoriaComunRef == "" || tx.commits != 2 {
		t.Fatalf("COMMIT abortado no repetido: err=%v commits=%d", err, tx.commits)
	}
	// La marca conserva la clase de error para los demás consumidores.
	tx2 := &txSelectorAuditadoPrueba{carrerasCommit: 1}
	_, _, _, pool := escenarioSelectorAuditadoPrueba(t, tx2)
	err = (&PostgreSQL{pool: pool, reloj: relojSeleccion{time.Now()}}).transaccion(context.Background(), func(pgx.Tx) error { return nil })
	if !errors.Is(err, api.ErrConfiguracionIncompleta) || strings.Contains(err.Error(), "detalle interno") {
		t.Fatalf("COMMIT abortado sin clase o con detalle del proveedor: %v", err)
	}
}

type relojSecuenciaSelector struct {
	instantes []time.Time
	llamadas  *int
}

func (r relojSecuenciaSelector) Ahora() time.Time {
	i := min(*r.llamadas, len(r.instantes)-1)
	*r.llamadas++
	return r.instantes[i]
}

// Si la observación mTLS caduca entre dos intentos, no se abre otro.
func TestSelectorRevalidaObservacionEnCadaIntento(t *testing.T) {
	tx := &txSelectorAuditadoPrueba{carrerasPendientes: 1}
	s, ctx, o, pool := escenarioSelectorAuditadoPrueba(t, tx)
	ahora := time.Date(2026, 10, 3, 14, 0, 0, 0, time.UTC)
	llamadas := 0
	// Comprobación inicial y primer intento vigentes; después, CRL caducada.
	s.base.reloj = relojSecuenciaSelector{instantes: []time.Time{ahora, ahora, ahora.Add(2 * time.Minute)}, llamadas: &llamadas}
	_, err := s.ListarPropiosAuditadosADMIN(ctx, o)
	if !errors.Is(err, api.ErrAutenticacionRequerida) || pool.inicios != 1 || tx.commits != 0 {
		t.Fatalf("reintento con observación caducada: err=%v transacciones=%d", err, pool.inicios)
	}
}
