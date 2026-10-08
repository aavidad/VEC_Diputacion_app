package httpseguridad

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type lectorPoliticaFallido struct{ causa error }

func (l lectorPoliticaFallido) Read([]byte) (int, error) { return 0, l.causa }

func TestSesionMemoriaCausasInternasRedactadas(t *testing.T) {
	causa := errors.New("ruta privada de prueba")
	_, err := CargarPoliticaSesionMemoriaCertificado(lectorPoliticaFallido{causa: causa})
	if !errors.Is(err, causa) || !errors.Is(err, ErrTokenSesionMemoriaCertificadoNoValido) ||
		err.Error() != ErrTokenSesionMemoriaCertificadoNoValido.Error() ||
		strings.Contains(fmt.Sprintf("%#v", err), "ruta privada") {
		t.Fatalf("fallo de lectura no conservó causa redactada: %v", err)
	}
	err = clavesUnicasPoliticaSesion([]byte(`{"esquema":`))
	if err == nil || !errors.Is(err, io.EOF) || err.Error() != ErrTokenSesionMemoriaCertificadoNoValido.Error() {
		t.Fatalf("JSON truncado perdió causa: %v", err)
	}
	_, err = huellaTokenSesion(strings.Repeat("!", 43))
	var corrupto base64.CorruptInputError
	if !errors.As(err, &corrupto) || err.Error() != ErrTokenSesionMemoriaCertificadoNoValido.Error() ||
		strings.Contains(fmt.Sprintf("%#v", err), "!") {
		t.Fatalf("token mal codificado perdió causa redactada: %v", err)
	}
}

func politicaSesionMemoriaPrueba(t *testing.T, capacidad int) PoliticaSesionMemoriaCertificado {
	t.Helper()
	documento := fmt.Sprintf(`{"esquema":"vec.identidad.sesion-memoria-certificado.v1",`+
		`"referencia":"politica:identidad:sesion-memoria-certificado:prueba-v1",`+
		`"version":1,"ttl_segundos":120,"capacidad":%d}`, capacidad)
	politica, err := CargarPoliticaSesionMemoriaCertificado(strings.NewReader(documento))
	if err != nil {
		t.Fatal(err)
	}
	return politica
}

func TestSesionMemoriaPoliticaVersionadaSinDefecto(t *testing.T) {
	contenido, err := os.ReadFile("../../../../config/identidad_sesion_memoria_v1.ejemplo.json")
	if err != nil {
		t.Fatal(err)
	}
	politica, err := CargarPoliticaSesionMemoriaCertificado(bytes.NewReader(contenido))
	if err != nil || politica.version != 1 || politica.vida != 120*time.Second || politica.capacidad != 4096 {
		t.Fatalf("ejemplo de datos inválido: %v", err)
	}
	for nombre, dato := range map[string]string{
		"duplicado":     `{"esquema":"vec.identidad.sesion-memoria-certificado.v1","esquema":"otro","referencia":"politica:identidad:sesion-memoria-certificado:prueba-v1","version":1,"ttl_segundos":120,"capacidad":1}`,
		"desconocido":   `{"esquema":"vec.identidad.sesion-memoria-certificado.v1","referencia":"politica:identidad:sesion-memoria-certificado:prueba-v1","version":1,"ttl_segundos":120,"capacidad":1,"actor":"cliente"}`,
		"sin TTL":       `{"esquema":"vec.identidad.sesion-memoria-certificado.v1","referencia":"politica:identidad:sesion-memoria-certificado:prueba-v1","version":1,"ttl_segundos":0,"capacidad":1}`,
		"sin capacidad": `{"esquema":"vec.identidad.sesion-memoria-certificado.v1","referencia":"politica:identidad:sesion-memoria-certificado:prueba-v1","version":1,"ttl_segundos":120,"capacidad":0}`,
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := CargarPoliticaSesionMemoriaCertificado(strings.NewReader(dato)); err == nil {
				t.Fatal("política incompleta o ambigua admitida")
			}
		})
	}
	if _, err := NuevoRegistroSesionMemoriaCertificado(&relojFijo{ahora: time.Now().UTC()}, PoliticaSesionMemoriaCertificado{}); err == nil {
		t.Fatal("política por defecto admitida")
	}
}

