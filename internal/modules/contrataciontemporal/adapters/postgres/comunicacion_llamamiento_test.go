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

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func materialComunicacionLocalPrueba() MaterialRegistroComunicacionLlamamiento {
	return MaterialRegistroComunicacionLlamamiento{
		Solicitud: ports.SolicitudRegistrarComunicacionLlamamiento{
			ClaveIdempotencia: "018f47a6-5d2b-4c10-8a11-1234567890ab",
			OrganizacionRef:   "org:sintetica", ExpedienteRef: "exp:sintetico",
			LlamamientoRef: "llamamiento:sintetico", VersionEsperada: 1,
			PruebaEntregaRef: "recibo:seleccion-sintetica",
		},
		Canal: ports.ReferenciaGobernadaComunicacionLlamamiento{
			Referencia: "canal:registro-local", Version: 1, HuellaSHA256: strings.Repeat("a", 64),
		},
		Politica: ports.ReferenciaGobernadaComunicacionLlamamiento{
			Referencia: "politica:registro-local", Version: 1, HuellaSHA256: strings.Repeat("b", 64),
		},
	}
}

func reciboComunicacionLocalPrueba(m MaterialRegistroComunicacionLlamamiento) ports.ComunicacionProbatoria {
	return ports.ComunicacionProbatoria{
		Solicitud: m.Solicitud, ComunicacionRef: "comunicacion:sintetica",
		Canal: m.Canal, Politica: m.Politica, ReciboRef: "recibo:sintetico",
		AuditoriaRef: "auditoria:sintetica", VersionResultante: 2,
		Estado:            ports.ResultadoComunicacionLlamamientoLocal,
		RegistradaEn:      time.Date(2026, 9, 5, 12, 0, 0, 123000000, time.UTC),
		IntencionEnvioRef: "outbox:sintetico",
	}
}

func TestComunicacionLlamamientoMaterialLigaCadaCampo(t *testing.T) {
	m := materialComunicacionLocalPrueba()
	recurso, err := RecursoRegistroComunicacionLlamamiento(m)
	if err != nil {
		t.Fatal(err)
	}
	original, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	for nombre, mutar := range map[string]func(*MaterialRegistroComunicacionLlamamiento){
		"clave": func(m *MaterialRegistroComunicacionLlamamiento) {
			m.Solicitud.ClaveIdempotencia = "118f47a6-5d2b-4c10-8a11-1234567890ab"
		},
		"organizacion":     func(m *MaterialRegistroComunicacionLlamamiento) { m.Solicitud.OrganizacionRef += "b" },
		"expediente":       func(m *MaterialRegistroComunicacionLlamamiento) { m.Solicitud.ExpedienteRef += "b" },
		"llamamiento":      func(m *MaterialRegistroComunicacionLlamamiento) { m.Solicitud.LlamamientoRef += "b" },
		"version":          func(m *MaterialRegistroComunicacionLlamamiento) { m.Solicitud.VersionEsperada++ },
		"recibo_seleccion": func(m *MaterialRegistroComunicacionLlamamiento) { m.Solicitud.PruebaEntregaRef += "b" },
		"canal":            func(m *MaterialRegistroComunicacionLlamamiento) { m.Canal.Referencia += "b" },
		"canal_version":    func(m *MaterialRegistroComunicacionLlamamiento) { m.Canal.Version++ },
		"canal_huella":     func(m *MaterialRegistroComunicacionLlamamiento) { m.Canal.HuellaSHA256 = strings.Repeat("c", 64) },
		"politica":         func(m *MaterialRegistroComunicacionLlamamiento) { m.Politica.Referencia += "b" },
		"politica_version": func(m *MaterialRegistroComunicacionLlamamiento) { m.Politica.Version++ },
		"politica_huella":  func(m *MaterialRegistroComunicacionLlamamiento) { m.Politica.HuellaSHA256 = strings.Repeat("c", 64) },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterado := m
			mutar(&alterado)
			r, err := RecursoRegistroComunicacionLlamamiento(alterado)
			if err != nil {
				t.Fatal(err)
			}
			h, err := r.HuellaContextoAutorizacionSHA256()
			if err != nil || h == original {
				t.Fatal("campo no ligado a autorización")
			}
		})
	}
	if recurso.Referencia != m.Solicitud.ExpedienteRef || len(recurso.Ambitos) != 1 ||
		len(recurso.Atributos) != 1 || recurso.Tipo != TipoRecursoRegistroComunicacionLlamamiento {
		t.Fatal("recurso divergente")
	}
}

