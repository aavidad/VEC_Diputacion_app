package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type proveedorPropuestaPGPrueba struct {
	material  ports.MaterialPropuestaFormalizacion
	autorizar func(context.Context, ports.MaterialPropuestaFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func (p *proveedorPropuestaPGPrueba) PrepararPropuestaFormalizacion(context.Context, ports.SolicitudPropuestaFormalizacion) (ports.MaterialPropuestaFormalizacion, error) {
	return p.material.Clonar(), nil
}
func (p *proveedorPropuestaPGPrueba) AutorizarPropuestaFormalizacion(ctx context.Context, m ports.MaterialPropuestaFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return p.autorizar(ctx, m)
}

func datosPropuestaPGPrueba(t *testing.T) (ports.SolicitudPropuestaFormalizacion, ports.AntecedentePropuestaFormalizacion, ports.MaterialPropuestaFormalizacion, ports.ResultadoPropuestaFormalizacion) {
	t.Helper()
	_, r := resolucionManualPGPrueba(t)
	_, j := justificanteConsultaPGPrueba(t)
	if r.ResueltaEn.Before(j.Respuesta.RegistradaEn) {
		r.ResueltaEn = j.Respuesta.RegistradaEn.Add(time.Minute)
	}
	snapshot := ports.SnapshotGobernadoFormalizacion{Referencia: "publicacion:propuesta", Version: 1, HuellaSHA256: strings.Repeat("a", 64)}
	s := ports.SolicitudPropuestaFormalizacion{ClaveIdempotencia: "31111111-1111-4111-8111-111111111111",
		OrganizacionRef: r.Solicitud.OrganizacionRef, ExpedienteRef: r.Solicitud.ExpedienteRef, LlamamientoRef: r.Solicitud.LlamamientoRef,
		ResolucionLlamamientoAceptadaRef: r.ResolucionRef, ReciboResolucionAceptadaRef: r.ReciboLocalRef, VersionEsperada: 6,
		TipoFormalizacion: snapshot, Plantilla: snapshot, PoliticaFirma: snapshot, PlanFirma: snapshot}
	a := ports.AntecedentePropuestaFormalizacion{Resolucion: r, Justificante: j, SeleccionClave: "21111111-1111-4111-8111-111111111111"}
	e := ports.EvidenciaAceptacionBolsaPropuesta{OperacionRef: "operacion:aceptada", AperturaOperacionRef: j.Seleccion.OperacionRef,
		LlamamientoRef: s.LlamamientoRef, JustificanteRef: r.Solicitud.PruebaRespuestaRef, EvaluacionPlazoRef: r.EvaluacionPlazoRef,
		Politica:       ports.SnapshotGobernadoFormalizacion{Referencia: r.Politica.Referencia, Version: r.Politica.Version, HuellaSHA256: r.Politica.HuellaSHA256},
		RegistroSHA256: strings.Repeat("b", 64), ResueltaEn: r.ResueltaEn.Add(time.Second)}
	m := ports.MaterialPropuestaFormalizacion{Etapa: "confirmacion", Solicitud: s, AceptacionBolsa: &e}
	salida := ports.ResultadoPropuestaFormalizacion{Solicitud: s, PropuestaRef: "propuesta:local", ReciboLocalRef: "recibo:propuesta",
		AuditoriaRef: "auditoria:propuesta", VersionResultante: 7, ConfirmadaEn: e.ResueltaEn.Add(time.Second), Estado: ports.ResultadoPropuestaFormalizacionConfirmado}
	if a.ValidarPara(s) != nil || e.ValidarPara(a) != nil || m.Validar() != nil || salida.ValidarPara(s) != nil {
		t.Fatal("fixture propuesta inválido")
	}
	return s, a, m, salida
}

// Transporte exclusivamente de prueba para dobles pgx. No prueba criptografía,
// persistencia real ni publicaciones instaladas; nunca se envía a PostgreSQL.
func materialPropuestaPGPrueba(t *testing.T, m ports.MaterialPropuestaFormalizacion, accion string) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	recurso, err := RecursoPropuestaFormalizacion(m)
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

func TestPropuestaFormalizacionPGConsultaConfirmacionYReplay(t *testing.T) {
	s, a, m, original := datosPropuestaPGPrueba(t)
	tx := &transaccionEjecucionSeleccionO6Prueba{}
	pool := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
	permisos := 0
	p := &proveedorPropuestaPGPrueba{material: m, autorizar: func(_ context.Context, m ports.MaterialPropuestaFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		permisos++
		return materialPropuestaPGPrueba(t, m, AccionPropuestaFormalizacion), nil
	}}
	repo := &RegistroPropuestaFormalizacionPostgreSQL{pool: pool, proveedor: p}
	j, _ := json.Marshal(a)
	tx.fila = filaEjecucionSeleccionO6Prueba{valores: []any{string(j)}}
	leido, err := repo.LeerAntecedente(context.Background(), s)
	if err != nil || !reflect.DeepEqual(leido, a) {
		t.Fatal("consulta", err)
	}
	for _, estado := range []ports.EstadoResultadoPropuestaFormalizacion{ports.ResultadoPropuestaFormalizacionConfirmado, ports.ResultadoPropuestaFormalizacionReplay} {
		esperado := original
		esperado.Estado = estado
		j, _ = json.Marshal(esperado)
		tx.fila = filaEjecucionSeleccionO6Prueba{valores: []any{strings.ReplaceAll(string(j), "Z\"", "+00:00\"")}}
		obtenido, err := repo.ConfirmarPropuesta(context.Background(), s)
		if err != nil || !reflect.DeepEqual(obtenido, esperado) {
			t.Fatal("confirmación/replay", err)
		}
	}
	if permisos != 3 || tx.confirmaciones != 3 || pool.inicios != 3 || pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
		t.Fatal("frontera no fresca/atómica")
	}
	for i, args := range tx.argumentos {
		material := m
		if i == 0 {
			material = ports.MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}
		}
		canon, _ := json.Marshal(material)
		h := sha256.Sum256(canon)
		recurso, _ := RecursoPropuestaFormalizacion(material)
		if len(args) != 11 || args[0] != string(canon) || recurso.Atributos["material_sha256"] != hex.EncodeToString(h[:]) ||
			!strings.Contains(tx.consultas[i], "registrar_propuesta_formalizacion_v1(") {
			t.Fatal("bytes/función divergentes")
		}
	}
}

