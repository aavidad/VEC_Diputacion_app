package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type registradorCapacidadesPrueba struct {
	t          *testing.T
	pool       *poolCapacidadesPrueba
	ahora      time.Time
	ordenes    []ports.DatosOrdenIntentoAuditoria
	fallos     []error
	acuse      func(*ports.AcuseIntentoAuditoria)
	canceladas int
}

func (r *registradorCapacidadesPrueba) AppendIntentoAuditoria(ctx context.Context, orden ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	r.t.Helper()
	if r.pool.comienzos > 0 && r.pool.falloInicio == nil && r.pool.tx.rollbacks != 1 {
		r.t.Fatal("se auditó antes de cerrar la transacción original")
	}
	if ctx.Err() != nil {
		r.canceladas++
	}
	datos, err := orden.Datos()
	if err != nil {
		r.t.Fatal(err)
	}
	r.ordenes = append(r.ordenes, datos)
	if len(r.fallos) >= len(r.ordenes) && r.fallos[len(r.ordenes)-1] != nil {
		return ports.AcuseIntentoAuditoria{}, r.fallos[len(r.ordenes)-1]
	}
	acuse := ports.AcuseIntentoAuditoria{AuditoriaRef: "aud_v3_" + datos.IntentoRef, Secuencia: 1,
		HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef, RegistradaEn: r.ahora}
	if r.acuse != nil {
		r.acuse(&acuse)
	}
	return acuse, nil
}

func TestCapacidadesIntentosNominalesTrasCerrarOperacion(t *testing.T) {
	for _, caso := range []string{"sistemas", "asignacion_ausente", "rol_ausente", "actor_caducado", "fuente_error", "catalogo_error", "instantanea_invalida", "emisor_error", "emisor_denegado", "inicio_error", "sql_denegado", "sql_error", "acuse_lectura_invalido", "commit_incierto", "commit_denegado", "cancelada", "cancelada_tras_consulta"} {
		t.Run(caso, func(t *testing.T) {
			f, ctx, s, pool, emisor, salida := capacidadesPrueba(t)
			b, _ := json.Marshal(salida)
			pool.tx.fila = filaFalsa{dato: b}
			denegado := false
			switch caso {
			case "sistemas":
				catalogo := f.catalogo.(catalogoCapacidadesPrueba)
				catalogo.rol.CategoriaAdmin = "sistemas"
				f.catalogo, denegado = catalogo, true
			case "asignacion_ausente":
				f.fuente = fuenteCapacidadesPrueba{err: ports.ErrAsignacionPerfilNoEncontrada}
				denegado = true
			case "rol_ausente":
				f.catalogo = catalogoCapacidadesPrueba{err: ports.ErrVersionRolNoEncontrada}
				denegado = true
			case "actor_caducado":
				// El par registrado conserva valor histórico; no se modifica
				// para aparentar una identidad todavía vigente.
				f.reloj = relojFijo(s.Actor.Instantanea.VigenteHasta.Add(time.Hour))
				denegado = true
			case "fuente_error":
				f.fuente = fuenteCapacidadesPrueba{err: errors.New("dato privado")}
			case "catalogo_error":
				f.catalogo = catalogoCapacidadesPrueba{err: errors.New("dato privado")}
			case "instantanea_invalida":
				f.fuente = fuenteCapacidadesPrueba{}
			case "emisor_error":
				emisor.err = errors.New("dato privado")
			case "emisor_denegado":
				emisor.err, denegado = domain.ErrAutorizacionDenegada, true
			case "inicio_error":
				pool.falloInicio = errors.New("dato privado")
			case "sql_denegado":
				pool.tx.fila = filaFalsa{err: &pgconn.PgError{Code: "42501", Message: "dato privado"}}
				denegado = true
			case "sql_error":
				pool.tx.fila = filaFalsa{err: errors.New("dato privado")}
			case "acuse_lectura_invalido":
				pool.tx.fila = filaFalsa{dato: []byte(`{}`)}
			case "commit_incierto":
				pool.tx.falloCommit = errors.New("dato privado")
			case "commit_denegado":
				pool.tx.falloCommit = &pgconn.PgError{Code: "42501", Message: "dato privado"}
			case "cancelada", "cancelada_tras_consulta":
				var cancelar context.CancelFunc
				ctx, cancelar = context.WithCancel(ctx)
				defer cancelar()
				if caso == "cancelada" {
					cancelar()
				} else {
					pool.tx.fila = filaCapacidadesCanceladaPrueba{fila: pool.tx.fila, cancelar: cancelar}
				}
			}
			resultado, err := f.Capacidades(ctx, s.Actor, s.Evidencia)
			if err == nil || !reflect.DeepEqual(resultado, api.Capacidades{}) || strings.Contains(err.Error(), "privad") {
				t.Fatalf("se devolvieron datos o un error privado: %+v %v", resultado, err)
			}
			registrador := f.intentos.(*registradorCapacidadesPrueba)
			if len(registrador.ordenes) != 1 {
				t.Fatal("el intento nominal no se registró exactamente una vez")
			}
			orden := registrador.ordenes[0]
			correlacion, _ := ports.CorrelacionIncidenciasPeticion(ctx)
			esperado := domain.ResultadoIntentoAuditoriaError
			motivo := f.configuracionIntentos.MotivoError
			if denegado {
				esperado, motivo = domain.ResultadoIntentoAuditoriaDenegado, f.configuracionIntentos.MotivoDenegado
			}
			if orden.Datos.Resultado != esperado || orden.Datos.Motivo != motivo || orden.Datos.Accion != "administracion.perfiles.consultar" ||
				orden.Datos.ModuloID != "administracion" || orden.Datos.RecursoRef != s.Actor.PerfilActivoRef ||
				orden.Datos.CorrelacionRef != "correlacion_"+correlacion || orden.Datos.Proceso != f.configuracionIntentos.Proceso ||
				orden.Datos.Canal != f.configuracionIntentos.Canal || orden.Datos.FinalidadRef != f.configuracionIntentos.FinalidadRef ||
				orden.ResultadoContexto.HuellaSHA256 != s.Evidencia.ResultadoContexto.HuellaSHA256 ||
				orden.ResultadoContexto.Contexto.PersonaRef != s.Actor.PersonaRef || orden.ResultadoContexto.Contexto.PerfilActivoRef != s.Actor.PerfilActivoRef ||
				orden.Vinculo.ValidarPara(s.Evidencia.ResultadoContexto) != nil || !strings.HasPrefix(orden.IntentoRef, "intento_") {
				t.Fatal("el registro sustituyó resultado, identidad, origen, recurso o correlación")
			}
			if denegado != errors.Is(err, domain.ErrAutorizacionDenegada) {
				t.Fatal("el error nominal perdió su clasificación")
			}
			if strings.HasPrefix(caso, "cancelada") && (registrador.canceladas != 1 || !errors.Is(err, context.Canceled)) {
				t.Fatal("la cancelación omitió el registro histórico o cambió su resultado")
			}
		})
	}
}

