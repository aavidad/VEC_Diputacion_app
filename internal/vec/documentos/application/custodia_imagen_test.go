package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestCustodiaImagenSoloPNG256Completo(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	img.Set(0, 0, color.RGBA{R: 8, A: 255})
	var salida bytes.Buffer
	if err := png.Encode(&salida, img); err != nil {
		t.Fatal(err)
	}
	if !bytesPNG256(salida.Bytes()) {
		t.Fatal("PNG 256 válido rechazado")
	}
	truncado := salida.Bytes()[:salida.Len()-8]
	if bytesPNG256(truncado) {
		t.Fatal("no se debe aceptar un PNG incompleto")
	}
	img2 := image.NewRGBA(image.Rect(0, 0, 257, 256))
	salida.Reset()
	if err := png.Encode(&salida, img2); err != nil {
		t.Fatal(err)
	}
	if bytesPNG256(salida.Bytes()) {
		t.Fatal("dimensiones distintas de 256")
	}
}

type referenciaImagenPrueba struct {
	activa   bool
	err      error
	llamadas int
}

func (r *referenciaImagenPrueba) ReferenciaActiva(_ context.Context, _ ports.OperacionImagen) (bool, error) {
	r.llamadas++
	return r.activa, r.err
}
func TestCustodiaImagenReferenciaRetiradaCierraLectura(t *testing.T) {
	fuente := &referenciaImagenPrueba{activa: false}
	s := ServicioCustodiaImagen{Usuarios: fuente}
	if err := s.referenciaActiva(context.Background(), ports.OperacionImagen{}); !errors.Is(err, ports.ErrImagenProhibida) {
		t.Fatalf("retirada: %v", err)
	}
	fuente.activa = true
	if err := s.referenciaActiva(context.Background(), ports.OperacionImagen{}); err != nil {
		t.Fatalf("activa: %v", err)
	}
	fuente.err = errors.New("usuarios indisponible")
	if err := s.referenciaActiva(context.Background(), ports.OperacionImagen{}); !errors.Is(err, ports.ErrImagenProhibida) {
		t.Fatalf("fallo de Usuarios debe cerrar: %v", err)
	}
	if fuente.llamadas != 3 {
		t.Fatalf("cada apertura debe consultar de nuevo: %d", fuente.llamadas)
	}
}

type autoridadRevocadaImagen struct{}

func (autoridadRevocadaImagen) AutorizarImagen(context.Context, ports.OperacionImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenProhibida
}

type registroImagenNoUsado struct{ ports.RegistroImagen }
type contextosImagenNoUsados struct{ ports.ContextosAlmacenImagen }
type almacenImagenNoUsado struct{ vecports.AlmacenObjetos }
type admisorImagenNoUsado struct{ ports.AdmisorImagen }

func actorImagenPrueba(t *testing.T, ahora time.Time) vecdomain.ContextoActor {
	t.Helper()
	z := strings.Repeat("a", 24)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return actor
}
func TestCustodiaImagenRevocacionDocumentosDuranteLectura(t *testing.T) {
	ahora := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	actor := actorImagenPrueba(t, ahora)
	usuarios := &referenciaImagenPrueba{activa: true}
	s := ServicioCustodiaImagen{Registro: &registroImagenNoUsado{}, Autoridad: autoridadRevocadaImagen{}, Contextos: &contextosImagenNoUsados{}, Almacen: &almacenImagenNoUsado{}, Admisor: &admisorImagenNoUsado{}, Usuarios: usuarios, AhoraUTC: func() time.Time { return ahora }}
	op := ports.OperacionImagen{Actor: actor, TitularPersonaRef: actor.PersonaRef, Audiencia: ports.AudienciaImagenPersonal, Finalidad: ports.FinalidadImagenPropia, Accion: ports.AccionImagenAbrirPropia, DocumentoRef: "doc_1234567890123456"}
	// El buffer simula los bytes ya leídos del almacén. Usuarios sigue activo;
	// solo Documentos ha revocado el permiso mientras se recibían los bytes.
	buffer := []byte("bytes privados ya leídos")
	got, err := s.entregarTrasLectura(context.Background(), op, ports.ReservaImagen{}, buffer)
	if !errors.Is(err, ports.ErrImagenProhibida) || got != nil {
		t.Fatalf("se entregaron bytes tras revocación Documentos: %v, %q", err, got)
	}
	for _, b := range buffer {
		if b != 0 {
			t.Fatal("el buffer denegado no quedó limpio")
		}
	}
	if !usuarios.activa {
		t.Fatal("la prueba debe conservar Usuarios activo")
	}
}