type proveedorComunicacionLocalPrueba struct {
	m              MaterialRegistroComunicacionLlamamiento
	err            error
	autorizaciones int
}

func (p *proveedorComunicacionLocalPrueba) PrepararRegistroComunicacion(context.Context, ports.SolicitudRegistrarComunicacionLlamamiento) (MaterialRegistroComunicacionLlamamiento, error) {
	return p.m, p.err
}
func (p *proveedorComunicacionLocalPrueba) AutorizarRegistroComunicacion(context.Context, MaterialRegistroComunicacionLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.autorizaciones++
	return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, p.err
}

func TestComunicacionLlamamientoFallaAntesDeSQLSinAutoridad(t *testing.T) {
	m := materialComunicacionLocalPrueba()
	p := &proveedorComunicacionLocalPrueba{m: m}
	pool := &iniciadorEjecucionSeleccionO6Prueba{}
	adaptador := &TransaccionComunicacionLlamamientoPostgreSQL{pool: pool, proveedor: p}
	versionExpediente := m.Solicitud
	versionExpediente.VersionEsperada = 6
	if _, err := adaptador.RegistrarComunicacion(context.Background(), versionExpediente); !errors.Is(err, ports.ErrVersionComunicacionLlamamientoEnConflicto) || p.autorizaciones != 0 {
		t.Fatal("versión del expediente confundida con la del llamamiento")
	}
	resultado, err := adaptador.RegistrarComunicacion(context.Background(), m.Solicitud)
	if !errors.Is(err, ports.ErrOperacionComunicacionLlamamientoDenegada) ||
		resultado != (ports.ComunicacionProbatoria{}) || pool.inicios != 0 || p.autorizaciones != 1 {
		t.Fatal("material vacío alcanzó persistencia")
	}
	p.m.Solicitud.ExpedienteRef = "exp:otro"
	_, err = adaptador.RegistrarComunicacion(context.Background(), m.Solicitud)
	if !errors.Is(err, ports.ErrResultadoComunicacionLlamamientoNoConfiable) || p.autorizaciones != 1 {
		t.Fatal("preparación divergente alcanzó autorización")
	}
	if a, err := NuevaTransaccionComunicacionLlamamientoPostgreSQL(nil, p); a != nil || err == nil {
		t.Fatal("constructor admitió pool nulo")
	}
}

func TestComunicacionLlamamientoReciboLocalNoInventaEntrega(t *testing.T) {
	m := materialComunicacionLocalPrueba()
	r := reciboComunicacionLocalPrueba(m)
	for _, estado := range []ports.EstadoResultadoComunicacionLlamamiento{
		ports.ResultadoComunicacionLlamamientoLocal, ports.ResultadoComunicacionLlamamientoReplayLocal,
	} {
		r.Estado = estado
		b, _ := json.Marshal(r)
		obtenido, err := decodificarComunicacionLlamamientoLocal(string(b), m.Solicitud)
		if err != nil || obtenido != r {
			t.Fatal("recibo local no recuperable", err)
		}
	}
	r.RespuestaHasta = r.RegistradaEn.Add(time.Hour)
	b, _ := json.Marshal(r)
	if _, err := decodificarComunicacionLlamamientoLocal(string(b), m.Solicitud); err == nil {
		t.Fatal("plazo ficticio aceptado")
	}
	r.RespuestaHasta = time.Time{}
	r.EntregadaEn = r.RegistradaEn
	b, _ = json.Marshal(r)
	if _, err := decodificarComunicacionLlamamientoLocal(string(b), m.Solicitud); err == nil {
		t.Fatal("entrega ficticia aceptada")
	}
	r = reciboComunicacionLocalPrueba(m)
	b, _ = json.Marshal(r)
	if _, err := decodificarComunicacionLlamamientoLocal(string(b)+"{}", m.Solicitud); err == nil {
		t.Fatal("JSON sobrante aceptado")
	}
}

