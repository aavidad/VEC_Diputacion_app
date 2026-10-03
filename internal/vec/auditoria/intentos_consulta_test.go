package auditoria_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	bolsafuente "vec-diputacion-granada/internal/modules/bolsa/adapters/auditoriaconsulta"
	ctfuente "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/auditoriaconsulta"
	"vec-diputacion-granada/internal/vec/auditoria"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

const decisionConsulta = "decision:auditoria:prueba"

type exportadorConsulta struct {
	material ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (e exportadorConsulta) ExportarMaterialParaConsumidor() (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return e.material, nil
}
func (exportadorConsulta) String() string         { return "[EXPORTADOR-PRUEBA]" }
func (e exportadorConsulta) LogValue() slog.Value { return slog.StringValue(e.String()) }

type emisorConsulta struct {
	t        *testing.T
	instante time.Time
	eventos  *[]string
	err      error
	cancelar context.CancelFunc
}

func (e *emisorConsulta) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, s domain.SolicitudAutorizacionLigadaV3, resultado domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, ports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	*e.eventos = append(*e.eventos, "emisor")
	if e.cancelar != nil {
		e.cancelar()
	}
	if e.err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, e.err
	}
	datos, err := s.Datos()
	if err != nil {
		e.t.Fatal(err)
	}
	c, err := pruebas.NuevaConcesionV3Prueba(datosConcesion(e.instante, datos.Recurso))
	if err != nil {
		e.t.Fatal(err)
	}
	if c.Decision.ValidarPara(s) != nil {
		e.t.Fatal("decisión no ligada a la solicitud original")
	}
	dh, _ := domain.HuellaSHA256DecisionAutorizacionV3(c.Decision)
	mh, _ := domain.HuellaSHA256MotivoAutorizacionV2(datos.ReferenciaMotivo)
	rh, _ := datos.Recurso.HuellaContextoAutorizacionSHA256()
	resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionConsulta, dh, mh, resultado.RegistroContextoRef, resultado.HuellaSHA256, datos.Accion, datos.Recurso.Referencia, rh, auditoria.AudienciaConsumo, e.instante.Add(time.Second), e.instante.Add(5*time.Second))
	if err != nil {
		e.t.Fatal(err)
	}
	dc, _ := domain.RepresentacionCanonicaDecisionAutorizacionV3(c.Decision)
	mc, _ := domain.RepresentacionCanonicaMotivoAutorizacionV2(datos.ReferenciaMotivo)
	// Material estructural sintético; no acredita COSE ni persistencia PostgreSQL.
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(clave.Public())
	if err != nil {
		e.t.Fatal(err)
	}
	material, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{'b'}, ports.TamanoMinimoCapacidadCanonicaV3), resumen, dc, mc, resultado.RepresentacionCanonica, resultado.Contexto.Instantanea.PersonaVersion, resultado.Contexto.Instantanea.PerfilVersion, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		e.t.Fatal(err)
	}
	return c.Decision, c.Confirmacion, exportadorConsulta{material}, nil
}
func datosConcesion(instante time.Time, r domain.RecursoAutorizable) pruebas.DatosConcesionV3Prueba {
	return pruebas.DatosConcesionV3Prueba{Instante: instante, PersonaRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl", Accion: auditoria.AccionConsultar, Recurso: r, Finalidad: "auditoria_rrhh", Campos: auditoria.CamposPermitidos(), Obligaciones: []string{}, DecisionRef: decisionConsulta}
}

type filasConsulta struct {
	pgx.Rows
	eventos   *[]string
	siguiente bool
	scan      func(...any) error
	err       error
}

func (r *filasConsulta) Next() bool {
	if !r.siguiente {
		return false
	}
	r.siguiente = false
	return true
}
func (r *filasConsulta) Scan(dest ...any) error { return r.scan(dest...) }
func (r *filasConsulta) Err() error             { return r.err }
func (r *filasConsulta) Close()                 { *r.eventos = append(*r.eventos, "rows-close") }

type txConsulta struct {
	pgx.Tx
	t                                *testing.T
	eventos                          *[]string
	rows                             *filasConsulta
	queryErr, commitErr, rollbackErr error
	cancelar                         context.CancelFunc
	confirmado                       bool
}

func (x *txConsulta) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (x *txConsulta) Query(context.Context, string, ...any) (pgx.Rows, error) {
	*x.eventos = append(*x.eventos, "query")
	return x.rows, x.queryErr
}
func (x *txConsulta) Commit(context.Context) error {
	*x.eventos = append(*x.eventos, "commit")
	x.confirmado = x.commitErr == nil
	return x.commitErr
}
func (x *txConsulta) Rollback(ctx context.Context) error {
	if ctx.Err() != nil {
		x.t.Fatal("rollback heredó la cancelación HTTP")
	}
	if _, ok := ctx.Deadline(); !ok {
		x.t.Fatal("rollback sin plazo")
	}
	*x.eventos = append(*x.eventos, "rollback")
	if x.cancelar != nil {
		x.cancelar()
	}
	if x.confirmado {
		return pgx.ErrTxClosed
	}
	return x.rollbackErr
}

type poolConsulta struct{ tx *txConsulta }

func (p poolConsulta) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite {
		p.tx.t.Fatal("transacción no nominal")
	}
	*p.tx.eventos = append(*p.tx.eventos, "begin")
	return p.tx, nil
}