func TestPropuestaFormalizacionPGSucesorNormalizaUTCNoTrunca(t *testing.T) {
	s, a, _, _ := datosPropuestaPGPrueba(t)
	_, _, c := datosContinuacionPG(t)
	c.Solicitud.OrganizacionRef, c.Solicitud.ExpedienteRef = s.OrganizacionRef, s.ExpedienteRef
	c.LlamamientoAnteriorRef = a.Justificante.Seleccion.LlamamientoRef
	c.ReciboBolsa.ConfirmadaEn = a.Justificante.Seleccion.ConfirmadaEn.Add(time.Second)
	c.ConfirmadaEn = c.ReciboBolsa.ConfirmadaEn.Add(time.Microsecond)
	a.Justificante.Continuacion = &c
	s.LlamamientoRef, a.Resolucion.Solicitud.LlamamientoRef = c.ReciboBolsa.LlamamientoRef, c.ReciboBolsa.LlamamientoRef
	a.Justificante.Respuesta.Solicitud.LlamamientoRef = s.LlamamientoRef
	if err := a.ValidarPara(s); err != nil {
		t.Fatal("fixture sucesor", err)
	}
	tx := &transaccionEjecucionSeleccionO6Prueba{}
	permisos := 0
	p := &proveedorPropuestaPGPrueba{autorizar: func(_ context.Context, m ports.MaterialPropuestaFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		permisos++
		return materialPropuestaPGPrueba(t, m, AccionPropuestaFormalizacion), nil
	}}
	repo := &RegistroPropuestaFormalizacionPostgreSQL{pool: &iniciadorEjecucionSeleccionO6Prueba{tx: tx}, proveedor: p}
	canon, _ := json.Marshal(a)
	// Solo las dos fechas añadidas. No convertir ceros de la estructura Go:
	// la aceptación SQL conserva IntencionSiguiente={}, no una fecha cero local.
	salida := string(canon)
	for _, fecha := range []time.Time{c.ConfirmadaEn, c.ReciboBolsa.ConfirmadaEn} {
		utc := fecha.Format(time.RFC3339Nano)
		salida = strings.ReplaceAll(salida, utc, strings.TrimSuffix(utc, "Z")+"+00:00")
	}
	tx.fila = filaEjecucionSeleccionO6Prueba{valores: []any{salida}}
	for i := 1; i <= 2; i++ {
		leido, err := repo.LeerAntecedente(context.Background(), s)
		if err != nil || !reflect.DeepEqual(leido, a) || permisos != i || tx.confirmaciones != i {
			t.Fatal("continuación no preservada con permiso nuevo", err)
		}
	}
	c.ConfirmadaEn = c.ConfirmadaEn.Add(time.Nanosecond)
	canon, _ = json.Marshal(a)
	tx.fila = filaEjecucionSeleccionO6Prueba{valores: []any{string(canon)}}
	if _, err := repo.LeerAntecedente(context.Background(), s); err == nil || tx.confirmaciones != 2 {
		t.Fatal("se truncó fecha no canónica")
	}
}