func TestCapacidadesCadaInvocacionTieneSuReferenciaDeIntento(t *testing.T) {
	f, ctx, s, _, _, _ := capacidadesPrueba(t)
	f.fuente = fuenteCapacidadesPrueba{err: ports.ErrAsignacionPerfilNoEncontrada}
	for range 2 {
		if _, err := f.Capacidades(ctx, s.Actor, s.Evidencia); !errors.Is(err, domain.ErrAutorizacionDenegada) {
			t.Fatal("no confirmó la auditoría de la denegación")
		}
	}
	r := f.intentos.(*registradorCapacidadesPrueba)
	if len(r.ordenes) != 2 || r.ordenes[0].IntentoRef == r.ordenes[1].IntentoRef ||
		r.ordenes[0].Datos.CorrelacionRef != r.ordenes[1].Datos.CorrelacionRef {
		t.Fatal("se fusionaron invocaciones o se generó una correlación sustitutiva")
	}
}

type filaCapacidadesCanceladaPrueba struct {
	fila     pgx.Row
	cancelar context.CancelFunc
}

func (f filaCapacidadesCanceladaPrueba) Scan(destinos ...any) error {
	err := f.fila.Scan(destinos...)
	f.cancelar()
	return err
}

func TestCapacidadesReintentaOrdenOriginalYExigeAcuse(t *testing.T) {
	for _, caso := range []string{"replay", "no_disponible", "orden_invalida", "conflicto", "error_ajeno", "acuse_ajeno", "acuse_ausente", "acuse_fecha_ausente", "acuse_secuencia_ausente", "acuse_huella_ausente"} {
		t.Run(caso, func(t *testing.T) {
			f, ctx, s, pool, _, _ := capacidadesPrueba(t)
			pool.tx.fila = filaFalsa{err: &pgconn.PgError{Code: "42501"}}
			r := f.intentos.(*registradorCapacidadesPrueba)
			switch caso {
			case "replay":
				r.fallos = []error{ports.ErrIntentoAuditoriaNoDisponible}
			case "no_disponible":
				r.fallos = []error{ports.ErrIntentoAuditoriaNoDisponible, ports.ErrIntentoAuditoriaNoDisponible}
			case "orden_invalida":
				r.fallos = []error{ports.ErrOrdenIntentoAuditoriaInvalida}
			case "conflicto":
				r.fallos = []error{ports.ErrIntentoAuditoriaConflicto}
			case "error_ajeno":
				r.fallos = []error{errors.New("dato privado")}
			case "acuse_ajeno":
				r.acuse = func(a *ports.AcuseIntentoAuditoria) { a.CorrelacionRef = "correlacion_" + strings.Repeat("0", 32) }
			case "acuse_ausente":
				r.acuse = func(a *ports.AcuseIntentoAuditoria) { *a = ports.AcuseIntentoAuditoria{} }
			case "acuse_fecha_ausente":
				r.acuse = func(a *ports.AcuseIntentoAuditoria) { a.RegistradaEn = time.Time{} }
			case "acuse_secuencia_ausente":
				r.acuse = func(a *ports.AcuseIntentoAuditoria) { a.Secuencia = 0 }
			case "acuse_huella_ausente":
				r.acuse = func(a *ports.AcuseIntentoAuditoria) { a.HuellaSHA256 = "" }
			}
			resultado, err := f.Capacidades(ctx, s.Actor, s.Evidencia)
			esperado := ports.ErrAutoridadAdministracionPerfilesNoDisponible
			llamadas := 1
			if caso == "replay" || caso == "no_disponible" {
				llamadas = 2
				if len(r.ordenes) != 2 || !reflect.DeepEqual(r.ordenes[0], r.ordenes[1]) {
					t.Fatal("el reintento generó otra referencia u otra orden")
				}
			}
			if caso == "replay" {
				esperado = domain.ErrAutorizacionDenegada
			}
			if !errors.Is(err, esperado) || !reflect.DeepEqual(resultado, api.Capacidades{}) || len(r.ordenes) != llamadas {
				t.Fatal("se devolvió un error nominal sin acuse o se reintentó un rechazo definitivo")
			}
		})
	}
}

