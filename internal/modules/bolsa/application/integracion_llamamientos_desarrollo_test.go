package application

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/fuentesintetica"
	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorIntegracionPrueba func(context.Context, string, dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)

func (f autorizadorIntegracionPrueba) AutorizarOperacion(c context.Context, a string, r dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return f(c, a, r)
}

// Doble exclusivo de unidad; no acredita PostgreSQL, autorización ni durabilidad.
type repositorioIntegracionPrueba struct {
	filas     map[string][]byte
	recibos   map[string]ports.ReciboLlamamientoDesarrollo
	fallo     bool
	guardados int
}

func (r *repositorioIntegracionPrueba) BuscarOperacion(_ context.Context, ref string) (ports.RegistroLlamamientoDesarrollo, bool, error) {
	b, ok := r.filas[ref]
	var registro ports.RegistroLlamamientoDesarrollo
	if ok {
		_ = json.Unmarshal(b, &registro)
	}
	return registro, ok, nil
}
func (r *repositorioIntegracionPrueba) Guardar(_ context.Context, d ports.RegistroLlamamientoDesarrollo, _ puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboLlamamientoDesarrollo, error) {
	r.guardados++
	if r.fallo {
		return ports.ReciboLlamamientoDesarrollo{}, ports.ErrIntegracionLlamamientoDesarrollo
	}
	if recibo, existe := r.recibos[d.OperacionRef]; existe {
		return recibo, nil
	}
	b, err := d.Canonico()
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	r.filas[d.OperacionRef] = b
	recibo := ports.ReciboLlamamientoDesarrollo{Registro: d, ReciboRef: "recibo:" + d.OperacionRef, AuditoriaRef: "auditoria:" + d.OperacionRef,
		EventoRef: "evento:" + d.OperacionRef, ConfirmadaEn: instanteAplicacionLlamamientoPrueba.Add(time.Second)}
	r.recibos[d.OperacionRef] = recibo
	return recibo, nil
}
func fuenteIntegracionPrueba(t *testing.T) (*fuentesintetica.FuenteLlamamientos, *relojFijoLlamamiento, []byte, []byte, ed25519.PublicKey) {
	t.Helper()
	d := datosAutoritativosAplicacionPrueba(t, 3)
	estados := []string{"disponible", "ocupado", "no_disponible", "excluido", "renuncia_pendiente"}
	reglas := make([]fuentesintetica.ReglaEstado, 0, 5)
	for _, estado := range estados {
		h := sha256.Sum256([]byte(estado))
		resultado := domain.ResultadoNoElegible
		if estado == "disponible" {
			resultado = domain.ResultadoElegible
		}
		reglas = append(reglas, fuentesintetica.ReglaEstado{EstadoClave: estado, EstadoVersion: 1, HuellaEstadoSHA256: hex.EncodeToString(h[:]),
			Resultado: resultado, Motivo: domain.MotivoEvaluacionLlamamiento{Clave: "fuente_sintetica", ReglaRef: "regla:" + estado, VersionRegla: 1, HuellaReglaSHA256: hex.EncodeToString(h[:])}})
	}
	for i := range d.Entradas {
		regla := reglas[0]
		if i == 0 {
			regla = reglas[1]
		}
		for j := range d.Entradas[i].Participacion.Situaciones {
			s := &d.Entradas[i].Participacion.Situaciones[j]
			s.EstadoClave = regla.EstadoClave
			s.EstadoVersion = 1
			s.HuellaEstadoSHA256 = regla.HuellaEstadoSHA256
		}
	}
	reloj := &relojFijoLlamamiento{instante: instanteAplicacionLlamamientoPrueba}
	doc := fuentesintetica.DocumentoFuenteLlamamientos{Esquema: fuentesintetica.EsquemaFuenteLlamamientos,
		OrigenRef: "origen:sintetico", Version: 1, VigenteDesde: reloj.instante.Add(-time.Hour), VigenteHasta: reloj.instante.Add(time.Hour),
		Datos: d, Reglas: reglas}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	firma := ed25519.Sign(privada, fuentesintetica.MaterialFirmaFuenteLlamamientos(b))
	fuente, err := fuentesintetica.NuevaFuenteLlamamientos(b, firma, publica, reloj)
	if err != nil {
		t.Fatal(err)
	}
	return fuente, reloj, b, firma, publica
}
func materialIntegracionPrueba(t *testing.T, accion string, recurso dominiovec.RecursoAutorizable) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:unidad", h, h, "contexto:unidad", h,
		accion, recurso.Referencia, huella, ports.AudienciaIntegracionLlamamientoDesarrollo, instanteAplicacionLlamamientoPrueba, instanteAplicacionLlamamientoPrueba.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	publica, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	// Solo transporte estructural: deliberadamente no se presenta como capacidad
	// criptográfica válida. El consumidor SQL debe rechazar estas piezas.
	e, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen,
		[]byte("{}"), []byte("{}"), []byte("{}"), 1, 1, []byte("unidad"), []byte("unidad"), []byte("unidad"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func servicioIntegracionPrueba(t *testing.T) (*ServicioIntegracionLlamamientosDesarrollo, *repositorioIntegracionPrueba, *relojFijoLlamamiento, *int) {
	t.Helper()
	fuente, reloj, _, _, _ := fuenteIntegracionPrueba(t)
	repositorio := &repositorioIntegracionPrueba{filas: map[string][]byte{}, recibos: map[string]ports.ReciboLlamamientoDesarrollo{}}
	autorizaciones := 0
	a := autorizadorIntegracionPrueba(func(_ context.Context, accion string, recurso dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		autorizaciones++
		return materialIntegracionPrueba(t, accion, recurso), nil
	})
	s, err := NuevoServicioIntegracionLlamamientosDesarrollo(fuente, repositorio, a, reloj)
	if err != nil {
		t.Fatal(err)
	}
	return s, repositorio, reloj, &autorizaciones
}

func TestIntegracionLlamamientosDesarrolloOrdenCompletaPrimerElegibleYAgregado(t *testing.T) {
	s, r, _, autorizaciones := servicioIntegracionPrueba(t)
	ctx := context.Background()
	d, err := s.ConsultarDisponibilidad(ctx, "necesidad:aplicacion:0001", 3)
	if err != nil {
		t.Fatal(err)
	}
	if d.CantidadDisponible != 2 || !d.CantidadExacta {
		t.Fatalf("disponibilidad incorrecta: %+v", d)
	}
	orden := ports.PeticionLlamamientoDesarrollo{OperacionRef: "operacion:orden", NecesidadRef: d.Necesidad.NecesidadRef, MaximoPosiciones: 3}
	ro, err := s.PrepararOrden(ctx, orden)
	if err != nil {
		t.Fatal(err)
	}
	if len(ro.Registro.Instantanea.Entradas) != 3 || len(r.filas) != 1 {
		t.Fatal("orden truncada o no guardada")
	}
	peticion := ports.PeticionLlamamientoDesarrollo{OperacionRef: "operacion:apertura", NecesidadRef: orden.NecesidadRef, OrdenOperacionRef: orden.OperacionRef, MaximoPosiciones: 3}
	rp, err := s.SolicitarLlamamiento(ctx, peticion)
	if err != nil {
		t.Fatal(err)
	}
	if rp.Registro.Propuesta.OrdenSeleccionado != 2 || len(rp.Registro.Propuesta.Evaluaciones) != 2 ||
		rp.Registro.Llamamiento == nil || rp.Registro.EstadoLlamamiento != domain.EstadoLlamamientoAbierto ||
		rp.Registro.Llamamiento.PropuestaRef != rp.Registro.Propuesta.PropuestaRef || len(r.filas) != 2 {
		t.Fatal("propuesta o apertura incoherente")
	}
	// Nuevo servicio, sin estado de aplicación: el registro existente gobierna
	// replay y se solicita una autorización nueva antes de devolverlo.
	s2, err := NuevoServicioIntegracionLlamamientosDesarrollo(s.fuente, r, s.autorizador, s.reloj)
	if err != nil {
		t.Fatal(err)
	}
	rr, err := s2.SolicitarLlamamiento(ctx, peticion)
	if err != nil {
		t.Fatal(err)
	}
	if rr.ReciboRef != rp.ReciboRef || rr.EventoRef != rp.EventoRef || len(r.filas) != 2 || *autorizaciones != 3 {
		t.Fatal("replay duplicó o evitó autorización")
	}
}
func TestIntegracionLlamamientosDesarrolloDeniegaAntesDeConfirmar(t *testing.T) {
	s, r, _, _ := servicioIntegracionPrueba(t)
	p := ports.PeticionLlamamientoDesarrollo{OperacionRef: "operacion:orden", NecesidadRef: "necesidad:aplicacion:0001", MaximoPosiciones: 3}
	s.autorizador = autorizadorIntegracionPrueba(func(context.Context, string, dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("denegada")
	})
	if _, err := s.PrepararOrden(context.Background(), p); err == nil || r.guardados != 0 || len(r.filas) != 0 {
		t.Fatal("efecto sin autorización")
	}
}
func TestIntegracionLlamamientosDesarrolloFalloRepositorioNoEsExito(t *testing.T) {
	s, r, _, _ := servicioIntegracionPrueba(t)
	r.fallo = true
	p := ports.PeticionLlamamientoDesarrollo{OperacionRef: "operacion:orden", NecesidadRef: "necesidad:aplicacion:0001", MaximoPosiciones: 3}
	recibo, err := s.PrepararOrden(context.Background(), p)
	if err == nil || recibo.ReciboRef != "" || len(r.filas) != 0 {
		t.Fatal("fallo convertido en recibo")
	}
}
func TestIntegracionLlamamientosDesarrolloFuenteFirmaYCaducidad(t *testing.T) {
	f, reloj, b, firma, publica := fuenteIntegracionPrueba(t)
	bAlterado := append([]byte(nil), b...)
	bAlterado[len(bAlterado)-2] ^= 1
	if _, err := fuentesintetica.NuevaFuenteLlamamientos(bAlterado, firma, publica, reloj); err == nil {
		t.Fatal("fuente alterada aceptada")
	}
	otra, _, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := fuentesintetica.NuevaFuenteLlamamientos(b, firma, otra, reloj); err == nil {
		t.Fatal("raíz ajena aceptada")
	}
	copia, _, err := f.ExportarFuenteFirmada(context.Background(), "necesidad:aplicacion:0001")
	if err != nil {
		t.Fatal(err)
	}
	copia[0] = 'x'
	original, _, _ := f.ExportarFuenteFirmada(context.Background(), "necesidad:aplicacion:0001")
	if original[0] == 'x' {
		t.Fatal("fuente compartió memoria")
	}
	reloj.instante = reloj.instante.Add(2 * time.Hour)
	if _, err := f.CargarDatosAutoritativosLlamamiento(context.Background(), "necesidad:aplicacion:0001"); err == nil {
		t.Fatal("fuente expirada aceptada")
	}
}
