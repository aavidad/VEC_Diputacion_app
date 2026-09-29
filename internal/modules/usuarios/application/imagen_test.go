package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorImagenPrueba struct {
	denegar    bool
	materiales []ports.MaterialImagen
}

func (p *proveedorImagenPrueba) ProveerMaterialImagen(_ context.Context, vinculo vecdomain.VinculoAutenticacionActorV2, m ports.MaterialImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	datos, err := vinculo.Datos()
	if err != nil || datos.Superficie != m.Superficie {
		return vacia, ports.ErrImagenProhibido
	}
	p.materiales = append(p.materiales, m)
	if p.denegar {
		return vacia, ports.ErrImagenProhibido
	}
	recurso, err := canonico.RecursoImagen(m)
	if err != nil {
		return vacia, err
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vacia, err
	}
	audiencia, _ := canonico.AudienciaImagen(m.Accion, m.Superficie)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(fmt.Sprintf("dec_imagen_%d", len(p.materiales)), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion, m.PersonaRef, huella, audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		return vacia, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte(fmt.Sprint(len(p.materiales))), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
}

type registroImagenPrueba struct {
	persona                         string
	catalogo                        domain.CatalogoImagen
	estado                          *ports.EstadoImagen
	foto                            *ports.FotoImagen
	replay                          bool
	consultas, recuperaciones, usos int
	material                        ports.MaterialImagen
	bytesGuardados                  []byte
}

func (r *registroImagenPrueba) CatalogoVigente(context.Context, ports.OrdenImagen) (domain.CatalogoImagen, error) {
	return r.catalogo, nil
}
func (r *registroImagenPrueba) Consultar(_ context.Context, _ ports.OrdenImagen, m ports.MaterialImagen, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.EstadoImagen, bool, *ports.FotoImagen, error) {
	r.consultas++
	r.material = m
	if r.estado == nil {
		return ports.EstadoImagen{PersonaRef: r.persona}, false, nil, nil
	}
	return *r.estado, true, r.foto, nil
}
func (r *registroImagenPrueba) RecuperarOperacion(_ context.Context, _ ports.OrdenImagen, m ports.MaterialImagen, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboImagen, bool, error) {
	r.recuperaciones++
	r.material = m
	if r.replay {
		return ports.ReciboImagen{ReciboRef: "img_original", PersonaRef: r.persona, Version: m.VersionEsperada + 1, CatalogoVersionRef: m.CatalogoVersionRef, Eleccion: m.Eleccion, FechaUTC: time.Now().UTC()}, true, nil
	}
	return ports.ReciboImagen{}, false, nil
}
func (r *registroImagenPrueba) Guardar(_ context.Context, _ ports.OrdenImagen, m ports.MaterialImagen, foto []byte, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboImagen, error) {
	r.usos++
	r.material = m
	r.bytesGuardados = append([]byte(nil), foto...)
	return ports.ReciboImagen{ReciboRef: "img_nuevo", PersonaRef: r.persona, Version: m.VersionEsperada + 1, CatalogoVersionRef: m.CatalogoVersionRef, Eleccion: m.Eleccion, FotoNueva: m.FotoSHA256 != "", FechaUTC: time.Now().UTC()}, nil
}

type transformadorPrueba struct {
	llamadas int
	err      error
}

func (t *transformadorPrueba) Procesar(_ context.Context, b []byte) (ports.FotoProcesada, error) {
	t.llamadas++
	if t.err != nil {
		return ports.FotoProcesada{}, t.err
	}
	salida := append([]byte{0xff, 0xd8, 0xff}, append(bytes.ToUpper(b), 0xff, 0xd9)...)
	h := sha256.Sum256(salida)
	return ports.FotoProcesada{Bytes: salida, SHA256: hex.EncodeToString(h[:])}, nil
}

type entornoImagenPrueba struct {
	orden         ports.OrdenImagen
	proveedor     *proveedorImagenPrueba
	registro      *registroImagenPrueba
	transformador *transformadorPrueba
	servicio      *ServicioImagen
}

func prepararServicioImagen(t *testing.T) *entornoImagenPrueba {
	t.Helper()
	actor, vinculo := identidadCorreosPrueba(t, vecdomain.SuperficieAutenticacionExternaPersonalV1, "q")
	e := &entornoImagenPrueba{proveedor: &proveedorImagenPrueba{}, transformador: &transformadorPrueba{}}
	var err error
	if e.orden, err = NuevaOrdenImagen(actor, vinculo, vecdomain.SuperficieAutenticacionExternaPersonalV1, e.proveedor); err != nil {
		t.Fatal(err)
	}
	e.registro = &registroImagenPrueba{persona: actor.PersonaRef, catalogo: domain.CatalogoBaseImagen()}
	if e.servicio, err = NuevoServicioImagen(e.registro, e.transformador, time.Now); err != nil {
		t.Fatal(err)
	}
	return e
}

func peticionImagen(operacion string, e domain.EleccionImagen, foto []byte) ports.PeticionImagen {
	return ports.PeticionImagen{Operacion: operacion, CatalogoVersionRef: "usuarios-imagen-v1", ClaveOperacion: "web-imagen-1234567890", Eleccion: e, Foto: foto}
}

func TestConsultaSinEstadoDevuelvePredeterminadaConV3DeConsulta(t *testing.T) {
	e := prepararServicioImagen(t)
	vista, err := e.servicio.Consultar(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	if vista.Estado.Version != 0 || vista.Estado.Eleccion != (domain.EleccionImagen{Modo: domain.ModoImagenIniciales, Paleta: "azul"}) || vista.Foto != nil {
		t.Fatalf("vista inicial incorrecta: %+v", vista)
	}
	m := e.proveedor.materiales[0]
	if m.Accion != ports.AccionConsultarImagen || m.ClaveOperacion != "" || m.HuellaPeticion != "" || m.FotoSHA256 != "" || m.Superficie != vecdomain.SuperficieAutenticacionExternaPersonalV1 {
		t.Fatalf("material de consulta incorrecto: %+v", m)
	}
}

func TestConsultaEntregaSoloFotoJPEGDelModoFoto(t *testing.T) {
	e := prepararServicioImagen(t)
	e.registro.estado = &ports.EstadoImagen{PersonaRef: e.registro.persona, Version: 3, CatalogoVersionRef: "usuarios-imagen-v1", Eleccion: domain.EleccionImagen{Modo: domain.ModoImagenFoto, Paleta: "verde"}}
	e.registro.foto = &ports.FotoImagen{Tipo: ports.TipoFotoImagen, Datos: []byte{0xff, 0xd8, 0xff, 1, 2, 0xff, 0xd9}}
	vista, err := e.servicio.Consultar(context.Background(), e.orden)
	if err != nil || vista.Foto == nil || vista.Estado.Version != 3 {
		t.Fatalf("la foto del titular no se entrega: %v %+v", err, vista)
	}
	e.registro.foto = &ports.FotoImagen{Tipo: "image/svg+xml", Datos: []byte("<svg/>")}
	if _, err := e.servicio.Consultar(context.Background(), e.orden); !errors.Is(err, ports.ErrImagenNoDisponible) {
		t.Fatalf("contenido que no es JPEG entregado: %v", err)
	}
	e.registro.estado.Eleccion = domain.EleccionImagen{Modo: domain.ModoImagenIniciales, Paleta: "verde"}
	e.registro.foto = &ports.FotoImagen{Tipo: ports.TipoFotoImagen, Datos: []byte{0xff, 0xd8, 0xff, 1, 0xff, 0xd9}}
	if _, err := e.servicio.Consultar(context.Background(), e.orden); !errors.Is(err, ports.ErrImagenNoDisponible) {
		t.Fatalf("foto entregada fuera del modo foto: %v", err)
	}
}

func TestElegirIconoAutorizaDosVecesYGuardaSinBytes(t *testing.T) {
	e := prepararServicioImagen(t)
	eleccion := domain.EleccionImagen{Modo: domain.ModoImagenIcono, Paleta: "morado", Icono: "hoja"}
	recibo, err := e.servicio.Guardar(context.Background(), e.orden, peticionImagen(ports.OperacionElegirImagen, eleccion, nil))
	if err != nil || recibo.Replay || recibo.Version != 1 || recibo.FotoNueva {
		t.Fatalf("elección no guardada: %v %+v", err, recibo)
	}
	if e.transformador.llamadas != 0 || e.registro.bytesGuardados != nil || len(e.proveedor.materiales) != 2 || e.registro.recuperaciones != 1 || e.registro.usos != 1 {
		t.Fatal("la elección debe autorizar recuperación y guardado por separado y sin foto")
	}
	m := e.registro.material
	if m.HuellaPeticion != canonico.HuellaPeticionImagen(m.PersonaRef, 0, "usuarios-imagen-v1", eleccion, "") {
		t.Fatal("huella de petición distinta de la canónica")
	}
}

func TestSubirFotoRecodificaYLigaLaV3ALaHuellaDeSalida(t *testing.T) {
	e := prepararServicioImagen(t)
	original := []byte("foto original con exif")
	recibo, err := e.servicio.Guardar(context.Background(), e.orden, peticionImagen(ports.OperacionSubirFotoImagen, domain.EleccionImagen{Modo: domain.ModoImagenFoto, Paleta: "azul"}, original))
	if err != nil || !recibo.FotoNueva {
		t.Fatalf("foto no guardada: %v %+v", err, recibo)
	}
	h := sha256.Sum256(e.registro.bytesGuardados)
	if e.transformador.llamadas != 1 || e.registro.material.FotoSHA256 != hex.EncodeToString(h[:]) || bytes.Contains(e.registro.bytesGuardados, original) {
		t.Fatal("se custodió algo distinto de la salida recodificada")
	}
	if string(original) != "foto original con exif" {
		t.Fatal("el caso de uso alteró el búfer del llamante")
	}
}

func TestRepeticionDevuelveReciboOriginalSinGuardar(t *testing.T) {
	e := prepararServicioImagen(t)
	e.registro.replay = true
	recibo, err := e.servicio.Guardar(context.Background(), e.orden, peticionImagen(ports.OperacionElegirImagen, domain.EleccionImagen{Modo: domain.ModoImagenIniciales, Paleta: "gris"}, nil))
	if err != nil || !recibo.Replay || recibo.ReciboRef != "img_original" || e.registro.usos != 0 {
		t.Fatalf("la repetición no devolvió el recibo original: %v %+v", err, recibo)
	}
}

func TestPeticionesImagenRechazadas(t *testing.T) {
	foto := domain.EleccionImagen{Modo: domain.ModoImagenFoto, Paleta: "azul"}
	casos := []struct {
		nombre   string
		peticion ports.PeticionImagen
		esperado error
	}{
		{"elegir con fichero", peticionImagen(ports.OperacionElegirImagen, foto, []byte("x")), ports.ErrImagenPeticionInvalida},
		{"subir sin fichero", peticionImagen(ports.OperacionSubirFotoImagen, foto, nil), ports.ErrImagenPeticionInvalida},
		{"subir en modo icono", peticionImagen(ports.OperacionSubirFotoImagen, domain.EleccionImagen{Modo: domain.ModoImagenIcono, Paleta: "azul", Icono: "sol"}, []byte("x")), ports.ErrImagenPeticionInvalida},
		{"icono sin código", peticionImagen(ports.OperacionElegirImagen, domain.EleccionImagen{Modo: domain.ModoImagenIcono, Paleta: "azul"}, nil), ports.ErrImagenPeticionInvalida},
		{"paleta libre", peticionImagen(ports.OperacionElegirImagen, domain.EleccionImagen{Modo: domain.ModoImagenIniciales, Paleta: "#ff0000"}, nil), ports.ErrImagenPeticionInvalida},
		{"operación desconocida", peticionImagen("borrar", foto, nil), ports.ErrImagenPeticionInvalida},
		{"fichero enorme", peticionImagen(ports.OperacionSubirFotoImagen, foto, make([]byte, ports.TamanoMaximoFotoImagen+1)), ports.ErrImagenFotoGrande},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			e := prepararServicioImagen(t)
			if _, err := e.servicio.Guardar(context.Background(), e.orden, c.peticion); !errors.Is(err, c.esperado) {
				t.Fatalf("esperado %v, obtenido %v", c.esperado, err)
			}
			if e.transformador.llamadas != 0 || len(e.proveedor.materiales) != 0 {
				t.Fatal("se decodificó o autorizó una petición inválida")
			}
		})
	}
}

