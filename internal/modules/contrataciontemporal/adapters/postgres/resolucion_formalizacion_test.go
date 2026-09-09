package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"os"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func solicitudResolucionPrueba() ports.SolicitudResolucionFormalizacion {
	return ports.SolicitudResolucionFormalizacion{ExpedienteRef: "expediente:prueba", VersionEsperada: 7, PropuestaRef: "propuesta:prueba",
		ClaveIdempotencia: "31111111-1111-4111-8111-111111111111", NumeroResolucion: "EJ-2026/1", FechaResolucion: "2026-09-06",
		Motivo: "Revisión manual del ejercicio.", ConfirmaRevisionPropuesta: true, ConfirmaEjercicioManual: true}
}
func reciboResolucionPrueba(s ports.SolicitudResolucionFormalizacion) ports.ResultadoResolucionFormalizacion {
	return ports.ResultadoResolucionFormalizacion{Solicitud: s, Estado: "registrada", ResolucionRef: "resolucion:prueba",
		DocumentoRef: ports.ReferenciaDocumentoResolucion(s.PropuestaRef), DocumentoSHA256: strings.Repeat("a", 64), DocumentoVersion: 7,
		ActuacionRef: "resolucion:prueba", AuditoriaRef: "auditoria:prueba", OutboxRef: "evento:prueba", ReciboRef: "recibo:prueba",
		VersionResultante: 8, RegistradaEn: time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)}
}

