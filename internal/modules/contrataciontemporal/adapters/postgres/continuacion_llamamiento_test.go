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

type proveedorContinuacionPrueba func(context.Context, ports.MaterialContinuacionLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)

func (p proveedorContinuacionPrueba) AutorizarContinuacionLlamamiento(ctx context.Context, m ports.MaterialContinuacionLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return p(ctx, m)
}

func datosContinuacionPG(t *testing.T) (ports.SolicitudContinuarLlamamiento, ports.AntecedenteContinuacionLlamamiento, ports.ResultadoContinuacionLlamamiento) {
	t.Helper()
	_, renuncia := resolucionManualPGPrueba(t)
	renuncia.Solicitud.Respuesta = ports.RespuestaLlamamientoRenunciada
	renuncia.IntencionSiguiente = ports.IntencionOutboxSiguienteCandidato{Solicitud: renuncia.Solicitud,
		ResolucionRef: renuncia.ResolucionRef, LlamamientoRef: renuncia.Solicitud.LlamamientoRef,
		ClaveIdempotencia: renuncia.Solicitud.ClaveIdempotencia, VersionEsperada: 2, VersionResultante: 3,
		IntencionRef: "intencion:siguiente", ComandoOpacoRef: "comando:siguiente",
		Estado: ports.OutboxSiguienteCandidatoPendiente, ActualizadaEn: renuncia.ResueltaEn}
	s := ports.SolicitudContinuarLlamamiento{ClaveIdempotencia: "31111111-1111-4111-8111-111111111111", OrganizacionRef: renuncia.Solicitud.OrganizacionRef, ExpedienteRef: renuncia.Solicitud.ExpedienteRef, ResolucionRef: renuncia.ResolucionRef, IntencionRef: "intencion:siguiente"}
	a := ports.AntecedenteContinuacionLlamamiento{Resolucion: renuncia, ComandoSiguienteRef: "comando:siguiente",
		ComandoSiguiente: ports.ComandoSiguienteLlamamiento{Esquema: "vec.contratacion-temporal.siguiente-candidato.intencion.v1",
			ComandoRef: "comando:siguiente", IntencionRef: s.IntencionRef, OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
			LlamamientoRef: renuncia.Solicitud.LlamamientoRef, JustificanteRef: renuncia.Solicitud.PruebaRespuestaRef, SeleccionClave: renuncia.Solicitud.ClaveIdempotencia}}
	b := ports.ReciboBolsaContinuacion{IntencionRef: s.IntencionRef, TerminalOperacionRef: "operacion:renuncia", OperacionRef: "operacion:siguiente",
		LlamamientoRef: "llamamiento:siguiente", PropuestaRef: "propuesta:siguiente", ReciboRef: "recibo:bolsa", AuditoriaRef: "auditoria:bolsa", EventoRef: "evento:bolsa",
		RegistroSHA256: strings.Repeat("a", 64), ConfirmadaEn: renuncia.ResueltaEn.Add(time.Minute)}
	r := ports.ResultadoContinuacionLlamamiento{Solicitud: s, LlamamientoAnteriorRef: renuncia.Solicitud.LlamamientoRef,
		ReciboBolsa: b, ReciboRef: "recibo:continuacion", AuditoriaRef: "auditoria:continuacion", ConfirmadaEn: b.ConfirmadaEn.Add(time.Second), Estado: "confirmado"}
	if a.ValidarPara(s) != nil || r.ValidarPara(s) != nil {
		t.Fatal("fixture incoherente")
	}
	return s, a, r
}

