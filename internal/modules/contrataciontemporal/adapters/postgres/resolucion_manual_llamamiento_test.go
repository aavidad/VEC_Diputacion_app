package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type proveedorResolucionManualPrueba struct {
	proveedorComunicacionLocalPrueba
	preparar  func(context.Context, ports.SolicitudResolverLlamamiento) (MaterialResolucionManualLlamamiento, error)
	autorizar func(context.Context, MaterialResolucionManualLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func (p *proveedorResolucionManualPrueba) PrepararResolucionManual(ctx context.Context, s ports.SolicitudResolverLlamamiento) (MaterialResolucionManualLlamamiento, error) {
	return p.preparar(ctx, s)
}

func (p *proveedorResolucionManualPrueba) AutorizarResolucionManual(ctx context.Context, m MaterialResolucionManualLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return p.autorizar(ctx, m)
}

func resolucionManualPGPrueba(t *testing.T) (MaterialResolucionManualLlamamiento, ports.ResultadoResolucionLlamamiento) {
	t.Helper()
	s, _ := justificanteConsultaPGPrueba(t)
	s.RevisionRespuestaRRHH, s.RevisionPlazoRRHH = true, true
	s.CriterioValidacionRef = "criterio:revision-manual-sintetica"
	m := MaterialResolucionManualLlamamiento{Solicitud: s, Politica: ports.ReferenciaGobernadaComunicacionLlamamiento{
		Referencia: s.CriterioValidacionRef, Version: 1, HuellaSHA256: strings.Repeat("b", 64),
	}}
	r := ports.ResultadoResolucionLlamamiento{
		Solicitud: s, Politica: m.Politica, EvaluacionPlazoRef: "evaluacion:revision-manual-sintetica",
		EstadoPlazo: ports.PlazoLlamamientoVigente, ResolucionRef: "resolucion:sintetica",
		ReciboLocalRef: "recibo:resolucion-sintetica", AuditoriaRef: "auditoria:resolucion-sintetica", VersionResultante: 3,
		ResueltaEn: time.Date(2026, 9, 6, 10, 0, 0, 123456000, time.UTC), Estado: ports.ResultadoComunicacionLlamamientoConfirmado,
	}
	if m.Validar() != nil || r.ValidarPara(s) != nil {
		t.Fatal("fixture manual incoherente")
	}
	return m, r
}

// Transporte no criptográfico exclusivo de dobles pgx; nunca válido ante SQL.
func autorizacionResolucionManualPrueba(t *testing.T, m MaterialResolucionManualLlamamiento, accion, audiencia string, numero int) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	recurso, err := RecursoResolucionManualLlamamiento(m)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	ahora := time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)
	r, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:manual-unidad:"+strconv.Itoa(numero), h, h, "contexto:unidad", h,
		accion, m.Solicitud.ExpedienteRef, huella, audiencia, ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public())
	if err != nil {
		t.Fatal(err)
	}
	a, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), r,
		[]byte("{}"), []byte("{}"), []byte("{}"), 1, 1, []byte("unidad"), []byte("unidad"), []byte("unidad"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestResolucionManualLlamamientoMaterialCompleto(t *testing.T) {
	m, _ := resolucionManualPGPrueba(t)
	recurso, err := RecursoResolucionManualLlamamiento(m)
	b, _ := json.Marshal(m)
	h := sha256.Sum256(b)
	var material map[string]map[string]json.RawMessage
	if err != nil || json.Unmarshal(b, &material) != nil || len(material) != 2 || len(material["Solicitud"]) != 11 || len(material["Politica"]) != 3 ||
		recurso.Referencia != m.Solicitud.ExpedienteRef || recurso.ModuloID != "contratacion_temporal" || recurso.Tipo != TipoRecursoResolucionManualLlamamiento ||
		len(recurso.Ambitos) != 1 || recurso.Ambitos["organizacion_ref"] != m.Solicitud.OrganizacionRef ||
		len(recurso.Atributos) != 1 || recurso.Atributos["material_sha256"] != hex.EncodeToString(h[:]) {
		t.Fatal("material incompleto o contrato JSON divergente", err)
	}
	for nombre, mutar := range map[string]func(*MaterialResolucionManualLlamamiento){
		"revisiones": func(m *MaterialResolucionManualLlamamiento) { m.Solicitud.RevisionPlazoRRHH = false },
		"criterio":   func(m *MaterialResolucionManualLlamamiento) { m.Solicitud.CriterioValidacionRef += "otro" },
		"politica":   func(m *MaterialResolucionManualLlamamiento) { m.Politica.Version = 0 },
	} {
		t.Run(nombre, func(t *testing.T) {
			otro := m
			mutar(&otro)
			if otro.Validar() == nil {
				t.Fatal("material incompleto aceptado")
			}
		})
	}
}

func TestResolucionManualLlamamientoConfirmaYReautorizaReplay(t *testing.T) {
	m, original := resolucionManualPGPrueba(t)
	tx := &transaccionEjecucionSeleccionO6Prueba{}
	pool := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
	preparaciones, permisos := 0, 0
	p := &proveedorResolucionManualPrueba{
		preparar: func(_ context.Context, s ports.SolicitudResolverLlamamiento) (MaterialResolucionManualLlamamiento, error) {
			preparaciones++
			if s != m.Solicitud {
				t.Fatal("solicitud divergente")
			}
			return m, nil
		},
		autorizar: func(_ context.Context, recibido MaterialResolucionManualLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
			permisos++
			if recibido != m {
				t.Fatal("material divergente")
			}
			return autorizacionResolucionManualPrueba(t, recibido, AccionResolucionManualLlamamiento, AudienciaRegistroComunicacionLlamamiento, permisos), nil
		},
	}
	adaptador := &TransaccionComunicacionLlamamientoPostgreSQL{pool: pool, proveedor: p}
	for _, estado := range []ports.EstadoResultadoComunicacionLlamamiento{ports.ResultadoComunicacionLlamamientoConfirmado, ports.ResultadoComunicacionLlamamientoReplay} {
		r := original
		r.Estado = estado
		b, _ := json.Marshal(r)
		tx.fila = filaEjecucionSeleccionO6Prueba{valores: []any{strings.ReplaceAll(string(b), "Z\"", "+00:00\"")}}
		obtenido, err := adaptador.ResolverLlamamiento(context.Background(), m.Solicitud)
		if err != nil || obtenido != r {
			t.Fatal("recibo divergente", err)
		}
	}
	if preparaciones != 2 || permisos != 2 || p.autorizaciones != 0 || pool.inicios != 2 || tx.confirmaciones != 2 ||
		pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
		t.Fatal("replay sin permiso fresco propio o sin commit")
	}
	b, _ := json.Marshal(m)
	for i, sql := range tx.consultas {
		if !strings.Contains(sql, "vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(") ||
			len(tx.argumentos[i]) != 11 || tx.argumentos[i][0] != string(b) || tx.argumentos[i][5] != int64(1) || tx.argumentos[i][6] != int64(1) {
			t.Fatal("SQL o material divergente")
		}
	}
}