func TestCatalogoAntiguoEsConflictoYFotoNoAdmitidaSinAutorizar(t *testing.T) {
	e := prepararServicioImagen(t)
	p := peticionImagen(ports.OperacionElegirImagen, domain.EleccionImagen{Modo: domain.ModoImagenIniciales, Paleta: "azul"}, nil)
	p.CatalogoVersionRef = "usuarios-imagen-v0"
	if _, err := e.servicio.Guardar(context.Background(), e.orden, p); !errors.Is(err, ports.ErrImagenConflicto) || e.registro.usos != 0 {
		t.Fatalf("catálogo antiguo no rechazado: %v", err)
	}
	e = prepararServicioImagen(t)
	e.transformador.err = ports.ErrImagenFotoNoAdmitida
	if _, err := e.servicio.Guardar(context.Background(), e.orden, peticionImagen(ports.OperacionSubirFotoImagen, domain.EleccionImagen{Modo: domain.ModoImagenFoto, Paleta: "azul"}, []byte("svg"))); !errors.Is(err, ports.ErrImagenFotoNoAdmitida) || len(e.proveedor.materiales) != 0 {
		t.Fatalf("foto no admitida o autorizada antes de validarse: %v", err)
	}
}

func TestDenegacionV3YSuperficieCruzada(t *testing.T) {
	e := prepararServicioImagen(t)
	e.proveedor.denegar = true
	if _, err := e.servicio.Consultar(context.Background(), e.orden); !errors.Is(err, ports.ErrImagenProhibido) || e.registro.consultas != 0 {
		t.Fatalf("denegación V3 no respetada: %v", err)
	}
	actor, vinculo := identidadCorreosPrueba(t, vecdomain.SuperficieAutenticacionExternaPersonalV1, "q")
	if _, err := NuevaOrdenImagen(actor, vinculo, vecdomain.SuperficieAutenticacionInternaCorporativaV1, e.proveedor); !errors.Is(err, ports.ErrImagenProhibido) {
		t.Fatalf("una sesión del Área personal no debe usar la ruta de RRHH: %v", err)
	}
}