func TestCapacidadesNoInventaPerfilConfiguracionNiIdentidad(t *testing.T) {
	for _, caso := range []string{"registrador_ausente", "registrador_nulo", "proceso_ausente", "canal_ausente", "canal_ajeno", "finalidad_ausente", "motivo_denegado_ausente", "motivo_error_ausente", "sin_perfil", "actor_ajeno", "evidencia_ausente", "sin_correlacion", "contexto_nil", "fuente_nil"} {
		t.Run(caso, func(t *testing.T) {
			f, ctx, s, pool, emisor, _ := capacidadesPrueba(t)
			r := f.intentos.(*registradorCapacidadesPrueba)
			switch caso {
			case "registrador_ausente":
				f.intentos = nil
			case "registrador_nulo":
				f.intentos = (*registradorCapacidadesPrueba)(nil)
			case "proceso_ausente":
				f.configuracionIntentos.Proceso = ""
			case "canal_ausente":
				f.configuracionIntentos.Canal = ""
			case "canal_ajeno":
				f.configuracionIntentos.Canal = "externa_personal"
			case "finalidad_ausente":
				f.configuracionIntentos.FinalidadRef = ""
			case "motivo_denegado_ausente":
				f.configuracionIntentos.MotivoDenegado = domain.ReferenciaEntradaCatalogo{}
			case "motivo_error_ausente":
				f.configuracionIntentos.MotivoError = domain.ReferenciaEntradaCatalogo{}
			case "sin_perfil":
				s.Actor.PerfilActivoRef = ""
			case "actor_ajeno":
				s.Actor.PersonaRef = "per_" + strings.Repeat("c", 22)
			case "evidencia_ausente":
				s.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{}
			case "sin_correlacion":
				ctx = context.Background()
			case "contexto_nil":
				ctx = nil
			case "fuente_nil":
				f = nil
			}
			resultado, err := f.Capacidades(ctx, s.Actor, s.Evidencia)
			if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || !reflect.DeepEqual(resultado, api.Capacidades{}) ||
				len(r.ordenes) != 0 || emisor.llamadas != 0 || pool.comienzos != 0 {
				t.Fatal("una dependencia o evidencia ausente abrió capacidades o inventó un registro nominal")
			}
		})
	}
}
