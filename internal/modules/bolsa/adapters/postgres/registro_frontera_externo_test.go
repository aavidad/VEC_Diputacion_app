package postgres

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type filaRegistroFronteraBolsaPrueba struct {
	valor bool
	err   error
}

func (f filaRegistroFronteraBolsaPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destinos[0].(*bool)) = f.valor
	return nil
}

type consultorRegistroFronteraBolsaPrueba struct {
	preflight   bool
	guardado    bool
	consultas   int
	argumentos  []any
	registroSQL string
}

type proveedorJustificanteOrigenAusentePrueba struct{ llamadas *int }

func (p proveedorJustificanteOrigenAusentePrueba) PrepararJustificanteRenuncia(*http.Request) (bolsahttp.IdentidadJustificanteRenuncia, error) {
	*p.llamadas = *p.llamadas + 1
	return bolsahttp.IdentidadJustificanteRenuncia{}, nil
}

type fabricaJustificanteOrigenAusentePrueba struct{ llamadas *int }

func (f fabricaJustificanteOrigenAusentePrueba) ServicioJustificanteRenuncia(context.Context, bolsahttp.IdentidadJustificanteRenuncia, string) (docports.SubidorLocal, error) {
	*f.llamadas = *f.llamadas + 1
	return nil, nil
}

func (c *consultorRegistroFronteraBolsaPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	c.consultas++
	if sql == preflightFronteraBolsaExternaSQL {
		return filaRegistroFronteraBolsaPrueba{valor: c.preflight}
	}
	c.argumentos = args
	c.registroSQL = sql
	return filaRegistroFronteraBolsaPrueba{valor: c.guardado}
}

func TestRegistroFronteraBolsaExternaJustificanteAuditaRechazosExactos(t *testing.T) {
	var llamadasIdentidad, llamadasEfecto int
	h, err := bolsahttp.NuevoHandlerJustificanteRenuncia(
		proveedorJustificanteOrigenAusentePrueba{&llamadasIdentidad},
		fabricaJustificanteOrigenAusentePrueba{&llamadasEfecto})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "https://example.invalid"+bolsahttp.RutaJustificantesMiBolsa, nil)
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || llamadasIdentidad != 0 || llamadasEfecto != 0 {
		t.Fatalf("POST sin Origin alcanzó identidad o efecto: estado=%d identidad=%d efecto=%d", w.Code, llamadasIdentidad, llamadasEfecto)
	}

	consultor := &consultorRegistroFronteraBolsaPrueba{preflight: true, guardado: true}
	registrador, err := nuevoRegistradorFronteraBolsaExternaPostgreSQL(t.Context(), consultor)
	if err != nil {
		t.Fatal(err)
	}
	orden := ordenRegistroFronteraBolsaPrueba()
	orden.Ruta = bolsahttp.RutaJustificantesMiBolsa
	if err := registrador.RegistrarAuditoriaFronteraRutaExacta(t.Context(), orden); err != nil {
		t.Fatalf("rechazo con actor reconocido: %v", err)
	}
	if consultor.registroSQL != registrarFronteraBolsaExternaSQL ||
		consultor.argumentos[3] != bolsahttp.RutaJustificantesMiBolsa || consultor.argumentos[4] != orden.ActorRef {
		t.Fatal("el rechazo no llegó a la única función de auditoría")
	}

	orden.Motivo = vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
	orden.ActorRef = ""
	if err := registrador.RegistrarAuditoriaFronteraRutaExacta(t.Context(), orden); err != nil {
		t.Fatalf("rechazo sin actor reconocido: %v", err)
	}
	if consultor.consultas != 3 || consultor.argumentos[4] != "" {
		t.Fatal("el rechazo sin actor no llegó a auditoría")
	}

	orden.Motivo = vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado
	orden.ActorRef = "per_malformada"
	if err := registrador.RegistrarAuditoriaFronteraRutaExacta(t.Context(), orden); !errors.Is(err, vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida) {
		t.Fatalf("actor malformado admitido: %v", err)
	}
	orden.ActorRef = "per_" + strings.Repeat("b", 22)
	orden.Ruta += "/otra"
	if err := registrador.RegistrarAuditoriaFronteraRutaExacta(t.Context(), orden); !errors.Is(err, vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida) {
		t.Fatalf("ruta no exacta admitida: %v", err)
	}
	if consultor.consultas != 3 {
		t.Fatal("una orden inválida llegó a PostgreSQL")
	}
}

func ordenRegistroFronteraBolsaPrueba() vecports.OrdenAuditoriaFronteraRutaExacta {
	return vecports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_" + strings.Repeat("a", 32),
		Motivo:         vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
		Superficie:     vecports.SuperficieAuditoriaFronteraRutaExactaBolsaCandidato,
		Ruta:           "/api/vec/bolsa/mi-bolsa/contacto",
		ActorRef:       "per_" + strings.Repeat("b", 22),
	}
}

func TestRegistroFronteraBolsaExternaPreflightYOrdenCerrados(t *testing.T) {
	denegado := &consultorRegistroFronteraBolsaPrueba{}
	if _, err := nuevoRegistradorFronteraBolsaExternaPostgreSQL(context.Background(), denegado); !errors.Is(err, ErrRegistroFronteraBolsaExternaNoDisponible) {
		t.Fatalf("preflight negativo: %v", err)
	}
	consultor := &consultorRegistroFronteraBolsaPrueba{preflight: true, guardado: true}
	registrador, err := nuevoRegistradorFronteraBolsaExternaPostgreSQL(context.Background(), consultor)
	if err != nil {
		t.Fatal(err)
	}
	orden := ordenRegistroFronteraBolsaPrueba()
	if err := registrador.RegistrarAuditoriaFronteraRutaExacta(context.Background(), orden); err != nil {
		t.Fatal(err)
	}
	if consultor.consultas != 2 || consultor.argumentos[4] != orden.ActorRef {
		t.Fatalf("registro no encaminado: consultas=%d", consultor.consultas)
	}
	for _, mutar := range []func(*vecports.OrdenAuditoriaFronteraRutaExacta){
		func(o *vecports.OrdenAuditoriaFronteraRutaExacta) {
			o.Superficie = "api.usuarios.preferencias.ruta_exacta"
		},
		func(o *vecports.OrdenAuditoriaFronteraRutaExacta) { o.Ruta += "?dato=privado" },
		func(o *vecports.OrdenAuditoriaFronteraRutaExacta) { o.ActorRef = "per_123" },
		func(o *vecports.OrdenAuditoriaFronteraRutaExacta) { o.CorrelacionRef = "peticion-libre" },
		func(o *vecports.OrdenAuditoriaFronteraRutaExacta) {
			o.Motivo = vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
		},
	} {
		invalida := orden
		mutar(&invalida)
		if err := registrador.RegistrarAuditoriaFronteraRutaExacta(context.Background(), invalida); !errors.Is(err, vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida) {
			t.Fatalf("orden abierta: %v", err)
		}
	}
	if consultor.consultas != 2 {
		t.Fatalf("escribió orden inválida: %d", consultor.consultas)
	}
	orden.Motivo = vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
	orden.ActorRef = ""
	if err := registrador.RegistrarAuditoriaFronteraRutaExacta(context.Background(), orden); err != nil {
		t.Fatal(err)
	}
	consultor.guardado = false
	if err := registrador.RegistrarAuditoriaFronteraRutaExacta(context.Background(), orden); !errors.Is(err, ErrRegistroFronteraBolsaExternaNoDisponible) {
		t.Fatalf("fallo de escritura permitido: %v", err)
	}
}