func TestSesionMemoriaEmisionUnicaConcurrenteYRevocacion(t *testing.T) {
	s, v, _, _, reloj, estado := entornoPresentacionCertificadoPrueba(t)
	almacen, err := NuevoRegistroSesionMemoriaCertificado(reloj, politicaSesionMemoriaPrueba(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	ctx, credencial, prueba := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	capsula, err := s.IniciarYConsumirPresentacion(ctx, credencial, prueba)
	if err != nil {
		t.Fatal(err)
	}
	vinculado, err := s.VincularCapsulaPresentacion(ctx, capsula, credencial.canal)
	if err != nil {
		t.Fatal(err)
	}
	doble := context.WithValue(vinculado, claveCapsulaIdentidad{}, capsulaIdentidadVinculada{})
	if _, err := capsulaTokenCertificadoVinculada(doble, s, prueba, credencial.canal, reloj.Ahora(), true); !errors.Is(err, ErrPresentacionCertificadoNoValida) ||
		err.Error() != ErrTokenSesionMemoriaCertificadoNoValido.Error() {
		t.Fatalf("fallo interno de cápsula no propagado y redactado: %v", err)
	}
	cancelado, cancelar := context.WithCancel(vinculado)
	cancelar()
	if _, _, err := almacen.EmitirDesdeInicio(cancelado, s, prueba, credencial.canal); err == nil {
		t.Fatal("contexto cancelado emitió token")
	}
	type salida struct {
		token   string
		vinculo VinculoSesionMemoriaCertificado
		err     error
	}
	var resultados [2]salida
	var wg sync.WaitGroup
	for i := range resultados {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resultados[i].token, resultados[i].vinculo, resultados[i].err =
				almacen.EmitirDesdeInicio(vinculado, s, prueba, credencial.canal)
		}(i)
	}
	wg.Wait()
	var emitido salida
	exitos := 0
	for _, resultado := range resultados {
		if resultado.err == nil {
			exitos++
			emitido = resultado
		}
	}
	if exitos != 1 || len(emitido.token) != 43 || len(almacen.asientos) != 1 {
		t.Fatalf("emisiones concurrentes=%d, asientos=%d", exitos, len(almacen.asientos))
	}
	if _, _, err := almacen.EmitirDesdeInicio(vinculado, s, prueba, credencial.canal); err == nil {
		t.Fatal("cápsula emitió otro token")
	}
	if got := fmt.Sprintf("%#v", emitido.vinculo); strings.Contains(got, emitido.vinculo.sesionRef) ||
		strings.Contains(got, fmt.Sprintf("%x", emitido.vinculo.huellaToken)) {
		t.Fatal("vínculo filtrado por fmt")
	}
	if _, err := json.Marshal(emitido.vinculo); err == nil {
		t.Fatal("vínculo serializable")
	}
	ctxActual, credencialActual, pruebaActual := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	if _, err := almacen.VerificarActual(ctxActual, emitido.token, s, pruebaActual, credencialActual.canal); err != nil {
		t.Fatalf("token válido no verificó: %v", err)
	}
	if err := almacen.Revocar(emitido.vinculo); err != nil {
		t.Fatal(err)
	}
	if _, err := almacen.VerificarActual(ctxActual, emitido.token, s, pruebaActual, credencialActual.canal); err == nil {
		t.Fatal("token revocado reutilizado")
	}
	if _, _, err := almacen.EmitirDesdeInicio(vinculado, s, prueba, credencial.canal); err == nil {
		t.Fatal("revocación permitió reemitir desde el mismo inicio")
	}
}

func TestSesionMemoriaVerificaAntesDeSQLYNoRenuevaTTL(t *testing.T) {
	s, v, _, registro, reloj, estado := entornoPresentacionCertificadoPrueba(t)
	almacen, err := NuevoRegistroSesionMemoriaCertificado(reloj, politicaSesionMemoriaPrueba(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	ctxInicio, credencialInicio, pruebaInicio := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	inicio, err := s.IniciarYConsumirPresentacion(ctxInicio, credencialInicio, pruebaInicio)
	if err != nil {
		t.Fatal(err)
	}
	registro.original = inicio.datos.resultado.SesionOriginal
	registro.modo = "reanudada"
	ctxVinculado, err := s.VincularCapsulaPresentacion(ctxInicio, inicio, credencialInicio.canal)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := almacen.EmitirDesdeInicio(ctxVinculado, s, pruebaInicio, credencialInicio.canal)
	if err != nil {
		t.Fatal(err)
	}
	ctxActual, credencialActual, pruebaActual := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	vinculo, err := almacen.VerificarActual(ctxActual, token, s, pruebaActual, credencialActual.canal)
	if err != nil || registro.reanudaciones != 0 {
		t.Fatalf("verificación escribió sesión: %v", err)
	}
	reanudada, err := s.ReanudarYConsumirPresentacion(ctxActual, credencialActual, pruebaActual)
	if err != nil || reanudada.datos.resultado.Recibo.ModoInicio != "reanudada" {
		t.Fatalf("reanudación durable: %v", err)
	}
	ctxReanudado, err := s.VincularCapsulaPresentacion(ctxActual, reanudada, credencialActual.canal)
	if err != nil || almacen.ConfirmarReanudacion(ctxReanudado, vinculo) != nil {
		t.Fatalf("confirmación vinculada: %v", err)
	}
	if _, _, err := almacen.EmitirDesdeInicio(ctxReanudado, s, pruebaActual, credencialActual.canal); err == nil {
		t.Fatal("GET reanudada emitió token")
	}
	// El resultado durable 'reanudada' también puede proceder de POST; sólo
	// el marcador privado de origen permite emitir en ese caso.
	ctxPost, credencialPost, pruebaPost := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	postReanudado, err := s.IniciarYConsumirPresentacion(ctxPost, credencialPost, pruebaPost)
	if err != nil || postReanudado.datos.resultado.Recibo.ModoInicio != "reanudada" {
		t.Fatalf("POST replay durable: %v", err)
	}
	ctxPostVinculado, err := s.VincularCapsulaPresentacion(ctxPost, postReanudado, credencialPost.canal)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := almacen.EmitirDesdeInicio(ctxPostVinculado, s, pruebaPost, credencialPost.canal); err != nil {
		t.Fatalf("POST replay no emitió: %v", err)
	}
	ctxAjeno, credencialAjena, pruebaAjena := credencialPresentacionCertificadoPrueba(t, s, v,
		estadoTLSMutuoReal(t, sanDNSConfigurado(s.identidad.configuracion)))
	if _, err := almacen.VerificarActual(ctxAjeno, token, s, pruebaAjena, credencialAjena.canal); err == nil {
		t.Fatal("certificado ajeno reutilizó token")
	}
	sOtro, vOtro, _, _, _, _ := entornoPresentacionCertificadoPrueba(t)
	ctxOtro, credencialOtro, pruebaOtro := credencialPresentacionCertificadoPrueba(t, sOtro, vOtro, estado)
	if _, err := almacen.VerificarActual(ctxOtro, token, sOtro, pruebaOtro, credencialOtro.canal); err == nil {
		t.Fatal("otra instancia de autoridad reutilizó token")
	}
	instanteInicial := reloj.Ahora().UTC()
	reloj.fijar(instanteInicial.Add(31 * time.Second))
	if _, err := almacen.VerificarActual(ctxActual, token, s, pruebaActual, credencialActual.canal); err == nil {
		t.Fatal("GET prolongó TTL")
	}
	nuevo, err := NuevoRegistroSesionMemoriaCertificado(reloj, politicaSesionMemoriaPrueba(t, 4))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nuevo.VerificarActual(ctxActual, token, s, pruebaActual, credencialActual.canal); err == nil {
		t.Fatal("reinicio recuperó token efímero")
	}
}

func TestSesionMemoriaCapacidadAgotadaConservaTokenOriginal(t *testing.T) {
	s, v, _, registro, reloj, estado := entornoPresentacionCertificadoPrueba(t)
	almacen, err := NuevoRegistroSesionMemoriaCertificado(reloj, politicaSesionMemoriaPrueba(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	ctx, credencial, prueba := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	primera, err := s.IniciarYConsumirPresentacion(ctx, credencial, prueba)
	if err != nil {
		t.Fatal(err)
	}
	registro.original = primera.datos.resultado.SesionOriginal
	registro.modo = "reanudada"
	vinculada, err := s.VincularCapsulaPresentacion(ctx, primera, credencial.canal)
	if err != nil {
		t.Fatal(err)
	}
	token, enlace, err := almacen.EmitirDesdeInicio(vinculada, s, prueba, credencial.canal)
	if err != nil {
		t.Fatal(err)
	}
	ctxSegundo, credencialSegunda, pruebaSegunda := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	segunda, err := s.IniciarYConsumirPresentacion(ctxSegundo, credencialSegunda, pruebaSegunda)
	if err != nil {
		t.Fatal(err)
	}
	vinculadaSegunda, err := s.VincularCapsulaPresentacion(ctxSegundo, segunda, credencialSegunda.canal)
	if err != nil {
		t.Fatal(err)
	}
	if nuevo, _, err := almacen.EmitirDesdeInicio(vinculadaSegunda, s, pruebaSegunda, credencialSegunda.canal); err == nil || nuevo != "" {
		t.Fatal("capacidad agotada sobrescribió o emitió token")
	}
	if len(almacen.asientos) != 1 {
		t.Fatalf("asientos tras agotamiento: %d", len(almacen.asientos))
	}
	ctxActual, credencialActual, pruebaActual := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	if _, err := almacen.VerificarActual(ctxActual, token, s, pruebaActual, credencialActual.canal); err != nil {
		t.Fatalf("token original perdido: %v", err)
	}
	if err := almacen.RevocarGeneracion(enlace); err != nil {
		t.Fatal(err)
	}
	if _, err := almacen.VerificarActual(ctxActual, token, s, pruebaActual, credencialActual.canal); err == nil {
		t.Fatal("generación local revocada conservó token")
	}
}

func TestSesionMemoriaGeneracionDurableDistintaInvalidaToken(t *testing.T) {
	s, v, _, registro, reloj, estado := entornoPresentacionCertificadoPrueba(t)
	almacen, err := NuevoRegistroSesionMemoriaCertificado(reloj, politicaSesionMemoriaPrueba(t, 2))
	if err != nil {
		t.Fatal(err)
	}
	ctxInicio, credencialInicio, pruebaInicio := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	inicio, err := s.IniciarYConsumirPresentacion(ctxInicio, credencialInicio, pruebaInicio)
	if err != nil {
		t.Fatal(err)
	}
	registro.original = inicio.datos.resultado.SesionOriginal
	registro.modo = "reanudada"
	vinculado, err := s.VincularCapsulaPresentacion(ctxInicio, inicio, credencialInicio.canal)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := almacen.EmitirDesdeInicio(vinculado, s, pruebaInicio, credencialInicio.canal)
	if err != nil {
		t.Fatal(err)
	}
	ctxActual, credencialActual, pruebaActual := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	enlace, err := almacen.VerificarActual(ctxActual, token, s, pruebaActual, credencialActual.canal)
	if err != nil {
		t.Fatal(err)
	}
	reanudada, err := s.ReanudarYConsumirPresentacion(ctxActual, credencialActual, pruebaActual)
	if err != nil {
		t.Fatal(err)
	}
	// Simula una generación durable más reciente sin falsificar el certificado.
	reanudada.datos.resultado.Recibo.SesionGeneracion++
	ctxReanudado, err := s.VincularCapsulaPresentacion(ctxActual, reanudada, credencialActual.canal)
	if err != nil {
		t.Fatal(err)
	}
	if err := almacen.ConfirmarReanudacion(ctxReanudado, enlace); err == nil {
		t.Fatal("generación durable distinta admitida")
	}
	ctxSiguiente, credencialSiguiente, pruebaSiguiente := credencialPresentacionCertificadoPrueba(t, s, v, estado)
	if _, err := almacen.VerificarActual(ctxSiguiente, token, s, pruebaSiguiente, credencialSiguiente.canal); err == nil {
		t.Fatal("token de generación anterior siguió vivo")
	}
}