func TestComunicacionLlamamientoUnaTransaccionSinReintentoCiego(t *testing.T) {
	// Dobles de pgx: prueban el protocolo Go, no acreditan dinámica PostgreSQL.
	m := materialComunicacionLocalPrueba()
	r := reciboComunicacionLocalPrueba(m)
	contenido, _ := codificarMaterialComunicacionLlamamiento(m)
	b, _ := json.Marshal(r)
	for _, caso := range []struct {
		nombre                          string
		configuracion, consulta, commit error
		resultado                       string
	}{
		{"confirmado", nil, nil, nil, string(b)},
		{"configuracion", errors.New("fallo"), nil, nil, string(b)},
		{"sql", nil, errors.New("fallo"), nil, string(b)},
		{"recibo_invalido", nil, nil, nil, "{}"},
		{"commit_incierto", nil, nil, errors.New("respuesta perdida"), string(b)},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &transaccionEjecucionSeleccionO6Prueba{
				fila:          filaEjecucionSeleccionO6Prueba{valores: []any{caso.resultado}, err: caso.consulta},
				errConfigurar: caso.configuracion, errCommit: caso.commit,
			}
			pool := &iniciadorEjecucionSeleccionO6Prueba{tx: tx}
			a := &TransaccionComunicacionLlamamientoPostgreSQL{pool: pool}
			resultado, err := a.registrar(context.Background(), m.Solicitud, m, contenido, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{})
			if caso.nombre == "confirmado" {
				if err != nil || resultado != r || tx.confirmaciones != 1 {
					t.Fatal(err)
				}
				if len(tx.consultas) != 1 || len(tx.argumentos[0]) != 11 ||
					!strings.Contains(tx.consultas[0], "registrar_comunicacion_llamamiento_local_v1") {
					t.Fatal("la operación no usa su única fachada")
				}
			} else if err == nil || resultado != (ports.ComunicacionProbatoria{}) {
				t.Fatal("fallo convertido en éxito")
			}
			if pool.inicios != 1 || pool.opciones.IsoLevel != pgx.Serializable ||
				pool.opciones.AccessMode != pgx.ReadWrite || tx.reversiones != 1 {
				t.Fatal("transacción abierta o reintentada")
			}
			if caso.nombre != "confirmado" && caso.nombre != "commit_incierto" && tx.confirmaciones != 0 {
				t.Fatal("confirmado tras fallo previo")
			}
		})
	}
}

func TestComunicacionLlamamientoErroresNoFiltranSQL(t *testing.T) {
	for codigo, esperado := range map[string]error{
		"42501": ports.ErrOperacionComunicacionLlamamientoDenegada,
		"P0541": ports.ErrClaveComunicacionLlamamientoUsada,
		"P0542": ports.ErrVersionComunicacionLlamamientoEnConflicto,
		"23505": ErrPersistenciaComunicacionLlamamientoNoDisponible,
	} {
		err := normalizarErrorComunicacionLlamamiento(context.Background(), &pgconn.PgError{
			Code: codigo, Message: "detalle interno sintético",
		})
		if !errors.Is(err, esperado) || strings.Contains(err.Error(), "detalle") {
			t.Fatal(err)
		}
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if !errors.Is(normalizarErrorComunicacionLlamamiento(ctx, errors.New("fallo")), context.Canceled) {
		t.Fatal("cancelación perdida")
	}
}

func TestComunicacionLlamamientoResolverLocalPermaneceDenegado(t *testing.T) {
	a := &TransaccionComunicacionLlamamientoPostgreSQL{}
	s := ports.SolicitudResolverLlamamiento{
		ClaveIdempotencia: "018f47a6-5d2b-4c10-8a11-1234567890ab",
		OrganizacionRef:   "org:sintetica", ExpedienteRef: "exp:sintetico",
		LlamamientoRef: "llamamiento:sintetico", ComunicacionRef: "comunicacion:sintetica",
		VersionEsperada: 2, Respuesta: ports.RespuestaLlamamientoExpirada,
	}
	r, err := a.ResolverLlamamiento(context.Background(), s)
	if !errors.Is(err, ports.ErrOperacionComunicacionLlamamientoDenegada) ||
		!reflect.DeepEqual(r, ports.ResultadoResolucionLlamamiento{}) {
		t.Fatal("expiración inventada")
	}
}
