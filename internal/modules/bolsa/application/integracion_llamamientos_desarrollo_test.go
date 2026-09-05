package application

import (
	"bytes"
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

func aperturaAceptacionIntegracionPrueba(t *testing.T) (*ServicioIntegracionLlamamientosDesarrollo, *repositorioIntegracionPrueba, *relojFijoLlamamiento, *int, ports.PeticionResolverLlamamientoDesarrollo) {
	t.Helper()
	s, r, reloj, autorizaciones := servicioIntegracionPrueba(t)
	ctx := context.Background()
	_, err := s.PrepararOrden(ctx, ports.PeticionLlamamientoDesarrollo{
		OperacionRef: "operacion:orden", NecesidadRef: "necesidad:aplicacion:0001", MaximoPosiciones: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.SolicitarLlamamiento(ctx, ports.PeticionLlamamientoDesarrollo{
		OperacionRef: "operacion:apertura", OrdenOperacionRef: "operacion:orden", NecesidadRef: "necesidad:aplicacion:0001", MaximoPosiciones: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Referencias exclusivas de unidad. No simulan un verificador de plazo.
	p := ports.PeticionResolverLlamamientoDesarrollo{OperacionRef: "operacion:aceptacion", Resolucion: ports.ResolucionLlamamientoDesarrollo{
		AperturaOperacionRef: "operacion:apertura", JustificanteRef: "justificante:unidad", EvaluacionPlazoRef: "evaluacion:unidad",
		PoliticaRef: "politica:unidad", PoliticaVersion: 1, PoliticaSHA256: strings.Repeat("a", 64), VersionEsperada: 1,
	}}
	return s, r, reloj, autorizaciones, p
}

type repositorioAperturaCompartidaPrueba struct {
	ports.RepositorioLlamamientoDesarrollo
	apertura ports.RegistroLlamamientoDesarrollo
}

func (r repositorioAperturaCompartidaPrueba) BuscarOperacion(ctx context.Context, ref string) (ports.RegistroLlamamientoDesarrollo, bool, error) {
	if ref == r.apertura.OperacionRef {
		return r.apertura, true, nil
	}
	return r.RepositorioLlamamientoDesarrollo.BuscarOperacion(ctx, ref)
}

func TestIntegracionLlamamientosDesarrolloAceptacionDominioClonadoYReplay(t *testing.T) {
	for _, caso := range []struct {
		tipo     string
		estado   domain.EstadoLlamamiento
		accion   string
		resolver func(*ServicioIntegracionLlamamientosDesarrollo, context.Context, ports.PeticionResolverLlamamientoDesarrollo) (ports.ReciboLlamamientoDesarrollo, error)
	}{
		{"aceptacion_rrhh", domain.EstadoLlamamientoAceptado, ports.AccionAceptarLlamamientoRRHHDesarrollo, (*ServicioIntegracionLlamamientosDesarrollo).AceptarLlamamiento},
		{"renuncia_rrhh", domain.EstadoLlamamientoRenunciado, ports.AccionRenunciarLlamamientoRRHHDesarrollo, (*ServicioIntegracionLlamamientosDesarrollo).RenunciarLlamamiento},
	} {
		t.Run(caso.tipo, func(t *testing.T) {
			s, r, reloj, autorizaciones, p := aperturaAceptacionIntegracionPrueba(t)
			p.OperacionRef = "operacion:" + caso.tipo
			apertura := r.recibos[p.Resolucion.AperturaOperacionRef].Registro
			canonApertura, err := apertura.Canonico()
			if err != nil {
				t.Fatal(err)
			}
			s.repositorio = repositorioAperturaCompartidaPrueba{r, apertura}
			// El reloj puede venir de otra zona: la fecha persistida sí debe ser UTC.
			reloj.instante = reloj.instante.In(time.FixedZone("desarrollo", 3600))
			recibo, err := caso.resolver(s, context.Background(), p)
			if err != nil {
				t.Fatal(err)
			}
			registro := recibo.Registro
			if registro.Tipo != caso.tipo || registro.EstadoLlamamiento != caso.estado || registro.Accion() != caso.accion ||
				registro.Llamamiento.Version != 2 || registro.Llamamiento.LlamamientoRef != apertura.Llamamiento.LlamamientoRef ||
				registro.OrdenOperacionRef != apertura.OrdenOperacionRef || registro.Resolucion == nil ||
				registro.Resolucion.ResueltaEn.Location() != time.UTC || !registro.Resolucion.ResueltaEn.Equal(reloj.instante) ||
				len(r.filas) != 3 || *autorizaciones != 3 || !p.Resolucion.ResueltaEn.IsZero() {
				t.Fatal("terminal, referencias, fecha o autorización incorrectos")
			}
			reloj.instante = reloj.instante.Add(time.Minute)
			s2, err := NuevoServicioIntegracionLlamamientosDesarrollo(s.fuente, s.repositorio, s.autorizador, reloj)
			if err != nil {
				t.Fatal(err)
			}
			replay, err := caso.resolver(s2, context.Background(), p)
			if err != nil {
				t.Fatal(err)
			}
			if replay.ReciboRef != recibo.ReciboRef || replay.EventoRef != recibo.EventoRef ||
				replay.ConfirmadaEn != recibo.ConfirmadaEn || *replay.Registro.Resolucion != *registro.Resolucion ||
				*autorizaciones != 4 || r.guardados != 4 || len(r.filas) != 3 {
				t.Fatal("replay cambió fecha, duplicó o evitó reautorización")
			}
			registro.Fuente[0] ^= 1
			registro.FirmaFuente[0] ^= 1
			registro.Propuesta.Evaluaciones[0].Motivos[0].Clave = "alterada"
			registro.Instantanea.Entradas[0].Participacion.Situaciones[0].EstadoClave = "alterada"
			registro.Llamamiento.Version = 9
			despues, err := apertura.Canonico()
			if err != nil || !bytes.Equal(canonApertura, despues) {
				t.Fatal("aceptación comparte memoria o mutó la apertura")
			}
		})
	}
}

func TestIntegracionLlamamientosDesarrolloRenunciaNoReutilizaAceptacionNiCambiaReplay(t *testing.T) {
	s, repo, _, autorizaciones, p := aperturaAceptacionIntegracionPrueba(t)
	p.OperacionRef = "operacion:renuncia"
	if _, err := s.RenunciarLlamamiento(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	guardados, permisos := repo.guardados, *autorizaciones
	if r, err := s.AceptarLlamamiento(context.Background(), p); err == nil || r.ReciboRef != "" || repo.guardados != guardados || *autorizaciones != permisos {
		t.Fatal("aceptación reinterpretó una renuncia durable")
	}
	otra := p
	otra.Resolucion.PoliticaVersion++
	if r, err := s.RenunciarLlamamiento(context.Background(), otra); err == nil || r.ReciboRef != "" || repo.guardados != guardados || *autorizaciones != permisos {
		t.Fatal("replay divergente alcanzó autorización o guardado")
	}
	s.autorizador = autorizadorIntegracionPrueba(func(_ context.Context, accion string, recurso dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		if accion != ports.AccionRenunciarLlamamientoRRHHDesarrollo {
			t.Fatal("pidió otro permiso")
		}
		return materialIntegracionPrueba(t, ports.AccionAceptarLlamamientoRRHHDesarrollo, recurso), nil
	})
	if r, err := s.RenunciarLlamamiento(context.Background(), p); err == nil || r.ReciboRef != "" || repo.guardados != guardados {
		t.Fatal("permiso de aceptación habilitó replay de renuncia")
	}
}

func TestIntegracionLlamamientosDesarrolloAceptacionRechazaCambiosEnReplay(t *testing.T) {
	s, r, _, _, p := aperturaAceptacionIntegracionPrueba(t)
	if _, err := s.AceptarLlamamiento(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	guardados := r.guardados
	for nombre, cambiar := range map[string]func(*ports.PeticionResolverLlamamientoDesarrollo){
		"justificante": func(p *ports.PeticionResolverLlamamientoDesarrollo) {
			p.Resolucion.JustificanteRef = "justificante:otro"
		},
		"evaluacion": func(p *ports.PeticionResolverLlamamientoDesarrollo) {
			p.Resolucion.EvaluacionPlazoRef = "evaluacion:otra"
		},
		"politica":         func(p *ports.PeticionResolverLlamamientoDesarrollo) { p.Resolucion.PoliticaRef = "politica:otra" },
		"version_politica": func(p *ports.PeticionResolverLlamamientoDesarrollo) { p.Resolucion.PoliticaVersion = 2 },
		"huella_politica": func(p *ports.PeticionResolverLlamamientoDesarrollo) {
			p.Resolucion.PoliticaSHA256 = strings.Repeat("b", 64)
		},
		"apertura": func(p *ports.PeticionResolverLlamamientoDesarrollo) {
			p.Resolucion.AperturaOperacionRef = "operacion:orden"
		},
		"version_apertura": func(p *ports.PeticionResolverLlamamientoDesarrollo) { p.Resolucion.VersionEsperada = 2 },
		"fecha_externa": func(p *ports.PeticionResolverLlamamientoDesarrollo) {
			p.Resolucion.ResueltaEn = instanteAplicacionLlamamientoPrueba
		},
		"colision_orden": func(p *ports.PeticionResolverLlamamientoDesarrollo) { p.OperacionRef = "operacion:orden" },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := p
			cambiar(&alterada)
			recibo, err := s.AceptarLlamamiento(context.Background(), alterada)
			if err == nil || recibo.ReciboRef != "" || r.guardados != guardados || len(r.filas) != 3 {
				t.Fatal("replay incompatible obtuvo éxito o un guardado")
			}
		})
	}
	// Incluso una alteración que sigue siendo canónica debe contrastarse con
	// la apertura: no basta comparar únicamente la resolución declarada.
	var alterado ports.RegistroLlamamientoDesarrollo
	if err := json.Unmarshal(r.filas[p.OperacionRef], &alterado); err != nil {
		t.Fatal(err)
	}
	alterado.FirmaFuente[0] ^= 1
	r.filas[p.OperacionRef], _ = alterado.Canonico()
	if recibo, err := s.AceptarLlamamiento(context.Background(), p); err == nil || recibo.ReciboRef != "" || r.guardados != guardados {
		t.Fatal("replay aceptó antecedentes diferentes de la apertura")
	}
}

func TestIntegracionLlamamientosDesarrolloAceptacionAutorizacionPropiaYFallos(t *testing.T) {
	for _, modo := range []string{"denegada_nueva", "denegada_replay", "permiso_apertura", "repositorio", "fecha_recibo"} {
		t.Run(modo, func(t *testing.T) {
			s, r, _, _, p := aperturaAceptacionIntegracionPrueba(t)
			if modo == "denegada_replay" || modo == "fecha_recibo" {
				if _, err := s.AceptarLlamamiento(context.Background(), p); err != nil {
					t.Fatal(err)
				}
			}
			guardados, filas := r.guardados, len(r.filas)
			switch modo {
			case "denegada_nueva", "denegada_replay":
				s.autorizador = autorizadorIntegracionPrueba(func(context.Context, string, dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
					return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("denegada")
				})
			case "permiso_apertura":
				s.autorizador = autorizadorIntegracionPrueba(func(_ context.Context, accion string, recurso dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
					if accion != ports.AccionAceptarLlamamientoRRHHDesarrollo {
						t.Fatal("no solicitó permiso propio")
					}
					return materialIntegracionPrueba(t, ports.AccionAbrirLlamamientoDesarrollo, recurso), nil
				})
			case "repositorio":
				r.fallo = true
			case "fecha_recibo":
				recibo := r.recibos[p.OperacionRef]
				recibo.ConfirmadaEn = recibo.ConfirmadaEn.In(time.FixedZone("no-canonica", 0))
				r.recibos[p.OperacionRef] = recibo
			}
			recibo, err := s.AceptarLlamamiento(context.Background(), p)
			if err == nil || recibo.ReciboRef != "" || len(r.filas) != filas {
				t.Fatal("fallo convertido en éxito o duplicado")
			}
			if modo != "repositorio" && modo != "fecha_recibo" && r.guardados != guardados {
				t.Fatal("guardó sin autorización exacta")
			}
		})
	}
}

func TestIntegracionLlamamientosDesarrolloAceptacionSinAperturaOCancelada(t *testing.T) {
	s, r, _, _, p := aperturaAceptacionIntegracionPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := s.AceptarLlamamiento(ctx, p); !errors.Is(err, context.Canceled) || r.guardados != 2 {
		t.Fatal("no respetó cancelación")
	}
	delete(r.filas, p.Resolucion.AperturaOperacionRef)
	if recibo, err := s.AceptarLlamamiento(context.Background(), p); err == nil || recibo.ReciboRef != "" || r.guardados != 2 {
		t.Fatal("aceptación sin apertura real")
	}
}