type registroConsulta struct {
	t           *testing.T
	eventos     *[]string
	ordenes     []ports.OrdenIntentoAuditoria
	err         error
	invalido    bool
	primerFallo bool
}

func (r *registroConsulta) AppendIntentoAuditoria(ctx context.Context, o ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	if ctx.Err() != nil {
		r.t.Fatal("append heredó la cancelación")
	}
	if _, ok := ctx.Deadline(); !ok {
		r.t.Fatal("append sin plazo")
	}
	*r.eventos = append(*r.eventos, "append")
	r.ordenes = append(r.ordenes, o)
	if r.primerFallo && len(r.ordenes) == 1 {
		return ports.AcuseIntentoAuditoria{}, ports.ErrIntentoAuditoriaNoDisponible
	}
	if r.err != nil {
		return ports.AcuseIntentoAuditoria{}, r.err
	}
	if r.invalido {
		return ports.AcuseIntentoAuditoria{}, nil
	}
	d, err := o.Datos()
	if err != nil {
		r.t.Fatal(err)
	}
	return ports.AcuseIntentoAuditoria{AuditoriaRef: "aud_v3_i_prueba", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: time.Now().UTC().Truncate(time.Microsecond)}, nil
}

type escenarioConsulta struct {
	servicio *auditoria.Servicio
	peticion auditoria.Peticion
	cfg      auditoria.ConfiguracionIntentosConsulta
	tx       *txConsulta
	emisor   *emisorConsulta
	registro *registroConsulta
	eventos  *[]string
}

