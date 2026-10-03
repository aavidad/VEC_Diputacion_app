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
	"github.com/jackc/pgx/v5/pgxpool"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteCapacidadesPrueba struct {
	instantanea domain.InstantaneaAutorizacion
	err         error
}

func (f fuenteCapacidadesPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (domain.InstantaneaAutorizacion, error) {
	return f.instantanea, f.err
}

type catalogoCapacidadesPrueba struct {
	rol ports.RolAdministrable
	err error
}

func (c catalogoCapacidadesPrueba) ResolverRolAdministrable(context.Context, string) (ports.RolAdministrable, error) {
	return c.rol, c.err
}

type emisorCapacidadesPrueba struct {
	t        *testing.T
	ahora    time.Time
	efecto   Efecto
	llamadas int
	err      error
}

func (e *emisorCapacidadesPrueba) EmitirAdministracionPerfiles(_ context.Context, _ domain.ContextoActor, _ domain.EvidenciaSesionAdministracionPerfiles, _ domain.InstantaneaAutorizacion, efecto Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	e.efecto = efecto
	e.efecto.Material = append([]byte(nil), efecto.Material...)
	return materialSintetico(e.t, efecto, e.ahora), e.err
}

type txCapacidadesPrueba struct {
	txFalsa
	consulta string
	material string
	args     int
}

func (t *txCapacidadesPrueba) QueryRow(_ context.Context, consulta string, args ...any) pgx.Row {
	t.consultas++
	t.consulta, t.args = consulta, len(args)
	if len(args) > 0 {
		t.material, _ = args[0].(string)
	}
	return t.fila
}

type poolCapacidadesPrueba struct {
	conexion
	tx          *txCapacidadesPrueba
	comienzos   int
	opciones    pgx.TxOptions
	falloInicio error
}

func (p *poolCapacidadesPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	p.comienzos++
	p.opciones = opciones
	return p.tx, p.falloInicio
}

func capacidadesPrueba(t *testing.T) (*FuenteCapacidades, context.Context, domain.SolicitudActoAdministracionPerfiles, *poolCapacidadesPrueba, *emisorCapacidadesPrueba, map[string]any) {
	t.Helper()
	s, _, catalogo, _ := contratoV2Prueba(t)
	var rol rolJSON
	if err := decodificar(catalogo.roles[s.InstantaneaAutorizacion.VersionRol.Referencia()], &rol); err != nil {
		t.Fatal(err)
	}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	correlacion, _ := ports.CorrelacionIncidenciasPeticion(ctx)
	ahora := s.Actor.ResueltoEn
	e := &emisorCapacidadesPrueba{t: t, ahora: ahora}
	p := &poolCapacidadesPrueba{tx: &txCapacidadesPrueba{}}
	f := &FuenteCapacidades{pool: p, emisor: e, reloj: relojFijo(ahora), fuente: fuenteCapacidadesPrueba{instantanea: s.InstantaneaAutorizacion},
		catalogo: catalogoCapacidadesPrueba{rol: ports.RolAdministrable{VersionRef: rol.VersionRef, Clase: rol.Clase, CategoriaAdmin: *rol.CategoriaAdmin,
			HuellaSHA256: rol.HuellaSHA256, VigenteDesde: rol.VigenteDesde, VigenteHasta: rol.VigenteHasta, UnidadRequerida: *rol.UnidadRequerida}}}
	f.intentos = &registradorCapacidadesPrueba{t: t, pool: p, ahora: ahora}
	vinculo, err := s.Evidencia.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	f.configuracionIntentos = ConfiguracionIntentosCapacidades{Proceso: "vec-server", Canal: string(vinculo.Superficie), FinalidadRef: "gestion_perfiles",
		MotivoDenegado: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_auditoria", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "acceso_denegado"},
		MotivoError:    domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_auditoria", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "consulta_error"}}
	salida := map[string]any{"version": "1", "actor_persona_ref": s.Actor.PersonaRef, "acciones": []string{},
		"perfil_activo_ref": s.Actor.PerfilActivoRef, "asignacion_perfil_ref": s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(),
		"correlacion_ref": "correlacion_" + correlacion, "auditoria_ref": "aud_v3_" + strings.Repeat("a", 32), "registrada_en": ahora}
	return f, ctx, s, p, e, salida
}

func TestCapacidadesConsumoYAuditoriaAntesDePublicar(t *testing.T) {
	f, ctx, s, pool, emisor, salida := capacidadesPrueba(t)
	b, _ := json.Marshal(salida)
	pool.tx.fila = filaFalsa{dato: b}
	resultado, err := f.Capacidades(ctx, s.Actor, s.Evidencia)
	if err != nil || resultado.Version != "1" || resultado.ActorPersonaRef != s.Actor.PersonaRef || resultado.Acciones == nil || len(resultado.Acciones) != 0 {
		t.Fatalf("respuesta confirmada: %+v %v", resultado, err)
	}
	if pool.comienzos != 1 || pool.tx.commits != 1 || pool.tx.rollbacks != 1 || pool.tx.consulta != capacidadesLecturaSQL || pool.tx.args != 11 ||
		pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
		t.Fatal("consulta no consumida en transacción nominal")
	}
	if len(f.intentos.(*registradorCapacidadesPrueba).ordenes) != 0 {
		t.Fatal("el permitido duplicó la auditoría común de AD168")
	}
	var material map[string]any
	if json.Unmarshal([]byte(pool.tx.material), &material) != nil || material["correlacion_ref"] != salida["correlacion_ref"] ||
		emisor.efecto.CorrelacionAccesoRef != salida["correlacion_ref"] || material["actor_persona_ref"] != s.Actor.PersonaRef ||
		material["actor_perfil_ref"] != s.Actor.PerfilActivoRef || material["asignacion_perfil_ref"] != s.InstantaneaAutorizacion.AsignacionPerfil.Referencia() ||
		len(material) != 8 || emisor.efecto.Referencia != s.Actor.PerfilActivoRef {
		t.Fatal("se sustituyó correlación o identidad acreditada")
	}
}

func TestCapacidadesAcuseAusenteAjenoOEscrituraNoPublica(t *testing.T) {
	for _, caso := range []string{"version", "actor_persona_ref", "perfil_activo_ref", "asignacion_perfil_ref", "correlacion_ref", "auditoria_ref", "registrada_en", "acciones", "actor_ajeno", "correlacion_ajena", "auditoria_vacia", "fecha_futura", "escritura", "consulta_no_montada", "duplicada", "desconocido", "commit_incierto", "sql_denegado", "sql_error"} {
		t.Run(caso, func(t *testing.T) {
			f, ctx, s, pool, _, salida := capacidadesPrueba(t)
			switch caso {
			case "actor_ajeno":
				salida["actor_persona_ref"] = "per_" + strings.Repeat("c", 22)
			case "correlacion_ajena":
				salida["correlacion_ref"] = "correlacion_" + strings.Repeat("f", 32)
			case "auditoria_vacia":
				salida["auditoria_ref"] = ""
			case "fecha_futura":
				salida["registrada_en"] = f.reloj.Ahora().Add(time.Hour)
			case "escritura":
				salida["acciones"] = []string{"aplicar_ordinario"}
			case "consulta_no_montada":
				salida["acciones"] = []string{"consultar"}
			case "duplicada":
				salida["acciones"] = []string{"consultar", "consultar"}
			case "desconocido":
				salida["secreto"] = "dato privado"
			case "commit_incierto":
				pool.tx.falloCommit = errors.New("dato privado de conexión")
			case "sql_denegado", "sql_error":
			default:
				delete(salida, caso)
			}
			b, _ := json.Marshal(salida)
			pool.tx.fila = filaFalsa{dato: b}
			if caso == "sql_denegado" {
				pool.tx.fila = filaFalsa{err: &pgconn.PgError{Code: "42501", Message: "dato privado"}}
			}
			if caso == "sql_error" {
				pool.tx.fila = filaFalsa{err: errors.New("dato privado")}
			}
			resultado, err := f.Capacidades(ctx, s.Actor, s.Evidencia)
			if err == nil || resultado.Version != "" || resultado.ActorPersonaRef != "" || resultado.Acciones != nil || strings.Contains(err.Error(), "privad") {
				t.Fatalf("salida abierta: %+v %v", resultado, err)
			}
			if pool.tx.commits != map[bool]int{true: 1, false: 0}[caso == "commit_incierto"] || pool.tx.rollbacks != 1 {
				t.Fatal("confirmó una respuesta incompleta o ajena")
			}
			if caso == "sql_denegado" && !errors.Is(err, domain.ErrAutorizacionDenegada) {
				t.Fatal("la denegación SQL perdió su clasificación")
			}
		})
	}
}

func TestCapacidadesSinCategoriaAplicacionOCorrelacionNoEmiteV3(t *testing.T) {
	for _, caso := range []string{"sin_correlacion", "sistemas", "categoria_ausente", "huella_ajena", "rol_ajeno", "actor_ajeno", "evidencia_ausente", "fuente_error", "catalogo_error", "cancelada"} {
		t.Run(caso, func(t *testing.T) {
			f, ctx, s, pool, emisor, _ := capacidadesPrueba(t)
			catalogo := f.catalogo.(catalogoCapacidadesPrueba)
			switch caso {
			case "sin_correlacion":
				ctx = context.Background()
			case "sistemas":
				catalogo.rol.CategoriaAdmin = "sistemas"
			case "categoria_ausente":
				catalogo.rol.CategoriaAdmin = ""
			case "huella_ajena":
				catalogo.rol.HuellaSHA256 = strings.Repeat("f", 64)
			case "rol_ajeno":
				catalogo.rol.VersionRef = "rol:sistemas:v1"
			case "actor_ajeno":
				s.Actor.PersonaRef = "per_" + strings.Repeat("c", 22)
			case "evidencia_ausente":
				s.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{}
			case "fuente_error":
				f.fuente = fuenteCapacidadesPrueba{err: errors.New("dato privado")}
			case "catalogo_error":
				catalogo.err = errors.New("dato privado")
			case "cancelada":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			f.catalogo = catalogo
			resultado, err := f.Capacidades(ctx, s.Actor, s.Evidencia)
			if err == nil || resultado.Version != "" || resultado.Acciones != nil || emisor.llamadas != 0 || pool.comienzos != 0 {
				t.Fatal("entrada no acreditada llegó a V3 o SQL")
			}
		})
	}
}

func TestFuenteCapacidadesMantieneSeisConsultasCerradas(t *testing.T) {
	f, ctx, s, pool, emisor, _ := capacidadesPrueba(t)
	_, personas := f.BuscarPersonas(ctx, s.Actor, s.Evidencia, api.ConsultaPersonas{})
	_, persona := f.ConsultarPersona(ctx, s.Actor, s.Evidencia, s.Actor.PersonaRef)
	_, roles := f.ListarRoles(ctx, s.Actor, s.Evidencia)
	_, propuestas := f.ListarPropuestas(ctx, s.Actor, s.Evidencia)
	_, propuesta := f.ConsultarPropuesta(ctx, s.Actor, s.Evidencia, "propuesta_admin:"+strings.Repeat("a", 32))
	_, recibo := f.ConsultarRecibo(ctx, s.Actor, s.Evidencia, "recibo_admin:"+strings.Repeat("a", 32))
	for _, err := range []error{personas, persona, roles, propuestas, propuesta, recibo} {
		if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
			t.Fatal("lectura heredada abierta desde fuente limitada")
		}
	}
	if emisor.llamadas != 0 || pool.comienzos != 0 {
		t.Fatal("una lectura cerrada intentó consumir V3")
	}
}

func TestConstructorCapacidadesSigueCerradoSinAuditorComunDenegados(t *testing.T) {
	f, ctx, _, _, _, _ := capacidadesPrueba(t)
	for _, pool := range []*pgxpool.Pool{nil, {}} {
		resultado, err := NuevaFuenteCapacidades(ctx, pool, f.emisor, f.fuente, f.catalogo, f.reloj)
		if resultado != nil || !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
			t.Fatal("constructor abrió el proveedor sin auditoría común de fallos")
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if resultado, err := NuevaFuenteCapacidades(ctx, &pgxpool.Pool{}, f.emisor, f.fuente, f.catalogo, f.reloj); resultado != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("constructor perdió cancelación")
	}
}