func atestacionResolucionPrueba(t *testing.T, m ports.MaterialResolucionFormalizacion, accion string) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	recurso, err := RecursoResolucionFormalizacion(m)
	if err != nil {
		t.Fatal(err)
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	f := strings.Repeat("a", 64)
	ahora := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:propuesta:unidad", f, f, "contexto:unidad", f,
		accion, m.Solicitud.ExpedienteRef, h, AudienciaRegistroComunicacionLlamamiento, ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, 32)).Public())
	if err != nil {
		t.Fatal(err)
	}
	a, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen, []byte("{}"), []byte("{}"), []byte("{}"),
		1, 1, []byte("unidad"), []byte("unidad"), []byte("unidad"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

type proveedorResolucionPGPrueba struct {
	material  ports.MaterialResolucionFormalizacion
	autorizar func(context.Context, ports.MaterialResolucionFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func (p *proveedorResolucionPGPrueba) PrepararResolucionFormalizacion(context.Context, ports.SolicitudResolucionFormalizacion) (ports.MaterialResolucionFormalizacion, error) {
	return p.material, nil
}
func (p *proveedorResolucionPGPrueba) AutorizarResolucionFormalizacion(c context.Context, m ports.MaterialResolucionFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return p.autorizar(c, m)
}
func materialResolucionPGPrueba() ports.MaterialResolucionFormalizacion {
	s := solicitudResolucionPrueba()
	return ports.MaterialResolucionFormalizacion{Solicitud: s, OrganizacionRef: "organizacion:prueba",
		PropuestaConfirmadaEn: time.Date(2026, 9, 6, 11, 0, 0, 0, time.UTC), DocumentoRef: ports.ReferenciaDocumentoResolucion(s.PropuestaRef),
		DocumentoVersion: 7, DocumentoSHA256: strings.Repeat("a", 64)}
}
func TestResolucionFormalizacionPGTransaccionYReplay(t *testing.T) {
	m := materialResolucionPGPrueba()
	s := m.Solicitud
	tx := &transaccionEjecucionSeleccionO6Prueba{}
	pool := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
	permisos := 0
	p := &proveedorResolucionPGPrueba{material: m, autorizar: func(_ context.Context, m ports.MaterialResolucionFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		permisos++
		return atestacionResolucionPrueba(t, m, AccionResolucionFormalizacion), nil
	}}
	repo := &RegistroResolucionFormalizacionPostgreSQL{pool: pool, proveedor: p}
	original := reciboResolucionPrueba(s)
	for _, estado := range []string{"registrada", "replay_registrada"} {
		esperado := original
		esperado.Estado = estado
		j, _ := json.Marshal(esperado)
		tx.fila = filaEjecucionSeleccionO6Prueba{valores: []any{string(j)}}
		r, e := repo.RegistrarResolucionFormalizacion(context.Background(), s)
		if e != nil || r != esperado {
			t.Fatal(r, e)
		}
	}
	if permisos != 2 || tx.confirmaciones != 2 || pool.inicios != 2 || pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
		t.Fatal("frontera no atómica")
	}
	for i, args := range tx.argumentos {
		b, _ := json.Marshal(m)
		if len(args) != 11 || args[0] != string(b) || !strings.Contains(tx.consultas[i], "registrar_resolucion_formalizacion_v1") {
			t.Fatal("contrato SQL divergente")
		}
	}
}
func TestResolucionFormalizacionPGDenegacionDivergenciaYCommitIncierto(t *testing.T) {
	for _, caso := range []string{"permiso_otro", "documento_otro", "sin_autoridad", "resultado_ajeno", "resultado_temporal", "commit", "sql_conflicto", "sql_permiso", "cancelado"} {
		t.Run(caso, func(t *testing.T) {
			m := materialResolucionPGPrueba()
			s := m.Solicitud
			r := reciboResolucionPrueba(s)
			if caso == "resultado_ajeno" {
				r.DocumentoSHA256 = strings.Repeat("b", 64)
			}
			if caso == "resultado_temporal" {
				r.RegistradaEn = m.PropuestaConfirmadaEn.Add(-time.Second)
			}
			j, _ := json.Marshal(r)
			tx := &transaccionEjecucionSeleccionO6Prueba{fila: filaEjecucionSeleccionO6Prueba{valores: []any{string(j)}}}
			pool := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
			p := &proveedorResolucionPGPrueba{material: m, autorizar: func(_ context.Context, m ports.MaterialResolucionFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				if caso == "sin_autoridad" {
					return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
				}
				accion := AccionResolucionFormalizacion
				if caso == "permiso_otro" {
					accion = AccionPropuestaFormalizacion
				}
				if caso == "documento_otro" {
					m.DocumentoSHA256 = strings.Repeat("b", 64)
				}
				return atestacionResolucionPrueba(t, m, accion), nil
			}}
			if caso == "commit" {
				tx.errCommit = errors.New("detalle privado")
			}
			if caso == "sql_conflicto" {
				tx.fila = filaEjecucionSeleccionO6Prueba{err: &pgconn.PgError{Code: "P0691", Message: "detalle privado"}}
			}
			if caso == "sql_permiso" {
				tx.fila = filaEjecucionSeleccionO6Prueba{err: &pgconn.PgError{Code: "P0693", Message: "detalle privado"}}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if caso == "cancelado" {
				cancel()
			}
			repo := &RegistroResolucionFormalizacionPostgreSQL{pool: pool, proveedor: p}
			out, e := repo.RegistrarResolucionFormalizacion(ctx, s)
			if e == nil || out != (ports.ResultadoResolucionFormalizacion{}) || strings.Contains(e.Error(), "privado") || pool.inicios > 1 {
				t.Fatal("éxito falso o reintento", e)
			}
			if caso == "sql_conflicto" && !errors.Is(e, ports.ErrClaveResolucionFormalizacionUsada) {
				t.Fatal(e)
			}
			if caso != "commit" && tx.confirmaciones != 0 {
				t.Fatal("commit sin resultado válido")
			}
		})
	}
}
func TestResolucionFormalizacionPreparacionPGContratoCerrado(t *testing.T) {
	s := solicitudResolucionPrueba()
	r := reciboResolucionPrueba(s)
	p := ports.PreparacionResolucionFormalizacion{ExpedienteRef: s.ExpedienteRef, PropuestaRef: s.PropuestaRef, VersionEsperada: 7, VersionActual: 7}
	for _, version := range []uint64{7, 8} {
		p.VersionActual = version
		if version == 8 {
			p.Recibo = &r
		}
		b, _ := json.Marshal(p)
		obtenida, e := decodificarPreparacionResolucion(string(b), s.ExpedienteRef, version)
		if e != nil || obtenida.ValidarPara(s.ExpedienteRef) != nil {
			t.Fatal(obtenida, e)
		}
		for _, raw := range []string{string(b) + "{}", strings.Replace(string(b), `"VersionActual":`, `"actor":"browser","VersionActual":`, 1),
			"null", strings.Repeat(" ", 16385)} {
			if _, e = decodificarPreparacionResolucion(raw, s.ExpedienteRef, version); e == nil {
				t.Fatal("JSON abierto")
			}
		}
		if _, e = decodificarPreparacionResolucion(string(b), "expediente:ajeno", version); e == nil {
			t.Fatal("expediente ajeno")
		}
		if _, e = decodificarPreparacionResolucion(string(b), s.ExpedienteRef, 9); e == nil {
			t.Fatal("versión ajena")
		}
	}
}

func TestResolucionFormalizacionPreparacionSQLLecturaSinEfecto(t *testing.T) {
	b, e := os.ReadFile("../../../../../deploy/postgresql/contratacion_temporal/migraciones/000069_resolucion_formalizacion_manual_ejercicio.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	up := string(b)
	i := strings.Index(up, "CREATE FUNCTION vec_contratacion_temporal.consultar_preparacion_resolucion_v1(")
	if i < 0 {
		t.Fatal("sin preparación")
	}
	lectura := up[i:]
	permiso := strings.Index(lectura, "SELECT * INTO STRICT v_lectura FROM vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(")
	negocio := strings.Index(lectura, "SELECT v.* INTO STRICT v_actual")
	if permiso < 0 || negocio < permiso {
		t.Fatal("lectura previa a autorización")
	}
	for _, prohibido := range []string{"INSERT INTO", "UPDATE ", "DELETE FROM", "registrar_resolucion_formalizacion_v1", "registrar_y_consumir_resolucion_formalizacion"} {
		if strings.Contains(lectura, prohibido) {
			t.Fatal("lectura con efecto", prohibido)
		}
	}
	for _, guardia := range []string{"pg_advisory_xact_lock_shared", "version_observada IS DISTINCT FROM 0", "v_propuesta.propuesta_ref",
		"v_resolucion.recibo_json", "TO vec_contratacion_temporal_consultor_rrhh", "SET row_security=on", "SECURITY DEFINER"} {
		if !strings.Contains(lectura, guardia) {
			t.Fatal("falta guardia", guardia)
		}
	}
	b, e = os.ReadFile("../../../../../deploy/postgresql/contratacion_temporal/migraciones/000069_resolucion_formalizacion_manual_ejercicio.down.sql")
	if e != nil || !strings.Contains(string(b), "DROP FUNCTION vec_contratacion_temporal.consultar_preparacion_resolucion_v1(") {
		t.Fatal("retirada sin dependencia", e)
	}
}

func TestResolucionFormalizacionMigracionesContratoYRetirada(t *testing.T) {
	base := "../../../../../deploy/postgresql/"
	leer := func(path string) string {
		b, e := os.ReadFile(base + path)
		if e != nil {
			t.Fatal(e)
		}
		return string(b)
	}
	adUp := leer("autorizacion_atestada_v3/migraciones/000025_resolucion_formalizacion_manual_ejercicio.up.sql")
	adDown := leer("autorizacion_atestada_v3/migraciones/000025_resolucion_formalizacion_manual_ejercicio.down.sql")
	ctUp := leer("contratacion_temporal/migraciones/000069_resolucion_formalizacion_manual_ejercicio.up.sql")
	ctDown := leer("contratacion_temporal/migraciones/000069_resolucion_formalizacion_manual_ejercicio.down.sql")
	extraer := func(s, tag string) string {
		i := strings.Index(s, tag)
		if i < 0 {
			t.Fatal(tag)
		}
		s = s[i+len(tag):]
		i = strings.Index(s, tag)
		if i < 0 {
			t.Fatal(tag)
		}
		return s[:i]
	}
	if extraer(adUp, "$perfil$") != extraer(adDown, "$perfil$") || extraer(ctUp, "$extension$") != extraer(ctDown, "$extension$") {
		t.Fatal("DOWN no deshace el bloque exacto")
	}
	for _, s := range []string{adUp, adDown, ctUp, ctDown} {
		if !strings.Contains(s, "vec:resolucion-formalizacion:dependencia:000025-000069") {
			t.Fatal("sin lock de dependencia")
		}
	}
	for _, s := range []string{adUp, ctUp} {
		if !strings.Contains(s, AccionResolucionFormalizacion) || !strings.Contains(s, TipoRecursoResolucionFormalizacion) {
			t.Fatal("autoridad divergente")
		}
	}
	if !strings.Contains(adDown, "atestacion_decision_v3") || !strings.Contains(ctDown, "historia de resolución conservada") {
		t.Fatal("retirada no protege historia")
	}
	consumir := strings.Index(ctUp, "SELECT * INTO STRICT v_consumo")
	negocio := strings.Index(ctUp, "SELECT * INTO v_propuesta FROM")
	replay := strings.Index(ctUp, "RETURN v_previa.recibo_json")
	version := strings.Index(ctUp, "SELECT v.* INTO v_actual")
	if consumir < 0 || negocio < consumir || replay < negocio || version < replay {
		t.Fatal("orden de autorización/replay/OCC incorrecto")
	}
	for _, tabla := range []string{"expediente_version_integral", "actuacion_expediente_integral", "outbox_expediente_integral", "resolucion_formalizacion"} {
		if !strings.Contains(ctUp, "INSERT INTO vec_contratacion_temporal."+tabla) {
			t.Fatal("falta escritura durable", tabla)
		}
	}
	if strings.Contains(ctUp, "UPDATE vec_contratacion_temporal.propuesta_formalizacion") || strings.Contains(ctUp, "DELETE FROM") {
		t.Fatal("mutación de antecedentes")
	}
}
