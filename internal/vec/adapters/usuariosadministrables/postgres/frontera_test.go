package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type revalidadorAdminPrueba struct {
	autenticacion domain.AutenticacionRevalidadaV1
}

func (r revalidadorAdminPrueba) RevalidarAutenticacionActorV1(context.Context, domain.SolicitudRevalidacionAutenticacionActorV1) (domain.AutenticacionRevalidadaV1, error) {
	return r.autenticacion, nil
}

type resolutorAdminPrueba struct {
	resultado domain.ResultadoContextoActorRegistradoV2
}

func (r resolutorAdminPrueba) ResolverContextoActorRegistradoV2(context.Context, domain.SolicitudContextoActor) (domain.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type relojAdminPrueba struct{ ahora time.Time }

func (r relojAdminPrueba) Ahora() time.Time { return r.ahora }

func evidenciaAdminPrueba(t *testing.T, ahora time.Time) (domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) {
	t.Helper()
	resultado, anterior, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_"+strings.Repeat("a", 22), "prf_"+strings.Repeat("b", 22), domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := anterior.Datos()
	if err != nil {
		t.Fatal(err)
	}
	a := datos.Autenticacion()
	a.CuentaOrdinariaRef = "cta_" + strings.Repeat("f", 24)
	a.CuentaPrivilegiada = true
	a.Superficie = domain.SuperficieAutenticacionAdministracionPrivilegiadaV1
	vinculo, registrado, err := domain.CrearVinculoAutenticacionActorV2ConResultado(context.Background(), revalidadorAdminPrueba{a},
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: a.AutenticacionRef, SesionRef: a.SesionRef},
		resolutorAdminPrueba{resultado}, domain.SolicitudContextoActor{Cuenta: domain.CuentaAutenticadaContextoActor{CuentaRef: resultado.Contexto.Instantanea.CuentaRef, Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh}, PerfilActivoRef: resultado.Contexto.PerfilActivoRef}, relojAdminPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	e := domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: registrado, Vinculo: vinculo}
	if e.ValidarEn(registrado.Contexto, ahora) != nil {
		t.Fatal("evidencia administrativa inválida")
	}
	return registrado.Contexto, e
}

type capturadorIntentoUsuarios struct {
	datos []domain.DatosIntentoAuditoria
	ahora time.Time
}

func (c *capturadorIntentoUsuarios) AppendIntentoAuditoria(_ context.Context, o ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	d, err := o.Datos()
	if err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	c.datos = append(c.datos, d.Datos)
	return ports.AcuseIntentoAuditoria{AuditoriaRef: "aud_v3_" + strings.Repeat("a", 32), Secuencia: 1, HuellaSHA256: strings.Repeat("b", 64), CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: c.ahora}, nil
}

func TestCursorIncompatibleAuditaConjuntoPrivadoSinGuardarCursor(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	actor, evidencia := evidenciaAdminPrueba(t, ahora)
	registrador := &capturadorIntentoUsuarios{ahora: ahora}
	f := &Fuente{ambito: ambito{"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "unidad_admin_sintetica"}, config: Configuracion{Proceso: "vec_admin", Canal: "administracion_privilegiada",
		MotivoDenegado: motivoIntentoPrueba(), MotivoError: motivoIntentoPrueba()}, intentos: registrador}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	lista, err := f.ListarUsuarios(ctx, actor, evidencia, ports.FiltrosUsuariosAdministrables{Cursor: "usuarios:cursor_incompatible"})
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || len(lista.Personas) != 0 || len(registrador.datos) != 1 {
		t.Fatalf("rechazo y auditoría: %v %+v", err, registrador.datos)
	}
	conjunto, _ := conjuntoUsuarios(f.ambito)
	d := registrador.datos[0]
	if d.Accion != accionListar || d.RecursoRef != conjunto || d.Resultado != domain.ResultadoIntentoAuditoriaDenegado || strings.Contains(d.RecursoRef, "cursor") {
		t.Fatalf("dato auditado: %+v", d)
	}
	_, err = f.ConsultarUsuario(ctx, actor, evidencia, "persona_invalida")
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || len(registrador.datos) != 1 {
		t.Fatalf("Persona malformada fabricó auditoría/recurso: %v %+v", err, registrador.datos)
	}
}

func motivoIntentoPrueba() domain.ReferenciaEntradaCatalogo {
	return domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: "motivo_" + strings.Repeat("f", 32)}
}

type acreditacionPoolPrueba struct{ consultas int }

func (*acreditacionPoolPrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	panic("no debe abrir transacción")
}
func (p *acreditacionPoolPrueba) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	p.consultas++
	if sql != acreditarSQL {
		panic("consulta de acreditación distinta")
	}
	return filaAcreditacionPrueba{}
}

type filaAcreditacionPrueba struct{}

func (filaAcreditacionPrueba) Scan(dest ...any) error { *dest[0].(*bool) = false; return nil }

type fuenteAcreditacionPrueba struct{}

func (fuenteAcreditacionPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (domain.InstantaneaAutorizacion, error) {
	panic("no debe consultar fuente")
}

type emisorAcreditacionPrueba struct{}

func (emisorAcreditacionPrueba) EmitirLecturaUsuariosAdministrables(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion, ports.EmisionUsuariosAdministrables) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	panic("no debe emitir")
}

func TestConstructorNoAbreFuenteConACLNoAcreditada(t *testing.T) {
	p := &acreditacionPoolPrueba{}
	c := Configuracion{OrganizacionRef: "org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", UnidadRef: "unidad_admin_sintetica", Proceso: "vec_admin", Canal: "administracion_privilegiada", MotivoDenegado: motivoIntentoPrueba(), MotivoError: motivoIntentoPrueba()}
	f, err := nueva(context.Background(), p, emisorAcreditacionPrueba{}, fuenteAcreditacionPrueba{}, &capturadorIntentoUsuarios{}, relojAdminPrueba{time.Now().UTC()}, c)
	if f != nil || !errors.Is(err, ports.ErrLecturaUsuariosAdministrablesNoDisponible) || p.consultas != 1 {
		t.Fatalf("constructor: fuente=%v error=%v consultas=%d", f, err, p.consultas)
	}
}

func TestDependenciasSoloExponenDenegacionTipada(t *testing.T) {
	if !errors.Is(clasificarDependencia(domain.ErrAutorizacionDenegada), domain.ErrAutorizacionDenegada) {
		t.Fatal("denegación tipada perdida")
	}
	for _, e := range []error{errors.New("dato privado"), &pgconn.PgError{Code: "42501", Message: "dato privado"}} {
		x := clasificarDependencia(e)
		if !errors.Is(x, ports.ErrLecturaUsuariosAdministrablesNoDisponible) || strings.Contains(x.Error(), "dato privado") || errors.Is(x, domain.ErrAutorizacionDenegada) {
			t.Fatalf("error de dependencia filtrado: %v", x)
		}
	}
}