func TestPropuestaFormalizacionPGFalloNoConfirmaNiFiltra(t *testing.T) {
	s, _, m, original := datosPropuestaPGPrueba(t)
	for _, caso := range []string{"permiso", "etapa", "proveedor", "recibo", "sql", "commit", "cancelacion"} {
		t.Run(caso, func(t *testing.T) {
			esperado := original.Clonar()
			if caso == "recibo" {
				esperado.Solicitud.ExpedienteRef = "expediente:otro"
			}
			j, _ := json.Marshal(esperado)
			tx := &transaccionEjecucionSeleccionO6Prueba{fila: filaEjecucionSeleccionO6Prueba{valores: []any{string(j)}}}
			pool := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
			permisos := 0
			p := &proveedorPropuestaPGPrueba{material: m.Clonar(), autorizar: func(_ context.Context, entrada ports.MaterialPropuestaFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				permisos++
				accion := AccionPropuestaFormalizacion
				if caso == "permiso" {
					accion = AccionResolucionManualLlamamiento
				}
				if caso == "etapa" {
					entrada.Etapa = "consulta"
					entrada.AceptacionBolsa = nil
				}
				return materialPropuestaPGPrueba(t, entrada, accion), nil
			}}
			if caso == "proveedor" {
				p.material.Solicitud.ExpedienteRef = "expediente:otro"
			}
			if caso == "sql" {
				tx.fila = filaEjecucionSeleccionO6Prueba{err: &pgconn.PgError{Code: "P0613", Message: "detalle privado"}}
			}
			if caso == "commit" {
				tx.errCommit = errors.New("detalle privado")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if caso == "cancelacion" {
				cancel()
			}
			repo := &RegistroPropuestaFormalizacionPostgreSQL{pool: pool, proveedor: p}
			obtenido, err := repo.ConfirmarPropuesta(ctx, s)
			if err == nil || !obtenido.EsCero() || strings.Contains(err.Error(), "privado") || permisos > 1 || pool.inicios > 1 {
				t.Fatal("éxito, fuga o reintento")
			}
			if caso != "commit" && tx.confirmaciones != 0 {
				t.Fatal("commit con fallo")
			}
		})
	}
	for code, esperado := range map[string]error{"P0610": ports.ErrSolicitudPropuestaFormalizacionInvalida,
		"P0611": ports.ErrClavePropuestaFormalizacionUsada, "P0612": ports.ErrVersionPropuestaFormalizacionEnConflicto,
		"P0613": ports.ErrOperacionPropuestaFormalizacionDenegada, "P0614": ErrPersistenciaPropuestaFormalizacionNoDisponible, "P0615": ports.ErrResolucionLlamamientoNoAceptada} {
		if !errors.Is(normalizarErrorPropuestaFormalizacion(context.Background(), &pgconn.PgError{Code: code}), esperado) {
			t.Fatal("normalización", code)
		}
	}
}