func TestResolucionManualLlamamientoFallaCerradoSinReintento(t *testing.T) {
	m, original := resolucionManualPGPrueba(t)
	for _, caso := range []string{"sin_proveedor", "proveedor_tipado_nulo", "legacy", "preparacion", "permiso_aviso", "otra_audiencia", "otro_material", "recibo", "politica", "submicro", "sql", "commit"} {
		t.Run(caso, func(t *testing.T) {
			b, _ := json.Marshal(original)
			tx := &transaccionEjecucionSeleccionO6Prueba{fila: filaEjecucionSeleccionO6Prueba{valores: []any{string(b)}}}
			pool := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
			p := &proveedorResolucionManualPrueba{
				preparar: func(_ context.Context, _ ports.SolicitudResolverLlamamiento) (MaterialResolucionManualLlamamiento, error) {
					otro := m
					if caso == "preparacion" {
						otro.Solicitud.ExpedienteRef += "otro"
					}
					return otro, nil
				},
				autorizar: func(_ context.Context, material MaterialResolucionManualLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
					accion, audiencia := AccionResolucionManualLlamamiento, AudienciaRegistroComunicacionLlamamiento
					if caso == "permiso_aviso" {
						accion = AccionRegistroComunicacionLlamamiento
					}
					if caso == "otra_audiencia" {
						audiencia = "vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1"
					}
					if caso == "otro_material" {
						material.Politica.Version++
					}
					return autorizacionResolucionManualPrueba(t, material, accion, audiencia, 1), nil
				},
			}
			s := m.Solicitud
			a := &TransaccionComunicacionLlamamientoPostgreSQL{pool: pool, proveedor: p}
			esperado := ports.ErrOperacionComunicacionLlamamientoDenegada
			switch caso {
			case "sin_proveedor":
				a.proveedor = &proveedorComunicacionLocalPrueba{}
			case "proveedor_tipado_nulo":
				a.proveedor = (*proveedorResolucionManualPrueba)(nil)
			case "legacy":
				s = s.ParaConsultaJustificante()
			case "preparacion":
				esperado = ports.ErrResultadoComunicacionLlamamientoNoConfiable
			case "recibo", "politica", "submicro":
				r := original
				if caso == "recibo" {
					r.Solicitud.RevisionPlazoRRHH = false
				}
				if caso == "politica" {
					r.Politica.Version++
				}
				if caso == "submicro" {
					r.ResueltaEn = r.ResueltaEn.Add(time.Nanosecond)
				}
				b, _ := json.Marshal(r)
				tx.fila = filaEjecucionSeleccionO6Prueba{valores: []any{string(b)}}
				esperado = ports.ErrResultadoComunicacionLlamamientoNoConfiable
			case "sql":
				tx.fila = filaEjecucionSeleccionO6Prueba{err: &pgconn.PgError{Code: "40001", Message: "dato privado"}}
				esperado = ErrPersistenciaComunicacionLlamamientoNoDisponible
			case "commit":
				tx.errCommit = errors.New("dato privado")
				esperado = ErrPersistenciaComunicacionLlamamientoNoDisponible
			}
			r, err := a.ResolverLlamamiento(context.Background(), s)
			if !errors.Is(err, esperado) || r != (ports.ResultadoResolucionLlamamiento{}) || strings.Contains(err.Error(), "privado") || pool.inicios > 1 {
				t.Fatal("fallo abierto, filtrado o reintentado", err)
			}
			if esperado == ports.ErrOperacionComunicacionLlamamientoDenegada && pool.inicios != 0 {
				t.Fatal("permiso incorrecto alcanzó SQL")
			}
			if caso != "commit" && tx.confirmaciones != 0 {
				t.Fatal("confirmación de resultado inválido")
			}
			if pool.inicios == 1 && tx.reversiones != 1 {
				t.Fatal("transacción sin cerrar")
			}
		})
	}
}

func TestResolucionManualLlamamientoErroresSQLSinDatos(t *testing.T) {
	for codigo, esperado := range map[string]error{
		"P0580": ports.ErrSolicitudComunicacionLlamamientoInvalida,
		"P0581": ports.ErrClaveComunicacionLlamamientoUsada,
		"P0582": ports.ErrVersionComunicacionLlamamientoEnConflicto,
		"P0583": ports.ErrOperacionComunicacionLlamamientoDenegada,
		"P0584": ErrPersistenciaComunicacionLlamamientoNoDisponible,
		"42501": ports.ErrOperacionComunicacionLlamamientoDenegada,
		"08006": ErrPersistenciaComunicacionLlamamientoNoDisponible,
	} {
		if err := normalizarErrorResolucionManualLlamamiento(context.Background(), &pgconn.PgError{Code: codigo, Message: "dato privado"}); err != esperado {
			t.Fatal(codigo, err)
		}
	}
}