func nuevoEscenarioConsulta(t *testing.T, fuente string) escenarioConsulta {
	t.Helper()
	instante := time.Now().UTC().Truncate(time.Microsecond).Add(-2 * time.Second)
	c, err := pruebas.NuevaConcesionV3Prueba(datosConcesion(instante, domain.RecursoAutorizable{Referencia: "expediente:ct:prueba", ModuloID: auditoria.ModuloAutorizacion, Tipo: auditoria.TipoRecurso, Ambitos: map[string]string{"expediente_ref": "expediente:ct:prueba", "fuente": "ct"}}))
	if err != nil {
		t.Fatal(err)
	}
	d, err := c.Solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(instante, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	recurso := "expediente:ct:prueba"
	if fuente == "bolsa" {
		recurso = "participacion:bolsa:prueba"
	}
	p := auditoria.Peticion{Filtro: auditoria.Filtro{Fuente: fuente, ExpedienteRef: recurso, Desde: instante.Add(-time.Hour), Hasta: instante.Add(time.Hour), Limite: 5, FinalidadRef: "auditoria_rrhh", MotivoRef: d.ReferenciaMotivo.Referencia()}, Contexto: auditoria.ContextoConsulta{Vinculo: vinculo, Resultado: resultado, Motivo: d.ReferenciaMotivo, Correlacion: d.Correlacion}}
	eventos := []string{}
	tx := &txConsulta{t: t, eventos: &eventos}
	tx.rows = &filasConsulta{eventos: &eventos}
	ct, err := ctfuente.NuevaFuente(poolConsulta{tx})
	if err != nil {
		t.Fatal(err)
	}
	bolsa, err := bolsafuente.NuevaFuente(poolConsulta{tx})
	if err != nil {
		t.Fatal(err)
	}
	emisor := &emisorConsulta{t: t, instante: instante, eventos: &eventos}
	registro := &registroConsulta{t: t, eventos: &eventos}
	cfg := auditoria.ConfiguracionIntentosConsulta{Proceso: "vec-rrhh-prueba", Canal: "interna_corporativa", RecursoCTRef: "expediente:ct:prueba", RecursoBolsaRef: "participacion:bolsa:prueba", FinalidadRef: "auditoria_rrhh", Motivo: d.ReferenciaMotivo, Plazo: time.Second}
	s, err := auditoria.NuevoServicioConIntentos(emisor, ct, bolsa, registro, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioConsulta{s, p, cfg, tx, emisor, registro, &eventos}
}
func asignarFila(dest []any, valores []any) error {
	for n, v := range valores {
		reflect.ValueOf(dest[n]).Elem().Set(reflect.ValueOf(v))
	}
	return nil
}
func filaConsulta(e escenarioConsulta, invalida bool) func(...any) error {
	return func(dest ...any) error {
		instante := e.emisor.instante
		if e.peticion.Filtro.Fuente == "ct" {
			estado := []byte(`{"fase":"inicio","estado":"en_curso"}`)
			if invalida {
				estado = []byte(`{"nombre":"dato personal"}`)
			}
			return asignarFila(dest, []any{"evento:1", "ct", "contratacion_temporal", "alta", "per_0123456789abcdefghijkl", "confirmado", e.peticion.Filtro.ExpedienteRef, "recibo:prueba", "", strings.Repeat("a", 64), "", instante, []byte(nil), estado, true})
		}
		campo := "correo"
		valor := "version:2"
		if invalida {
			valor = "persona@example.invalid"
		}
		return asignarFila(dest, []any{"evento:1", instante, "bolsa.consultar", "per_0123456789abcdefghijkl", "confirmado", e.peticion.Filtro.ExpedienteRef, "recibo:prueba", (*string)(nil), &campo, (*string)(nil), &valor})
	}
}
func TestIntentosConsultasFuentesCierranAntesDelAppend(t *testing.T) {
	for _, fuente := range []string{"ct", "bolsa"} {
		for _, caso := range []string{"42501", "scan", "proyeccion", "cancelacion", "commit-ambiguo", "rollback-fallido"} {
			t.Run(fuente+"/"+caso, func(t *testing.T) {
				e := nuevoEscenarioConsulta(t, fuente)
				ctx, cancelar := context.WithCancel(t.Context())
				defer cancelar()
				switch caso {
				case "42501":
					e.tx.queryErr = &pgconn.PgError{Code: "42501", Message: "detalle reservado"}
				case "scan":
					e.tx.rows.siguiente = true
					e.tx.rows.scan = func(...any) error { return errors.New("dato reservado") }
				case "proyeccion":
					e.tx.rows.siguiente = true
					e.tx.rows.scan = filaConsulta(e, true)
				case "cancelacion":
					e.tx.rows.siguiente = true
					e.tx.rows.scan = func(...any) error { cancelar(); return context.Canceled }
				case "commit-ambiguo":
					e.tx.commitErr = errors.New("resultado indeterminado")
				case "rollback-fallido":
					e.tx.queryErr = errors.New("consulta fallida")
					e.tx.rollbackErr = errors.New("cierre no confirmado")
				}
				pagina, err := e.servicio.Consultar(ctx, e.peticion)
				esperado := auditoria.ErrNoDisponible
				if caso == "42501" {
					esperado = auditoria.ErrDenegada
				}
				if !errors.Is(err, esperado) || len(pagina.Registros) != 0 || pagina.SiguienteCursor != "" {
					t.Fatalf("salida insegura: %v", err)
				}
				if len(e.registro.ordenes) != 1 {
					t.Fatalf("intentos=%d eventos=%v", len(e.registro.ordenes), *e.eventos)
				}
				eventos := *e.eventos
				if eventos[len(eventos)-2] != "rollback" || eventos[len(eventos)-1] != "append" {
					t.Fatalf("orden=%v", eventos)
				}
				orden, _ := e.registro.ordenes[0].Datos()
				resultado := domain.ResultadoIntentoAuditoriaError
				if caso == "42501" {
					resultado = domain.ResultadoIntentoAuditoriaDenegado
				}
				if orden.Datos.Resultado != resultado || orden.Datos.RecursoRef != e.peticion.Filtro.ExpedienteRef || orden.ResultadoContexto.HuellaSHA256 != e.peticion.Contexto.Resultado.HuellaSHA256 {
					t.Fatal("resultado o identidad originales perdidos")
				}
			})
		}
	}
}
func TestIntentosConsultaPermitidaNoDuplicaAuditoria(t *testing.T) {
	for _, fuente := range []string{"ct", "bolsa"} {
		t.Run(fuente, func(t *testing.T) {
			e := nuevoEscenarioConsulta(t, fuente)
			e.tx.rows.siguiente = true
			e.tx.rows.scan = filaConsulta(e, false)
			pagina, err := e.servicio.Consultar(t.Context(), e.peticion)
			if err != nil || len(pagina.Registros) != 1 || len(e.registro.ordenes) != 0 {
				t.Fatalf("permitido: %v, filas=%d", err, len(pagina.Registros))
			}
		})
	}
}
func TestIntentosConsultaFalloEmisorYCancelacionInicial(t *testing.T) {
	for _, caso := range []string{"emisor", "denegado", "cancelacion-inicial", "cancelacion-emisor"} {
		t.Run(caso, func(t *testing.T) {
			e := nuevoEscenarioConsulta(t, "ct")
			ctx, cancelar := context.WithCancel(t.Context())
			defer cancelar()
			switch caso {
			case "emisor":
				e.emisor.err = errors.New("validador caído")
			case "denegado":
				e.emisor.err = domain.ErrAutorizacionDenegada
			case "cancelacion-inicial":
				cancelar()
			case "cancelacion-emisor":
				e.emisor.cancelar = cancelar
			}
			pagina, err := e.servicio.Consultar(ctx, e.peticion)
			if err == nil || len(pagina.Registros) != 0 || len(e.registro.ordenes) != 1 {
				t.Fatalf("intento perdido: %v eventos=%v", err, *e.eventos)
			}
			for _, evento := range *e.eventos {
				if evento == "begin" {
					t.Fatal("la fuente recibió material inválido")
				}
			}
		})
	}
}
func TestIntentosConsultaAcuseYRecuperacionConMismaReferencia(t *testing.T) {
	for _, caso := range []string{"caido", "acuse-invalido", "recuperacion"} {
		t.Run(caso, func(t *testing.T) {
			e := nuevoEscenarioConsulta(t, "ct")
			e.tx.queryErr = &pgconn.PgError{Code: "42501"}
			switch caso {
			case "caido":
				e.registro.err = ports.ErrIntentoAuditoriaNoDisponible
			case "acuse-invalido":
				e.registro.invalido = true
			case "recuperacion":
				e.registro.primerFallo = true
			}
			pagina, err := e.servicio.Consultar(t.Context(), e.peticion)
			esperado := auditoria.ErrNoDisponible
			if caso == "recuperacion" {
				esperado = auditoria.ErrDenegada
			}
			if !errors.Is(err, esperado) || len(pagina.Registros) != 0 {
				t.Fatalf("acuse fallido permitió salida: %v", err)
			}
			if len(e.registro.ordenes) == 2 {
				a, _ := e.registro.ordenes[0].Datos()
				b, _ := e.registro.ordenes[1].Datos()
				if a.IntentoRef != b.IntentoRef || !reflect.DeepEqual(a, b) {
					t.Fatal("el reintento cambió orden/referencia")
				}
			}
		})
	}
}
func TestIntentosConsultaNoReflejaEntradaLibre(t *testing.T) {
	for _, campo := range []string{"recurso", "finalidad", "motivo", "fuente"} {
		t.Run(campo, func(t *testing.T) {
			e := nuevoEscenarioConsulta(t, "ct")
			switch campo {
			case "recurso":
				e.peticion.Filtro.ExpedienteRef = "persona:identificador-libre"
			case "finalidad":
				e.peticion.Filtro.FinalidadRef = "finalidad-libre"
			case "motivo":
				e.peticion.Filtro.MotivoRef = "motivo-libre"
			case "fuente":
				e.peticion.Filtro.Fuente = "otra"
			}
			pagina, err := e.servicio.Consultar(t.Context(), e.peticion)
			if !errors.Is(err, auditoria.ErrDenegada) || len(pagina.Registros) != 0 || len(e.registro.ordenes) != 0 || len(*e.eventos) != 0 {
				t.Fatalf("entrada libre propagada: %v", err)
			}
		})
	}
}
func TestConstructorIntentosConsultaFallaCerrado(t *testing.T) {
	e := nuevoEscenarioConsulta(t, "ct")
	for _, caso := range []string{"sin-registrador", "proceso", "canal", "recurso", "plazo", "motivo"} {
		t.Run(caso, func(t *testing.T) {
			cfg := e.cfg
			var r ports.RegistradorIntentosAuditoria = e.registro
			switch caso {
			case "sin-registrador":
				r = nil
			case "proceso":
				cfg.Proceso = "proceso libre"
			case "canal":
				cfg.Canal = "externa_personal"
			case "recurso":
				cfg.RecursoCTRef = "persona@example.invalid"
			case "plazo":
				cfg.Plazo = 31 * time.Second
			case "motivo":
				cfg.Motivo = domain.ReferenciaEntradaCatalogo{}
			}
			fuente := fuenteNoUsada{}
			s, err := auditoria.NuevoServicioConIntentos(e.emisor, fuente, fuente, r, cfg)
			if s != nil || !errors.Is(err, auditoria.ErrNoDisponible) {
				t.Fatal("constructor abrió dependencias incompletas")
			}
		})
	}
}

type fuenteNoUsada struct{}

func (fuenteNoUsada) ConsultarAuditoria(context.Context, auditoria.ConsultaAutorizada) (auditoria.PaginaFuente, error) {
	return auditoria.PaginaFuente{}, errors.New("no debe llamarse")
}
