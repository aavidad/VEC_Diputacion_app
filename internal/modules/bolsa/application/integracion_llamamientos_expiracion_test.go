package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Sin respuesta en plazo: el terminal cierra el llamamiento como expirado con
// el permiso de no aceptación de RRHH, y el siguiente llamamiento continúa
// desde ese terminal igual que tras una renuncia.
func TestIntegracionLlamamientosDesarrolloExpiracionCierraYContinua(t *testing.T) {
	s, repo, _, permisos, p := aperturaAceptacionIntegracionPrueba(t)
	p.OperacionRef = "operacion:expiracion"
	var acciones []string
	anterior := s.autorizador
	s.autorizador = autorizadorIntegracionPrueba(func(ctx context.Context, accion string, recurso dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		acciones = append(acciones, accion)
		return anterior.AutorizarOperacion(ctx, accion, recurso)
	})
	r, err := s.ExpirarLlamamiento(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if r.Registro.Tipo != ports.TipoExpiracionRRHHDesarrollo || r.Registro.EstadoLlamamiento != domain.EstadoLlamamientoExpirado ||
		r.Registro.Accion() != ports.AccionRenunciarLlamamientoRRHHDesarrollo || r.Registro.Llamamiento.Version != 2 ||
		len(acciones) != 1 || acciones[0] != ports.AccionRenunciarLlamamientoRRHHDesarrollo {
		t.Fatal("expiración sin terminal «sin respuesta» o con otro permiso", acciones)
	}
	guardados, autorizaciones := repo.guardados, *permisos
	if x, err := s.RenunciarLlamamiento(context.Background(), p); err == nil || x.ReciboRef != "" || repo.guardados != guardados || *permisos != autorizaciones {
		t.Fatal("renuncia reinterpretó una expiración durable")
	}
	replay, err := s.ExpirarLlamamiento(context.Background(), p)
	if err != nil || replay.ReciboRef != r.ReciboRef || *replay.Registro.Resolucion != *r.Registro.Resolucion {
		t.Fatal("replay de expiración divergente", err)
	}
	sig, err := s.SolicitarSiguienteLlamamiento(context.Background(), ports.PeticionSiguienteLlamamientoDesarrollo{
		OperacionRef: "operacion:siguiente", IntencionRef: "intencion:siguiente", TerminalOperacionRef: p.OperacionRef})
	if err != nil {
		t.Fatal(err)
	}
	if sig.Registro.Tipo != "propuesta" || sig.Registro.Propuesta.Continuacion == nil ||
		sig.Registro.Propuesta.Continuacion.TerminalOperacionRef != p.OperacionRef ||
		sig.Registro.Propuesta.OrdenSeleccionado != 3 || sig.Registro.EstadoLlamamiento != domain.EstadoLlamamientoAbierto ||
		sig.Registro.Accion() != ports.AccionAbrirSiguienteLlamamientoDesarrollo {
		t.Fatal("siguiente tras expiración incorrecto")
	}
}

func TestIntegracionLlamamientosDesarrolloExpiracionCanonExigeEstadoExpirado(t *testing.T) {
	s, repo, _, _, p := aperturaAceptacionIntegracionPrueba(t)
	p.OperacionRef = "operacion:expiracion"
	if _, err := s.ExpirarLlamamiento(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	var r ports.RegistroLlamamientoDesarrollo
	if err := json.Unmarshal(repo.filas[p.OperacionRef], &r); err != nil {
		t.Fatal(err)
	}
	r.EstadoLlamamiento = domain.EstadoLlamamientoRenunciado
	if _, err := r.Canonico(); err == nil || r.Accion() != "" {
		t.Fatal("expiración con estado de renuncia aceptada")
	}
	r.Tipo, r.EstadoLlamamiento = "renuncia_rrhh", domain.EstadoLlamamientoExpirado
	if _, err := r.Canonico(); err == nil || r.Accion() != "" {
		t.Fatal("renuncia con estado expirado aceptada")
	}
}

type repositorioRelojPrueba struct {
	*repositorioIntegracionPrueba
	reloj *relojFijoLlamamiento
}

// Guardar fecha la confirmación con el reloj del escenario, como haría SQL.
func (r repositorioRelojPrueba) Guardar(ctx context.Context, d ports.RegistroLlamamientoDesarrollo, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboLlamamientoDesarrollo, error) {
	recibo, err := r.repositorioIntegracionPrueba.Guardar(ctx, d, m)
	recibo.ConfirmadaEn = r.reloj.instante
	return recibo, err
}

// Genera, solo si se pide con VEC_BOLSA39_FIXTURES, los registros canónicos
// que la prueba PostgreSQL de Bolsa 000039 guarda con la función real. La
// orden y la apertura quedan dos minutos antes de ahora; la expiración se
// resuelve a +20 s y el siguiente se propone a +40 s, para que SQL los
// confirme en orden cronológico real esperando a esos instantes.
func TestGenerarFixturesBolsa39(t *testing.T) {
	destino := os.Getenv("VEC_BOLSA39_FIXTURES")
	if destino == "" {
		t.Skip("solo bajo demanda de la prueba PostgreSQL de Bolsa 000039")
	}
	generado := time.Now().UTC().Truncate(time.Microsecond)
	original := instanteAplicacionLlamamientoPrueba
	instanteAplicacionLlamamientoPrueba = generado.Add(-2 * time.Minute)
	defer func() { instanteAplicacionLlamamientoPrueba = original }()
	s, repo, reloj, _, p := aperturaAceptacionIntegracionPrueba(t)
	s.repositorio = repositorioRelojPrueba{repo, reloj}
	p.OperacionRef = "operacion:expiracion"
	// Referencia real de una regla del catálogo (con «_»), como en la composición.
	p.Resolucion.PoliticaRef = "vec.bolsa.reglas:3:b08.sin_respuesta_baja"
	reloj.instante = generado.Add(20 * time.Second)
	if _, err := s.ExpirarLlamamiento(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	reloj.instante = generado.Add(40 * time.Second)
	if _, err := s.SolicitarSiguienteLlamamiento(context.Background(), ports.PeticionSiguienteLlamamientoDesarrollo{
		OperacionRef: "operacion:siguiente", IntencionRef: "intencion:siguiente", TerminalOperacionRef: p.OperacionRef}); err != nil {
		t.Fatal(err)
	}
	salida := map[string]string{"generado_en": generado.Format(time.RFC3339Nano)}
	for _, ref := range []string{"operacion:orden", "operacion:apertura", p.OperacionRef, "operacion:siguiente"} {
		salida[ref] = base64.StdEncoding.EncodeToString(repo.filas[ref])
	}
	b, err := json.Marshal(salida)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destino, "registros.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}