func TestCustodiaImagenLecturaParcialYHuellaLimpianBytes(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	img.Set(0, 0, color.RGBA{R: 8, A: 255})
	var pngValido bytes.Buffer
	if err := png.Encode(&pngValido, img); err != nil {
		t.Fatal(err)
	}
	contenidoValido := pngValido.Bytes()
	if !bytesPNG256(contenidoValido) {
		t.Fatal("fixture PNG 256 inválido")
	}
	for _, tc := range []struct {
		nombre string
		err    error
		sha    string
		tamano int64
	}{
		{"lectura parcial", errors.New("corte de S3"), huella(contenidoValido), int64(len(contenidoValido))},
		{"huella incorrecta", nil, strings.Repeat("a", 64), int64(len(contenidoValido))},
		{"tamano incorrecto", nil, huella(contenidoValido), int64(len(contenidoValido) - 1)},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			buffer := bytes.Clone(contenidoValido)
			got, err := validarContenidoLeidoImagen(buffer, tc.err, tc.sha, tc.tamano)
			if !errors.Is(err, ports.ErrImagenNoDisponible) || got != nil {
				t.Fatalf("se expusieron bytes: %v %q", err, got)
			}
			for _, b := range buffer {
				if b != 0 {
					t.Fatal("buffer denegado no borrado")
				}
			}
		})
	}
}

type autoridadMaterialImagen struct {
	material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (a autoridadMaterialImagen) AutorizarImagen(context.Context, ports.OperacionImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return a.material, nil
}

type registroRecuperacionImagen struct {
	ports.RegistroImagen
	reserva ports.ReservaImagen
}

func (r *registroRecuperacionImagen) RecuperarImagen(context.Context, ports.OperacionImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReservaImagen, bool, error) {
	return r.reserva, true, nil
}
func materialImagenPrueba(t *testing.T, actor vecdomain.ContextoActor, ahora time.Time, accion, efecto string) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h := strings.Repeat("a", 64)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", h, h, "ctx_prueba", h, accion, efecto, h, "vec_documentos.imagen.v1", ahora, ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	material, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, 1, 1, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return material
}
func TestCustodiaImagenRecuperaReservaIncompletaTrasCaida(t *testing.T) {
	ahora := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	actor := actorImagenPrueba(t, ahora)
	id := docdomain.IdentidadCustodiaImagen{PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Audiencia: ports.AudienciaImagenPersonal, Finalidad: ports.FinalidadImagenPropia, VersionEsperada: 7, CatalogoVersionRef: "usuarios-imagen-v1", Paleta: "azul", ClaveOperacion: "operacion-1234567890", HuellaPeticion: strings.Repeat("a", 64), OriginalSHA256: strings.Repeat("b", 64), ContenidoSHA256: strings.Repeat("c", 64), DocumentoRef: "doc_1234567890123456"}
	if err := id.Validar(); err != nil {
		t.Fatal(err)
	}
	registro := &registroRecuperacionImagen{}
	s := ServicioCustodiaImagen{Registro: registro, Autoridad: autoridadMaterialImagen{materialImagenPrueba(t, actor, ahora, ports.AccionImagenRecuperar, actor.PersonaRef)}, Contextos: &contextosImagenNoUsados{}, Almacen: &almacenImagenNoUsado{}, Admisor: &admisorImagenNoUsado{}, Usuarios: &referenciaImagenPrueba{activa: true}, AhoraUTC: func() time.Time { return ahora }}
	op := ports.OperacionImagen{Actor: actor, TitularPersonaRef: actor.PersonaRef, Audiencia: ports.AudienciaImagenPersonal, Finalidad: ports.FinalidadImagenPropia, Accion: ports.AccionImagenRecuperar, ClaveOperacion: id.ClaveOperacion}
	for _, estado := range []docdomain.EstadoCustodiaImagen{docdomain.EstadoImagenReservada, docdomain.EstadoImagenCuarentena} {
		registro.reserva = ports.ReservaImagen{Identidad: id, Estado: estado}
		recuperada, existe, err := s.Recuperar(context.Background(), op)
		if err != nil || !existe || recuperada.Estado != estado || recuperada.Identidad != id {
			t.Fatalf("caída en %s no recuperable: existe=%v err=%v", estado, existe, err)
		}
	}
}

type autoridadFaseImagen struct {
	material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	vista    ports.OperacionImagen
}

func (a *autoridadFaseImagen) AutorizarImagen(_ context.Context, op ports.OperacionImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.vista = op
	return a.material, nil
}

type registroAdmisionImagen struct {
	ports.RegistroImagen
	vista     ports.OperacionImagen
	evidencia string
}