// Material de transporte solo para dobles pgx, deliberadamente no válido
// criptográficamente ante PostgreSQL; nunca se instala ni sale al bootstrap.
func materialContinuacionPGPrueba(t *testing.T, m ports.MaterialContinuacionLlamamiento, accion string, n int) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	recurso, err := RecursoContinuacionLlamamiento(m)
	if err != nil {
		t.Fatal(err)
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	f := strings.Repeat("a", 64)
	ahora := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	c, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:continuacion:"+strconv.Itoa(n), f, f, "contexto:unidad", f,
		accion, m.Solicitud.ExpedienteRef, h, AudienciaRegistroComunicacionLlamamiento, ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, 32)).Public())
	if err != nil {
		t.Fatal(err)
	}
	a, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), c, []byte("{}"), []byte("{}"), []byte("{}"),
		1, 1, []byte("unidad"), []byte("unidad"), []byte("unidad"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestContinuacionLlamamientoPGMaterialYRecuperacion(t *testing.T) {
	s, a, original := datosContinuacionPG(t)
	tx := &transaccionEjecucionSeleccionO6Prueba{}
	pool := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
	permisos := 0
	p := proveedorContinuacionPrueba(func(_ context.Context, m ports.MaterialContinuacionLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		permisos++
		return materialContinuacionPGPrueba(t, m, AccionContinuacionLlamamiento, permisos), nil
	})
	repo := &RegistroContinuacionLlamamientoPostgreSQL{pool: pool, proveedor: p}
	j, _ := json.Marshal(a)
	tx.fila = filaEjecucionSeleccionO6Prueba{valores: []any{string(j)}}
	leido, err := repo.LeerAntecedente(context.Background(), s)
	if err != nil || leido != a {
		t.Fatal("lectura", err)
	}
	for _, estado := range []string{"confirmado", "replay_confirmado"} {
		r := original
		r.Estado = estado
		j, _ = json.Marshal(r)
		tx.fila = filaEjecucionSeleccionO6Prueba{valores: []any{strings.ReplaceAll(string(j), "Z\"", "+00:00\"")}}
		obtenido, err := repo.Confirmar(context.Background(), s, original.ReciboBolsa)
		if err != nil || obtenido != r {
			t.Fatal("confirmación/replay", err)
		}
	}
	if permisos != 3 || tx.confirmaciones != 3 || pool.inicios != 3 || pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
		t.Fatal("falta consumo/commit fresco")
	}
	for i, args := range tx.argumentos {
		m := ports.MaterialContinuacionLlamamiento{Etapa: "consulta", Solicitud: s}
		if i > 0 {
			m.Etapa = "confirmacion"
			b := original.ReciboBolsa
			m.ReciboBolsa = &b
		}
		j, _ := json.Marshal(m)
		h := sha256.Sum256(j)
		recurso, _ := RecursoContinuacionLlamamiento(m)
		if len(args) != 11 || args[0] != string(j) || recurso.Atributos["material_sha256"] != hex.EncodeToString(h[:]) ||
			!strings.Contains(tx.consultas[i], "continuar_llamamiento_rrhh_v1(") {
			t.Fatal("material SQL distinto del autorizado")
		}
	}
	consulta, _ := RecursoContinuacionLlamamiento(ports.MaterialContinuacionLlamamiento{Etapa: "consulta", Solicitud: s})
	confirmacion, _ := RecursoContinuacionLlamamiento(ports.MaterialContinuacionLlamamiento{Etapa: "confirmacion", Solicitud: s, ReciboBolsa: &original.ReciboBolsa})
	if consulta.Atributos["material_sha256"] == confirmacion.Atributos["material_sha256"] {
		t.Fatal("etapas no separadas")
	}
}

func TestContinuacionLlamamientoPGFallaSinExitoNiReintento(t *testing.T) {
	s, _, original := datosContinuacionPG(t)
	for _, caso := range []string{"permiso", "etapa", "recibo_cruzado", "sql", "commit", "cancelacion"} {
		t.Run(caso, func(t *testing.T) {
			r := original
			if caso == "recibo_cruzado" {
				r.ReciboBolsa.ReciboRef = "recibo:otro"
			}
			j, _ := json.Marshal(r)
			tx := &transaccionEjecucionSeleccionO6Prueba{fila: filaEjecucionSeleccionO6Prueba{valores: []any{string(j)}}}
			pool := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
			permisos := 0
			p := proveedorContinuacionPrueba(func(_ context.Context, m ports.MaterialContinuacionLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				permisos++
				accion := AccionContinuacionLlamamiento
				if caso == "permiso" {
					accion = AccionResolucionManualLlamamiento
				}
				if caso == "etapa" {
					m.Etapa = "consulta"
					m.ReciboBolsa = nil
				}
				return materialContinuacionPGPrueba(t, m, accion, permisos), nil
			})
			if caso == "sql" {
				tx.fila = filaEjecucionSeleccionO6Prueba{err: &pgconn.PgError{Code: "P0603", Message: "detalle privado"}}
			}
			if caso == "commit" {
				tx.errCommit = errors.New("detalle privado")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if caso == "cancelacion" {
				cancel()
			}
			repo := &RegistroContinuacionLlamamientoPostgreSQL{pool: pool, proveedor: p}
			obtenido, err := repo.Confirmar(ctx, s, original.ReciboBolsa)
			if err == nil || obtenido != (ports.ResultadoContinuacionLlamamiento{}) || permisos > 1 || pool.inicios > 1 || strings.Contains(err.Error(), "privado") {
				t.Fatal("filtración, éxito o reintento")
			}
			if caso != "commit" && tx.confirmaciones != 0 {
				t.Fatal("confirmó fallo")
			}
		})
	}
	for code, esperado := range map[string]error{"P0600": ports.ErrOperacionContinuacionInvalida, "P0601": ports.ErrOperacionContinuacionConflicto, "P0602": ports.ErrOperacionContinuacionConflicto, "P0603": ports.ErrOperacionContinuacionDenegada, "P0604": ports.ErrOperacionContinuacionNoDisponible} {
		if !errors.Is(normalizarErrorContinuacion(context.Background(), &pgconn.PgError{Code: code}), esperado) {
			t.Fatal("normalización", code)
		}
	}
}