func (r *registroAdmisionImagen) AdmitirImagen(_ context.Context, op ports.OperacionImagen, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, anterior ports.ReservaImagen, evidencia string, objeto vecports.ObjetoAlmacenado) (ports.ReservaImagen, error) {
	r.vista = op
	r.evidencia = evidencia
	anterior.Estado = docdomain.EstadoImagenAdmitida
	anterior.EvidenciaAnalisisRef = evidencia
	anterior.ObjetoAdmitido = objeto
	return anterior, nil
}
func TestCustodiaImagenAdmisionUsaAccionDocumentoYEvidenciaExacta(t *testing.T) {
	ahora := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	actor := actorImagenPrueba(t, ahora)
	id := docdomain.IdentidadCustodiaImagen{PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Audiencia: ports.AudienciaImagenPersonal, Finalidad: ports.FinalidadImagenPropia, CatalogoVersionRef: "usuarios-imagen-v1", Paleta: "azul", ClaveOperacion: "operacion.1234567890", HuellaPeticion: strings.Repeat("a", 64), OriginalSHA256: strings.Repeat("b", 64), ContenidoSHA256: strings.Repeat("c", 64), DocumentoRef: "docimg:12345678-1234-1234-1234-123456789abc"}
	if err := id.Validar(); err != nil {
		t.Fatal(err)
	}
	r := ports.ReservaImagen{Identidad: id, Estado: docdomain.EstadoImagenCuarentena}
	objeto := vecports.ObjetoAlmacenado{Objeto: vecports.ReferenciaObjetoAlmacen{Referencia: "obj_1234567890123456", Version: "ver_1234567890123456"}, ConectorID: "s3_1234567890123456", Zona: vecports.ZonaAlmacenAdmitida, MIME: "image/png", Tamano: 123, HuellaSHA256: id.ContenidoSHA256, EvidenciaCreacionRef: "evi_promocion_123456", AlmacenadoEn: ahora}
	if err := objeto.Validar(); err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadFaseImagen{material: materialImagenPrueba(t, actor, ahora, ports.AccionImagenAdmitir, id.DocumentoRef)}
	registro := &registroAdmisionImagen{}
	s := ServicioCustodiaImagen{Registro: registro, Autoridad: autoridad, Contextos: &contextosImagenNoUsados{}, Almacen: &almacenImagenNoUsado{}, Admisor: &admisorImagenNoUsado{}, Usuarios: &referenciaImagenPrueba{activa: true}, AhoraUTC: func() time.Time { return ahora }}
	op := ports.OperacionImagen{Actor: actor, TitularPersonaRef: actor.PersonaRef, Audiencia: ports.AudienciaImagenPersonal, Finalidad: ports.FinalidadImagenPropia, Accion: ports.AccionImagenReservar, ClaveOperacion: id.ClaveOperacion, HuellaPeticion: id.HuellaPeticion}
	evidencia := "evi_analisis_12345678"
	got, err := s.registrarAdmision(context.Background(), op, r, evidencia, objeto)
	if err != nil {
		t.Fatal(err)
	}
	if autoridad.vista.Accion != ports.AccionImagenAdmitir || autoridad.vista.DocumentoRef != id.DocumentoRef ||
		registro.vista.Accion != autoridad.vista.Accion || registro.vista.DocumentoRef != autoridad.vista.DocumentoRef ||
		registro.evidencia != evidencia || got.EvidenciaAnalisisRef != evidencia {
		t.Fatal("la admisión perdió acción, documento o evidencia exactos")
	}
	if op.DocumentoRef != "" || op.Accion != ports.AccionImagenReservar {
		t.Fatal("la operación base fue mutada")
	}
	if _, err := s.registrarAdmision(context.Background(), op, r, objeto.EvidenciaCreacionRef, objeto); !errors.Is(err, ports.ErrImagenNoDisponible) {
		t.Fatalf("análisis y promoción no pueden compartir evidencia: %v", err)
	}
	autoridad.material = materialImagenPrueba(t, actor, ahora, ports.AccionImagenReservar, actor.PersonaRef)
	if _, err := s.registrarAdmision(context.Background(), op, r, evidencia, objeto); !errors.Is(err, ports.ErrImagenNoDisponible) {
		t.Fatalf("material de reservar no autoriza admitir: %v", err)
	}
	fase := operacionImagenFase(op, r, ports.AccionImagenRegistrarCuarentena)
	if fase.Accion != ports.AccionImagenRegistrarCuarentena || fase.DocumentoRef != id.DocumentoRef {
		t.Fatal("cuarentena carece de acción/recurso exacto")
	}
}
